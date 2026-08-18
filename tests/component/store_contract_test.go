package component_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssdk "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	_ "github.com/jackc/pgx/v5/stdlib"
	dynamoadapter "github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/dynamodb"
	kafkaadapter "github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/kafka"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/memory"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/ndjson"
	postgresadapter "github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/postgres"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/domain"
)

func TestKafkaPublisherReportsUnavailableBroker(t *testing.T) {
	if os.Getenv("KAFKA_COMPONENT_UNAVAILABLE") == "" {
		t.Skip("KAFKA_COMPONENT_UNAVAILABLE is required for the Kafka failure component")
	}
	writer, err := kafkaadapter.NewWriter(kafkaadapter.WriterConfig{
		Brokers: []string{"127.0.0.1:1"}, Topic: "transaction.review-candidates.v1",
	})
	if err != nil {
		t.Fatalf("new production Kafka writer: %v", err)
	}
	defer writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := kafkaadapter.NewPublisher(writer, "transaction.review-candidates.v1").Publish(ctx, contractAcceptance("evt-kafka-unavailable").Event); err == nil {
		t.Fatal("expected unavailable broker failure")
	}
}

type storeFactory func(*testing.T) app.TransactionStore

func TestLocalStoresSatisfyTransactionStoreContract(t *testing.T) {
	t.Parallel()
	factories := map[string]storeFactory{
		"memory": func(*testing.T) app.TransactionStore { return memory.NewStore() },
		"ndjson": func(t *testing.T) app.TransactionStore {
			store, err := ndjson.NewStore(filepath.Join(t.TempDir(), "transactions.ndjson"))
			if err != nil {
				t.Fatalf("new NDJSON store: %v", err)
			}
			return store
		},
	}
	for name, factory := range factories {
		name, factory := name, factory
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runTransactionStoreContract(t, factory)
		})
	}
}

func TestDynamoDBLocalSatisfiesTransactionStoreContract(t *testing.T) {
	endpoint := os.Getenv("DYNAMODB_COMPONENT_ENDPOINT")
	if endpoint == "" {
		t.Skip("DYNAMODB_COMPONENT_ENDPOINT is required for the DynamoDB Local component contract")
	}
	client := awssdk.New(awssdk.Options{
		Region: "us-west-2", BaseEndpoint: aws.String(endpoint),
		Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: "local", SecretAccessKey: "local", Source: "component-test"}, nil
		}),
	})
	created := []string{}
	t.Cleanup(func() {
		for _, table := range created {
			_, _ = client.DeleteTable(context.Background(), &awssdk.DeleteTableInput{TableName: aws.String(table)})
		}
	})
	sequence := 0
	factory := func(t *testing.T) app.TransactionStore {
		sequence++
		table := fmt.Sprintf("transactions-contract-%d", sequence)
		if err := dynamoadapter.EnsureTable(context.Background(), client, table); err != nil {
			t.Fatalf("ensure DynamoDB table: %v", err)
		}
		created = append(created, table)
		return dynamoadapter.NewStore(client, table)
	}
	runTransactionStoreContract(t, factory)
	concurrent := factory(t)
	runConcurrentLeaseContract(t, concurrent, contractAcceptance("evt-dynamo-concurrent"))
	runConcurrentAcceptanceContract(t, factory(t))
	unavailable := awssdk.New(awssdk.Options{
		Region: "us-west-2", BaseEndpoint: aws.String("http://127.0.0.1:1"), RetryMaxAttempts: 1,
		HTTPClient: &http.Client{Timeout: 500 * time.Millisecond},
		Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: "local", SecretAccessKey: "local", Source: "component-test"}, nil
		}),
	})
	if _, err := dynamoadapter.NewStore(unavailable, "transactions").Accept(context.Background(), contractAcceptance("evt-dynamo-unavailable")); err == nil {
		t.Fatal("expected unavailable DynamoDB adapter failure")
	}
}

func TestPostgresSatisfiesTransactionStoreContract(t *testing.T) {
	databaseURL := os.Getenv("POSTGRES_COMPONENT_URL")
	if databaseURL == "" {
		t.Skip("POSTGRES_COMPONENT_URL is required for the PostgreSQL component contract")
	}
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := database.PingContext(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	if err := postgresadapter.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate PostgreSQL: %v", err)
	}
	constraintTx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin unique-constraint check: %v", err)
	}
	if _, err = constraintTx.ExecContext(ctx, `INSERT INTO transactions (id, source, source_reference, fingerprint, payload, event_id) VALUES ('constraint-id-1', 'constraint-source', 'constraint-ref', repeat('a', 64), '{}', 'constraint-event-1')`); err != nil {
		t.Fatalf("seed unique-constraint check: %v", err)
	}
	if _, err = constraintTx.ExecContext(ctx, `INSERT INTO transactions (id, source, source_reference, fingerprint, payload, event_id) VALUES ('constraint-id-2', 'constraint-source', 'constraint-ref', repeat('b', 64), '{}', 'constraint-event-2')`); err == nil {
		t.Fatal("expected partner/transaction unique constraint")
	}
	_ = constraintTx.Rollback()

	if _, err = database.ExecContext(ctx, `CREATE OR REPLACE FUNCTION reject_component_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'component rollback'; END $$; CREATE TRIGGER reject_component_outbox BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION reject_component_outbox()`); err != nil {
		t.Fatalf("install rollback trigger: %v", err)
	}
	rollbackAcceptance := contractAcceptance("evt-postgres-rollback")
	rollbackAcceptance.Transaction.ID = "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e11"
	rollbackAcceptance.Event.TransactionID = rollbackAcceptance.Transaction.ID
	rollbackAcceptance.Event.Key = rollbackAcceptance.Transaction.Source + ":" + rollbackAcceptance.Transaction.SourceReference
	rollbackAcceptance.Event.Payload = rollbackAcceptance.Transaction
	rollbackAcceptance.Fingerprint, _ = rollbackAcceptance.Transaction.Fingerprint()
	if _, err = postgresadapter.NewStore(database).Accept(ctx, rollbackAcceptance); err == nil {
		t.Fatal("expected real outbox insert failure")
	}
	if _, err = database.ExecContext(ctx, `DROP TRIGGER reject_component_outbox ON outbox_events; DROP FUNCTION reject_component_outbox()`); err != nil {
		t.Fatalf("remove rollback trigger: %v", err)
	}
	var rolledBack int
	if err = database.QueryRowContext(ctx, `SELECT count(*) FROM transactions WHERE id = $1`, rollbackAcceptance.Transaction.ID).Scan(&rolledBack); err != nil || rolledBack != 0 {
		t.Fatalf("transaction row survived outbox rollback: count=%d err=%v", rolledBack, err)
	}
	factory := func(t *testing.T) app.TransactionStore {
		if _, err := database.ExecContext(ctx, "TRUNCATE outbox_events, transactions"); err != nil {
			t.Fatalf("reset PostgreSQL: %v", err)
		}
		return postgresadapter.NewStore(database)
	}
	runTransactionStoreContract(t, factory)
	persistenceAcceptance := contractAcceptance("evt-postgres-persistence")
	persistenceAcceptance.Transaction.ID = "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e12"
	persistenceAcceptance.Transaction.SourceReference = "persistence-ref"
	persistenceAcceptance.Event.TransactionID = persistenceAcceptance.Transaction.ID
	persistenceAcceptance.Event.Key = persistenceAcceptance.Transaction.Source + ":" + persistenceAcceptance.Transaction.SourceReference
	persistenceAcceptance.Event.Payload = persistenceAcceptance.Transaction
	persistenceAcceptance.Fingerprint, _ = persistenceAcceptance.Transaction.Fingerprint()
	if _, err = postgresadapter.NewStore(database).Accept(ctx, persistenceAcceptance); err != nil {
		t.Fatalf("seed restart persistence: %v", err)
	}
	restartedDatabase, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("reopen PostgreSQL: %v", err)
	}
	defer restartedDatabase.Close()
	restarted, err := postgresadapter.NewStore(restartedDatabase).Accept(ctx, persistenceAcceptance)
	if err != nil || restarted.Kind != app.StoreReplay || restarted.EventID != persistenceAcceptance.Event.ID {
		t.Fatalf("restart replay: %#v %v", restarted, err)
	}
	concurrent := factory(t)
	runConcurrentLeaseContract(t, concurrent, contractAcceptance("evt-postgres-concurrent"))
	runConcurrentAcceptanceContract(t, factory(t))
	unavailable, err := sql.Open("pgx", "postgres://transaction_test:local-component-only@127.0.0.1:1/transactions?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("open unavailable PostgreSQL client: %v", err)
	}
	defer unavailable.Close()
	if _, err := postgresadapter.NewStore(unavailable).Accept(context.Background(), contractAcceptance("evt-postgres-unavailable")); err == nil {
		t.Fatal("expected unavailable PostgreSQL adapter failure")
	}
}

func runConcurrentLeaseContract(t *testing.T, store app.TransactionStore, acceptance app.Acceptance) {
	t.Helper()
	acceptance.Transaction.ID = "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e03"
	acceptance.Event.TransactionID = acceptance.Transaction.ID
	acceptance.Event.Key = acceptance.Transaction.Source + ":" + acceptance.Transaction.SourceReference
	acceptance.Event.Payload = acceptance.Transaction
	acceptance.Fingerprint, _ = acceptance.Transaction.Fingerprint()
	if _, err := store.Accept(context.Background(), acceptance); err != nil {
		t.Fatalf("accept concurrent event: %v", err)
	}
	now := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	start := make(chan struct{})
	counts := make(chan int, 2)
	errors := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			claimed, err := store.ClaimPending(context.Background(), now, 30*time.Second, 1)
			counts <- len(claimed)
			errors <- err
		}()
	}
	close(start)
	total := 0
	for range 2 {
		total += <-counts
		if err := <-errors; err != nil {
			t.Fatalf("concurrent claim: %v", err)
		}
	}
	if total != 1 {
		t.Fatalf("expected one concurrent lease winner, got %d", total)
	}
}

func runConcurrentAcceptanceContract(t *testing.T, store app.TransactionStore) {
	t.Helper()
	acceptance := contractAcceptance("evt-concurrent-accept")
	start := make(chan struct{})
	results := make(chan app.StoreOutcome, 2)
	errorsCh := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			result, err := store.Accept(context.Background(), acceptance)
			results <- result
			errorsCh <- err
		}()
	}
	close(start)
	accepted, replayed := 0, 0
	for range 2 {
		result, err := <-results, <-errorsCh
		if err != nil {
			t.Fatalf("concurrent identical acceptance: %v", err)
		}
		switch result.Kind {
		case app.StoreAccepted:
			accepted++
		case app.StoreReplay:
			replayed++
		default:
			t.Fatalf("unexpected concurrent outcome: %#v", result)
		}
	}
	if accepted != 1 || replayed != 1 {
		t.Fatalf("expected one accept and one replay, got accepted=%d replayed=%d", accepted, replayed)
	}
}

func runTransactionStoreContract(t *testing.T, factory storeFactory) {
	t.Helper()
	ctx := context.Background()
	store := factory(t)
	accepted := contractAcceptance("evt-contract")
	result, err := store.Accept(ctx, accepted)
	if err != nil || result.Kind != app.StoreAccepted || result.EventID != accepted.Event.ID || result.TransactionID != accepted.Transaction.ID {
		t.Fatalf("accept: %#v, %v", result, err)
	}
	replay := accepted
	replay.Event.ID = "evt-replacement"
	result, err = store.Accept(ctx, replay)
	if err != nil || result.Kind != app.StoreReplay || result.EventID != accepted.Event.ID || result.TransactionID != accepted.Transaction.ID {
		t.Fatalf("replay: %#v, %v", result, err)
	}
	conflict := accepted
	conflict.Fingerprint = "changed"
	result, err = store.Accept(ctx, conflict)
	if err != nil || result.Kind != app.StoreConflict {
		t.Fatalf("conflict: %#v, %v", result, err)
	}

	now := time.Date(2026, 8, 17, 7, 0, 0, 0, time.UTC)
	claimed, err := store.ClaimPending(ctx, now, 30*time.Second, 1)
	if err != nil || len(claimed) != 1 || claimed[0].Event.ID != accepted.Event.ID || claimed[0].Attempts != 0 {
		t.Fatalf("first claim: %#v, %v", claimed, err)
	}
	firstClaim := claimed[0]
	claimed, err = store.ClaimPending(ctx, now.Add(time.Second), 30*time.Second, 1)
	if err != nil || len(claimed) != 0 {
		t.Fatalf("active lease reclaimed: %#v, %v", claimed, err)
	}
	claimed, err = store.ClaimPending(ctx, now.Add(31*time.Second), 30*time.Second, 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("expired lease unavailable: %#v, %v", claimed, err)
	}
	if err := store.MarkPublished(ctx, accepted.Event.ID, firstClaim.ClaimToken, now.Add(31*time.Second)); !errors.Is(err, app.ErrLeaseLost) {
		t.Fatalf("stale publication should lose lease: %v", err)
	}
	if err := store.RecordFailure(ctx, app.PublishFailure{EventID: accepted.Event.ID, ClaimToken: firstClaim.ClaimToken, Attempts: 1}); !errors.Is(err, app.ErrLeaseLost) {
		t.Fatalf("stale failure should lose lease: %v", err)
	}

	retryAt := now.Add(time.Minute)
	if err := store.RecordFailure(ctx, app.PublishFailure{EventID: accepted.Event.ID, ClaimToken: claimed[0].ClaimToken, Attempts: 1, RetryAt: retryAt, Reason: "publish_failed"}); err != nil {
		t.Fatalf("record retry: %v", err)
	}
	claimed, _ = store.ClaimPending(ctx, retryAt.Add(-time.Nanosecond), 30*time.Second, 1)
	if len(claimed) != 0 {
		t.Fatalf("claimed before retry: %#v", claimed)
	}
	claimed, err = store.ClaimPending(ctx, retryAt, 30*time.Second, 1)
	if err != nil || len(claimed) != 1 || claimed[0].Attempts != 1 {
		t.Fatalf("retry claim: %#v, %v", claimed, err)
	}
	if err := store.MarkPublished(ctx, accepted.Event.ID, claimed[0].ClaimToken, retryAt); err != nil {
		t.Fatalf("publish: %v", err)
	}
	claimed, _ = store.ClaimPending(ctx, retryAt.Add(time.Hour), 30*time.Second, 1)
	if len(claimed) != 0 {
		t.Fatalf("published event reclaimed: %#v", claimed)
	}

	terminalStore := factory(t)
	terminal := contractAcceptance("evt-terminal")
	terminal.Transaction.ID = "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e02"
	terminal.Fingerprint, _ = terminal.Transaction.Fingerprint()
	if _, err := terminalStore.Accept(ctx, terminal); err != nil {
		t.Fatalf("accept terminal: %v", err)
	}
	terminalClaim, err := terminalStore.ClaimPending(ctx, now, 30*time.Second, 1)
	if err != nil || len(terminalClaim) != 1 {
		t.Fatalf("claim terminal: %#v, %v", terminalClaim, err)
	}
	if err := terminalStore.RecordFailure(ctx, app.PublishFailure{EventID: terminal.Event.ID, ClaimToken: terminalClaim[0].ClaimToken, Attempts: 5, Terminal: true, Reason: "publish_failed"}); err != nil {
		t.Fatalf("record terminal: %v", err)
	}
	claimed, _ = terminalStore.ClaimPending(ctx, now.Add(time.Hour), 30*time.Second, 1)
	if len(claimed) != 0 {
		t.Fatalf("terminal event reclaimed: %#v", claimed)
	}
}

func contractAcceptance(eventID string) app.Acceptance {
	transaction := domain.Transaction{ID: "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01", Source: "partner-contract", SourceReference: "source-ref", TransactionType: "toll", TransactionTimeUTC: time.Date(2026, 8, 17, 6, 0, 0, 0, time.UTC), BaseAmount: "7.25", Currency: "USD", TransponderNumber: "tag"}
	fingerprint, _ := transaction.Fingerprint()
	return app.Acceptance{Transaction: transaction, Fingerprint: fingerprint, Event: app.OutboxEvent{
		ID: eventID, Type: app.ReviewCandidateEventType, SchemaVersion: 1,
		OccurredAt: time.Date(2026, 8, 17, 6, 1, 0, 0, time.UTC), Source: transaction.Source, SourceReference: transaction.SourceReference,
		TransactionID: transaction.ID, Key: transaction.Source + ":" + transaction.SourceReference, Payload: transaction,
	}}
}
