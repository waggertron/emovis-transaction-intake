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

type fakeSQLStateError string

func (err fakeSQLStateError) Error() string    { return string(err) }
func (err fakeSQLStateError) SQLState() string { return "23505" }

func postgresAcceptance() app.Acceptance {
	transaction := domain.Transaction{
		ID: "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01", Source: "partner-west", SourceReference: "source-ref",
		TransactionType: "toll", TransactionTimeUTC: time.Date(2026, 8, 16, 20, 30, 0, 0, time.UTC),
		BaseAmount: "7.25", Currency: "USD", TransponderNumber: "tag",
		LocationRaw: json.RawMessage(`{ "lane" : 9007199254740993 }`), MetadataRaw: json.RawMessage(`{ "rate" : 12.50 }`),
	}
	fingerprint, _ := transaction.Fingerprint()
	return app.Acceptance{Transaction: transaction, Fingerprint: fingerprint, Event: app.OutboxEvent{
		ID: "evt-1", Type: app.ReviewCandidateEventType, SchemaVersion: 1, OccurredAt: time.Date(2026, 8, 16, 22, 0, 0, 0, time.UTC),
		Source: transaction.Source, SourceReference: transaction.SourceReference, TransactionID: transaction.ID, Key: transaction.Source + ":" + transaction.SourceReference, Payload: transaction,
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
	mock.ExpectQuery(regexp.QuoteMeta(selectIdentitySQL)).WithArgs(acceptance.Transaction.Source, acceptance.Transaction.SourceReference).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta(insertTransactionSQL)).WithArgs(
		acceptance.Transaction.ID, acceptance.Transaction.Source, acceptance.Transaction.SourceReference, acceptance.Fingerprint, sqlmock.AnyArg(), acceptance.Transaction.LocationRaw, acceptance.Transaction.MetadataRaw, acceptance.Event.ID,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(insertOutboxSQL)).WithArgs(acceptance.Event.ID, sqlmock.AnyArg(), acceptance.Event.Payload.LocationRaw, acceptance.Event.Payload.MetadataRaw, acceptance.Event.OccurredAt).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := NewStore(database).Accept(context.Background(), acceptance)
	if err != nil || result.Kind != app.StoreAccepted || result.EventID != "evt-1" || result.TransactionID != acceptance.Transaction.ID {
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
			mock.ExpectQuery(regexp.QuoteMeta(selectIdentitySQL)).WithArgs(acceptance.Transaction.Source, acceptance.Transaction.SourceReference).
				WillReturnRows(sqlmock.NewRows([]string{"id", "fingerprint", "event_id"}).AddRow("transaction-original", test.fingerprint, "evt-original"))
			mock.ExpectCommit()
			result, err := NewStore(database).Accept(context.Background(), acceptance)
			if err != nil || result.Kind != test.want {
				t.Fatalf("unexpected result %#v, %v", result, err)
			}
			if test.want == app.StoreReplay && (result.EventID != "evt-original" || result.TransactionID != "transaction-original") {
				t.Fatalf("expected original identity, got %#v", result)
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

func TestStoreReclassifiesConcurrentUniqueRaceAsReplay(t *testing.T) {
	t.Parallel()

	database, mock, _ := sqlmock.New()
	defer database.Close()
	acceptance := postgresAcceptance()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(selectIdentitySQL)).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta(insertTransactionSQL)).WillReturnError(fakeSQLStateError("duplicate identity"))
	mock.ExpectRollback()
	mock.ExpectQuery(regexp.QuoteMeta(readIdentitySQL)).WithArgs(acceptance.Transaction.Source, acceptance.Transaction.SourceReference).
		WillReturnRows(sqlmock.NewRows([]string{"id", "fingerprint", "event_id"}).AddRow("transaction-winner", acceptance.Fingerprint, "evt-winner"))

	result, err := NewStore(database).Accept(context.Background(), acceptance)
	if err != nil || result.Kind != app.StoreReplay || result.EventID != "evt-winner" || result.TransactionID != "transaction-winner" {
		t.Fatalf("unique race was not reclassified: %#v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreConcurrencyClassificationConflictAndFailure(t *testing.T) {
	t.Parallel()
	acceptance := postgresAcceptance()
	for _, test := range []struct {
		name        string
		fingerprint string
		queryErr    error
		wantKind    app.StoreOutcomeKind
	}{
		{name: "conflict", fingerprint: "different", wantKind: app.StoreConflict},
		{name: "reread failure", queryErr: errors.New("read unavailable")},
	} {
		t.Run(test.name, func(t *testing.T) {
			database, mock, _ := sqlmock.New()
			defer database.Close()
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(selectIdentitySQL)).WillReturnError(sql.ErrNoRows)
			mock.ExpectExec(regexp.QuoteMeta(insertTransactionSQL)).WillReturnError(fakeSQLStateError("duplicate identity"))
			mock.ExpectRollback()
			query := mock.ExpectQuery(regexp.QuoteMeta(readIdentitySQL)).WithArgs(acceptance.Transaction.Source, acceptance.Transaction.SourceReference)
			if test.queryErr != nil {
				query.WillReturnError(test.queryErr)
			} else {
				query.WillReturnRows(sqlmock.NewRows([]string{"id", "fingerprint", "event_id"}).AddRow("transaction-winner", test.fingerprint, "evt-winner"))
			}
			result, err := NewStore(database).Accept(context.Background(), acceptance)
			if test.queryErr != nil && err == nil {
				t.Fatal("expected classification read failure")
			}
			if test.queryErr == nil && (err != nil || result.Kind != test.wantKind) {
				t.Fatalf("classification result: %#v, %v", result, err)
			}
		})
	}
	if isConcurrencyConflict(errors.New("ordinary")) {
		t.Fatal("ordinary errors must not be concurrency conflicts")
	}
}

func TestStoreReadinessPingsDatabase(t *testing.T) {
	t.Parallel()
	database, mock, _ := sqlmock.New(sqlmock.MonitorPingsOption(true))
	defer database.Close()
	mock.ExpectPing()
	if err := NewStore(database).Ready(context.Background()); err != nil {
		t.Fatalf("ready: %v", err)
	}
	mock.ExpectPing().WillReturnError(errors.New("unavailable"))
	if err := NewStore(database).Ready(context.Background()); err == nil {
		t.Fatal("expected readiness failure")
	}
}

func TestStoreClaimsOutboxWithSkipLockedLease(t *testing.T) {
	t.Parallel()

	database, mock, _ := sqlmock.New()
	defer database.Close()
	now := time.Date(2026, 8, 16, 23, 0, 0, 0, time.UTC)
	acceptance := postgresAcceptance()
	payload, _ := json.Marshal(acceptance.Event)
	mock.ExpectQuery(regexp.QuoteMeta(claimPendingSQL)).WithArgs(now, 10, now.Add(30*time.Second), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"event_payload", "location_raw", "metadata_raw", "attempts", "claim_token"}).AddRow(payload, acceptance.Event.Payload.LocationRaw, acceptance.Event.Payload.MetadataRaw, 2, "claim-1"))
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
	mock.ExpectExec(regexp.QuoteMeta(markPublishedSQL)).WithArgs(now, "evt-1", "claim-1").WillReturnResult(sqlmock.NewResult(0, 1))
	store := NewStore(database)
	if err := store.MarkPublished(context.Background(), "evt-1", "claim-1", now); err != nil {
		t.Fatalf("mark published: %v", err)
	}
	failure := app.PublishFailure{EventID: "evt-2", ClaimToken: "claim-2", Attempts: 3, RetryAt: now.Add(time.Minute), Reason: "publish_failed"}
	mock.ExpectExec(regexp.QuoteMeta(recordFailureSQL)).WithArgs("pending", failure.Attempts, failure.RetryAt, failure.Reason, failure.EventID, failure.ClaimToken).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := store.RecordFailure(context.Background(), failure); err != nil {
		t.Fatalf("record failure: %v", err)
	}
	terminal := app.PublishFailure{EventID: "evt-3", ClaimToken: "claim-3", Attempts: 5, Terminal: true, Reason: "publish_failed"}
	mock.ExpectExec(regexp.QuoteMeta(recordFailureSQL)).WithArgs("failed", terminal.Attempts, terminal.RetryAt, terminal.Reason, terminal.EventID, terminal.ClaimToken).
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
		"PRIMARY KEY (id)", "UNIQUE (source, source_reference)", "payload JSONB", "event_payload JSONB", "location_raw BYTEA", "metadata_raw BYTEA",
		"lease_until TIMESTAMPTZ", "claim_token TEXT", "ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS claim_token TEXT", "retry_at TIMESTAMPTZ", "CREATE INDEX IF NOT EXISTS outbox_dispatch_idx",
	} {
		if !strings.Contains(schema, required) {
			t.Fatalf("schema missing %q", required)
		}
	}
}

func TestStorePropagatesTransactionAndQueryFailures(t *testing.T) {
	t.Parallel()
	want := errors.New("database unavailable")
	acceptance := postgresAcceptance()

	database, mock, _ := sqlmock.New()
	mock.ExpectBegin().WillReturnError(want)
	if _, err := NewStore(database).Accept(context.Background(), acceptance); !errors.Is(err, want) {
		t.Fatalf("begin: %v", err)
	}
	database.Close()

	database, mock, _ = sqlmock.New()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(selectIdentitySQL)).WillReturnError(want)
	mock.ExpectRollback()
	if _, err := NewStore(database).Accept(context.Background(), acceptance); !errors.Is(err, want) {
		t.Fatalf("lookup: %v", err)
	}
	database.Close()

	database, mock, _ = sqlmock.New()
	mock.ExpectQuery(regexp.QuoteMeta(claimPendingSQL)).WillReturnError(want)
	if _, err := NewStore(database).ClaimPending(context.Background(), time.Now(), time.Second, 1); !errors.Is(err, want) {
		t.Fatalf("claim: %v", err)
	}
	database.Close()

	database, mock, _ = sqlmock.New()
	mock.ExpectQuery(regexp.QuoteMeta(claimPendingSQL)).WillReturnRows(sqlmock.NewRows([]string{"event_payload", "attempts", "claim_token"}).AddRow("bad-json", 0, "claim"))
	if _, err := NewStore(database).ClaimPending(context.Background(), time.Now(), time.Second, 1); err == nil {
		t.Fatal("expected payload decode failure")
	}
	database.Close()
}

func TestStorePropagatesOutcomeAndAffectedRowFailures(t *testing.T) {
	t.Parallel()
	want := errors.New("update unavailable")
	database, mock, _ := sqlmock.New()
	store := NewStore(database)
	mock.ExpectExec(regexp.QuoteMeta(markPublishedSQL)).WillReturnError(want)
	if err := store.MarkPublished(context.Background(), "evt", "claim", time.Now()); !errors.Is(err, want) {
		t.Fatalf("publish: %v", err)
	}
	mock.ExpectExec(regexp.QuoteMeta(markPublishedSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
	if err := store.MarkPublished(context.Background(), "missing", "claim", time.Now()); err == nil {
		t.Fatal("expected missing event")
	}
	mock.ExpectExec(regexp.QuoteMeta(recordFailureSQL)).WillReturnError(want)
	if err := store.RecordFailure(context.Background(), app.PublishFailure{EventID: "evt"}); !errors.Is(err, want) {
		t.Fatalf("failure: %v", err)
	}
	mock.ExpectExec(regexp.QuoteMeta(markPublishedSQL)).WillReturnResult(sqlmock.NewErrorResult(want))
	if err := store.MarkPublished(context.Background(), "evt", "claim", time.Now()); !errors.Is(err, want) {
		t.Fatalf("rows affected: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	database.Close()
}

func TestMigrateAppliesEmbeddedSchemaAndWrapsFailure(t *testing.T) {
	t.Parallel()
	database, mock, _ := sqlmock.New()
	mock.ExpectBegin()
	mock.ExpectExec("pg_advisory_xact_lock").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS transactions").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	if err := Migrate(context.Background(), database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	database.Close()

	want := errors.New("migration unavailable")
	database, mock, _ = sqlmock.New()
	mock.ExpectBegin()
	mock.ExpectExec("pg_advisory_xact_lock").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS transactions").WillReturnError(want)
	mock.ExpectRollback()
	if err := Migrate(context.Background(), database); !errors.Is(err, want) {
		t.Fatalf("migration error: %v", err)
	}
	database.Close()
}
