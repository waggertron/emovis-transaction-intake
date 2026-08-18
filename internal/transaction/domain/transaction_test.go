package domain

import (
	"encoding/json"
	"reflect"
	"strings"
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
	cases := []func(*Transaction){func(v *Transaction) { v.Plate = &Plate{Number: "", Jurisdiction: "TX"}; v.TransponderNumber = "" }, func(v *Transaction) { v.TransponderNumber = string(make([]byte, 65)) }}
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

func TestTransactionRequiresExplicitConfiguredType(t *testing.T) {
	transaction := validTransaction()
	transaction.TransactionType = ""
	if err := transaction.Validate(map[string]struct{}{"toll": {}}); err == nil {
		t.Fatal("missing transaction_type must not default to toll")
	}
	transaction.TransactionType = "video-toll"
	if err := transaction.Validate(map[string]struct{}{"toll": {}, "video-toll": {}}); err != nil {
		t.Fatalf("runtime-configured type rejected: %v", err)
	}
}

func TestTransactionUsesOpenAPIUnicodeLengthBoundaries(t *testing.T) {
	transaction := validTransaction()
	transaction.Source = strings.Repeat("界", 64)
	transaction.SourceReference = strings.Repeat("界", 128)
	transaction.TransponderNumber = strings.Repeat("界", 64)
	if err := transaction.Validate(); err != nil {
		t.Fatalf("valid Unicode code-point boundaries rejected: %v", err)
	}
	transaction.Source = strings.Repeat("界", 65)
	if err := transaction.Validate(); err == nil {
		t.Fatal("65-code-point source accepted")
	}
}

func TestTransactionAcceptsSchemaValidCurrencyValues(t *testing.T) {
	for _, currency := range []string{"", "X", "usd", "US-DOLL", "界界界界界界界界"} {
		transaction := validTransaction()
		transaction.Currency = currency
		if err := transaction.Validate(); err != nil {
			t.Errorf("currency %q rejected: %v", currency, err)
		}
	}
	transaction := validTransaction()
	transaction.Currency = strings.Repeat("界", 9)
	if err := transaction.Validate(); err == nil {
		t.Fatal("nine-code-point currency accepted")
	}
}

func TestTransactionDecimalGrammar(t *testing.T) {
	for _, amount := range []string{"0", "12", "12.50", "-1", "-0.25", "00.50"} {
		transaction := validTransaction()
		transaction.BaseAmount = amount
		if err := transaction.Validate(); err != nil {
			t.Errorf("valid amount %q rejected: %v", amount, err)
		}
	}
	for _, amount := range []string{"", ".5", "1.", "+1", "1e2", "NaN"} {
		transaction := validTransaction()
		transaction.BaseAmount = amount
		if err := transaction.Validate(); err == nil {
			t.Errorf("invalid amount %q accepted", amount)
		}
	}
}

func TestTransactionRequiresUTCAndAllowsHistoricalFractionalTime(t *testing.T) {
	transaction := validTransaction()
	transaction.TransactionTimeUTC = time.Date(2020, 1, 1, 0, 0, 0, 123, time.UTC)
	if err := transaction.Validate(); err != nil {
		t.Fatalf("historical UTC time rejected: %v", err)
	}
	transaction.TransactionTimeUTC = time.Date(2026, 8, 14, 13, 45, 2, 0, time.FixedZone("PDT", -7*60*60))
	if err := transaction.Validate(); err == nil {
		t.Fatal("non-zero UTC offset accepted")
	}
}

func TestTransactionDoesNotAddUndocumentedIdentifierCharacterRules(t *testing.T) {
	transaction := validTransaction()
	transaction.Source = "lane\ncontroller"
	if err := transaction.Validate(); err != nil {
		t.Fatalf("schema-valid identifier characters rejected: %v", err)
	}
}

func TestFingerprintIgnoresDeprecatedCompatibilityFields(t *testing.T) {
	typeOfTransaction := reflect.TypeOf(Transaction{})
	for _, field := range []string{"PartnerID", "OccurredAt", "AmountMinor", "AgencyID", "PlazaID", "LaneID", "VehicleClass"} {
		if _, found := typeOfTransaction.FieldByName(field); found {
			t.Errorf("deprecated field %s remains", field)
		}
	}
}

func TestFingerprintCanonicalizesRawObjectFormattingWithoutLosingNumbers(t *testing.T) {
	first := validTransaction()
	first.LocationRaw = json.RawMessage(`{ "lane": 9007199254740993, "rate": 12.50 }`)
	first.MetadataRaw = json.RawMessage(`{"nested":{"b":2,"a":1}}`)
	second := validTransaction()
	second.LocationRaw = json.RawMessage(`{"rate":12.50,"lane":9007199254740993}`)
	second.MetadataRaw = json.RawMessage(`{"nested":{"a":1,"b":2}}`)
	firstFingerprint, err := first.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	secondFingerprint, err := second.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if firstFingerprint != secondFingerprint {
		t.Fatal("whitespace or object-key order changed fingerprint")
	}
	second.LocationRaw = json.RawMessage(`{"rate":12.5,"lane":9007199254740993}`)
	secondFingerprint, err = second.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if firstFingerprint == secondFingerprint {
		t.Fatal("meaningful decimal lexeme change did not change fingerprint")
	}
}

func TestTransactionRejectsInvalidRawPassthroughShape(t *testing.T) {
	for _, raw := range []json.RawMessage{json.RawMessage(`null`), json.RawMessage(`[]`), json.RawMessage(`{"broken":`), json.RawMessage(`{} trailing`), json.RawMessage(`{} {}`)} {
		transaction := validTransaction()
		transaction.MetadataRaw = raw
		if err := transaction.Validate(); err == nil {
			t.Errorf("invalid raw object accepted: %s", raw)
		}
	}
}
