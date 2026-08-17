package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeOutboxStore struct {
	events     []PendingEvent
	claimErr   error
	markErr    error
	recordErr  error
	marked     []string
	failures   []PublishFailure
	claimLease time.Duration
	claimLimit int
}

func (store *fakeOutboxStore) ClaimPending(_ context.Context, _ time.Time, lease time.Duration, limit int) ([]PendingEvent, error) {
	store.claimLease, store.claimLimit = lease, limit
	return store.events, store.claimErr
}

func (store *fakeOutboxStore) MarkPublished(_ context.Context, eventID, _ string, _ time.Time) error {
	store.marked = append(store.marked, eventID)
	return store.markErr
}

func (store *fakeOutboxStore) RecordFailure(_ context.Context, failure PublishFailure) error {
	store.failures = append(store.failures, failure)
	return store.recordErr
}

type fakePublisher struct {
	fail map[string]error
	seen []OutboxEvent
}

func (publisher *fakePublisher) Publish(_ context.Context, event OutboxEvent) error {
	publisher.seen = append(publisher.seen, event)
	return publisher.fail[event.ID]
}

func TestDispatcherPublishesAndMarksClaimedEvents(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 16, 22, 0, 0, 0, time.UTC)
	store := &fakeOutboxStore{events: []PendingEvent{{Event: OutboxEvent{ID: "evt-1"}}, {Event: OutboxEvent{ID: "evt-2"}}}}
	publisher := &fakePublisher{fail: map[string]error{}}
	dispatcher := NewDispatcher(store, publisher, func() time.Time { return now }, DefaultDispatcherConfig())

	result, err := dispatcher.RunBatch(context.Background())
	if err != nil {
		t.Fatalf("run batch: %v", err)
	}
	if result.Claimed != 2 || result.Published != 2 || result.Failed != 0 {
		t.Fatalf("unexpected result %#v", result)
	}
	if store.claimLease != 30*time.Second || store.claimLimit != 10 {
		t.Fatalf("unexpected claim settings: %s, %d", store.claimLease, store.claimLimit)
	}
	if len(store.marked) != 2 || len(publisher.seen) != 2 {
		t.Fatalf("expected both events published and marked")
	}
}

func TestDispatcherRetriesAndTerminatesFailedEvents(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 16, 22, 0, 0, 0, time.UTC)
	store := &fakeOutboxStore{events: []PendingEvent{
		{Event: OutboxEvent{ID: "evt-retry"}, Attempts: 0},
		{Event: OutboxEvent{ID: "evt-terminal"}, Attempts: 4},
	}}
	publisher := &fakePublisher{fail: map[string]error{
		"evt-retry":    errors.New("temporary broker failure"),
		"evt-terminal": errors.New("credential detail must not be persisted"),
	}}
	dispatcher := NewDispatcher(store, publisher, func() time.Time { return now }, DefaultDispatcherConfig())

	result, err := dispatcher.RunBatch(context.Background())
	if err != nil {
		t.Fatalf("run batch: %v", err)
	}
	if result.Failed != 2 || len(store.failures) != 2 {
		t.Fatalf("unexpected failure result %#v / %#v", result, store.failures)
	}
	if store.failures[0].Attempts != 1 || store.failures[0].Terminal || !store.failures[0].RetryAt.Equal(now.Add(time.Second)) {
		t.Fatalf("unexpected retry failure %#v", store.failures[0])
	}
	if store.failures[1].Attempts != 5 || !store.failures[1].Terminal {
		t.Fatalf("unexpected terminal failure %#v", store.failures[1])
	}
	for _, failure := range store.failures {
		if failure.Reason != "publish_failed" {
			t.Fatalf("unsafe failure reason persisted: %q", failure.Reason)
		}
	}
}

func TestDispatcherReturnsClaimFailure(t *testing.T) {
	t.Parallel()

	claimErr := errors.New("store unavailable")
	store := &fakeOutboxStore{claimErr: claimErr}
	dispatcher := NewDispatcher(store, &fakePublisher{}, time.Now, DefaultDispatcherConfig())
	if _, err := dispatcher.RunBatch(context.Background()); !errors.Is(err, claimErr) {
		t.Fatalf("expected claim error, got %v", err)
	}
}

func TestDispatcherReturnsOutcomePersistenceFailures(t *testing.T) {
	t.Parallel()

	markErr := errors.New("mark failed")
	store := &fakeOutboxStore{events: []PendingEvent{{Event: OutboxEvent{ID: "evt"}}}, markErr: markErr}
	dispatcher := NewDispatcher(store, &fakePublisher{fail: map[string]error{}}, time.Now, DefaultDispatcherConfig())
	if _, err := dispatcher.RunBatch(context.Background()); !errors.Is(err, markErr) {
		t.Fatalf("expected mark error, got %v", err)
	}

	recordErr := errors.New("record failed")
	store = &fakeOutboxStore{events: []PendingEvent{{Event: OutboxEvent{ID: "evt"}}}, recordErr: recordErr}
	dispatcher = NewDispatcher(store, &fakePublisher{fail: map[string]error{"evt": errors.New("publish failed")}}, time.Now, DefaultDispatcherConfig())
	if _, err := dispatcher.RunBatch(context.Background()); !errors.Is(err, recordErr) {
		t.Fatalf("expected record error, got %v", err)
	}
}
