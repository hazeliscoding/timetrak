package showcase

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// accentRule captures a CSS rule whose declarations reference an accent token.
type accentRule struct{ Selector, Decls string }

// TestAccentRationingAudit enforces the accent-rationing rule from
// openspec/specs/ui-component-identity/spec.md (Accent rationing).
//
// It parses web/static/css/app.css, enumerates every CSS rule whose
// declaration block references a --color-accent* token (or the legacy
// --accent / --accent-soft / --accent-hover aliases), and fails if any
// selector is not on the allow-list.
//
// The allow-list is the canonical enumeration from the spec. Adding a
// selector here without amending the spec is a review block.
func TestAccentRationingAudit(t *testing.T) {
	t.Parallel()

	cssPath := findRepoFile(t, filepath.Join("web", "static", "css", "app.css"))
	src, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("read css: %v", err)
	}

	// Strip /* ... */ comments so comment prose referencing "accent"
	// never reaches the scanner.
	commentRe := regexp.MustCompile(`(?s)/\*.*?\*/`)
	clean := commentRe.ReplaceAllString(string(src), "")

	// Accent token pattern — semantic alias family + grandfathered aliases.
	accentTokenRe := regexp.MustCompile(`var\(\s*--(color-accent(-soft|-hover)?|accent(-soft|-hover)?)\s*\)`)

	hits := collectAccentRules(clean, accentTokenRe)

	// Allow-list, in spec enumeration order. Selectors are whitespace-
	// normalised before comparison.
	allow := normaliseAll([]string{
		// 1. Running timer
		".tt-timer-running",
		".tt-timer-dot",
		".tt-timer-elapsed",
		// 2. Focus ring uses --color-focus (resolves to accent-600), not
		//    --color-accent directly; not matched by this test.
		// 3. Selected / focused table row
		`.table tbody tr[aria-selected="true"], .table tbody tr:focus-within`,
		// 4. Primary button
		".btn-primary",
		".btn-primary:hover",
		// 5. Link text
		"a",
		"a:hover",
		// 6. Active nav item
		`.nav a[aria-current="page"]`,
		// 7. Billable + running chips
		".tt-chip-billable",
		".tt-chip-running",
		// 8. Selected theme-switch segment — answers "which theme is active?"
		//    See ui-component-identity Accent rationing allow-list item 8.
		`.tt-theme-seg[aria-pressed="true"]`,
		`.tt-theme-seg[aria-pressed="true"]:hover`,
		// 9. Running-entry card top border — reserved for follow-on change.
	})

	var violations []string
	for _, h := range hits {
		sel := normalise(h.Selector)
		if !allow[sel] {
			violations = append(violations, h.Selector)
		}
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		t.Fatalf("accent-rationing audit failed: %d disallowed selector(s) reference a --color-accent* token:\n  %s\n\nIf the usage is legitimate, amend openspec/specs/ui-component-identity/spec.md (Accent rationing) and add the selector to the allow-list in internal/showcase/identity_audit_test.go.",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// collectAccentRules walks a CSS source (or at-rule block body) and
// returns every concrete rule whose declaration block references an
// accent token. Nested at-rules (@layer, @media, @keyframes) are
// recursed into; the at-rule itself never counts as a rule.
func collectAccentRules(block string, accentRe *regexp.Regexp) []accentRule {
	var out []accentRule
	i, n := 0, len(block)
	selStart := 0
	for i < n {
		if block[i] == '{' {
			selector := strings.TrimSpace(block[selStart:i])
			j := i + 1
			nested := 1
			for j < n && nested > 0 {
				switch block[j] {
				case '{':
					nested++
				case '}':
					nested--
				}
				j++
			}
			inner := block[i+1 : j-1]
			if isAtRule(selector) {
				out = append(out, collectAccentRules(inner, accentRe)...)
			} else if accentRe.MatchString(inner) {
				out = append(out, accentRule{Selector: selector, Decls: inner})
			}
			i = j
			selStart = j
			continue
		}
		i++
	}
	return out
}

func isAtRule(selector string) bool {
	return strings.HasPrefix(strings.TrimSpace(selector), "@")
}

var wsRe = regexp.MustCompile(`\s+`)

func normalise(s string) string {
	return wsRe.ReplaceAllString(strings.TrimSpace(s), " ")
}

func normaliseAll(ss []string) map[string]bool {
	out := make(map[string]bool, len(ss))
	for _, s := range ss {
		out[normalise(s)] = true
	}
	return out
}

// findRepoFile ascends from cwd until it finds the named relative path,
// or fails the test.
func findRepoFile(t *testing.T, rel string) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := cwd
	for range 10 {
		candidate := filepath.Join(dir, rel)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate %s starting from %s", rel, cwd)
	return ""
}
