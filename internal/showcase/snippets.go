package showcase

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

// snippetsFS holds the copy-ready template snippets colocated with the
// showcase package. Each file's base name (minus the ".tmpl" extension)
// is the snippet id referenced from ComponentExample.SnippetID.
//
//go:embed snippets/*.tmpl
var snippetsFS embed.FS

// snippetCache is a one-shot load of every fixture so handlers don't
// stat the embedded FS per request.
var snippetCache = mustLoadSnippets()

// LookupSnippet returns the copy-ready template snippet for the given id.
// The returned string preserves trailing whitespace from the source
// fixture, minus a single trailing newline (fixtures typically end with
// a newline; we strip it so <pre><code> blocks don't have a blank line).
func LookupSnippet(id string) (string, error) {
	s, ok := snippetCache[id]
	if !ok {
		return "", fmt.Errorf("showcase: snippet %q not found", id)
	}
	return s, nil
}

// SnippetIDs returns every registered snippet id. Used by the
// snippet-integrity test.
func SnippetIDs() []string {
	ids := make([]string, 0, len(snippetCache))
	for id := range snippetCache {
		ids = append(ids, id)
	}
	return ids
}

func mustLoadSnippets() map[string]string {
	out := make(map[string]string)
	entries, err := fs.ReadDir(snippetsFS, "snippets")
	if err != nil {
		// No snippets directory is unusual but not fatal at init; the
		// snippet-integrity test will catch any ComponentExample that
		// references a missing id.
		return out
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".tmpl") {
			continue
		}
		id := strings.TrimSuffix(name, ".tmpl")
		b, err := fs.ReadFile(snippetsFS, "snippets/"+name)
		if err != nil {
			panic(fmt.Sprintf("showcase: read snippet %s: %v", name, err))
		}
		out[id] = strings.TrimRight(string(b), "\n")
	}
	return out
}
