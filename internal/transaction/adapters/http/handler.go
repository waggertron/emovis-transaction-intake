package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
)

const MaxRequestBodyBytes int64 = 64 * 1024

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type Intake interface {
	Accept(context.Context, app.AcceptCommand) (app.AcceptResult, error)
}

type Authenticator interface {
	Authenticate(apiKey string) (partnerID string, ok bool)
}

type handler struct {
	intake       Intake
	auth         Authenticator
	newRequestID func() string
	ready        func() bool
}

func NewHandler(intake Intake, auth Authenticator, newRequestID func() string, ready func() bool) http.Handler {
	handler := &handler{intake: intake, auth: auth, newRequestID: newRequestID, ready: ready}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transactions", handler.transactions)
	mux.HandleFunc("/healthz", handler.health)
	mux.HandleFunc("/readyz", handler.readiness)
	mux.HandleFunc("/metrics", handler.metrics)
	return securityHeaders(mux)
}

func (handler *handler) transactions(response http.ResponseWriter, request *http.Request) {
	requestID := request.Header.Get("X-Request-ID")
	if !requestIDPattern.MatchString(requestID) {
		requestID = handler.newRequestID()
	}
	response.Header().Set("X-Request-ID", requestID)
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", http.MethodPost)
		writeError(response, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestID)
		return
	}

	partnerID, ok := handler.auth.Authenticate(request.Header.Get("X-API-Key"))
	if !ok {
		writeError(response, http.StatusUnauthorized, "unauthorized", "invalid API key", requestID)
		return
	}
	if request.ContentLength > MaxRequestBodyBytes {
		writeError(response, http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large", requestID)
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, MaxRequestBodyBytes)

	var input transactionRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(response, http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large", requestID)
			return
		}
		writeError(response, http.StatusBadRequest, "invalid_json", "request body must be valid JSON", requestID)
		return
	}
	if err := ensureJSONEnd(decoder); err != nil {
		writeError(response, http.StatusBadRequest, "invalid_json", "request body must contain one JSON object", requestID)
		return
	}

	transaction, err := input.toDomain(partnerID)
	if err != nil {
		writeError(response, http.StatusUnprocessableEntity, "invalid_transaction", "transaction is invalid", requestID)
		return
	}
	result, err := handler.intake.Accept(request.Context(), app.AcceptCommand{Transaction: transaction, CorrelationID: requestID})
	if err != nil {
		switch {
		case errors.Is(err, app.ErrInvalidTransaction):
			writeError(response, http.StatusUnprocessableEntity, "invalid_transaction", "transaction is invalid", requestID)
		case errors.Is(err, app.ErrConflict):
			writeError(response, http.StatusConflict, "transaction_conflict", "transaction ID already has different content", requestID)
		default:
			writeError(response, http.StatusServiceUnavailable, "service_unavailable", "transaction service is unavailable", requestID)
		}
		return
	}

	status := http.StatusCreated
	if result.Kind == app.Replayed {
		status = http.StatusOK
		response.Header().Set("Idempotent-Replay", "true")
	}
	writeJSON(response, status, map[string]string{
		"transactionId": result.TransactionID,
		"eventId":       result.EventID,
		"status":        string(result.Kind),
	})
}

func (handler *handler) health(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(response, http.StatusOK, map[string]string{"status": "ok"})
}

func (handler *handler) readiness(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !handler.ready() {
		writeJSON(response, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(response, http.StatusOK, map[string]string{"status": "ready"})
}

func (handler *handler) metrics(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	response.Header().Set("Content-Type", "text/plain; version=0.0.4")
	response.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(response, "transaction_service_up 1\n")
}

type transactionRequest struct {
	TransactionID string              `json:"transactionId"`
	OccurredAt    string              `json:"occurredAt"`
	AmountMinor   int64               `json:"amountMinor"`
	Currency      string              `json:"currency"`
	AgencyID      string              `json:"agencyId"`
	PlazaID       string              `json:"plazaId"`
	LaneID        string              `json:"laneId"`
	VehicleClass  domain.VehicleClass `json:"vehicleClass"`
}

func (input transactionRequest) toDomain(partnerID string) (domain.Transaction, error) {
	occurredAt, err := time.Parse(time.RFC3339, input.OccurredAt)
	if err != nil {
		return domain.Transaction{}, err
	}
	transaction := domain.Transaction{
		ID: input.TransactionID, PartnerID: partnerID, OccurredAt: occurredAt,
		AmountMinor: input.AmountMinor, Currency: input.Currency, AgencyID: input.AgencyID,
		PlazaID: input.PlazaID, LaneID: input.LaneID, VehicleClass: input.VehicleClass,
	}
	if err := transaction.Validate(); err != nil {
		return domain.Transaction{}, err
	}
	return transaction, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func writeError(response http.ResponseWriter, status int, code, message, requestID string) {
	writeJSON(response, status, map[string]any{
		"error":     map[string]string{"code": code, "message": message},
		"requestId": requestID,
	})
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("X-Frame-Options", "DENY")
		response.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(response, request)
	})
}
