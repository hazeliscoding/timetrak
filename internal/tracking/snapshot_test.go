package tracking_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/rates"
	"timetrak/internal/shared/clock"
	"timetrak/internal/shared/testdb"
	"timetrak/internal/tracking"
)

// TestTimerStopWritesRateSnapshot verifies StopTimer resolves the current
// rate inside the same transaction and writes the three snapshot columns.
func TestTimerStopWritesRateSnapshot(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	ratesSvc := rates.NewService(pool)
	trk := tracking.NewService(pool, clock.System{}, ratesSvc)

	// Workspace-default rule active now.
	jan1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	ruleID, err := ratesSvc.Create(ctx, f.WorkspaceA, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 12500, EffectiveFrom: jan1,
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	if _, err := trk.StartTimer(ctx, f.WorkspaceA, f.UserA, tracking.StartInput{ProjectID: f.ProjectA}); err != nil {
		t.Fatalf("start: %v", err)
	}
	// Backdate so duration is positive.
	if _, err := pool.Exec(ctx, `UPDATE time_entries SET started_at = now() - interval '1 hour' WHERE workspace_id = $1 AND ended_at IS NULL`, f.WorkspaceA); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	entry, err := trk.StopTimer(ctx, f.WorkspaceA, f.UserA)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}

	var gotRule uuid.UUID
	var gotMinor int64
	var gotCcy string
	if err := pool.QueryRow(ctx, `SELECT rate_rule_id, hourly_rate_minor, currency_code FROM time_entries WHERE id = $1`, entry.ID).Scan(&gotRule, &gotMinor, &gotCcy); err != nil {
		t.Fatalf("scan snapshot: %v", err)
	}
	if gotRule != ruleID || gotMinor != 12500 || gotCcy != "USD" {
		t.Fatalf("snapshot mismatch: rule=%s minor=%d ccy=%s", gotRule, gotMinor, gotCcy)
	}
}

// TestTimerStopWithNoRuleWritesNullSnapshot verifies that when no rule is
// defined, the three snapshot columns are NULL (atomic CHECK enforces all-or-none).
func TestTimerStopWithNoRuleWritesNullSnapshot(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	ratesSvc := rates.NewService(pool)
	trk := tracking.NewService(pool, clock.System{}, ratesSvc)

	if _, err := trk.StartTimer(ctx, f.WorkspaceA, f.UserA, tracking.StartInput{ProjectID: f.ProjectA}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE time_entries SET started_at = now() - interval '1 hour' WHERE workspace_id = $1 AND ended_at IS NULL`, f.WorkspaceA); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	entry, err := trk.StopTimer(ctx, f.WorkspaceA, f.UserA)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}

	var rule *uuid.UUID
	var minor *int64
	var ccy *string
	if err := pool.QueryRow(ctx, `SELECT rate_rule_id, hourly_rate_minor, currency_code FROM time_entries WHERE id = $1`, entry.ID).Scan(&rule, &minor, &ccy); err != nil {
		t.Fatal(err)
	}
	if rule != nil || minor != nil || ccy != nil {
		t.Fatalf("expected all-NULL snapshot; got rule=%v minor=%v ccy=%v", rule, minor, ccy)
	}
}

// TestEntryEditReSnapshotsRate verifies that editing a rate-determining field
// (project_id here) causes the snapshot to be re-resolved from the new project.
func TestEntryEditReSnapshotsRate(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	ratesSvc := rates.NewService(pool)
	trk := tracking.NewService(pool, clock.System{}, ratesSvc)

	// Create two projects in W1, each with its own project-level rule.
	// ProjectA already exists from fixture; create a second project.
	var projectA2 uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO projects (workspace_id, client_id, name) VALUES ($1,$2,'P-alt') RETURNING id`, f.WorkspaceA, f.ClientA).Scan(&projectA2); err != nil {
		t.Fatal(err)
	}
	jan1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	// ProjectA rule 10000
	rule1, err := ratesSvc.Create(ctx, f.WorkspaceA, rates.Input{
		ProjectID: f.ProjectA, CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1,
	})
	if err != nil {
		t.Fatal(err)
	}
	// ProjectA2 rule 20000
	rule2, err := ratesSvc.Create(ctx, f.WorkspaceA, rates.Input{
		ProjectID: projectA2, CurrencyCode: "USD", HourlyRateMinor: 20000, EffectiveFrom: jan1,
	})
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now().UTC().Add(-2 * time.Hour)
	end := start.Add(time.Hour)
	entry, err := trk.CreateManual(ctx, f.WorkspaceA, f.UserA, tracking.ManualInput{
		ProjectID: f.ProjectA, StartedAt: start, EndedAt: end, IsBillable: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Confirm initial snapshot.
	var gotRule uuid.UUID
	var gotMinor int64
	if err := pool.QueryRow(ctx, `SELECT rate_rule_id, hourly_rate_minor FROM time_entries WHERE id=$1`, entry.ID).Scan(&gotRule, &gotMinor); err != nil {
		t.Fatal(err)
	}
	if gotRule != rule1 || gotMinor != 10000 {
		t.Fatalf("initial snapshot: rule=%s minor=%d", gotRule, gotMinor)
	}

	// Edit to ProjectA2 — same interval, billable unchanged, but project changed.
	if _, err := trk.Edit(ctx, f.WorkspaceA, f.UserA, entry.ID, tracking.ManualInput{
		ProjectID: projectA2, StartedAt: start, EndedAt: end, IsBillable: true,
	}); err != nil {
		t.Fatal(err)
	}

	if err := pool.QueryRow(ctx, `SELECT rate_rule_id, hourly_rate_minor FROM time_entries WHERE id=$1`, entry.ID).Scan(&gotRule, &gotMinor); err != nil {
		t.Fatal(err)
	}
	if gotRule != rule2 || gotMinor != 20000 {
		t.Fatalf("re-snapshot: rule=%s minor=%d", gotRule, gotMinor)
	}
}

// TestEntryEditUnrelatedFieldsPreservesSnapshot verifies that editing a non-
// rate-determining field (description) leaves the snapshot untouched. To
// demonstrate "leaves untouched" conclusively, we first mutate the snapshot
// row directly to an out-of-band value and verify Edit does not clobber it.
func TestEntryEditUnrelatedFieldsPreservesSnapshot(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	ratesSvc := rates.NewService(pool)
	trk := tracking.NewService(pool, clock.System{}, ratesSvc)

	jan1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	ruleID, err := ratesSvc.Create(ctx, f.WorkspaceA, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: jan1,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = ruleID

	// Truncate to microseconds because PostgreSQL timestamptz stores at µs
	// precision. Without this, round-tripping through the DB drops nanoseconds
	// and the equality compare in Edit sees a false "changed" diff.
	start := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Microsecond)
	end := start.Add(time.Hour)
	entry, err := trk.CreateManual(ctx, f.WorkspaceA, f.UserA, tracking.ManualInput{
		ProjectID: f.ProjectA, StartedAt: start, EndedAt: end, IsBillable: true,
		Description: "before",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seed an out-of-band snapshot amount so we can detect re-snapshot clobbers.
	if _, err := pool.Exec(ctx, `UPDATE time_entries SET hourly_rate_minor = 99999 WHERE id = $1`, entry.ID); err != nil {
		t.Fatal(err)
	}

	// Edit description only — every rate-determining field unchanged.
	if _, err := trk.Edit(ctx, f.WorkspaceA, f.UserA, entry.ID, tracking.ManualInput{
		ProjectID: f.ProjectA, StartedAt: start, EndedAt: end, IsBillable: true,
		Description: "after",
	}); err != nil {
		t.Fatal(err)
	}

	var minor int64
	if err := pool.QueryRow(ctx, `SELECT hourly_rate_minor FROM time_entries WHERE id=$1`, entry.ID).Scan(&minor); err != nil {
		t.Fatal(err)
	}
	if minor != 99999 {
		t.Fatalf("snapshot was re-written on unrelated edit: got %d, expected 99999 sentinel", minor)
	}
}

// TestCrossWorkspaceRuleDoesNotLeakIntoSnapshot verifies workspace scoping in
// Resolve. A rule in W2 that would otherwise match project P1 (if workspace
// scope were ignored) must NOT influence snapshots in W1.
func TestCrossWorkspaceRuleDoesNotLeakIntoSnapshot(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	ratesSvc := rates.NewService(pool)
	trk := tracking.NewService(pool, clock.System{}, ratesSvc)

	jan1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	// Create a workspace-default rule in W2 — it should never affect W1 entries.
	if _, err := ratesSvc.Create(ctx, f.WorkspaceB, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 99999, EffectiveFrom: jan1,
	}); err != nil {
		t.Fatal(err)
	}

	// Create an entry in W1 with no W1 rule.
	start := time.Now().UTC().Add(-2 * time.Hour)
	end := start.Add(time.Hour)
	entry, err := trk.CreateManual(ctx, f.WorkspaceA, f.UserA, tracking.ManualInput{
		ProjectID: f.ProjectA, StartedAt: start, EndedAt: end, IsBillable: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var rule *uuid.UUID
	var minor *int64
	if err := pool.QueryRow(ctx, `SELECT rate_rule_id, hourly_rate_minor FROM time_entries WHERE id=$1`, entry.ID).Scan(&rule, &minor); err != nil {
		t.Fatal(err)
	}
	if rule != nil || minor != nil {
		t.Fatalf("cross-workspace leak: rule=%v minor=%v", rule, minor)
	}
}
