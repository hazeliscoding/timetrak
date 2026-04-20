// Package workspace manages workspaces and membership-scoped access.
package workspace

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/db"
	"timetrak/internal/shared/session"
)

// Workspace is a materialized row.
type Workspace struct {
	ID                uuid.UUID
	Name              string
	Slug              string
	ReportingTimezone string
}

// Service provides workspace operations.
type Service struct {
	pool  *db.Pool
	authz *authz.Service
	store *session.Store
}

// NewService constructs the workspace service.
func NewService(pool *db.Pool, a *authz.Service, store *session.Store) *Service {
	return &Service{pool: pool, authz: a, store: store}
}

// ErrForbidden reports cross-workspace access attempts; handlers should 404.
var ErrForbidden = errors.New("workspace: not a member")

// ErrInvalidTimezone is returned when a caller tries to set
// reporting_timezone to a value that is not present in
// pg_timezone_names (i.e., Postgres cannot resolve it).
var ErrInvalidTimezone = errors.New("workspace: invalid reporting timezone")

// ListForUser returns every workspace the user is a member of (for the header switcher).
func (s *Service) ListForUser(ctx context.Context, userID uuid.UUID) ([]Workspace, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT w.id, w.name, w.slug, w.reporting_timezone
		FROM workspaces w
		JOIN workspace_members m ON m.workspace_id = w.id
		WHERE m.user_id = $1
		ORDER BY w.name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Workspace
	for rows.Next() {
		var w Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &w.ReportingTimezone); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// Get returns the workspace by id if the user is a member, or ErrForbidden otherwise.
func (s *Service) Get(ctx context.Context, userID, workspaceID uuid.UUID) (Workspace, error) {
	if err := s.authz.IsMember(ctx, userID, workspaceID); err != nil {
		return Workspace{}, ErrForbidden
	}
	var w Workspace
	// authz:ok: membership is verified above by s.authz.IsMember; querying
	// the workspaces table by id alone is safe because non-members cannot
	// reach this line.
	err := s.pool.QueryRow(ctx, `SELECT id, name, slug, reporting_timezone FROM workspaces WHERE id = $1`, workspaceID).
		Scan(&w.ID, &w.Name, &w.Slug, &w.ReportingTimezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return Workspace{}, ErrForbidden
	}
	return w, err
}

// UpdateReportingTimezone sets the workspace's reporting_timezone after
// verifying membership and validating the name against pg_timezone_names.
// Returns ErrForbidden when the user is not a member (handlers MUST 404),
// or ErrInvalidTimezone when the tz is not recognized by Postgres.
func (s *Service) UpdateReportingTimezone(ctx context.Context, userID, workspaceID uuid.UUID, tz string) error {
	if err := s.authz.IsMember(ctx, userID, workspaceID); err != nil {
		return ErrForbidden
	}
	tz = trimTimezone(tz)
	if tz == "" {
		return ErrInvalidTimezone
	}
	var exists bool
	// authz:ok: pg_timezone_names is a Postgres-internal catalog view, not a
	// tenant table — no workspace_id column exists. Membership is verified
	// via s.authz.IsMember above before any workspace data is touched.
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_timezone_names WHERE name = $1)`, tz,
	).Scan(&exists); err != nil {
		return fmt.Errorf("validate timezone: %w", err)
	}
	if !exists {
		return ErrInvalidTimezone
	}
	// authz:ok: membership verified above; tz validated against
	// pg_timezone_names; update is scoped by id.
	_, err := s.pool.Exec(ctx,
		`UPDATE workspaces SET reporting_timezone = $1, updated_at = now() WHERE id = $2`,
		tz, workspaceID,
	)
	return err
}

func trimTimezone(tz string) string {
	// Minimal trim; reject leading/trailing whitespace, but do not
	// lowercase (IANA names are case-sensitive, e.g. America/New_York).
	end := len(tz)
	start := 0
	for start < end && (tz[start] == ' ' || tz[start] == '\t' || tz[start] == '\n' || tz[start] == '\r') {
		start++
	}
	for end > start && (tz[end-1] == ' ' || tz[end-1] == '\t' || tz[end-1] == '\n' || tz[end-1] == '\r') {
		end--
	}
	return tz[start:end]
}

// ListTimezones returns all IANA timezone names known to Postgres, sorted.
// Used to populate the settings select and cache the list at startup.
func (s *Service) ListTimezones(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT name FROM pg_timezone_names ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// SwitchActive updates the session's active workspace after verifying membership.
// Returns ErrForbidden if the user is not a member.
func (s *Service) SwitchActive(ctx context.Context, sessionID, userID, workspaceID uuid.UUID) error {
	if err := s.authz.IsMember(ctx, userID, workspaceID); err != nil {
		return ErrForbidden
	}
	if err := s.store.SetActiveWorkspace(ctx, sessionID, workspaceID); err != nil {
		return fmt.Errorf("set active workspace: %w", err)
	}
	return nil
}

// CreatePersonalWorkspace is kept available for tests and admin flows.
// Signup uses the auth.Service transaction directly so all three inserts commit together.
func (s *Service) CreatePersonalWorkspace(ctx context.Context, userID uuid.UUID, displayName string) (uuid.UUID, error) {
	var wsID uuid.UUID
	err := s.pool.InTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		slug := fmt.Sprintf("ws-%s", uuid.New().String()[:8])
		if err := tx.QueryRow(ctx, `
			INSERT INTO workspaces (name, slug) VALUES ($1, $2) RETURNING id
		`, displayName+"'s workspace", slug).Scan(&wsID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'owner')
		`, wsID, userID)
		return err
	})
	return wsID, err
}
