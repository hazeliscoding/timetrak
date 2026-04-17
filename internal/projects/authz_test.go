package projects_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"timetrak/internal/clients"
	"timetrak/internal/projects"
	"timetrak/internal/shared/authz"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/testdb"
	"timetrak/internal/web/layout"
	"timetrak/internal/workspace"
)

// TestProjectsCrossWorkspaceDenialMatrix exercises every projects handler
// route as UserA (W1) against ProjectB (W2). Each MUST yield 404. Includes
// the "create referencing other-workspace client" denial scenario.
func TestProjectsCrossWorkspaceDenialMatrix(t *testing.T) {
	pool := testdb.Open(t)
	tpls := testdb.LoadTemplates(t)
	notFound := sharedhttp.NewNotFoundRenderer(tpls)
	sharedhttp.SetGlobalNotFound(notFound.Render)
	authz.SetNotFoundRenderer(notFound.Render)

	f := testdb.SeedAuthzFixture(t, pool)

	clientsSvc := clients.NewService(pool)
	projectsSvc := projects.NewService(pool)
	authzSvc := authz.NewService(pool.Pool)
	wsSvc := workspace.NewService(pool, authzSvc, nil)
	lay := layout.New(pool, wsSvc)
	h := projects.NewHandler(projectsSvc, clientsSvc, tpls, lay)

	mux := http.NewServeMux()
	h.Register(mux, func(next http.Handler) http.Handler { return next })

	cases := []struct {
		name   string
		method string
		path   string
		body   url.Values
	}{
		{"row", http.MethodGet, "/projects/" + f.ProjectB.String() + "/row", nil},
		{"edit-row", http.MethodGet, "/projects/" + f.ProjectB.String() + "/edit", nil},
		{"update", http.MethodPatch, "/projects/" + f.ProjectB.String(), url.Values{"name": {"Renamed"}}},
		{"archive", http.MethodPost, "/projects/" + f.ProjectB.String() + "/archive", nil},
		{"unarchive", http.MethodPost, "/projects/" + f.ProjectB.String() + "/unarchive", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRequest(tc.method, tc.path, tc.body)
			r = f.AttachAsUserA(r)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Result().StatusCode != http.StatusNotFound {
				t.Fatalf("%s %s: got status %d, want 404", tc.method, tc.path, w.Result().StatusCode)
			}
			if strings.Contains(w.Body.String(), f.ProjectB.String()) {
				t.Fatalf("%s %s: body discloses foreign project id", tc.method, tc.path)
			}
		})
	}

	// Create referencing other-workspace client returns 404 (NOT 422).
	t.Run("create-with-other-workspace-client", func(t *testing.T) {
		body := url.Values{
			"name":      {"X"},
			"client_id": {f.ClientB.String()}, // belongs to W2
		}
		r := newRequest(http.MethodPost, "/projects", body)
		r = f.AttachAsUserA(r)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Result().StatusCode != http.StatusNotFound {
			t.Fatalf("create with foreign client: got %d want 404", w.Result().StatusCode)
		}
	})

	t.Run("list-scoped", func(t *testing.T) {
		r := newRequest(http.MethodGet, "/projects", nil)
		r = f.AttachAsUserA(r)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("list: got %d want 200", w.Result().StatusCode)
		}
		if strings.Contains(w.Body.String(), f.ProjectB.String()) {
			t.Fatalf("list: leaked W2 project into W1 user's view")
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
