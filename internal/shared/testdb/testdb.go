// Package testdb is a tiny helper for integration tests: it opens the pool
// from $DATABASE_URL, truncates domain tables between tests, and skips when
// DATABASE_URL is unset (so `go test ./...` without Postgres still passes).
package testdb

import (
	"context"
	"os"
	"testing"

	"timetrak/internal/shared/db"
)

// Open returns a connected Pool or calls t.Skip if DATABASE_URL is unset.
func Open(t *testing.T) *db.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	pool, err := db.Open(context.Background(), dsn)
	if err != nil {
		t.Skipf("db open failed (skipping): %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	Truncate(t, pool)
	return pool
}

// Truncate clears every mutable domain table. `schema_migrations` is preserved.
func Truncate(t *testing.T, pool *db.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			time_entries, rate_rules, tasks, projects, clients,
			workspace_members, workspaces, sessions, users
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
