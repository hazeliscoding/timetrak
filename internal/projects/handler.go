package projects

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"timetrak/internal/clients"
	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/csrf"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/templates"
	"timetrak/internal/web/layout"
)

// Handler serves the projects domain HTTP endpoints.
type Handler struct {
	svc      *Service
	clients  *clients.Service
	tpls     *templates.Registry
	lay      *layout.Builder
}

// NewHandler constructs the handler.
func NewHandler(svc *Service, clientsSvc *clients.Service, tpls *templates.Registry, lay *layout.Builder) *Handler {
	return &Handler{svc: svc, clients: clientsSvc, tpls: tpls, lay: lay}
}

// Register wires routes.
func (h *Handler) Register(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.Handle("GET /projects", protect(http.HandlerFunc(h.list)))
	mux.Handle("POST /projects", protect(http.HandlerFunc(h.create)))
	mux.Handle("GET /projects/{id}/edit", protect(http.HandlerFunc(h.editRow)))
	mux.Handle("GET /projects/{id}/row", protect(http.HandlerFunc(h.row)))
	mux.Handle("PATCH /projects/{id}", protect(http.HandlerFunc(h.update)))
	mux.Handle("POST /projects/{id}/archive", protect(http.HandlerFunc(h.archive)))
	mux.Handle("POST /projects/{id}/unarchive", protect(http.HandlerFunc(h.unarchive)))
}

type listView struct {
	layout.BaseView
	Projects        []Project
	ActiveClients   []clients.Client
	IncludeArchived bool
	FilterClientID  string
	NewForm         newFormView
}

type newFormView struct {
	Name            string
	Code            string
	ClientID        string
	DefaultBillable bool
	Error           string
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	h.renderList(w, r, http.StatusOK, newFormView{DefaultBillable: true}, "")
}

func (h *Handler) renderList(w http.ResponseWriter, r *http.Request, status int, form newFormView, flash string) {
	wsID := authz.ActiveWorkspace(r.Context())
	include := r.URL.Query().Get("archived") == "1"
	clientFilter := r.URL.Query().Get("client_id")
	var filterID uuid.UUID
	if clientFilter != "" {
		if id, err := uuid.Parse(clientFilter); err == nil {
			filterID = id
		}
	}
	items, err := h.svc.List(r.Context(), wsID, Filters{IncludeArchived: include, ClientID: filterID})
	if err != nil {
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	activeClients, _ := h.clients.ListActive(r.Context(), wsID)
	base, _ := h.lay.Base(r, "projects")
	if flash != "" {
		base.Flash = append(base.Flash, layout.FlashMessage{Kind: "error", Message: flash})
	}
	_ = h.tpls.Render(w, status, "projects.index", listView{
		BaseView:        base,
		Projects:        items,
		ActiveClients:   activeClients,
		IncludeArchived: include,
		FilterClientID:  clientFilter,
		NewForm:         form,
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	wsID := authz.ActiveWorkspace(r.Context())
	clientID, err := uuid.Parse(r.FormValue("client_id"))
	if err != nil {
		h.renderList(w, r, http.StatusUnprocessableEntity, newFormView{
			Name: r.FormValue("name"), Code: r.FormValue("code"),
			DefaultBillable: r.FormValue("default_billable") == "on",
			Error:           "Pick a client.",
		}, "")
		return
	}
	in := CreateInput{
		ClientID:        clientID,
		Name:            r.FormValue("name"),
		Code:            r.FormValue("code"),
		DefaultBillable: r.FormValue("default_billable") == "on",
	}
	if _, err := h.svc.Create(r.Context(), wsID, in); err != nil {
		msg := "Could not create project."
		switch {
		case errors.Is(err, ErrEmptyName):
			msg = "Project name is required."
		case errors.Is(err, ErrClientArchived):
			msg = "Cannot create a project under an archived client."
		case errors.Is(err, ErrClientMismatch):
			http.NotFound(w, r)
			return
		}
		h.renderList(w, r, http.StatusUnprocessableEntity, newFormView{
			Name: in.Name, Code: in.Code, ClientID: clientID.String(),
			DefaultBillable: in.DefaultBillable, Error: msg,
		}, "")
		return
	}
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (h *Handler) projectFromPath(r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	wsID := authz.ActiveWorkspace(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}
	return wsID, id, true
}

type rowView struct {
	CSRFToken string
	Project   Project
	Edit      bool
	Error     string
}

func (h *Handler) row(w http.ResponseWriter, r *http.Request) {
	wsID, id, ok := h.projectFromPath(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	p, err := h.svc.Get(r.Context(), wsID, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = h.tpls.RenderPartial(w, http.StatusOK, "projects.index", "project_row", rowView{CSRFToken: csrf.Token(r), Project: p})
}

func (h *Handler) editRow(w http.ResponseWriter, r *http.Request) {
	wsID, id, ok := h.projectFromPath(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	p, err := h.svc.Get(r.Context(), wsID, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = h.tpls.RenderPartial(w, http.StatusOK, "projects.index", "project_row", rowView{CSRFToken: csrf.Token(r), Project: p, Edit: true})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	wsID, id, ok := h.projectFromPath(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	in := UpdateInput{
		Name:            r.FormValue("name"),
		Code:            r.FormValue("code"),
		DefaultBillable: r.FormValue("default_billable") == "on",
	}
	p, err := h.svc.Update(r.Context(), wsID, id, in)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		existing, gerr := h.svc.Get(r.Context(), wsID, id)
		if gerr != nil {
			http.NotFound(w, r)
			return
		}
		_ = h.tpls.RenderPartial(w, http.StatusUnprocessableEntity, "projects.index", "project_row", rowView{
			CSRFToken: csrf.Token(r), Project: existing, Edit: true, Error: "Project name is required.",
		})
		return
	}
	sharedhttp.TriggerEvent(w, "projects-changed")
	_ = h.tpls.RenderPartial(w, http.StatusOK, "projects.index", "project_row", rowView{CSRFToken: csrf.Token(r), Project: p})
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) { h.toggleArchive(w, r, true) }

func (h *Handler) unarchive(w http.ResponseWriter, r *http.Request) { h.toggleArchive(w, r, false) }

func (h *Handler) toggleArchive(w http.ResponseWriter, r *http.Request, archived bool) {
	wsID, id, ok := h.projectFromPath(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	p, err := h.svc.SetArchived(r.Context(), wsID, id, archived)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to archive", http.StatusInternalServerError)
		return
	}
	sharedhttp.TriggerEvent(w, "projects-changed")
	_ = h.tpls.RenderPartial(w, http.StatusOK, "projects.index", "project_row", rowView{CSRFToken: csrf.Token(r), Project: p})
}
