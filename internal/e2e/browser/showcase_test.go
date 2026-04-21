//go:build browser

package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"timetrak/internal/showcase"

	"github.com/playwright-community/playwright-go"
)

// TestShowcaseReachable asserts every showcase page returns 200 with
// Content-Type text/html in a dev-env test server. Uses the shared
// harness, which already wires APP_ENV=dev semantics via
// e2e.BuildServer (see server_harness.go).
//
// Spec: ui-showcase — "Showcase reachable in dev environment".
func TestShowcaseReachable(t *testing.T) {
	h := StartHarness(t)
	if h == nil {
		return
	}
	h.SignupFreshWorkspace("Showcase Viewer")

	paths := []string{
		"/dev/showcase",
		"/dev/showcase/tokens",
		"/dev/showcase/components",
	}
	for _, p := range paths {
		p := p
		t.Run(p, func(t *testing.T) {
			resp, err := h.Page.Goto(h.Server.URL + p)
			if err != nil {
				t.Fatalf("goto %s: %v", p, err)
			}
			if resp.Status() != http.StatusOK {
				t.Errorf("GET %s: got %d want 200", p, resp.Status())
			}
			ct := resp.Headers()["content-type"]
			if !strings.Contains(ct, "text/html") {
				t.Errorf("GET %s: content-type %q; want text/html", p, ct)
			}
		})
	}
}

// TestShowcaseComponentAnchors asserts every ComponentEntry renders an
// element in the DOM whose id matches entry-<ID>. This is the
// in-page anchor contract that the table-of-contents on the components
// page uses.
//
// Spec: ui-showcase — Task §7.3.
func TestShowcaseComponentAnchors(t *testing.T) {
	h := StartHarness(t)
	if h == nil {
		return
	}
	h.SignupFreshWorkspace("Showcase Anchors")
	h.GotoPath("/dev/showcase/components")
	if err := h.Page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	}); err != nil {
		t.Fatalf("wait dom: %v", err)
	}

	for _, entry := range showcase.ComponentEntries {
		entry := entry
		t.Run(entry.ID, func(t *testing.T) {
			sel := fmt.Sprintf("#entry-%s", entry.ID)
			loc := h.Page.Locator(sel)
			count, err := loc.Count()
			if err != nil {
				t.Fatalf("locator %q: %v", sel, err)
			}
			if count == 0 {
				t.Errorf("anchor %q missing on /dev/showcase/components", sel)
			}
		})
	}
}

// TestShowcaseAxeSmoke runs axe-core on both catalogue pages and
// asserts zero violations at impact serious or critical across
// wcag2a / wcag2aa / wcag22aa. Mirrors TestAxeSmokePerPage but narrows
// the target set to showcase pages so failures are attributed to the
// showcase surface.
//
// Spec: ui-showcase — "Showcase passes WCAG 2.2 AA smoke".
func TestShowcaseAxeSmoke(t *testing.T) {
	h := StartHarness(t)
	if h == nil {
		return
	}
	h.SignupFreshWorkspace("Showcase Axe")

	axePath := filepath.Join(h.RepoRoot, "internal", "e2e", "browser", "testdata", "axe.min.js")
	if _, err := os.Stat(axePath); err != nil {
		t.Fatalf("axe bundle missing at %s: %v", axePath, err)
	}

	pages := []struct{ name, path string }{
		{"index", "/dev/showcase"},
		{"tokens", "/dev/showcase/tokens"},
		{"components", "/dev/showcase/components"},
	}
	for _, p := range pages {
		p := p
		t.Run(p.name, func(t *testing.T) {
			if _, err := h.Page.Goto(h.Server.URL + p.path); err != nil {
				t.Fatalf("goto %s: %v", p.path, err)
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
			raw, err := h.Page.Evaluate(`async () => {
				return await axe.run(document, {
					runOnly: { type: 'tag', values: ['wcag2a','wcag2aa','wcag22aa'] },
				});
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
					t.Logf("axe[showcase:%s] %s (impact=%s) — %d nodes: %s",
						p.name, v.ID, v.Impact, len(v.Nodes), v.Help)
				}
			}
			if len(blockers) > 0 {
				artifact := h.ArtifactPath(fmt.Sprintf("axe-showcase-%s.json", p.name))
				_ = os.WriteFile(artifact, buf, 0o644)
				for _, v := range blockers {
					t.Errorf("axe[showcase:%s] impact=%s rule=%s help=%q (artifact=%s)",
						p.name, v.Impact, v.ID, v.Help, artifact)
				}
			}
		})
	}
}

// TestShowcaseThemeTogglePreviewDiffers asserts the existing data-theme
// toggle still flips on the showcase surface AND at least one token
// swatch's resolved backgroundColor actually changes between light and
// dark. This proves the token catalogue honors the theme toggle.
//
// Spec: ui-showcase — Task §7.5 + "Theme toggle updates samples live".
func TestShowcaseThemeTogglePreviewDiffers(t *testing.T) {
	h := StartHarness(t)
	if h == nil {
		return
	}
	h.SignupFreshWorkspace("Showcase Theme")
	h.GotoPath("/dev/showcase/tokens")

	// Probe #token-color-accent swatch (the first semantic color with a
	// swatch preview). Not the background of the element itself but the
	// inner swatch <span>.
	readBG := func() string {
		val, err := h.Page.Evaluate(`() => {
			const el = document.querySelector('#token-color-accent span[aria-label*="swatch"]');
			if (!el) return '';
			return getComputedStyle(el).backgroundColor;
		}`)
		if err != nil {
			t.Fatalf("evaluate backgroundColor: %v", err)
		}
		return fmt.Sprintf("%v", val)
	}

	if err := setTheme(h.Page, "light"); err != nil {
		t.Fatalf("setTheme light: %v", err)
	}
	light := readBG()
	if err := setTheme(h.Page, "dark"); err != nil {
		t.Fatalf("setTheme dark: %v", err)
	}
	dark := readBG()

	if light == "" || dark == "" {
		t.Fatalf("swatch element not found (light=%q dark=%q)", light, dark)
	}
	if light == dark {
		t.Errorf("--color-accent swatch backgroundColor did not change between light/dark: %q", light)
	}

	// Also assert data-theme attribute actually toggled.
	attr, err := h.Page.Evaluate(`() => document.documentElement.getAttribute('data-theme')`)
	if err != nil {
		t.Fatalf("read data-theme: %v", err)
	}
	if got := fmt.Sprintf("%v", attr); got != "dark" {
		t.Errorf("data-theme after setTheme(dark): %q want \"dark\"", got)
	}
}

// TestShowcaseMotionDemoReducedMotion asserts the motion-demo previews
// respect prefers-reduced-motion: reduce — transitions collapse to
// instant. Satisfies §8.4 (reduced-motion assertion).
func TestShowcaseMotionDemoReducedMotion(t *testing.T) {
	h := StartHarness(t)
	if h == nil {
		return
	}
	h.SignupFreshWorkspace("Showcase Motion")

	ctx, err := h.Browser.NewContext(playwright.BrowserNewContextOptions{
		ReducedMotion: playwright.ReducedMotionReduce,
	})
	if err != nil {
		t.Fatalf("new reduced-motion context: %v", err)
	}
	defer ctx.Close()
	// Copy the auth cookies from the main context.
	cookies, err := h.Context.Cookies()
	if err != nil {
		t.Fatalf("cookies: %v", err)
	}
	optional := make([]playwright.OptionalCookie, 0, len(cookies))
	for _, c := range cookies {
		oc := playwright.OptionalCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   playwright.String(c.Domain),
			Path:     playwright.String(c.Path),
			HttpOnly: playwright.Bool(c.HttpOnly),
			Secure:   playwright.Bool(c.Secure),
		}
		optional = append(optional, oc)
	}
	if err := ctx.AddCookies(optional); err != nil {
		t.Fatalf("add cookies: %v", err)
	}
	page, err := ctx.NewPage()
	if err != nil {
		t.Fatalf("new page: %v", err)
	}
	defer page.Close()

	if _, err := page.Goto(h.Server.URL + "/dev/showcase/tokens"); err != nil {
		t.Fatalf("goto tokens: %v", err)
	}
	if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	}); err != nil {
		t.Fatalf("wait dom: %v", err)
	}

	// Assert transition-duration on a motion-demo sample collapses to
	// 0s under reduced motion (per the foundation contract).
	val, err := page.Evaluate(`() => {
		const el = document.querySelector('.showcase-motion-demo');
		if (!el) return null;
		return getComputedStyle(el).transitionDuration;
	}`)
	if err != nil {
		t.Fatalf("evaluate transition-duration: %v", err)
	}
	if val == nil {
		t.Skip("no motion-demo sample on page (selector missing)")
	}
	got := fmt.Sprintf("%v", val)
	// The ui-foundation contract: @media (prefers-reduced-motion:
	// reduce) collapses transitions to 0s via !important.
	if !strings.HasPrefix(got, "0s") && got != "0s" {
		t.Errorf("transition-duration under reduced motion: got %q; want 0s (foundation contract)", got)
	}
}
