package rates_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/rates"
	"timetrak/internal/shared/testdb"
)

func TestRateResolutionPrecedence(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()

	// Seed workspace + client + project.
	var userID, workspaceID, clientID, projectID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name)
		VALUES ('r@example.com', 'x', 'R') RETURNING id
	`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug) VALUES ('W','w') RETURNING id
	`).Scan(&workspaceID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')
	`, workspaceID, userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO clients (workspace_id, name) VALUES ($1,'Acme') RETURNING id
	`, workspaceID).Scan(&clientID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, client_id, name) VALUES ($1,$2,'Web') RETURNING id
	`, workspaceID, clientID).Scan(&projectID); err != nil {
		t.Fatal(err)
	}

	svc := rates.NewService(pool)

	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	apr17 := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)

	// Workspace default: 10000 minor units/hr, from 2026-01-01 open-ended.
	if _, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1,
	}); err != nil {
		t.Fatalf("ws default: %v", err)
	}

	// Expect workspace fallback.
	res, err := svc.Resolve(ctx, workspaceID, projectID, apr17)
	if err != nil || !res.Found || res.Level != rates.LevelWorkspace || res.HourlyRateMinor != 10000 {
		t.Fatalf("ws fallback: %+v err=%v", res, err)
	}

	// Client rule: 12000 from 2026-02-01 open-ended.
	if _, err := svc.Create(ctx, workspaceID, rates.Input{
		ClientID: clientID, CurrencyCode: "USD", HourlyRateMinor: 12000,
		EffectiveFrom: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("client rule: %v", err)
	}
	res, _ = svc.Resolve(ctx, workspaceID, projectID, apr17)
	if res.Level != rates.LevelClient || res.HourlyRateMinor != 12000 {
		t.Fatalf("expected client precedence: %+v", res)
	}

	// Project rule: 15000 from 2026-03-01 open-ended.
	if _, err := svc.Create(ctx, workspaceID, rates.Input{
		ProjectID: projectID, CurrencyCode: "USD", HourlyRateMinor: 15000,
		EffectiveFrom: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("project rule: %v", err)
	}
	res, _ = svc.Resolve(ctx, workspaceID, projectID, apr17)
	if res.Level != rates.LevelProject || res.HourlyRateMinor != 15000 {
		t.Fatalf("expected project precedence: %+v", res)
	}

	// Historical correctness: February date should get client rate (project not yet effective).
	feb15 := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	res, _ = svc.Resolve(ctx, workspaceID, projectID, feb15)
	if res.Level != rates.LevelClient || res.HourlyRateMinor != 12000 {
		t.Fatalf("historical client rate: %+v", res)
	}

	// Historical correctness: January date should get workspace default.
	res, _ = svc.Resolve(ctx, workspaceID, projectID, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	if res.Level != rates.LevelWorkspace || res.HourlyRateMinor != 10000 {
		t.Fatalf("historical ws rate: %+v", res)
	}
}

func TestOverlapRejection(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	var userID, workspaceID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('o@e','x','O') RETURNING id`).Scan(&userID)
	_ = pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug) VALUES ('W','w-overlap') RETURNING id`).Scan(&workspaceID)
	_, _ = pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID)

	svc := rates.NewService(pool)
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1,
	}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 12000,
		EffectiveFrom: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected overlap rejection")
	}

	// Adjacent non-overlapping is OK: close out the first with an effective_to, then a new one.
	jun30 := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	// Need to fetch + update the first rule.
	rules, _ := svc.List(ctx, workspaceID)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if err := svc.Update(ctx, workspaceID, rules[0].ID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1, EffectiveTo: &jun30,
	}); err != nil {
		t.Fatalf("close out first: %v", err)
	}
	if _, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 12000,
		EffectiveFrom: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("adjacent should be allowed: %v", err)
	}
}
