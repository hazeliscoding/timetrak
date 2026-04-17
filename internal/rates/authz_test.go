package rates_test

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
	"timetrak/internal/shared/authz"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/testdb"
	"timetrak/internal/web/layout"
	"timetrak/internal/workspace"
)

// TestRatesCrossWorkspaceDenialMatrix exercises every rates handler route
// as UserA (W1) against rates resources in W2. Each MUST yield 404. Also
// verifies the Resolve service path: resolving a foreign project as if it
// were W1's MUST not return W2 rate rules.
func TestRatesCrossWorkspaceDenialMatrix(t *testing.T) {
	pool := testdb.Open(t)
	tpls := testdb.LoadTemplates(t)
	notFound := sharedhttp.NewNotFoundRenderer(tpls)
	sharedhttp.SetGlobalNotFound(notFound.Render)
	authz.SetNotFoundRenderer(notFound.Render)

	f := testdb.SeedAuthzFixture(t, pool)

	// Seed a rate rule in W2 for UserB's client.
	ruleB := uuid.New()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO rate_rules (id, workspace_id, client_id, currency_code, hourly_rate_minor, effective_from)
		VALUES ($1, $2, $3, 'USD', 9999, CURRENT_DATE - INTERVAL '7 days')
	`, ruleB, f.WorkspaceB, f.ClientB); err != nil {
		t.Fatalf("seed rule: %v", err)
	}

	clientsSvc := clients.NewService(pool)
	projectsSvc := projects.NewService(pool)
	ratesSvc := rates.NewService(pool)
	authzSvc := authz.NewService(pool.Pool)
	wsSvc := workspace.NewService(pool, authzSvc, nil)
	lay := layout.New(pool, wsSvc)
	h := rates.NewHandler(ratesSvc, clientsSvc, projectsSvc, tpls, lay)

	mux := http.NewServeMux()
	h.Register(mux, func(next http.Handler) http.Handler { return next })

	// Delete a foreign rate rule: 404.
	t.Run("delete-foreign-rule", func(t *testing.T) {
		body := url.Values{}
		r := newRequest(http.MethodPost, "/rates/"+ruleB.String()+"/delete", body)
		r = f.AttachAsUserA(r)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Result().StatusCode != http.StatusNotFound {
			t.Fatalf("delete foreign rule: got %d want 404", w.Result().StatusCode)
		}
		// Verify the rule still exists in W2.
		var n int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM rate_rules WHERE id = $1`, ruleB).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("foreign rate rule was deleted across workspaces — count=%d", n)
		}
	})

	// Create scoped to foreign client: 404.
	t.Run("create-with-foreign-client", func(t *testing.T) {
		body := url.Values{
			"scope":          {"client"},
			"client_id":      {f.ClientB.String()},
			"currency_code":  {"USD"},
			"hourly_decimal": {"100"},
			"effective_from": {time.Now().UTC().Format("2006-01-02")},
		}
		r := newRequest(http.MethodPost, "/rates", body)
		r = f.AttachAsUserA(r)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Result().StatusCode != http.StatusNotFound {
			t.Fatalf("create with foreign client: got %d want 404", w.Result().StatusCode)
		}
	})

	t.Run("create-with-foreign-project", func(t *testing.T) {
		body := url.Values{
			"scope":          {"project"},
			"project_id":     {f.ProjectB.String()},
			"currency_code":  {"USD"},
			"hourly_decimal": {"100"},
			"effective_from": {time.Now().UTC().Format("2006-01-02")},
		}
		r := newRequest(http.MethodPost, "/rates", body)
		r = f.AttachAsUserA(r)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Result().StatusCode != http.StatusNotFound {
			t.Fatalf("create with foreign project: got %d want 404", w.Result().StatusCode)
		}
	})

	// List view scoped to active workspace: never includes W2 rules.
	t.Run("list-scoped", func(t *testing.T) {
		r := newRequest(http.MethodGet, "/rates", nil)
		r = f.AttachAsUserA(r)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("list: got %d want 200", w.Result().StatusCode)
		}
		if strings.Contains(w.Body.String(), ruleB.String()) {
			t.Fatalf("list: leaked W2 rule into W1 user's view")
		}
	})

	// Resolve service path: asking W1 to resolve a W2 project MUST NOT
	// surface the W2 rule. The current Resolve implementation returns
	// Found=false because the project isn't in W1.
	t.Run("resolve-other-workspace-project", func(t *testing.T) {
		res, err := ratesSvc.Resolve(context.Background(), f.WorkspaceA, f.ProjectB, time.Now().UTC())
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		if res.Found && res.RuleID == ruleB {
			t.Fatalf("Resolve leaked a foreign workspace's rate rule")
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
