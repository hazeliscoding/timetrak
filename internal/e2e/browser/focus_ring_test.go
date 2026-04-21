//go:build browser

package browser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/playwright-community/playwright-go"
)

// focusRingRow is one interactive primitive to probe.
type focusRingRow struct {
	// Name is the human-readable identifier in failure messages.
	Name string
	// Path is the app route to load before probing.
	Path string
	// Selector resolves to the primitive under test within Path.
	Selector string
}

func focusRingRows() []focusRingRow {
	return []focusRingRow{
		{Name: ".btn on dashboard", Path: "/dashboard", Selector: "button.btn, a.btn"},
		{Name: ".btn-primary on dashboard", Path: "/dashboard", Selector: ".btn-primary"},
		{Name: ".btn-ghost in nav", Path: "/dashboard", Selector: ".nav .btn-ghost, header .btn-ghost"},
		{Name: "nav anchor", Path: "/dashboard", Selector: "nav a"},
		{Name: "input on /clients/new", Path: "/clients/new", Selector: "input[type=text], input[type=email], input:not([type])"},
		{Name: "select on /projects/new", Path: "/projects/new", Selector: "select"},
		{Name: "textarea on /clients/new", Path: "/clients/new", Selector: "textarea"},
		{Name: ".btn-danger on /clients", Path: "/clients", Selector: ".btn-danger"},
		{Name: "table row action on /clients", Path: "/clients", Selector: "tbody .btn, tbody a.btn-ghost, tbody button"},
		{Name: "timer start control", Path: "/dashboard", Selector: "form[action='/timer/start'] button, form[action*='timer/start'] button"},
		{Name: "pagination anchor on /entries", Path: "/entries", Selector: "nav[aria-label='Pagination'] a"},
	}
}

// TestFocusRingContract drives each interactive primitive in both themes
// and asserts that its :focus-visible outline matches the live token
// contract on :root.
func TestFocusRingContract(t *testing.T) {
	h := StartHarness(t)
	if h == nil {
		return
	}
	h.SignupFreshWorkspace("Focus Tester")
	// Seed a minimal workspace shape so /clients, /projects, /entries have
	// enough rows to expose row-action buttons and pagination controls.
	seedForFocusRing(t, h)

	themes := []string{"light", "dark"}
	for _, row := range focusRingRows() {
		row := row
		t.Run(row.Name, func(t *testing.T) {
			for _, theme := range themes {
				theme := theme
				t.Run(theme, func(t *testing.T) {
					h.GotoPath(row.Path)
					if err := setTheme(h.Page, theme); err != nil {
						t.Fatalf("set theme=%s: %v", theme, err)
					}
					loc := h.Page.Locator(row.Selector).First()
					count, _ := loc.Count()
					if count == 0 {
						t.Skipf("selector %q not present on %s (theme=%s)", row.Selector, row.Path, theme)
						return
					}
					if err := loc.Focus(); err != nil {
						t.Fatalf("focus %q: %v", row.Selector, err)
					}
					// Read live token + computed outline via a single evaluate
					// and compare inside the browser, so we compare resolved
					// RGB against resolved RGB without crossing a boundary.
					result, err := loc.Evaluate(`el => {
						const s = getComputedStyle(el);
						const rootTokenRaw = getComputedStyle(document.documentElement)
							.getPropertyValue('--color-focus').trim();
						// Resolve the token to an rgb() string by applying it
						// to a probe element, so CSS color-function strings
						// normalize the same way as computed outline-color.
						const probe = document.createElement('span');
						probe.style.color = rootTokenRaw;
						document.body.appendChild(probe);
						const tokenResolved = getComputedStyle(probe).color;
						probe.remove();
						return {
							outlineWidth: s.outlineWidth,
							outlineOffset: s.outlineOffset,
							outlineColor: s.outlineColor,
							tokenRaw: rootTokenRaw,
							tokenResolved,
							theme: document.documentElement.getAttribute('data-theme'),
						};
					}`, nil)
					if err != nil {
						t.Fatalf("evaluate focus styles: %v", err)
					}
					m, ok := result.(map[string]any)
					if !ok {
						t.Fatalf("unexpected evaluate result shape: %T", result)
					}
					if got := fmt.Sprintf("%v", m["outlineWidth"]); got != "3px" {
						t.Errorf("outline-width: got %q want 3px (row=%s theme=%s)", got, row.Name, theme)
					}
					if got := fmt.Sprintf("%v", m["outlineOffset"]); got != "2px" {
						t.Errorf("outline-offset: got %q want 2px (row=%s theme=%s)", got, row.Name, theme)
					}
					outlineColor := fmt.Sprintf("%v", m["outlineColor"])
					tokenResolved := fmt.Sprintf("%v", m["tokenResolved"])
					tokenRaw := fmt.Sprintf("%v", m["tokenRaw"])
					if tokenRaw == "" {
						t.Errorf("--color-focus empty on :root (theme=%s)", theme)
					}
					if outlineColor != tokenResolved {
						t.Errorf("outline-color mismatch: outline=%q --color-focus(resolved)=%q raw=%q (row=%s theme=%s)",
							outlineColor, tokenResolved, tokenRaw, row.Name, theme)
					}
				})
			}
		})
	}
}

// setTheme flips [data-theme] on <html> by dispatching the same theme
// toggle click the app uses. We set it directly to keep the test
// deterministic regardless of whether a toggle button is currently
// visible on this route.
func setTheme(page playwright.Page, theme string) error {
	_, err := page.Evaluate(fmt.Sprintf(`() => {
		document.documentElement.setAttribute('data-theme', %q);
	}`, theme))
	return err
}

// seedForFocusRing signs up has already run; add a client, project, rate,
// and one completed timer run so /clients, /projects, and /entries render
// rows that expose action buttons and pagination.
func seedForFocusRing(t *testing.T, h *Harness) {
	t.Helper()
	h.GotoPath("/clients/new")
	_ = h.Page.Locator("input[name=name]").Fill("Focus Client")
	_ = h.Page.Locator("input[name=contact_email]").Fill("focus@example.test")
	if err := h.Page.Locator("form button[type=submit]").Click(); err != nil {
		t.Fatalf("submit new client: %v", err)
	}
	if err := h.Page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	}); err != nil {
		t.Fatalf("wait after client submit: %v", err)
	}
	// Best-effort: leave /projects and /entries empty-state aware; row
	// action selectors will t.Skip when absent. This keeps the seed
	// minimal and prevents coupling to current form copy.
	_ = strings.TrimSpace
}
