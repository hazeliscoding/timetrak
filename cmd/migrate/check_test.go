package main

import (
	"context"
	"testing"
	"time"
)

// TestCheckRateSnapshotsFlagsNullSnapshotEntry verifies a seeded closed
// billable entry with NULL snapshot is surfaced as an offender; after
// backfill the same check reports zero.
func TestCheckRateSnapshotsFlagsNullSnapshotEntry(t *testing.T) {
	pool := openPoolForTest(t)
	ctx := context.Background()
	workspaceID, _, _, _ := seedOneRuleAndEntry(t, pool, "check-null")

	sum, err := CheckRateSnapshots(ctx, pool)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if sum.Total != 1 {
		t.Fatalf("total = %d, want 1 (one NULL-snapshot closed billable entry seeded)", sum.Total)
	}
	if len(sum.Offenders) != 1 || sum.Offenders[0].WorkspaceID != workspaceID {
		t.Fatalf("offenders = %+v, want one entry for %s", sum.Offenders, workspaceID)
	}

	// After backfill, offenders must be zero.
	if _, err := BackfillRateSnapshots(ctx, pool, false); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	sum, err = CheckRateSnapshots(ctx, pool)
	if err != nil {
		t.Fatalf("check after backfill: %v", err)
	}
	if sum.Total != 0 || len(sum.Offenders) != 0 {
		t.Fatalf("after backfill: total=%d offenders=%+v, want zeroes", sum.Total, sum.Offenders)
	}
}

// TestCheckRateSnapshotsIgnoresRunningTimer confirms a running timer (no
// ended_at, no snapshot) never causes the check to fail.
func TestCheckRateSnapshotsIgnoresRunningTimer(t *testing.T) {
	pool := openPoolForTest(t)
	ctx := context.Background()
	workspaceID, projectID, _, _ := seedOneRuleAndEntry(t, pool, "check-running")

	// Backfill the seeded closed entry so only a fresh running timer remains without a snapshot.
	if _, err := BackfillRateSnapshots(ctx, pool, false); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	var userID string
	if err := pool.QueryRow(ctx, `SELECT user_id::text FROM workspace_members WHERE workspace_id=$1 LIMIT 1`, workspaceID).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,duration_seconds,is_billable)
		VALUES ($1,$2,$3,$4,0,true)
	`, workspaceID, userID, projectID, time.Now().UTC().Add(-30*time.Minute)); err != nil {
		t.Fatal(err)
	}

	sum, err := CheckRateSnapshots(ctx, pool)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if sum.Total != 0 {
		t.Fatalf("running timer should not count as offender; total = %d", sum.Total)
	}
}
