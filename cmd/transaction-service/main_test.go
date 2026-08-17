package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/waggertron/emovis-transaction-intake/internal/bootstrap"
)

type stubSecretLoader struct {
	values map[string]string
	err    error
}

func (loader stubSecretLoader) Load(context.Context) (map[string]string, error) {
	return loader.values, loader.err
}

func TestRunLoadsConfigurationFromAWSSecretProvider(t *testing.T) {
	t.Parallel()
	lookup := func(name string) string {
		if name == "AWS_SECRET_ID" {
			return "runtime-secret"
		}
		if name == "API_KEY" {
			return "explicit-key"
		}
		return ""
	}
	called := false
	starters := map[bootstrap.Mode]starter{bootstrap.ModeWorker: func(_ context.Context, config bootstrap.Config) error {
		called = true
		if config.APIKey != "explicit-key" || config.PartnerID != "aws-partner" {
			t.Fatalf("unexpected AWS config: %s", config.String())
		}
		return nil
	}}
	err := runWithAWSProvider(context.Background(), []string{"worker"}, lookup, starters, func(context.Context, string) (secretLoader, error) {
		return stubSecretLoader{values: map[string]string{"API_KEY": "provider-key", "PARTNER_ID": "aws-partner"}}, nil
	})
	if err != nil || !called {
		t.Fatalf("run with AWS provider: called=%t err=%v", called, err)
	}
}

func TestRunRejectsAmbiguousSecretProviders(t *testing.T) {
	t.Parallel()
	lookup := func(name string) string {
		if name == "LOCAL_SECRET_FILE" || name == "AWS_SECRET_ID" {
			return "configured"
		}
		return ""
	}
	if err := runWithAWSProvider(context.Background(), []string{"api"}, lookup, nil, nil); err == nil {
		t.Fatal("expected mutually exclusive secret providers")
	}
}

func TestRunRejectsAWSProviderInitializationAndLoadFailures(t *testing.T) {
	t.Parallel()
	lookup := func(name string) string {
		if name == "AWS_SECRET_ID" {
			return "runtime-secret"
		}
		return ""
	}
	for _, test := range []struct {
		name    string
		factory awsProviderFactory
	}{
		{name: "missing factory"},
		{name: "initialization", factory: func(context.Context, string) (secretLoader, error) {
			return nil, errors.New("configuration unavailable")
		}},
		{name: "load", factory: func(context.Context, string) (secretLoader, error) {
			return stubSecretLoader{err: errors.New("provider unavailable")}, nil
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := runWithAWSProvider(context.Background(), []string{"api"}, lookup, nil, test.factory); err == nil {
				t.Fatal("expected AWS provider failure")
			}
		})
	}
}

func TestProductionAWSProviderRejectsCanceledConfiguration(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := productionAWSProvider(ctx, "runtime-secret"); err == nil {
		t.Fatal("expected canceled AWS configuration")
	}
}

func TestProductionAWSProviderBuildsClientFromEnvironment(t *testing.T) {
	t.Setenv("AWS_REGION", "us-west-2")
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	provider, err := productionAWSProvider(context.Background(), "runtime-secret")
	if err != nil || provider == nil {
		t.Fatalf("build production AWS provider: %v", err)
	}
}

func TestExecuteReturnsRunError(t *testing.T) {
	t.Parallel()
	if err := execute(context.Background(), nil, commandLookup, nil); err == nil {
		t.Fatal("expected command usage error")
	}
}

func commandLookup(name string) string {
	values := map[string]string{"API_KEY": "test-key", "PARTNER_ID": "partner-west"}
	return values[name]
}

func TestRunLoadsConfigurationFromLocalSecretFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"API_KEY":"file-key","PARTNER_ID":"file-partner"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	called := false
	lookup := func(name string) string {
		if name == "LOCAL_SECRET_FILE" {
			return path
		}
		return ""
	}
	starters := map[bootstrap.Mode]starter{bootstrap.ModeAPI: func(_ context.Context, config bootstrap.Config) error {
		called = true
		if config.APIKey != "file-key" || config.PartnerID != "file-partner" {
			t.Fatalf("unexpected file config: %s", config.String())
		}
		return nil
	}}
	if err := run(context.Background(), []string{"api"}, lookup, starters); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Fatal("starter was not called")
	}
}

func TestRunRejectsUnavailableLocalSecretFile(t *testing.T) {
	t.Parallel()
	lookup := func(name string) string {
		if name == "LOCAL_SECRET_FILE" {
			return filepath.Join(t.TempDir(), "absent.json")
		}
		return ""
	}
	if err := run(context.Background(), []string{"api"}, lookup, nil); err == nil {
		t.Fatal("expected unavailable local secret file")
	}
}

func TestRunRejectsMissingAndUnknownModesBeforeStarting(t *testing.T) {
	t.Parallel()

	starters := map[bootstrap.Mode]starter{}
	if err := run(context.Background(), nil, commandLookup, starters); err == nil {
		t.Fatal("expected missing mode to fail")
	}
	if err := run(context.Background(), []string{"unknown"}, commandLookup, starters); err == nil {
		t.Fatal("expected unknown mode to fail")
	}
}

func TestRunLoadsConfigAndSelectsExactMode(t *testing.T) {
	t.Parallel()

	for _, wanted := range []bootstrap.Mode{bootstrap.ModeAPI, bootstrap.ModeWorker, bootstrap.ModeLocal} {
		wanted := wanted
		t.Run(string(wanted), func(t *testing.T) {
			t.Parallel()
			called := false
			starters := map[bootstrap.Mode]starter{
				wanted: func(_ context.Context, config bootstrap.Config) error {
					called = true
					if config.PartnerID != "partner-west" || config.APIKey != "test-key" {
						t.Fatalf("unexpected config")
					}
					return nil
				},
			}
			if err := run(context.Background(), []string{string(wanted)}, commandLookup, starters); err != nil {
				t.Fatalf("run: %v", err)
			}
			if !called {
				t.Fatal("selected starter was not called")
			}
		})
	}
}

func TestRunStartsWithSafeLocalConfig(t *testing.T) {
	t.Parallel()

	called := false
	starters := map[bootstrap.Mode]starter{bootstrap.ModeAPI: func(context.Context, bootstrap.Config) error {
		called = true
		return nil
	}}
	if err := run(context.Background(), []string{"api"}, func(string) string { return "" }, starters); err != nil {
		t.Fatalf("expected safe local config: %v", err)
	}
	if !called {
		t.Fatal("starter not called with safe local config")
	}
}

func TestRunRejectsNDJSONForSeparateProcesses(t *testing.T) {
	t.Parallel()

	lookup := func(name string) string {
		values := map[string]string{"API_KEY": "key", "PARTNER_ID": "partner", "STORE_DRIVER": "ndjson"}
		return values[name]
	}
	for _, mode := range []string{"api", "worker"} {
		called := false
		starters := map[bootstrap.Mode]starter{bootstrap.Mode(mode): func(context.Context, bootstrap.Config) error {
			called = true
			return nil
		}}
		if err := run(context.Background(), []string{mode}, lookup, starters); err == nil {
			t.Fatalf("expected %s NDJSON mode to fail", mode)
		}
		if called {
			t.Fatalf("%s starter called with unsafe NDJSON config", mode)
		}
	}
}
