package session

import (
	"context"
	"errors"
	"net/http"
)

// key type separate from Store so authz can keep its own.
type ctxKey int

const ctxKeySession ctxKey = 0

// Loader middleware attempts to load a session from the request cookie and
// stores it in the request context. Unauthenticated requests are allowed to
// proceed; handlers that require auth gate with authz.RequireAuth.
func (s *Store) Loader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := s.Load(r.Context(), r)
		if err != nil && !errors.Is(err, ErrNotFound) {
			http.Error(w, "session lookup failed", http.StatusInternalServerError)
			return
		}
		if err == nil {
			ctx := context.WithValue(r.Context(), ctxKeySession, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// FromContext returns the session loaded by Loader, if any.
//
// SCOPE: this accessor is reserved for auth flows (login/logout/signup),
// the workspace switcher, and the layout builder which needs to display
// the current user's workspaces. Domain handlers (clients, projects,
// tracking, rates, reporting) MUST NOT call FromContext for authorization
// purposes; they read the verified WorkspaceContext from
// authz.MustFromContext(ctx) instead. The forbid-list lint test in
// internal/shared/authz enforces that handlers do not read workspace_id
// from request input; this comment documents the analogous discipline for
// session reads.
func FromContext(ctx context.Context) (Session, bool) {
	v, ok := ctx.Value(ctxKeySession).(Session)
	return v, ok
}

// WithSession injects a session.Session into ctx in the same way Loader
// would, so tests (and other middleware) can simulate an authenticated
// request without round-tripping a cookie.
func WithSession(ctx context.Context, s Session) context.Context {
	return context.WithValue(ctx, ctxKeySession, s)
}
