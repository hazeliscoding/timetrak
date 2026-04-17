package tracking

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/clients"
	"timetrak/internal/projects"
	"timetrak/internal/reporting"
	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/csrf"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/session"
	"timetrak/internal/shared/templates"
	"timetrak/internal/web/layout"
)

// Handler wires tracking routes and the dashboard.
type Handler struct {
	svc         *Service
	projectsSvc *projects.Service
	clientsSvc  *clients.Service
	reportSvc   *reporting.Service
	tpls        *templates.Registry
	lay         *layout.Builder
}

// NewHandler constructs the handler.
func NewHandler(svc *Service, ps *projects.Service, cs *clients.Service, reportSvc *reporting.Service, tpls *templates.Registry, lay *layout.Builder) *Handler {
	return &Handler{svc: svc, projectsSvc: ps, clientsSvc: cs, reportSvc: reportSvc, tpls: tpls, lay: lay}
}

// Register wires routes behind the workspace-protect middleware.
func (h *Handler) Register(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.Handle("GET /dashboard", protect(http.HandlerFunc(h.dashboard)))
	mux.Handle("GET /dashboard/summary", protect(http.HandlerFunc(h.dashboardSummary)))
	mux.Handle("GET /dashboard/timer", protect(http.HandlerFunc(h.timerWidget)))
	mux.Handle("POST /timer/start", protect(http.HandlerFunc(h.startTimer)))
	mux.Handle("POST /timer/stop", protect(http.HandlerFunc(h.stopTimer)))

	mux.Handle("GET /time", protect(http.HandlerFunc(h.entriesList)))
	mux.Handle("POST /time-entries", protect(http.HandlerFunc(h.createManual)))
	mux.Handle("GET /time-entries/{id}/edit", protect(http.HandlerFunc(h.editEntry)))
	mux.Handle("GET /time-entries/{id}/row", protect(http.HandlerFunc(h.entryRow)))
	mux.Handle("PATCH /time-entries/{id}", protect(http.HandlerFunc(h.updateEntry)))
	mux.Handle("DELETE /time-entries/{id}", protect(http.HandlerFunc(h.deleteEntry)))
}

// ---- Dashboard ----

type dashboardView struct {
	layout.BaseView
	Timer    timerView
	Summary  reporting.DashboardSummary
	Projects []projects.Project
}

type timerView struct {
	CSRFToken string
	Running   *Entry
	Projects  []projects.Project
	Error     string
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	wsID := authz.ActiveWorkspace(r.Context())
	sess, _ := session.FromContext(r.Context())
	running, _ := h.svc.GetRunning(r.Context(), wsID, sess.UserID)
	ps, _ := h.projectsSvc.ListActive(r.Context(), wsID)
	summary, _ := h.reportSvc.Dashboard(r.Context(), wsID, sess.UserID, time.Now())
	base, _ := h.lay.Base(r, "dashboard")
	_ = h.tpls.Render(w, http.StatusOK, "dashboard", dashboardView{
		BaseView: base,
		Timer:    timerView{CSRFToken: base.CSRFToken, Running: running, Projects: ps},
		Summary:  summary,
		Projects: ps,
	})
}

func (h *Handler) dashboardSummary(w http.ResponseWriter, r *http.Request) {
	wsID := authz.ActiveWorkspace(r.Context())
	sess, _ := session.FromContext(r.Context())
	summary, _ := h.reportSvc.Dashboard(r.Context(), wsID, sess.UserID, time.Now())
	_ = h.tpls.RenderPartial(w, http.StatusOK, "dashboard", "dashboard_summary", summary)
}

func (h *Handler) timerWidget(w http.ResponseWriter, r *http.Request) {
	h.renderTimer(w, r, http.StatusOK, "")
}

func (h *Handler) renderTimer(w http.ResponseWriter, r *http.Request, status int, errMsg string) {
	wsID := authz.ActiveWorkspace(r.Context())
	sess, _ := session.FromContext(r.Context())
	running, _ := h.svc.GetRunning(r.Context(), wsID, sess.UserID)
	ps, _ := h.projectsSvc.ListActive(r.Context(), wsID)
	_ = h.tpls.RenderPartial(w, status, "dashboard", "timer_widget", timerView{
		CSRFToken: csrf.Token(r), Running: running, Projects: ps, Error: errMsg,
	})
}

func (h *Handler) startTimer(w http.ResponseWriter, r *http.Request) {
	wsID := authz.ActiveWorkspace(r.Context())
	sess, _ := session.FromContext(r.Context())
	projectID, err := uuid.Parse(r.FormValue("project_id"))
	if err != nil {
		h.renderTimer(w, r, http.StatusUnprocessableEntity, "Choose a project to start the timer.")
		return
	}
	in := StartInput{ProjectID: projectID, Description: r.FormValue("description")}
	if _, err := h.svc.StartTimer(r.Context(), wsID, sess.UserID, in); err != nil {
		switch {
		case errors.Is(err, ErrActiveTimerExists):
			h.renderTimer(w, r, http.StatusConflict, "A timer is already running. Stop it first.")
		case errors.Is(err, ErrProjectArchived):
			h.renderTimer(w, r, http.StatusUnprocessableEntity, "That project is archived.")
		case errors.Is(err, ErrProjectNotFound):
			http.NotFound(w, r)
		default:
			h.renderTimer(w, r, http.StatusInternalServerError, "Could not start the timer.")
		}
		return
	}
	sharedhttp.TriggerEvent(w, "timer-changed", "entries-changed")
	h.renderTimer(w, r, http.StatusOK, "")
}

func (h *Handler) stopTimer(w http.ResponseWriter, r *http.Request) {
	wsID := authz.ActiveWorkspace(r.Context())
	sess, _ := session.FromContext(r.Context())
	if _, err := h.svc.StopTimer(r.Context(), wsID, sess.UserID); err != nil {
		if errors.Is(err, ErrNoActiveTimer) {
			h.renderTimer(w, r, http.StatusConflict, "No timer is running.")
			return
		}
		h.renderTimer(w, r, http.StatusInternalServerError, "Could not stop the timer.")
		return
	}
	sharedhttp.TriggerEvent(w, "timer-changed", "entries-changed")
	h.renderTimer(w, r, http.StatusOK, "")
}

// ---- Entries list ----

type entriesView struct {
	layout.BaseView
	Entries         []Entry
	Total           int
	Page            int
	TotalPages      int
	PrevQuery       string
	NextQuery       string
	ActiveClients   []clients.Client
	ActiveProjects  []projects.Project
	Filters         filterForm
	ManualForm      manualFormView
}

type filterForm struct {
	From      string
	To        string
	ClientID  string
	ProjectID string
	Billable  string // "", "yes", "no"
}

type manualFormView struct {
	Date        string
	StartTime   string
	EndTime     string
	ProjectID   string
	Description string
	Billable    bool
	Error       string
}

func (h *Handler) entriesList(w http.ResponseWriter, r *http.Request) {
	h.renderEntries(w, r, manualFormView{Billable: true, Date: time.Now().Format("2006-01-02")})
}

func (h *Handler) renderEntries(w http.ResponseWriter, r *http.Request, form manualFormView) {
	wsID := authz.ActiveWorkspace(r.Context())
	sess, _ := session.FromContext(r.Context())

	q := r.URL.Query()
	filt := filterForm{
		From:      q.Get("from"),
		To:        q.Get("to"),
		ClientID:  q.Get("client_id"),
		ProjectID: q.Get("project_id"),
		Billable:  q.Get("billable"),
	}
	lf := ListFilters{UserID: sess.UserID, Page: atoi(q.Get("page")), PageSize: 25}
	if filt.From != "" {
		if t, err := time.Parse("2006-01-02", filt.From); err == nil {
			lf.From = &t
		}
	}
	if filt.To != "" {
		if t, err := time.Parse("2006-01-02", filt.To); err == nil {
			end := t.Add(24*time.Hour - time.Second)
			lf.To = &end
		}
	}
	if filt.ClientID != "" {
		if id, err := uuid.Parse(filt.ClientID); err == nil {
			lf.ClientID = id
		}
	}
	if filt.ProjectID != "" {
		if id, err := uuid.Parse(filt.ProjectID); err == nil {
			lf.ProjectID = id
		}
	}
	if filt.Billable == "yes" {
		t := true
		lf.Billable = &t
	} else if filt.Billable == "no" {
		f := false
		lf.Billable = &f
	}

	res, err := h.svc.List(r.Context(), wsID, lf)
	if err != nil {
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	acs, _ := h.clientsSvc.ListActive(r.Context(), wsID)
	aps, _ := h.projectsSvc.ListActive(r.Context(), wsID)
	base, _ := h.lay.Base(r, "time")

	baseQuery := url.Values{}
	for k, vs := range q {
		if k == "page" {
			continue
		}
		for _, v := range vs {
			baseQuery.Add(k, v)
		}
	}
	mk := func(page int) string {
		v := url.Values{}
		for k, vs := range baseQuery {
			for _, x := range vs {
				v.Add(k, x)
			}
		}
		v.Set("page", strconv.Itoa(page))
		return v.Encode()
	}

	_ = h.tpls.Render(w, http.StatusOK, "time.index", entriesView{
		BaseView:       base,
		Entries:        res.Entries,
		Total:          res.Total,
		Page:           res.Page,
		TotalPages:     res.TotalPages,
		PrevQuery:      mk(res.Page - 1),
		NextQuery:      mk(res.Page + 1),
		ActiveClients:  acs,
		ActiveProjects: aps,
		Filters:        filt,
		ManualForm:     form,
	})
}

func (h *Handler) createManual(w http.ResponseWriter, r *http.Request) {
	wsID := authz.ActiveWorkspace(r.Context())
	sess, _ := session.FromContext(r.Context())
	form := manualFormView{
		Date:        r.FormValue("date"),
		StartTime:   r.FormValue("start_time"),
		EndTime:     r.FormValue("end_time"),
		ProjectID:   r.FormValue("project_id"),
		Description: r.FormValue("description"),
		Billable:    r.FormValue("is_billable") == "on",
	}
	in, err := parseManualForm(form)
	if err != nil {
		form.Error = err.Error()
		h.renderEntries(w, r, form)
		return
	}
	if _, err := h.svc.CreateManual(r.Context(), wsID, sess.UserID, in); err != nil {
		switch {
		case errors.Is(err, ErrInvalidRange):
			form.Error = "End must be on or after start."
		case errors.Is(err, ErrProjectArchived):
			form.Error = "That project is archived."
		case errors.Is(err, ErrProjectNotFound):
			http.NotFound(w, r)
			return
		default:
			form.Error = "Could not create entry."
		}
		h.renderEntries(w, r, form)
		return
	}
	http.Redirect(w, r, "/time", http.StatusSeeOther)
}

type entryRowView struct {
	CSRFToken string
	Entry     Entry
	Edit      bool
	Error     string
	Projects  []projects.Project
}

func (h *Handler) entryRow(w http.ResponseWriter, r *http.Request) {
	wsID := authz.ActiveWorkspace(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	e, err := h.svc.Get(r.Context(), wsID, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = h.tpls.RenderPartial(w, http.StatusOK, "time.index", "entry_row", entryRowView{CSRFToken: csrf.Token(r), Entry: e})
}

func (h *Handler) editEntry(w http.ResponseWriter, r *http.Request) {
	wsID := authz.ActiveWorkspace(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	e, err := h.svc.Get(r.Context(), wsID, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ps, _ := h.projectsSvc.ListActive(r.Context(), wsID)
	_ = h.tpls.RenderPartial(w, http.StatusOK, "time.index", "entry_row", entryRowView{
		CSRFToken: csrf.Token(r), Entry: e, Edit: true, Projects: ps,
	})
}

func (h *Handler) updateEntry(w http.ResponseWriter, r *http.Request) {
	wsID := authz.ActiveWorkspace(r.Context())
	sess, _ := session.FromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	projectID, err := uuid.Parse(r.FormValue("project_id"))
	if err != nil {
		http.Error(w, "invalid project", http.StatusUnprocessableEntity)
		return
	}
	startedAt, err := time.Parse(time.RFC3339, r.FormValue("started_at"))
	if err != nil {
		http.Error(w, "invalid started_at", http.StatusUnprocessableEntity)
		return
	}
	endedAt, err := time.Parse(time.RFC3339, r.FormValue("ended_at"))
	if err != nil {
		http.Error(w, "invalid ended_at", http.StatusUnprocessableEntity)
		return
	}
	in := ManualInput{
		ProjectID:   projectID,
		Description: r.FormValue("description"),
		StartedAt:   startedAt.UTC(),
		EndedAt:     endedAt.UTC(),
		IsBillable:  r.FormValue("is_billable") == "on",
	}
	e, err := h.svc.Edit(r.Context(), wsID, sess.UserID, id, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrEntryNotFound):
			http.NotFound(w, r)
		case errors.Is(err, ErrActiveTimerExists):
			http.Error(w, "conflict: edit would create a second running timer", http.StatusConflict)
		case errors.Is(err, ErrInvalidRange):
			http.Error(w, "end must be on or after start", http.StatusUnprocessableEntity)
		default:
			http.Error(w, "update failed", http.StatusInternalServerError)
		}
		return
	}
	sharedhttp.TriggerEvent(w, "entries-changed")
	_ = h.tpls.RenderPartial(w, http.StatusOK, "time.index", "entry_row", entryRowView{CSRFToken: csrf.Token(r), Entry: e})
}

func (h *Handler) deleteEntry(w http.ResponseWriter, r *http.Request) {
	wsID := authz.ActiveWorkspace(r.Context())
	sess, _ := session.FromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.svc.Delete(r.Context(), wsID, sess.UserID, id); err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	sharedhttp.TriggerEvent(w, "entries-changed")
	w.WriteHeader(http.StatusOK)
}

func parseManualForm(f manualFormView) (ManualInput, error) {
	in := ManualInput{IsBillable: f.Billable, Description: f.Description}
	pid, err := uuid.Parse(f.ProjectID)
	if err != nil {
		return in, errors.New("pick a project")
	}
	in.ProjectID = pid
	start, err := time.Parse("2006-01-02T15:04", f.Date+"T"+f.StartTime)
	if err != nil {
		return in, errors.New("pick a valid start date and time")
	}
	end, err := time.Parse("2006-01-02T15:04", f.Date+"T"+f.EndTime)
	if err != nil {
		return in, errors.New("pick a valid end date and time")
	}
	in.StartedAt = start.UTC()
	in.EndedAt = end.UTC()
	return in, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	if n < 1 {
		return 1
	}
	return n
}
