package showcase

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"timetrak/internal/shared/authz"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/templates"
	"timetrak/internal/web/layout"
)

// Handler serves /dev/showcase and its sub-routes.
//
// Every handler also short-circuits to the shared not-found renderer
// when APP_ENV is not "dev" (belt-and-suspenders: registration in
// cmd/web/main.go is ALREADY gated by IsDev; the runtime check here
// catches the case where registration regresses but the route is
// somehow still reachable).
type Handler struct {
	tpls   *templates.Registry
	lay    *layout.Builder
	appEnv string
}

// NewHandler constructs the showcase handler.
//
// Dev-only: callers MUST guard the call to Register with IsDev(appEnv)
// so that in non-dev environments the route is not registered at all.
// The handler independently re-checks appEnv at request time.
func NewHandler(tpls *templates.Registry, lay *layout.Builder, appEnv string) *Handler {
	return &Handler{tpls: tpls, lay: lay, appEnv: appEnv}
}

// IsDev reports whether the given APP_ENV value enables the showcase.
// The single definition of "dev" for this surface — matches the
// existing convention in cmd/web/main.go.
func IsDev(appEnv string) bool {
	return strings.TrimSpace(appEnv) == "dev"
}

// Register mounts the showcase routes behind auth only (not workspace).
// The showcase is the one authenticated surface in TimeTrak that does
// NOT require workspace scoping — a freshly signed-up user with no
// workspace can still view it.
//
// CALLER CONTRACT: only call Register when IsDev(appEnv) is true.
// cmd/web/main.go wraps this call in an IsDev gate.
func (h *Handler) Register(mux *http.ServeMux) {
	// authz.RequireAuth only — no RequireWorkspace. Documented in the
	// ui-showcase spec: the showcase is the single authenticated-without-
	// workspace surface in the app.
	wrap := func(fn http.HandlerFunc) http.Handler {
		return authz.RequireAuth(http.HandlerFunc(fn))
	}
	mux.Handle("GET /dev/showcase", wrap(h.index))
	mux.Handle("GET /dev/showcase/tokens", wrap(h.tokens))
	mux.Handle("GET /dev/showcase/components", wrap(h.components))
	mux.Handle("GET /dev/showcase/dashboard-states", wrap(h.dashboardStates))
	mux.Handle("GET /dev/showcase/empty-states", wrap(h.emptyStates))
}

// view is the common base every showcase page composes.
type indexView struct {
	layout.BaseView
	ComponentEntries []ComponentEntry
	TokenEntries     []TokenEntry
}

type tokensView struct {
	layout.BaseView
	SemanticColors []TokenEntry
	ScaleTokens    []TokenEntry
	PrimitiveRamps []TokenEntry
}

type componentsView struct {
	layout.BaseView
	Entries []renderedComponentEntry
}

// renderedComponentEntry mirrors ComponentEntry but with Examples
// already rendered to template.HTML. That render happens through the
// real template loader — the showcase never re-implements markup.
type renderedComponentEntry struct {
	ComponentEntry
	RenderedExamples []renderedExample
}

type renderedExample struct {
	ComponentExample
	HTML    template.HTML
	Snippet string
}

func (h *Handler) devOnly(w http.ResponseWriter, r *http.Request) bool {
	if !IsDev(h.appEnv) {
		// Belt-and-suspenders 404 — the route SHOULD NOT be registered
		// outside dev, but if it somehow is, deny via the same shared
		// not-found renderer used for cross-workspace denial.
		sharedhttp.NotFound(w, r)
		return false
	}
	return true
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	if !h.devOnly(w, r) {
		return
	}
	base, _ := h.lay.Base(r, "dev-showcase")
	_ = h.tpls.Render(w, http.StatusOK, "showcase.index", indexView{
		BaseView:         base,
		ComponentEntries: ComponentEntries,
		TokenEntries:     TokenEntries,
	})
}

func (h *Handler) tokens(w http.ResponseWriter, r *http.Request) {
	if !h.devOnly(w, r) {
		return
	}
	base, _ := h.lay.Base(r, "dev-showcase")
	sem, scale, prim := splitTokenFamilies(TokenEntries)
	_ = h.tpls.Render(w, http.StatusOK, "showcase.tokens", tokensView{
		BaseView:       base,
		SemanticColors: sem,
		ScaleTokens:    scale,
		PrimitiveRamps: prim,
	})
}

func (h *Handler) components(w http.ResponseWriter, r *http.Request) {
	if !h.devOnly(w, r) {
		return
	}
	base, _ := h.lay.Base(r, "dev-showcase")

	// Pre-render every example through the live template loader. This
	// is the contract enforced by the ui-showcase spec: examples are
	// produced by the same loader that serves product pages. A mismatch
	// between a partial's dict contract and an Example.Dict fails here,
	// in dev, not silently in production.
	rendered := make([]renderedComponentEntry, 0, len(ComponentEntries))
	for _, entry := range ComponentEntries {
		renderedEntry := renderedComponentEntry{ComponentEntry: entry}
		for _, ex := range entry.Examples {
			var buf bytes.Buffer
			if err := h.tpls.RenderPartialTo(&buf, "showcase.components", ex.PartialName, ex.Dict); err != nil {
				// Surface the exact failure in the response so a dev
				// editing the catalogue sees it immediately.
				http.Error(w, "showcase render "+ex.PartialName+": "+err.Error(), http.StatusInternalServerError)
				return
			}
			snippet, err := LookupSnippet(ex.SnippetID)
			if err != nil {
				http.Error(w, "showcase snippet "+ex.SnippetID+": "+err.Error(), http.StatusInternalServerError)
				return
			}
			renderedEntry.RenderedExamples = append(renderedEntry.RenderedExamples, renderedExample{
				ComponentExample: ex,
				HTML:             template.HTML(buf.String()),
				Snippet:          snippet,
			})
		}
		rendered = append(rendered, renderedEntry)
	}

	_ = h.tpls.Render(w, http.StatusOK, "showcase.components", componentsView{
		BaseView: base,
		Entries:  rendered,
	})
}

func (h *Handler) dashboardStates(w http.ResponseWriter, r *http.Request) {
	if !h.devOnly(w, r) {
		return
	}
	base, _ := h.lay.Base(r, "dev-showcase")
	_ = h.tpls.Render(w, http.StatusOK, "showcase.dashboard-states", indexView{BaseView: base})
}

func (h *Handler) emptyStates(w http.ResponseWriter, r *http.Request) {
	if !h.devOnly(w, r) {
		return
	}
	base, _ := h.lay.Base(r, "dev-showcase")
	_ = h.tpls.Render(w, http.StatusOK, "showcase.empty-states", indexView{BaseView: base})
}

// splitTokenFamilies groups tokens into the three sections the tokens
// page renders: semantic colors first, scale tokens second, primitive
// ramps last.
func splitTokenFamilies(all []TokenEntry) (semantic, scale, primitive []TokenEntry) {
	for _, t := range all {
		switch t.Family {
		case "semantic-color":
			semantic = append(semantic, t)
		case "primitive-ramp":
			primitive = append(primitive, t)
		default:
			scale = append(scale, t)
		}
	}
	return
}
