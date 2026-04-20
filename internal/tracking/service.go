// Package tracking owns the time-entry lifecycle: start/stop a timer,
// manual entry create/edit/delete, list with filters + pagination.
package tracking

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"timetrak/internal/rates"
	"timetrak/internal/shared/clock"
	"timetrak/internal/shared/db"
)

// Entry represents a single row in time_entries plus joined labels.
type Entry struct {
	ID              uuid.UUID
	WorkspaceID     uuid.UUID
	UserID          uuid.UUID
	ProjectID       uuid.UUID
	ProjectName     string
	ClientID        uuid.UUID
	ClientName      string
	Description     string
	StartedAt       time.Time
	EndedAt         *time.Time
	DurationSeconds int64
	IsBillable      bool
}

// Errors. See errors.go for the integrity-error taxonomy
// (ErrInvalidInterval, ErrCrossWorkspaceProject, plus helpers).
var (
	ErrActiveTimerExists = errors.New("tracking: a timer is already running")
	ErrNoActiveTimer     = errors.New("tracking: no running timer")
	ErrProjectNotFound   = errors.New("tracking: project not found in workspace")
	ErrProjectArchived   = errors.New("tracking: project is archived")
	ErrEntryNotFound     = errors.New("tracking: entry not found")

	// ErrInvalidRange is retained as an alias of ErrInvalidInterval so
	// existing callers (and tests) that switch on it continue to work.
	// New code should use ErrInvalidInterval.
	ErrInvalidRange = ErrInvalidInterval
)

// Service encapsulates tracking use cases.
type Service struct {
	pool  *db.Pool
	clock clock.Clock
	rates *rates.Service // used to snapshot the resolved rate at entry close/save time
}

// NewService constructs the service. `rates` may be nil in tests that exercise
// only the timer integrity / pagination paths; production wiring (cmd/web)
// always passes a real *rates.Service so historical figures are stable.
func NewService(pool *db.Pool, clk clock.Clock, ratesSvc *rates.Service) *Service {
	if clk == nil {
		clk = clock.System{}
	}
	return &Service{pool: pool, clock: clk, rates: ratesSvc}
}

// StartInput describes payload for StartTimer.
type StartInput struct {
	ProjectID   uuid.UUID
	Description string
	IsBillable  *bool // if nil, inherit project.default_billable
}

// StartTimer inserts a running entry. On the partial-unique violation it returns ErrActiveTimerExists.
func (s *Service) StartTimer(ctx context.Context, workspaceID, userID uuid.UUID, in StartInput) (Entry, error) {
	var entry Entry
	err := s.pool.InTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// Verify project is in workspace and not archived.
		var isArchived, defaultBillable bool
		err := tx.QueryRow(ctx, `
			SELECT is_archived, default_billable FROM projects
			WHERE id = $1 AND workspace_id = $2
		`, in.ProjectID, workspaceID).Scan(&isArchived, &defaultBillable)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrProjectNotFound
		}
		if err != nil {
			return err
		}
		if isArchived {
			return ErrProjectArchived
		}
		billable := defaultBillable
		if in.IsBillable != nil {
			billable = *in.IsBillable
		}
		now := s.clock.Now().UTC()
		var id uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO time_entries
			  (workspace_id, user_id, project_id, description, started_at, ended_at, duration_seconds, is_billable)
			VALUES ($1, $2, $3, NULLIF($4, ''), $5, NULL, 0, $6)
			RETURNING id
		`, workspaceID, userID, in.ProjectID, strings.TrimSpace(in.Description), now, billable).Scan(&id)
		if err != nil {
			return translatePgError(err)
		}
		entry, err = getEntryTx(ctx, tx, workspaceID, id)
		return err
	})
	return entry, err
}

// StopTimer stops the user's running entry, or returns ErrNoActiveTimer.
//
// Concurrency contract (see openspec/changes/improve-timer-concurrency-...):
//   - Executes inside pool.InTx.
//   - SELECT ... FOR UPDATE acquires a row-level lock on the running entry so
//     two concurrent stops serialize deterministically.
//   - Uses the database server clock (now()) to set ended_at, not the handler
//     clock, so ended_at is monotonic on the DB host.
//   - The UPDATE is guarded by `ended_at IS NULL`, making stop idempotent: if
//     a prior stop already committed, the second caller acquires the lock,
//     sees ended_at IS NOT NULL, and returns the already-stopped row
//     unchanged (both callers observe identical ended_at).
func (s *Service) StopTimer(ctx context.Context, workspaceID, userID uuid.UUID) (Entry, error) {
	var entry Entry
	err := s.pool.InTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// Look for any entry (running or already-stopped) that belongs to
		// this user; FOR UPDATE serializes concurrent stops. If the user has
		// no running timer AND no recently-stopped row to return, we fall
		// through to ErrNoActiveTimer. In practice the partial unique index
		// caps active rows at one, so the WHERE below matches at most one
		// running row; an idempotent second caller will find ended_at set.
		// Look for the user's running entry OR an entry stopped within a
		// short idempotency window (5s). This gives a concurrent second stop
		// — which acquires the FOR UPDATE lock *after* the first committed —
		// a deterministic answer: the just-stopped row, with the winner's
		// ended_at intact. Outside the window, a stopped entry does not
		// qualify and we return ErrNoActiveTimer.
		var (
			id        uuid.UUID
			startedAt time.Time
			endedAt   *time.Time
		)
		err := tx.QueryRow(ctx, `
			SELECT id, started_at, ended_at FROM time_entries
			WHERE workspace_id = $1 AND user_id = $2
			  AND (ended_at IS NULL OR ended_at > now() - interval '5 seconds')
			ORDER BY started_at DESC
			LIMIT 1
			FOR UPDATE
		`, workspaceID, userID).Scan(&id, &startedAt, &endedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNoActiveTimer
		}
		if err != nil {
			return err
		}

		// Guarded UPDATE: only writes if the row is still running. On the
		// idempotent path (race-loser after the winner has committed, OR a
		// retry within the 5s window), the WHERE clause matches zero rows
		// and we return the existing row unchanged.
		tag, err := tx.Exec(ctx, `
			UPDATE time_entries
			SET ended_at = now(),
			    duration_seconds = EXTRACT(EPOCH FROM (now() - started_at))::int,
			    updated_at = now()
			WHERE id = $1 AND workspace_id = $2 AND ended_at IS NULL
		`, id, workspaceID)
		if err != nil {
			return translatePgError(err)
		}

		// Snapshot the resolved rate inside this same transaction — only when
		// THIS call closed the entry. On the idempotent path (zero rows
		// affected) the snapshot was already written by the winner.
		if tag.RowsAffected() > 0 {
			if err := s.writeRateSnapshotTx(ctx, tx, workspaceID, id); err != nil {
				return err
			}
		}

		entry, err = getEntryTx(ctx, tx, workspaceID, id)
		return err
	})
	return entry, err
}

// GetRunning returns the current user's running entry (if any) in a workspace.
func (s *Service) GetRunning(ctx context.Context, workspaceID, userID uuid.UUID) (*Entry, error) {
	rows, err := s.pool.Query(ctx, runningQuery, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	e, err := scanEntry(rows)
	if err != nil {
		return nil, err
	}
	return &e, rows.Err()
}

const runningQuery = `
	SELECT te.id, te.workspace_id, te.user_id, te.project_id, p.name, p.client_id, c.name,
	       COALESCE(te.description, ''), te.started_at, te.ended_at, te.duration_seconds, te.is_billable
	FROM time_entries te
	JOIN projects p ON p.id = te.project_id
	JOIN clients  c ON c.id = p.client_id
	WHERE te.workspace_id = $1 AND te.user_id = $2 AND te.ended_at IS NULL
	LIMIT 1
`

// ManualInput describes payload for CreateManual / Edit.
type ManualInput struct {
	ProjectID   uuid.UUID
	Description string
	StartedAt   time.Time
	EndedAt     time.Time
	IsBillable  bool
}

// CreateManual inserts a completed entry. EndedAt must be strictly greater
// than StartedAt; equal timestamps (zero duration) are rejected.
func (s *Service) CreateManual(ctx context.Context, workspaceID, userID uuid.UUID, in ManualInput) (Entry, error) {
	if !in.EndedAt.After(in.StartedAt) {
		return Entry{}, ErrInvalidInterval
	}
	var entry Entry
	err := s.pool.InTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var isArchived bool
		err := tx.QueryRow(ctx, `
			SELECT is_archived FROM projects WHERE id = $1 AND workspace_id = $2
		`, in.ProjectID, workspaceID).Scan(&isArchived)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrProjectNotFound
		}
		if err != nil {
			return err
		}
		if isArchived {
			return ErrProjectArchived
		}
		duration := int64(in.EndedAt.Sub(in.StartedAt).Seconds())
		var id uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO time_entries
			  (workspace_id, user_id, project_id, description, started_at, ended_at, duration_seconds, is_billable)
			VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8)
			RETURNING id
		`, workspaceID, userID, in.ProjectID, strings.TrimSpace(in.Description),
			in.StartedAt, in.EndedAt, duration, in.IsBillable).Scan(&id)
		if err != nil {
			if translated := translatePgError(err); translated != err {
				return translated
			}
			return fmt.Errorf("insert entry: %w", err)
		}
		// Manual entries are inserted closed; snapshot the historical rate at
		// started_at inside this transaction (no-rate is allowed and writes NULLs).
		if err := s.writeRateSnapshotTx(ctx, tx, workspaceID, id); err != nil {
			return err
		}
		entry, err = getEntryTx(ctx, tx, workspaceID, id)
		return err
	})
	return entry, err
}

// Edit updates an entry. Rejects edits that would create a second running timer.
// EndedAt must be strictly greater than StartedAt.
func (s *Service) Edit(ctx context.Context, workspaceID, userID, entryID uuid.UUID, in ManualInput) (Entry, error) {
	if !in.EndedAt.After(in.StartedAt) {
		return Entry{}, ErrInvalidInterval
	}
	var entry Entry
	err := s.pool.InTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// Load enough of the existing row to decide whether re-snapshotting is
		// required (per spec: re-snapshot only when project_id, started_at,
		// ended_at, duration_seconds, or is_billable changes).
		var (
			existingProject    uuid.UUID
			existingStarted    time.Time
			existingEnded      *time.Time
			existingDuration   int64
			existingIsBillable bool
		)
		err := tx.QueryRow(ctx, `
			SELECT project_id, started_at, ended_at, duration_seconds, is_billable
			FROM time_entries
			WHERE id = $1 AND workspace_id = $2 AND user_id = $3
		`, entryID, workspaceID, userID).Scan(&existingProject, &existingStarted, &existingEnded, &existingDuration, &existingIsBillable)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEntryNotFound
		}
		if err != nil {
			return err
		}
		duration := int64(in.EndedAt.Sub(in.StartedAt).Seconds())
		_, err = tx.Exec(ctx, `
			UPDATE time_entries SET
				project_id = $4, description = NULLIF($5, ''),
				started_at = $6, ended_at = $7, duration_seconds = $8, is_billable = $9,
				updated_at = now()
			WHERE id = $1 AND workspace_id = $2 AND user_id = $3
		`, entryID, workspaceID, userID, in.ProjectID, strings.TrimSpace(in.Description),
			in.StartedAt, in.EndedAt, duration, in.IsBillable)
		if err != nil {
			return translatePgError(err)
		}

		// Re-snapshot only if a rate-determining field changed.
		newEnded := in.EndedAt
		oldEnded := time.Time{}
		if existingEnded != nil {
			oldEnded = *existingEnded
		}
		rateInputChanged := existingProject != in.ProjectID ||
			!existingStarted.Equal(in.StartedAt) ||
			!oldEnded.Equal(newEnded) ||
			existingDuration != duration ||
			existingIsBillable != in.IsBillable
		if rateInputChanged {
			if err := s.writeRateSnapshotTx(ctx, tx, workspaceID, entryID); err != nil {
				return err
			}
		}

		entry, err = getEntryTx(ctx, tx, workspaceID, entryID)
		return err
	})
	return entry, err
}

// Delete removes an entry the user owns in the workspace.
func (s *Service) Delete(ctx context.Context, workspaceID, userID, entryID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM time_entries WHERE id = $1 AND workspace_id = $2 AND user_id = $3
	`, entryID, workspaceID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrEntryNotFound
	}
	return nil
}

// ListFilters is the entries list query.
type ListFilters struct {
	From      *time.Time
	To        *time.Time
	ClientID  uuid.UUID
	ProjectID uuid.UUID
	Billable  *bool
	UserID    uuid.UUID // zero = all users in workspace (MVP: filter to current user at handler level)
	Page      int
	PageSize  int
}

// ListResult packages paged entries.
type ListResult struct {
	Entries    []Entry
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

// List returns paginated entries matching the filters.
func (s *Service) List(ctx context.Context, workspaceID uuid.UUID, f ListFilters) (ListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 25
	}
	args := []any{workspaceID}
	where := "WHERE te.workspace_id = $1"
	if f.UserID != uuid.Nil {
		args = append(args, f.UserID)
		where += fmt.Sprintf(" AND te.user_id = $%d", len(args))
	}
	if f.From != nil {
		args = append(args, *f.From)
		where += fmt.Sprintf(" AND te.started_at >= $%d", len(args))
	}
	if f.To != nil {
		args = append(args, *f.To)
		where += fmt.Sprintf(" AND te.started_at <= $%d", len(args))
	}
	if f.ClientID != uuid.Nil {
		args = append(args, f.ClientID)
		where += fmt.Sprintf(" AND p.client_id = $%d", len(args))
	}
	if f.ProjectID != uuid.Nil {
		args = append(args, f.ProjectID)
		where += fmt.Sprintf(" AND te.project_id = $%d", len(args))
	}
	if f.Billable != nil {
		args = append(args, *f.Billable)
		where += fmt.Sprintf(" AND te.is_billable = $%d", len(args))
	}
	var total int
	// authz:ok: where-clause is built dynamically above and always begins
	// with `WHERE te.workspace_id = $1` (see top of List); the literal
	// here is a fragment concatenated with that scoped predicate.
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM time_entries te
		JOIN projects p ON p.id = te.project_id
		`+where, args...).Scan(&total); err != nil {
		return ListResult{}, err
	}
	offset := (f.Page - 1) * f.PageSize
	args = append(args, f.PageSize, offset)
	rows, err := s.pool.Query(ctx, `
		SELECT te.id, te.workspace_id, te.user_id, te.project_id, p.name, p.client_id, c.name,
		       COALESCE(te.description, ''), te.started_at, te.ended_at, te.duration_seconds, te.is_billable
		FROM time_entries te
		JOIN projects p ON p.id = te.project_id
		JOIN clients  c ON c.id = p.client_id
		`+where+fmt.Sprintf(` ORDER BY te.started_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)),
		args...)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return ListResult{}, err
		}
		out = append(out, e)
	}
	totalPages := (total + f.PageSize - 1) / f.PageSize
	if totalPages < 1 {
		totalPages = 1
	}
	return ListResult{Entries: out, Total: total, Page: f.Page, PageSize: f.PageSize, TotalPages: totalPages}, rows.Err()
}

// Get returns an entry by id within the workspace.
func (s *Service) Get(ctx context.Context, workspaceID, entryID uuid.UUID) (Entry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT te.id, te.workspace_id, te.user_id, te.project_id, p.name, p.client_id, c.name,
		       COALESCE(te.description, ''), te.started_at, te.ended_at, te.duration_seconds, te.is_billable
		FROM time_entries te
		JOIN projects p ON p.id = te.project_id
		JOIN clients  c ON c.id = p.client_id
		WHERE te.id = $1 AND te.workspace_id = $2
	`, entryID, workspaceID)
	if err != nil {
		return Entry{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Entry{}, ErrEntryNotFound
	}
	return scanEntry(rows)
}

func getEntryTx(ctx context.Context, tx pgx.Tx, workspaceID, entryID uuid.UUID) (Entry, error) {
	rows, err := tx.Query(ctx, `
		SELECT te.id, te.workspace_id, te.user_id, te.project_id, p.name, p.client_id, c.name,
		       COALESCE(te.description, ''), te.started_at, te.ended_at, te.duration_seconds, te.is_billable
		FROM time_entries te
		JOIN projects p ON p.id = te.project_id
		JOIN clients  c ON c.id = p.client_id
		WHERE te.id = $1 AND te.workspace_id = $2
	`, entryID, workspaceID)
	if err != nil {
		return Entry{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Entry{}, ErrEntryNotFound
	}
	return scanEntry(rows)
}

// writeRateSnapshotTx resolves the historical rate for the entry at its
// started_at and persists it on the row inside the same transaction. Called
// from every entry close/save path so reporting can read snapshots directly
// (see openspec/specs/rates and openspec/specs/reporting). On no-rate, all
// three columns are written as NULL — the atomic CHECK constraint enforces
// they travel together.
func (s *Service) writeRateSnapshotTx(ctx context.Context, tx pgx.Tx, workspaceID, entryID uuid.UUID) error {
	if s.rates == nil {
		return nil // rates wiring is optional in narrow tests; production wires it.
	}
	var projectID uuid.UUID
	var startedAt time.Time
	if err := tx.QueryRow(ctx, `
		SELECT project_id, started_at FROM time_entries
		WHERE id = $1 AND workspace_id = $2
	`, entryID, workspaceID).Scan(&projectID, &startedAt); err != nil {
		return fmt.Errorf("snapshot lookup: %w", err)
	}
	res, err := s.rates.Resolve(ctx, workspaceID, projectID, startedAt)
	if err != nil {
		return fmt.Errorf("snapshot resolve: %w", err)
	}
	if res.Found {
		_, err = tx.Exec(ctx, `
			UPDATE time_entries
			SET rate_rule_id = $3, hourly_rate_minor = $4, currency_code = $5, updated_at = now()
			WHERE id = $1 AND workspace_id = $2
		`, entryID, workspaceID, res.RuleID, res.HourlyRateMinor, res.CurrencyCode)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE time_entries
			SET rate_rule_id = NULL, hourly_rate_minor = NULL, currency_code = NULL, updated_at = now()
			WHERE id = $1 AND workspace_id = $2
		`, entryID, workspaceID)
	}
	return err
}

func scanEntry(rows pgx.Rows) (Entry, error) {
	var e Entry
	var ended *time.Time
	err := rows.Scan(&e.ID, &e.WorkspaceID, &e.UserID, &e.ProjectID, &e.ProjectName,
		&e.ClientID, &e.ClientName, &e.Description,
		&e.StartedAt, &ended, &e.DurationSeconds, &e.IsBillable)
	e.EndedAt = ended
	return e, err
}
