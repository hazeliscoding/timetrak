// Package templates loads all HTML templates at startup and exposes a
// Render helper that writes a named page (composed with the shared layouts
// and partials catalog).
package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// Registry holds every parsed template in the app.
type Registry struct {
	pages    map[string]*template.Template // keyed by page name, e.g. "dashboard"
	funcs    template.FuncMap
	baseGlob []string
}

// Config controls template loading.
type Config struct {
	// Root is the filesystem root containing layouts/, partials/, and per-domain dirs.
	Root fs.FS
}

// Load parses layouts, partials, and every page template found under Root.
// Page templates live alongside their domain directory (e.g. `clients/index.html`).
func Load(root fs.FS) (*Registry, error) {
	funcs := baseFuncs()

	// Collect layouts and partials — included with every page.
	layoutFiles, err := collect(root, "layouts")
	if err != nil {
		return nil, err
	}
	partialFiles, err := collect(root, "partials")
	if err != nil {
		return nil, err
	}

	// Pages are every .html under root that is NOT a layout or partial.
	var pageFiles []string
	err = fs.WalkDir(root, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".html") {
			return nil
		}
		if strings.HasPrefix(path, "layouts/") || strings.HasPrefix(path, "partials/") {
			return nil
		}
		pageFiles = append(pageFiles, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	reg := &Registry{
		pages: make(map[string]*template.Template, len(pageFiles)),
		funcs: funcs,
	}
	for _, pf := range pageFiles {
		name := pageName(pf)
		files := append(append([]string{}, layoutFiles...), partialFiles...)
		files = append(files, pf)
		t, err := parseFS(root, funcs, files...)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", pf, err)
		}
		reg.pages[name] = t
	}
	return reg, nil
}

// Render writes the named page template using "base" as the outermost definition.
// The data struct should embed the layout fields the app shell needs
// (CurrentUser, ActivePage, Workspaces, ActiveWorkspaceID, CSRFToken, Flash).
func (r *Registry) Render(w http.ResponseWriter, status int, name string, data any) error {
	t, ok := r.pages[name]
	if !ok {
		return fmt.Errorf("template: page %q not registered", name)
	}
	// Buffer first so a render error can still set a 500 cleanly.
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "base", data); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := io.Copy(w, &buf)
	return err
}

// RenderPartial writes a single defined template by name (for HTMX swaps).
func (r *Registry) RenderPartial(w http.ResponseWriter, status int, page, partial string, data any) error {
	t, ok := r.pages[page]
	if !ok {
		// Fall back to the first page that has this partial (partials are defined
		// identically across all parsed trees, so any registered page works).
		for _, any := range r.pages {
			t = any
			break
		}
		if t == nil {
			return fmt.Errorf("template: no templates loaded")
		}
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, partial, data); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := io.Copy(w, &buf)
	return err
}

// RenderPartialTo writes a single named partial to an io.Writer without
// setting HTTP headers. Use this to compose multiple partials (e.g. the
// main swap target plus one or more `hx-swap-oob` partials) into a single
// HTTP response body.
func (r *Registry) RenderPartialTo(w io.Writer, page, partial string, data any) error {
	t, ok := r.pages[page]
	if !ok {
		for _, any := range r.pages {
			t = any
			break
		}
		if t == nil {
			return fmt.Errorf("template: no templates loaded")
		}
	}
	return t.ExecuteTemplate(w, partial, data)
}

// collect returns all .html files directly under a subdir.
func collect(root fs.FS, dir string) ([]string, error) {
	var files []string
	entries, err := fs.ReadDir(root, dir)
	if err != nil {
		// Missing subdir is fine (e.g. in tests).
		return nil, nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
			continue
		}
		files = append(files, filepath.ToSlash(filepath.Join(dir, e.Name())))
	}
	return files, nil
}

// pageName turns "clients/index.html" into "clients.index".
func pageName(path string) string {
	name := strings.TrimSuffix(path, ".html")
	name = strings.ReplaceAll(name, "/", ".")
	return name
}

// parseFS parses the given files out of an fs.FS into a single template set.
func parseFS(root fs.FS, funcs template.FuncMap, files ...string) (*template.Template, error) {
	t := template.New("").Funcs(funcs)
	for _, f := range files {
		b, err := fs.ReadFile(root, f)
		if err != nil {
			return nil, err
		}
		if _, err := t.Parse(string(b)); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
	}
	return t, nil
}

// baseFuncs are helpers available in every template.
func baseFuncs() template.FuncMap {
	return template.FuncMap{
		"formatDate": func(t time.Time) string { return t.Format("2006-01-02") },
		"formatTime": func(t time.Time) string { return t.Format("15:04") },
		// formatLocalDate / formatLocalTime convert a UTC time.Time to
		// the named IANA timezone before formatting, so entry-edit
		// forms can prefill split date+time inputs in the workspace's
		// reporting timezone. Empty or unknown tz falls back to UTC so
		// the function never panics in templates. See
		// openspec/specs/tracking/spec.md (Datetime input parse and
		// display is workspace-timezone-aware).
		"formatLocalDate": func(t time.Time, tz string) string {
			return toLocation(t, tz).Format("2006-01-02")
		},
		"formatLocalTime": func(t time.Time, tz string) string {
			return toLocation(t, tz).Format("15:04")
		},
		"formatDuration": func(seconds int64) string {
			if seconds < 0 {
				seconds = 0
			}
			h := seconds / 3600
			m := (seconds % 3600) / 60
			return fmt.Sprintf("%d:%02d", h, m)
		},
		"iso": func(t time.Time) string { return t.UTC().Format(time.RFC3339) },
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict: odd argument count")
			}
			m := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				k, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict: keys must be strings")
				}
				m[k] = values[i+1]
			}
			return m, nil
		},
		"seq": func(start, end int) []int {
			if end < start {
				return nil
			}
			out := make([]int, 0, end-start+1)
			for i := start; i <= end; i++ {
				out = append(out, i)
			}
			return out
		},
		"formatMinor": func(minor int64, currency string) string {
			// Simple display: <whole>.<fraction> <code> with 2 digits for common currencies.
			neg := minor < 0
			if neg {
				minor = -minor
			}
			whole := minor / 100
			frac := minor % 100
			sign := ""
			if neg {
				sign = "-"
			}
			return fmt.Sprintf("%s%d.%02d %s", sign, whole, frac, currency)
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"cssVar": func(name string) template.CSS {
			return template.CSS("var(" + name + ")")
		},
	}
}

// toLocation converts a time.Time into the named IANA timezone for
// display. Empty or unknown tz falls back to UTC so a broken workspace
// config cannot panic a render. Consumed by formatLocalDate /
// formatLocalTime.
func toLocation(t time.Time, tz string) time.Time {
	if tz == "" {
		return t.UTC()
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return t.UTC()
	}
	return t.In(loc)
}
