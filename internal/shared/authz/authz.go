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
func (s *Service) RequireWorkspaceMember(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := session.FromContext(r.Context())
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if sess.ActiveWorkspaceID == nil {
			http.NotFound(w, r)
			return
		}
		if err := s.IsMember(r.Context(), sess.UserID, *sess.ActiveWorkspaceID); err != nil {
			http.NotFound(w, r)
			return
		}
		ctx := WithActiveWorkspace(r.Context(), *sess.ActiveWorkspaceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
