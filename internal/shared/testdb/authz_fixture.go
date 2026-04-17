package testdb

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/db"
	"timetrak/internal/shared/session"
	"timetrak/internal/shared/templates"
)

// AuthzFixture is the canonical two-workspace, two-user setup used by the
// cross-workspace authz integration test matrix. Each domain test calls
// SeedAuthzFixture(t) at the top, then invokes its handler routes as
// UserA (member of W1) targeting resources that live in W2 (where Bob is
// the only member). Every handler MUST respond 404.
type AuthzFixture struct {
	UserA      uuid.UUID // member of W1 only
	UserB      uuid.UUID // member of W2 only
	WorkspaceA uuid.UUID
	WorkspaceB uuid.UUID
	ClientA    uuid.UUID // belongs to W1
	ClientB    uuid.UUID // belongs to W2
	ProjectA   uuid.UUID // belongs to W1
	ProjectB   uuid.UUID // belongs to W2
}

// SeedAuthzFixture inserts the canonical two-workspace shape and returns
// its ids. Callers can then use BuildRequest to issue requests as either
// user against either workspace's resources.
func SeedAuthzFixture(t *testing.T, pool *db.Pool) AuthzFixture {
	t.Helper()
	ctx := context.Background()
	f := AuthzFixture{
		UserA:      uuid.New(),
		UserB:      uuid.New(),
		WorkspaceA: uuid.New(),
		WorkspaceB: uuid.New(),
		ClientA:    uuid.New(),
		ClientB:    uuid.New(),
		ProjectA:   uuid.New(),
		ProjectB:   uuid.New(),
	}
	mustExec(t, pool, `INSERT INTO users (id, email, password_hash, display_name) VALUES
		($1, $3, 'x', 'Alice'),
		($2, $4, 'x', 'Bob')`,
		f.UserA, f.UserB,
		f.UserA.String()+"@example.test", f.UserB.String()+"@example.test")
	mustExec(t, pool, `INSERT INTO workspaces (id, name, slug) VALUES
		($1, 'W1', $3),
		($2, 'W2', $4)`,
		f.WorkspaceA, f.WorkspaceB,
		f.WorkspaceA.String(), f.WorkspaceB.String())
	mustExec(t, pool, `INSERT INTO workspace_members (workspace_id, user_id, role) VALUES
		($1, $3, 'owner'),
		($2, $4, 'owner')`,
		f.WorkspaceA, f.WorkspaceB, f.UserA, f.UserB)
	mustExec(t, pool, `INSERT INTO clients (id, workspace_id, name) VALUES
		($1, $3, 'Acme W1'),
		($2, $4, 'Acme W2')`,
		f.ClientA, f.ClientB, f.WorkspaceA, f.WorkspaceB)
	mustExec(t, pool, `INSERT INTO projects (id, workspace_id, client_id, name) VALUES
		($1, $3, $5, 'P1'),
		($2, $4, $6, 'P2')`,
		f.ProjectA, f.ProjectB,
		f.WorkspaceA, f.WorkspaceB,
		f.ClientA, f.ClientB)
	_ = ctx
	return f
}

// AsUserA returns a context populated as if RequireWorkspace had run for
// UserA's session pointing at WorkspaceA. Use this to invoke handlers
// directly in unit-style tests without a full HTTP stack.
func (f AuthzFixture) AsUserA(parent context.Context) context.Context {
	return f.as(parent, f.UserA, f.WorkspaceA, "owner")
}

// AsUserB returns a context populated as UserB pointing at WorkspaceB.
func (f AuthzFixture) AsUserB(parent context.Context) context.Context {
	return f.as(parent, f.UserB, f.WorkspaceB, "owner")
}

func (f AuthzFixture) as(parent context.Context, userID, wsID uuid.UUID, role string) context.Context {
	ctx := session.WithSession(parent, session.Session{UserID: userID, ActiveWorkspaceID: &wsID})
	ctx = authz.WithActiveWorkspace(ctx, wsID)
	ctx = authz.WithWorkspaceContext(ctx, authz.WorkspaceContext{
		UserID: userID, WorkspaceID: wsID, Role: role,
	})
	return ctx
}

// AttachAsUserA wraps r so its context behaves as if RequireWorkspace had
// already authenticated UserA in WorkspaceA. The handler-under-test sees
// the same context shape it would in production after middleware ran.
func (f AuthzFixture) AttachAsUserA(r *http.Request) *http.Request {
	return r.WithContext(f.AsUserA(r.Context()))
}

// AttachAsUserB attaches UserB's WorkspaceB context.
func (f AuthzFixture) AttachAsUserB(r *http.Request) *http.Request {
	return r.WithContext(f.AsUserB(r.Context()))
}

func mustExec(t *testing.T, pool *db.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("seed exec: %v", err)
	}
}

// LoadTemplates walks up from the test cwd to find web/templates and loads
// the registry. Useful for handler tests that need a real templates.Registry.
func LoadTemplates(t *testing.T) *templates.Registry {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		candidate := filepath.Join(dir, "web", "templates")
		if _, err := os.Stat(candidate); err == nil {
			reg, err := templates.Load(os.DirFS(candidate))
			if err != nil {
				t.Fatalf("templates.Load: %v", err)
			}
			return reg
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find web/templates walking up from %s", dir)
		}
		dir = parent
	}
}
