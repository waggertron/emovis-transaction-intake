package ndjson

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
)

func ndjsonAcceptance() app.Acceptance {
	transaction := domain.Transaction{
		ID: "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01", PartnerID: "partner-west",
		OccurredAt: time.Date(2026, 8, 16, 20, 30, 0, 0, time.UTC), AmountMinor: 725,
		Currency: "USD", AgencyID: "agency-17", PlazaID: "plaza-4", LaneID: "lane-2",
		VehicleClass: domain.VehicleClassCar,
	}
	fingerprint, _ := transaction.Fingerprint()
	return app.Acceptance{Transaction: transaction, Fingerprint: fingerprint, Event: app.OutboxEvent{
		ID: "evt-1", Type: app.ReviewCandidateEventType, SchemaVersion: 1,
		OccurredAt: time.Date(2026, 8, 16, 22, 0, 0, 0, time.UTC), PartnerID: transaction.PartnerID,
		TransactionID: transaction.ID, Key: transaction.PartnerID + ":" + transaction.ID, Payload: transaction,
	}}
}

func TestStoreRejectsInvalidPathsAndLogRecords(t *testing.T) {
	t.Parallel()
	if _, err := NewStore(""); err == nil {
		t.Fatal("expected empty path failure")
	}
	root := t.TempDir()
	if _, err := NewStore(root); err == nil {
		t.Fatal("expected directory open failure")
	}
	for name, content := range map[string]string{
		"invalid-json":        "not-json\n",
		"unknown-kind":        `{"kind":"unknown"}` + "\n",
		"missing-acceptance":  `{"kind":"accepted"}` + "\n",
		"missing-failure":     `{"kind":"failed"}` + "\n",
		"unknown-publication": `{"kind":"published","eventId":"missing"}` + "\n",
		"unknown-failure":     `{"kind":"failed","failure":{"eventId":"missing"}}` + "\n",
	} {
		path := filepath.Join(root, name+".ndjson")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewStore(path); err == nil {
			t.Fatalf("expected %s failure", name)
		}
	}
	long := filepath.Join(root, "too-long.ndjson")
	if err := os.WriteFile(long, []byte(strings.Repeat("x", 1024*1024+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(long); err == nil {
		t.Fatal("expected scanner limit failure")
	}
}

func TestStoreReportsCancellationMissingEventsAndAppendFailure(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store, err := NewStore(filepath.Join(t.TempDir(), "events.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Accept(ctx, ndjsonAcceptance()); err == nil {
		t.Fatal("expected cancelled accept")
	}
	if _, err := store.ClaimPending(ctx, time.Now(), time.Second, 1); err == nil {
		t.Fatal("expected cancelled claim")
	}
	if err := store.MarkPublished(ctx, "missing", "claim", time.Now()); err == nil {
		t.Fatal("expected cancelled publication")
	}
	if err := store.RecordFailure(ctx, app.PublishFailure{EventID: "missing"}); err == nil {
		t.Fatal("expected cancelled failure")
	}
	if err := store.MarkPublished(context.Background(), "missing", "claim", time.Now()); err == nil {
		t.Fatal("expected missing publication")
	}
	if err := store.RecordFailure(context.Background(), app.PublishFailure{EventID: "missing"}); err == nil {
		t.Fatal("expected missing failure")
	}

	directoryPath := t.TempDir()
	broken := &Store{path: directoryPath, transactions: map[string]storedTransaction{}, events: map[string]*storedEvent{}}
	if _, err := broken.Accept(context.Background(), ndjsonAcceptance()); err == nil {
		t.Fatal("expected append open failure")
	}
}

func TestStorePersistsAcceptanceAndOriginalReplayAcrossRestart(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "transactions.ndjson")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	first, err := store.Accept(context.Background(), ndjsonAcceptance())
	if err != nil || first.Kind != app.StoreAccepted {
		t.Fatalf("accept: %#v, %v", first, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("expected owner-only store file, got %v, %v", info, err)
	}

	reopened, err := NewStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	retry := ndjsonAcceptance()
	retry.Event.ID = "evt-retry"
	replayed, err := reopened.Accept(context.Background(), retry)
	if err != nil || replayed.Kind != app.StoreReplay || replayed.EventID != "evt-1" || replayed.TransactionID != retry.Transaction.ID {
		t.Fatalf("replay: %#v, %v", replayed, err)
	}
	retry.Fingerprint = "changed"
	conflict, err := reopened.Accept(context.Background(), retry)
	if err != nil || conflict.Kind != app.StoreConflict {
		t.Fatalf("conflict: %#v, %v", conflict, err)
	}
}

func TestStorePersistsRetryAndPublishedStateAcrossRestart(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "transactions.ndjson")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if _, err := store.Accept(ctx, ndjsonAcceptance()); err != nil {
		t.Fatalf("accept: %v", err)
	}
	retryAt := time.Date(2026, 8, 16, 23, 0, 0, 0, time.UTC)
	initial, err := store.ClaimPending(ctx, retryAt.Add(-time.Minute), 30*time.Second, 1)
	if err != nil || len(initial) != 1 {
		t.Fatalf("claim initial: %#v, %v", initial, err)
	}
	if err := store.RecordFailure(ctx, app.PublishFailure{EventID: "evt-1", ClaimToken: initial[0].ClaimToken, Attempts: 1, RetryAt: retryAt, Reason: "publish_failed"}); err != nil {
		t.Fatalf("record retry: %v", err)
	}

	reopened, err := NewStore(path)
	if err != nil {
		t.Fatalf("reopen retry: %v", err)
	}
	claimed, err := reopened.ClaimPending(ctx, retryAt, 30*time.Second, 10)
	if err != nil || len(claimed) != 1 || claimed[0].Attempts != 1 {
		t.Fatalf("claim retry: %#v, %v", claimed, err)
	}
	if err := reopened.MarkPublished(ctx, "evt-1", claimed[0].ClaimToken, retryAt); err != nil {
		t.Fatalf("mark published: %v", err)
	}

	reopened, err = NewStore(path)
	if err != nil {
		t.Fatalf("reopen published: %v", err)
	}
	claimed, err = reopened.ClaimPending(ctx, retryAt.Add(time.Hour), 30*time.Second, 10)
	if err != nil || len(claimed) != 0 {
		t.Fatalf("published event reclaimed: %#v, %v", claimed, err)
	}
}
