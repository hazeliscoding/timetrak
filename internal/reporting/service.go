// Package reporting aggregates time entries into totals and estimated billable
// amounts.
//
// Invariant (steady state): for closed (`ended_at IS NOT NULL`) entries, the
// per-entry rate snapshot (`rate_rule_id`, `hourly_rate_minor`,
// `currency_code`) persisted at stop/save time by the tracking domain is the
// sole source of truth for estimated billable amounts. The reporting read
// path MUST NOT call `rates.Service.Resolve` for any closed entry, regardless
// of environment configuration — doing so would let retroactive rule edits
// silently move historical totals. Closed billable entries with a NULL
// snapshot contribute zero and are counted toward `EntriesWithoutRate` /
// `NoRateCount`; the operator-facing fix is `migrate backfill-rate-snapshots`.
package reporting

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/shared/db"
)

// Service exposes report queries.
type Service struct {
	pool *db.Pool
}

// NewService constructs the reporting service. The reporting read path has no
// runtime dependency on rate resolution; closed entries are scored entirely
// from their own snapshot columns.
func NewService(pool *db.Pool) *Service {
	return &Service{pool: pool}
}

// DashboardSummary is the at-a-glance card set.
type DashboardSummary struct {
	TodayTotalSeconds       int64
	TodayBillableSeconds    int64
	TodayNonBillableSeconds int64
	WeekTotalSeconds        int64
	WeekBillableSeconds     int64
	WeekNonBillableSeconds  int64
	WeekEstimatedBillable   map[string]int64 // currency → minor units
	EntriesWithoutRate      int
	RunningTimer            bool
}

// Dashboard computes today's and this-week's totals.
func (s *Service) Dashboard(ctx context.Context, workspaceID, userID uuid.UUID, now time.Time) (DashboardSummary, error) {
	now = now.UTC()
	today := startOfDay(now)
	weekStart := startOfISOWeek(now)

	dayTotal, dayBill, err := s.totals(ctx, workspaceID, userID, today, today.Add(24*time.Hour-time.Second))
	if err != nil {
		return DashboardSummary{}, err
	}
	weekTotal, weekBill, err := s.totals(ctx, workspaceID, userID, weekStart, weekStart.Add(7*24*time.Hour-time.Second))
	if err != nil {
		return DashboardSummary{}, err
	}
	billable, noRate, err := s.estimateBillable(ctx, workspaceID, userID, weekStart, weekStart.Add(7*24*time.Hour-time.Second))
	if err != nil {
		return DashboardSummary{}, err
	}
	var running bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM time_entries WHERE workspace_id = $1 AND user_id = $2 AND ended_at IS NULL)
	`, workspaceID, userID).Scan(&running); err != nil {
		return DashboardSummary{}, err
	}
	return DashboardSummary{
		TodayTotalSeconds:       dayTotal,
		TodayBillableSeconds:    dayBill,
		TodayNonBillableSeconds: dayTotal - dayBill,
		WeekTotalSeconds:        weekTotal,
		WeekBillableSeconds:     weekBill,
		WeekNonBillableSeconds:  weekTotal - weekBill,
		WeekEstimatedBillable:   billable,
		EntriesWithoutRate:      noRate,
		RunningTimer:            running,
	}, nil
}

func (s *Service) totals(ctx context.Context, workspaceID, userID uuid.UUID, from, to time.Time) (total, billable int64, err error) {
	row := s.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(duration_seconds), 0),
			COALESCE(SUM(CASE WHEN is_billable THEN duration_seconds ELSE 0 END), 0)
		FROM time_entries
		WHERE workspace_id = $1 AND user_id = $2
		  AND started_at >= $3 AND started_at <= $4
	`, workspaceID, userID, from, to)
	err = row.Scan(&total, &billable)
	return
}

// estimateBillable sums billable amount per currency using the per-entry rate
// snapshot columns. Only closed (`ended_at IS NOT NULL`) billable entries are
// considered. Entries without a snapshot contribute zero and are counted in
// the `no_rate` aggregate. The rate-resolution service is never consulted.
func (s *Service) estimateBillable(ctx context.Context, workspaceID, userID uuid.UUID, from, to time.Time) (map[string]int64, int, error) {
	return s.estimateScoped(ctx, workspaceID, userID, from, to, "", uuid.Nil)
}

// estimateByClient / estimateByProject are scoped wrappers over estimateScoped.
func (s *Service) estimateByClient(ctx context.Context, workspaceID, userID, clientID uuid.UUID, rng Range) (map[string]int64, error) {
	out, _, err := s.estimateScoped(ctx, workspaceID, userID, dayStart(rng.From), dayEnd(rng.To), "client", clientID)
	return out, err
}

func (s *Service) estimateByProject(ctx context.Context, workspaceID, userID, projectID uuid.UUID, rng Range) (map[string]int64, error) {
	out, _, err := s.estimateScoped(ctx, workspaceID, userID, dayStart(rng.From), dayEnd(rng.To), "project", projectID)
	return out, err
}

// estimateScoped performs the aggregating query. `scope` ∈ {"", "client", "project"}.
// For closed entries whose snapshot is NULL (`hourly_rate_minor IS NULL`), the
// row is counted toward `noRate` (only used for the whole-range variant) and
// contributes zero to the per-currency total. The reporting read path does
// not consult `rates.Service.Resolve` for any closed entry.
func (s *Service) estimateScoped(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
	from, to time.Time,
	scope string,
	scopeID uuid.UUID,
) (map[string]int64, int, error) {
	out := map[string]int64{}

	// Aggregate billable amount by currency in SQL. Only closed entries.
	aggSQL := `
		SELECT te.currency_code,
		       COALESCE(SUM((te.duration_seconds * te.hourly_rate_minor) / 3600), 0)
		FROM time_entries te
		JOIN projects p ON p.id = te.project_id
		WHERE te.workspace_id = $1 AND te.user_id = $2
		  AND te.is_billable = true
		  AND te.ended_at IS NOT NULL
		  AND te.hourly_rate_minor IS NOT NULL
		  AND te.currency_code IS NOT NULL
		  AND te.started_at >= $3 AND te.started_at <= $4
	`
	args := []any{workspaceID, userID, from, to}
	switch scope {
	case "client":
		aggSQL += ` AND p.client_id = $5`
		args = append(args, scopeID)
	case "project":
		aggSQL += ` AND te.project_id = $5`
		args = append(args, scopeID)
	}
	aggSQL += ` GROUP BY te.currency_code`

	rows, err := s.pool.Query(ctx, aggSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	for rows.Next() {
		var ccy string
		var amt int64
		if err := rows.Scan(&ccy, &amt); err != nil {
			rows.Close()
			return nil, 0, err
		}
		out[ccy] += amt
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Count closed billable entries missing a snapshot (whole-range only).
	// Per spec: running entries (`ended_at IS NULL`) MUST NOT contribute and
	// MUST NOT be counted here.
	var noRate int
	if scope == "" {
		noRateSQL := `
			SELECT count(*)
			FROM time_entries te
			WHERE te.workspace_id = $1 AND te.user_id = $2
			  AND te.is_billable = true
			  AND te.ended_at IS NOT NULL
			  AND te.hourly_rate_minor IS NULL
			  AND te.started_at >= $3 AND te.started_at <= $4
		`
		if err := s.pool.QueryRow(ctx, noRateSQL, workspaceID, userID, from, to).Scan(&noRate); err != nil {
			return nil, 0, err
		}
	}

	return out, noRate, nil
}

// Report is the table view returned by Report.
type Report struct {
	Range       Range
	ByDay       []Bucket
	ByClient    []GroupedBucket
	ByProject   []GroupedBucket
	Totals      TotalsBlock
	NoRateCount int
	Grouping    string
}

// Range is the inclusive date window.
type Range struct {
	From time.Time
	To   time.Time
}

// Bucket is a single day/week row.
type Bucket struct {
	Label              string
	TotalSeconds       int64
	BillableSeconds    int64
	NonBillableSeconds int64
}

// GroupedBucket aggregates per client or project with amount.
type GroupedBucket struct {
	ID                  uuid.UUID
	Label               string
	ClientLabel         string
	Archived            bool
	TotalSeconds        int64
	BillableSeconds     int64
	NonBillableSeconds  int64
	EstimatedByCurrency map[string]int64
}

// TotalsBlock is the grand total across the range.
type TotalsBlock struct {
	TotalSeconds        int64
	BillableSeconds     int64
	NonBillableSeconds  int64
	EstimatedByCurrency map[string]int64
}

// Report produces the aggregates; `grouping` ∈ {"day","client","project"} (default day).
func (s *Service) Report(ctx context.Context, workspaceID, userID uuid.UUID, rng Range, grouping string) (Report, error) {
	if grouping == "" {
		grouping = "day"
	}
	out := Report{Range: rng, Grouping: grouping, Totals: TotalsBlock{EstimatedByCurrency: map[string]int64{}}}

	// Overall totals + estimated billable.
	totalsRow := s.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(duration_seconds),0),
			COALESCE(SUM(CASE WHEN is_billable THEN duration_seconds ELSE 0 END),0)
		FROM time_entries
		WHERE workspace_id = $1 AND user_id = $2
		  AND started_at::date >= $3 AND started_at::date <= $4
	`, workspaceID, userID, rng.From, rng.To)
	if err := totalsRow.Scan(&out.Totals.TotalSeconds, &out.Totals.BillableSeconds); err != nil {
		return out, err
	}
	out.Totals.NonBillableSeconds = out.Totals.TotalSeconds - out.Totals.BillableSeconds

	est, noRate, err := s.estimateBillable(ctx, workspaceID, userID, dayStart(rng.From), dayEnd(rng.To))
	if err != nil {
		return out, err
	}
	out.Totals.EstimatedByCurrency = est
	out.NoRateCount = noRate

	switch grouping {
	case "day":
		rows, err := s.pool.Query(ctx, `
			SELECT started_at::date AS d,
			       SUM(duration_seconds),
			       SUM(CASE WHEN is_billable THEN duration_seconds ELSE 0 END)
			FROM time_entries
			WHERE workspace_id = $1 AND user_id = $2
			  AND started_at::date >= $3 AND started_at::date <= $4
			GROUP BY d ORDER BY d ASC
		`, workspaceID, userID, rng.From, rng.To)
		if err != nil {
			return out, err
		}
		defer rows.Close()
		for rows.Next() {
			var d time.Time
			var tot, bill int64
			if err := rows.Scan(&d, &tot, &bill); err != nil {
				return out, err
			}
			out.ByDay = append(out.ByDay, Bucket{
				Label: d.Format("2006-01-02"), TotalSeconds: tot, BillableSeconds: bill, NonBillableSeconds: tot - bill,
			})
		}
		return out, rows.Err()
	case "client":
		buckets, err := s.groupByClient(ctx, workspaceID, userID, rng)
		if err != nil {
			return out, err
		}
		out.ByClient = buckets
		return out, nil
	case "project":
		buckets, err := s.groupByProject(ctx, workspaceID, userID, rng)
		if err != nil {
			return out, err
		}
		out.ByProject = buckets
		return out, nil
	default:
		return out, fmt.Errorf("unknown grouping %q", grouping)
	}
}

func (s *Service) groupByClient(ctx context.Context, workspaceID, userID uuid.UUID, rng Range) ([]GroupedBucket, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.name, c.is_archived,
		       SUM(te.duration_seconds),
		       SUM(CASE WHEN te.is_billable THEN te.duration_seconds ELSE 0 END)
		FROM time_entries te
		JOIN projects p ON p.id = te.project_id
		JOIN clients c ON c.id = p.client_id
		WHERE te.workspace_id = $1 AND te.user_id = $2
		  AND te.started_at::date >= $3 AND te.started_at::date <= $4
		GROUP BY c.id, c.name, c.is_archived
		ORDER BY c.name ASC
	`, workspaceID, userID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GroupedBucket
	for rows.Next() {
		var gb GroupedBucket
		var tot, bill int64
		if err := rows.Scan(&gb.ID, &gb.Label, &gb.Archived, &tot, &bill); err != nil {
			return nil, err
		}
		gb.TotalSeconds = tot
		gb.BillableSeconds = bill
		gb.NonBillableSeconds = tot - bill
		est, err := s.estimateByClient(ctx, workspaceID, userID, gb.ID, rng)
		if err != nil {
			return nil, err
		}
		gb.EstimatedByCurrency = est
		out = append(out, gb)
	}
	return out, rows.Err()
}

func (s *Service) groupByProject(ctx context.Context, workspaceID, userID uuid.UUID, rng Range) ([]GroupedBucket, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.name, c.name, p.is_archived,
		       SUM(te.duration_seconds),
		       SUM(CASE WHEN te.is_billable THEN te.duration_seconds ELSE 0 END)
		FROM time_entries te
		JOIN projects p ON p.id = te.project_id
		JOIN clients c ON c.id = p.client_id
		WHERE te.workspace_id = $1 AND te.user_id = $2
		  AND te.started_at::date >= $3 AND te.started_at::date <= $4
		GROUP BY p.id, p.name, c.name, p.is_archived
		ORDER BY c.name ASC, p.name ASC
	`, workspaceID, userID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GroupedBucket
	for rows.Next() {
		var gb GroupedBucket
		var tot, bill int64
		if err := rows.Scan(&gb.ID, &gb.Label, &gb.ClientLabel, &gb.Archived, &tot, &bill); err != nil {
			return nil, err
		}
		gb.TotalSeconds = tot
		gb.BillableSeconds = bill
		gb.NonBillableSeconds = tot - bill
		est, err := s.estimateByProject(ctx, workspaceID, userID, gb.ID, rng)
		if err != nil {
			return nil, err
		}
		gb.EstimatedByCurrency = est
		out = append(out, gb)
	}
	return out, rows.Err()
}

// ---- Date helpers ----

// startOfISOWeek returns the Monday 00:00 UTC of the week containing t.
func startOfISOWeek(t time.Time) time.Time {
	t = startOfDay(t)
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return t.AddDate(0, 0, -(wd - 1))
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func dayStart(t time.Time) time.Time {
	return startOfDay(t)
}

func dayEnd(t time.Time) time.Time {
	return startOfDay(t).Add(24*time.Hour - time.Second)
}

// PresetRange returns an inclusive date range for a preset name.
func PresetRange(now time.Time, name string) Range {
	now = now.UTC()
	switch name {
	case "today":
		d := startOfDay(now)
		return Range{From: d, To: d}
	case "last_week":
		ws := startOfISOWeek(now).AddDate(0, 0, -7)
		return Range{From: ws, To: ws.AddDate(0, 0, 6)}
	case "this_month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, -1)
		return Range{From: start, To: end}
	case "last_month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
		end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
		return Range{From: start, To: end}
	case "this_week":
		fallthrough
	default:
		ws := startOfISOWeek(now)
		return Range{From: ws, To: ws.AddDate(0, 0, 6)}
	}
}
