package memory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
)

func testAcceptance() app.Acceptance {
	transaction := domain.Transaction{
		ID: "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01", PartnerID: "partner-west",
		OccurredAt: time.Date(2026, 8, 16, 20, 30, 0, 0, time.UTC), AmountMinor: 725,
		Currency: "USD", AgencyID: "agency-17", PlazaID: "plaza-4", LaneID: "lane-2",
		VehicleClass: domain.VehicleClassCar,
	}
	fingerprint, _ := transaction.Fingerprint()
	return app.Acceptance{
		Transaction: transaction,
		Fingerprint: fingerprint,
		Event:       app.OutboxEvent{ID: "evt-1", TransactionID: transaction.ID, PartnerID: transaction.PartnerID},
	}
}

func TestStoreAcceptsAtomicallyAndReplaysOriginalEvent(t *testing.T) {
	t.Parallel()

	store := NewStore()
	acceptance := testAcceptance()
	first, err := store.Accept(context.Background(), acceptance)
	if err != nil || first.Kind != app.StoreAccepted || first.EventID != "evt-1" {
		t.Fatalf("unexpected first result %#v, %v", first, err)
	}

	retry := acceptance
	retry.Event.ID = "evt-retry"
	second, err := store.Accept(context.Background(), retry)
	if err != nil || second.Kind != app.StoreReplay || second.EventID != "evt-1" {
		t.Fatalf("unexpected replay result %#v, %v", second, err)
	}
	if len(store.transactions) != 1 || len(store.events) != 1 {
		t.Fatalf("acceptance was not atomic: %d transactions, %d events", len(store.transactions), len(store.events))
	}
}

func TestStoreConflictsOnChangedFingerprint(t *testing.T) {
	t.Parallel()

	store := NewStore()
	acceptance := testAcceptance()
	if _, err := store.Accept(context.Background(), acceptance); err != nil {
		t.Fatalf("first accept: %v", err)
	}
	acceptance.Fingerprint = "different"
	result, err := store.Accept(context.Background(), acceptance)
	if err != nil || result.Kind != app.StoreConflict {
		t.Fatalf("expected conflict, got %#v, %v", result, err)
	}
}

func TestStoreSerializesConcurrentDuplicates(t *testing.T) {
	t.Parallel()

	store := NewStore()
	acceptance := testAcceptance()
	results := make(chan app.StoreOutcome, 8)
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			result, err := store.Accept(context.Background(), acceptance)
			if err != nil {
				t.Errorf("accept: %v", err)
			}
			results <- result
		}()
	}
	group.Wait()
	close(results)

	accepted := 0
	for result := range results {
		if result.Kind == app.StoreAccepted {
			accepted++
		}
	}
	if accepted != 1 || len(store.transactions) != 1 || len(store.events) != 1 {
		t.Fatalf("expected one atomic acceptance, got %d", accepted)
	}
}

func TestStoreOutboxLeaseRetryAndCompletionLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 8, 16, 22, 0, 0, 0, time.UTC)
	store := NewStore()
	if _, err := store.Accept(ctx, testAcceptance()); err != nil {
		t.Fatalf("accept: %v", err)
	}

	claimed, err := store.ClaimPending(ctx, now, 30*time.Second, 10)
	if err != nil || len(claimed) != 1 || claimed[0].Attempts != 0 {
		t.Fatalf("unexpected first claim %#v, %v", claimed, err)
	}
	claimed, err = store.ClaimPending(ctx, now.Add(29*time.Second), 30*time.Second, 10)
	if err != nil || len(claimed) != 0 {
		t.Fatalf("active lease was reclaimed: %#v, %v", claimed, err)
	}
	claimed, err = store.ClaimPending(ctx, now.Add(31*time.Second), 30*time.Second, 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("expired lease was not reclaimed: %#v, %v", claimed, err)
	}

	retryAt := now.Add(time.Minute)
	if err := store.RecordFailure(ctx, app.PublishFailure{EventID: "evt-1", ClaimToken: claimed[0].ClaimToken, Attempts: 1, RetryAt: retryAt, Reason: "publish_failed"}); err != nil {
		t.Fatalf("record failure: %v", err)
	}
	claimed, _ = store.ClaimPending(ctx, retryAt.Add(-time.Nanosecond), 30*time.Second, 10)
	if len(claimed) != 0 {
		t.Fatal("event claimed before retry time")
	}
	claimed, _ = store.ClaimPending(ctx, retryAt, 30*time.Second, 10)
	if len(claimed) != 1 || claimed[0].Attempts != 1 {
		t.Fatalf("retry event not claimed: %#v", claimed)
	}
	if err := store.MarkPublished(ctx, "evt-1", claimed[0].ClaimToken, retryAt); err != nil {
		t.Fatalf("mark published: %v", err)
	}
	claimed, _ = store.ClaimPending(ctx, retryAt.Add(time.Hour), 30*time.Second, 10)
	if len(claimed) != 0 {
		t.Fatal("published event was reclaimed")
	}
}

func TestStoreDoesNotClaimTerminalFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewStore()
	if _, err := store.Accept(ctx, testAcceptance()); err != nil {
		t.Fatalf("accept: %v", err)
	}
	claimed, err := store.ClaimPending(ctx, time.Now(), 30*time.Second, 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim terminal event: %#v, %v", claimed, err)
	}
	if err := store.RecordFailure(ctx, app.PublishFailure{EventID: "evt-1", ClaimToken: claimed[0].ClaimToken, Attempts: 5, Terminal: true, Reason: "publish_failed"}); err != nil {
		t.Fatalf("record terminal failure: %v", err)
	}
	claimed, err = store.ClaimPending(ctx, time.Now(), 30*time.Second, 10)
	if err != nil || len(claimed) != 0 {
		t.Fatalf("terminal event was claimable: %#v, %v", claimed, err)
	}
}

func TestStoreRejectsCompletionFromExpiredClaimOwner(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 8, 17, 6, 0, 0, 0, time.UTC)
	store := NewStore()
	if _, err := store.Accept(ctx, testAcceptance()); err != nil {
		t.Fatalf("accept: %v", err)
	}
	first, err := store.ClaimPending(ctx, now, 30*time.Second, 1)
	if err != nil || len(first) != 1 {
		t.Fatalf("first claim: %#v, %v", first, err)
	}
	second, err := store.ClaimPending(ctx, now.Add(31*time.Second), 30*time.Second, 1)
	if err != nil || len(second) != 1 || second[0].ClaimToken == first[0].ClaimToken {
		t.Fatalf("replacement claim: %#v, %v", second, err)
	}
	if err := store.MarkPublished(ctx, "evt-1", first[0].ClaimToken, now.Add(32*time.Second)); !errors.Is(err, app.ErrLeaseLost) {
		t.Fatalf("stale publication should lose lease, got %v", err)
	}
	if err := store.RecordFailure(ctx, app.PublishFailure{EventID: "evt-1", ClaimToken: first[0].ClaimToken, Attempts: 1}); !errors.Is(err, app.ErrLeaseLost) {
		t.Fatalf("stale failure should lose lease, got %v", err)
	}
	if err := store.MarkPublished(ctx, "evt-1", second[0].ClaimToken, now.Add(32*time.Second)); err != nil {
		t.Fatalf("current owner publication: %v", err)
	}
}
