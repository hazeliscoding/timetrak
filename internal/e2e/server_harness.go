// Package e2e exposes a shared server bootstrap used by both the HTTP-only
// integration tests in this directory (happy_path_test.go) and the opt-in
// browser contract tests under ./browser/ (gated by //go:build browser).
//
// The helper builds a real HTTP handler stack backed by a real Postgres
// connection (via internal/shared/testdb) and returns it as an
// httptest.Server. It is intentionally identical to what cmd/web wires up,
// with the single difference that the session/csrf secret is a fixed
// test-only value.
package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"timetrak/internal/auth"
	"timetrak/internal/clients"
	"timetrak/internal/projects"
	"timetrak/internal/rates"
	"timetrak/internal/reporting"
	"timetrak/internal/settings"
	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/clock"
	"timetrak/internal/shared/csrf"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/logging"
	"timetrak/internal/shared/session"
	"timetrak/internal/shared/templates"
	"timetrak/internal/shared/testdb"
	"timetrak/internal/showcase"
	"timetrak/internal/tracking"
	"timetrak/internal/web/layout"
	"timetrak/internal/workspace"
)

// BuildServer wires the full TimeTrak HTTP stack against a freshly truncated
// database and returns it as an httptest.Server. The returned server is
// registered with t.Cleanup and will be closed at test teardown.
//
// This helper is shared with the browser contract tests under
// internal/e2e/browser/; any change here must stay behavior-identical to the
// non-browser e2e flow in happy_path_test.go.
func BuildServer(t *testing.T) *httptest.Server {
	t.Helper()
	pool := testdb.Open(t)

	// Session + CSRF need a secret. Fixed 32-byte test-only value.
	secret := []byte("0123456789abcdef0123456789abcdef")
	store, err := session.NewStore(pool.Pool, secret, false)
	if err != nil {
		t.Fatal(err)
	}

	tmplDir := FindTemplatesDir(t)
	root := os.DirFS(tmplDir)
	tpls, err := templates.Load(root)
	if err != nil {
		t.Fatalf("templates: %v", err)
	}

	azSvc := authz.NewService(pool.Pool)
	authSvc := auth.NewService(pool)
	limiter := auth.NewRateLimiter()
	wsSvc := workspace.NewService(pool, azSvc, store)
	clientsSvc := clients.NewService(pool)
	projectsSvc := projects.NewService(pool)
	ratesSvc := rates.NewService(pool)
	reportingSvc := reporting.NewService(pool)
	trackingSvc := tracking.NewService(pool, clock.System{}, ratesSvc)

	lay := layout.New(pool, wsSvc)

	// Timezones: snapshot once for the settings handler, mirroring the
	// production bootstrap in cmd/web/main.go. The Postgres list is
	// effectively static per version.
	tzList, err := wsSvc.ListTimezones(context.Background())
	if err != nil {
		t.Fatalf("list timezones: %v", err)
	}

	mux := http.NewServeMux()

	// Static assets — served from the repo's web/static/ so browser
	// contract tests render pages with the real compiled CSS and JS.
	// Mirrors cmd/web/main.go:161-162. See
	// openspec/specs/ui-browser-tests/spec.md (Harness MUST reuse the
	// existing server bootstrap).
	staticDir := filepath.Join(FindRepoRoot(t), "web", "static")
	staticFS := http.FileServer(http.Dir(filepath.Clean(staticDir)))
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticFS))

	auth.NewHandler(authSvc, store, tpls, limiter).Register(mux)
	workspace.NewHandler(wsSvc).Register(mux)
	protect := func(next http.Handler) http.Handler {
		return authz.RequireAuth(azSvc.RequireWorkspaceMember(next))
	}
	settings.NewHandler(wsSvc, tpls, lay, tzList).Register(mux, protect)
	clients.NewHandler(clientsSvc, tpls, lay).Register(mux, protect)
	projects.NewHandler(projectsSvc, clientsSvc, tpls, lay).Register(mux, protect)
	rates.NewHandler(ratesSvc, clientsSvc, projectsSvc, tpls, lay).Register(mux, protect)
	tracking.NewHandler(trackingSvc, projectsSvc, clientsSvc, reportingSvc, wsSvc, tpls, lay).Register(mux, protect)
	reporting.NewHandler(reportingSvc, clientsSvc, projectsSvc, wsSvc, tpls, lay).Register(mux, protect)

	// Showcase is mounted in the test server so browser contract tests
	// can exercise /dev/showcase. We pass "dev" explicitly — this
	// bootstrap mirrors cmd/web with the single difference that
	// configuration is hard-coded test-only values.
	showcase.NewHandler(tpls, lay, "dev").Register(mux)

	var handler http.Handler = mux
	handler = csrf.Middleware(secret, false)(handler)
	handler = store.Loader(handler)
	handler = sharedhttp.Logging(logging.New("dev"))(handler)
	handler = sharedhttp.RequestID(handler)
	handler = sharedhttp.Recover(handler)

	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return ts
}

// FindTemplatesDir walks up from the current working directory to locate
// web/templates (tests run with $CWD = the test's package dir).
func FindTemplatesDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(wd, "web", "templates")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	t.Fatalf("web/templates not found from test working dir")
	return ""
}

// FindRepoRoot walks up from the current working directory to locate the
// repo root (identified by the presence of go.mod). Used by browser tests
// that need to resolve static asset paths (axe.min.js, testdata fixtures).
func FindRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	t.Fatalf("go.mod not found from test working dir")
	return ""
}
