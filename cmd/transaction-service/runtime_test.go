package main

import (
	"path/filepath"
	"testing"

	"github.com/waggertron/emovis-transaction-intake/internal/bootstrap"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/memory"
	"github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/ndjson"
)

func TestNewStoreSelectsMemory(t *testing.T) {
	t.Parallel()

	store, err := newStore(bootstrap.ModeLocal, bootstrap.Config{StoreDriver: "memory"})
	if err != nil {
		t.Fatalf("new memory store: %v", err)
	}
	if _, ok := store.(*memory.Store); !ok {
		t.Fatalf("expected memory store, got %T", store)
	}
}

func TestNewStoreSelectsNDJSONOnlyForCombinedLocal(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state", "transactions.ndjson")
	store, err := newStore(bootstrap.ModeLocal, bootstrap.Config{StoreDriver: "ndjson", StorePath: path})
	if err != nil {
		t.Fatalf("new NDJSON store: %v", err)
	}
	if _, ok := store.(*ndjson.Store); !ok {
		t.Fatalf("expected NDJSON store, got %T", store)
	}
	if _, err := newStore(bootstrap.ModeAPI, bootstrap.Config{StoreDriver: "ndjson", StorePath: path}); err == nil {
		t.Fatal("expected separate API NDJSON store to fail")
	}
}
