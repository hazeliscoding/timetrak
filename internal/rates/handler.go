package rates

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/clients"
	"timetrak/internal/projects"
	"timetrak/internal/shared/authz"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/templates"
	"timetrak/internal/web/layout"
)

// Handler exposes the rates UI.
type Handler struct {
	svc         *Service
	clientsSvc  *clients.Service
	projectsSvc *projects.Service
	tpls        *templates.Registry
	lay         *layout.Builder
}

// NewHandler constructs the rates handler.
func NewHandler(svc *Service, cs *clients.Service, ps *projects.Service, tpls *templates.Registry, lay *layout.Builder) *Handler {
	return &Handler{svc: svc, clientsSvc: cs, projectsSvc: ps, tpls: tpls, lay: lay}
}

// Register mounts /rates routes.
func (h *Handler) Register(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.Handle("GET /rates", protect(http.HandlerFunc(h.list)))
	mux.Handle("POST /rates", protect(http.HandlerFunc(h.create)))
	mux.Handle("POST /rates/{id}/delete", protect(http.HandlerFunc(h.delete)))
}

type listView struct {
	layout.BaseView
	Rules    []Rule
	Clients  []clients.Client
	Projects []projects.Project
	Form     formView
}

type formView struct {
	Scope         string // "workspace" | "client" | "project"
	ClientID      string
	ProjectID     string
	CurrencyCode  string
	HourlyDecimal string // user input, e.g. "125.50"
	EffectiveFrom string // yyyy-mm-dd
	EffectiveTo   string // yyyy-mm-dd or ""
	Error         string
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, http.StatusOK, formView{Scope: "workspace", CurrencyCode: "USD", EffectiveFrom: time.Now().UTC().Format("2006-01-02")})
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, status int, form formView) {
	wsID := authz.MustFromContext(r.Context()).WorkspaceID
	rules, _ := h.svc.List(r.Context(), wsID)
	cs, _ := h.clientsSvc.ListActive(r.Context(), wsID)
	ps, _ := h.projectsSvc.ListActive(r.Context(), wsID)
	base, _ := h.lay.Base(r, "rates")
	_ = h.tpls.Render(w, status, "rates.index", listView{
		BaseView: base,
		Rules:    rules,
		Clients:  cs,
		Projects: ps,
		Form:     form,
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	wsID := authz.MustFromContext(r.Context()).WorkspaceID
	form := formView{
		Scope:         strings.TrimSpace(r.FormValue("scope")),
		ClientID:      r.FormValue("client_id"),
		ProjectID:     r.FormValue("project_id"),
		CurrencyCode:  strings.ToUpper(strings.TrimSpace(r.FormValue("currency_code"))),
		HourlyDecimal: strings.TrimSpace(r.FormValue("hourly_decimal")),
		EffectiveFrom: r.FormValue("effective_from"),
		EffectiveTo:   r.FormValue("effective_to"),
	}
	in, err := parseInput(form)
	if err != nil {
		form.Error = err.Error()
		h.render(w, r, http.StatusUnprocessableEntity, form)
		return
	}
	if _, err := h.svc.Create(r.Context(), wsID, in); err != nil {
		switch {
		case errors.Is(err, ErrOverlap):
			form.Error = "A rule at this level already covers part of this date range."
		case errors.Is(err, ErrNegativeRate):
			form.Error = "Hourly rate must be zero or positive."
		case errors.Is(err, ErrInvalidCurrency):
			form.Error = "Use a 3-letter ISO currency code (e.g. USD)."
		case errors.Is(err, ErrInvalidWindow):
			form.Error = "End date must be on or after start date."
		case errors.Is(err, ErrClientNotInWS), errors.Is(err, ErrProjectNotInWS):
			sharedhttp.NotFound(w, r)
			return
		default:
			form.Error = "Could not save rate rule."
		}
		h.render(w, r, http.StatusUnprocessableEntity, form)
		return
	}
	http.Redirect(w, r, "/rates", http.StatusSeeOther)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	wsID := authz.MustFromContext(r.Context()).WorkspaceID
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	if err := h.svc.Delete(r.Context(), wsID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			sharedhttp.NotFound(w, r)
			return
		}
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/rates", http.StatusSeeOther)
}

func parseInput(f formView) (Input, error) {
	in := Input{CurrencyCode: f.CurrencyCode}
	if f.CurrencyCode == "" {
		return in, errors.New("currency code is required")
	}
	from, err := time.Parse("2006-01-02", f.EffectiveFrom)
	if err != nil {
		return in, errors.New("pick a valid effective-from date")
	}
	in.EffectiveFrom = from.UTC()
	if f.EffectiveTo != "" {
		to, err := time.Parse("2006-01-02", f.EffectiveTo)
		if err != nil {
			return in, errors.New("pick a valid effective-to date")
		}
		t := to.UTC()
		in.EffectiveTo = &t
	}
	minor, err := parseMinor(f.HourlyDecimal)
	if err != nil {
		return in, err
	}
	in.HourlyRateMinor = minor
	switch f.Scope {
	case "client":
		cid, err := uuid.Parse(f.ClientID)
		if err != nil {
			return in, errors.New("pick a client")
		}
		in.ClientID = cid
	case "project":
		pid, err := uuid.Parse(f.ProjectID)
		if err != nil {
			return in, errors.New("pick a project")
		}
		in.ProjectID = pid
	case "workspace", "":
		// defaults
	default:
		return in, errors.New("invalid scope")
	}
	return in, nil
}

// parseMinor turns user input like "125.50" or "100" into 12550 / 10000 minor units.
// Rejects negative values here; non-negative check happens in service for belt-and-braces.
func parseMinor(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("enter an hourly rate")
	}
	negative := strings.HasPrefix(s, "-")
	if negative {
		return 0, errors.New("rate must not be negative")
	}
	parts := strings.SplitN(s, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, errors.New("enter a valid hourly rate (e.g. 125.50)")
	}
	var frac int64
	if len(parts) == 2 {
		f := parts[1]
		if len(f) > 2 {
			f = f[:2]
		} else if len(f) == 1 {
			f = f + "0"
		}
		if f == "" {
			f = "00"
		}
		frac, err = strconv.ParseInt(f, 10, 64)
		if err != nil {
			return 0, errors.New("enter a valid hourly rate (e.g. 125.50)")
		}
	}
	return whole*100 + frac, nil
}
