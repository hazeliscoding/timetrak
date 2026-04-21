//go:build browser

// Package browser holds TimeTrak's opt-in browser-driven UI contract
// tests. Every file in this directory carries //go:build browser so the
// default `make test` (and plain `go test ./...`) does not compile any of
// it — browsers do not need to be installed to run the main suite.
//
// # Pinned Playwright-Go version
//
// This harness is pinned to github.com/playwright-community/playwright-go
// v0.5700.1 (see go.mod). Upgrading the pin requires its own OpenSpec
// change: the pinned Playwright driver bundles a specific Chromium build,
// and bumping it has cascading implications for CI cache keys, axe-core
// compatibility, and the contract assertions below. Do NOT bump this in
// passing.
//
// # Running
//
//	make browser-install   # one-time: downloads driver + Chromium (~200MB)
//	make test-browser      # runs go test -tags=browser ./internal/e2e/browser/...
//
// When browser binaries are missing, every test in this package skips
// gracefully with a pointer to `make browser-install`, mirroring the
// internal/shared/testdb skip pattern.
//
// # CI note
//
// At the time this harness landed, the repository had no CI configuration
// checked in. A ready-to-copy GitHub Actions job shape lives at
// docs/ci/browser-tests.yml.example; adopt it (or an equivalent for
// whatever CI platform the team picks) when the repo first gains CI. The
// job MUST cache the Playwright install directory keyed on the pinned
// driver version and upload testdata/browser-artifacts/ on failure.
//
// # Determinism
//
// Browser tests synchronize on deterministic events ONLY:
// htmx:afterSettle, completed responses, WaitForSelector,
// WaitForLoadState. Never time.Sleep. Never arbitrary sleeps in the page.
// See waitForHTMXSettle below.
//
// # Token reads
//
// Focus-ring and related token-backed assertions MUST read live values
// from getComputedStyle(document.documentElement). Hardcoded hex/rgb
// strings for tokenized properties are prohibited — they'd miss the one
// regression class this harness exists to catch (token renamed in one
// place, missed in another).
package browser

import (
	"fmt"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"timetrak/internal/e2e"

	"github.com/playwright-community/playwright-go"
)

// PlaywrightVersion is the exact Playwright-Go release pinned for this
// harness. Surfaced here (and at the top of the file) so a reader can
// confirm the pin without opening go.mod.
const PlaywrightVersion = "v0.5700.1"

// Harness bundles everything a browser test needs: the test server, the
// Playwright instance, the browser + context + page, and the repo root
// so tests can resolve static files like the vendored axe-core bundle.
type Harness struct {
	T        *testing.T
	Server   *httptest.Server
	PW       *playwright.Playwright
	Browser  playwright.Browser
	Context  playwright.BrowserContext
	Page     playwright.Page
	RepoRoot string

	artifactsDir string
	traceStopped bool
}

// StartHarness brings up a fresh TimeTrak server, launches Chromium, opens
// a browser context with a cookie jar and tracing enabled, and returns a
// Harness whose Page is parked on about:blank.
//
// If Playwright's driver or browser binaries are not installed, this
// function calls t.Skipf with a message pointing at `make browser-install`
// and returns nil. It does NOT fail the test.
//
// Callers should defer h.Close() to flush traces and clean up processes.
func StartHarness(t *testing.T) *Harness {
	t.Helper()

	server := e2e.BuildServer(t)
	repoRoot := e2e.FindRepoRoot(t)

	pw, err := playwright.Run()
	if err != nil {
		t.Skipf("browser binaries not installed; run 'make browser-install' — %v", err)
		return nil
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		_ = pw.Stop()
		t.Skipf("browser binaries not installed; run 'make browser-install' — %v", err)
		return nil
	}

	artifactsDir := filepath.Join(repoRoot, "testdata", "browser-artifacts", sanitizeTestName(t.Name()))
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		_ = browser.Close()
		_ = pw.Stop()
		t.Fatalf("mkdir artifacts dir: %v", err)
	}

	ctx, err := browser.NewContext()
	if err != nil {
		_ = browser.Close()
		_ = pw.Stop()
		t.Fatalf("new context: %v", err)
	}

	// Start a single trace chunk for the whole harness; dumped on failure.
	if err := ctx.Tracing().Start(playwright.TracingStartOptions{
		Screenshots: playwright.Bool(true),
		Snapshots:   playwright.Bool(true),
		Sources:     playwright.Bool(true),
	}); err != nil {
		t.Logf("tracing start failed (continuing): %v", err)
	}

	page, err := ctx.NewPage()
	if err != nil {
		_ = ctx.Close()
		_ = browser.Close()
		_ = pw.Stop()
		t.Fatalf("new page: %v", err)
	}

	h := &Harness{
		T:            t,
		Server:       server,
		PW:           pw,
		Browser:      browser,
		Context:      ctx,
		Page:         page,
		RepoRoot:     repoRoot,
		artifactsDir: artifactsDir,
	}
	t.Cleanup(func() { h.Close() })
	return h
}

// Close flushes the trace (writing it to the artifacts dir if the test
// failed) and tears down Playwright. Safe to call multiple times.
func (h *Harness) Close() {
	if h == nil {
		return
	}
	if !h.traceStopped && h.Context != nil {
		if h.T.Failed() {
			tracePath := filepath.Join(h.artifactsDir, "trace.zip")
			// Best-effort screenshot as well.
			if h.Page != nil {
				shotPath := filepath.Join(h.artifactsDir, "failure.png")
				_, _ = h.Page.Screenshot(playwright.PageScreenshotOptions{
					Path:     playwright.String(shotPath),
					FullPage: playwright.Bool(true),
				})
				h.T.Logf("browser failure artifacts: %s and %s", shotPath, tracePath)
			} else {
				h.T.Logf("browser failure trace: %s", tracePath)
			}
			_ = h.Context.Tracing().Stop(tracePath)
		} else {
			_ = h.Context.Tracing().Stop()
		}
		h.traceStopped = true
	}
	if h.Browser != nil {
		_ = h.Browser.Close()
		h.Browser = nil
	}
	if h.PW != nil {
		_ = h.PW.Stop()
		h.PW = nil
	}
}

// ArtifactPath returns a path under the per-test artifacts directory.
// Useful for axe JSON dumps and ad-hoc screenshots.
func (h *Harness) ArtifactPath(name string) string {
	return filepath.Join(h.artifactsDir, name)
}

// SignupFreshWorkspace runs the real signup flow over HTTP against the
// test server, obtains a session cookie, and installs it on the browser
// context. Mirrors the hermetic approach used by happy_path_test.go — we
// never use the demo seed from browser tests.
//
// Returns the email of the created user.
func (h *Harness) SignupFreshWorkspace(displayName string) string {
	h.T.Helper()
	jar, _ := cookiejar.New(nil)
	hc := newSignupHTTPClient(jar)

	// Warm the CSRF cookie.
	warm, err := hc.Get(h.Server.URL + "/")
	if err != nil {
		h.T.Fatalf("warm GET /: %v", err)
	}
	_ = warm.Body.Close()

	csrfToken := ""
	u, _ := url.Parse(h.Server.URL)
	for _, ck := range jar.Cookies(u) {
		if ck.Name == "tt_csrf" {
			csrfToken = ck.Value
		}
	}
	if csrfToken == "" {
		h.T.Fatalf("csrf cookie not set after warm request")
	}

	email := fmt.Sprintf("browser-%s@example.test", randSuffix())
	form := url.Values{
		"email":        {email},
		"password":     {"correct-horse-battery"},
		"display_name": {displayName},
		"csrf_token":   {csrfToken},
	}
	resp, err := hc.PostForm(h.Server.URL+"/signup", form)
	if err != nil {
		h.T.Fatalf("signup POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		h.T.Fatalf("signup: status=%d", resp.StatusCode)
	}

	// Copy cookies into the browser context.
	pwCookies := make([]playwright.OptionalCookie, 0, 4)
	for _, ck := range jar.Cookies(u) {
		pwCookies = append(pwCookies, playwright.OptionalCookie{
			Name:     ck.Name,
			Value:    ck.Value,
			Domain:   playwright.String(mustHost(h.Server.URL)),
			Path:     playwright.String("/"),
			HttpOnly: playwright.Bool(ck.HttpOnly),
			Secure:   playwright.Bool(ck.Secure),
		})
	}
	if err := h.Context.AddCookies(pwCookies); err != nil {
		h.T.Fatalf("AddCookies: %v", err)
	}
	return email
}

// GotoPath navigates the harness's page to the given server-relative path
// and waits for the DOM to be loaded. HTMX swap tests should still call
// WaitForHTMXSettle after user-triggered interactions.
func (h *Harness) GotoPath(path string) {
	h.T.Helper()
	if _, err := h.Page.Goto(h.Server.URL + path); err != nil {
		h.T.Fatalf("goto %s: %v", path, err)
	}
	if err := h.Page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	}); err != nil {
		h.T.Fatalf("wait domcontentloaded: %v", err)
	}
}

// WaitForHTMXSettle resolves once the browser emits htmx:afterSettle
// on document.body. Use this after every HTMX-driven interaction before
// asserting on post-swap DOM state. NEVER paper over timing with
// time.Sleep — this is why the helper exists.
func WaitForHTMXSettle(page playwright.Page) error {
	// Register a one-shot promise on window that resolves on the next
	// htmx:afterSettle, then wait for it. Using WaitForFunction on the
	// flag set inside the listener keeps the wait deterministic.
	_, err := page.Evaluate(`() => {
		window.__ttSettled = false;
		document.body.addEventListener('htmx:afterSettle', () => {
			window.__ttSettled = true;
		}, { once: true });
	}`)
	if err != nil {
		return fmt.Errorf("install settle listener: %w", err)
	}
	_, err = page.WaitForFunction(`window.__ttSettled === true`, nil)
	if err != nil {
		return fmt.Errorf("wait for htmx:afterSettle: %w", err)
	}
	return nil
}

// --- internals ---

func sanitizeTestName(name string) string {
	return strings.NewReplacer("/", "_", " ", "_", ":", "_").Replace(name)
}

func mustHost(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return "127.0.0.1"
	}
	h := parsed.Hostname()
	if h == "" {
		return "127.0.0.1"
	}
	return h
}
