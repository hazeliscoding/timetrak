package rates

import (
	"bytes"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/clients"
	"timetrak/internal/projects"
	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/csrf"
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
	mux.Handle("GET /rates/{id}/edit", protect(http.HandlerFunc(h.editRow)))
	mux.Handle("GET /rates/{id}/row", protect(http.HandlerFunc(h.row)))
	mux.Handle("POST /rates/{id}", protect(http.HandlerFunc(h.update)))
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

type rateFormView struct {
	Form      formView
	Clients   []clients.Client
	Projects  []projects.Project
	CSRFToken string
	OOB       bool
}

type rateRowView struct {
	Rule        Rule
	Edit        bool
	Error       string
	AttemptedTo string
	CSRFToken   string
}

type ratesTableView struct {
	Rules     []Rule
	CSRFToken string
}

func defaultFormView() formView {
	return formView{Scope: "workspace", CurrencyCode: "USD", EffectiveFrom: time.Now().UTC().Format("2006-01-02")}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, http.StatusOK, defaultFormView())
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

func (h *Handler) parseForm(r *http.Request) formView {
	return formView{
		Scope:         strings.TrimSpace(r.FormValue("scope")),
		ClientID:      r.FormValue("client_id"),
		ProjectID:     r.FormValue("project_id"),
		CurrencyCode:  strings.ToUpper(strings.TrimSpace(r.FormValue("currency_code"))),
		HourlyDecimal: strings.TrimSpace(r.FormValue("hourly_decimal")),
		EffectiveFrom: r.FormValue("effective_from"),
		EffectiveTo:   r.FormValue("effective_to"),
	}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	wsID := authz.MustFromContext(r.Context()).WorkspaceID
	form := h.parseForm(r)
	in, err := parseInput(form)
	if err != nil {
		form.Error = err.Error()
		h.renderFormError(w, r, http.StatusUnprocessableEntity, form)
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
		h.renderFormError(w, r, http.StatusUnprocessableEntity, form)
		return
	}
	if sharedhttp.IsHTMX(r) {
		sharedhttp.TriggerEvent(w, "rates-changed")
		h.renderTableAndFormReset(w, r, http.StatusOK)
		return
	}
	http.Redirect(w, r, "/rates", http.StatusSeeOther)
}

// renderFormError renders the validation-failed response for POST /rates. On
// HX requests this returns the rate_form partial only (422). Non-HX falls
// back to a full page re-render to match the pre-HTMX behavior.
func (h *Handler) renderFormError(w http.ResponseWriter, r *http.Request, status int, form formView) {
	if sharedhttp.IsHTMX(r) {
		wsID := authz.MustFromContext(r.Context()).WorkspaceID
		cs, _ := h.clientsSvc.ListActive(r.Context(), wsID)
		ps, _ := h.projectsSvc.ListActive(r.Context(), wsID)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		_ = h.tpls.RenderPartialTo(w, "rates.index", "rate_form", rateFormView{
			Form: form, Clients: cs, Projects: ps, CSRFToken: csrf.Token(r),
		})
		return
	}
	h.render(w, r, status, form)
}

// renderTableAndFormReset writes the rates_table partial (main swap) plus an
// OOB rate_form partial reset to defaults. Called after a successful HX
// create or delete.
func (h *Handler) renderTableAndFormReset(w http.ResponseWriter, r *http.Request, status int) {
	wsID := authz.MustFromContext(r.Context()).WorkspaceID
	rules, _ := h.svc.List(r.Context(), wsID)
	cs, _ := h.clientsSvc.ListActive(r.Context(), wsID)
	ps, _ := h.projectsSvc.ListActive(r.Context(), wsID)
	token := csrf.Token(r)
	var buf bytes.Buffer
	_ = h.tpls.RenderPartialTo(&buf, "rates.index", "rates_table", ratesTableView{Rules: rules, CSRFToken: token})
	_ = h.tpls.RenderPartialTo(&buf, "rates.index", "rate_form", rateFormView{
		Form: defaultFormView(), Clients: cs, Projects: ps, CSRFToken: token, OOB: true,
	})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

// update edits an existing rule. Only changes permitted by the service's
// historical-safety policy succeed; otherwise the page re-renders with an
// inline 409 message.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	wsID := authz.MustFromContext(r.Context()).WorkspaceID
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	form := h.parseForm(r)
	in, err := parseInput(form)
	if err != nil {
		h.renderRowEditError(w, r, wsID, id, http.StatusUnprocessableEntity, err.Error(), form.EffectiveTo)
		return
	}
	if err := h.svc.Update(r.Context(), wsID, id, in); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			sharedhttp.NotFound(w, r)
			return
		case errors.Is(err, ErrRuleReferenced):
			h.renderRowEditError(w, r, wsID, id, http.StatusConflict,
				"This rule is referenced by historical time entries. Only the end date may be extended; other changes would move past totals.",
				form.EffectiveTo)
			return
		case errors.Is(err, ErrOverlap):
			h.renderRowEditError(w, r, wsID, id, http.StatusUnprocessableEntity,
				"A rule at this level already covers part of this date range.", form.EffectiveTo)
			return
		case errors.Is(err, ErrNegativeRate):
			h.renderRowEditError(w, r, wsID, id, http.StatusUnprocessableEntity,
				"Hourly rate must be zero or positive.", form.EffectiveTo)
			return
		case errors.Is(err, ErrInvalidCurrency):
			h.renderRowEditError(w, r, wsID, id, http.StatusUnprocessableEntity,
				"Use a 3-letter ISO currency code (e.g. USD).", form.EffectiveTo)
			return
		case errors.Is(err, ErrInvalidWindow):
			h.renderRowEditError(w, r, wsID, id, http.StatusUnprocessableEntity,
				"End date must be on or after start date.", form.EffectiveTo)
			return
		case errors.Is(err, ErrClientNotInWS), errors.Is(err, ErrProjectNotInWS):
			sharedhttp.NotFound(w, r)
			return
		default:
			h.renderRowEditError(w, r, wsID, id, http.StatusUnprocessableEntity,
				"Could not update rate rule.", form.EffectiveTo)
			return
		}
	}
	if sharedhttp.IsHTMX(r) {
		rule, gerr := h.svc.Get(r.Context(), wsID, id)
		if gerr != nil {
			sharedhttp.NotFound(w, r)
			return
		}
		sharedhttp.TriggerEvent(w, "rates-changed")
		_ = h.tpls.RenderPartial(w, http.StatusOK, "rates.index", "rate_row", rateRowView{
			Rule: rule, CSRFToken: csrf.Token(r),
		})
		return
	}
	http.Redirect(w, r, "/rates", http.StatusSeeOther)
}

// renderRowEditError renders an error response for an in-edit rate row. On
// HX requests it returns the rate_row partial in edit mode with the inline
// error. Non-HX falls back to a full page re-render using the form error
// surface to match the pre-HTMX UX.
func (h *Handler) renderRowEditError(w http.ResponseWriter, r *http.Request, wsID, id uuid.UUID, status int, msg, attemptedTo string) {
	if sharedhttp.IsHTMX(r) {
		rule, gerr := h.svc.Get(r.Context(), wsID, id)
		if gerr != nil {
			sharedhttp.NotFound(w, r)
			return
		}
		_ = h.tpls.RenderPartial(w, status, "rates.index", "rate_row", rateRowView{
			Rule: rule, Edit: true, Error: msg, AttemptedTo: attemptedTo, CSRFToken: csrf.Token(r),
		})
		return
	}
	form := defaultFormView()
	form.Error = msg
	h.render(w, r, status, form)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	wsID := authz.MustFromContext(r.Context()).WorkspaceID
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	if err := h.svc.Delete(r.Context(), wsID, id); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			sharedhttp.NotFound(w, r)
			return
		case errors.Is(err, ErrRuleReferenced):
			msg := "This rule is referenced by historical time entries and cannot be deleted. Create a successor rule from a future date instead."
			if sharedhttp.IsHTMX(r) {
				rule, gerr := h.svc.Get(r.Context(), wsID, id)
				if gerr != nil {
					sharedhttp.NotFound(w, r)
					return
				}
				_ = h.tpls.RenderPartial(w, http.StatusConflict, "rates.index", "rate_row", rateRowView{
					Rule: rule, Error: msg, CSRFToken: csrf.Token(r),
				})
				return
			}
			form := defaultFormView()
			form.Error = msg
			h.render(w, r, http.StatusConflict, form)
			return
		default:
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
	}
	if sharedhttp.IsHTMX(r) {
		sharedhttp.TriggerEvent(w, "rates-changed")
		h.renderTableAndFormReset(w, r, http.StatusOK)
		return
	}
	http.Redirect(w, r, "/rates", http.StatusSeeOther)
}

// editRow returns the rate_row partial in edit mode. Workspace-scoped — a
// rule id that does not belong to the active workspace returns 404.
func (h *Handler) editRow(w http.ResponseWriter, r *http.Request) {
	h.renderRow(w, r, true)
}

// row returns the rate_row partial in display mode. Used by the Cancel
// button to restore the row after abandoning an edit.
func (h *Handler) row(w http.ResponseWriter, r *http.Request) {
	h.renderRow(w, r, false)
}

func (h *Handler) renderRow(w http.ResponseWriter, r *http.Request, edit bool) {
	wsID := authz.MustFromContext(r.Context()).WorkspaceID
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	rule, err := h.svc.Get(r.Context(), wsID, id)
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	_ = h.tpls.RenderPartial(w, http.StatusOK, "rates.index", "rate_row", rateRowView{
		Rule: rule, Edit: edit, CSRFToken: csrf.Token(r),
	})
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
