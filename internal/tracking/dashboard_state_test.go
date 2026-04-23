package tracking

import "testing"

// TestDashboardStateFor pins the dashboard three-state derivation. See
// openspec/specs/ui-partials/spec.md (Dashboard surface renders three
// documented states) and the archived sharpen-dashboard-and-empty-states
// design Decision 1/2.
//
// The spec is intentionally defined on the (project count, timer running)
// pair rather than an explicit "recent entries" count — entries require
// projects, so zero projects implies zero entries ever, and a running
// timer implies at least one project. This keeps the handler free of an
// extra query.
func TestDashboardStateFor(t *testing.T) {
	tests := []struct {
		name         string
		projectCount int
		timerRunning bool
		want         string
	}{
		{"fresh workspace — no projects, no timer", 0, false, "zero"},
		{"post-cleanup — projects gone, no timer", 0, false, "zero"},
		{"projects exist, no timer", 3, false, "idle"},
		{"projects exist, timer running", 3, true, "running"},
		// Defensive: a running timer with zero projects shouldn't be
		// reachable domain-wise, but the state machine MUST prefer
		// "running" over "zero" if it ever happens.
		{"running overrides zero (defensive)", 0, true, "running"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dashboardStateFor(tc.projectCount, tc.timerRunning)
			if got != tc.want {
				t.Fatalf("dashboardStateFor(%d, %v) = %q, want %q", tc.projectCount, tc.timerRunning, got, tc.want)
			}
		})
	}
}
