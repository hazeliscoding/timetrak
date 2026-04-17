package http_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestAuthzRouteCoverage enumerates every route registered by the covered
// handler families (clients, projects, tracking, rates, reporting) by
// statically reading their handler.go source for `mux.Handle("METHOD /path", ...)`
// calls, and asserts the corresponding domain `authz_test.go` mentions
// each route's path template.
//
// This makes it impossible to register a new route in a covered family
// without adding a corresponding cross-workspace denial test row.
//
// The check is intentionally textual: it confirms each route path
// substring appears somewhere in the domain's authz_test.go. False
// positives are vanishingly rare given seeded UUIDs and unique path
// fragments, and a false positive is preferable to silently letting a
// new route slip past.
func TestAuthzRouteCoverage(t *testing.T) {
	root := repoRoot(t)
	domains := []struct {
		name        string
		handlerFile string
		testFile    string
		// optional allowlist: route paths to skip (e.g. /reports has only
		// one GET route; the authz test exercises it via different
		// query-string variations rather than path variation).
		allowlist []string
	}{
		{"clients", "internal/clients/handler.go", "internal/clients/authz_test.go", nil},
		{"projects", "internal/projects/handler.go", "internal/projects/authz_test.go", nil},
		{"rates", "internal/rates/handler.go", "internal/rates/authz_test.go", nil},
		{"reporting", "internal/reporting/handler.go", "internal/reporting/authz_test.go", nil},
		{"tracking", "internal/tracking/handler.go", "internal/tracking/authz_test.go", nil},
	}
	// `mux.Handle("METHOD /path/template", protect(...))`
	re := regexp.MustCompile(`mux\.Handle\(\s*"([A-Z]+\s+/[^"]*)"`)
	for _, d := range domains {
		d := d
		t.Run(d.name, func(t *testing.T) {
			handlerSrc, err := os.ReadFile(filepath.Join(root, d.handlerFile))
			if err != nil {
				t.Fatalf("read %s: %v", d.handlerFile, err)
			}
			testSrc, err := os.ReadFile(filepath.Join(root, d.testFile))
			if err != nil {
				t.Fatalf("read %s: %v (every covered domain MUST ship an authz_test.go)", d.testFile, err)
			}
			matches := re.FindAllStringSubmatch(string(handlerSrc), -1)
			if len(matches) == 0 {
				t.Fatalf("no mux.Handle calls found in %s — adjust the regex", d.handlerFile)
			}
			testText := string(testSrc)
			for _, m := range matches {
				route := m[1] // e.g. `GET /clients/{id}/row`
				if inAllowlist(route, d.allowlist) {
					continue
				}
				if !routeMentionedIn(testText, route) {
					t.Errorf("%s: route %q is registered but not exercised by %s.\n  Add a cross-workspace denial scenario for it.",
						d.name, route, d.testFile)
				}
			}
		})
	}
}

func inAllowlist(route string, allow []string) bool {
	for _, a := range allow {
		if a == route {
			return true
		}
	}
	return false
}

// routeMentionedIn returns true when the test source mentions the static
// portion of a route path. For path templates with a `{...}` placeholder
// (e.g. `/clients/{id}/row`), both the prefix and suffix MUST appear on
// the same line so we don't get spurious matches.
func routeMentionedIn(src, route string) bool {
	parts := strings.SplitN(route, " ", 2)
	if len(parts) != 2 {
		return false
	}
	path := parts[1]
	open := strings.Index(path, "{")
	if open < 0 {
		return strings.Contains(src, path)
	}
	closeIdx := strings.Index(path, "}")
	if closeIdx < open {
		return false
	}
	prefix := path[:open]
	suffix := path[closeIdx+1:]
	for _, line := range strings.Split(src, "\n") {
		if (prefix == "" || strings.Contains(line, prefix)) &&
			(suffix == "" || strings.Contains(line, suffix)) {
			return true
		}
	}
	return false
}

// repoRoot walks up from the test cwd to find go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.mod walking up from %s", dir)
		}
		dir = parent
	}
}
