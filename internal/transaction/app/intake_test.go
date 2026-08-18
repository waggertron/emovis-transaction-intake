package app

import (
	"context"
	"errors"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
	"testing"
	"time"
)

type fakeStore struct {
	outcome StoreOutcome
	calls   int
	err     error
}

func (s *fakeStore) Accept(_ context.Context, a Acceptance) (StoreOutcome, error) {
	s.calls++
	return s.outcome, s.err
}
func tx() domain.Transaction {
	return domain.Transaction{Source: "source", SourceReference: "ref", TransactionType: "toll", TransactionTimeUTC: time.Now(), BaseAmount: "1.25", TransponderNumber: "tag"}
}
func TestAcceptNewAndReplay(t *testing.T) {
	for _, x := range []struct {
		k    StoreOutcomeKind
		want ResultKind
	}{{StoreAccepted, Accepted}, {StoreReplay, Replayed}} {
		s := &fakeStore{outcome: StoreOutcome{Kind: x.k, TransactionID: "id", EventID: "evt"}}
		r, e := NewIntakeService(s, time.Now, func() string { return "id" }).Accept(context.Background(), AcceptCommand{Transaction: tx()})
		if e != nil || r.Kind != x.want || s.calls != 1 {
			t.Fatalf("result=%+v err=%v", r, e)
		}
	}
}

func TestAcceptReplayReturnsPersistedTransactionID(t *testing.T) {
	store := &fakeStore{outcome: StoreOutcome{Kind: StoreReplay, EventID: "evt-existing", TransactionID: "transaction-existing"}}
	result, err := NewIntakeService(store, time.Now, func() string { return "transaction-new" }).Accept(context.Background(), AcceptCommand{Transaction: tx()})
	if err != nil {
		t.Fatalf("accept replay: %v", err)
	}
	if result.ID != "transaction-existing" || result.TransactionID != "transaction-existing" {
		t.Fatalf("replay returned a generated transaction ID: %#v", result)
	}
}
func TestAcceptRejectsMissingIdentifier(t *testing.T) {
	s := &fakeStore{}
	v := tx()
	v.TransponderNumber = ""
	_, e := NewIntakeService(s, time.Now, func() string { return "id" }).Accept(context.Background(), AcceptCommand{Transaction: v})
	if !errors.Is(e, ErrInvalidTransaction) || s.calls != 0 {
		t.Fatalf("err=%v calls=%d", e, s.calls)
	}
}

func TestAcceptMapsConflictAndStoreFailure(t *testing.T) {
	for _, store := range []*fakeStore{{outcome: StoreOutcome{Kind: StoreConflict}}, {err: errors.New("down")}} {
		_, err := NewIntakeService(store, time.Now, func() string { return "id" }).Accept(context.Background(), AcceptCommand{Transaction: tx()})
		if err == nil {
			t.Fatal("expected error")
		}
	}
}
