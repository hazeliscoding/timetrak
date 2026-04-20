package reporting_test

import (
	"context"
	"testing"
	"time"

	"timetrak/internal/reporting"
	"timetrak/internal/shared/testdb"
)

// TestReportDSTSpringForward seeds four 1-hour billable entries on
// America/New_York's 2026-03-08 (a 23-hour local day due to DST). The day
// total MUST be exactly 4*3600s and nothing MUST leak to 2026-03-09.
func TestReportDSTSpringForward(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	if _, err := pool.Exec(ctx, `UPDATE workspaces SET reporting_timezone='America/New_York' WHERE id=$1`, f.WorkspaceA); err != nil {
		t.Fatal(err)
	}

	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	// Four 1-hour entries: 08:00, 10:00, 13:00, 16:00 local on 2026-03-08.
	starts := []time.Time{
		time.Date(2026, 3, 8, 8, 0, 0, 0, ny),
		time.Date(2026, 3, 8, 10, 0, 0, 0, ny),
		time.Date(2026, 3, 8, 13, 0, 0, 0, ny),
		time.Date(2026, 3, 8, 16, 0, 0, 0, ny),
	}
	for _, s := range starts {
		if _, err := pool.Exec(ctx, `
			INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable)
			VALUES ($1,$2,$3,$4,$5,3600,true)
		`, f.WorkspaceA, f.UserA, f.ProjectA, s, s.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}

	svc := reporting.NewService(pool)
	rng := reporting.Range{
		From: time.Date(2026, 3, 8, 0, 0, 0, 0, ny),
		To:   time.Date(2026, 3, 8, 0, 0, 0, 0, ny),
	}
	rep, err := svc.Report(ctx, f.WorkspaceA, f.UserA, rng, "day")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Totals.TotalSeconds != 4*3600 {
		t.Fatalf("day total = %d, want 14400 (DST spring-forward)", rep.Totals.TotalSeconds)
	}
	if len(rep.ByDay) != 1 || rep.ByDay[0].Label != "2026-03-08" {
		t.Fatalf("ByDay = %+v, want a single 2026-03-08 bucket", rep.ByDay)
	}
	if rep.ByDay[0].TotalSeconds != 4*3600 {
		t.Fatalf("ByDay[0] total = %d, want 14400", rep.ByDay[0].TotalSeconds)
	}
}

// TestReportDSTFallBack: four 1-hour entries on 2026-11-01 America/New_York
// (a 25-hour local day). Day total MUST be 4*3600s and nothing leaks to
// 2026-10-31 or 2026-11-02.
func TestReportDSTFallBack(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	if _, err := pool.Exec(ctx, `UPDATE workspaces SET reporting_timezone='America/New_York' WHERE id=$1`, f.WorkspaceA); err != nil {
		t.Fatal(err)
	}
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	starts := []time.Time{
		time.Date(2026, 11, 1, 8, 0, 0, 0, ny),
		time.Date(2026, 11, 1, 10, 0, 0, 0, ny),
		time.Date(2026, 11, 1, 13, 0, 0, 0, ny),
		time.Date(2026, 11, 1, 16, 0, 0, 0, ny),
	}
	for _, s := range starts {
		if _, err := pool.Exec(ctx, `
			INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable)
			VALUES ($1,$2,$3,$4,$5,3600,true)
		`, f.WorkspaceA, f.UserA, f.ProjectA, s, s.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}

	svc := reporting.NewService(pool)
	rng := reporting.Range{
		From: time.Date(2026, 10, 31, 0, 0, 0, 0, ny),
		To:   time.Date(2026, 11, 2, 0, 0, 0, 0, ny),
	}
	rep, err := svc.Report(ctx, f.WorkspaceA, f.UserA, rng, "day")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Totals.TotalSeconds != 4*3600 {
		t.Fatalf("total = %d, want 14400 (DST fall-back)", rep.Totals.TotalSeconds)
	}
	var got map[string]int64 = map[string]int64{}
	for _, b := range rep.ByDay {
		got[b.Label] = b.TotalSeconds
	}
	if got["2026-11-01"] != 4*3600 {
		t.Fatalf("ByDay[2026-11-01] = %d, want 14400", got["2026-11-01"])
	}
	if got["2026-10-31"] != 0 {
		t.Fatalf("leaked into 2026-10-31: %d seconds", got["2026-10-31"])
	}
	if got["2026-11-02"] != 0 {
		t.Fatalf("leaked into 2026-11-02: %d seconds", got["2026-11-02"])
	}
}

// TestReportLateNightLocalEntry: an entry that spans 23:30-00:30 local time
// MUST be attributed to the start date in local tz, not the UTC date.
func TestReportLateNightLocalEntry(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)
	if _, err := pool.Exec(ctx, `UPDATE workspaces SET reporting_timezone='America/New_York' WHERE id=$1`, f.WorkspaceA); err != nil {
		t.Fatal(err)
	}
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	// 2026-04-17 23:30 NY = 2026-04-18 03:30 UTC.
	start := time.Date(2026, 4, 17, 23, 30, 0, 0, ny)
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable)
		VALUES ($1,$2,$3,$4,$5,3600,true)
	`, f.WorkspaceA, f.UserA, f.ProjectA, start, start.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	svc := reporting.NewService(pool)
	rng := reporting.Range{
		From: time.Date(2026, 4, 17, 0, 0, 0, 0, ny),
		To:   time.Date(2026, 4, 17, 0, 0, 0, 0, ny),
	}
	rep, err := svc.Report(ctx, f.WorkspaceA, f.UserA, rng, "day")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Totals.TotalSeconds != 3600 {
		t.Fatalf("2026-04-17 total = %d, want 3600 (tz-local bucketing)", rep.Totals.TotalSeconds)
	}
	// And MUST NOT appear on 2026-04-18.
	rng2 := reporting.Range{
		From: time.Date(2026, 4, 18, 0, 0, 0, 0, ny),
		To:   time.Date(2026, 4, 18, 0, 0, 0, 0, ny),
	}
	rep2, err := svc.Report(ctx, f.WorkspaceA, f.UserA, rng2, "day")
	if err != nil {
		t.Fatal(err)
	}
	if rep2.Totals.TotalSeconds != 0 {
		t.Fatalf("leaked into 2026-04-18: %d seconds (UTC-bucketing bug)", rep2.Totals.TotalSeconds)
	}
}
