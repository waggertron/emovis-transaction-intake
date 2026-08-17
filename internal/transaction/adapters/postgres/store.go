package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
)

const (
	selectIdentitySQL    = `SELECT fingerprint, event_id FROM transactions WHERE partner_id = $1 AND transaction_id = $2 FOR UPDATE`
	insertTransactionSQL = `INSERT INTO transactions (partner_id, transaction_id, fingerprint, payload, event_id) VALUES ($1, $2, $3, $4, $5)`
	insertOutboxSQL      = `INSERT INTO outbox_events (event_id, event_payload, occurred_at, status, attempts) VALUES ($1, $2, $3, 'pending', 0)`
	claimPendingSQL      = `WITH candidates AS (SELECT event_id FROM outbox_events WHERE status = 'pending' AND (retry_at IS NULL OR retry_at <= $1) AND (lease_until IS NULL OR lease_until <= $1) ORDER BY occurred_at FOR UPDATE SKIP LOCKED LIMIT $2) UPDATE outbox_events AS event SET lease_until = $3 FROM candidates WHERE event.event_id = candidates.event_id RETURNING event.event_payload, event.attempts`
	markPublishedSQL     = `UPDATE outbox_events SET status = 'published', published_at = $1, lease_until = NULL WHERE event_id = $2`
	recordFailureSQL     = `UPDATE outbox_events SET status = $1, attempts = $2, retry_at = $3, last_error = $4, lease_until = NULL WHERE event_id = $5`
)

type Store struct {
	database *sql.DB
}

func NewStore(database *sql.DB) *Store {
	return &Store{database: database}
}

func (store *Store) Accept(ctx context.Context, acceptance app.Acceptance) (outcome app.StoreOutcome, err error) {
	transactionPayload, err := json.Marshal(acceptance.Transaction)
	if err != nil {
		return app.StoreOutcome{}, fmt.Errorf("encode transaction: %w", err)
	}
	eventPayload, err := json.Marshal(acceptance.Event)
	if err != nil {
		return app.StoreOutcome{}, fmt.Errorf("encode outbox event: %w", err)
	}

	tx, err := store.database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return app.StoreOutcome{}, fmt.Errorf("begin transaction acceptance: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var existingFingerprint, existingEventID string
	err = tx.QueryRowContext(ctx, selectIdentitySQL, acceptance.Transaction.PartnerID, acceptance.Transaction.ID).
		Scan(&existingFingerprint, &existingEventID)
	if err == nil {
		if commitErr := tx.Commit(); commitErr != nil {
			return app.StoreOutcome{}, fmt.Errorf("commit idempotency lookup: %w", commitErr)
		}
		if existingFingerprint == acceptance.Fingerprint {
			return app.StoreOutcome{Kind: app.StoreReplay, EventID: existingEventID}, nil
		}
		return app.StoreOutcome{Kind: app.StoreConflict}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return app.StoreOutcome{}, fmt.Errorf("lookup transaction identity: %w", err)
	}

	if _, err = tx.ExecContext(ctx, insertTransactionSQL,
		acceptance.Transaction.PartnerID, acceptance.Transaction.ID, acceptance.Fingerprint,
		transactionPayload, acceptance.Event.ID,
	); err != nil {
		return app.StoreOutcome{}, fmt.Errorf("insert transaction: %w", err)
	}
	if _, err = tx.ExecContext(ctx, insertOutboxSQL, acceptance.Event.ID, eventPayload, acceptance.Event.OccurredAt); err != nil {
		return app.StoreOutcome{}, fmt.Errorf("insert outbox event: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return app.StoreOutcome{}, fmt.Errorf("commit transaction acceptance: %w", err)
	}
	return app.StoreOutcome{Kind: app.StoreAccepted, EventID: acceptance.Event.ID}, nil
}

func (store *Store) ClaimPending(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]app.PendingEvent, error) {
	rows, err := store.database.QueryContext(ctx, claimPendingSQL, now, limit, now.Add(lease))
	if err != nil {
		return nil, fmt.Errorf("claim pending outbox events: %w", err)
	}
	defer rows.Close()
	events := make([]app.PendingEvent, 0, limit)
	for rows.Next() {
		var payload []byte
		var pending app.PendingEvent
		if err := rows.Scan(&payload, &pending.Attempts); err != nil {
			return nil, fmt.Errorf("scan claimed outbox event: %w", err)
		}
		if err := json.Unmarshal(payload, &pending.Event); err != nil {
			return nil, fmt.Errorf("decode claimed outbox event: %w", err)
		}
		events = append(events, pending)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed outbox events: %w", err)
	}
	return events, nil
}

func (store *Store) MarkPublished(ctx context.Context, eventID string, at time.Time) error {
	result, err := store.database.ExecContext(ctx, markPublishedSQL, at, eventID)
	if err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}
	return requireUpdated(result, eventID)
}

func (store *Store) RecordFailure(ctx context.Context, failure app.PublishFailure) error {
	status := "pending"
	if failure.Terminal {
		status = "failed"
	}
	result, err := store.database.ExecContext(ctx, recordFailureSQL, status, failure.Attempts, failure.RetryAt, failure.Reason, failure.EventID)
	if err != nil {
		return fmt.Errorf("record outbox event failure: %w", err)
	}
	return requireUpdated(result, failure.EventID)
}

func requireUpdated(result sql.Result, eventID string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read outbox update result: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("outbox event %q not found", eventID)
	}
	return nil
}
