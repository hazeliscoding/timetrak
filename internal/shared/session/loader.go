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
func FromContext(ctx context.Context) (Session, bool) {
	v, ok := ctx.Value(ctxKeySession).(Session)
	return v, ok
}
