package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
)

func postgresAcceptance() app.Acceptance {
	transaction := domain.Transaction{
		ID: "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01", PartnerID: "partner-west",
		OccurredAt: time.Date(2026, 8, 16, 20, 30, 0, 0, time.UTC), AmountMinor: 725,
		Currency: "USD", AgencyID: "agency-17", PlazaID: "plaza-4", LaneID: "lane-2", VehicleClass: domain.VehicleClassCar,
	}
	fingerprint, _ := transaction.Fingerprint()
	return app.Acceptance{Transaction: transaction, Fingerprint: fingerprint, Event: app.OutboxEvent{
		ID: "evt-1", Type: app.ReviewCandidateEventType, SchemaVersion: 1, OccurredAt: time.Date(2026, 8, 16, 22, 0, 0, 0, time.UTC),
		PartnerID: transaction.PartnerID, TransactionID: transaction.ID, Key: transaction.PartnerID + ":" + transaction.ID, Payload: transaction,
	}}
}

func TestStoreAcceptsTransactionAndOutboxInOneSQLTransaction(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	acceptance := postgresAcceptance()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(selectIdentitySQL)).WithArgs(acceptance.Transaction.PartnerID, acceptance.Transaction.ID).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta(insertTransactionSQL)).WithArgs(
		acceptance.Transaction.PartnerID, acceptance.Transaction.ID, acceptance.Fingerprint, sqlmock.AnyArg(), acceptance.Event.ID,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(insertOutboxSQL)).WithArgs(acceptance.Event.ID, sqlmock.AnyArg(), acceptance.Event.OccurredAt).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := NewStore(database).Accept(context.Background(), acceptance)
	if err != nil || result.Kind != app.StoreAccepted || result.EventID != "evt-1" {
		t.Fatalf("unexpected accept result %#v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreReturnsReplayOrConflictFromLockedIdentity(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name        string
		fingerprint string
		want        app.StoreOutcomeKind
	}{
		{name: "replay", fingerprint: postgresAcceptance().Fingerprint, want: app.StoreReplay},
		{name: "conflict", fingerprint: "different", want: app.StoreConflict},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			database, mock, _ := sqlmock.New()
			defer database.Close()
			acceptance := postgresAcceptance()
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(selectIdentitySQL)).WithArgs(acceptance.Transaction.PartnerID, acceptance.Transaction.ID).
				WillReturnRows(sqlmock.NewRows([]string{"fingerprint", "event_id"}).AddRow(test.fingerprint, "evt-original"))
			mock.ExpectCommit()
			result, err := NewStore(database).Accept(context.Background(), acceptance)
			if err != nil || result.Kind != test.want {
				t.Fatalf("unexpected result %#v, %v", result, err)
			}
			if test.want == app.StoreReplay && result.EventID != "evt-original" {
				t.Fatalf("expected original event, got %q", result.EventID)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStoreRollsBackWhenOutboxInsertFails(t *testing.T) {
	t.Parallel()

	database, mock, _ := sqlmock.New()
	defer database.Close()
	acceptance := postgresAcceptance()
	insertErr := errors.New("insert failed")
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(selectIdentitySQL)).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta(insertTransactionSQL)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(insertOutboxSQL)).WillReturnError(insertErr)
	mock.ExpectRollback()
	if _, err := NewStore(database).Accept(context.Background(), acceptance); !errors.Is(err, insertErr) {
		t.Fatalf("expected insert error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreClaimsOutboxWithSkipLockedLease(t *testing.T) {
	t.Parallel()

	database, mock, _ := sqlmock.New()
	defer database.Close()
	now := time.Date(2026, 8, 16, 23, 0, 0, 0, time.UTC)
	payload, _ := json.Marshal(postgresAcceptance().Event)
	mock.ExpectQuery(regexp.QuoteMeta(claimPendingSQL)).WithArgs(now, 10, now.Add(30*time.Second)).
		WillReturnRows(sqlmock.NewRows([]string{"event_payload", "attempts"}).AddRow(payload, 2))
	events, err := NewStore(database).ClaimPending(context.Background(), now, 30*time.Second, 10)
	if err != nil || len(events) != 1 || events[0].Event.ID != "evt-1" || events[0].Attempts != 2 {
		t.Fatalf("unexpected claim %#v, %v", events, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreRecordsPublishedAndFailedOutcomes(t *testing.T) {
	t.Parallel()

	database, mock, _ := sqlmock.New()
	defer database.Close()
	now := time.Date(2026, 8, 16, 23, 0, 0, 0, time.UTC)
	mock.ExpectExec(regexp.QuoteMeta(markPublishedSQL)).WithArgs(now, "evt-1").WillReturnResult(sqlmock.NewResult(0, 1))
	store := NewStore(database)
	if err := store.MarkPublished(context.Background(), "evt-1", now); err != nil {
		t.Fatalf("mark published: %v", err)
	}
	failure := app.PublishFailure{EventID: "evt-2", Attempts: 3, RetryAt: now.Add(time.Minute), Reason: "publish_failed"}
	mock.ExpectExec(regexp.QuoteMeta(recordFailureSQL)).WithArgs("pending", failure.Attempts, failure.RetryAt, failure.Reason, failure.EventID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := store.RecordFailure(context.Background(), failure); err != nil {
		t.Fatalf("record failure: %v", err)
	}
	terminal := app.PublishFailure{EventID: "evt-3", Attempts: 5, Terminal: true, Reason: "publish_failed"}
	mock.ExpectExec(regexp.QuoteMeta(recordFailureSQL)).WithArgs("failed", terminal.Attempts, terminal.RetryAt, terminal.Reason, terminal.EventID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := store.RecordFailure(context.Background(), terminal); err != nil {
		t.Fatalf("record terminal failure: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSchemaDefinesAtomicIdentityAndOutboxLeaseState(t *testing.T) {
	t.Parallel()

	payload, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	schema := string(payload)
	for _, required := range []string{
		"PRIMARY KEY (partner_id, transaction_id)", "payload JSONB", "event_payload JSONB",
		"lease_until TIMESTAMPTZ", "retry_at TIMESTAMPTZ", "CREATE INDEX outbox_dispatch_idx",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("schema missing %q", required)
		}
	}
}
