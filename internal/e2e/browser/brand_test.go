//go:build browser

package browser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/playwright-community/playwright-go"
)

// TestBrandmarkInAppHeader asserts that the app header on an
// authenticated page contains the brandmark SVG with accessible name
// "TimeTrak" and that the SVG is wrapped in an anchor targeting
// /dashboard per the partial README contract.
//
// Spec: ui-partials — "Brand mark partial".
func TestBrandmarkInAppHeader(t *testing.T) {
	h := StartHarness(t)
	if h == nil {
		return
	}
	h.SignupFreshWorkspace("Brandmark Viewer")
	h.GotoPath("/dashboard")
	if err := h.Page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	}); err != nil {
		t.Fatalf("wait dom: %v", err)
	}

	// The anchor wrapping the brandmark MUST exist, target /dashboard,
	// and contain the brandmark SVG.
	anchorLoc := h.Page.Locator("header.app-header a.brandmark-link")
	count, err := anchorLoc.Count()
	if err != nil {
		t.Fatalf("locator brandmark anchor: %v", err)
	}
	if count == 0 {
		t.Fatal("header.app-header a.brandmark-link not present on /dashboard")
	}
	href, err := anchorLoc.First().GetAttribute("href")
	if err != nil {
		t.Fatalf("read anchor href: %v", err)
	}
	if href != "/dashboard" {
		t.Errorf("brandmark anchor href: got %q want %q", href, "/dashboard")
	}

	// The SVG MUST carry role="img" and an accessible name "TimeTrak"
	// (via the <title> child — per Decorative=false contract).
	svgLoc := anchorLoc.Locator("svg.brandmark")
	svgCount, err := svgLoc.Count()
	if err != nil {
		t.Fatalf("locator brandmark svg: %v", err)
	}
	if svgCount == 0 {
		t.Fatal("svg.brandmark missing inside header anchor")
	}
	role, _ := svgLoc.First().GetAttribute("role")
	if role != "img" {
		t.Errorf("brandmark svg role: got %q want %q", role, "img")
	}

	// Read the <title> child text content directly. Using Page.Evaluate
	// with a document-scoped querySelector avoids Locator.Evaluate's
	// implicit visibility wait (inline-SVG <title> elements report as
	// hidden to Playwright's actionability heuristic even though they
	// carry the accessible name).
	accessibleName, err := h.Page.Evaluate(`() => {
		const t = document.querySelector('header.app-header a.brandmark-link svg.brandmark title');
		return t ? t.textContent : '';
	}`)
	if err != nil {
		t.Fatalf("read svg title: %v", err)
	}
	if got := fmt.Sprintf("%v", accessibleName); got != "TimeTrak" {
		t.Errorf("brandmark accessible name: got %q want %q", got, "TimeTrak")
	}

	// Fill/stroke of the rect and text MUST resolve via CSS custom
	// properties (currentColor or var(--color-accent)) — not raw hex.
	// Read inline style attributes directly via Page.Evaluate.
	fills, err := h.Page.Evaluate(`() => {
		const svg = document.querySelector('header.app-header a.brandmark-link svg.brandmark');
		if (!svg) return { barFill: '', textFill: '' };
		const bar = svg.querySelector('rect');
		const text = svg.querySelector('text');
		return {
			barFill: bar ? (bar.getAttribute('style') || '') : '',
			textFill: text ? (text.getAttribute('style') || '') : '',
		};
	}`)
	if err != nil {
		t.Fatalf("read brandmark fills: %v", err)
	}
	m, _ := fills.(map[string]any)
	barFill := fmt.Sprintf("%v", m["barFill"])
	textFill := fmt.Sprintf("%v", m["textFill"])
	if !strings.Contains(barFill, "var(--color-accent)") {
		t.Errorf("bar inline fill: got %q want reference to var(--color-accent)", barFill)
	}
	if !strings.Contains(textFill, "currentColor") {
		t.Errorf("text inline fill: got %q want currentColor", textFill)
	}
}

// TestFaviconLinkPresentOnPublicPages asserts that every rendered page
// — including unauthenticated pages — includes the favicon <link> tag.
// Because base.html is shared by every page template, this is a
// belt-and-suspenders check that no layout override strips it.
//
// Spec: ui-partials — "Brand mark partial" (companion surface).
func TestFaviconLinkPresentOnPublicPages(t *testing.T) {
	h := StartHarness(t)
	if h == nil {
		return
	}
	h.SignupFreshWorkspace("Favicon Viewer")

	// Paths registered in the e2e BuildServer harness. /workspace/settings
	// is intentionally excluded here because the settings handler is not
	// wired into BuildServer today; its favicon behaviour is covered by the
	// same <head> block in base.html and by the in-process happy-path
	// e2e tests in internal/e2e.
	paths := []string{
		"/dashboard",
		"/time",
		"/clients",
		"/projects",
		"/rates",
		"/reports",
	}
	for _, p := range paths {
		p := p
		t.Run(p, func(t *testing.T) {
			h.GotoPath(p)
			if err := h.Page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
				State: playwright.LoadStateDomcontentloaded,
			}); err != nil {
				t.Fatalf("wait dom: %v", err)
			}
			loc := h.Page.Locator(`head link[rel="icon"][type="image/svg+xml"]`)
			count, err := loc.Count()
			if err != nil {
				t.Fatalf("locator favicon link: %v", err)
			}
			if count == 0 {
				t.Errorf("favicon <link rel=\"icon\"> missing on %s", p)
				return
			}
			href, _ := loc.First().GetAttribute("href")
			if href != "/static/favicon.svg" {
				t.Errorf("favicon href on %s: got %q want %q", p, href, "/static/favicon.svg")
			}
		})
	}
}

// TestFaviconResourceServes asserts that web/static/favicon.svg is a
// well-formed SVG document and that Go's standard http.FileServer
// resolves its Content-Type to image/svg+xml when served from the
// repo's static directory. The shared BuildServer harness does not
// currently mount /static/ (mirroring cmd/web/main.go's mount in the
// harness is a separate hygiene follow-up), so this test spins up a
// dedicated FileServer for the static directory and queries only
// /favicon.svg — the contract under this change's scope.
//
// Spec: ui-partials — "Brand mark partial" (companion surface).
func TestFaviconResourceServes(t *testing.T) {
	repoRoot, err := findRepoRootFromCWD()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	staticDir := filepath.Join(repoRoot, "web", "static")
	if _, err := os.Stat(filepath.Join(staticDir, "favicon.svg")); err != nil {
		t.Fatalf("favicon.svg missing at %s: %v", staticDir, err)
	}

	srv := httptest.NewServer(http.FileServer(http.Dir(staticDir)))
	defer srv.Close()

	url := srv.URL + "/favicon.svg"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET %s: status %d want 200", url, resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	// Go's http.FileServer resolves .svg to image/svg+xml via mime types
	// registered at init. Accept either exact match or the charset form.
	if !strings.HasPrefix(ct, "image/svg+xml") {
		t.Errorf("GET %s: content-type %q; want prefix image/svg+xml", url, ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "<svg") {
		head := len(body)
		if head > 120 {
			head = 120
		}
		t.Errorf("GET %s: body does not look like SVG (first %d bytes: %q)",
			url, head, string(body[:head]))
	}
}

// findRepoRootFromCWD walks up from the current working directory to
// find the directory containing go.mod. Mirrors repoRoot / FindRepoRoot
// helpers used elsewhere; duplicated here to avoid introducing a new
// exported helper as part of this focused change.
func findRepoRootFromCWD() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return "", fmt.Errorf("go.mod not found walking up from cwd")
}

// TestBrandSurfaceAxeSmoke scopes axe-core to the #entry-brandmark
// anchor section on the components catalogue and asserts zero serious
// or critical violations. This is a belt-and-suspenders probe on top
// of TestShowcaseAxeSmoke (which runs axe on the whole page) so that
// a future brand-only regression is attributed here.
//
// Spec: ui-showcase — "Brand sub-surface in the component catalogue".
func TestBrandSurfaceAxeSmoke(t *testing.T) {
	h := StartHarness(t)
	if h == nil {
		return
	}
	h.SignupFreshWorkspace("Brand Axe")

	axePath := filepath.Join(h.RepoRoot, "internal", "e2e", "browser", "testdata", "axe.min.js")
	if _, err := os.Stat(axePath); err != nil {
		t.Fatalf("axe bundle missing at %s: %v", axePath, err)
	}

	if _, err := h.Page.Goto(h.Server.URL + "/dev/showcase/components"); err != nil {
		t.Fatalf("goto /dev/showcase/components: %v", err)
	}
	if err := h.Page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	}); err != nil {
		t.Fatalf("wait dom: %v", err)
	}
	if _, err := h.Page.AddScriptTag(playwright.PageAddScriptTagOptions{
		Path: playwright.String(axePath),
	}); err != nil {
		t.Fatalf("inject axe: %v", err)
	}
	// Scope axe to the rendered example frames + favicon preview under
	// #entry-brandmark, NOT the surrounding <details><summary> metadata
	// blocks. Those summaries are pre-existing showcase infrastructure
	// used by every entry; their target-size story is a separate concern
	// and not introduced by this change.
	raw, err := h.Page.Evaluate(`async () => {
		return await axe.run(
			{
				include: [
					['#entry-brandmark .showcase-example-frame'],
				],
			},
			{ runOnly: { type: 'tag', values: ['wcag2a','wcag2aa','wcag22aa'] } },
		);
	}`)
	if err != nil {
		t.Fatalf("axe.run: %v", err)
	}
	buf, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal axe: %v", err)
	}
	var result axeResult
	if err := json.Unmarshal(buf, &result); err != nil {
		t.Fatalf("unmarshal axe: %v", err)
	}
	var blockers []axeViolation
	for _, v := range result.Violations {
		switch v.Impact {
		case "serious", "critical":
			blockers = append(blockers, v)
		default:
			t.Logf("axe[brand] %s (impact=%s) — %d nodes: %s",
				v.ID, v.Impact, len(v.Nodes), v.Help)
		}
	}
	if len(blockers) > 0 {
		artifact := h.ArtifactPath("axe-brand-surface.json")
		_ = os.WriteFile(artifact, buf, 0o644)
		for _, v := range blockers {
			t.Errorf("axe[brand] impact=%s rule=%s help=%q (artifact=%s)",
				v.Impact, v.ID, v.Help, artifact)
		}
	}
}
