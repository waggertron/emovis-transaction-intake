package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var amountPattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

type Plate struct {
	Number       string `json:"number"`
	Jurisdiction string `json:"jurisdiction"`
}

type Transaction struct {
	ID                 string          `json:"id"`
	Source             string          `json:"source"`
	SourceReference    string          `json:"source_reference"`
	TransactionType    string          `json:"transaction_type"`
	TransactionTimeUTC time.Time       `json:"transaction_time_utc"`
	BaseAmount         string          `json:"base_amount"`
	Currency           string          `json:"currency"`
	Plate              *Plate          `json:"plate,omitempty"`
	TransponderNumber  string          `json:"transponder_number,omitempty"`
	Location           map[string]any  `json:"-"`
	Metadata           map[string]any  `json:"-"`
	LocationRaw        json.RawMessage `json:"location,omitempty"`
	MetadataRaw        json.RawMessage `json:"metadata,omitempty"`
}

func (t Transaction) Validate(typeSets ...map[string]struct{}) error {
	allowedTypes := map[string]struct{}{"toll": {}}
	if len(typeSets) > 0 && typeSets[0] != nil {
		allowedTypes = typeSets[0]
	}
	if !validBounded(t.Source, 1, 64) {
		return fmt.Errorf("source must be 1-64 characters")
	}
	if !validBounded(t.SourceReference, 1, 128) {
		return fmt.Errorf("source_reference must be 1-128 characters")
	}
	if t.TransactionType == "" {
		return fmt.Errorf("transaction_type must be set")
	}
	if _, ok := allowedTypes[t.TransactionType]; !ok {
		return fmt.Errorf("unrecognized transaction_type")
	}
	if t.TransactionTimeUTC.IsZero() {
		return fmt.Errorf("transaction_time_utc must be set")
	}
	_, offset := t.TransactionTimeUTC.Zone()
	if offset != 0 {
		return fmt.Errorf("transaction_time_utc must use UTC")
	}
	if !amountPattern.MatchString(t.BaseAmount) {
		return fmt.Errorf("base_amount must be a decimal string")
	}
	if utf8.RuneCountInString(t.Currency) > 8 {
		return fmt.Errorf("currency must be at most 8 characters")
	}
	if t.Plate == nil && strings.TrimSpace(t.TransponderNumber) == "" {
		return fmt.Errorf("plate or transponder_number is required")
	}
	if t.Plate != nil && (!validBounded(t.Plate.Number, 1, 16) || !validBounded(t.Plate.Jurisdiction, 1, 8)) {
		return fmt.Errorf("plate is invalid")
	}
	if utf8.RuneCountInString(t.TransponderNumber) > 64 {
		return fmt.Errorf("transponder_number is too long")
	}
	if _, err := canonicalObject(t.LocationRaw, t.Location); err != nil {
		return fmt.Errorf("location must be a JSON object: %w", err)
	}
	if _, err := canonicalObject(t.MetadataRaw, t.Metadata); err != nil {
		return fmt.Errorf("metadata must be a JSON object: %w", err)
	}
	return nil
}

func validBounded(value string, min, max int) bool {
	count := utf8.RuneCountInString(value)
	return utf8.ValidString(value) && count >= min && count <= max
}

func (t Transaction) Fingerprint(typeSets ...map[string]struct{}) (string, error) {
	if err := t.Validate(typeSets...); err != nil {
		return "", err
	}
	location, err := canonicalObject(t.LocationRaw, t.Location)
	if err != nil {
		return "", fmt.Errorf("canonicalize location: %w", err)
	}
	metadata, err := canonicalObject(t.MetadataRaw, t.Metadata)
	if err != nil {
		return "", fmt.Errorf("canonicalize metadata: %w", err)
	}
	payload, err := json.Marshal(struct {
		Source             string          `json:"source"`
		SourceReference    string          `json:"source_reference"`
		TransactionType    string          `json:"transaction_type"`
		TransactionTimeUTC string          `json:"transaction_time_utc"`
		BaseAmount         string          `json:"base_amount"`
		Currency           string          `json:"currency"`
		Plate              *Plate          `json:"plate,omitempty"`
		TransponderNumber  string          `json:"transponder_number,omitempty"`
		Location           json.RawMessage `json:"location,omitempty"`
		Metadata           json.RawMessage `json:"metadata,omitempty"`
	}{
		Source: t.Source, SourceReference: t.SourceReference, TransactionType: t.TransactionType,
		TransactionTimeUTC: t.TransactionTimeUTC.UTC().Format(time.RFC3339Nano), BaseAmount: t.BaseAmount,
		Currency: t.Currency, Plate: t.Plate, TransponderNumber: t.TransponderNumber,
		Location: location, Metadata: metadata,
	})
	if err != nil {
		return "", fmt.Errorf("marshal transaction fingerprint: %w", err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func canonicalObject(raw json.RawMessage, parsed map[string]any) (json.RawMessage, error) {
	if len(raw) == 0 {
		if parsed == nil {
			return nil, nil
		}
		payload, err := json.Marshal(parsed)
		return json.RawMessage(payload), err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("value is null")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}
