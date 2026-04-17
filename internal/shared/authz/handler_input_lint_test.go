package authz_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestHandlersDoNotReadWorkspaceIDFromInput enforces decision 1 of the
// harden-workspace-authorization-boundaries change: handler bodies in the
// covered domain families MUST NOT read `workspace_id` from form, query,
// path, or body input. Authorization scope comes exclusively from the typed
// WorkspaceContext populated by RequireWorkspace middleware.
//
// The lint walks `internal/{clients,projects,tracking,rates,reporting}`
// `handler*.go` files and fails the build if any forbidden pattern matches.
// To suppress a false positive, add a trailing `// authz:ok: <reason>` on
// the offending line.
func TestHandlersDoNotReadWorkspaceIDFromInput(t *testing.T) {
	root := repoRoot(t)
	domains := []string{"clients", "projects", "tracking", "rates", "reporting"}

	// Patterns of input-derived workspace reads that must not appear.
	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`r\.FormValue\(\s*"workspace_id"\s*\)`),
		regexp.MustCompile(`r\.PostFormValue\(\s*"workspace_id"\s*\)`),
		regexp.MustCompile(`r\.URL\.Query\(\)\.Get\(\s*"workspace_id"\s*\)`),
		regexp.MustCompile(`r\.PathValue\(\s*"workspace_id"\s*\)`),
		regexp.MustCompile(`r\.Header\.Get\(\s*"X-Workspace-Id"\s*\)`),
	}

	var findings []string
	for _, d := range domains {
		dir := filepath.Join(root, "internal", d)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasPrefix(e.Name(), "handler") || !strings.HasSuffix(e.Name(), ".go") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			lines := strings.Split(string(b), "\n")
			for i, line := range lines {
				if strings.Contains(line, "// authz:ok") {
					continue
				}
				for _, re := range forbidden {
					if re.MatchString(line) {
						findings = append(findings, formatFinding(path, i+1, line))
					}
				}
			}
		}
	}
	if len(findings) > 0 {
		t.Fatalf("handler input lint: forbidden workspace_id reads found:\n  %s\n\nHandlers MUST read workspace context from authz.MustFromContext(ctx), never from request input.\nIf this is a confirmed-safe exception, append // authz:ok: <reason> to the line.",
			strings.Join(findings, "\n  "))
	}
}

func formatFinding(path string, line int, content string) string {
	return path + ":" + itoa(line) + ": " + strings.TrimSpace(content)
}

// itoa avoids pulling strconv into a test helper.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// repoRoot walks upward from the test's working directory looking for
// go.mod, returning that directory. Tests run with cwd = the package dir,
// so we walk up from there.
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
			t.Fatalf("could not find go.mod walking up from test cwd")
		}
		dir = parent
	}
}
