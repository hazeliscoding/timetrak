package showcase_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"timetrak/internal/showcase"
)

// repoRoot walks up from the test cwd to locate go.mod so we can read
// web/templates/partials/ regardless of where go test is invoked.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("go.mod not found from %s", dir)
	return ""
}

// TestComponentCatalogueCoverage enumerates every .html file under
// web/templates/partials/ and asserts each non-grandfathered file stem
// appears exactly once as a ComponentEntry.PartialName.
//
// Spec: ui-showcase — "Component catalogue covers every reusable partial".
func TestComponentCatalogueCoverage(t *testing.T) {
	root := repoRoot(t)
	partialsDir := filepath.Join(root, "web", "templates", "partials")
	entries, err := os.ReadDir(partialsDir)
	if err != nil {
		t.Fatalf("read %s: %v", partialsDir, err)
	}

	grandfathered := make(map[string]bool, len(showcase.GrandfatheredPartials))
	for _, g := range showcase.GrandfatheredPartials {
		grandfathered[g] = true
	}

	// Walk partials dir; collect every file stem, skipping README.
	var partialStems []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".html") {
			continue
		}
		stem := strings.TrimSuffix(name, ".html")
		if grandfathered[stem] {
			continue
		}
		partialStems = append(partialStems, stem)
	}

	// Build a map from stem -> count across ComponentEntries. Note the
	// spec key is the file stem of the source file; a few entries (the
	// reports composites) carry dot-namespaced block names
	// (reports.partial.results / reports.partial.empty) whose source
	// files are report_results.html and report_empty.html. The coverage
	// test checks via SourcePath's basename rather than PartialName for
	// that reason.
	counts := make(map[string]int)
	entryLocations := make(map[string][]string)
	for _, ce := range showcase.ComponentEntries {
		stem := strings.TrimSuffix(filepath.Base(ce.SourcePath), ".html")
		counts[stem]++
		entryLocations[stem] = append(entryLocations[stem], ce.Name)
	}

	for _, stem := range partialStems {
		switch c := counts[stem]; {
		case c == 0:
			t.Errorf("partial %q has no ComponentEntry — add one to internal/showcase/catalogue.go or grandfather it", stem)
		case c > 1:
			t.Errorf("partial %q appears in %d ComponentEntries (%v) — must be exactly one", stem, c, entryLocations[stem])
		}
	}

	// Also assert no ComponentEntry references a SourcePath that does
	// NOT exist on disk (catches typos in catalogue.go).
	for _, ce := range showcase.ComponentEntries {
		path := filepath.Join(root, ce.SourcePath)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("ComponentEntry %q: SourcePath %q does not exist: %v", ce.Name, ce.SourcePath, err)
		}
	}
}

// TestNoProductionLinkToShowcase greps every user-facing template under
// web/templates/ EXCLUDING web/templates/showcase/ and asserts no
// reference to "/dev/showcase" appears. The showcase must never be
// linked from a production surface.
//
// Spec: ui-showcase — "Showcase is not linked from user-facing templates".
func TestNoProductionLinkToShowcase(t *testing.T) {
	root := repoRoot(t)
	tplDir := filepath.Join(root, "web", "templates")
	showcaseDir := filepath.Join(tplDir, "showcase")

	err := filepath.Walk(tplDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip the showcase dir entirely.
			if path == showcaseDir {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".html") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "/dev/showcase") {
			t.Errorf("%s: references /dev/showcase — user-facing templates must not link to the dev showcase", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
