package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/waggertron/emovis-transaction-intake/internal/bootstrap"
)

type starter func(context.Context, bootstrap.Config) error

func run(ctx context.Context, arguments []string, lookup func(string) string, starters map[bootstrap.Mode]starter) error {
	if len(arguments) != 1 {
		return fmt.Errorf("usage: transaction-service <api|worker|local>")
	}
	mode, err := bootstrap.ParseMode(arguments[0])
	if err != nil {
		return err
	}
	config, err := bootstrap.LoadConfig(lookup)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	start, found := starters[mode]
	if !found {
		return fmt.Errorf("mode %q has no starter", mode)
	}
	return start(ctx, config)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Getenv, productionStarters()); err != nil {
		slog.Error("transaction service stopped", "error", err)
		os.Exit(1)
	}
}
