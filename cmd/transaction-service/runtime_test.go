package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/waggertron/emovis-transaction-intake/internal/bootstrap"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/memory"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/ndjson"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
)

func TestNewStoreSelectsMemory(t *testing.T) {
	t.Parallel()

	store, err := newStore(context.Background(), bootstrap.ModeLocal, bootstrap.Config{StoreDriver: "memory"})
	if err != nil {
		t.Fatalf("new memory store: %v", err)
	}
	defer store.Close()
	if _, ok := store.TransactionStore.(*memory.Store); !ok {
		t.Fatalf("expected memory store, got %T", store)
	}
}

func TestStoreHandleClosesOwnedResourceAndDynamoLocalConfigUsesStaticCredentials(t *testing.T) {
	t.Parallel()
	want := errors.New("close failed")
	store := &storeHandle{TransactionStore: memory.NewStore(), close: func() error { return want }}
	if err := store.Close(); !errors.Is(err, want) {
		t.Fatalf("close result: %v", err)
	}
	config, err := dynamoConfig(context.Background(), bootstrap.Config{DynamoEndpoint: "http://localhost:8000", DynamoRegion: "us-west-2"})
	if err != nil || config.Region != "us-west-2" || config.Credentials == nil {
		t.Fatalf("local Dynamo config: %#v %v", config, err)
	}
}

func TestDynamoCloudConfigUsesTheDefaultAWSCredentialChain(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "local-test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "local-test-secret-key")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	config, err := dynamoConfig(context.Background(), bootstrap.Config{DynamoRegion: "us-east-2"})
	if err != nil || config.Region != "us-east-2" || config.Credentials == nil {
		t.Fatalf("cloud Dynamo config: %#v %v", config, err)
	}
	credentials, err := config.Credentials.Retrieve(context.Background())
	if err != nil || credentials.AccessKeyID != "local-test-access-key" {
		t.Fatalf("default credentials: %#v %v", credentials, err)
	}
}

func TestProductionStartersExposeEveryMode(t *testing.T) {
	t.Parallel()
	starters := productionStarters()
	for _, mode := range []bootstrap.Mode{bootstrap.ModeAPI, bootstrap.ModeWorker, bootstrap.ModeLocal} {
		if starters[mode] == nil {
			t.Fatalf("missing starter for %s", mode)
		}
	}
}

func TestRuntimeCompositionRejectsInvalidDependencies(t *testing.T) {
	t.Parallel()
	invalidStore := bootstrap.Config{StoreDriver: "not-wired"}
	if err := startAPI(context.Background(), invalidStore); err == nil {
		t.Fatal("expected API store failure")
	}
	if err := startWorker(context.Background(), invalidStore); err == nil {
		t.Fatal("expected worker store failure")
	}
	if err := startLocal(context.Background(), invalidStore); err == nil {
		t.Fatal("expected local store failure")
	}

	invalidKafka := bootstrap.Config{StoreDriver: "memory", KafkaBrokers: []string{"broker:9092"}, KafkaTopic: "events", KafkaSASLUsername: "user"}
	if err := startWorker(context.Background(), invalidKafka); err == nil {
		t.Fatal("expected incomplete Kafka security failure")
	}
	if err := startLocal(context.Background(), invalidKafka); err == nil {
		t.Fatal("expected incomplete local Kafka security failure")
	}
}

func TestServeAPIReturnsBindFailure(t *testing.T) {
	t.Parallel()
	config := bootstrap.Config{Address: "invalid-address", PartnerID: "partner", APIKey: "key"}
	if err := serveAPI(context.Background(), config, memory.NewStore()); err == nil {
		t.Fatal("expected invalid listen address to fail")
	}
}

type fakeHTTPServer struct {
	shutdown    chan struct{}
	listenErr   error
	shutdownErr error
}

func (server *fakeHTTPServer) ListenAndServe() error {
	if server.listenErr != nil {
		return server.listenErr
	}
	<-server.shutdown
	return http.ErrServerClosed
}
func (server *fakeHTTPServer) Shutdown(context.Context) error {
	close(server.shutdown)
	return server.shutdownErr
}

func TestServeHTTPServerGracefullyShutsDown(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	server := &fakeHTTPServer{shutdown: make(chan struct{})}
	if err := serveHTTPServer(ctx, server); err != nil {
		t.Fatalf("graceful shutdown: %v", err)
	}
	want := errors.New("listen failed")
	if err := serveHTTPServer(ctx, &fakeHTTPServer{shutdown: make(chan struct{}), listenErr: want}); !errors.Is(err, want) {
		t.Fatalf("listen error: %v", err)
	}
}

func TestStartersComposeValidLocalDependencies(t *testing.T) {
	t.Parallel()
	config := bootstrap.Config{
		Address: "invalid-address", PartnerID: "partner", APIKey: "key", StoreDriver: "memory",
		KafkaBrokers: []string{"localhost:9092"}, KafkaTopic: "events",
	}
	if err := startAPI(context.Background(), config); err == nil {
		t.Fatal("expected composed API bind failure")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := startWorker(cancelled, config); err != nil {
		t.Fatalf("cancel composed worker: %v", err)
	}
	if err := startLocal(context.Background(), config); err == nil {
		t.Fatal("expected composed local HTTP failure")
	}
}

type noOpPublisher struct{}

func (noOpPublisher) Publish(context.Context, app.OutboxEvent) error { return nil }

type readinessStore struct{ err error }

func (store readinessStore) Ready(context.Context) error { return store.err }

func TestReadinessReflectsStoreAvailability(t *testing.T) {
	t.Parallel()
	if !readyForRequests(readinessStore{}) {
		t.Fatal("healthy store should be ready")
	}
	if readyForRequests(readinessStore{err: errors.New("unavailable")}) {
		t.Fatal("unavailable store should not be ready")
	}
	if readyForRequests(struct{}{}) {
		t.Fatal("store without readiness contract should fail closed")
	}
	handle := &storeHandle{TransactionStore: memory.NewStore()}
	if err := handle.Ready(context.Background()); err != nil {
		t.Fatalf("store handle readiness: %v", err)
	}
}

type failingOutboxStore struct{ err error }

func (store failingOutboxStore) ClaimPending(context.Context, time.Time, time.Duration, int) ([]app.PendingEvent, error) {
	return nil, store.err
}
func (failingOutboxStore) MarkPublished(context.Context, string, string, time.Time) error { return nil }
func (failingOutboxStore) RecordFailure(context.Context, app.PublishFailure) error        { return nil }

func TestWorkerLoopStopsOnCancellationAndWrapsStoreFailure(t *testing.T) {
	t.Parallel()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	dispatcher := app.NewDispatcher(memory.NewStore(), noOpPublisher{}, time.Now, app.DefaultDispatcherConfig())
	if err := runWorkerLoop(cancelled, dispatcher); err != nil {
		t.Fatalf("cancel worker: %v", err)
	}
	want := errors.New("store unavailable")
	failing := app.NewDispatcher(failingOutboxStore{err: want}, noOpPublisher{}, time.Now, app.DefaultDispatcherConfig())
	if err := runWorkerLoop(context.Background(), failing); !errors.Is(err, want) {
		t.Fatalf("expected wrapped store failure, got %v", err)
	}
	canceledFailure, cancelFailure := context.WithCancel(context.Background())
	cancelFailure()
	if err := runWorkerLoop(canceledFailure, failing); err != nil {
		t.Fatalf("canceled dependency failure should stop cleanly: %v", err)
	}
}

func TestNewDispatcherAcceptsLocalPlaintextConfiguration(t *testing.T) {
	t.Parallel()
	writer, dispatcher, err := newDispatcher(bootstrap.Config{KafkaBrokers: []string{"localhost:9092"}, KafkaTopic: "events"}, memory.NewStore())
	if err != nil || writer == nil || dispatcher == nil {
		t.Fatalf("new dispatcher: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
}

func TestNewStoreReportsDirectoryAndLogFailures(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	parentFile := filepath.Join(root, "file")
	if err := os.WriteFile(parentFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := newStore(context.Background(), bootstrap.ModeLocal, bootstrap.Config{StoreDriver: "ndjson", StorePath: filepath.Join(parentFile, "state.ndjson")}); err == nil {
		t.Fatal("expected directory creation failure")
	}
	malformed := filepath.Join(root, "malformed.ndjson")
	if err := os.WriteFile(malformed, []byte("not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := newStore(context.Background(), bootstrap.ModeLocal, bootstrap.Config{StoreDriver: "ndjson", StorePath: malformed}); err == nil {
		t.Fatal("expected malformed store failure")
	}
}

func TestNewStoreSelectsNDJSONOnlyForCombinedLocal(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state", "transactions.ndjson")
	store, err := newStore(context.Background(), bootstrap.ModeLocal, bootstrap.Config{StoreDriver: "ndjson", StorePath: path})
	if err != nil {
		t.Fatalf("new NDJSON store: %v", err)
	}
	defer store.Close()
	if _, ok := store.TransactionStore.(*ndjson.Store); !ok {
		t.Fatalf("expected NDJSON store, got %T", store)
	}
	if _, err := newStore(context.Background(), bootstrap.ModeAPI, bootstrap.Config{StoreDriver: "ndjson", StorePath: path}); err == nil {
		t.Fatal("expected separate API NDJSON store to fail")
	}
}

func TestNewStoreRejectsUnavailablePostgresAndDynamoDB(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := newStore(ctx, bootstrap.ModeAPI, bootstrap.Config{
		StoreDriver: "postgres", PostgresURL: "postgres://user:password@127.0.0.1:1/transactions?sslmode=disable&connect_timeout=1",
	}); err == nil {
		t.Fatal("expected unavailable PostgreSQL failure")
	}
	if _, err := newStore(ctx, bootstrap.ModeAPI, bootstrap.Config{
		StoreDriver: "dynamodb", DynamoEndpoint: "http://127.0.0.1:1", DynamoRegion: "us-west-2", DynamoTable: "transactions",
	}); err == nil {
		t.Fatal("expected unavailable DynamoDB failure")
	}
}
