// Package authz centralizes access-control primitives used by every domain.
//
// The rule across TimeTrak: every repository method that touches domain data
// MUST take workspaceID as a parameter (not derive it from the row). Handlers
// resolve workspaceID from the session and pass it down. Cross-workspace
// access never leaks identity — it returns 404.
package authz

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"timetrak/internal/shared/session"
)

type ctxKey int

const ctxKeyWorkspace ctxKey = 0

// ErrNotMember is returned when a user is not a member of the requested workspace.
var ErrNotMember = errors.New("authz: not a workspace member")

// Service checks workspace membership.
type Service struct{ pool *pgxpool.Pool }

// NewService returns a membership-check service.
func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// IsMember returns nil if the user is a member of workspaceID, ErrNotMember otherwise.
func (s *Service) IsMember(ctx context.Context, userID, workspaceID uuid.UUID) error {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM workspace_members
			WHERE workspace_id = $1 AND user_id = $2
		)
	`, workspaceID, userID).Scan(&ok)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if !ok {
		return ErrNotMember
	}
	return nil
}

// RequireAuth redirects to /login if no authenticated session is attached.
// It must run after session.Loader.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := session.FromContext(r.Context()); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireWorkspaceMember verifies the session has an active workspace and the
// user is a member of it. Cross-workspace access returns 404.
//
// On success, the request context is augmented with two values:
//
//  1. The legacy active-workspace UUID (via WithActiveWorkspace) for any
//     handler still reading via authz.ActiveWorkspace during migration.
//  2. The typed WorkspaceContext (via WithWorkspaceContext) which is the
//     preferred accessor for Stage 2 handlers.
//
// Both are populated together so domain handlers can be migrated incrementally
// without breaking siblings.
func (s *Service) RequireWorkspaceMember(next http.Handler) http.Handler {
	return s.RequireWorkspace(next)
}

// RequireWorkspace is the canonical middleware Stage 2 handlers should sit
// behind. It resolves the active workspace from the session, verifies
// membership (including the role) against workspace_members, and on success
// populates the typed WorkspaceContext on the request. On any failure it
// short-circuits with HTTP 404 via the shared not-found renderer.
func (s *Service) RequireWorkspace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := session.FromContext(r.Context())
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if sess.ActiveWorkspaceID == nil {
			s.renderNotFound(w, r)
			return
		}
		role, err := s.memberRole(r.Context(), sess.UserID, *sess.ActiveWorkspaceID)
		if err != nil {
			s.renderNotFound(w, r)
			return
		}
		wc := WorkspaceContext{
			UserID:      sess.UserID,
			WorkspaceID: *sess.ActiveWorkspaceID,
			Role:        role,
		}
		ctx := WithActiveWorkspace(r.Context(), wc.WorkspaceID)
		ctx = WithWorkspaceContext(ctx, wc)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// memberRole returns the membership role for (user, workspace) or
// ErrNotMember if no row exists. Result is plain text matching the
// workspace_members.role check constraint.
func (s *Service) memberRole(ctx context.Context, userID, workspaceID uuid.UUID) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT role FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotMember
	}
	if err != nil {
		return "", err
	}
	return role, nil
}

// notFoundRenderer is a function the application wires in at startup that
// renders the shared not-found template. Defaulting to http.NotFound keeps
// authz importable in tests and the migration runner where templates are
// not loaded.
var notFoundRenderer = func(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

// SetNotFoundRenderer wires the shared not-found renderer used by middleware
// when access is denied. cmd/web calls this once at startup.
func SetNotFoundRenderer(fn func(http.ResponseWriter, *http.Request)) {
	if fn != nil {
		notFoundRenderer = fn
	}
}

// renderNotFound delegates to the wired renderer.
func (s *Service) renderNotFound(w http.ResponseWriter, r *http.Request) {
	notFoundRenderer(w, r)
}

// WithActiveWorkspace stashes the verified active workspace id on ctx.
func WithActiveWorkspace(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKeyWorkspace, id)
}

// ActiveWorkspace returns the workspace id stashed by RequireWorkspaceMember.
// Returns uuid.Nil if called outside a workspace-scoped handler.
func ActiveWorkspace(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(ctxKeyWorkspace).(uuid.UUID)
	return v
}
