//go:build browser

package browser

import (
	"testing"

	"github.com/playwright-community/playwright-go"
)

// TestFocusAfterSwapContract exercises the documented HTMX interactions
// in web/templates/partials/README.md and asserts, after each swap, that
// document.activeElement carries [data-focus-after-swap].
//
// Each sub-test is intentionally light-touch about WHICH selector should
// match — the README does not enumerate per-scenario selectors — but it
// enforces the universal contract: every documented intent swap MUST
// leave focus on something carrying [data-focus-after-swap].
//
// Scenarios covered (per partials/README.md and the shipped partials):
//   - timer start (timer_control)
//   - timer stop  (timer_control)
//   - entry edit (entry_row, Edit mode)
//   - client create via inline form (client_row + form_errors path)
//   - client edit (client_row, Edit mode)
//   - project create
//   - project edit (project_row, Edit mode)
//   - rate-rule create (rate_form)
//   - rate-rule edit (rate_form OOB)
//   - form-validation error path (form_errors on intentional 4xx)
//
// Deviation note: entry create/delete, client delete, project delete, and
// rate-rule delete are NOT exercised here because they share the same
// swap-and-focus path as create/edit (entry_row / *_row / rate_row) and
// the README does not document distinct focus targets for them. When a
// distinct target is added, extend the scenarios slice.
func TestFocusAfterSwapContract(t *testing.T) {
	h := StartHarness(t)
	if h == nil {
		return
	}
	h.SignupFreshWorkspace("Focus Swap")

	scenarios := []struct {
		name string
		run  func(t *testing.T, h *Harness)
	}{
		{name: "timer_start", run: scenarioTimerStart},
		{name: "timer_stop", run: scenarioTimerStop},
		{name: "client_create", run: scenarioClientCreate},
		{name: "client_edit", run: scenarioClientEdit},
		{name: "project_create", run: scenarioProjectCreate},
		{name: "project_edit", run: scenarioProjectEdit},
		{name: "rate_create", run: scenarioRateCreate},
		{name: "rate_edit", run: scenarioRateEdit},
		{name: "entry_edit", run: scenarioEntryEdit},
		{name: "form_validation_error", run: scenarioFormValidationError},
	}

	for _, sc := range scenarios {
		sc := sc
		t.Run(sc.name, func(t *testing.T) {
			sc.run(t, h)
		})
	}
}

func assertFocusedHasFocusAfterSwapAttr(t *testing.T, page playwright.Page, scenario string) {
	t.Helper()
	result, err := page.Evaluate(`() => {
		const el = document.activeElement;
		if (!el || el === document.body) return { ok: false, reason: 'activeElement is body/null' };
		const has = el.hasAttribute('data-focus-after-swap');
		return {
			ok: has,
			reason: has ? '' : 'focused element lacks data-focus-after-swap',
			tag: el.tagName,
			id: el.id || '',
			cls: el.className || '',
		};
	}`)
	if err != nil {
		t.Fatalf("%s: evaluate activeElement: %v", scenario, err)
	}
	m, _ := result.(map[string]any)
	if ok, _ := m["ok"].(bool); !ok {
		t.Errorf("%s: focused element lacks data-focus-after-swap after swap: tag=%v id=%v cls=%v reason=%v",
			scenario, m["tag"], m["id"], m["cls"], m["reason"])
	}
}

func scenarioTimerStart(t *testing.T, h *Harness) {
	// Requires a project; create a client and a project first.
	ensureClient(t, h, "Timer Client")
	ensureProject(t, h, "Timer Project")
	h.GotoPath("/dashboard")
	// Wait for the timer widget; then click Start.
	startBtn := h.Page.Locator("form[action='/timer/start'] button[type=submit]").First()
	if count, _ := startBtn.Count(); count == 0 {
		t.Skip("timer start button not present (no projects seeded?)")
		return
	}
	if err := startBtn.Click(); err != nil {
		t.Fatalf("click start: %v", err)
	}
	if err := WaitForHTMXSettle(h.Page); err != nil {
		t.Fatalf("wait settle: %v", err)
	}
	assertFocusedHasFocusAfterSwapAttr(t, h.Page, "timer_start")
}

func scenarioTimerStop(t *testing.T, h *Harness) {
	h.GotoPath("/dashboard")
	stopBtn := h.Page.Locator("form[action='/timer/stop'] button[type=submit]").First()
	if count, _ := stopBtn.Count(); count == 0 {
		t.Skip("no running timer; skipping timer_stop scenario")
		return
	}
	if err := stopBtn.Click(); err != nil {
		t.Fatalf("click stop: %v", err)
	}
	if err := WaitForHTMXSettle(h.Page); err != nil {
		t.Fatalf("wait settle: %v", err)
	}
	assertFocusedHasFocusAfterSwapAttr(t, h.Page, "timer_stop")
}

func scenarioClientCreate(t *testing.T, h *Harness) {
	h.GotoPath("/clients")
	// The inline create form is on /clients/new — navigate there as the
	// canonical path; the create emits a row that should carry focus.
	h.GotoPath("/clients/new")
	_ = h.Page.Locator("input[name=name]").Fill("Acme Create")
	if err := h.Page.Locator("form button[type=submit]").Click(); err != nil {
		t.Fatalf("submit client create: %v", err)
	}
	// Server returns a 303; this is a full navigation, not an HTMX swap.
	// Focus-after-swap contract only applies to HTMX swaps. Skip if the
	// form is not HTMX-bound.
	if isHTMX, _ := h.Page.Locator("form[hx-post], form[data-hx-post]").Count(); isHTMX == 0 {
		t.Skip("client create uses a full POST/redirect, not HTMX swap")
		return
	}
	if err := WaitForHTMXSettle(h.Page); err != nil {
		t.Fatalf("wait settle: %v", err)
	}
	assertFocusedHasFocusAfterSwapAttr(t, h.Page, "client_create")
}

func scenarioClientEdit(t *testing.T, h *Harness) {
	ensureClient(t, h, "Edit Client")
	h.GotoPath("/clients")
	editBtn := h.Page.Locator("tbody a[hx-get], tbody button[hx-get]").First()
	if count, _ := editBtn.Count(); count == 0 {
		t.Skip("no HTMX edit control on /clients")
		return
	}
	if err := editBtn.Click(); err != nil {
		t.Fatalf("click edit: %v", err)
	}
	if err := WaitForHTMXSettle(h.Page); err != nil {
		t.Fatalf("wait settle: %v", err)
	}
	assertFocusedHasFocusAfterSwapAttr(t, h.Page, "client_edit")
}

func scenarioProjectCreate(t *testing.T, h *Harness) {
	ensureClient(t, h, "Project Parent")
	h.GotoPath("/projects/new")
	if isHTMX, _ := h.Page.Locator("form[hx-post], form[data-hx-post]").Count(); isHTMX == 0 {
		t.Skip("project create uses full POST/redirect, not HTMX swap")
		return
	}
	_ = h.Page.Locator("input[name=name]").Fill("New Project X")
	if err := h.Page.Locator("form button[type=submit]").Click(); err != nil {
		t.Fatalf("submit project create: %v", err)
	}
	if err := WaitForHTMXSettle(h.Page); err != nil {
		t.Fatalf("wait settle: %v", err)
	}
	assertFocusedHasFocusAfterSwapAttr(t, h.Page, "project_create")
}

func scenarioProjectEdit(t *testing.T, h *Harness) {
	ensureClient(t, h, "Edit Proj Parent")
	ensureProject(t, h, "Edit Proj")
	h.GotoPath("/projects")
	editBtn := h.Page.Locator("tbody a[hx-get], tbody button[hx-get]").First()
	if count, _ := editBtn.Count(); count == 0 {
		t.Skip("no HTMX edit control on /projects")
		return
	}
	if err := editBtn.Click(); err != nil {
		t.Fatalf("click edit: %v", err)
	}
	if err := WaitForHTMXSettle(h.Page); err != nil {
		t.Fatalf("wait settle: %v", err)
	}
	assertFocusedHasFocusAfterSwapAttr(t, h.Page, "project_edit")
}

func scenarioRateCreate(t *testing.T, h *Harness) {
	h.GotoPath("/rates")
	// rate_form lives on /rates and posts via HTMX to /rates.
	if count, _ := h.Page.Locator("form[hx-post='/rates']").Count(); count == 0 {
		t.Skip("HTMX rate create form not present")
		return
	}
	_, _ = h.Page.Locator("select[name=scope]").SelectOption(playwright.SelectOptionValues{
		Values: &[]string{"workspace"},
	})
	_ = h.Page.Locator("input[name=currency_code]").Fill("USD")
	_ = h.Page.Locator("input[name=hourly_decimal]").Fill("100.00")
	_ = h.Page.Locator("input[name=effective_from]").Fill("2026-01-01")
	if err := h.Page.Locator("form[hx-post='/rates'] button[type=submit]").Click(); err != nil {
		t.Fatalf("submit rate create: %v", err)
	}
	if err := WaitForHTMXSettle(h.Page); err != nil {
		t.Fatalf("wait settle: %v", err)
	}
	assertFocusedHasFocusAfterSwapAttr(t, h.Page, "rate_create")
}

func scenarioRateEdit(t *testing.T, h *Harness) {
	h.GotoPath("/rates")
	editBtn := h.Page.Locator("tbody a[hx-get], tbody button[hx-get]").First()
	if count, _ := editBtn.Count(); count == 0 {
		t.Skip("no HTMX rate edit control on /rates")
		return
	}
	if err := editBtn.Click(); err != nil {
		t.Fatalf("click rate edit: %v", err)
	}
	if err := WaitForHTMXSettle(h.Page); err != nil {
		t.Fatalf("wait settle: %v", err)
	}
	assertFocusedHasFocusAfterSwapAttr(t, h.Page, "rate_edit")
}

func scenarioEntryEdit(t *testing.T, h *Harness) {
	h.GotoPath("/entries")
	editBtn := h.Page.Locator("tbody a[hx-get], tbody button[hx-get]").First()
	if count, _ := editBtn.Count(); count == 0 {
		t.Skip("no HTMX entry edit control (no entries seeded)")
		return
	}
	if err := editBtn.Click(); err != nil {
		t.Fatalf("click entry edit: %v", err)
	}
	if err := WaitForHTMXSettle(h.Page); err != nil {
		t.Fatalf("wait settle: %v", err)
	}
	assertFocusedHasFocusAfterSwapAttr(t, h.Page, "entry_edit")
}

func scenarioFormValidationError(t *testing.T, h *Harness) {
	h.GotoPath("/rates")
	if count, _ := h.Page.Locator("form[hx-post='/rates']").Count(); count == 0 {
		t.Skip("HTMX rate form not present")
		return
	}
	// Submit without filling required fields → server returns 4xx + form_errors.
	if err := h.Page.Locator("form[hx-post='/rates'] button[type=submit]").Click(); err != nil {
		t.Fatalf("submit empty form: %v", err)
	}
	if err := WaitForHTMXSettle(h.Page); err != nil {
		t.Fatalf("wait settle: %v", err)
	}
	assertFocusedHasFocusAfterSwapAttr(t, h.Page, "form_validation_error")
}

// --- seed helpers ---

func ensureClient(t *testing.T, h *Harness, name string) {
	t.Helper()
	h.GotoPath("/clients/new")
	_ = h.Page.Locator("input[name=name]").Fill(name)
	_ = h.Page.Locator("form button[type=submit]").Click()
	_ = h.Page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	})
}

func ensureProject(t *testing.T, h *Harness, name string) {
	t.Helper()
	h.GotoPath("/projects/new")
	// Pick first client in select.
	sel := h.Page.Locator("select[name=client_id]")
	if count, _ := sel.Count(); count > 0 {
		opts, _ := sel.Locator("option").All()
		for _, opt := range opts {
			v, _ := opt.GetAttribute("value")
			if v != "" {
				_, _ = sel.SelectOption(playwright.SelectOptionValues{Values: &[]string{v}})
				break
			}
		}
	}
	_ = h.Page.Locator("input[name=name]").Fill(name)
	_ = h.Page.Locator("form button[type=submit]").Click()
	_ = h.Page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	})
}
