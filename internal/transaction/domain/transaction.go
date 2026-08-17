package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	identifierPattern = regexp.MustCompile(`^[^\x00-\x1f\x7f]+$`)
	amountPattern     = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)
	currencyPattern   = regexp.MustCompile(`^[A-Z]{3}$`)
)

type Plate struct {
	Number       string `json:"number"`
	Jurisdiction string `json:"jurisdiction"`
}

type Transaction struct {
	ID                 string         `json:"id"`
	Source             string         `json:"source"`
	SourceReference    string         `json:"source_reference"`
	TransactionType    string         `json:"transaction_type"`
	TransactionTimeUTC time.Time      `json:"transaction_time_utc"`
	BaseAmount         string         `json:"base_amount"`
	Currency           string         `json:"currency"`
	Plate              *Plate         `json:"plate,omitempty"`
	TransponderNumber  string         `json:"transponder_number,omitempty"`
	Location           map[string]any `json:"location,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	// Deprecated: legacy fields retained only for source compatibility during migration.
	PartnerID    string
	OccurredAt   time.Time
	AmountMinor  int64
	AgencyID     string
	PlazaID      string
	LaneID       string
	VehicleClass string
}

// Deprecated legacy constants retained for source compatibility during migration.
const (
	VehicleClassCar        = "CAR"
	VehicleClassMotorcycle = "MOTORCYCLE"
	VehicleClassTruck      = "TRUCK"
	VehicleClassBus        = "BUS"
)

func (t Transaction) Validate(typeSets ...map[string]struct{}) error {
	if t.Source == "" && t.PartnerID != "" {
		t.Source = t.PartnerID
	}
	if t.SourceReference == "" && t.ID != "" {
		t.SourceReference = t.ID
	}
	if t.TransactionType == "" {
		t.TransactionType = "toll"
	}
	if t.TransactionTimeUTC.IsZero() {
		t.TransactionTimeUTC = t.OccurredAt
	}
	if t.BaseAmount == "" && t.AmountMinor > 0 {
		t.BaseAmount = fmt.Sprintf("%d", t.AmountMinor)
	}
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
	if _, ok := allowedTypes[t.TransactionType]; !ok {
		return fmt.Errorf("unrecognized transaction_type")
	}
	if t.TransactionTimeUTC.IsZero() {
		return fmt.Errorf("transaction_time_utc must be set")
	}
	if !amountPattern.MatchString(t.BaseAmount) {
		return fmt.Errorf("base_amount must be a decimal string")
	}
	if t.Currency != "" && !currencyPattern.MatchString(t.Currency) {
		return fmt.Errorf("currency must be three uppercase letters")
	}
	if t.Plate == nil && strings.TrimSpace(t.TransponderNumber) == "" {
		return fmt.Errorf("plate or transponder_number is required")
	}
	if t.Plate != nil && (!validBounded(t.Plate.Number, 1, 16) || !validBounded(t.Plate.Jurisdiction, 1, 8)) {
		return fmt.Errorf("plate is invalid")
	}
	if len(t.TransponderNumber) > 64 {
		return fmt.Errorf("transponder_number is too long")
	}
	return nil
}

func validBounded(value string, min, max int) bool {
	return len(value) >= min && len(value) <= max && identifierPattern.MatchString(value)
}

func (t Transaction) Fingerprint(typeSets ...map[string]struct{}) (string, error) {
	allowedTypes := map[string]struct{}{"toll": {}}
	if len(typeSets) > 0 && typeSets[0] != nil {
		allowedTypes = typeSets[0]
	}
	if err := t.Validate(allowedTypes); err != nil {
		return "", err
	}
	payload, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("marshal transaction: %w", err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
