package domain

import (
	"testing"
	"time"
)

func validTransaction() Transaction {
	return Transaction{
		ID:           "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01",
		PartnerID:    "partner-west",
		OccurredAt:   time.Date(2026, time.August, 16, 20, 30, 0, 0, time.UTC),
		AmountMinor:  725,
		Currency:     "USD",
		AgencyID:     "agency-17",
		PlazaID:      "plaza-4",
		LaneID:       "lane-2",
		VehicleClass: VehicleClassCar,
	}
}

func TestTransactionValidateAcceptsValidFixture(t *testing.T) {
	t.Parallel()

	if err := validTransaction().Validate(); err != nil {
		t.Fatalf("expected valid transaction, got %v", err)
	}
}

func TestTransactionValidateRejectsInvalidFields(t *testing.T) {
	t.Parallel()

	tests := map[string]func(*Transaction){
		"transaction ID": func(tx *Transaction) { tx.ID = "not-a-uuid" },
		"partner ID":     func(tx *Transaction) { tx.PartnerID = "" },
		"timestamp":      func(tx *Transaction) { tx.OccurredAt = time.Time{} },
		"amount":         func(tx *Transaction) { tx.AmountMinor = 0 },
		"currency":       func(tx *Transaction) { tx.Currency = "usd" },
		"agency":         func(tx *Transaction) { tx.AgencyID = "" },
		"plaza":          func(tx *Transaction) { tx.PlazaID = "" },
		"lane":           func(tx *Transaction) { tx.LaneID = "" },
		"vehicle class":  func(tx *Transaction) { tx.VehicleClass = "TRACTOR_BEAM" },
	}

	for name, invalidate := range tests {
		name, invalidate := name, invalidate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tx := validTransaction()
			invalidate(&tx)
			if err := tx.Validate(); err == nil {
				t.Fatalf("expected %s validation error", name)
			}
		})
	}
}

func TestTransactionFingerprintIsCanonicalAndSensitive(t *testing.T) {
	t.Parallel()

	tx := validTransaction()
	first, err := tx.Fingerprint()
	if err != nil {
		t.Fatalf("first fingerprint: %v", err)
	}
	second, err := tx.Fingerprint()
	if err != nil {
		t.Fatalf("second fingerprint: %v", err)
	}
	if first == "" || first != second {
		t.Fatalf("expected stable non-empty fingerprint, got %q and %q", first, second)
	}

	changed := tx
	changed.AmountMinor++
	changedFingerprint, err := changed.Fingerprint()
	if err != nil {
		t.Fatalf("changed fingerprint: %v", err)
	}
	if first == changedFingerprint {
		t.Fatal("expected a payload change to change the fingerprint")
	}
}

func TestTransactionFingerprintRejectsInvalidTransaction(t *testing.T) {
	t.Parallel()

	tx := validTransaction()
	tx.Currency = "US"
	if _, err := tx.Fingerprint(); err == nil {
		t.Fatal("expected invalid transaction fingerprint to fail")
	}
}
