package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

type VehicleClass string

const (
	VehicleClassCar        VehicleClass = "CAR"
	VehicleClassMotorcycle VehicleClass = "MOTORCYCLE"
	VehicleClassTruck      VehicleClass = "TRUCK"
	VehicleClassBus        VehicleClass = "BUS"
)

var (
	uuidPattern       = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)
	currencyPattern   = regexp.MustCompile(`^[A-Z]{3}$`)
)

type Transaction struct {
	ID           string
	PartnerID    string
	OccurredAt   time.Time
	AmountMinor  int64
	Currency     string
	AgencyID     string
	PlazaID      string
	LaneID       string
	VehicleClass VehicleClass
}

func (transaction Transaction) Validate() error {
	if !uuidPattern.MatchString(transaction.ID) {
		return fmt.Errorf("transaction ID must be a UUID")
	}
	for name, value := range map[string]string{
		"partner ID": transaction.PartnerID,
		"agency ID":  transaction.AgencyID,
		"plaza ID":   transaction.PlazaID,
		"lane ID":    transaction.LaneID,
	} {
		if !identifierPattern.MatchString(value) {
			return fmt.Errorf("%s must be 1-64 safe identifier characters", name)
		}
	}
	if transaction.OccurredAt.IsZero() {
		return fmt.Errorf("occurred at must be set")
	}
	if transaction.AmountMinor <= 0 {
		return fmt.Errorf("amount minor must be positive")
	}
	if !currencyPattern.MatchString(transaction.Currency) {
		return fmt.Errorf("currency must be three uppercase letters")
	}
	switch transaction.VehicleClass {
	case VehicleClassCar, VehicleClassMotorcycle, VehicleClassTruck, VehicleClassBus:
	default:
		return fmt.Errorf("unsupported vehicle class")
	}
	return nil
}

func (transaction Transaction) Fingerprint() (string, error) {
	if err := transaction.Validate(); err != nil {
		return "", err
	}

	canonical := struct {
		ID           string       `json:"transactionId"`
		PartnerID    string       `json:"partnerId"`
		OccurredAt   string       `json:"occurredAt"`
		AmountMinor  int64        `json:"amountMinor"`
		Currency     string       `json:"currency"`
		AgencyID     string       `json:"agencyId"`
		PlazaID      string       `json:"plazaId"`
		LaneID       string       `json:"laneId"`
		VehicleClass VehicleClass `json:"vehicleClass"`
	}{
		ID:           transaction.ID,
		PartnerID:    transaction.PartnerID,
		OccurredAt:   transaction.OccurredAt.UTC().Format(time.RFC3339Nano),
		AmountMinor:  transaction.AmountMinor,
		Currency:     transaction.Currency,
		AgencyID:     transaction.AgencyID,
		PlazaID:      transaction.PlazaID,
		LaneID:       transaction.LaneID,
		VehicleClass: transaction.VehicleClass,
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("marshal canonical transaction: %w", err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
