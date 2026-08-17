package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/waggertron/emovis-transaction-intake/internal/bootstrap"
	httpadapter "github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/http"
	kafkaadapter "github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/kafka"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/memory"
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
	store := memory.NewStore()
	return serveAPI(ctx, config, store)
}

func startWorker(ctx context.Context, config bootstrap.Config) error {
	store := memory.NewStore()
	writer, dispatcher, err := newDispatcher(config, store)
	if err != nil {
		return err
	}
	defer writer.Close()
	return runWorkerLoop(ctx, dispatcher)
}

func startLocal(ctx context.Context, config bootstrap.Config) error {
	store := memory.NewStore()
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

func serveAPI(ctx context.Context, config bootstrap.Config, store *memory.Store) error {
	intake := app.NewIntakeService(store, time.Now, rand.Text)
	auth := httpadapter.NewStaticAPIKeys(map[string]string{config.PartnerID: config.APIKey})
	handler := httpadapter.NewHandler(intake, auth, rand.Text, func() bool { return true })
	server := bootstrap.NewHTTPServer(config.Address, handler)

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
	slog.Info("HTTP server starting", "address", config.Address)
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		<-shutdownComplete
		return nil
	}
	return fmt.Errorf("serve HTTP: %w", err)
}

func newDispatcher(config bootstrap.Config, store *memory.Store) (interface{ Close() error }, *app.Dispatcher, error) {
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
