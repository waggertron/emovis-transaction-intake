package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"time"
	"unicode/utf8"

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
	intake          Intake
	auth            Authenticator
	newRequestID    func() string
	ready           func() bool
	defaultCurrency string
}

func NewHandler(intake Intake, auth Authenticator, newRequestID func() string, ready func() bool, currencies ...string) http.Handler {
	defaultCurrency := "USD"
	if len(currencies) > 0 && currencies[0] != "" {
		defaultCurrency = currencies[0]
	}
	handler := &handler{intake: intake, auth: auth, newRequestID: newRequestID, ready: ready, defaultCurrency: defaultCurrency}
	mux := http.NewServeMux()
	mux.HandleFunc("/ingest/v1/transactions", handler.transactions)
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

	partnerID := ""
	if handler.auth != nil {
		var ok bool
		partnerID, ok = handler.auth.Authenticate(request.Header.Get("X-API-Key"))
		if !ok {
			writeError(response, http.StatusUnauthorized, "unauthorized", "invalid API key", requestID)
			return
		}
	}
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(response, http.StatusUnsupportedMediaType, "unsupported_media_type", "content type must be application/json", requestID)
		return
	}
	if request.ContentLength > MaxRequestBodyBytes {
		writeError(response, http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large", requestID)
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, MaxRequestBodyBytes)

	var input transactionRequest
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&input.fields); err != nil {
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

	transaction, err := input.toDomain(partnerID, handler.defaultCurrency)
	if err != nil {
		writeError(response, http.StatusBadRequest, "invalid_transaction", "transaction is invalid", requestID)
		return
	}
	result, err := handler.intake.Accept(request.Context(), app.AcceptCommand{Transaction: transaction, CorrelationID: requestID})
	if err != nil {
		switch {
		case errors.Is(err, app.ErrInvalidTransaction):
			writeError(response, http.StatusBadRequest, "invalid_transaction", "transaction is invalid", requestID)
		case errors.Is(err, app.ErrConflict):
			writeError(response, http.StatusBadRequest, "transaction_conflict", "source reference already has different content", requestID)
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
	writeJSON(response, status, map[string]any{
		"id":                 result.ID,
		"association_status": "received",
		"settlement_status":  "priced",
		"duplicate":          result.Kind == app.Replayed,
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
	fields map[string]json.RawMessage
}

func (input transactionRequest) toDomain(partnerID, defaultCurrency string) (domain.Transaction, error) {
	source, err := input.requiredString("source")
	if err != nil {
		return domain.Transaction{}, err
	}
	sourceReference, err := input.requiredString("source_reference")
	if err != nil {
		return domain.Transaction{}, err
	}
	transactionType, err := input.requiredString("transaction_type")
	if err != nil {
		return domain.Transaction{}, err
	}
	timestamp, err := input.requiredString("transaction_time_utc")
	if err != nil {
		return domain.Transaction{}, err
	}
	baseAmount, err := input.requiredString("base_amount")
	if err != nil {
		return domain.Transaction{}, err
	}
	occurredAt, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return domain.Transaction{}, err
	}
	_, offset := occurredAt.Zone()
	if offset != 0 {
		return domain.Transaction{}, errors.New("transaction_time_utc must use UTC")
	}

	currency := defaultCurrency
	if raw, found := input.fields["currency"]; found && !isNull(raw) {
		if err := json.Unmarshal(raw, &currency); err != nil {
			return domain.Transaction{}, fmt.Errorf("currency: %w", err)
		}
	}
	if utf8.RuneCountInString(currency) > 8 {
		return domain.Transaction{}, errors.New("currency is too long")
	}
	transponder := ""
	if raw, found := input.fields["transponder_number"]; found && !isNull(raw) {
		if err := json.Unmarshal(raw, &transponder); err != nil {
			return domain.Transaction{}, fmt.Errorf("transponder_number: %w", err)
		}
	}
	plate, err := input.optionalPlate()
	if err != nil {
		return domain.Transaction{}, err
	}
	location, locationRaw, err := input.optionalObject("location")
	if err != nil {
		return domain.Transaction{}, err
	}
	metadata, metadataRaw, err := input.optionalObject("metadata")
	if err != nil {
		return domain.Transaction{}, err
	}
	transaction := domain.Transaction{
		Source: source, SourceReference: sourceReference, TransactionType: transactionType,
		TransactionTimeUTC: occurredAt, BaseAmount: baseAmount, Currency: currency, Plate: plate,
		TransponderNumber: transponder, Location: location, Metadata: metadata,
		LocationRaw: locationRaw, MetadataRaw: metadataRaw,
	}
	_ = partnerID
	return transaction, nil
}

func (input transactionRequest) requiredString(name string) (string, error) {
	raw, found := input.fields[name]
	if !found || isNull(raw) {
		return "", fmt.Errorf("%s is required", name)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return value, nil
}

func (input transactionRequest) optionalPlate() (*domain.Plate, error) {
	raw, found := input.fields["plate"]
	if !found {
		return nil, nil
	}
	if isNull(raw) {
		return nil, errors.New("plate cannot be null")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, errors.New("plate must be an object")
	}
	number, err := requiredRawString(fields, "number")
	if err != nil {
		return nil, err
	}
	jurisdiction, err := requiredRawString(fields, "jurisdiction")
	if err != nil {
		return nil, err
	}
	return &domain.Plate{Number: number, Jurisdiction: jurisdiction}, nil
}

func (input transactionRequest) optionalObject(name string) (map[string]any, json.RawMessage, error) {
	raw, found := input.fields[name]
	if !found {
		return nil, nil, nil
	}
	if isNull(raw) {
		return nil, nil, fmt.Errorf("%s cannot be null", name)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil || value == nil {
		return nil, nil, fmt.Errorf("%s must be an object", name)
	}
	return value, append(json.RawMessage(nil), raw...), nil
}

func requiredRawString(fields map[string]json.RawMessage, name string) (string, error) {
	raw, found := fields[name]
	if !found || isNull(raw) {
		return "", fmt.Errorf("plate.%s is required", name)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("plate.%s: %w", name, err)
	}
	return value, nil
}

func isNull(raw json.RawMessage) bool { return bytes.Equal(bytes.TrimSpace(raw), []byte("null")) }

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("request contains more than one JSON value")
	}
	return err
}

func writeError(response http.ResponseWriter, status int, _ string, message, _ string) {
	writeJSON(response, status, map[string]any{"code": status, "message": message})
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
