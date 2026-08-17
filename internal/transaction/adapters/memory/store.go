package memory

import (
	"context"
	"crypto/rand"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
)

type storedTransaction struct {
	fingerprint string
	eventID     string
}

type storedEvent struct {
	event      app.OutboxEvent
	attempts   int
	retryAt    time.Time
	leaseUntil time.Time
	claimToken string
	published  bool
	terminal   bool
}

type Store struct {
	mu           sync.Mutex
	transactions map[string]storedTransaction
	events       map[string]*storedEvent
}

func NewStore() *Store {
	return &Store{
		transactions: make(map[string]storedTransaction),
		events:       make(map[string]*storedEvent),
	}
}

func (store *Store) Ready(ctx context.Context) error { return ctx.Err() }

func (store *Store) Accept(ctx context.Context, acceptance app.Acceptance) (app.StoreOutcome, error) {
	if err := ctx.Err(); err != nil {
		return app.StoreOutcome{}, err
	}

	key := acceptance.Transaction.PartnerID + ":" + acceptance.Transaction.ID
	store.mu.Lock()
	defer store.mu.Unlock()

	if existing, found := store.transactions[key]; found {
		if existing.fingerprint == acceptance.Fingerprint {
			return app.StoreOutcome{Kind: app.StoreReplay, EventID: existing.eventID}, nil
		}
		return app.StoreOutcome{Kind: app.StoreConflict}, nil
	}

	store.transactions[key] = storedTransaction{
		fingerprint: acceptance.Fingerprint,
		eventID:     acceptance.Event.ID,
	}
	store.events[acceptance.Event.ID] = &storedEvent{event: acceptance.Event}
	return app.StoreOutcome{Kind: app.StoreAccepted, EventID: acceptance.Event.ID}, nil
}

func (store *Store) ClaimPending(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]app.PendingEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		return nil, nil
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	ids := make([]string, 0, len(store.events))
	for id := range store.events {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	claimed := make([]app.PendingEvent, 0, limit)
	for _, id := range ids {
		record := store.events[id]
		if record.published || record.terminal || record.retryAt.After(now) || record.leaseUntil.After(now) {
			continue
		}
		record.leaseUntil = now.Add(lease)
		record.claimToken = rand.Text()
		claimed = append(claimed, app.PendingEvent{Event: record.event, Attempts: record.attempts, ClaimToken: record.claimToken})
		if len(claimed) == limit {
			break
		}
	}
	return claimed, nil
}

func (store *Store) MarkPublished(ctx context.Context, eventID, claimToken string, _ time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	record, found := store.events[eventID]
	if !found {
		return fmt.Errorf("outbox event %q not found", eventID)
	}
	if record.claimToken == "" || record.claimToken != claimToken {
		return app.ErrLeaseLost
	}
	record.published = true
	record.leaseUntil = time.Time{}
	record.claimToken = ""
	return nil
}

func (store *Store) RecordFailure(ctx context.Context, failure app.PublishFailure) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	record, found := store.events[failure.EventID]
	if !found {
		return fmt.Errorf("outbox event %q not found", failure.EventID)
	}
	if record.claimToken == "" || record.claimToken != failure.ClaimToken {
		return app.ErrLeaseLost
	}
	record.attempts = failure.Attempts
	record.retryAt = failure.RetryAt
	record.terminal = failure.Terminal
	record.leaseUntil = time.Time{}
	record.claimToken = ""
	return nil
}
