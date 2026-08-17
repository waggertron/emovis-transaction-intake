package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
)

type fakeIntake struct {
	result  app.AcceptResult
	err     error
	command app.AcceptCommand
	calls   int
}

func (intake *fakeIntake) Accept(_ context.Context, command app.AcceptCommand) (app.AcceptResult, error) {
	intake.calls++
	intake.command = command
	return intake.result, intake.err
}

type fakeAuthenticator struct{}

func (fakeAuthenticator) Authenticate(apiKey string) (string, bool) {
	return "partner-west", apiKey == "test-key"
}

func validJSON() string {
	return `{"transactionId":"018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01","occurredAt":"2026-08-16T20:30:00Z","amountMinor":725,"currency":"USD","agencyId":"agency-17","plazaId":"plaza-4","laneId":"lane-2","vehicleClass":"CAR"}`
}

func testHandler(intake *fakeIntake, ready func() bool) http.Handler {
	return NewHandler(intake, fakeAuthenticator{}, func() string { return "req-generated" }, ready)
}

func TestPostTransactionAccepted(t *testing.T) {
	t.Parallel()

	intake := &fakeIntake{result: app.AcceptResult{Kind: app.Accepted, TransactionID: "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01", EventID: "evt-1"}}
	request := httptest.NewRequest(http.MethodPost, "/v1/transactions", strings.NewReader(validJSON()))
	request.Header.Set("X-API-Key", "test-key")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", "req-client")
	recorder := httptest.NewRecorder()

	testHandler(intake, func() bool { return true }).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if intake.calls != 1 || intake.command.Transaction.PartnerID != "partner-west" || intake.command.CorrelationID != "req-client" {
		t.Fatalf("unexpected application command: %#v", intake.command)
	}
	if recorder.Header().Get("Content-Type") != "application/json" || recorder.Header().Get("X-Request-ID") != "req-client" {
		t.Fatalf("unexpected response headers: %#v", recorder.Header())
	}
}

func TestPostTransactionReplay(t *testing.T) {
	t.Parallel()

	intake := &fakeIntake{result: app.AcceptResult{Kind: app.Replayed, TransactionID: "tx", EventID: "evt-original"}}
	request := httptest.NewRequest(http.MethodPost, "/v1/transactions", strings.NewReader(validJSON()))
	request.Header.Set("X-API-Key", "test-key")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	testHandler(intake, func() bool { return true }).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || recorder.Header().Get("Idempotent-Replay") != "true" {
		t.Fatalf("expected replay response, got %d %#v", recorder.Code, recorder.Header())
	}
}

func TestPostTransactionErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		key        string
		serviceErr error
		wantStatus int
	}{
		{name: "unauthorized", body: validJSON(), wantStatus: http.StatusUnauthorized},
		{name: "malformed", body: `{`, key: "test-key", wantStatus: http.StatusBadRequest},
		{name: "unknown field", body: strings.TrimSuffix(validJSON(), "}") + `,"extra":true}`, key: "test-key", wantStatus: http.StatusBadRequest},
		{name: "semantic invalid", body: validJSON(), key: "test-key", serviceErr: app.ErrInvalidTransaction, wantStatus: http.StatusUnprocessableEntity},
		{name: "conflict", body: validJSON(), key: "test-key", serviceErr: app.ErrConflict, wantStatus: http.StatusConflict},
		{name: "dependency", body: validJSON(), key: "test-key", serviceErr: errors.New("database unavailable"), wantStatus: http.StatusServiceUnavailable},
		{name: "too large", body: strings.Repeat("x", int(MaxRequestBodyBytes)+1), key: "test-key", wantStatus: http.StatusRequestEntityTooLarge},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			intake := &fakeIntake{err: test.serviceErr}
			request := httptest.NewRequest(http.MethodPost, "/v1/transactions", strings.NewReader(test.body))
			request.Header.Set("X-API-Key", test.key)
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			testHandler(intake, func() bool { return true }).ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("expected %d, got %d: %s", test.wantStatus, recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("expected JSON error, got %#v", recorder.Header())
			}
			if strings.Contains(recorder.Body.String(), "database unavailable") {
				t.Fatal("internal dependency detail leaked to client")
			}
			var response struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
				RequestID string `json:"requestId"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil || response.Error.Code == "" || response.Error.Message == "" || response.RequestID == "" {
				t.Fatalf("error body does not satisfy OpenAPI Error schema: %#v err=%v", response, err)
			}
		})
	}
}

func TestPostTransactionRejectsUnsupportedMediaType(t *testing.T) {
	t.Parallel()
	intake := &fakeIntake{}
	request := httptest.NewRequest(http.MethodPost, "/v1/transactions", strings.NewReader(validJSON()))
	request.Header.Set("X-API-Key", "test-key")
	request.Header.Set("Content-Type", "text/plain")
	recorder := httptest.NewRecorder()
	testHandler(intake, func() bool { return true }).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnsupportedMediaType || intake.calls != 0 {
		t.Fatalf("unsupported media type: status=%d calls=%d", recorder.Code, intake.calls)
	}
}

func TestOperationalEndpointsAndMethodGuard(t *testing.T) {
	t.Parallel()

	handler := testHandler(&fakeIntake{}, func() bool { return false })
	for _, test := range []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/healthz", http.StatusOK},
		{http.MethodGet, "/readyz", http.StatusServiceUnavailable},
		{http.MethodGet, "/metrics", http.StatusOK},
		{http.MethodGet, "/v1/transactions", http.StatusMethodNotAllowed},
		{http.MethodGet, "/missing", http.StatusNotFound},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
		{http.MethodPost, "/readyz", http.StatusMethodNotAllowed},
		{http.MethodPost, "/metrics", http.StatusMethodNotAllowed},
	} {
		request := httptest.NewRequest(test.method, test.path, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != test.want {
			t.Errorf("%s %s: expected %d, got %d", test.method, test.path, test.want, recorder.Code)
		}
	}
}

func TestReadinessReadyAndRequestConversionFailures(t *testing.T) {
	t.Parallel()
	handler := testHandler(&fakeIntake{}, func() bool { return true })
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("ready status: %d", recorder.Code)
	}
	for _, body := range []string{
		strings.Replace(validJSON(), "2026-08-16T20:30:00Z", "not-a-time", 1),
		strings.Replace(validJSON(), `"amountMinor":725`, `"amountMinor":0`, 1),
		validJSON() + validJSON(),
	} {
		request = httptest.NewRequest(http.MethodPost, "/v1/transactions", strings.NewReader(body))
		request.Header.Set("X-API-Key", "test-key")
		request.Header.Set("Content-Type", "application/json")
		recorder = httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnprocessableEntity && recorder.Code != http.StatusBadRequest {
			t.Fatalf("unexpected conversion status %d for %s", recorder.Code, body)
		}
	}
}
