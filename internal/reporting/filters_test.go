package reporting_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/rates"
	"timetrak/internal/reporting"
	"timetrak/internal/shared/testdb"
)

// TestReportClientFilterNarrowsAllAggregates seeds entries across two
// clients in W1 and asserts every aggregate is reduced to C1's contribution
// when `ClientID = C1`.
func TestReportClientFilterNarrowsAllAggregates(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	// Seed a second client + project in W1.
	c2 := uuid.New()
	p2 := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO clients (id, workspace_id, name) VALUES ($1,$2,'W1-C2')`,
		c2, f.WorkspaceA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id, workspace_id, client_id, name) VALUES ($1,$2,$3,'W1-P2')`,
		p2, f.WorkspaceA, c2); err != nil {
		t.Fatal(err)
	}

	feb := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)

	// Create a workspace-default rate rule so the snapshot FK is satisfied.
	ratesSvc := rates.NewService(pool)
	ruleID, err := ratesSvc.Create(ctx, f.WorkspaceA, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000,
		EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	// C1 billable with snapshot — 1h @ 100.00 USD/h.
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable,rate_rule_id,hourly_rate_minor,currency_code)
		VALUES ($1,$2,$3,$4,$5,3600,true,$6,10000,'USD')
	`, f.WorkspaceA, f.UserA, f.ProjectA, feb, feb.Add(time.Hour), ruleID); err != nil {
		t.Fatal(err)
	}
	// C2 billable with snapshot — 2h @ 50.00 USD/h.
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable,rate_rule_id,hourly_rate_minor,currency_code)
		VALUES ($1,$2,$3,$4,$5,7200,true,$6,5000,'USD')
	`, f.WorkspaceA, f.UserA, p2, feb.Add(2*time.Hour), feb.Add(4*time.Hour), ruleID); err != nil {
		t.Fatal(err)
	}
	// C2 billable with NULL snapshot (should add to NoRateCount when not filtered).
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable)
		VALUES ($1,$2,$3,$4,$5,1800,true)
	`, f.WorkspaceA, f.UserA, p2, feb.Add(5*time.Hour), feb.Add(5*time.Hour+30*time.Minute)); err != nil {
		t.Fatal(err)
	}

	svc := reporting.NewService(pool)
	rng := reporting.Range{
		From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
	}

	// Unfiltered: totals include everything.
	all, err := svc.ReportWithFilters(ctx, f.WorkspaceA, f.UserA, rng, "day", reporting.Filters{})
	if err != nil {
		t.Fatal(err)
	}
	if all.Totals.TotalSeconds != 3600+7200+1800 {
		t.Fatalf("unfiltered total = %d, want %d", all.Totals.TotalSeconds, 3600+7200+1800)
	}
	if all.NoRateCount != 1 {
		t.Fatalf("unfiltered NoRateCount = %d, want 1", all.NoRateCount)
	}
	// USD estimated: C1 = 10000 + C2 = 10000 (2h * 5000/h) = 20000.
	if got := all.Totals.EstimatedByCurrency["USD"]; got != 20000 {
		t.Fatalf("unfiltered USD total = %d, want 20000", got)
	}

	// Filter by C1: totals reflect only C1.
	c1Only, err := svc.ReportWithFilters(ctx, f.WorkspaceA, f.UserA, rng, "day",
		reporting.Filters{ClientID: f.ClientA})
	if err != nil {
		t.Fatal(err)
	}
	if c1Only.Totals.TotalSeconds != 3600 {
		t.Fatalf("C1 filter total = %d, want 3600", c1Only.Totals.TotalSeconds)
	}
	if c1Only.Totals.BillableSeconds != 3600 {
		t.Fatalf("C1 filter billable = %d, want 3600", c1Only.Totals.BillableSeconds)
	}
	if got := c1Only.Totals.EstimatedByCurrency["USD"]; got != 10000 {
		t.Fatalf("C1 filter USD = %d, want 10000", got)
	}
	if c1Only.NoRateCount != 0 {
		t.Fatalf("C1 filter NoRateCount = %d, want 0 (only C2 has a NULL-snapshot entry)", c1Only.NoRateCount)
	}
}

// TestReportBillableTriState covers all three billable filter values.
func TestReportBillableTriState(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)
	feb := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)

	ratesSvc := rates.NewService(pool)
	ruleID, err := ratesSvc.Create(ctx, f.WorkspaceA, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000,
		EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Billable with snapshot.
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable,rate_rule_id,hourly_rate_minor,currency_code)
		VALUES ($1,$2,$3,$4,$5,3600,true,$6,10000,'USD')
	`, f.WorkspaceA, f.UserA, f.ProjectA, feb, feb.Add(time.Hour), ruleID); err != nil {
		t.Fatal(err)
	}
	// Non-billable.
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable)
		VALUES ($1,$2,$3,$4,$5,1800,false)
	`, f.WorkspaceA, f.UserA, f.ProjectA, feb.Add(2*time.Hour), feb.Add(2*time.Hour+30*time.Minute)); err != nil {
		t.Fatal(err)
	}

	svc := reporting.NewService(pool)
	rng := reporting.Range{
		From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
	}

	// billable=yes: NonBillable MUST be 0.
	yes, _ := svc.ReportWithFilters(ctx, f.WorkspaceA, f.UserA, rng, "day", reporting.Filters{Billable: "yes"})
	if yes.Totals.NonBillableSeconds != 0 {
		t.Fatalf("billable=yes NonBillable = %d, want 0", yes.Totals.NonBillableSeconds)
	}
	if yes.Totals.BillableSeconds != 3600 {
		t.Fatalf("billable=yes Billable = %d, want 3600", yes.Totals.BillableSeconds)
	}

	// billable=no: Billable MUST be 0, EstimatedByCurrency empty, NoRateCount 0.
	no, _ := svc.ReportWithFilters(ctx, f.WorkspaceA, f.UserA, rng, "day", reporting.Filters{Billable: "no"})
	if no.Totals.BillableSeconds != 0 {
		t.Fatalf("billable=no Billable = %d, want 0", no.Totals.BillableSeconds)
	}
	if len(no.Totals.EstimatedByCurrency) != 0 {
		t.Fatalf("billable=no EstimatedByCurrency = %v, want empty", no.Totals.EstimatedByCurrency)
	}
	if no.NoRateCount != 0 {
		t.Fatalf("billable=no NoRateCount = %d, want 0", no.NoRateCount)
	}
	if no.Totals.NonBillableSeconds != 1800 {
		t.Fatalf("billable=no NonBillable = %d, want 1800", no.Totals.NonBillableSeconds)
	}

	// billable="": both included.
	all, _ := svc.ReportWithFilters(ctx, f.WorkspaceA, f.UserA, rng, "day", reporting.Filters{})
	if all.Totals.TotalSeconds != 3600+1800 {
		t.Fatalf("billable='' total = %d, want %d", all.Totals.TotalSeconds, 3600+1800)
	}
}

// TestReportMultiCurrencyGrandTotal seeds entries in USD and EUR and
// confirms both keys appear in the per-currency grand total.
func TestReportMultiCurrencyGrandTotal(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)
	feb := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)

	ratesSvc := rates.NewService(pool)
	usdRule, err := ratesSvc.Create(ctx, f.WorkspaceA, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000,
		EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Second rule with non-overlapping scope (per-project) to satisfy any
	// adjacency checks.
	eurRule := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO rate_rules (id, workspace_id, project_id, client_id, currency_code, hourly_rate_minor, effective_from)
		VALUES ($1, $2, $3, NULL, 'EUR', 8000, $4)
	`, eurRule, f.WorkspaceA, f.ProjectA, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable,rate_rule_id,hourly_rate_minor,currency_code)
		VALUES ($1,$2,$3,$4,$5,3600,true,$6,10000,'USD')
	`, f.WorkspaceA, f.UserA, f.ProjectA, feb, feb.Add(time.Hour), usdRule); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable,rate_rule_id,hourly_rate_minor,currency_code)
		VALUES ($1,$2,$3,$4,$5,3600,true,$6,8000,'EUR')
	`, f.WorkspaceA, f.UserA, f.ProjectA, feb.Add(2*time.Hour), feb.Add(3*time.Hour), eurRule); err != nil {
		t.Fatal(err)
	}

	svc := reporting.NewService(pool)
	rng := reporting.Range{
		From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
	}
	rep, _ := svc.ReportWithFilters(ctx, f.WorkspaceA, f.UserA, rng, "day", reporting.Filters{})
	if got := rep.Totals.EstimatedByCurrency["USD"]; got != 10000 {
		t.Fatalf("USD = %d, want 10000", got)
	}
	if got := rep.Totals.EstimatedByCurrency["EUR"]; got != 8000 {
		t.Fatalf("EUR = %d, want 8000", got)
	}
	if len(rep.Totals.EstimatedByCurrency) != 2 {
		t.Fatalf("currencies = %v, want 2 keys", rep.Totals.EstimatedByCurrency)
	}
}
