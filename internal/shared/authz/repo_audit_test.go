package authz_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRepositoryAudit walks every Go source file under
// internal/{auth,clients,projects,tracking,rates,reporting,workspace}
// and inspects each function or method that accepts a parameter named
// `workspaceID` of type `uuid.UUID`. For every SQL string literal that
// appears in such a function body, the audit asserts the literal contains
// the substring `workspace_id`.
//
// To suppress a confirmed-safe exception, place an inline comment
// `// authz:ok: <reason>` on the same line as the SQL literal. The
// comment MUST include a non-empty reason after the colon.
//
// This is the canonical workspace-scope enforcement check. It runs as
// part of `make test` because the repository SQL strings are hand-written
// and a missing WHERE clause is the #1 way a workspace boundary leaks.
func TestRepositoryAudit(t *testing.T) {
	root := repoRoot(t)
	dirs := []string{
		"internal/auth",
		"internal/clients",
		"internal/projects",
		"internal/tracking",
		"internal/rates",
		"internal/reporting",
		"internal/workspace",
	}
	var findings []string
	for _, d := range dirs {
		full := filepath.Join(root, d)
		entries, err := os.ReadDir(full)
		if err != nil {
			t.Fatalf("read %s: %v", full, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
				continue
			}
			if strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			findings = append(findings, auditFile(t, filepath.Join(full, e.Name()))...)
		}
	}
	if len(findings) > 0 {
		t.Fatalf("repository audit: %d finding(s):\n  %s\n\nEvery query inside a function that accepts workspaceID MUST constrain by workspace_id.\nIf a query is genuinely safe to omit the predicate (e.g. a session lookup), append // authz:ok: <reason> to the SQL literal line.",
			len(findings), strings.Join(findings, "\n  "))
	}
}

// auditFile parses one Go source file and returns audit findings.
func auditFile(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
		return nil
	}
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
		return nil
	}
	srcLines := strings.Split(string(src), "\n")
	var findings []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Type.Params == nil {
			continue
		}
		if !acceptsWorkspaceID(fn.Type.Params) {
			continue
		}
		// Walk the function body and check every basic-literal string for SQL-ness.
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			val := strings.Trim(lit.Value, "`\"")
			if !looksLikeSQL(val) {
				return true
			}
			if strings.Contains(val, "workspace_id") {
				return true
			}
			pos := fset.Position(lit.Pos())
			// Allowlist: same-line `// authz:ok: <reason>` comment.
			if hasAuthzOK(srcLines, pos.Line) {
				return true
			}
			findings = append(findings, formatRepoFinding(pos.Filename, pos.Line, fn.Name.Name, val))
			return true
		})
	}
	return findings
}

func acceptsWorkspaceID(params *ast.FieldList) bool {
	for _, field := range params.List {
		for _, name := range field.Names {
			if name.Name != "workspaceID" {
				continue
			}
			// Type should be uuid.UUID.
			if sel, ok := field.Type.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "uuid" && sel.Sel.Name == "UUID" {
					return true
				}
			}
		}
	}
	return false
}

// looksLikeSQL is a heuristic. We check for SQL keywords commonly issued
// from this codebase's repos. False positives are tolerated because the
// audit only fails when the literal also OMITS workspace_id.
func looksLikeSQL(s string) bool {
	upper := strings.ToUpper(s)
	keywords := []string{"SELECT ", "INSERT INTO", "UPDATE ", "DELETE FROM"}
	for _, k := range keywords {
		if strings.Contains(upper, k) {
			return true
		}
	}
	return false
}

func hasAuthzOK(lines []string, line int) bool {
	if line <= 0 || line > len(lines) {
		return false
	}
	// A multi-line raw string literal may span many lines; check the
	// closing line and a few preceding for the marker.
	for offset := 0; offset < 5 && line-offset > 0; offset++ {
		l := lines[line-offset-1]
		idx := strings.Index(l, "// authz:ok:")
		if idx < 0 {
			continue
		}
		reason := strings.TrimSpace(l[idx+len("// authz:ok:"):])
		if reason == "" {
			continue
		}
		return true
	}
	return false
}

func formatRepoFinding(file string, line int, fn, sql string) string {
	// Trim multi-line SQL preview to one logical line.
	preview := strings.Join(strings.Fields(sql), " ")
	if len(preview) > 120 {
		preview = preview[:117] + "..."
	}
	return file + ":" + itoa(line) + ": " + fn + ": " + preview
}
