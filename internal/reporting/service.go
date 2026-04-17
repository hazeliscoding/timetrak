// Package reporting aggregates time entries into totals and estimated billable
// amounts. Rate resolution is always delegated to `rates.Service.Resolve` so
// reports and (future) invoicing share a single source of truth.
package reporting

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/rates"
	"timetrak/internal/shared/db"
	"timetrak/internal/shared/money"
)

// Service exposes report queries.
type Service struct {
	pool  *db.Pool
	rates *rates.Service
}

// NewService constructs the reporting service.
func NewService(pool *db.Pool, rs *rates.Service) *Service {
	return &Service{pool: pool, rates: rs}
}

// DashboardSummary is the at-a-glance card set.
type DashboardSummary struct {
	TodayTotalSeconds        int64
	TodayBillableSeconds     int64
	TodayNonBillableSeconds  int64
	WeekTotalSeconds         int64
	WeekBillableSeconds      int64
	WeekNonBillableSeconds   int64
	WeekEstimatedBillable    map[string]int64 // currency → minor units
	EntriesWithoutRate       int
	RunningTimer             bool
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

// estimateBillable walks each billable entry in range and asks the rate service
// for the rate active at entry.started_at. Accumulates per-currency minor units
// and a no-rate counter.
func (s *Service) estimateBillable(ctx context.Context, workspaceID, userID uuid.UUID, from, to time.Time) (map[string]int64, int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT project_id, started_at, duration_seconds
		FROM time_entries
		WHERE workspace_id = $1 AND user_id = $2 AND is_billable = true
		  AND started_at >= $3 AND started_at <= $4
	`, workspaceID, userID, from, to)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := map[string]int64{}
	var noRate int
	for rows.Next() {
		var projectID uuid.UUID
		var started time.Time
		var seconds int64
		if err := rows.Scan(&projectID, &started, &seconds); err != nil {
			return nil, 0, err
		}
		res, err := s.rates.Resolve(ctx, workspaceID, projectID, started)
		if err != nil {
			return nil, 0, err
		}
		if !res.Found {
			noRate++
			continue
		}
		out[res.CurrencyCode] += money.DurationBillable(seconds, res.HourlyRateMinor)
	}
	return out, noRate, rows.Err()
}

// Report is the table view returned by Report.
type Report struct {
	Range        Range
	ByDay        []Bucket
	ByClient     []GroupedBucket
	ByProject    []GroupedBucket
	Totals       TotalsBlock
	NoRateCount  int
	Grouping     string
}

// Range is the inclusive date window.
type Range struct {
	From time.Time
	To   time.Time
}

// Bucket is a single day/week row.
type Bucket struct {
	Label             string
	TotalSeconds      int64
	BillableSeconds   int64
	NonBillableSeconds int64
}

// GroupedBucket aggregates per client or project with amount.
type GroupedBucket struct {
	ID                 uuid.UUID
	Label              string
	ClientLabel        string
	Archived           bool
	TotalSeconds       int64
	BillableSeconds    int64
	NonBillableSeconds int64
	EstimatedByCurrency map[string]int64
}

// TotalsBlock is the grand total across the range.
type TotalsBlock struct {
	TotalSeconds       int64
	BillableSeconds    int64
	NonBillableSeconds int64
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
		// Aggregate estimated billable via per-entry rate resolve.
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

func (s *Service) estimateByClient(ctx context.Context, workspaceID, userID, clientID uuid.UUID, rng Range) (map[string]int64, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT te.project_id, te.started_at, te.duration_seconds
		FROM time_entries te
		JOIN projects p ON p.id = te.project_id
		WHERE te.workspace_id = $1 AND te.user_id = $2 AND te.is_billable
		  AND p.client_id = $3
		  AND te.started_at::date >= $4 AND te.started_at::date <= $5
	`, workspaceID, userID, clientID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var pid uuid.UUID
		var started time.Time
		var sec int64
		if err := rows.Scan(&pid, &started, &sec); err != nil {
			return nil, err
		}
		res, err := s.rates.Resolve(ctx, workspaceID, pid, started)
		if err != nil {
			return nil, err
		}
		if res.Found {
			out[res.CurrencyCode] += money.DurationBillable(sec, res.HourlyRateMinor)
		}
	}
	return out, rows.Err()
}

func (s *Service) estimateByProject(ctx context.Context, workspaceID, userID, projectID uuid.UUID, rng Range) (map[string]int64, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT started_at, duration_seconds
		FROM time_entries
		WHERE workspace_id = $1 AND user_id = $2 AND is_billable
		  AND project_id = $3
		  AND started_at::date >= $4 AND started_at::date <= $5
	`, workspaceID, userID, projectID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var started time.Time
		var sec int64
		if err := rows.Scan(&started, &sec); err != nil {
			return nil, err
		}
		res, err := s.rates.Resolve(ctx, workspaceID, projectID, started)
		if err != nil {
			return nil, err
		}
		if res.Found {
			out[res.CurrencyCode] += money.DurationBillable(sec, res.HourlyRateMinor)
		}
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
