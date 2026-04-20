// Package rates implements the billing-rate domain: CRUD for `rate_rules`,
// overlap validation, and the authoritative rate resolver used by reporting.
package rates

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"timetrak/internal/shared/db"
)

// Level describes which precedence tier a rule sits at.
type Level string

// Level values.
const (
	LevelWorkspace Level = "workspace"
	LevelClient    Level = "client"
	LevelProject   Level = "project"
)

// Rule is a row in rate_rules (and its level derived from NULL-ness).
type Rule struct {
	ID                uuid.UUID
	WorkspaceID       uuid.UUID
	ClientID          uuid.UUID // uuid.Nil for workspace-default
	ClientName        string
	ProjectID         uuid.UUID // uuid.Nil when not per-project
	ProjectName       string
	CurrencyCode      string
	HourlyRateMinor   int64
	EffectiveFrom     time.Time // date-only, UTC midnight
	EffectiveTo       *time.Time
	Level             Level
	ReferencedByCount int // number of time_entries whose rate_rule_id = this rule.
}

// Resolution is the answer returned by Resolve.
type Resolution struct {
	Found           bool
	RuleID          uuid.UUID
	Level           Level
	HourlyRateMinor int64
	CurrencyCode    string
}

// Errors.
var (
	ErrNotFound        = errors.New("rates: not found")
	ErrInvalidWindow   = errors.New("rates: effective_to must be on or after effective_from")
	ErrNegativeRate    = errors.New("rates: hourly rate must be zero or positive")
	ErrInvalidCurrency = errors.New("rates: currency must be a 3-letter code")
	ErrOverlap         = errors.New("rates: window overlaps an existing rule at the same level")
	ErrClientNotInWS   = errors.New("rates: client does not belong to the workspace")
	ErrProjectNotInWS  = errors.New("rates: project does not belong to the workspace")
	// ErrRuleReferenced is returned by Update/Delete when at least one time_entries.rate_rule_id
	// points at the rule and the mutation would alter the historical view for those entries.
	// The safe updates (extend open-ended effective_to, shorten effective_to to a date on or
	// after the latest referencing entry's started_at::date) do not return this error.
	ErrRuleReferenced = errors.New("rates: rule is referenced by time entries")
)

// Service holds the rate use cases.
type Service struct{ pool *db.Pool }

// NewService constructs the service.
func NewService(pool *db.Pool) *Service { return &Service{pool: pool} }

// Input describes the payload for create/update.
type Input struct {
	ClientID        uuid.UUID // uuid.Nil = workspace-default or project-only
	ProjectID       uuid.UUID
	CurrencyCode    string
	HourlyRateMinor int64
	EffectiveFrom   time.Time // date-only
	EffectiveTo     *time.Time
}

// Get returns a single rule scoped to workspaceID, with its ReferencedByCount
// populated. Used by handlers that render row-level HTMX partials. Returns
// ErrNotFound when the rule is not in the workspace.
func (s *Service) Get(ctx context.Context, workspaceID, ruleID uuid.UUID) (Rule, error) {
	var r Rule
	var et *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT r.id, r.workspace_id,
		       COALESCE(r.client_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       COALESCE(c.name, ''),
		       COALESCE(r.project_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       COALESCE(p.name, ''),
		       r.currency_code, r.hourly_rate_minor, r.effective_from, r.effective_to,
		       (SELECT count(*) FROM time_entries te
		          WHERE te.workspace_id = r.workspace_id AND te.rate_rule_id = r.id)
		FROM rate_rules r
		LEFT JOIN clients  c ON c.id = r.client_id
		LEFT JOIN projects p ON p.id = r.project_id
		WHERE r.workspace_id = $1 AND r.id = $2
	`, workspaceID, ruleID).Scan(&r.ID, &r.WorkspaceID, &r.ClientID, &r.ClientName,
		&r.ProjectID, &r.ProjectName, &r.CurrencyCode, &r.HourlyRateMinor,
		&r.EffectiveFrom, &et, &r.ReferencedByCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Rule{}, ErrNotFound
	}
	if err != nil {
		return Rule{}, err
	}
	r.EffectiveTo = et
	r.Level = deriveLevel(r.ClientID, r.ProjectID)
	return r, nil
}

// List returns every rule in a workspace, grouped by level, with a referenced-count
// populated via correlated subquery (used by the UI to disable destructive controls).
func (s *Service) List(ctx context.Context, workspaceID uuid.UUID) ([]Rule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.workspace_id,
		       COALESCE(r.client_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       COALESCE(c.name, ''),
		       COALESCE(r.project_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       COALESCE(p.name, ''),
		       r.currency_code, r.hourly_rate_minor, r.effective_from, r.effective_to,
		       (SELECT count(*) FROM time_entries te
		          WHERE te.workspace_id = r.workspace_id AND te.rate_rule_id = r.id)
		FROM rate_rules r
		LEFT JOIN clients  c ON c.id = r.client_id
		LEFT JOIN projects p ON p.id = r.project_id
		WHERE r.workspace_id = $1
		ORDER BY r.effective_from DESC, r.created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Rule{}
	for rows.Next() {
		var r Rule
		var et *time.Time
		if err := rows.Scan(&r.ID, &r.WorkspaceID, &r.ClientID, &r.ClientName,
			&r.ProjectID, &r.ProjectName, &r.CurrencyCode, &r.HourlyRateMinor,
			&r.EffectiveFrom, &et, &r.ReferencedByCount); err != nil {
			return nil, err
		}
		r.EffectiveTo = et
		r.Level = deriveLevel(r.ClientID, r.ProjectID)
		out = append(out, r)
	}
	return out, rows.Err()
}

func deriveLevel(clientID, projectID uuid.UUID) Level {
	if projectID != uuid.Nil {
		return LevelProject
	}
	if clientID != uuid.Nil {
		return LevelClient
	}
	return LevelWorkspace
}

// Create inserts a rule after validation + overlap check.
func (s *Service) Create(ctx context.Context, workspaceID uuid.UUID, in Input) (uuid.UUID, error) {
	if err := s.validate(ctx, workspaceID, in); err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	err := s.pool.InTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := assertNoOverlap(ctx, tx, workspaceID, uuid.Nil, in); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `
			INSERT INTO rate_rules
			  (workspace_id, client_id, project_id, currency_code, hourly_rate_minor, effective_from, effective_to)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`, workspaceID, nullUUID(in.ClientID), nullUUID(in.ProjectID),
			strings.ToUpper(in.CurrencyCode), in.HourlyRateMinor,
			in.EffectiveFrom, in.EffectiveTo).Scan(&id)
	})
	return id, err
}

// Update edits an existing rule. If the rule is already referenced by at least one
// time entry, the update is rejected with ErrRuleReferenced UNLESS the only change is:
//   - extending `effective_to` from NULL to a future date (open → closed-future), or
//   - shortening `effective_to` to a date on or after the latest referencing entry's
//     `started_at::date`.
//
// Everything else (amount, currency, client, project, effective_from, shortening into
// the past) is rejected so historical snapshots remain stable.
func (s *Service) Update(ctx context.Context, workspaceID, ruleID uuid.UUID, in Input) error {
	if err := s.validate(ctx, workspaceID, in); err != nil {
		return err
	}
	return s.pool.InTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// Load the existing rule so we can compare the diff against the
		// immutability policy.
		var existing Input
		var existingClient, existingProject *uuid.UUID
		err := tx.QueryRow(ctx, `
			SELECT client_id, project_id, currency_code, hourly_rate_minor, effective_from, effective_to
			FROM rate_rules
			WHERE id = $1 AND workspace_id = $2
		`, ruleID, workspaceID).Scan(&existingClient, &existingProject, &existing.CurrencyCode,
			&existing.HourlyRateMinor, &existing.EffectiveFrom, &existing.EffectiveTo)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if existingClient != nil {
			existing.ClientID = *existingClient
		}
		if existingProject != nil {
			existing.ProjectID = *existingProject
		}

		count, latest, err := countReferencingEntries(ctx, tx, workspaceID, ruleID)
		if err != nil {
			return err
		}
		if count > 0 {
			if !isHistoricallySafeUpdate(existing, in, latest) {
				return ErrRuleReferenced
			}
		}

		if err := assertNoOverlap(ctx, tx, workspaceID, ruleID, in); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			UPDATE rate_rules
			SET client_id = $3, project_id = $4, currency_code = $5, hourly_rate_minor = $6,
			    effective_from = $7, effective_to = $8, updated_at = now()
			WHERE id = $1 AND workspace_id = $2
		`, ruleID, workspaceID, nullUUID(in.ClientID), nullUUID(in.ProjectID),
			strings.ToUpper(in.CurrencyCode), in.HourlyRateMinor,
			in.EffectiveFrom, in.EffectiveTo)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// Delete removes a rule scoped to workspace. Rejected with ErrRuleReferenced if any
// time_entries row still references the rule.
func (s *Service) Delete(ctx context.Context, workspaceID, ruleID uuid.UUID) error {
	return s.pool.InTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		count, _, err := countReferencingEntries(ctx, tx, workspaceID, ruleID)
		if err != nil {
			return err
		}
		if count > 0 {
			return ErrRuleReferenced
		}
		tag, err := tx.Exec(ctx, `DELETE FROM rate_rules WHERE id = $1 AND workspace_id = $2`, ruleID, workspaceID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// countReferencingEntries returns how many time_entries reference the given rule
// in the workspace, and the latest referencing started_at truncated to a UTC date
// (zero time.Time if count == 0). Uses the partial index
// ix_time_entries_workspace_rate_rule.
func countReferencingEntries(ctx context.Context, tx pgx.Tx, workspaceID, ruleID uuid.UUID) (int, time.Time, error) {
	var count int
	var latest *time.Time
	err := tx.QueryRow(ctx, `
		SELECT count(*), max(started_at)
		FROM time_entries
		WHERE workspace_id = $1 AND rate_rule_id = $2
	`, workspaceID, ruleID).Scan(&count, &latest)
	if err != nil {
		return 0, time.Time{}, err
	}
	if latest == nil {
		return count, time.Time{}, nil
	}
	// Truncate to UTC date — callers compare against effective_to which is a date.
	d := latest.UTC()
	return count, time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC), nil
}

// isHistoricallySafeUpdate reports whether `in` differs from `existing` only in ways
// that leave the view seen by already-referencing entries intact.
//
// The only safe diffs are:
//  1. `EffectiveTo` NULL → non-NULL future date, everything else unchanged.
//  2. `EffectiveTo` shortened to a date >= `latestRef` (inclusive), everything else
//     unchanged. The equality case preserves the referencing entries' active-at view
//     because effective_to is inclusive.
//
// "Unchanged" comparisons use strings.EqualFold for currency (both canonicalize later)
// and uuid.UUID equality for IDs.
func isHistoricallySafeUpdate(existing, in Input, latestRef time.Time) bool {
	sameClient := existing.ClientID == in.ClientID
	sameProject := existing.ProjectID == in.ProjectID
	sameCurrency := strings.EqualFold(strings.TrimSpace(existing.CurrencyCode), strings.TrimSpace(in.CurrencyCode))
	sameRate := existing.HourlyRateMinor == in.HourlyRateMinor
	sameFrom := existing.EffectiveFrom.Equal(in.EffectiveFrom)
	if !(sameClient && sameProject && sameCurrency && sameRate && sameFrom) {
		return false
	}

	// Compare effective_to.
	switch {
	case existing.EffectiveTo == nil && in.EffectiveTo == nil:
		// No change at all — technically a no-op update; safe.
		return true
	case existing.EffectiveTo == nil && in.EffectiveTo != nil:
		// Opening → closing: must be on or after the latest referencing date.
		return !in.EffectiveTo.Before(latestRef)
	case existing.EffectiveTo != nil && in.EffectiveTo == nil:
		// Re-opening (closed → open) is not safe: it extends the rule's reach,
		// but reporting snapshots already cover historical entries. To keep the
		// rule table tidy we still reject this to avoid surprising operators.
		return false
	default: // both non-nil
		if in.EffectiveTo.Equal(*existing.EffectiveTo) {
			return true
		}
		if in.EffectiveTo.After(*existing.EffectiveTo) {
			// Extending further in time is always safe for history.
			return true
		}
		// Shortening: must still cover the latest referencing entry's date.
		return !in.EffectiveTo.Before(latestRef)
	}
}

// Resolve returns the rate applicable to a given project at a given instant.
// Precedence: project → client → workspace-default → no-rate.
//
// Boundary semantics (MUST be preserved across refactors):
//   - `at` is converted to a UTC date via `at.UTC()` truncated to midnight — the
//     local-time billing boundary is a Stage 3 concern.
//   - `effective_from` is inclusive: a rule is active on and after this date.
//   - `effective_to` is inclusive: a rule is active up to and including this date.
//     NULL means open-ended.
//   - Two rules at the same precedence tier may not overlap, including sharing
//     a boundary date (`A.effective_to = B.effective_from` is rejected by
//     assertNoOverlap). With overlap strictly disjoint, the ORDER BY + LIMIT 1
//     tie-break below is defensive only — there is at most one active rule per
//     tier on a given date.
//
// Resolve MUST be invoked exactly once per entry at stop/save time; the tracking
// service snapshots the result on `time_entries`. Reporting reads the snapshot and
// MUST NOT call Resolve on the hot read path — live preview UIs that do call
// Resolve MUST label the result as a preview, not a historical figure.
func (s *Service) Resolve(ctx context.Context, workspaceID, projectID uuid.UUID, at time.Time) (Resolution, error) {
	date := at.UTC().Truncate(24 * time.Hour)

	// Resolve project's client (needed for client-level lookup).
	var clientID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT client_id FROM projects WHERE id = $1 AND workspace_id = $2
	`, projectID, workspaceID).Scan(&clientID)
	if errors.Is(err, pgx.ErrNoRows) {
		clientID = uuid.Nil // project may not exist; fall through with no client scope.
	} else if err != nil {
		return Resolution{}, err
	}

	type row struct {
		id              uuid.UUID
		hourlyRateMinor int64
		currency        string
	}

	query := func(sql string, args ...any) (row, bool, error) {
		var r row
		err := s.pool.QueryRow(ctx, sql, args...).Scan(&r.id, &r.hourlyRateMinor, &r.currency)
		if errors.Is(err, pgx.ErrNoRows) {
			return row{}, false, nil
		}
		if err != nil {
			return row{}, false, err
		}
		return r, true, nil
	}

	// 1. project
	if projectID != uuid.Nil {
		r, ok, err := query(`
			SELECT id, hourly_rate_minor, currency_code
			FROM rate_rules
			WHERE workspace_id = $1 AND project_id = $2
			  AND effective_from <= $3 AND (effective_to IS NULL OR $3 <= effective_to)
			ORDER BY effective_from DESC LIMIT 1
		`, workspaceID, projectID, date)
		if err != nil {
			return Resolution{}, err
		}
		if ok {
			return Resolution{Found: true, RuleID: r.id, Level: LevelProject, HourlyRateMinor: r.hourlyRateMinor, CurrencyCode: r.currency}, nil
		}
	}

	// 2. client
	if clientID != uuid.Nil {
		r, ok, err := query(`
			SELECT id, hourly_rate_minor, currency_code
			FROM rate_rules
			WHERE workspace_id = $1 AND client_id = $2 AND project_id IS NULL
			  AND effective_from <= $3 AND (effective_to IS NULL OR $3 <= effective_to)
			ORDER BY effective_from DESC LIMIT 1
		`, workspaceID, clientID, date)
		if err != nil {
			return Resolution{}, err
		}
		if ok {
			return Resolution{Found: true, RuleID: r.id, Level: LevelClient, HourlyRateMinor: r.hourlyRateMinor, CurrencyCode: r.currency}, nil
		}
	}

	// 3. workspace-default
	r, ok, err := query(`
		SELECT id, hourly_rate_minor, currency_code
		FROM rate_rules
		WHERE workspace_id = $1 AND client_id IS NULL AND project_id IS NULL
		  AND effective_from <= $2 AND (effective_to IS NULL OR $2 <= effective_to)
		ORDER BY effective_from DESC LIMIT 1
	`, workspaceID, date)
	if err != nil {
		return Resolution{}, err
	}
	if ok {
		return Resolution{Found: true, RuleID: r.id, Level: LevelWorkspace, HourlyRateMinor: r.hourlyRateMinor, CurrencyCode: r.currency}, nil
	}

	return Resolution{Found: false}, nil
}

func (s *Service) validate(ctx context.Context, workspaceID uuid.UUID, in Input) error {
	if in.HourlyRateMinor < 0 {
		return ErrNegativeRate
	}
	cur := strings.ToUpper(strings.TrimSpace(in.CurrencyCode))
	if len(cur) != 3 {
		return ErrInvalidCurrency
	}
	for _, r := range cur {
		if r < 'A' || r > 'Z' {
			return ErrInvalidCurrency
		}
	}
	if in.EffectiveTo != nil && in.EffectiveTo.Before(in.EffectiveFrom) {
		return ErrInvalidWindow
	}
	if in.ClientID != uuid.Nil {
		var wsID uuid.UUID
		err := s.pool.QueryRow(ctx, `SELECT workspace_id FROM clients WHERE id = $1`, in.ClientID).Scan(&wsID)
		if errors.Is(err, pgx.ErrNoRows) || wsID != workspaceID {
			return ErrClientNotInWS
		}
		if err != nil {
			return err
		}
	}
	if in.ProjectID != uuid.Nil {
		var wsID uuid.UUID
		err := s.pool.QueryRow(ctx, `SELECT workspace_id FROM projects WHERE id = $1`, in.ProjectID).Scan(&wsID)
		if errors.Is(err, pgx.ErrNoRows) || wsID != workspaceID {
			return ErrProjectNotInWS
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// assertNoOverlap rejects new/edited rules whose [from,to] overlaps any other
// rule at the same level in the same workspace (excluding the rule being edited).
//
// Overlap semantics (tightened in stabilize-rate-resolution-history): two windows
// overlap if they share any date, INCLUDING sharing the boundary date. Adjacency
// now requires a one-day gap (A.effective_to + 1 day = B.effective_from).
func assertNoOverlap(ctx context.Context, tx pgx.Tx, workspaceID, selfID uuid.UUID, in Input) error {
	// Build level-matching predicate and its args.
	//
	// Overlap predicate (inclusive, rejects shared boundaries):
	//     existing.effective_from <= COALESCE(new.effective_to,  'infinity'::date)
	//  AND (existing.effective_to IS NULL OR existing.effective_to >= new.effective_from)
	//
	// Two windows overlap iff they share at least one UTC date, INCLUDING sharing
	// the boundary date (`A.effective_to = B.effective_from`). Adjacency requires
	// a one-day gap (`A.effective_to + 1 day = B.effective_from`).
	args := []any{workspaceID, in.EffectiveFrom, in.EffectiveTo}

	var levelPredicate string
	switch deriveLevel(in.ClientID, in.ProjectID) {
	case LevelProject:
		levelPredicate = ` AND project_id = $4`
		args = append(args, in.ProjectID)
	case LevelClient:
		levelPredicate = ` AND project_id IS NULL AND client_id = $4`
		args = append(args, in.ClientID)
	default:
		levelPredicate = ` AND project_id IS NULL AND client_id IS NULL`
	}

	sql := `SELECT EXISTS(
			SELECT 1 FROM rate_rules
			WHERE workspace_id = $1
			  AND effective_from <= COALESCE($3::date, 'infinity'::date)
			  AND (effective_to IS NULL OR effective_to >= $2)
			  ` + levelPredicate
	if selfID != uuid.Nil {
		sql += ` AND id <> $` + itoa(len(args)+1)
		args = append(args, selfID)
	}
	sql += `)`
	var exists bool
	if err := tx.QueryRow(ctx, sql, args...).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return ErrOverlap
	}
	return nil
}

func itoa(n int) string {
	// small helper to avoid pulling strconv for one call.
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func nullUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
