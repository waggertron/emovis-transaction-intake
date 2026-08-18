package contracts_test

import (
	"bytes"
	"os"
	"reflect"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	activeContract   = "../../api/openapi.yaml"
	suppliedContract = "../../docs/2026-08-18-supplied-openapi.yaml"
)

func TestOpenAPIContractMatchesSuppliedArtifactExactly(t *testing.T) {
	active := readFile(t, activeContract)
	supplied := readFile(t, suppliedContract)
	if !bytes.Equal(active, supplied) {
		t.Fatal("api/openapi.yaml must be a byte-for-byte copy of the supplied contract")
	}
}

func TestOpenAPIContractIsStructurallyComplete(t *testing.T) {
	document := parseDocument(t, activeContract)
	if document["openapi"] != "3.0.3" {
		t.Fatalf("expected OpenAPI 3.0.3, got %#v", document["openapi"])
	}
	info := mapping(t, document, "info")
	assertEqual(t, info["title"], "Transaction Ingest API", "title")
	assertEqual(t, info["version"], "1.0.0", "version")
	assertContains(t, stringValue(t, info, "description"), "idempotent on (source, source_reference)", "info description")

	paths := mapping(t, document, "paths")
	if got := sortedKeys(paths); !reflect.DeepEqual(got, []string{"/ingest/v1/transactions"}) {
		t.Fatalf("supplied contract paths changed: %#v", got)
	}
	operation := mapping(t, mapping(t, paths, "/ingest/v1/transactions"), "post")
	assertEqual(t, operation["operationId"], "ingestTransaction", "operationId")
	assertEqual(t, operation["summary"], "Ingest a billable transaction (producer; idempotent)", "summary")
	assertContains(t, stringValue(t, operation, "description"), "duplicate=true", "operation description")
	security, ok := operation["security"].([]any)
	if !ok || len(security) != 0 {
		t.Fatalf("operation security must be an explicit empty array: %#v", operation["security"])
	}
	requestBody := mapping(t, operation, "requestBody")
	assertEqual(t, requestBody["required"], true, "request body required")
	requestSchema := mapping(t, mapping(t, mapping(t, requestBody, "content"), "application/json"), "schema")
	assertEqual(t, requestSchema["$ref"], "#/components/schemas/IngestRequest", "request schema ref")

	responses := mapping(t, operation, "responses")
	if got := sortedKeys(responses); !reflect.DeepEqual(got, []string{"200", "201", "400"}) {
		t.Fatalf("response set changed: %#v", got)
	}
	for status, ref := range map[string]string{"200": "#/components/schemas/IngestResult", "201": "#/components/schemas/IngestResult", "400": "#/components/schemas/Error"} {
		response := mapping(t, responses, status)
		if stringValue(t, response, "description") == "" {
			t.Fatalf("response %s description is empty", status)
		}
		schema := mapping(t, mapping(t, mapping(t, response, "content"), "application/json"), "schema")
		assertEqual(t, schema["$ref"], ref, "response schema ref")
	}
}

func TestOpenAPIRequestAndResponseSchemasRetainEveryConstraint(t *testing.T) {
	schemas := mapping(t, mapping(t, parseDocument(t, activeContract), "components"), "schemas")
	if got := sortedKeys(schemas); !reflect.DeepEqual(got, []string{"AssociationStatus", "Error", "IngestRequest", "IngestResult", "Location", "PlateRef", "SettlementStatus"}) {
		t.Fatalf("named schema set changed: %#v", got)
	}
	ingest := mapping(t, schemas, "IngestRequest")
	assertStringSet(t, ingest["required"], []string{"source", "source_reference", "transaction_type", "transaction_time_utc", "base_amount"})
	if _, tightened := ingest["additionalProperties"]; tightened {
		t.Fatal("IngestRequest must retain OpenAPI's default additional-properties behavior")
	}
	properties := mapping(t, ingest, "properties")
	if got := sortedKeys(properties); !reflect.DeepEqual(got, []string{"base_amount", "currency", "location", "metadata", "plate", "source", "source_reference", "transaction_time_utc", "transaction_type", "transponder_number"}) {
		t.Fatalf("request property set changed: %#v", got)
	}
	assertBounds(t, mapping(t, properties, "source"), 1, 64)
	assertBounds(t, mapping(t, properties, "source_reference"), 1, 128)
	assertEqual(t, mapping(t, properties, "transaction_time_utc")["format"], "date-time", "timestamp format")
	assertNullableMax(t, mapping(t, properties, "transponder_number"), 64)
	assertNullableMax(t, mapping(t, properties, "currency"), 8)
	assertEqual(t, mapping(t, properties, "plate")["$ref"], "#/components/schemas/PlateRef", "plate ref")
	assertEqual(t, mapping(t, properties, "location")["$ref"], "#/components/schemas/Location", "location ref")
	assertEqual(t, mapping(t, properties, "metadata")["additionalProperties"], true, "metadata additional properties")

	plate := mapping(t, schemas, "PlateRef")
	assertStringSet(t, plate["required"], []string{"number", "jurisdiction"})
	plateProperties := mapping(t, plate, "properties")
	assertBounds(t, mapping(t, plateProperties, "number"), 1, 16)
	assertBounds(t, mapping(t, plateProperties, "jurisdiction"), 1, 8)
	assertEqual(t, mapping(t, schemas, "Location")["additionalProperties"], true, "location additional properties")
	assertStringSet(t, mapping(t, schemas, "AssociationStatus")["enum"], []string{"received", "resolving", "associated", "exception"})
	assertStringSet(t, mapping(t, schemas, "SettlementStatus")["enum"], []string{"unpriced", "priced", "payable", "paid"})

	result := mapping(t, schemas, "IngestResult")
	assertStringSet(t, result["required"], []string{"id", "association_status", "settlement_status", "duplicate"})
	resultProperties := mapping(t, result, "properties")
	assertEqual(t, mapping(t, resultProperties, "association_status")["$ref"], "#/components/schemas/AssociationStatus", "association ref")
	assertEqual(t, mapping(t, resultProperties, "settlement_status")["$ref"], "#/components/schemas/SettlementStatus", "settlement ref")
	errorProperties := mapping(t, mapping(t, schemas, "Error"), "properties")
	assertEqual(t, mapping(t, errorProperties, "code")["format"], "int32", "error code format")
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func parseDocument(t *testing.T, path string) map[string]any {
	t.Helper()
	var document map[string]any
	if err := yaml.Unmarshal(readFile(t, path), &document); err != nil {
		t.Fatalf("parse OpenAPI YAML: %v", err)
	}
	return document
}

func mapping(t *testing.T, source map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := source[key].(map[string]any)
	if !ok {
		t.Fatalf("%s must be a mapping, got %#v", key, source[key])
	}
	return value
}

func sortedKeys(source map[string]any) []string {
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func assertEqual(t *testing.T, got, want any, label string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %#v want %#v", label, got, want)
	}
}

func assertContains(t *testing.T, got, want, label string) {
	t.Helper()
	if !bytes.Contains([]byte(got), []byte(want)) {
		t.Fatalf("%s does not contain %q", label, want)
	}
}

func stringValue(t *testing.T, source map[string]any, key string) string {
	t.Helper()
	value, ok := source[key].(string)
	if !ok {
		t.Fatalf("%s must be a string, got %#v", key, source[key])
	}
	return value
}

func assertStringSet(t *testing.T, got any, want []string) {
	t.Helper()
	values, ok := got.([]any)
	if !ok {
		t.Fatalf("expected string array, got %#v", got)
	}
	actual := make([]string, len(values))
	for index, value := range values {
		actual[index], ok = value.(string)
		if !ok {
			t.Fatalf("array value is not a string: %#v", value)
		}
	}
	sort.Strings(actual)
	sort.Strings(want)
	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("string set: got %#v want %#v", actual, want)
	}
}

func assertBounds(t *testing.T, schema map[string]any, min, max int) {
	t.Helper()
	assertEqual(t, schema["type"], "string", "bounded type")
	assertEqual(t, schema["minLength"], min, "minimum length")
	assertEqual(t, schema["maxLength"], max, "maximum length")
}

func assertNullableMax(t *testing.T, schema map[string]any, max int) {
	t.Helper()
	assertEqual(t, schema["type"], "string", "nullable type")
	assertEqual(t, schema["nullable"], true, "nullable")
	assertEqual(t, schema["maxLength"], max, "nullable maximum length")
}
