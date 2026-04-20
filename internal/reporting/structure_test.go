package reporting_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReportingDoesNotImportRates is the structural snapshot-only
// invariant: no file under internal/reporting may import
// timetrak/internal/rates. The reporting read path is scored entirely from
// rate snapshots persisted on time_entries; a future `import "…/rates"`
// would reintroduce retroactive-edit drift in historical totals.
//
// This complements the runtime snapshot test by failing at `go test` time
// if a contributor adds the import, even before any code path exercises it.
func TestReportingDoesNotImportRates(t *testing.T) {
	dir := packageDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	fset := token.NewFileSet()
	forbidden := `"timetrak/internal/rates"`
	checked := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		// Tests are allowed to import rates (the snapshot_test.go does).
		if strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, imp := range file.Imports {
			if imp.Path.Value == forbidden {
				t.Errorf("%s imports %s — reporting read path MUST remain snapshot-only", e.Name(), forbidden)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no .go files found under internal/reporting — refusing to claim coverage")
	}
}

// packageDir returns the absolute path to internal/reporting from the test
// cwd. Integration tests run from the package dir, but handle the general
// case by walking up to go.mod.
func packageDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// If we're already inside the package, return it.
	if strings.HasSuffix(filepath.ToSlash(dir), "internal/reporting") {
		return dir
	}
	// Walk up to find go.mod, then append internal/reporting.
	cur := dir
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return filepath.Join(cur, "internal", "reporting")
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			t.Fatalf("could not find go.mod from %s", dir)
		}
		cur = parent
	}
}
