package tracking

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// Stable error-code strings for the tracking integrity taxonomy. Handlers
// expose these to HTMX partials; structured logs use them as the value of
// the `tracking.error_kind` field. Do NOT invent new codes here without a
// corresponding change proposal — the low cardinality is the point.
const (
	ErrCodeActiveTimer     = "tracking.active_timer"
	ErrCodeNoActiveTimer   = "tracking.no_active_timer"
	ErrCodeInvalidInterval = "tracking.invalid_interval"
	ErrCodeCrossWorkspace  = "tracking.cross_workspace"
)

// PostgreSQL constraint names the taxonomy mapper recognizes.
const (
	constraintActiveTimerUnique  = "ux_time_entries_one_active_per_user_workspace"
	constraintIntervalCheck      = "chk_time_entries_interval"
	constraintLegacyRangeCheck   = "ck_time_entries_range"
	constraintProjectWorkspaceFK = "time_entries_project_workspace_fk"
)

// Integrity-failure errors. Handlers pattern-match these via errors.Is.
var (
	// ErrInvalidInterval means a write violated
	// `CHECK (ended_at IS NULL OR ended_at > started_at)` — zero-duration or
	// inverted intervals are rejected at both the service and DB layers.
	ErrInvalidInterval = errors.New("tracking: ended_at must be strictly greater than started_at")

	// ErrCrossWorkspaceProject means a write violated the composite FK
	// `(project_id, workspace_id) REFERENCES projects(id, workspace_id)` —
	// the time entry's project does not belong to the same workspace.
	ErrCrossWorkspaceProject = errors.New("tracking: project does not belong to this workspace")
)

// ErrorCode returns the stable error-code string for a known taxonomy error,
// or "" for unknown errors (callers fall back to their default 500 copy).
func ErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrActiveTimerExists):
		return ErrCodeActiveTimer
	case errors.Is(err, ErrNoActiveTimer):
		return ErrCodeNoActiveTimer
	case errors.Is(err, ErrInvalidInterval):
		return ErrCodeInvalidInterval
	case errors.Is(err, ErrCrossWorkspaceProject):
		return ErrCodeCrossWorkspace
	default:
		return ""
	}
}

// translatePgError maps a *pgconn.PgError into the tracking taxonomy based on
// SQLSTATE and the offending constraint name. Unknown constraints fall through
// and are returned as the original error so callers can log them at `error`
// and respond 500.
func translatePgError(err error) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	switch pg.Code {
	case "23505": // unique_violation
		if pg.ConstraintName == constraintActiveTimerUnique {
			return ErrActiveTimerExists
		}
	case "23514": // check_violation
		if pg.ConstraintName == constraintIntervalCheck || pg.ConstraintName == constraintLegacyRangeCheck {
			return ErrInvalidInterval
		}
	case "23503": // foreign_key_violation
		if pg.ConstraintName == constraintProjectWorkspaceFK {
			return ErrCrossWorkspaceProject
		}
	}
	return err
}
