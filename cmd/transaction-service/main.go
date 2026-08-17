package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/waggertron/emovis-transaction-intake/internal/bootstrap"
	secretconfig "github.com/waggertron/emovis-transaction-intake/internal/bootstrap/secrets"
)

type starter func(context.Context, bootstrap.Config) error
type secretLoader interface {
	Load(context.Context) (map[string]string, error)
}
type awsProviderFactory func(context.Context, string) (secretLoader, error)

func run(ctx context.Context, arguments []string, lookup func(string) string, starters map[bootstrap.Mode]starter) error {
	return runWithAWSProvider(ctx, arguments, lookup, starters, productionAWSProvider)
}

func runWithAWSProvider(ctx context.Context, arguments []string, lookup func(string) string, starters map[bootstrap.Mode]starter, awsFactory awsProviderFactory) error {
	if len(arguments) != 1 {
		return fmt.Errorf("usage: transaction-service <api|worker|local>")
	}
	mode, err := bootstrap.ParseMode(arguments[0])
	if err != nil {
		return err
	}
	configurationLookup := lookup
	localPath, awsSecretID := lookup("LOCAL_SECRET_FILE"), lookup("AWS_SECRET_ID")
	if localPath != "" && awsSecretID != "" {
		return fmt.Errorf("LOCAL_SECRET_FILE and AWS_SECRET_ID are mutually exclusive")
	}
	var provider secretLoader
	if localPath != "" {
		provider = secretconfig.NewFileProvider(localPath)
	}
	if awsSecretID != "" {
		if awsFactory == nil {
			return fmt.Errorf("initialize AWS secret provider: unavailable")
		}
		provider, err = awsFactory(ctx, awsSecretID)
		if err != nil {
			return fmt.Errorf("initialize AWS secret provider: %w", err)
		}
	}
	if provider != nil {
		values, err := provider.Load(ctx)
		if err != nil {
			return fmt.Errorf("load secret configuration: %w", err)
		}
		configurationLookup = secretconfig.Overlay(lookup, values)
	}
	config, err := bootstrap.LoadConfig(configurationLookup)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	if config.StoreDriver == "ndjson" && mode != bootstrap.ModeLocal {
		return fmt.Errorf("NDJSON storage is supported only in combined local mode")
	}
	start, found := starters[mode]
	if !found {
		return fmt.Errorf("mode %q has no starter", mode)
	}
	return start(ctx, config)
}

func productionAWSProvider(ctx context.Context, secretID string) (secretLoader, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	config, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration")
	}
	return secretconfig.NewAWSProvider(secretsmanager.NewFromConfig(config), secretID), nil
}

func execute(ctx context.Context, arguments []string, lookup func(string) string, starters map[bootstrap.Mode]starter) error {
	return run(ctx, arguments, lookup, starters)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := execute(ctx, os.Args[1:], os.Getenv, productionStarters()); err != nil {
		slog.Error("transaction service stopped", "error", err)
		os.Exit(1)
	}
}
