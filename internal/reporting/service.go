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
//
// Timezone handling: every aggregation query buckets entries by
// `(te.started_at AT TIME ZONE $tz)::date` and compares the incoming
// `from`/`to` date range against the same expression, where `$tz` is the
// workspace's `reporting_timezone` (IANA name). The tz is joined from the
// `workspaces` row inside each SQL statement, so callers pass only the
// workspace id — no tz parameter threading. To keep the existing
// `(workspace_id, started_at)` index useful, each query also carries a
// redundant raw-`started_at` envelope of `[$from - 1 day, $to + 2 days)`
// that the planner can serve via the index.
package reporting

import (
	"context"
	"fmt"
	"strings"
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

// Dashboard computes today's and this-week's totals in the workspace's
// reporting timezone. When the workspace's tz is UTC (the default), behavior
// is unchanged relative to the pre-tz baseline.
func (s *Service) Dashboard(ctx context.Context, workspaceID, userID uuid.UUID, now time.Time) (DashboardSummary, error) {
	tz, err := s.workspaceTZ(ctx, workspaceID)
	if err != nil {
		return DashboardSummary{}, err
	}
	loc, err := loadLocation(tz)
	if err != nil {
		return DashboardSummary{}, err
	}
	nowLocal := now.In(loc)
	todayLocal := startOfDayIn(nowLocal, loc)
	weekStartLocal := startOfISOWeekIn(nowLocal, loc)
	weekEndLocal := weekStartLocal.AddDate(0, 0, 7).Add(-time.Second)

	// Today.
	dayReport, err := s.aggregate(ctx, reportQuery{
		workspaceID: workspaceID,
		userID:      userID,
		from:        todayLocal,
		to:          todayLocal,
		grouping:    "day",
	})
	if err != nil {
		return DashboardSummary{}, err
	}
	// Week.
	weekReport, err := s.aggregate(ctx, reportQuery{
		workspaceID: workspaceID,
		userID:      userID,
		from:        weekStartLocal,
		to:          weekEndLocal,
		grouping:    "day",
	})
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
		TodayTotalSeconds:       dayReport.Totals.TotalSeconds,
		TodayBillableSeconds:    dayReport.Totals.BillableSeconds,
		TodayNonBillableSeconds: dayReport.Totals.NonBillableSeconds,
		WeekTotalSeconds:        weekReport.Totals.TotalSeconds,
		WeekBillableSeconds:     weekReport.Totals.BillableSeconds,
		WeekNonBillableSeconds:  weekReport.Totals.NonBillableSeconds,
		WeekEstimatedBillable:   weekReport.Totals.EstimatedByCurrency,
		EntriesWithoutRate:      weekReport.NoRateCount,
		RunningTimer:            running,
	}, nil
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

// Filters narrow every aggregate at the SQL layer. Empty / zero values are
// ignored. `Billable` is tri-state: "" (all), "yes", "no".
type Filters struct {
	ClientID  uuid.UUID
	ProjectID uuid.UUID
	Billable  string
}

// Report produces the aggregates; `grouping` ∈ {"day","client","project"} (default day).
//
// Deprecated: prefer ReportWithFilters so callers can narrow at the SQL layer.
// This wrapper is kept for tests and call sites that do not pass filters.
func (s *Service) Report(ctx context.Context, workspaceID, userID uuid.UUID, rng Range, grouping string) (Report, error) {
	return s.ReportWithFilters(ctx, workspaceID, userID, rng, grouping, Filters{})
}

// ReportWithFilters is the primary read path. `grouping` picks the detail
// table (day/client/project); `filters` narrows at the SQL layer.
func (s *Service) ReportWithFilters(ctx context.Context, workspaceID, userID uuid.UUID, rng Range, grouping string, filters Filters) (Report, error) {
	if grouping == "" {
		grouping = "day"
	}
	return s.aggregate(ctx, reportQuery{
		workspaceID: workspaceID,
		userID:      userID,
		from:        rng.From,
		to:          rng.To,
		grouping:    grouping,
		clientID:    filters.ClientID,
		projectID:   filters.ProjectID,
		billable:    filters.Billable,
	})
}

// reportQuery is the single struct the SQL builder operates on. All filter
// surfaces land here.
type reportQuery struct {
	workspaceID uuid.UUID
	userID      uuid.UUID
	from        time.Time // interpreted as a local date in workspace tz
	to          time.Time
	grouping    string // "day" | "client" | "project"
	clientID    uuid.UUID
	projectID   uuid.UUID
	billable    string // "" | "yes" | "no"
}

// whereSkeleton builds the shared WHERE fragment applied to every
// aggregation query for a given reportQuery. The first four bind positions
// are always (workspace_id, user_id, from, to). It returns the SQL
// fragment, the trailing arguments, and the next bind index.
func (q reportQuery) whereSkeleton() (where string, args []any, next int) {
	// Dual-bound: raw started_at envelope (so the planner uses the
	// (workspace_id, started_at) index) + tz-converted date predicate.
	var b strings.Builder
	b.WriteString(`te.workspace_id = $1 AND te.user_id = $2 ` +
		`AND te.started_at >= ($3::date - interval '1 day') ` +
		`AND te.started_at <  ($4::date + interval '2 days') ` +
		`AND (te.started_at AT TIME ZONE w.reporting_timezone)::date BETWEEN $3 AND $4`)
	args = []any{q.workspaceID, q.userID, q.from, q.to}
	idx := 4
	if q.clientID != uuid.Nil {
		idx++
		b.WriteString(fmt.Sprintf(" AND p.client_id = $%d", idx))
		args = append(args, q.clientID)
	}
	if q.projectID != uuid.Nil {
		idx++
		b.WriteString(fmt.Sprintf(" AND te.project_id = $%d", idx))
		args = append(args, q.projectID)
	}
	switch q.billable {
	case "yes":
		b.WriteString(" AND te.is_billable = true")
	case "no":
		b.WriteString(" AND te.is_billable = false")
	}
	return b.String(), args, idx
}

// aggregate runs the totals + per-currency + no-rate + grouping queries for
// a reportQuery. Every query joins `workspaces w` for the tz and `projects p`
// (the latter is needed for the client-id filter even when grouping by day).
func (s *Service) aggregate(ctx context.Context, q reportQuery) (Report, error) {
	if q.grouping == "" {
		q.grouping = "day"
	}
	out := Report{
		Range:    Range{From: q.from, To: q.to},
		Grouping: q.grouping,
		Totals:   TotalsBlock{EstimatedByCurrency: map[string]int64{}},
	}
	where, args, _ := q.whereSkeleton()

	// 1. Totals.
	totalsSQL := `
		SELECT
			COALESCE(SUM(te.duration_seconds), 0),
			COALESCE(SUM(CASE WHEN te.is_billable THEN te.duration_seconds ELSE 0 END), 0)
		FROM time_entries te
		JOIN workspaces w ON w.id = te.workspace_id
		JOIN projects   p ON p.id = te.project_id
		WHERE ` + where
	if err := s.pool.QueryRow(ctx, totalsSQL, args...).Scan(
		&out.Totals.TotalSeconds, &out.Totals.BillableSeconds,
	); err != nil {
		return out, err
	}
	out.Totals.NonBillableSeconds = out.Totals.TotalSeconds - out.Totals.BillableSeconds

	// 2. Estimated billable per currency (closed entries with snapshot).
	//    Short-circuit when `billable=no` — by construction, no rows qualify.
	if q.billable != "no" {
		estSQL := `
			SELECT te.currency_code,
			       COALESCE(SUM((te.duration_seconds * te.hourly_rate_minor) / 3600), 0)
			FROM time_entries te
			JOIN workspaces w ON w.id = te.workspace_id
			JOIN projects   p ON p.id = te.project_id
			WHERE ` + where + `
			  AND te.is_billable = true
			  AND te.ended_at IS NOT NULL
			  AND te.hourly_rate_minor IS NOT NULL
			  AND te.currency_code IS NOT NULL
			GROUP BY te.currency_code`
		rows, err := s.pool.Query(ctx, estSQL, args...)
		if err != nil {
			return out, err
		}
		for rows.Next() {
			var ccy string
			var amt int64
			if err := rows.Scan(&ccy, &amt); err != nil {
				rows.Close()
				return out, err
			}
			out.Totals.EstimatedByCurrency[ccy] = amt
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return out, err
		}

		// 3. NoRateCount: closed billable entries with NULL snapshot that
		//    also match the current filter set. Running entries never
		//    count. When `billable=no`, this is zero by construction.
		noRateSQL := `
			SELECT count(*)
			FROM time_entries te
			JOIN workspaces w ON w.id = te.workspace_id
			JOIN projects   p ON p.id = te.project_id
			WHERE ` + where + `
			  AND te.is_billable = true
			  AND te.ended_at IS NOT NULL
			  AND te.hourly_rate_minor IS NULL`
		if err := s.pool.QueryRow(ctx, noRateSQL, args...).Scan(&out.NoRateCount); err != nil {
			return out, err
		}
	}

	// 4. Detail grouping.
	switch q.grouping {
	case "day":
		daySQL := `
			SELECT (te.started_at AT TIME ZONE w.reporting_timezone)::date AS d,
			       SUM(te.duration_seconds),
			       SUM(CASE WHEN te.is_billable THEN te.duration_seconds ELSE 0 END)
			FROM time_entries te
			JOIN workspaces w ON w.id = te.workspace_id
			JOIN projects   p ON p.id = te.project_id
			WHERE ` + where + `
			GROUP BY d
			ORDER BY d ASC`
		rows, err := s.pool.Query(ctx, daySQL, args...)
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
				Label: d.Format("2006-01-02"), TotalSeconds: tot,
				BillableSeconds: bill, NonBillableSeconds: tot - bill,
			})
		}
		return out, rows.Err()
	case "client":
		buckets, err := s.groupByClient(ctx, q)
		if err != nil {
			return out, err
		}
		out.ByClient = buckets
		return out, nil
	case "project":
		buckets, err := s.groupByProject(ctx, q)
		if err != nil {
			return out, err
		}
		out.ByProject = buckets
		return out, nil
	default:
		return out, fmt.Errorf("unknown grouping %q", q.grouping)
	}
}

// groupByClient applies the same filter skeleton and scores each client. Per
// the spec, estimated billable per row is scored by currency for the same
// filter set (billable="no" yields an empty map).
func (s *Service) groupByClient(ctx context.Context, q reportQuery) ([]GroupedBucket, error) {
	where, args, _ := q.whereSkeleton()

	totalsSQL := `
		SELECT c.id, c.name, c.is_archived,
		       SUM(te.duration_seconds),
		       SUM(CASE WHEN te.is_billable THEN te.duration_seconds ELSE 0 END)
		FROM time_entries te
		JOIN workspaces w ON w.id = te.workspace_id
		JOIN projects   p ON p.id = te.project_id
		JOIN clients    c ON c.id = p.client_id
		WHERE ` + where + `
		GROUP BY c.id, c.name, c.is_archived
		ORDER BY c.name ASC`
	rows, err := s.pool.Query(ctx, totalsSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GroupedBucket
	seen := map[uuid.UUID]int{} // client id -> index in out
	for rows.Next() {
		var gb GroupedBucket
		var tot, bill int64
		if err := rows.Scan(&gb.ID, &gb.Label, &gb.Archived, &tot, &bill); err != nil {
			return nil, err
		}
		gb.TotalSeconds = tot
		gb.BillableSeconds = bill
		gb.NonBillableSeconds = tot - bill
		gb.EstimatedByCurrency = map[string]int64{}
		seen[gb.ID] = len(out)
		out = append(out, gb)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if q.billable == "no" || len(out) == 0 {
		return out, nil
	}

	// Per-client per-currency estimate, one query.
	estSQL := `
		SELECT c.id, te.currency_code,
		       COALESCE(SUM((te.duration_seconds * te.hourly_rate_minor) / 3600), 0)
		FROM time_entries te
		JOIN workspaces w ON w.id = te.workspace_id
		JOIN projects   p ON p.id = te.project_id
		JOIN clients    c ON c.id = p.client_id
		WHERE ` + where + `
		  AND te.is_billable = true
		  AND te.ended_at IS NOT NULL
		  AND te.hourly_rate_minor IS NOT NULL
		  AND te.currency_code IS NOT NULL
		GROUP BY c.id, te.currency_code`
	estRows, err := s.pool.Query(ctx, estSQL, args...)
	if err != nil {
		return nil, err
	}
	defer estRows.Close()
	for estRows.Next() {
		var cid uuid.UUID
		var ccy string
		var amt int64
		if err := estRows.Scan(&cid, &ccy, &amt); err != nil {
			return nil, err
		}
		if idx, ok := seen[cid]; ok {
			out[idx].EstimatedByCurrency[ccy] = amt
		}
	}
	return out, estRows.Err()
}

func (s *Service) groupByProject(ctx context.Context, q reportQuery) ([]GroupedBucket, error) {
	where, args, _ := q.whereSkeleton()

	totalsSQL := `
		SELECT p.id, p.name, c.name, p.is_archived,
		       SUM(te.duration_seconds),
		       SUM(CASE WHEN te.is_billable THEN te.duration_seconds ELSE 0 END)
		FROM time_entries te
		JOIN workspaces w ON w.id = te.workspace_id
		JOIN projects   p ON p.id = te.project_id
		JOIN clients    c ON c.id = p.client_id
		WHERE ` + where + `
		GROUP BY p.id, p.name, c.name, p.is_archived
		ORDER BY c.name ASC, p.name ASC`
	rows, err := s.pool.Query(ctx, totalsSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GroupedBucket
	seen := map[uuid.UUID]int{}
	for rows.Next() {
		var gb GroupedBucket
		var tot, bill int64
		if err := rows.Scan(&gb.ID, &gb.Label, &gb.ClientLabel, &gb.Archived, &tot, &bill); err != nil {
			return nil, err
		}
		gb.TotalSeconds = tot
		gb.BillableSeconds = bill
		gb.NonBillableSeconds = tot - bill
		gb.EstimatedByCurrency = map[string]int64{}
		seen[gb.ID] = len(out)
		out = append(out, gb)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if q.billable == "no" || len(out) == 0 {
		return out, nil
	}

	estSQL := `
		SELECT p.id, te.currency_code,
		       COALESCE(SUM((te.duration_seconds * te.hourly_rate_minor) / 3600), 0)
		FROM time_entries te
		JOIN workspaces w ON w.id = te.workspace_id
		JOIN projects   p ON p.id = te.project_id
		WHERE ` + where + `
		  AND te.is_billable = true
		  AND te.ended_at IS NOT NULL
		  AND te.hourly_rate_minor IS NOT NULL
		  AND te.currency_code IS NOT NULL
		GROUP BY p.id, te.currency_code`
	estRows, err := s.pool.Query(ctx, estSQL, args...)
	if err != nil {
		return nil, err
	}
	defer estRows.Close()
	for estRows.Next() {
		var pid uuid.UUID
		var ccy string
		var amt int64
		if err := estRows.Scan(&pid, &ccy, &amt); err != nil {
			return nil, err
		}
		if idx, ok := seen[pid]; ok {
			out[idx].EstimatedByCurrency[ccy] = amt
		}
	}
	return out, estRows.Err()
}

// workspaceTZ fetches the reporting_timezone for a workspace. Used by
// Dashboard and by PresetRange helpers on the handler side.
func (s *Service) workspaceTZ(ctx context.Context, workspaceID uuid.UUID) (string, error) {
	var tz string
	// authz:ok: querying the workspaces table by primary key alone is safe;
	// the caller has already passed through RequireWorkspace and holds
	// the workspace id, and this method reads a single public attribute.
	err := s.pool.QueryRow(ctx, `SELECT reporting_timezone FROM workspaces WHERE id = $1`, workspaceID).Scan(&tz)
	if err != nil {
		return "", err
	}
	if tz == "" {
		tz = "UTC"
	}
	return tz, nil
}

// ---- Date / tz helpers ----

// loadLocation resolves an IANA name to *time.Location; UTC fallback.
func loadLocation(tz string) (*time.Location, error) {
	if tz == "" || tz == "UTC" {
		return time.UTC, nil
	}
	return time.LoadLocation(tz)
}

func startOfDayIn(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// startOfISOWeekIn returns the Monday 00:00 of the ISO week containing t,
// in the given location.
func startOfISOWeekIn(t time.Time, loc *time.Location) time.Time {
	d := startOfDayIn(t, loc)
	wd := int(d.Weekday())
	if wd == 0 {
		wd = 7
	}
	return d.AddDate(0, 0, -(wd - 1))
}

// PresetRange returns an inclusive date range for a preset name. The range is
// computed in the given `loc` so "this week" in America/New_York starts on
// the local Monday, not the UTC one.
func PresetRange(now time.Time, name string, loc *time.Location) Range {
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	switch name {
	case "today":
		d := startOfDayIn(local, loc)
		return Range{From: d, To: d}
	case "last_week":
		ws := startOfISOWeekIn(local, loc).AddDate(0, 0, -7)
		return Range{From: ws, To: ws.AddDate(0, 0, 6)}
	case "this_month":
		start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
		end := start.AddDate(0, 1, -1)
		return Range{From: start, To: end}
	case "last_month":
		thisStart := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
		start := thisStart.AddDate(0, -1, 0)
		end := thisStart.AddDate(0, 0, -1)
		return Range{From: start, To: end}
	case "this_week":
		fallthrough
	default:
		ws := startOfISOWeekIn(local, loc)
		return Range{From: ws, To: ws.AddDate(0, 0, 6)}
	}
}
