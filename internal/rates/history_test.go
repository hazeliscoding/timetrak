package rates_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/rates"
	"timetrak/internal/shared/testdb"
)

// — Tests —

func TestRateRuleDeleteRejectedWhenReferenced(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	var userID, workspaceID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('del-ref@e','x','U') RETURNING id`).Scan(&userID)
	_ = pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug) VALUES ('W','ws-del-ref') RETURNING id`).Scan(&workspaceID)
	_, _ = pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID)
	var clientID, projectID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO clients (workspace_id,name) VALUES ($1,'A') RETURNING id`, workspaceID).Scan(&clientID)
	_ = pool.QueryRow(ctx, `INSERT INTO projects (workspace_id,client_id,name) VALUES ($1,$2,'P') RETURNING id`, workspaceID, clientID).Scan(&projectID)

	svc := rates.NewService(pool)
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ruleID, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed a closed entry that references the rule.
	feb := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	_, err = pool.Exec(ctx, `
		INSERT INTO time_entries
		  (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable,
		   rate_rule_id,hourly_rate_minor,currency_code)
		VALUES ($1,$2,$3,$4,$5,3600,true,$6,10000,'USD')
	`, workspaceID, userID, projectID, feb, feb.Add(time.Hour), ruleID)
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.Delete(ctx, workspaceID, ruleID); !errors.Is(err, rates.ErrRuleReferenced) {
		t.Fatalf("expected ErrRuleReferenced, got %v", err)
	}
	// Rule still exists.
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM rate_rules WHERE id=$1`, ruleID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("rule should still exist; n=%d err=%v", n, err)
	}
}

func TestRateRuleUpdateAmountRejectedWhenReferenced(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	var userID, workspaceID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('upd-amt@e','x','U') RETURNING id`).Scan(&userID)
	_ = pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug) VALUES ('W','ws-upd-amt') RETURNING id`).Scan(&workspaceID)
	_, _ = pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID)
	var clientID, projectID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO clients (workspace_id,name) VALUES ($1,'A') RETURNING id`, workspaceID).Scan(&clientID)
	_ = pool.QueryRow(ctx, `INSERT INTO projects (workspace_id,client_id,name) VALUES ($1,$2,'P') RETURNING id`, workspaceID, clientID).Scan(&projectID)

	svc := rates.NewService(pool)
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ruleID, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1,
	})
	if err != nil {
		t.Fatal(err)
	}
	feb := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	_, err = pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable,rate_rule_id,hourly_rate_minor,currency_code)
		VALUES ($1,$2,$3,$4,$5,3600,true,$6,10000,'USD')
	`, workspaceID, userID, projectID, feb, feb.Add(time.Hour), ruleID)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.Update(ctx, workspaceID, ruleID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 20000, EffectiveFrom: jan1,
	})
	if !errors.Is(err, rates.ErrRuleReferenced) {
		t.Fatalf("expected ErrRuleReferenced, got %v", err)
	}
}

func TestRateRuleExtendEffectiveToAllowed(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	var userID, workspaceID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('upd-ext@e','x','U') RETURNING id`).Scan(&userID)
	_ = pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug) VALUES ('W','ws-upd-ext') RETURNING id`).Scan(&workspaceID)
	_, _ = pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID)
	var clientID, projectID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO clients (workspace_id,name) VALUES ($1,'A') RETURNING id`, workspaceID).Scan(&clientID)
	_ = pool.QueryRow(ctx, `INSERT INTO projects (workspace_id,client_id,name) VALUES ($1,$2,'P') RETURNING id`, workspaceID, clientID).Scan(&projectID)

	svc := rates.NewService(pool)
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ruleID, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1,
	})
	if err != nil {
		t.Fatal(err)
	}
	feb := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	_, err = pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable,rate_rule_id,hourly_rate_minor,currency_code)
		VALUES ($1,$2,$3,$4,$5,3600,true,$6,10000,'USD')
	`, workspaceID, userID, projectID, feb, feb.Add(time.Hour), ruleID)
	if err != nil {
		t.Fatal(err)
	}

	future := time.Date(2030, 12, 31, 0, 0, 0, 0, time.UTC)
	if err := svc.Update(ctx, workspaceID, ruleID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1, EffectiveTo: &future,
	}); err != nil {
		t.Fatalf("extend effective_to should succeed: %v", err)
	}
	var got time.Time
	if err := pool.QueryRow(ctx, `SELECT effective_to FROM rate_rules WHERE id=$1`, ruleID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Equal(future) {
		t.Fatalf("effective_to: got %s want %s", got, future)
	}
}

func TestRateRuleShortenEffectiveToBelowReferencingEntryRejected(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	var userID, workspaceID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('upd-shrt@e','x','U') RETURNING id`).Scan(&userID)
	_ = pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug) VALUES ('W','ws-upd-shrt') RETURNING id`).Scan(&workspaceID)
	_, _ = pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID)
	var clientID, projectID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO clients (workspace_id,name) VALUES ($1,'A') RETURNING id`, workspaceID).Scan(&clientID)
	_ = pool.QueryRow(ctx, `INSERT INTO projects (workspace_id,client_id,name) VALUES ($1,$2,'P') RETURNING id`, workspaceID, clientID).Scan(&projectID)

	svc := rates.NewService(pool)
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	dec31 := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	ruleID, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1, EffectiveTo: &dec31,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Entry on 2026-06-15.
	jun15 := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)
	_, err = pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable,rate_rule_id,hourly_rate_minor,currency_code)
		VALUES ($1,$2,$3,$4,$5,3600,true,$6,10000,'USD')
	`, workspaceID, userID, projectID, jun15, jun15.Add(time.Hour), ruleID)
	if err != nil {
		t.Fatal(err)
	}

	// Attempt to shorten to May 31 — before the Jun 15 entry.
	may31 := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	err = svc.Update(ctx, workspaceID, ruleID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1, EffectiveTo: &may31,
	})
	if !errors.Is(err, rates.ErrRuleReferenced) {
		t.Fatalf("expected ErrRuleReferenced, got %v", err)
	}
}

func TestRateRuleOverlapSharedBoundaryRejected(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	var userID, workspaceID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('ovb@e','x','U') RETURNING id`).Scan(&userID)
	_ = pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug) VALUES ('W','ws-ovb') RETURNING id`).Scan(&workspaceID)
	_, _ = pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID)

	svc := rates.NewService(pool)
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	jun30 := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	if _, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1, EffectiveTo: &jun30,
	}); err != nil {
		t.Fatal(err)
	}
	// Starting on the same day (shared boundary) must be rejected.
	_, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 12000, EffectiveFrom: jun30,
	})
	if !errors.Is(err, rates.ErrOverlap) {
		t.Fatalf("expected ErrOverlap for shared boundary, got %v", err)
	}
}

// Boundary semantics (section 11)

func TestResolveOnFirstDayOfRule(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	var userID, workspaceID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('first@e','x','U') RETURNING id`).Scan(&userID)
	_ = pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug) VALUES ('W','ws-first') RETURNING id`).Scan(&workspaceID)
	_, _ = pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID)
	var clientID, projectID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO clients (workspace_id,name) VALUES ($1,'A') RETURNING id`, workspaceID).Scan(&clientID)
	_ = pool.QueryRow(ctx, `INSERT INTO projects (workspace_id,client_id,name) VALUES ($1,$2,'P') RETURNING id`, workspaceID, clientID).Scan(&projectID)

	svc := rates.NewService(pool)
	apr1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if _, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: apr1,
	}); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Resolve(ctx, workspaceID, projectID, apr1)
	if err != nil || !res.Found {
		t.Fatalf("resolve on effective_from: %+v err=%v", res, err)
	}
}

func TestResolveOnLastDayOfRule(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	var userID, workspaceID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('last@e','x','U') RETURNING id`).Scan(&userID)
	_ = pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug) VALUES ('W','ws-last') RETURNING id`).Scan(&workspaceID)
	_, _ = pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID)
	var clientID, projectID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO clients (workspace_id,name) VALUES ($1,'A') RETURNING id`, workspaceID).Scan(&clientID)
	_ = pool.QueryRow(ctx, `INSERT INTO projects (workspace_id,client_id,name) VALUES ($1,$2,'P') RETURNING id`, workspaceID, clientID).Scan(&projectID)

	svc := rates.NewService(pool)
	apr1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	apr30 := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	if _, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: apr1, EffectiveTo: &apr30,
	}); err != nil {
		t.Fatal(err)
	}
	// 2026-04-30T23:59:59Z should still match.
	late := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)
	res, _ := svc.Resolve(ctx, workspaceID, projectID, late)
	if !res.Found {
		t.Fatalf("resolve on effective_to should match: %+v", res)
	}
}

func TestResolveDayAfterEffectiveTo(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	var userID, workspaceID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('day-after@e','x','U') RETURNING id`).Scan(&userID)
	_ = pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug) VALUES ('W','ws-day-after') RETURNING id`).Scan(&workspaceID)
	_, _ = pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID)
	var clientID, projectID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO clients (workspace_id,name) VALUES ($1,'A') RETURNING id`, workspaceID).Scan(&clientID)
	_ = pool.QueryRow(ctx, `INSERT INTO projects (workspace_id,client_id,name) VALUES ($1,$2,'P') RETURNING id`, workspaceID, clientID).Scan(&projectID)

	svc := rates.NewService(pool)
	apr1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	apr30 := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	if _, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: apr1, EffectiveTo: &apr30,
	}); err != nil {
		t.Fatal(err)
	}
	may1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	res, _ := svc.Resolve(ctx, workspaceID, projectID, may1)
	if res.Found {
		t.Fatalf("expected no match on day after effective_to: %+v", res)
	}
}

func TestResolveAcrossUTCBoundary(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	var userID, workspaceID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('utc@e','x','U') RETURNING id`).Scan(&userID)
	_ = pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug) VALUES ('W','ws-utc') RETURNING id`).Scan(&workspaceID)
	_, _ = pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID)
	var clientID, projectID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO clients (workspace_id,name) VALUES ($1,'A') RETURNING id`, workspaceID).Scan(&clientID)
	_ = pool.QueryRow(ctx, `INSERT INTO projects (workspace_id,client_id,name) VALUES ($1,$2,'P') RETURNING id`, workspaceID, clientID).Scan(&projectID)

	svc := rates.NewService(pool)
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mar31 := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	apr1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	// R_old: Jan 1 → Mar 31
	if _, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1, EffectiveTo: &mar31,
	}); err != nil {
		t.Fatal(err)
	}
	// R_new: Apr 1 → open-ended
	rnewID, err := svc.Create(ctx, workspaceID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 20000, EffectiveFrom: apr1,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Entry in EDT (-04:00) at 23:30 local == 03:30Z Apr 1 … use a clearer
	// case: 2026-03-31T23:30:00-04:00 == 2026-04-01T03:30:00Z.
	loc := time.FixedZone("EDT", -4*3600)
	localT := time.Date(2026, 3, 31, 23, 30, 0, 0, loc)
	res, err := svc.Resolve(ctx, workspaceID, projectID, localT)
	if err != nil || !res.Found {
		t.Fatalf("resolve utc boundary: %+v err=%v", res, err)
	}
	if res.RuleID != rnewID {
		t.Fatalf("expected R_new (%s) got %s", rnewID, res.RuleID)
	}
}
