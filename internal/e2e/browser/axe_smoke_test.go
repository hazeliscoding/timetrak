//go:build browser

package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/playwright-community/playwright-go"
)

// TestAxeSmokePerPage runs axe-core against every top-level page and
// fails when any violation has impact `serious` or `critical`. `moderate`
// and `minor` violations are logged to t.Logf and dumped to the per-test
// artifacts directory for triage.
//
// Tags: wcag2a, wcag2aa, wcag22aa. See spec §6.4.
func TestAxeSmokePerPage(t *testing.T) {
	h := StartHarness(t)
	if h == nil {
		return
	}
	// Sign up once; some pages require auth. login / signup pages render
	// even while authenticated at their public routes.
	h.SignupFreshWorkspace("Axe Tester")

	axePath := filepath.Join(h.RepoRoot, "internal", "e2e", "browser", "testdata", "axe.min.js")
	if _, err := os.Stat(axePath); err != nil {
		t.Fatalf("axe bundle missing at %s: %v", axePath, err)
	}

	pages := []struct {
		name string
		path string
	}{
		{"login", "/login"},
		{"signup", "/signup"},
		{"dashboard", "/dashboard"},
		{"entries", "/time"},
		{"clients", "/clients"},
		{"projects", "/projects"},
		{"rates", "/rates"},
		{"reports", "/reports"},
		{"settings", "/workspace/settings"},
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
			// Serialize to JSON for structured parsing + artifact dump.
			buf, err := json.Marshal(raw)
			if err != nil {
				t.Fatalf("marshal axe result: %v", err)
			}
			var result axeResult
			if err := json.Unmarshal(buf, &result); err != nil {
				t.Fatalf("unmarshal axe result: %v", err)
			}

			var blockers []axeViolation
			for _, v := range result.Violations {
				switch v.Impact {
				case "serious", "critical":
					blockers = append(blockers, v)
				default:
					t.Logf("axe[%s] %s (impact=%s) — %d nodes: %s",
						p.name, v.ID, v.Impact, len(v.Nodes), v.Help)
				}
			}

			if len(blockers) > 0 {
				// Dump the full axe result alongside artifacts.
				artifact := h.ArtifactPath(fmt.Sprintf("axe-%s.json", p.name))
				_ = os.WriteFile(artifact, buf, 0o644)
				for _, v := range blockers {
					selectors := make([]string, 0, len(v.Nodes))
					for _, n := range v.Nodes {
						selectors = append(selectors, fmt.Sprintf("%v", n.Target))
					}
					t.Errorf("axe[%s] impact=%s rule=%s help=%q selectors=%v (artifact=%s)",
						p.name, v.Impact, v.ID, v.Help, selectors, artifact)
				}
			}
		})
	}
}

type axeResult struct {
	Violations []axeViolation `json:"violations"`
}

type axeViolation struct {
	ID     string    `json:"id"`
	Impact string    `json:"impact"`
	Help   string    `json:"help"`
	Nodes  []axeNode `json:"nodes"`
}

type axeNode struct {
	Target any `json:"target"`
}
