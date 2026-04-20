package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"timetrak/internal/rates"
	"timetrak/internal/shared/db"
)

// openPoolForTest returns a pgxpool.Pool or skips if DATABASE_URL isn't set.
func openPoolForTest(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("pgxpool open failed (skipping): %v", err)
	}
	t.Cleanup(pool.Close)
	// Truncate to a known state.
	if _, err := pool.Exec(context.Background(), `
		TRUNCATE TABLE
			time_entries, rate_rules, tasks, projects, clients,
			workspace_members, workspaces, sessions, users
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return pool
}

// seedOneRuleAndEntry creates a workspace, project, one rule, and a closed
// entry with NULL snapshot columns — exactly what the backfill must process.
func seedOneRuleAndEntry(t *testing.T, pool *pgxpool.Pool, slug string) (
	workspaceID, projectID, ruleID, entryID uuid.UUID,
) {
	t.Helper()
	ctx := context.Background()
	var userID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ($1,'x','U') RETURNING id`, slug+"@e").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name,slug) VALUES ('W',$1) RETURNING id`, slug).Scan(&workspaceID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID); err != nil {
		t.Fatal(err)
	}
	var clientID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO clients (workspace_id,name) VALUES ($1,'A') RETURNING id`, workspaceID).Scan(&clientID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO projects (workspace_id,client_id,name) VALUES ($1,$2,'P') RETURNING id`, workspaceID, clientID).Scan(&projectID); err != nil {
		t.Fatal(err)
	}
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var err error
	ruleID, err = rates.NewService(&db.Pool{Pool: pool}).Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1,
	})
	if err != nil {
		t.Fatal(err)
	}
	feb := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	if err := pool.QueryRow(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable)
		VALUES ($1,$2,$3,$4,$5,3600,true) RETURNING id
	`, workspaceID, userID, projectID, feb, feb.Add(time.Hour)).Scan(&entryID); err != nil {
		t.Fatal(err)
	}
	return
}

func TestBackfillPopulatesNullSnapshots(t *testing.T) {
	pool := openPoolForTest(t)
	workspaceID, _, ruleID, entryID := seedOneRuleAndEntry(t, pool, "bf-pop")

	sum, err := BackfillRateSnapshots(context.Background(), pool, false)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if sum.Backfilled != 1 || sum.Workspaces != 1 {
		t.Fatalf("summary = %+v", sum)
	}
	var gotRule uuid.UUID
	var gotMinor int64
	var gotCcy string
	if err := pool.QueryRow(context.Background(), `SELECT rate_rule_id,hourly_rate_minor,currency_code FROM time_entries WHERE id=$1`, entryID).Scan(&gotRule, &gotMinor, &gotCcy); err != nil {
		t.Fatal(err)
	}
	if gotRule != ruleID || gotMinor != 10000 || gotCcy != "USD" {
		t.Fatalf("snapshot not backfilled correctly: rule=%s minor=%d ccy=%s", gotRule, gotMinor, gotCcy)
	}
	_ = workspaceID
}

func TestBackfillIsIdempotent(t *testing.T) {
	pool := openPoolForTest(t)
	seedOneRuleAndEntry(t, pool, "bf-idem")

	if _, err := BackfillRateSnapshots(context.Background(), pool, false); err != nil {
		t.Fatalf("first pass: %v", err)
	}
	sum, err := BackfillRateSnapshots(context.Background(), pool, false)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if sum.Backfilled != 0 {
		t.Fatalf("second pass should write 0, got %d", sum.Backfilled)
	}
}

func TestBackfillDryRunWritesNothing(t *testing.T) {
	pool := openPoolForTest(t)
	_, _, _, entryID := seedOneRuleAndEntry(t, pool, "bf-dry")

	sum, err := BackfillRateSnapshots(context.Background(), pool, true)
	if err != nil {
		t.Fatal(err)
	}
	if !sum.DryRun || sum.Backfilled != 1 {
		t.Fatalf("summary: %+v", sum)
	}
	var rule *uuid.UUID
	if err := pool.QueryRow(context.Background(), `SELECT rate_rule_id FROM time_entries WHERE id=$1`, entryID).Scan(&rule); err != nil {
		t.Fatal(err)
	}
	if rule != nil {
		t.Fatalf("dry-run wrote rate_rule_id: %v", *rule)
	}
}

func TestBackfillAbortsOnAdjacencyConflict(t *testing.T) {
	pool := openPoolForTest(t)
	ctx := context.Background()
	// Seed a workspace with two workspace-default rules directly via SQL so
	// the shared-boundary pair exists (the app-level Create would reject it).
	var userID, workspaceID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('bf-adj@e','x','U') RETURNING id`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name,slug) VALUES ('W','bf-adj') RETURNING id`).Scan(&workspaceID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID); err != nil {
		t.Fatal(err)
	}
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	jun30 := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO rate_rules (workspace_id,currency_code,hourly_rate_minor,effective_from,effective_to) VALUES ($1,'USD',10000,$2,$3)`, workspaceID, jan1, jun30); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO rate_rules (workspace_id,currency_code,hourly_rate_minor,effective_from) VALUES ($1,'USD',12000,$2)`, workspaceID, jun30); err != nil {
		t.Fatal(err)
	}
	// Also need an eligible entry, otherwise the probe runs but the iteration
	// is empty — but the probe runs first, so no need. Still, seed to be explicit.
	sum, err := BackfillRateSnapshots(ctx, pool, true)
	if !errors.Is(err, ErrAdjacencyConflict) {
		t.Fatalf("expected ErrAdjacencyConflict, got %v", err)
	}
	if len(sum.AdjacencyBad) != 1 {
		t.Fatalf("adjacency list = %+v", sum.AdjacencyBad)
	}
}
