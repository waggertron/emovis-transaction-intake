package domain

import (
	"testing"
	"time"
)

func validTransaction() Transaction {
	return Transaction{Source: "lane-controller-07", SourceReference: "LC07-20260814-000918", TransactionType: "toll", TransactionTimeUTC: time.Date(2026, 8, 14, 13, 45, 2, 0, time.UTC), BaseAmount: "12.50", Currency: "USD", TransponderNumber: "0180012345678", Metadata: map[string]any{"batch": "a"}}
}

func TestTransactionValidation(t *testing.T) {
	allowed := map[string]struct{}{"toll": {}}
	if err := validTransaction().Validate(allowed); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Transaction){
		"source": func(v *Transaction) { v.Source = "" }, "reference": func(v *Transaction) { v.SourceReference = "" },
		"type": func(v *Transaction) { v.TransactionType = "other" }, "time": func(v *Transaction) { v.TransactionTimeUTC = time.Time{} },
		"amount": func(v *Transaction) { v.BaseAmount = "1e2" }, "identifier": func(v *Transaction) { v.TransponderNumber = "" },
	} {
		t.Run(name, func(t *testing.T) {
			v := validTransaction()
			mutate(&v)
			if err := v.Validate(allowed); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestFingerprintChangesWithPayload(t *testing.T) {
	v := validTransaction()
	a, err := v.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	v.BaseAmount = "12.51"
	b, err := v.Fingerprint()
	if err != nil || a == b {
		t.Fatal("fingerprint should change")
	}
}

func TestTransactionValidationRejectsRemainingInvalidValues(t *testing.T) {
	allowed := map[string]struct{}{"toll": {}}
	cases := []func(*Transaction){func(v *Transaction) { v.Currency = "usd" }, func(v *Transaction) { v.Plate = &Plate{Number: "", Jurisdiction: "TX"}; v.TransponderNumber = "" }, func(v *Transaction) { v.TransponderNumber = string(make([]byte, 65)) }, func(v *Transaction) { v.Source = string([]byte{0}) }}
	for _, mutate := range cases {
		v := validTransaction()
		mutate(&v)
		if err := v.Validate(allowed); err == nil {
			t.Fatal("expected invalid transaction")
		}
	}
}

func TestFingerprintRejectsInvalidTransaction(t *testing.T) {
	v := validTransaction()
	v.BaseAmount = "bad"
	if _, err := v.Fingerprint(); err == nil {
		t.Fatal("expected fingerprint validation error")
	}
}
