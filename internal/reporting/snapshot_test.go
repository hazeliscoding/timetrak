package reporting_test

import (
	"context"
	"testing"
	"time"

	"timetrak/internal/rates"
	"timetrak/internal/reporting"
	"timetrak/internal/shared/testdb"
)

// Contract A: a closed billable entry with a snapshot contributes the
// snapshot amount, and a later edit to the underlying rate rule does not
// move the historical total. The reporting read path never consults
// rates.Service.Resolve for closed entries.
func TestReportBillableAmountReadsSnapshotNotResolver(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	ratesSvc := rates.NewService(pool)

	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ruleID, err := ratesSvc.Create(ctx, f.WorkspaceA, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1,
	})
	if err != nil {
		t.Fatal(err)
	}
	feb := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	_, err = pool.Exec(ctx, `
		INSERT INTO time_entries
		  (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable,
		   rate_rule_id,hourly_rate_minor,currency_code)
		VALUES ($1,$2,$3,$4,$5,3600,true,$6,10000,'USD')
	`, f.WorkspaceA, f.UserA, f.ProjectA, feb, feb.Add(time.Hour), ruleID)
	if err != nil {
		t.Fatal(err)
	}

	repSvc := reporting.NewService(pool)

	rng := reporting.Range{
		From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
	}
	before, err := repSvc.Report(ctx, f.WorkspaceA, f.UserA, rng, "day")
	if err != nil {
		t.Fatal(err)
	}
	if got := before.Totals.EstimatedByCurrency["USD"]; got != 10000 {
		t.Fatalf("before: USD total = %d, want 10000", got)
	}

	// Extend effective_to on the rule — a permitted edit (no amount change).
	// This is the reporting-stability test: the snapshot, not the live rule,
	// is authoritative for historical totals.
	future := time.Date(2030, 12, 31, 0, 0, 0, 0, time.UTC)
	if err := ratesSvc.Update(ctx, f.WorkspaceA, ruleID, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1, EffectiveTo: &future,
	}); err != nil {
		t.Fatalf("extend rule: %v", err)
	}

	after, err := repSvc.Report(ctx, f.WorkspaceA, f.UserA, rng, "day")
	if err != nil {
		t.Fatal(err)
	}
	if got := after.Totals.EstimatedByCurrency["USD"]; got != 10000 {
		t.Fatalf("after rule edit: USD total = %d, want 10000 (snapshot is authoritative)", got)
	}
}

// Contract B: a closed billable entry with a NULL snapshot contributes zero
// and is counted in EntriesWithoutRate / NoRateCount. No environment variable
// changes this; the rates service is not consulted.
func TestReportNoRateSnapshotContributesZeroAndCounted(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	start := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable)
		VALUES ($1,$2,$3,$4,$5,3600,true)
	`, f.WorkspaceA, f.UserA, f.ProjectA, start, start.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	repSvc := reporting.NewService(pool)
	rng := reporting.Range{
		From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
	}
	r, err := repSvc.Report(ctx, f.WorkspaceA, f.UserA, rng, "day")
	if err != nil {
		t.Fatal(err)
	}
	if r.NoRateCount != 1 {
		t.Fatalf("NoRateCount = %d, want 1", r.NoRateCount)
	}
	if got := r.Totals.EstimatedByCurrency["USD"]; got != 0 {
		t.Fatalf("NULL-snapshot entry contributed %d to USD, want 0", got)
	}
}

// TestReportingRunningTimerExcludedFromBillable verifies that an open entry
// (ended_at IS NULL) does not contribute to billable totals and is not
// counted in the `Entries without a rate` aggregate, even when its snapshot
// columns are NULL.
func TestReportingRunningTimerExcludedFromBillable(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	now := time.Now().UTC().Add(-1 * time.Hour)
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,duration_seconds,is_billable)
		VALUES ($1,$2,$3,$4,0,true)
	`, f.WorkspaceA, f.UserA, f.ProjectA, now); err != nil {
		t.Fatal(err)
	}

	repSvc := reporting.NewService(pool)
	dash, err := repSvc.Dashboard(ctx, f.WorkspaceA, f.UserA, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if got := dash.WeekEstimatedBillable["USD"]; got != 0 {
		t.Fatalf("running entry contributed to billable total: %d", got)
	}
	if dash.EntriesWithoutRate != 0 {
		t.Fatalf("running entry counted as Entry without rate: %d", dash.EntriesWithoutRate)
	}
	if !dash.RunningTimer {
		t.Fatalf("RunningTimer should be true")
	}
}

// TestReportNoRateCountsClosedEntriesOnly verifies that closed billable
// entries with NULL snapshot are counted in NoRateCount, while running
// entries are not.
func TestReportNoRateCountsClosedEntriesOnly(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	start := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable)
		VALUES ($1,$2,$3,$4,$5,3600,true)
	`, f.WorkspaceA, f.UserA, f.ProjectA, start, start.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	runStart := time.Now().UTC().Add(-time.Hour)
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,duration_seconds,is_billable)
		VALUES ($1,$2,$3,$4,0,true)
	`, f.WorkspaceA, f.UserA, f.ProjectA, runStart); err != nil {
		t.Fatal(err)
	}

	repSvc := reporting.NewService(pool)
	rng := reporting.Range{From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)}
	r, err := repSvc.Report(ctx, f.WorkspaceA, f.UserA, rng, "day")
	if err != nil {
		t.Fatal(err)
	}
	if r.NoRateCount != 1 {
		t.Fatalf("NoRateCount = %d, want 1", r.NoRateCount)
	}
}
