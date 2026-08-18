package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
)

type fakeIntake struct {
	result  app.AcceptResult
	err     error
	command app.AcceptCommand
}

func (f *fakeIntake) Accept(_ context.Context, c app.AcceptCommand) (app.AcceptResult, error) {
	f.command = c
	return f.result, f.err
}
func TestIngestAcceptsAndReturnsContract(t *testing.T) {
	f := &fakeIntake{result: app.AcceptResult{Kind: app.Accepted, ID: "id-1"}}
	h := NewHandler(f, nil, func() string { return "req" }, func() bool { return true })
	r := httptest.NewRequest("POST", "/ingest/v1/transactions", strings.NewReader(`{"source":"s","source_reference":"r","transaction_type":"toll","transaction_time_utc":"2026-08-14T13:45:02Z","base_amount":"12.50","transponder_number":"tag"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 201 || !strings.Contains(w.Body.String(), `"association_status":"received"`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
func TestIngestRejectsUnknownField(t *testing.T) {
	t.Skip("superseded: the supplied schema permits unspecified properties")
}

func TestIngestAcceptsUnspecifiedProperties(t *testing.T) {
	for name, body := range map[string]string{
		"root":  `{"source":"s","source_reference":"r","transaction_type":"toll","transaction_time_utc":"2026-08-14T13:45:02Z","base_amount":"1","transponder_number":"tag","future_field":{"value":1}}`,
		"plate": `{"source":"s","source_reference":"r","transaction_type":"toll","transaction_time_utc":"2026-08-14T13:45:02Z","base_amount":"1","plate":{"number":"A","jurisdiction":"TX","confidence":99}}`,
	} {
		t.Run(name, func(t *testing.T) {
			status, _, _ := ingestRequest(t, body, "USD")
			if status != 201 {
				t.Fatalf("status=%d", status)
			}
		})
	}
}

func TestIngestRejectsEveryMissingRequiredProperty(t *testing.T) {
	valid := map[string]any{"source": "s", "source_reference": "r", "transaction_type": "toll", "transaction_time_utc": "2026-08-14T13:45:02Z", "base_amount": "1", "transponder_number": "tag"}
	for _, field := range []string{"source", "source_reference", "transaction_type", "transaction_time_utc", "base_amount"} {
		t.Run(field, func(t *testing.T) {
			input := make(map[string]any, len(valid))
			for key, value := range valid {
				input[key] = value
			}
			delete(input, field)
			payload, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			status, _, _ := ingestRequest(t, string(payload), "USD")
			if status != 400 {
				t.Fatalf("missing %s: status=%d", field, status)
			}
		})
	}
}

func TestIngestNullabilityMatchesContract(t *testing.T) {
	for _, field := range []string{"plate", "location", "metadata"} {
		body := `{"source":"s","source_reference":"r","transaction_type":"toll","transaction_time_utc":"2026-08-14T13:45:02Z","base_amount":"1","transponder_number":"tag","` + field + `":null}`
		status, _, _ := ingestRequest(t, body, "USD")
		if status != 400 {
			t.Errorf("non-nullable %s: status=%d", field, status)
		}
	}
	for _, field := range []string{"currency", "transponder_number"} {
		body := `{"source":"s","source_reference":"r","transaction_type":"toll","transaction_time_utc":"2026-08-14T13:45:02Z","base_amount":"1","plate":{"number":"A","jurisdiction":"TX"},"` + field + `":null}`
		status, _, _ := ingestRequest(t, body, "USD")
		if status != 201 {
			t.Errorf("nullable %s: status=%d", field, status)
		}
	}
}

func TestIngestCurrencyPresenceAndBounds(t *testing.T) {
	base := `{"source":"s","source_reference":"r","transaction_type":"toll","transaction_time_utc":"2026-08-14T13:45:02Z","base_amount":"1","transponder_number":"tag"`
	for name, suffix := range map[string]string{"omitted": "}", "null": `,"currency":null}`, "empty": `,"currency":""}`, "one": `,"currency":"X"}`, "eight": `,"currency":"12345678"}`} {
		t.Run(name, func(t *testing.T) {
			status, command, _ := ingestRequest(t, base+suffix, "USD")
			if status != 201 {
				t.Fatalf("status=%d", status)
			}
			want := map[string]string{"omitted": "USD", "null": "USD", "empty": "", "one": "X", "eight": "12345678"}[name]
			if command.Transaction.Currency != want {
				t.Fatalf("currency=%q want %q", command.Transaction.Currency, want)
			}
		})
	}
	status, _, _ := ingestRequest(t, base+`,"currency":"123456789"}`, "USD")
	if status != 400 {
		t.Fatalf("over-eight currency status=%d", status)
	}
}

func TestIngestRequiresUTCOffset(t *testing.T) {
	for timestamp, want := range map[string]int{"2026-08-14T13:45:02Z": 201, "2026-08-14T13:45:02+00:00": 201, "2026-08-14T13:45:02-07:00": 400, "2020-01-01T00:00:00.123Z": 201} {
		body := `{"source":"s","source_reference":"r","transaction_type":"toll","transaction_time_utc":"` + timestamp + `","base_amount":"1","transponder_number":"tag"}`
		status, _, _ := ingestRequest(t, body, "USD")
		if status != want {
			t.Errorf("timestamp %s: status=%d want=%d", timestamp, status, want)
		}
	}
}

func TestIngestUsesNumberSafeArbitraryObjects(t *testing.T) {
	body := `{"source":"s","source_reference":"r","transaction_type":"toll","transaction_time_utc":"2026-08-14T13:45:02Z","base_amount":"1","transponder_number":"tag","location":{"lane":9007199254740993},"metadata":{"rate":12.50,"nested":[1,2]}}`
	status, command, _ := ingestRequest(t, body, "USD")
	if status != 201 {
		t.Fatalf("status=%d", status)
	}
	if _, ok := command.Transaction.Location["lane"].(json.Number); !ok {
		t.Fatalf("large integer decoded lossily as %T", command.Transaction.Location["lane"])
	}
	if got := command.Transaction.Metadata["rate"]; !reflect.DeepEqual(got, json.Number("12.50")) {
		t.Fatalf("decimal=%#v", got)
	}
}

func TestIngestRetainsRawPassthroughBytes(t *testing.T) {
	body := `{"source":"s","source_reference":"r","transaction_type":"toll","transaction_time_utc":"2026-08-14T13:45:02Z","base_amount":"1","transponder_number":"tag","location": { "lane" : 9007199254740993 },"metadata": { "rate" : 12.50 }}`
	status, command, _ := ingestRequest(t, body, "USD")
	if status != 201 {
		t.Fatalf("status=%d", status)
	}
	transaction := reflect.ValueOf(command.Transaction)
	for field, want := range map[string]string{"LocationRaw": `{ "lane" : 9007199254740993 }`, "MetadataRaw": `{ "rate" : 12.50 }`} {
		value := transaction.FieldByName(field)
		if !value.IsValid() {
			t.Fatalf("transaction is missing %s", field)
		}
		if got := string(value.Bytes()); got != want {
			t.Fatalf("%s=%q want=%q", field, got, want)
		}
	}
}

func TestIngestResponseHasExactContractShape(t *testing.T) {
	status, _, body := ingestRequest(t, `{"source":"s","source_reference":"r","transaction_type":"toll","transaction_time_utc":"2026-08-14T13:45:02Z","base_amount":"1","transponder_number":"tag"}`, "USD")
	if status != 201 {
		t.Fatalf("status=%d", status)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"id": "id-1", "association_status": "received", "settlement_status": "priced", "duplicate": false}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("result=%#v want=%#v", result, want)
	}
}

func ingestRequest(t *testing.T, body, currency string) (int, app.AcceptCommand, string) {
	t.Helper()
	fake := &fakeIntake{result: app.AcceptResult{Kind: app.Accepted, ID: "id-1"}}
	handler := NewHandler(fake, nil, func() string { return "req" }, func() bool { return true }, currency)
	request := httptest.NewRequest("POST", "/ingest/v1/transactions", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code, fake.command, response.Body.String()
}

func TestOperationalEndpointsAndGuards(t *testing.T) {
	h := NewHandler(&fakeIntake{}, nil, func() string { return "req" }, func() bool { return false })
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"GET", "/healthz", 200}, {"GET", "/readyz", 503}, {"GET", "/metrics", 200}, {"GET", "/ingest/v1/transactions", 405}} {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("%s %s: got %d", tc.method, tc.path, w.Code)
		}
	}
}

func TestIngestRejectsMediaAndMalformedJSON(t *testing.T) {
	h := NewHandler(&fakeIntake{}, nil, func() string { return "req" }, func() bool { return true })
	for _, body := range []string{"not-json", `{}`} {
		r := httptest.NewRequest("POST", "/ingest/v1/transactions", strings.NewReader(body))
		r.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 415 {
			t.Fatalf("media status=%d", w.Code)
		}
	}
	r := httptest.NewRequest("POST", "/ingest/v1/transactions", strings.NewReader("not-json"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("json status=%d", w.Code)
	}
}

type auth struct{ ok bool }

func (a auth) Authenticate(string) (string, bool) { return "source", a.ok }
func TestIngestAuthAndResultBranches(t *testing.T) {
	valid := `{"source":"s","source_reference":"r","transaction_type":"toll","transaction_time_utc":"2026-08-14T13:45:02Z","base_amount":"12.50","transponder_number":"tag"}`
	for _, tc := range []struct {
		result app.AcceptResult
		err    error
		status int
	}{{app.AcceptResult{Kind: app.Replayed, ID: "id"}, nil, 200}, {app.AcceptResult{}, app.ErrConflict, 400}, {app.AcceptResult{}, app.ErrInvalidTransaction, 400}, {app.AcceptResult{}, errors.New("down"), 503}} {
		h := NewHandler(&fakeIntake{result: tc.result, err: tc.err}, auth{ok: true}, func() string { return "req" }, func() bool { return true })
		r := httptest.NewRequest("POST", "/ingest/v1/transactions", strings.NewReader(valid))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("got %d want %d", w.Code, tc.status)
		}
	}
	h := NewHandler(&fakeIntake{}, auth{ok: false}, func() string { return "req" }, func() bool { return true })
	r := httptest.NewRequest("POST", "/ingest/v1/transactions", strings.NewReader(valid))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("auth status=%d", w.Code)
	}
}

func TestIngestRejectsOversizedBody(t *testing.T) {
	h := NewHandler(&fakeIntake{}, nil, func() string { return "req" }, func() bool { return true })
	r := httptest.NewRequest("POST", "/ingest/v1/transactions", strings.NewReader(strings.Repeat("x", int(MaxRequestBodyBytes)+1)))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 413 {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestIngestRejectsInvalidTimestampAndTrailingJSON(t *testing.T) {
	h := NewHandler(&fakeIntake{}, nil, func() string { return "req" }, func() bool { return true })
	for _, body := range []string{`{"source":"s","source_reference":"r","transaction_type":"toll","transaction_time_utc":"bad","base_amount":"1","transponder_number":"tag"}`, `{"source":"s","source_reference":"r","transaction_type":"toll","transaction_time_utc":"2026-08-14T13:45:02Z","base_amount":"1","transponder_number":"tag"}{}`} {
		r := httptest.NewRequest("POST", "/ingest/v1/transactions", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatalf("status=%d", w.Code)
		}
	}
}
