package ndjson

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	published  bool
	terminal   bool
}

type logRecord struct {
	Kind       string              `json:"kind"`
	Acceptance *app.Acceptance     `json:"acceptance,omitempty"`
	Failure    *app.PublishFailure `json:"failure,omitempty"`
	EventID    string              `json:"eventId,omitempty"`
	RecordedAt time.Time           `json:"recordedAt,omitempty"`
}

type Store struct {
	mu           sync.Mutex
	path         string
	transactions map[string]storedTransaction
	events       map[string]*storedEvent
}

func NewStore(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("NDJSON store path is required")
	}
	store := &Store{path: path, transactions: make(map[string]storedTransaction), events: make(map[string]*storedEvent)}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open NDJSON store: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		var record logRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, fmt.Errorf("decode NDJSON record %d: %w", line, err)
		}
		if err := store.apply(record); err != nil {
			return nil, fmt.Errorf("apply NDJSON record %d: %w", line, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan NDJSON store: %w", err)
	}
	return store, nil
}

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
	record := logRecord{Kind: "accepted", Acceptance: &acceptance}
	if err := store.append(record); err != nil {
		return app.StoreOutcome{}, err
	}
	if err := store.apply(record); err != nil {
		return app.StoreOutcome{}, err
	}
	return app.StoreOutcome{Kind: app.StoreAccepted, EventID: acceptance.Event.ID}, nil
}

func (store *Store) ClaimPending(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]app.PendingEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
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
		event := store.events[id]
		if event.published || event.terminal || event.retryAt.After(now) || event.leaseUntil.After(now) {
			continue
		}
		event.leaseUntil = now.Add(lease)
		claimed = append(claimed, app.PendingEvent{Event: event.event, Attempts: event.attempts})
		if len(claimed) == limit {
			break
		}
	}
	return claimed, nil
}

func (store *Store) MarkPublished(ctx context.Context, eventID string, at time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, found := store.events[eventID]; !found {
		return fmt.Errorf("outbox event %q not found", eventID)
	}
	record := logRecord{Kind: "published", EventID: eventID, RecordedAt: at}
	if err := store.append(record); err != nil {
		return err
	}
	return store.apply(record)
}

func (store *Store) RecordFailure(ctx context.Context, failure app.PublishFailure) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, found := store.events[failure.EventID]; !found {
		return fmt.Errorf("outbox event %q not found", failure.EventID)
	}
	record := logRecord{Kind: "failed", Failure: &failure}
	if err := store.append(record); err != nil {
		return err
	}
	return store.apply(record)
}

func (store *Store) append(record logRecord) error {
	file, err := os.OpenFile(store.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open NDJSON append log: %w", err)
	}
	defer file.Close()
	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("secure NDJSON append log: %w", err)
	}
	if err := json.NewEncoder(file).Encode(record); err != nil {
		return fmt.Errorf("append NDJSON record: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync NDJSON record: %w", err)
	}
	return nil
}

func (store *Store) apply(record logRecord) error {
	switch record.Kind {
	case "accepted":
		if record.Acceptance == nil {
			return fmt.Errorf("accepted record has no acceptance")
		}
		acceptance := *record.Acceptance
		key := acceptance.Transaction.PartnerID + ":" + acceptance.Transaction.ID
		store.transactions[key] = storedTransaction{fingerprint: acceptance.Fingerprint, eventID: acceptance.Event.ID}
		store.events[acceptance.Event.ID] = &storedEvent{event: acceptance.Event}
	case "failed":
		if record.Failure == nil {
			return fmt.Errorf("failed record has no failure")
		}
		event, found := store.events[record.Failure.EventID]
		if !found {
			return fmt.Errorf("failure references unknown event %q", record.Failure.EventID)
		}
		event.attempts, event.retryAt, event.terminal = record.Failure.Attempts, record.Failure.RetryAt, record.Failure.Terminal
		event.leaseUntil = time.Time{}
	case "published":
		event, found := store.events[record.EventID]
		if !found {
			return fmt.Errorf("publication references unknown event %q", record.EventID)
		}
		event.published = true
		event.leaseUntil = time.Time{}
	default:
		return fmt.Errorf("unknown record kind %q", record.Kind)
	}
	return nil
}
