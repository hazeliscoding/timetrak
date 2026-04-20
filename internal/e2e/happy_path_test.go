// Package e2e_test exercises the full HTTP surface against a real Postgres.
// Skipped when DATABASE_URL is unset.
package e2e_test

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"timetrak/internal/auth"
	"timetrak/internal/clients"
	"timetrak/internal/projects"
	"timetrak/internal/rates"
	"timetrak/internal/reporting"
	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/clock"
	"timetrak/internal/shared/csrf"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/logging"
	"timetrak/internal/shared/session"
	"timetrak/internal/shared/templates"
	"timetrak/internal/shared/testdb"
	"timetrak/internal/tracking"
	"timetrak/internal/web/layout"
	"timetrak/internal/workspace"
)

func buildServer(t *testing.T) *httptest.Server {
	t.Helper()
	pool := testdb.Open(t)

	// Session + CSRF need a secret.
	secret := []byte("0123456789abcdef0123456789abcdef") // 32 bytes, test-only.
	store, err := session.NewStore(pool.Pool, secret, false)
	if err != nil {
		t.Fatal(err)
	}

	// Resolve templates dir from module root. Tests run with $CWD = the test file's package dir,
	// so walk up until we find web/templates.
	tmplDir := findTemplatesDir(t)
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
	mux := http.NewServeMux()

	auth.NewHandler(authSvc, store, tpls, limiter).Register(mux)
	workspace.NewHandler(wsSvc).Register(mux)
	protect := func(next http.Handler) http.Handler {
		return authz.RequireAuth(azSvc.RequireWorkspaceMember(next))
	}
	clients.NewHandler(clientsSvc, tpls, lay).Register(mux, protect)
	projects.NewHandler(projectsSvc, clientsSvc, tpls, lay).Register(mux, protect)
	rates.NewHandler(ratesSvc, clientsSvc, projectsSvc, tpls, lay).Register(mux, protect)
	tracking.NewHandler(trackingSvc, projectsSvc, clientsSvc, reportingSvc, tpls, lay).Register(mux, protect)
	reporting.NewHandler(reportingSvc, clientsSvc, projectsSvc, tpls, lay).Register(mux, protect)

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

func findTemplatesDir(t *testing.T) string {
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

type client struct {
	t  *testing.T
	ts *httptest.Server
	h  *http.Client
}

func newClient(t *testing.T, ts *httptest.Server) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, ts: ts, h: &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

func (c *client) get(path string) *http.Response {
	req, _ := http.NewRequest("GET", c.ts.URL+path, nil)
	resp, err := c.h.Do(req)
	if err != nil {
		c.t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func (c *client) post(path string, form url.Values) *http.Response {
	// Ensure CSRF cookie + token available.
	c.get("/") // warm the cookie jar.
	// Pick up csrf cookie.
	token := c.csrfToken()
	form.Set("csrf_token", token)
	req, _ := http.NewRequest("POST", c.ts.URL+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.h.Do(req)
	if err != nil {
		c.t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func (c *client) csrfToken() string {
	u, _ := url.Parse(c.ts.URL)
	for _, ck := range c.h.Jar.Cookies(u) {
		if ck.Name == "tt_csrf" {
			return ck.Value
		}
	}
	return ""
}

func body(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func TestHappyPathSignupToReport(t *testing.T) {
	ts := buildServer(t)
	c := newClient(t, ts)

	// Signup.
	resp := c.post("/signup", url.Values{
		"email":        {"alice@example.com"},
		"password":     {"correct-horse-battery"},
		"display_name": {"Alice"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("signup: status=%d body=%s", resp.StatusCode, body(t, resp))
	}

	// Dashboard renders.
	resp = c.get("/dashboard")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dashboard status=%d body=%s", resp.StatusCode, body(t, resp))
	}
	b := body(t, resp)
	if !strings.Contains(b, "Dashboard") {
		t.Fatalf("dashboard missing heading: %s", b[:200])
	}

	// Create a client.
	resp = c.post("/clients", url.Values{"name": {"Acme"}, "contact_email": {"acme@x.test"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create client: %d", resp.StatusCode)
	}

	// Fetch the client id.
	resp = c.get("/clients")
	page := body(t, resp)
	clientID := extractFirst(t, `id="client-([0-9a-f-]+)"`, page)

	// Create a project.
	resp = c.post("/projects", url.Values{
		"client_id":        {clientID},
		"name":             {"Website"},
		"default_billable": {"on"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create project: %d body=%s", resp.StatusCode, body(t, resp))
	}
	resp = c.get("/projects")
	page = body(t, resp)
	projectID := extractFirst(t, `id="project-([0-9a-f-]+)"`, page)

	// Create a workspace-default rate.
	todayStr := time.Now().UTC().Format("2006-01-02")
	resp = c.post("/rates", url.Values{
		"scope":          {"workspace"},
		"currency_code":  {"USD"},
		"hourly_decimal": {"125.00"},
		"effective_from": {todayStr},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create rate: %d body=%s", resp.StatusCode, body(t, resp))
	}

	// Start & stop a timer.
	resp = c.post("/timer/start", url.Values{"project_id": {projectID}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("start timer: %d body=%s", resp.StatusCode, body(t, resp))
	}
	resp = c.post("/timer/stop", url.Values{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop timer: %d body=%s", resp.StatusCode, body(t, resp))
	}

	// Reports page renders with our numbers.
	resp = c.get("/reports?preset=today")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reports: %d body=%s", resp.StatusCode, body(t, resp))
	}
	if !strings.Contains(body(t, resp), "Reports") {
		t.Fatalf("reports missing heading")
	}
}

func TestWorkspaceIsolation404(t *testing.T) {
	ts := buildServer(t)
	// Seed user A with a workspace + client.
	a := newClient(t, ts)
	if r := a.post("/signup", url.Values{
		"email": {"a@example.com"}, "password": {"correct-horse-battery"}, "display_name": {"A"},
	}); r.StatusCode != http.StatusSeeOther {
		t.Fatalf("signup A: %d", r.StatusCode)
	}
	if r := a.post("/clients", url.Values{"name": {"A-client"}}); r.StatusCode != http.StatusSeeOther {
		t.Fatalf("create client: %d", r.StatusCode)
	}
	r := a.get("/clients")
	aClientID := extractFirst(t, `id="client-([0-9a-f-]+)"`, body(t, r))

	// Seed user B with a fresh workspace.
	b := newClient(t, ts)
	if r := b.post("/signup", url.Values{
		"email": {"b@example.com"}, "password": {"correct-horse-battery"}, "display_name": {"B"},
	}); r.StatusCode != http.StatusSeeOther {
		t.Fatalf("signup B: %d", r.StatusCode)
	}

	// B tries to read or mutate A's client → 404.
	if r := b.get("/clients/" + aClientID + "/row"); r.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-workspace read should 404: got %d", r.StatusCode)
	}
	if r := b.post("/clients/"+aClientID+"/archive", url.Values{}); r.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-workspace archive should 404: got %d", r.StatusCode)
	}
}

// extractFirst returns the first capture group matching `pattern` in `s`, or fails the test.
func extractFirst(t *testing.T, pattern, s string) string {
	t.Helper()
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		t.Fatalf("pattern %q not found", pattern)
	}
	return m[1]
}

// Ensure tests never depend on a lingering background context.
var _ = context.Background
