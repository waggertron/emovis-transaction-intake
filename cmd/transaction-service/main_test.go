package main

import (
	"context"
	"testing"

	"github.com/waggertron/emovis-transaction-intake/internal/bootstrap"
)

func commandLookup(name string) string {
	values := map[string]string{"API_KEY": "test-key", "PARTNER_ID": "partner-west"}
	return values[name]
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

func TestRunDoesNotStartWithInvalidConfig(t *testing.T) {
	t.Parallel()

	called := false
	starters := map[bootstrap.Mode]starter{bootstrap.ModeAPI: func(context.Context, bootstrap.Config) error {
		called = true
		return nil
	}}
	if err := run(context.Background(), []string{"api"}, func(string) string { return "" }, starters); err == nil {
		t.Fatal("expected invalid config to fail")
	}
	if called {
		t.Fatal("starter called with invalid config")
	}
}
