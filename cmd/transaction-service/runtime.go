package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/waggertron/emovis-transaction-intake/internal/bootstrap"
	httpadapter "github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/http"
	kafkaadapter "github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/kafka"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/memory"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/ndjson"
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
	store, err := newStore(bootstrap.ModeAPI, config)
	if err != nil {
		return err
	}
	return serveAPI(ctx, config, store)
}

func startWorker(ctx context.Context, config bootstrap.Config) error {
	store, err := newStore(bootstrap.ModeWorker, config)
	if err != nil {
		return err
	}
	writer, dispatcher, err := newDispatcher(config, store)
	if err != nil {
		return err
	}
	defer writer.Close()
	return runWorkerLoop(ctx, dispatcher)
}

func startLocal(ctx context.Context, config bootstrap.Config) error {
	store, err := newStore(bootstrap.ModeLocal, config)
	if err != nil {
		return err
	}
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
	handler := httpadapter.NewHandler(intake, auth, rand.Text, func() bool { return true })
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
		SASLUsername: config.KafkaSASLUsername, SASLPassword: config.KafkaSASLPassword,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("configure Kafka: %w", err)
	}
	publisher := kafkaadapter.NewPublisher(writer, config.KafkaTopic)
	return writer, app.NewDispatcher(store, publisher, time.Now, app.DefaultDispatcherConfig()), nil
}

func newStore(mode bootstrap.Mode, config bootstrap.Config) (app.TransactionStore, error) {
	switch config.StoreDriver {
	case "", "memory":
		return memory.NewStore(), nil
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
		return store, nil
	default:
		return nil, fmt.Errorf("store driver %q is not wired", config.StoreDriver)
	}
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
