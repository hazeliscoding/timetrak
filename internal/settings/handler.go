// Package settings exposes workspace-scoped settings pages.
//
// Lives in its own package (not internal/workspace) to avoid a cycle:
// internal/web/layout imports internal/workspace for Workspace listings,
// and the settings page needs a layout.Builder.
package settings

import (
	"errors"
	"net/http"

	"timetrak/internal/shared/authz"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/templates"
	"timetrak/internal/web/layout"
	"timetrak/internal/workspace"
)

// Handler serves /workspace/settings*.
type Handler struct {
	svc       *workspace.Service
	tpls      *templates.Registry
	lay       *layout.Builder
	timezones []string // snapshot of pg_timezone_names at startup
}

// NewHandler constructs the settings handler. `timezones` should be the
// ordered list returned by workspace.Service.ListTimezones at startup.
func NewHandler(svc *workspace.Service, tpls *templates.Registry, lay *layout.Builder, timezones []string) *Handler {
	return &Handler{svc: svc, tpls: tpls, lay: lay, timezones: timezones}
}

// Register mounts the settings routes behind the auth+workspace middleware.
func (h *Handler) Register(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.Handle("GET /workspace/settings", protect(http.HandlerFunc(h.get)))
	mux.Handle("POST /workspace/settings/timezone", protect(http.HandlerFunc(h.postTimezone)))
}

type view struct {
	layout.BaseView
	Workspace workspace.Workspace
	Timezones []string
	FormError string
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	wc := authz.MustFromContext(r.Context())
	ws, err := h.svc.Get(r.Context(), wc.UserID, wc.WorkspaceID)
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	base, _ := h.lay.Base(r, "settings")
	_ = h.tpls.Render(w, http.StatusOK, "workspace.settings", view{
		BaseView:  base,
		Workspace: ws,
		Timezones: h.timezones,
	})
}

func (h *Handler) postTimezone(w http.ResponseWriter, r *http.Request) {
	wc := authz.MustFromContext(r.Context())
	tz := r.FormValue("reporting_timezone")
	err := h.svc.UpdateReportingTimezone(r.Context(), wc.UserID, wc.WorkspaceID, tz)
	if err != nil {
		if errors.Is(err, workspace.ErrForbidden) {
			sharedhttp.NotFound(w, r)
			return
		}
		if errors.Is(err, workspace.ErrInvalidTimezone) {
			ws, gerr := h.svc.Get(r.Context(), wc.UserID, wc.WorkspaceID)
			if gerr != nil {
				sharedhttp.NotFound(w, r)
				return
			}
			base, _ := h.lay.Base(r, "settings")
			_ = h.tpls.Render(w, http.StatusUnprocessableEntity, "workspace.settings", view{
				BaseView:  base,
				Workspace: ws,
				Timezones: h.timezones,
				FormError: "That timezone is not recognized. Pick one from the list.",
			})
			return
		}
		http.Error(w, "failed to update timezone", http.StatusInternalServerError)
		return
	}
	if sharedhttp.IsHTMX(r) {
		sharedhttp.TriggerEvent(w, "workspace-changed")
	}
	http.Redirect(w, r, "/workspace/settings", http.StatusSeeOther)
}
