package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkspaceOffender names a workspace that still has at least one closed
// billable time entry with a NULL rate snapshot. It is returned by
// CheckRateSnapshots and printed by the `check-rate-snapshots` subcommand
// so an operator can target remediation via `backfill-rate-snapshots`.
type WorkspaceOffender struct {
	WorkspaceID uuid.UUID
	Count       int64
}

// CheckRateSnapshotsSummary is the structured result of a snapshot check.
type CheckRateSnapshotsSummary struct {
	Offenders []WorkspaceOffender
	Total     int64
}

// CheckRateSnapshots reports every workspace with closed billable time
// entries whose rate snapshot (`rate_rule_id`) is NULL. A zero total means
// the read path is safe to deploy under the snapshot-only invariant; any
// non-zero total means the backfill has not yet covered all historical rows
// in this database. Running entries (`ended_at IS NULL`) are ignored.
func CheckRateSnapshots(ctx context.Context, pool *pgxpool.Pool) (CheckRateSnapshotsSummary, error) {
	var sum CheckRateSnapshotsSummary
	rows, err := pool.Query(ctx, `
		SELECT workspace_id, count(*)
		FROM time_entries
		WHERE ended_at IS NOT NULL
		  AND is_billable = true
		  AND rate_rule_id IS NULL
		GROUP BY workspace_id
		ORDER BY workspace_id
	`)
	if err != nil {
		return sum, err
	}
	defer rows.Close()
	for rows.Next() {
		var off WorkspaceOffender
		if err := rows.Scan(&off.WorkspaceID, &off.Count); err != nil {
			return sum, err
		}
		sum.Offenders = append(sum.Offenders, off)
		sum.Total += off.Count
	}
	return sum, rows.Err()
}

// cmdCheckRateSnapshots wires the subcommand invoked from main(). Exits the
// process with status 1 when any offender is found; prints a per-workspace
// summary to stdout either way.
func cmdCheckRateSnapshots(ctx context.Context, pool *pgxpool.Pool) error {
	sum, err := CheckRateSnapshots(ctx, pool)
	if err != nil {
		return err
	}
	if sum.Total == 0 {
		fmt.Println("check-rate-snapshots: OK (0 closed billable entries with a NULL snapshot)")
		return nil
	}
	fmt.Fprintf(os.Stderr, "check-rate-snapshots: FAIL (%d closed billable entr(ies) with a NULL snapshot)\n", sum.Total)
	for _, off := range sum.Offenders {
		fmt.Fprintf(os.Stderr, "  workspace=%s count=%d\n", off.WorkspaceID, off.Count)
	}
	fmt.Fprintln(os.Stderr, "run `make backfill-rate-snapshots` to remediate")
	os.Exit(1)
	return nil
}
