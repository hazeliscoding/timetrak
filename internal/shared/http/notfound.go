package http

import (
	"net/http"

	"timetrak/internal/shared/templates"
)

// NotFoundRenderer renders the shared not-found template (errors.not_found).
// Every cross-workspace 404 and every "row not found" 404 in domain handlers
// MUST go through this helper so the response body is byte-identical across
// "the resource truly does not exist" and "the resource exists in another
// workspace". This prevents an information-disclosure oracle via response
// body differences.
//
// For HTMX requests, the helper sets HX-Refresh: true so the browser does
// a full page navigation and the not-found page renders in its proper
// context (instead of being swapped into a tiny target). HTMX-initiated
// mutations that fail authorization MUST NOT emit any HX-Trigger event.
type NotFoundRenderer struct {
	tpls *templates.Registry
}

// NewNotFoundRenderer wires the shared template registry into the helper.
func NewNotFoundRenderer(tpls *templates.Registry) *NotFoundRenderer {
	return &NotFoundRenderer{tpls: tpls}
}

// Render writes the shared 404 response. For HTMX requests, it returns 200
// with HX-Refresh: true (HTMX swallows non-200 by default for hx-* targets,
// so we ask the browser to reload, which then renders the full 404 page).
// For non-HTMX requests, it writes a 404 with the rendered body.
//
// This deliberately strips any pending HX-Trigger headers; any caller that
// wrote one before deciding to render not-found should not have done so.
func (n *NotFoundRenderer) Render(w http.ResponseWriter, r *http.Request) {
	// Strip any HX-Trigger that an earlier handler may have set. Denied
	// mutations MUST NOT trigger downstream refreshes.
	w.Header().Del("HX-Trigger")

	if IsHTMX(r) {
		// Tell HTMX to do a full reload; the browser will follow and the
		// non-HTMX branch below will render the actual 404 page.
		w.Header().Set("HX-Refresh", "true")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Best-effort render. If the template is missing (e.g. tests without
	// templates loaded), fall back to a plain 404.
	if n.tpls == nil {
		http.NotFound(w, r)
		return
	}
	if err := n.tpls.Render(w, http.StatusNotFound, "errors.not_found", nil); err != nil {
		http.NotFound(w, r)
	}
}

// HandlerFunc returns an http.HandlerFunc bound to this renderer, suitable
// for wiring into authz.SetNotFoundRenderer.
func (n *NotFoundRenderer) HandlerFunc() func(http.ResponseWriter, *http.Request) {
	return n.Render
}

// globalRenderer holds the process-wide renderer set by SetGlobalNotFound.
// Domain handlers call NotFound(w, r) (the package function) to render the
// shared template without each handler having to carry a *NotFoundRenderer
// reference. Defaults to http.NotFound until cmd/web wires the real one.
var globalRenderer = func(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

// SetGlobalNotFound installs the renderer used by the package-level
// NotFound helper. cmd/web calls this once at startup with the wired
// NotFoundRenderer.
func SetGlobalNotFound(fn func(http.ResponseWriter, *http.Request)) {
	if fn != nil {
		globalRenderer = fn
	}
}

// NotFound is the package-level helper that renders the shared not-found
// template. Domain handlers call this instead of http.NotFound so that
// "the row does not exist" and "the row exists in another workspace"
// produce byte-identical responses.
func NotFound(w http.ResponseWriter, r *http.Request) {
	globalRenderer(w, r)
}
