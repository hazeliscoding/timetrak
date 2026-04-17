package authz_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/session"
	"timetrak/internal/shared/testdb"
)

// stubNextHandler captures whether next.ServeHTTP was invoked and what
// WorkspaceContext (if any) was on the request.
type stubNext struct {
	called bool
	wc     authz.WorkspaceContext
	had    bool
}

func (s *stubNext) ServeHTTP(_ http.ResponseWriter, r *http.Request) {
	s.called = true
	s.wc, s.had = authz.FromContext(r.Context())
}

func TestRequireWorkspace_NoSession_RedirectsToLogin(t *testing.T) {
	pool := testdb.Open(t)
	svc := authz.NewService(pool.Pool)

	next := &stubNext{}
	mw := svc.RequireWorkspace(next)

	r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)

	if next.called {
		t.Fatalf("next handler MUST NOT be called when no session is present")
	}
	if got, want := w.Result().StatusCode, http.StatusSeeOther; got != want {
		t.Fatalf("status: got %d want %d", got, want)
	}
	if loc := w.Header().Get("Location"); loc != "/login" {
		t.Fatalf("Location: got %q want %q", loc, "/login")
	}
}

func TestRequireWorkspace_SessionWithoutActiveWorkspace_404(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	userID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, display_name)
		VALUES ($1, $2, 'x', 'X')
	`, userID, userID.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	svc := authz.NewService(pool.Pool)

	next := &stubNext{}
	mw := svc.RequireWorkspace(next)

	r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	r = r.WithContext(injectSession(r.Context(), session.Session{UserID: userID}))
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)

	if next.called {
		t.Fatalf("next handler MUST NOT be called when no active workspace is set")
	}
	if got, want := w.Result().StatusCode, http.StatusNotFound; got != want {
		t.Fatalf("status: got %d want %d", got, want)
	}
}

func TestRequireWorkspace_SessionWithUnverifiedWorkspace_404(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	userID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, display_name)
		VALUES ($1, $2, 'x', 'X')
	`, userID, userID.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	// Workspace exists but user is NOT a member.
	wsID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id, name, slug) VALUES ($1, 'W', $2)`,
		wsID, wsID.String()); err != nil {
		t.Fatal(err)
	}
	svc := authz.NewService(pool.Pool)

	next := &stubNext{}
	mw := svc.RequireWorkspace(next)
	r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	r = r.WithContext(injectSession(r.Context(), session.Session{UserID: userID, ActiveWorkspaceID: &wsID}))
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)

	if next.called {
		t.Fatalf("next handler MUST NOT be called when membership is missing")
	}
	if got, want := w.Result().StatusCode, http.StatusNotFound; got != want {
		t.Fatalf("status: got %d want %d", got, want)
	}
}

func TestRequireWorkspace_SessionWithMembership_PopulatesContext(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	userID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, display_name)
		VALUES ($1, $2, 'x', 'X')
	`, userID, userID.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	wsID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id, name, slug) VALUES ($1, 'W', $2)`,
		wsID, wsID.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'owner')`,
		wsID, userID); err != nil {
		t.Fatal(err)
	}
	svc := authz.NewService(pool.Pool)

	next := &stubNext{}
	mw := svc.RequireWorkspace(next)
	r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	r = r.WithContext(injectSession(r.Context(), session.Session{UserID: userID, ActiveWorkspaceID: &wsID}))
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)

	if !next.called {
		t.Fatalf("next handler MUST be called for an active member")
	}
	if !next.had {
		t.Fatalf("WorkspaceContext MUST be present on the request context")
	}
	if next.wc.UserID != userID {
		t.Fatalf("WorkspaceContext.UserID: got %v want %v", next.wc.UserID, userID)
	}
	if next.wc.WorkspaceID != wsID {
		t.Fatalf("WorkspaceContext.WorkspaceID: got %v want %v", next.wc.WorkspaceID, wsID)
	}
	if next.wc.Role != "owner" {
		t.Fatalf("WorkspaceContext.Role: got %q want %q", next.wc.Role, "owner")
	}
}

func TestRequireWorkspace_TamperedWorkspaceID_404(t *testing.T) {
	// A session row carrying an active_workspace_id that doesn't exist (or
	// belongs to a workspace the user never joined) MUST be rejected. This
	// guards against an attacker who tampered with session state directly.
	pool := testdb.Open(t)
	ctx := context.Background()
	userID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, display_name)
		VALUES ($1, $2, 'x', 'X')
	`, userID, userID.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	bogus := uuid.New() // never inserted into workspaces

	svc := authz.NewService(pool.Pool)
	next := &stubNext{}
	mw := svc.RequireWorkspace(next)
	r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	r = r.WithContext(injectSession(r.Context(), session.Session{UserID: userID, ActiveWorkspaceID: &bogus}))
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)

	if next.called {
		t.Fatalf("next handler MUST NOT be called for a tampered workspace id")
	}
	if got, want := w.Result().StatusCode, http.StatusNotFound; got != want {
		t.Fatalf("status: got %d want %d", got, want)
	}
}

// injectSession is a test-only helper that places a session.Session on ctx
// in the same way session.Loader would, via the WithSession helper the
// session package exposes for tests.
func injectSession(ctx context.Context, s session.Session) context.Context {
	return session.WithSession(ctx, s)
}
