package showcase_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"timetrak/internal/shared/templates"
	"timetrak/internal/showcase"
)

// loadTemplates loads the real template registry used by the app,
// rooted at <repo>/web/templates. Shared helper for the integrity
// tests in this file.
func loadTemplates(t *testing.T) *templates.Registry {
	t.Helper()
	root := repoRoot(t)
	reg, err := templates.Load(os.DirFS(filepath.Join(root, "web", "templates")))
	if err != nil {
		t.Fatalf("load templates: %v", err)
	}
	return reg
}

// TestSnippetIntegrity asserts every ComponentEntry.PartialName and
// every Example.PartialName resolves against the live template loader,
// AND that every Example renders without error with its Dict payload.
//
// This is the drift-detector: if a partial's dict contract changes and
// the catalogue fixture is not updated, this test fails naming the
// partial and the missing key.
//
// Spec: ui-showcase — "Copy-ready snippets are colocated with live
// examples" + "Showcase renders real partials, never re-implementations".
func TestSnippetIntegrity(t *testing.T) {
	reg := loadTemplates(t)

	for _, entry := range showcase.ComponentEntries {
		entry := entry
		t.Run(entry.Name, func(t *testing.T) {
			// Entry-level PartialName MUST resolve.
			var buf bytes.Buffer
			if err := reg.RenderPartialTo(&buf, "showcase.components", entry.PartialName, map[string]any{}); err != nil {
				// An error here distinguishes "block not defined" from
				// "block exists but missing dict keys". The former is
				// caught by the "no such template" message; we match
				// that specifically.
				if strings.Contains(err.Error(), "no such template") {
					t.Fatalf("PartialName %q does not resolve against template loader: %v", entry.PartialName, err)
				}
				// Partial resolved but empty dict failed — that's OK
				// for this entry-level check; each Example has its own
				// render below.
			}

			if len(entry.Examples) == 0 {
				t.Fatalf("entry %q has no Examples — every ComponentEntry MUST ship at least one", entry.Name)
			}

			for _, ex := range entry.Examples {
				ex := ex
				t.Run(ex.ID, func(t *testing.T) {
					// 1. Snippet file must exist.
					if _, err := showcase.LookupSnippet(ex.SnippetID); err != nil {
						t.Errorf("snippet %q for entry %s/%s not found: %v", ex.SnippetID, entry.Name, ex.ID, err)
					}
					// 2. Example must render through the live loader.
					var buf bytes.Buffer
					if err := reg.RenderPartialTo(&buf, "showcase.components", ex.PartialName, ex.Dict); err != nil {
						t.Errorf("example %s/%s failed to render partial %q: %v", entry.Name, ex.ID, ex.PartialName, err)
					}
					if buf.Len() == 0 {
						t.Errorf("example %s/%s rendered empty output (partial=%q)", entry.Name, ex.ID, ex.PartialName)
					}
					// 3. A11y labelling assertion for §8.2 — every
					// example has a visible Label (not empty).
					if strings.TrimSpace(ex.Label) == "" {
						t.Errorf("example %s/%s has an empty Label; visible labels are required for every permutation", entry.Name, ex.ID)
					}
				})
			}

			// 4. Every ComponentEntry must populate a visible Name used
			// as the heading in the catalogue. Empty names would render
			// a nameless section — breaks the §8 a11y contract.
			if strings.TrimSpace(entry.Name) == "" {
				t.Errorf("entry has empty Name (PartialName=%q)", entry.PartialName)
			}
		})
	}
}

// TestTokenCatalogueLabels asserts every TokenEntry carries a visible
// Name + Role field so color-only conveyance is impossible on the
// tokens page. Part of the §8 a11y contract surfaced by tests.
func TestTokenCatalogueLabels(t *testing.T) {
	for _, tok := range showcase.TokenEntries {
		if strings.TrimSpace(tok.Name) == "" {
			t.Errorf("TokenEntry ID=%q has empty Name", tok.ID)
		}
		if strings.TrimSpace(tok.Role) == "" {
			t.Errorf("TokenEntry %q has empty Role — color / sample alone must never convey meaning", tok.Name)
		}
	}
}

// TestSnippetFilesHaveRegisteredID asserts every snippet fixture on
// disk is referenced by at least one ComponentExample.SnippetID. Catches
// orphaned fixtures left behind when an example is removed.
func TestSnippetFilesHaveRegisteredID(t *testing.T) {
	referenced := make(map[string]bool)
	for _, entry := range showcase.ComponentEntries {
		for _, ex := range entry.Examples {
			referenced[ex.SnippetID] = true
		}
	}
	for _, id := range showcase.SnippetIDs() {
		if !referenced[id] {
			t.Errorf("snippet fixture %q is not referenced by any ComponentExample", id)
		}
	}
}
