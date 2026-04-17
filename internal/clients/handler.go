package clients

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/csrf"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/templates"
	"timetrak/internal/web/layout"
)

// Handler serves the clients domain HTTP endpoints.
type Handler struct {
	svc  *Service
	tpls *templates.Registry
	lay  *layout.Builder
}

// NewHandler constructs the handler.
func NewHandler(svc *Service, tpls *templates.Registry, lay *layout.Builder) *Handler {
	return &Handler{svc: svc, tpls: tpls, lay: lay}
}

// Register wires the routes. Each route expects the workspace-member middleware in front.
func (h *Handler) Register(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.Handle("GET /clients", protect(http.HandlerFunc(h.list)))
	mux.Handle("POST /clients", protect(http.HandlerFunc(h.create)))
	mux.Handle("GET /clients/{id}/edit", protect(http.HandlerFunc(h.editRow)))
	mux.Handle("GET /clients/{id}/row", protect(http.HandlerFunc(h.row)))
	mux.Handle("PATCH /clients/{id}", protect(http.HandlerFunc(h.update)))
	mux.Handle("POST /clients/{id}/archive", protect(http.HandlerFunc(h.archive)))
	mux.Handle("POST /clients/{id}/unarchive", protect(http.HandlerFunc(h.unarchive)))
}

type listView struct {
	layout.BaseView
	Clients         []Client
	IncludeArchived bool
	NewForm         newFormView
}

type newFormView struct {
	Name         string
	ContactEmail string
	Error        string
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	wsID := authz.MustFromContext(r.Context()).WorkspaceID
	include := r.URL.Query().Get("archived") == "1"
	items, err := h.svc.List(r.Context(), wsID, include)
	if err != nil {
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	base, _ := h.lay.Base(r, "clients")
	_ = h.tpls.Render(w, http.StatusOK, "clients.index", listView{
		BaseView:        base,
		Clients:         items,
		IncludeArchived: include,
		NewForm:         newFormView{},
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	wsID := authz.MustFromContext(r.Context()).WorkspaceID
	name := r.FormValue("name")
	email := r.FormValue("contact_email")
	if _, err := h.svc.Create(r.Context(), wsID, name, email); err != nil {
		// Re-render list with inline error.
		items, _ := h.svc.List(r.Context(), wsID, false)
		base, _ := h.lay.Base(r, "clients")
		msg := "Client name is required."
		if !errors.Is(err, ErrEmptyName) {
			msg = "Could not create client."
		}
		_ = h.tpls.Render(w, http.StatusUnprocessableEntity, "clients.index", listView{
			BaseView: base,
			Clients:  items,
			NewForm:  newFormView{Name: name, ContactEmail: email, Error: msg},
		})
		return
	}
	http.Redirect(w, r, "/clients", http.StatusSeeOther)
}

func (h *Handler) clientFromPath(r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	wsID := authz.MustFromContext(r.Context()).WorkspaceID
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, false
	}
	return wsID, id, true
}

type rowView struct {
	CSRFToken string
	Client    Client
	Edit      bool
	Error     string
}

func (h *Handler) row(w http.ResponseWriter, r *http.Request) {
	wsID, id, ok := h.clientFromPath(r)
	if !ok {
		sharedhttp.NotFound(w, r)
		return
	}
	c, err := h.svc.Get(r.Context(), wsID, id)
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	_ = h.tpls.RenderPartial(w, http.StatusOK, "clients.index", "client_row", rowView{CSRFToken: csrf.Token(r), Client: c})
}

func (h *Handler) editRow(w http.ResponseWriter, r *http.Request) {
	wsID, id, ok := h.clientFromPath(r)
	if !ok {
		sharedhttp.NotFound(w, r)
		return
	}
	c, err := h.svc.Get(r.Context(), wsID, id)
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	_ = h.tpls.RenderPartial(w, http.StatusOK, "clients.index", "client_row", rowView{CSRFToken: csrf.Token(r), Client: c, Edit: true})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	wsID, id, ok := h.clientFromPath(r)
	if !ok {
		sharedhttp.NotFound(w, r)
		return
	}
	c, err := h.svc.Update(r.Context(), wsID, id, r.FormValue("name"), r.FormValue("contact_email"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			sharedhttp.NotFound(w, r)
			return
		}
		existing, gerr := h.svc.Get(r.Context(), wsID, id)
		if gerr != nil {
			sharedhttp.NotFound(w, r)
			return
		}
		_ = h.tpls.RenderPartial(w, http.StatusUnprocessableEntity, "clients.index", "client_row", rowView{
			CSRFToken: csrf.Token(r), Client: existing, Edit: true, Error: "Client name is required.",
		})
		return
	}
	sharedhttp.TriggerEvent(w, "clients-changed")
	_ = h.tpls.RenderPartial(w, http.StatusOK, "clients.index", "client_row", rowView{CSRFToken: csrf.Token(r), Client: c})
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) { h.toggleArchive(w, r, true) }

func (h *Handler) unarchive(w http.ResponseWriter, r *http.Request) { h.toggleArchive(w, r, false) }

func (h *Handler) toggleArchive(w http.ResponseWriter, r *http.Request, archived bool) {
	wsID, id, ok := h.clientFromPath(r)
	if !ok {
		sharedhttp.NotFound(w, r)
		return
	}
	c, err := h.svc.SetArchived(r.Context(), wsID, id, archived)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			sharedhttp.NotFound(w, r)
			return
		}
		http.Error(w, "failed to archive", http.StatusInternalServerError)
		return
	}
	sharedhttp.TriggerEvent(w, "clients-changed")
	_ = h.tpls.RenderPartial(w, http.StatusOK, "clients.index", "client_row", rowView{CSRFToken: csrf.Token(r), Client: c})
}
