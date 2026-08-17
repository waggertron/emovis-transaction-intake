package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awssdk "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/waggertron/emovis-transaction-intake/internal/bootstrap"
	dynamoadapter "github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/dynamodb"
	httpadapter "github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/http"
	kafkaadapter "github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/kafka"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/memory"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/ndjson"
	postgresadapter "github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/postgres"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/app"
)

func productionStarters() map[bootstrap.Mode]starter {
	return map[bootstrap.Mode]starter{
		bootstrap.ModeAPI:    startAPI,
		bootstrap.ModeWorker: startWorker,
		bootstrap.ModeLocal:  startLocal,
	}
}

func startAPI(ctx context.Context, config bootstrap.Config) error {
	store, err := newStore(ctx, bootstrap.ModeAPI, config)
	if err != nil {
		return err
	}
	defer store.Close()
	return serveAPI(ctx, config, store)
}

func startWorker(ctx context.Context, config bootstrap.Config) error {
	store, err := newStore(ctx, bootstrap.ModeWorker, config)
	if err != nil {
		return err
	}
	defer store.Close()
	writer, dispatcher, err := newDispatcher(config, store)
	if err != nil {
		return err
	}
	defer writer.Close()
	return runWorkerLoop(ctx, dispatcher)
}

func startLocal(ctx context.Context, config bootstrap.Config) error {
	store, err := newStore(ctx, bootstrap.ModeLocal, config)
	if err != nil {
		return err
	}
	defer store.Close()
	writer, dispatcher, err := newDispatcher(config, store)
	if err != nil {
		return err
	}
	defer writer.Close()

	workerContext, cancelWorker := context.WithCancel(ctx)
	defer cancelWorker()
	workerErrors := make(chan error, 1)
	go func() { workerErrors <- runWorkerLoop(workerContext, dispatcher) }()

	httpErrors := make(chan error, 1)
	go func() { httpErrors <- serveAPI(workerContext, config, store) }()
	select {
	case err := <-workerErrors:
		cancelWorker()
		return err
	case err := <-httpErrors:
		cancelWorker()
		return err
	case <-ctx.Done():
		cancelWorker()
		return nil
	}
}

func serveAPI(ctx context.Context, config bootstrap.Config, store app.IntakeStore) error {
	intake := app.NewIntakeService(store, time.Now, rand.Text)
	auth := httpadapter.NewStaticAPIKeys(map[string]string{config.PartnerID: config.APIKey})
	handler := httpadapter.NewHandler(intake, auth, rand.Text, func() bool { return readyForRequests(store) })
	server := bootstrap.NewHTTPServer(config.Address, handler)
	slog.Info("HTTP server starting", "address", config.Address)
	return serveHTTPServer(ctx, server)
}

type httpServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func serveHTTPServer(ctx context.Context, server httpServer) error {
	shutdownComplete := make(chan struct{})
	go func() {
		defer close(shutdownComplete)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP shutdown failed", "error", err)
		}
	}()
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		<-shutdownComplete
		return nil
	}
	return fmt.Errorf("serve HTTP: %w", err)
}

func newDispatcher(config bootstrap.Config, store app.OutboxStore) (interface{ Close() error }, *app.Dispatcher, error) {
	writer, err := kafkaadapter.NewWriter(kafkaadapter.WriterConfig{
		Brokers: config.KafkaBrokers, Topic: config.KafkaTopic, TLS: config.KafkaTLS,
		CAFile:       config.KafkaCAFile,
		SASLUsername: config.KafkaSASLUsername, SASLPassword: config.KafkaSASLPassword,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("configure Kafka: %w", err)
	}
	publisher := kafkaadapter.NewPublisher(writer, config.KafkaTopic)
	return writer, app.NewDispatcher(store, publisher, time.Now, app.DefaultDispatcherConfig()), nil
}

type storeHandle struct {
	app.TransactionStore
	close func() error
}

type readinessChecker interface {
	Ready(context.Context) error
}

func readyForRequests(store any) bool {
	checker, ok := store.(readinessChecker)
	if !ok {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return checker.Ready(ctx) == nil
}

func (store *storeHandle) Ready(ctx context.Context) error {
	checker, ok := store.TransactionStore.(readinessChecker)
	if !ok {
		return fmt.Errorf("store does not implement readiness")
	}
	return checker.Ready(ctx)
}

func (store *storeHandle) Close() error {
	if store.close == nil {
		return nil
	}
	return store.close()
}

func newStore(ctx context.Context, mode bootstrap.Mode, config bootstrap.Config) (*storeHandle, error) {
	switch config.StoreDriver {
	case "", "memory":
		return &storeHandle{TransactionStore: memory.NewStore()}, nil
	case "ndjson":
		if mode != bootstrap.ModeLocal {
			return nil, fmt.Errorf("NDJSON storage is supported only in combined local mode")
		}
		directory := filepath.Dir(config.StorePath)
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return nil, fmt.Errorf("create NDJSON store directory: %w", err)
		}
		store, err := ndjson.NewStore(config.StorePath)
		if err != nil {
			return nil, err
		}
		return &storeHandle{TransactionStore: store}, nil
	case "postgres":
		database, err := sql.Open("pgx", config.PostgresURL)
		if err != nil {
			return nil, fmt.Errorf("open PostgreSQL store: %w", err)
		}
		database.SetMaxOpenConns(20)
		database.SetMaxIdleConns(5)
		database.SetConnMaxLifetime(30 * time.Minute)
		if err := database.PingContext(ctx); err != nil {
			_ = database.Close()
			return nil, fmt.Errorf("connect PostgreSQL store: %w", err)
		}
		if err := postgresadapter.Migrate(ctx, database); err != nil {
			_ = database.Close()
			return nil, err
		}
		return &storeHandle{TransactionStore: postgresadapter.NewStore(database), close: database.Close}, nil
	case "dynamodb":
		awsConfig, err := dynamoConfig(ctx, config)
		if err != nil {
			return nil, err
		}
		client := awssdk.NewFromConfig(awsConfig, func(options *awssdk.Options) {
			if config.DynamoEndpoint != "" {
				options.BaseEndpoint = aws.String(config.DynamoEndpoint)
			}
		})
		if config.DynamoEndpoint != "" {
			if err := dynamoadapter.EnsureTable(ctx, client, config.DynamoTable); err != nil {
				return nil, err
			}
		}
		return &storeHandle{TransactionStore: dynamoadapter.NewStore(client, config.DynamoTable)}, nil
	default:
		return nil, fmt.Errorf("store driver %q is not wired", config.StoreDriver)
	}
}

func dynamoConfig(ctx context.Context, config bootstrap.Config) (aws.Config, error) {
	if config.DynamoEndpoint != "" {
		return aws.Config{
			Region:      config.DynamoRegion,
			Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider("local", "local", "")),
		}, nil
	}
	loaded, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(config.DynamoRegion))
	if err != nil {
		return aws.Config{}, fmt.Errorf("load AWS configuration: %w", err)
	}
	return loaded, nil
}

func runWorkerLoop(ctx context.Context, dispatcher *app.Dispatcher) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := dispatcher.RunBatch(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("dispatch outbox: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
