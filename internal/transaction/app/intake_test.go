package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
)

type fakeTransactionStore struct {
	outcome StoreOutcome
	err     error
	calls   int
	input   Acceptance
}

func (store *fakeTransactionStore) Accept(_ context.Context, acceptance Acceptance) (StoreOutcome, error) {
	store.calls++
	store.input = acceptance
	return store.outcome, store.err
}

func validIntakeTransaction() domain.Transaction {
	return domain.Transaction{
		ID:           "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01",
		PartnerID:    "partner-west",
		OccurredAt:   time.Date(2026, time.August, 16, 20, 30, 0, 0, time.UTC),
		AmountMinor:  725,
		Currency:     "USD",
		AgencyID:     "agency-17",
		PlazaID:      "plaza-4",
		LaneID:       "lane-2",
		VehicleClass: domain.VehicleClassCar,
	}
}

func TestIntakeAcceptsTransactionWithPendingOutboxEvent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 16, 21, 0, 0, 0, time.UTC)
	store := &fakeTransactionStore{outcome: StoreOutcome{Kind: StoreAccepted, EventID: "evt-1"}}
	service := NewIntakeService(store, func() time.Time { return now }, func() string { return "evt-1" })

	result, err := service.Accept(context.Background(), AcceptCommand{
		Transaction:   validIntakeTransaction(),
		CorrelationID: "req-1",
	})
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if result.Kind != Accepted || result.EventID != "evt-1" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if store.calls != 1 {
		t.Fatalf("expected one store call, got %d", store.calls)
	}
	if store.input.Event.ID != "evt-1" || store.input.Event.Key != "partner-west:018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01" {
		t.Fatalf("unexpected event identity: %#v", store.input.Event)
	}
	if store.input.Event.Type != ReviewCandidateEventType || store.input.Event.SchemaVersion != 1 {
		t.Fatalf("unexpected event contract: %#v", store.input.Event)
	}
	if store.input.Event.CorrelationID != "req-1" || !store.input.Event.OccurredAt.Equal(now) {
		t.Fatalf("unexpected event metadata: %#v", store.input.Event)
	}
	if store.input.Fingerprint == "" {
		t.Fatal("expected canonical transaction fingerprint")
	}
}

func TestIntakeReturnsIdempotentReplay(t *testing.T) {
	t.Parallel()

	store := &fakeTransactionStore{outcome: StoreOutcome{Kind: StoreReplay, EventID: "original-event"}}
	service := NewIntakeService(store, time.Now, func() string { return "unused-event" })

	result, err := service.Accept(context.Background(), AcceptCommand{Transaction: validIntakeTransaction()})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if result.Kind != Replayed || result.EventID != "original-event" {
		t.Fatalf("unexpected replay result: %#v", result)
	}
}

func TestIntakeMapsChangedDuplicateToConflict(t *testing.T) {
	t.Parallel()

	store := &fakeTransactionStore{outcome: StoreOutcome{Kind: StoreConflict}}
	service := NewIntakeService(store, time.Now, func() string { return "evt-1" })

	_, err := service.Accept(context.Background(), AcceptCommand{Transaction: validIntakeTransaction()})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestIntakeRejectsInvalidTransactionBeforeStorage(t *testing.T) {
	t.Parallel()

	store := &fakeTransactionStore{}
	service := NewIntakeService(store, time.Now, func() string { return "evt-1" })
	transaction := validIntakeTransaction()
	transaction.AmountMinor = 0

	_, err := service.Accept(context.Background(), AcceptCommand{Transaction: transaction})
	if !errors.Is(err, ErrInvalidTransaction) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if store.calls != 0 {
		t.Fatalf("invalid transaction reached store %d times", store.calls)
	}
}

func TestIntakePropagatesStoreFailure(t *testing.T) {
	t.Parallel()

	storeFailure := errors.New("storage unavailable")
	store := &fakeTransactionStore{err: storeFailure}
	service := NewIntakeService(store, time.Now, func() string { return "evt-1" })

	_, err := service.Accept(context.Background(), AcceptCommand{Transaction: validIntakeTransaction()})
	if !errors.Is(err, storeFailure) {
		t.Fatalf("expected wrapped store error, got %v", err)
	}
}
