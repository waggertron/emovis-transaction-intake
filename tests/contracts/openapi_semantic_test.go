package contracts_test

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPIContractIsStructurallyComplete(t *testing.T) {
	payload, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(payload, &document); err != nil {
		t.Fatalf("parse OpenAPI YAML: %v", err)
	}
	if document["openapi"] != "3.1.0" {
		t.Fatalf("expected OpenAPI 3.1.0, got %#v", document["openapi"])
	}
	paths := mapping(t, document, "paths")
	operations := map[string]string{
		"/v1/transactions": "post",
		"/healthz":         "get",
		"/readyz":          "get",
		"/metrics":         "get",
	}
	for path, method := range operations {
		operation := mapping(t, mapping(t, paths, path), method)
		responses := mapping(t, operation, "responses")
		if _, ok := responses["405"]; !ok {
			t.Errorf("%s %s does not document 405", method, path)
		}
	}
	transaction := mapping(t, mapping(t, paths, "/v1/transactions"), "post")
	responses := mapping(t, transaction, "responses")
	for _, status := range []string{"200", "201", "400", "401", "405", "409", "413", "415", "422", "503"} {
		if _, ok := responses[status]; !ok {
			t.Errorf("transaction endpoint does not document %s", status)
		}
	}
	content := mapping(t, mapping(t, transaction, "requestBody"), "content")
	if _, ok := content["application/json"]; !ok || len(content) != 1 {
		t.Errorf("transaction request must declare only application/json: %#v", content)
	}
	components := mapping(t, document, "components")
	schemas := mapping(t, components, "schemas")
	transactionSchema := mapping(t, schemas, "Transaction")
	required, ok := transactionSchema["required"].([]any)
	if !ok || len(required) != 8 {
		t.Fatalf("transaction schema required fields are incomplete: %#v", transactionSchema["required"])
	}
}

func mapping(t *testing.T, source map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := source[key].(map[string]any)
	if !ok {
		t.Fatalf("%s must be a mapping, got %#v", key, source[key])
	}
	return value
}
