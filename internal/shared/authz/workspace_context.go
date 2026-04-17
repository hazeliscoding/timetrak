// Package authz: WorkspaceContext is the typed principal handlers receive
// from RequireWorkspace middleware. It binds an authenticated user to a
// verified active workspace membership. Handlers MUST read workspace
// authorization context from this value, NEVER from request input.
package authz

import (
	"context"

	"github.com/google/uuid"
)

// wsCtxKey is the unexported context key under which WorkspaceContext is
// stashed. Using a private type prevents accidental overwrites or reads
// from outside this package.
type wsCtxKey struct{}

// WorkspaceContext carries the verified principal for a request that has
// passed RequireWorkspace. UserID is the authenticated user; WorkspaceID
// is the user's active workspace, confirmed against workspace_members.
// Role is included for forward compatibility with role-based features but
// MUST NOT be branched on by Stage 2 handlers.
type WorkspaceContext struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Role        string // "owner" | "admin" | "member"
}

// WithWorkspaceContext returns a child ctx carrying the given WorkspaceContext.
// Only RequireWorkspace middleware should call this in production code.
func WithWorkspaceContext(ctx context.Context, wc WorkspaceContext) context.Context {
	return context.WithValue(ctx, wsCtxKey{}, wc)
}

// FromContext returns the WorkspaceContext stashed by RequireWorkspace,
// reporting whether one was present. Handlers that may run outside the
// middleware (e.g. shared not-found rendering) should use this form.
func FromContext(ctx context.Context) (WorkspaceContext, bool) {
	wc, ok := ctx.Value(wsCtxKey{}).(WorkspaceContext)
	return wc, ok
}

// MustFromContext returns the WorkspaceContext stashed by RequireWorkspace
// and panics if none is present. This is safe to call from any handler that
// is mounted behind RequireWorkspace, because middleware guarantees the value
// is set before the handler runs. A panic here is a programmer error
// (missing middleware on a route), not a runtime user error.
func MustFromContext(ctx context.Context) WorkspaceContext {
	wc, ok := FromContext(ctx)
	if !ok {
		panic("authz: WorkspaceContext missing from request context — handler is not mounted behind RequireWorkspace")
	}
	return wc
}
