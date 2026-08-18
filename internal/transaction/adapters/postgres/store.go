package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
)

const (
	selectIdentitySQL    = `SELECT id, fingerprint, event_id FROM transactions WHERE source = $1 AND source_reference = $2 FOR UPDATE`
	readIdentitySQL      = `SELECT id, fingerprint, event_id FROM transactions WHERE source = $1 AND source_reference = $2`
	insertTransactionSQL = `INSERT INTO transactions (id, source, source_reference, fingerprint, payload, location_raw, metadata_raw, event_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	insertOutboxSQL      = `INSERT INTO outbox_events (event_id, event_payload, location_raw, metadata_raw, occurred_at, status, attempts) VALUES ($1, $2, $3, $4, $5, 'pending', 0)`
	claimPendingSQL      = `WITH candidates AS (SELECT event_id FROM outbox_events WHERE status = 'pending' AND (retry_at IS NULL OR retry_at <= $1) AND (lease_until IS NULL OR lease_until <= $1) ORDER BY occurred_at FOR UPDATE SKIP LOCKED LIMIT $2) UPDATE outbox_events AS event SET lease_until = $3, claim_token = $4 FROM candidates WHERE event.event_id = candidates.event_id RETURNING event.event_payload, event.location_raw, event.metadata_raw, event.attempts, event.claim_token`
	markPublishedSQL     = `UPDATE outbox_events SET status = 'published', published_at = $1, lease_until = NULL, claim_token = NULL WHERE event_id = $2 AND status = 'pending' AND claim_token = $3`
	recordFailureSQL     = `UPDATE outbox_events SET status = $1, attempts = $2, retry_at = $3, last_error = $4, lease_until = NULL, claim_token = NULL WHERE event_id = $5 AND status = 'pending' AND claim_token = $6`
)

type Store struct {
	database *sql.DB
}

func NewStore(database *sql.DB) *Store {
	return &Store{database: database}
}

func (store *Store) Ready(ctx context.Context) error {
	if err := store.database.PingContext(ctx); err != nil {
		return fmt.Errorf("check PostgreSQL readiness: %w", err)
	}
	return nil
}

func (store *Store) Accept(ctx context.Context, acceptance app.Acceptance) (outcome app.StoreOutcome, err error) {
	source, sourceReference := acceptance.Transaction.Source, acceptance.Transaction.SourceReference
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

	var existingTransactionID, existingFingerprint, existingEventID string
	err = tx.QueryRowContext(ctx, selectIdentitySQL, source, sourceReference).
		Scan(&existingTransactionID, &existingFingerprint, &existingEventID)
	if err == nil {
		if commitErr := tx.Commit(); commitErr != nil {
			return app.StoreOutcome{}, fmt.Errorf("commit idempotency lookup: %w", commitErr)
		}
		if existingFingerprint == acceptance.Fingerprint {
			return app.StoreOutcome{Kind: app.StoreReplay, TransactionID: existingTransactionID, EventID: existingEventID}, nil
		}
		return app.StoreOutcome{Kind: app.StoreConflict}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return app.StoreOutcome{}, fmt.Errorf("lookup transaction identity: %w", err)
	}

	if _, err = tx.ExecContext(ctx, insertTransactionSQL,
		acceptance.Transaction.ID, source, sourceReference, acceptance.Fingerprint,
		transactionPayload, []byte(acceptance.Transaction.LocationRaw), []byte(acceptance.Transaction.MetadataRaw), acceptance.Event.ID,
	); err != nil {
		if isConcurrencyConflict(err) {
			_ = tx.Rollback()
			return store.classifyExisting(ctx, acceptance, err)
		}
		return app.StoreOutcome{}, fmt.Errorf("insert transaction: %w", err)
	}
	if _, err = tx.ExecContext(ctx, insertOutboxSQL, acceptance.Event.ID, eventPayload,
		[]byte(acceptance.Event.Payload.LocationRaw), []byte(acceptance.Event.Payload.MetadataRaw), acceptance.Event.OccurredAt); err != nil {
		return app.StoreOutcome{}, fmt.Errorf("insert outbox event: %w", err)
	}
	if err = tx.Commit(); err != nil {
		if isConcurrencyConflict(err) {
			return store.classifyExisting(ctx, acceptance, err)
		}
		return app.StoreOutcome{}, fmt.Errorf("commit transaction acceptance: %w", err)
	}
	return app.StoreOutcome{Kind: app.StoreAccepted, TransactionID: acceptance.Transaction.ID, EventID: acceptance.Event.ID}, nil
}

type sqlStateError interface {
	SQLState() string
}

func isConcurrencyConflict(err error) bool {
	var state sqlStateError
	if !errors.As(err, &state) {
		return false
	}
	return state.SQLState() == "23505" || state.SQLState() == "40001"
}

func (store *Store) classifyExisting(ctx context.Context, acceptance app.Acceptance, cause error) (app.StoreOutcome, error) {
	source, sourceReference := acceptance.Transaction.Source, acceptance.Transaction.SourceReference
	var transactionID, fingerprint, eventID string
	err := store.database.QueryRowContext(ctx, readIdentitySQL, source, sourceReference).Scan(&transactionID, &fingerprint, &eventID)
	if err != nil {
		return app.StoreOutcome{}, fmt.Errorf("classify concurrent transaction after %v: %w", cause, err)
	}
	if fingerprint == acceptance.Fingerprint {
		return app.StoreOutcome{Kind: app.StoreReplay, TransactionID: transactionID, EventID: eventID}, nil
	}
	return app.StoreOutcome{Kind: app.StoreConflict}, nil
}

func (store *Store) ClaimPending(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]app.PendingEvent, error) {
	claimToken := rand.Text()
	rows, err := store.database.QueryContext(ctx, claimPendingSQL, now, limit, now.Add(lease), claimToken)
	if err != nil {
		return nil, fmt.Errorf("claim pending outbox events: %w", err)
	}
	defer rows.Close()
	events := make([]app.PendingEvent, 0, limit)
	for rows.Next() {
		var payload, locationRaw, metadataRaw []byte
		var pending app.PendingEvent
		if err := rows.Scan(&payload, &locationRaw, &metadataRaw, &pending.Attempts, &pending.ClaimToken); err != nil {
			return nil, fmt.Errorf("scan claimed outbox event: %w", err)
		}
		if err := json.Unmarshal(payload, &pending.Event); err != nil {
			return nil, fmt.Errorf("decode claimed outbox event: %w", err)
		}
		if locationRaw != nil {
			pending.Event.Payload.LocationRaw = append([]byte(nil), locationRaw...)
		}
		if metadataRaw != nil {
			pending.Event.Payload.MetadataRaw = append([]byte(nil), metadataRaw...)
		}
		events = append(events, pending)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed outbox events: %w", err)
	}
	return events, nil
}

func (store *Store) MarkPublished(ctx context.Context, eventID, claimToken string, at time.Time) error {
	result, err := store.database.ExecContext(ctx, markPublishedSQL, at, eventID, claimToken)
	if err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}
	return requireClaimUpdated(result, eventID)
}

func (store *Store) RecordFailure(ctx context.Context, failure app.PublishFailure) error {
	status := "pending"
	if failure.Terminal {
		status = "failed"
	}
	result, err := store.database.ExecContext(ctx, recordFailureSQL, status, failure.Attempts, failure.RetryAt, failure.Reason, failure.EventID, failure.ClaimToken)
	if err != nil {
		return fmt.Errorf("record outbox event failure: %w", err)
	}
	return requireClaimUpdated(result, failure.EventID)
}

func requireClaimUpdated(result sql.Result, eventID string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read outbox update result: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("outbox event %q: %w", eventID, app.ErrLeaseLost)
	}
	return nil
}
