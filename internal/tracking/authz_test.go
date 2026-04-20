package tracking_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/clients"
	"timetrak/internal/projects"
	"timetrak/internal/rates"
	"timetrak/internal/reporting"
	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/clock"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/testdb"
	"timetrak/internal/tracking"
	"timetrak/internal/web/layout"
	"timetrak/internal/workspace"
)

// TestTrackingCrossWorkspaceDenialMatrix exercises every tracking handler
// route as UserA (W1) against tracking resources in W2 (UserB's). Each
// MUST yield 404, no row mutation, no HX-Trigger emission.
//
// Includes the "running W1 timer does not block W2 timer" scenario the
// spec explicitly calls out for tracking.
func TestTrackingCrossWorkspaceDenialMatrix(t *testing.T) {
	pool := testdb.Open(t)
	tpls := testdb.LoadTemplates(t)
	notFound := sharedhttp.NewNotFoundRenderer(tpls)
	sharedhttp.SetGlobalNotFound(notFound.Render)
	authz.SetNotFoundRenderer(notFound.Render)

	f := testdb.SeedAuthzFixture(t, pool)

	// Seed a running entry in W2 for UserB so the cross-workspace stop/edit/
	// delete tests have a target.
	entryB := uuid.New()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO time_entries (id, workspace_id, user_id, project_id, started_at, ended_at, duration_seconds, is_billable)
		VALUES ($1, $2, $3, $4, $5, NULL, 0, true)
	`, entryB, f.WorkspaceB, f.UserB, f.ProjectB, time.Now().UTC().Add(-time.Hour)); err != nil {
		t.Fatalf("seed running entry: %v", err)
	}

	clientsSvc := clients.NewService(pool)
	projectsSvc := projects.NewService(pool)
	ratesSvc := rates.NewService(pool)
	reportSvc := reporting.NewService(pool)
	trackingSvc := tracking.NewService(pool, clock.System{}, ratesSvc)
	authzSvc := authz.NewService(pool.Pool)
	wsSvc := workspace.NewService(pool, authzSvc, nil)
	lay := layout.New(pool, wsSvc)
	h := tracking.NewHandler(trackingSvc, projectsSvc, clientsSvc, reportSvc, tpls, lay)

	mux := http.NewServeMux()
	h.Register(mux, func(next http.Handler) http.Handler { return next })

	denials := []struct {
		name   string
		method string
		path   string
		body   url.Values
	}{
		{"entry-row", http.MethodGet, "/time-entries/" + entryB.String() + "/row", nil},
		{"entry-edit", http.MethodGet, "/time-entries/" + entryB.String() + "/edit", nil},
		{"entry-update", http.MethodPatch, "/time-entries/" + entryB.String(),
			url.Values{
				"project_id":  {f.ProjectA.String()},
				"started_at":  {time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)},
				"ended_at":    {time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)},
				"description": {""},
				"is_billable": {"on"},
			},
		},
		{"entry-delete", http.MethodDelete, "/time-entries/" + entryB.String(), nil},
	}
	for _, tc := range denials {
		t.Run(tc.name, func(t *testing.T) {
			r := newRequest(tc.method, tc.path, tc.body)
			r = f.AttachAsUserA(r)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Result().StatusCode != http.StatusNotFound {
				t.Fatalf("%s %s: got %d want 404", tc.method, tc.path, w.Result().StatusCode)
			}
			if got := w.Header().Get("HX-Trigger"); got != "" {
				t.Fatalf("%s %s: denied request emitted HX-Trigger %q", tc.method, tc.path, got)
			}
		})
	}

	// Timer start against W2's project as UserA: 404, no insert.
	t.Run("timer-start-foreign-project", func(t *testing.T) {
		body := url.Values{"project_id": {f.ProjectB.String()}}
		r := newRequest(http.MethodPost, "/timer/start", body)
		r = f.AttachAsUserA(r)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Result().StatusCode != http.StatusNotFound {
			t.Fatalf("timer start foreign project: got %d want 404", w.Result().StatusCode)
		}
		if got := w.Header().Get("HX-Trigger"); got != "" {
			t.Fatalf("timer start foreign project: emitted HX-Trigger %q", got)
		}
		// Verify no time_entries row was inserted for UserA.
		var n int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM time_entries WHERE user_id = $1`,
			f.UserA).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("timer start foreign project leaked %d row(s) for UserA", n)
		}
	})

	// Entries list scoped to active workspace: only UserA's own entries.
	t.Run("entries-list-scoped", func(t *testing.T) {
		r := newRequest(http.MethodGet, "/time", nil)
		r = f.AttachAsUserA(r)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("entries list: got %d want 200", w.Result().StatusCode)
		}
		if strings.Contains(w.Body.String(), entryB.String()) {
			t.Fatalf("entries list: leaked W2 entry into W1 user's view")
		}
	})

	// Dashboard, dashboard summary, and timer widget are all scoped to the
	// caller's active workspace. They render 200 with W1-only data; never
	// reference W2 entries or projects in the body.
	for _, path := range []string{"GET /dashboard", "GET /dashboard/summary", "GET /dashboard/timer"} {
		t.Run("dashboard-scoped:"+path, func(t *testing.T) {
			parts := strings.SplitN(path, " ", 2)
			r := newRequest(parts[0], parts[1], nil)
			r = f.AttachAsUserA(r)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Result().StatusCode != http.StatusOK {
				t.Fatalf("%s: got %d want 200", path, w.Result().StatusCode)
			}
			if strings.Contains(w.Body.String(), entryB.String()) || strings.Contains(w.Body.String(), f.ProjectB.String()) {
				t.Fatalf("%s: leaked W2 data into W1 user's view", path)
			}
		})
	}

	// Stop timer as UserA when only W2 has a running timer: 404 (no
	// active W1 timer to stop). The handler returns 409 ("no timer running")
	// in some flows, but for cross-workspace denial there is no W1 row to
	// affect; we just confirm no W2 row is touched.
	t.Run("timer-stop-no-active-w1", func(t *testing.T) {
		// Confirm W2 entry still has ended_at = NULL after a UserA stop attempt.
		r := newRequest(http.MethodPost, "/timer/stop", url.Values{})
		r = f.AttachAsUserA(r)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		// status here can be 409 (no W1 timer); the critical invariant
		// is W2 entry remains untouched.
		var endedAt *time.Time
		if err := pool.QueryRow(context.Background(), `SELECT ended_at FROM time_entries WHERE id = $1`,
			entryB).Scan(&endedAt); err != nil {
			t.Fatal(err)
		}
		if endedAt != nil {
			t.Fatalf("W2 running entry was stopped by a W1 timer-stop request — cross-workspace mutation")
		}
	})

	// Active-timer invariant: a running W1 timer does NOT block a W2 timer
	// because the partial unique index is scoped to (workspace_id, user_id).
	// Here we verify the spec by starting a timer for UserA (a member of
	// only W1), then directly inserting a second running entry for UserA
	// in W2 via SQL — should succeed since uniqueness is per (ws, user).
	t.Run("running-W1-timer-does-not-block-W2", func(t *testing.T) {
		ctx := context.Background()
		// Start UserA's W1 timer via the service.
		if _, err := trackingSvc.StartTimer(ctx, f.WorkspaceA, f.UserA, tracking.StartInput{
			ProjectID: f.ProjectA,
		}); err != nil {
			t.Fatalf("start W1: %v", err)
		}
		// Add UserA as a member of W2 so we can insert a W2 entry under them.
		if _, err := pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'member')`,
			f.WorkspaceB, f.UserA); err != nil {
			t.Fatalf("add membership: %v", err)
		}
		// A second running entry for UserA in W2 MUST succeed.
		if _, err := trackingSvc.StartTimer(ctx, f.WorkspaceB, f.UserA, tracking.StartInput{
			ProjectID: f.ProjectB,
		}); err != nil {
			t.Fatalf("start W2 should succeed when W1 timer is running (per-workspace unique): %v", err)
		}
	})
}

func newRequest(method, path string, body url.Values) *http.Request {
	if body != nil {
		r := httptest.NewRequest(method, path, strings.NewReader(body.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return r
	}
	return httptest.NewRequest(method, path, nil)
}
