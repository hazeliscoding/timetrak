package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"timetrak/internal/rates"
	"timetrak/internal/shared/db"
)

// ErrAdjacencyConflict is returned by BackfillRateSnapshots when the
// pre-flight probe finds same-level `rate_rules` pairs with
// `A.effective_to = B.effective_from`. The operator must resolve the
// adjacency conflict before the backfill can proceed.
var ErrAdjacencyConflict = errors.New("rate_rules have same-level shared-boundary adjacency; fix before backfill")

// BackfillSummary is the structured result of a backfill run.
type BackfillSummary struct {
	Workspaces   int
	Backfilled   int
	NoRate       int
	Elapsed      time.Duration
	DryRun       bool
	AdjacencyBad []BackfillConflict // populated when ErrAdjacencyConflict is returned
}

// BackfillConflict identifies a pair of rules with shared boundaries.
type BackfillConflict struct {
	WorkspaceID uuid.UUID
	RuleA       uuid.UUID
	RuleB       uuid.UUID
	BoundaryAt  time.Time
}

// BackfillRateSnapshots iterates every workspace and for each closed
// time_entry with a NULL snapshot, resolves the historical rate via
// rates.Service.Resolve and writes `rate_rule_id`, `hourly_rate_minor`,
// `currency_code` atomically. Idempotent: re-running finds zero rows.
//
// Before starting the iteration, a pre-flight probe SELECTs any same-level
// rate-rule pairs where `A.effective_to = B.effective_from` and aborts with
// ErrAdjacencyConflict if any exist — once a stricter `assertNoOverlap`
// landed, that configuration has no well-defined answer for Resolve.
//
// If `dryRun` is true, the routine reports the same counts but writes nothing.
//
// NOTE: the reporting read path no longer has a resolve-at-read fallback for
// closed entries with a NULL snapshot — the `REPORTING_RESOLVE_FALLBACK`
// escape hatch was removed in change `tighten-reporting-snapshot-only`. That
// makes this backfill a hard deploy prerequisite for any database predating
// migration 0013: without it, affected closed billable entries will silently
// contribute zero and inflate the `Entries without a rate` counter until
// `check-rate-snapshots` is run. Pair this command with
// `check-rate-snapshots` as the CI / deploy gate.
func BackfillRateSnapshots(ctx context.Context, pool *pgxpool.Pool, dryRun bool) (BackfillSummary, error) {
	start := time.Now()
	sum := BackfillSummary{DryRun: dryRun}

	conflicts, err := findAdjacencyConflicts(ctx, pool)
	if err != nil {
		return sum, fmt.Errorf("pre-flight adjacency probe: %w", err)
	}
	if len(conflicts) > 0 {
		sum.AdjacencyBad = conflicts
		return sum, fmt.Errorf("%w: %d pair(s)", ErrAdjacencyConflict, len(conflicts))
	}

	dbPool := &db.Pool{Pool: pool}
	ratesSvc := rates.NewService(dbPool)

	wsRows, err := pool.Query(ctx, `
		SELECT DISTINCT workspace_id
		FROM time_entries
		WHERE ended_at IS NOT NULL AND rate_rule_id IS NULL
	`)
	if err != nil {
		return sum, err
	}
	var workspaces []uuid.UUID
	for wsRows.Next() {
		var w uuid.UUID
		if err := wsRows.Scan(&w); err != nil {
			wsRows.Close()
			return sum, err
		}
		workspaces = append(workspaces, w)
	}
	wsRows.Close()
	if err := wsRows.Err(); err != nil {
		return sum, err
	}
	sum.Workspaces = len(workspaces)

	for _, wsID := range workspaces {
		rows, err := pool.Query(ctx, `
			SELECT id, project_id, started_at
			FROM time_entries
			WHERE workspace_id = $1 AND ended_at IS NOT NULL AND rate_rule_id IS NULL
		`, wsID)
		if err != nil {
			return sum, err
		}
		type row struct {
			id      uuid.UUID
			pid     uuid.UUID
			started time.Time
		}
		var batch []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.pid, &r.started); err != nil {
				rows.Close()
				return sum, err
			}
			batch = append(batch, r)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return sum, err
		}

		for _, r := range batch {
			res, err := ratesSvc.Resolve(ctx, wsID, r.pid, r.started)
			if err != nil {
				return sum, fmt.Errorf("resolve entry %s: %w", r.id, err)
			}
			if !res.Found {
				sum.NoRate++
				continue
			}
			if dryRun {
				sum.Backfilled++
				continue
			}
			// Per the atomic CHECK constraint, all three snapshot columns
			// must be set or all NULL; here we set all three.
			_, err = pool.Exec(ctx, `
				UPDATE time_entries
				SET rate_rule_id = $3, hourly_rate_minor = $4, currency_code = $5, updated_at = now()
				WHERE id = $1 AND workspace_id = $2
				  AND rate_rule_id IS NULL AND ended_at IS NOT NULL
			`, r.id, wsID, res.RuleID, res.HourlyRateMinor, res.CurrencyCode)
			if err != nil {
				return sum, fmt.Errorf("update entry %s: %w", r.id, err)
			}
			sum.Backfilled++
		}
	}

	sum.Elapsed = time.Since(start)
	return sum, nil
}

// findAdjacencyConflicts returns same-level rate_rule pairs whose windows
// share a boundary. "Same level" means: same workspace AND same scope
// (workspace-default, or tied to the same client, or tied to the same
// project). Because the overlap check now rejects shared boundaries, any
// existing such pairs are data-quality debt.
func findAdjacencyConflicts(ctx context.Context, pool *pgxpool.Pool) ([]BackfillConflict, error) {
	// Pair (a, b) where `a` ends where `b` begins at the same workspace/scope.
	// The pair is directional (a.effective_to = b.effective_from) so UUID
	// ordering is irrelevant — each distinct (earlier, later) pair surfaces once.
	rows, err := pool.Query(ctx, `
		SELECT a.workspace_id, a.id, b.id, a.effective_to
		FROM rate_rules a
		JOIN rate_rules b
		  ON a.workspace_id = b.workspace_id
		 AND a.id <> b.id
		 AND a.effective_to IS NOT NULL
		 AND a.effective_to = b.effective_from
		 AND COALESCE(a.project_id::text, '') = COALESCE(b.project_id::text, '')
		 AND COALESCE(a.client_id::text, '')  = COALESCE(b.client_id::text, '')
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BackfillConflict
	for rows.Next() {
		var c BackfillConflict
		var boundary time.Time
		if err := rows.Scan(&c.WorkspaceID, &c.RuleA, &c.RuleB, &boundary); err != nil {
			return nil, err
		}
		c.BoundaryAt = boundary
		out = append(out, c)
	}
	return out, rows.Err()
}

// cmdBackfillRateSnapshots wires the subcommand invoked from main().
// Supported flags: --dry-run.
func cmdBackfillRateSnapshots(ctx context.Context, pool *pgxpool.Pool, args []string) error {
	dryRun := false
	for _, a := range args {
		switch a {
		case "--dry-run", "-n":
			dryRun = true
		default:
			return fmt.Errorf("unknown flag %q", a)
		}
	}
	sum, err := BackfillRateSnapshots(ctx, pool, dryRun)
	if err != nil {
		if errors.Is(err, ErrAdjacencyConflict) {
			fmt.Fprintln(os.Stderr, "adjacency conflict — fix these rule pairs before backfill:")
			for _, c := range sum.AdjacencyBad {
				fmt.Fprintf(os.Stderr, "  workspace=%s a=%s b=%s boundary=%s\n",
					c.WorkspaceID, c.RuleA, c.RuleB, c.BoundaryAt.Format(time.RFC3339))
			}
		}
		return err
	}
	// Structured summary log; non-zero no_rate is informational.
	fmt.Printf("backfill complete: backfilled=%d workspaces=%d no_rate=%d elapsed=%s dry_run=%v\n",
		sum.Backfilled, sum.Workspaces, sum.NoRate, sum.Elapsed, sum.DryRun)
	return nil
}
