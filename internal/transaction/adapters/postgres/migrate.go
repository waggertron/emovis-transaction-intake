package postgres

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schema string

func Migrate(ctx context.Context, database *sql.DB) error {
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin PostgreSQL schema migration: %w", err)
	}
	defer transaction.Rollback()
	// Serialize bootstrap across independently deployed API and worker processes.
	if _, err := transaction.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", int64(742021680)); err != nil {
		return fmt.Errorf("lock PostgreSQL schema migration: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("apply PostgreSQL schema: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit PostgreSQL schema migration: %w", err)
	}
	return nil
}
