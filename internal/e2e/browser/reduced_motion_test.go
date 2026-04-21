//go:build browser

package browser

import (
	"fmt"
	"testing"

	"github.com/playwright-community/playwright-go"
)

// TestReducedMotionContract emulates prefers-reduced-motion: reduce and
// asserts that elements which would otherwise transition collapse to 0s.
//
// The css README documents the approved exception: the
// @media (prefers-reduced-motion: reduce) rule lives outside @layer and
// carries !important on transition/animation. If that rule ever disappears
// or a new transition lands without honoring the preference, this test
// fails.
func TestReducedMotionContract(t *testing.T) {
	h := StartHarness(t)
	if h == nil {
		return
	}
	h.SignupFreshWorkspace("Reduce Motion")

	// Emulate BEFORE the first navigation so the initial layout applies
	// the preference.
	if err := h.Page.EmulateMedia(playwright.PageEmulateMediaOptions{
		ReducedMotion: playwright.ReducedMotionReduce,
	}); err != nil {
		t.Fatalf("emulate reduced-motion: %v", err)
	}

	type target struct {
		name     string
		path     string
		selector string
	}
	targets := []target{
		// Buttons rely on transition for hover feedback; under reduce they
		// must collapse to 0s.
		{name: "primary button", path: "/dashboard", selector: ".btn-primary"},
		{name: "nav anchor", path: "/dashboard", selector: "nav a"},
		{name: "flash card (if present)", path: "/dashboard", selector: ".flash"},
		{name: "card", path: "/clients", selector: ".card"},
		{name: "field input", path: "/clients/new", selector: "input[type=text], input:not([type])"},
	}

	for _, tg := range targets {
		tg := tg
		t.Run(tg.name, func(t *testing.T) {
			h.GotoPath(tg.path)
			loc := h.Page.Locator(tg.selector).First()
			count, _ := loc.Count()
			if count == 0 {
				t.Skipf("selector %q not present on %s", tg.selector, tg.path)
				return
			}
			result, err := loc.Evaluate(`el => {
				const s = getComputedStyle(el);
				return {
					transitionDuration: s.transitionDuration,
					animationDuration: s.animationDuration,
				};
			}`, nil)
			if err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			m, ok := result.(map[string]any)
			if !ok {
				t.Fatalf("unexpected result shape: %T", result)
			}
			if !allZeroDurations(fmt.Sprintf("%v", m["transitionDuration"])) {
				t.Errorf("transition-duration under reduce: got %v want all 0s (name=%s)",
					m["transitionDuration"], tg.name)
			}
			if !allZeroDurations(fmt.Sprintf("%v", m["animationDuration"])) {
				t.Errorf("animation-duration under reduce: got %v want all 0s (name=%s)",
					m["animationDuration"], tg.name)
			}
		})
	}
}

// allZeroDurations returns true when every comma-separated duration in the
// computed style value resolves to zero. Computed values can be "0s" or
// "0s, 0s, 0s" for multiple transitions.
func allZeroDurations(v string) bool {
	if v == "" {
		return false
	}
	seen := 0
	for i := 0; i < len(v); {
		// Walk comma-separated parts.
		j := i
		for j < len(v) && v[j] != ',' {
			j++
		}
		part := trimSpaces(v[i:j])
		if part != "0s" && part != "0ms" {
			return false
		}
		seen++
		i = j + 1
	}
	return seen > 0
}

func trimSpaces(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
