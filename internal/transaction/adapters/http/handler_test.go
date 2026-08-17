package httpadapter

import (
	"context"
	"errors"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
	"net/http/httptest"
	"strings"
	"testing"
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
	h := NewHandler(&fakeIntake{}, nil, func() string { return "req" }, func() bool { return true })
	r := httptest.NewRequest("POST", "/ingest/v1/transactions", strings.NewReader(`{"source":"s","unknown":1}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("status=%d", w.Code)
	}
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
