package app

import (
	"context"
	"fmt"
	"time"
)

type PendingEvent struct {
	Event    OutboxEvent
	Attempts int
}

type PublishFailure struct {
	EventID  string
	Attempts int
	RetryAt  time.Time
	Terminal bool
	Reason   string
}

type OutboxStore interface {
	ClaimPending(context.Context, time.Time, time.Duration, int) ([]PendingEvent, error)
	MarkPublished(context.Context, string, time.Time) error
	RecordFailure(context.Context, PublishFailure) error
}

type Publisher interface {
	Publish(context.Context, OutboxEvent) error
}

type DispatcherConfig struct {
	Lease       time.Duration
	BatchSize   int
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func DefaultDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		Lease: 30 * time.Second, BatchSize: 10, MaxAttempts: 5,
		BaseDelay: time.Second, MaxDelay: time.Minute,
	}
}

type DispatchResult struct {
	Claimed   int
	Published int
	Failed    int
}

type Dispatcher struct {
	store     OutboxStore
	publisher Publisher
	now       func() time.Time
	config    DispatcherConfig
}

func NewDispatcher(store OutboxStore, publisher Publisher, now func() time.Time, config DispatcherConfig) *Dispatcher {
	return &Dispatcher{store: store, publisher: publisher, now: now, config: config}
}

func (dispatcher *Dispatcher) RunBatch(ctx context.Context) (DispatchResult, error) {
	now := dispatcher.now().UTC()
	events, err := dispatcher.store.ClaimPending(ctx, now, dispatcher.config.Lease, dispatcher.config.BatchSize)
	if err != nil {
		return DispatchResult{}, fmt.Errorf("claim pending outbox events: %w", err)
	}
	result := DispatchResult{Claimed: len(events)}
	for _, pending := range events {
		if err := dispatcher.publisher.Publish(ctx, pending.Event); err == nil {
			if markErr := dispatcher.store.MarkPublished(ctx, pending.Event.ID, now); markErr != nil {
				return result, fmt.Errorf("mark event %s published: %w", pending.Event.ID, markErr)
			}
			result.Published++
			continue
		}

		attempts := pending.Attempts + 1
		failure := PublishFailure{
			EventID: pending.Event.ID, Attempts: attempts,
			Terminal: attempts >= dispatcher.config.MaxAttempts,
			Reason:   "publish_failed",
		}
		if !failure.Terminal {
			failure.RetryAt = now.Add(dispatcher.retryDelay(attempts))
		}
		if recordErr := dispatcher.store.RecordFailure(ctx, failure); recordErr != nil {
			return result, fmt.Errorf("record event %s failure: %w", pending.Event.ID, recordErr)
		}
		result.Failed++
	}
	return result, nil
}

func (dispatcher *Dispatcher) retryDelay(attempt int) time.Duration {
	delay := dispatcher.config.BaseDelay
	for current := 1; current < attempt && delay < dispatcher.config.MaxDelay; current++ {
		delay *= 2
		if delay > dispatcher.config.MaxDelay {
			return dispatcher.config.MaxDelay
		}
	}
	return delay
}
