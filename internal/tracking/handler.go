package tracking

// Taxonomy of tracking integrity errors (see internal/tracking/errors.go and
// openspec/specs/tracking). Handlers MUST map these to HTTP status + stable
// error codes as follows — do NOT invent new mappings here without an
// accompanying change proposal:
//
//   ErrActiveTimerExists     -> 409  tracking.active_timer
//   ErrNoActiveTimer         -> 409  tracking.no_active_timer
//   ErrInvalidInterval       -> 422  tracking.invalid_interval
//   ErrCrossWorkspaceProject -> 422  tracking.cross_workspace
//   (unmapped)               -> 500  (logged at error, no error_kind)
//
// Every taxonomy response MUST be logged at warn with structured fields
// `tracking.error_kind`, `workspace_id`, `user_id`, plus `entry_id`/`project_id`
// when known. HX-Trigger events (`timer-changed`, `entries-changed`) are only
// emitted on success.

import (
	"errors"
	"log/slog"
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
	logger      *slog.Logger
}

// NewHandler constructs the handler.
func NewHandler(svc *Service, ps *projects.Service, cs *clients.Service, reportSvc *reporting.Service, tpls *templates.Registry, lay *layout.Builder) *Handler {
	return &Handler{svc: svc, projectsSvc: ps, clientsSvc: cs, reportSvc: reportSvc, tpls: tpls, lay: lay, logger: slog.Default()}
}

// SetLogger overrides the structured logger. Used by tests to capture output.
func (h *Handler) SetLogger(l *slog.Logger) { h.logger = l }

// taxonomyResponse resolves a tracking error into its HTTP status, stable
// error code, and human copy. Unknown errors return ("", 0, "") so callers
// fall back to their default path (500 + generic copy, logged at error).
func taxonomyResponse(err error) (code string, status int, message string) {
	switch {
	case errors.Is(err, ErrActiveTimerExists):
		return ErrCodeActiveTimer, http.StatusConflict, "A timer is already running. Stop it first."
	case errors.Is(err, ErrNoActiveTimer):
		return ErrCodeNoActiveTimer, http.StatusConflict, "No timer is running."
	case errors.Is(err, ErrInvalidInterval):
		return ErrCodeInvalidInterval, http.StatusUnprocessableEntity, "End time must be after start time."
	case errors.Is(err, ErrCrossWorkspaceProject):
		return ErrCodeCrossWorkspace, http.StatusUnprocessableEntity, "That project is not in this workspace."
	default:
		return "", 0, ""
	}
}

// logTaxonomy emits a warn line for known taxonomy errors with the structured
// fields described in the handler taxonomy comment. attrs supplies optional
// context (entry_id, project_id).
func (h *Handler) logTaxonomy(r *http.Request, code string, attrs ...any) {
	if h.logger == nil {
		return
	}
	wc := authz.MustFromContext(r.Context())
	base := []any{
		"tracking.error_kind", code,
		"workspace_id", wc.WorkspaceID.String(),
		"user_id", wc.UserID.String(),
	}
	base = append(base, attrs...)
	h.logger.LogAttrs(r.Context(), slog.LevelWarn, "tracking integrity failure", slogAttrs(base)...)
}

// logUnmapped emits an error line for a SQLSTATE that falls outside the
// taxonomy (no tracking.error_kind field is set — dashboards key on its
// absence to distinguish known vs unknown failures).
func (h *Handler) logUnmapped(r *http.Request, err error, attrs ...any) {
	if h.logger == nil {
		return
	}
	wc := authz.MustFromContext(r.Context())
	base := []any{
		"workspace_id", wc.WorkspaceID.String(),
		"user_id", wc.UserID.String(),
		"err", err.Error(),
	}
	base = append(base, attrs...)
	h.logger.LogAttrs(r.Context(), slog.LevelError, "tracking unmapped failure", slogAttrs(base)...)
}

func slogAttrs(kv []any) []slog.Attr {
	out := make([]slog.Attr, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		key, _ := kv[i].(string)
		out = append(out, slog.Any(key, kv[i+1]))
	}
	return out
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
	Error     string // human copy, rendered if ErrorCode is empty
	ErrorCode string // stable taxonomy code, e.g. "tracking.active_timer"
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	wc := authz.MustFromContext(r.Context())
	running, _ := h.svc.GetRunning(r.Context(), wc.WorkspaceID, wc.UserID)
	ps, _ := h.projectsSvc.ListActive(r.Context(), wc.WorkspaceID)
	summary, _ := h.reportSvc.Dashboard(r.Context(), wc.WorkspaceID, wc.UserID, time.Now())
	base, _ := h.lay.Base(r, "dashboard")
	_ = h.tpls.Render(w, http.StatusOK, "dashboard", dashboardView{
		BaseView: base,
		Timer:    timerView{CSRFToken: base.CSRFToken, Running: running, Projects: ps},
		Summary:  summary,
		Projects: ps,
	})
}

func (h *Handler) dashboardSummary(w http.ResponseWriter, r *http.Request) {
	wc := authz.MustFromContext(r.Context())
	summary, _ := h.reportSvc.Dashboard(r.Context(), wc.WorkspaceID, wc.UserID, time.Now())
	_ = h.tpls.RenderPartial(w, http.StatusOK, "dashboard", "dashboard_summary", summary)
}

func (h *Handler) timerWidget(w http.ResponseWriter, r *http.Request) {
	h.renderTimer(w, r, http.StatusOK, "", "")
}

func (h *Handler) renderTimer(w http.ResponseWriter, r *http.Request, status int, errCode, errMsg string) {
	wc := authz.MustFromContext(r.Context())
	running, _ := h.svc.GetRunning(r.Context(), wc.WorkspaceID, wc.UserID)
	ps, _ := h.projectsSvc.ListActive(r.Context(), wc.WorkspaceID)
	_ = h.tpls.RenderPartial(w, status, "dashboard", "timer_widget", timerView{
		CSRFToken: csrf.Token(r), Running: running, Projects: ps, Error: errMsg, ErrorCode: errCode,
	})
}

func (h *Handler) startTimer(w http.ResponseWriter, r *http.Request) {
	wc := authz.MustFromContext(r.Context())
	projectID, err := uuid.Parse(r.FormValue("project_id"))
	if err != nil {
		h.renderTimer(w, r, http.StatusUnprocessableEntity, "", "Choose a project to start the timer.")
		return
	}
	in := StartInput{ProjectID: projectID, Description: r.FormValue("description")}
	if _, err := h.svc.StartTimer(r.Context(), wc.WorkspaceID, wc.UserID, in); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			// Cross-workspace denial runs first; composite FK is defense in
			// depth, not the primary gate.
			sharedhttp.NotFound(w, r)
			return
		}
		if errors.Is(err, ErrProjectArchived) {
			h.renderTimer(w, r, http.StatusUnprocessableEntity, "", "That project is archived.")
			return
		}
		if code, status, msg := taxonomyResponse(err); code != "" {
			h.logTaxonomy(r, code, "project_id", projectID.String())
			h.renderTimer(w, r, status, code, msg)
			return
		}
		h.logUnmapped(r, err, "project_id", projectID.String())
		h.renderTimer(w, r, http.StatusInternalServerError, "", "Could not start the timer.")
		return
	}
	sharedhttp.TriggerEvent(w, "timer-changed", "entries-changed")
	h.renderTimer(w, r, http.StatusOK, "", "")
}

func (h *Handler) stopTimer(w http.ResponseWriter, r *http.Request) {
	wc := authz.MustFromContext(r.Context())
	if _, err := h.svc.StopTimer(r.Context(), wc.WorkspaceID, wc.UserID); err != nil {
		if code, status, msg := taxonomyResponse(err); code != "" {
			h.logTaxonomy(r, code)
			h.renderTimer(w, r, status, code, msg)
			return
		}
		h.logUnmapped(r, err)
		h.renderTimer(w, r, http.StatusInternalServerError, "", "Could not stop the timer.")
		return
	}
	_ = wc
	sharedhttp.TriggerEvent(w, "timer-changed", "entries-changed")
	h.renderTimer(w, r, http.StatusOK, "", "")
}

// ---- Entries list ----

type entriesView struct {
	layout.BaseView
	Entries        []Entry
	Total          int
	Page           int
	TotalPages     int
	PrevQuery      string
	NextQuery      string
	ActiveClients  []clients.Client
	ActiveProjects []projects.Project
	Filters        filterForm
	ManualForm     manualFormView
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
	wc := authz.MustFromContext(r.Context())

	q := r.URL.Query()
	filt := filterForm{
		From:      q.Get("from"),
		To:        q.Get("to"),
		ClientID:  q.Get("client_id"),
		ProjectID: q.Get("project_id"),
		Billable:  q.Get("billable"),
	}
	lf := ListFilters{UserID: wc.UserID, Page: atoi(q.Get("page")), PageSize: 25}
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

	res, err := h.svc.List(r.Context(), wc.WorkspaceID, lf)
	if err != nil {
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	acs, _ := h.clientsSvc.ListActive(r.Context(), wc.WorkspaceID)
	aps, _ := h.projectsSvc.ListActive(r.Context(), wc.WorkspaceID)
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
	wc := authz.MustFromContext(r.Context())
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
	if _, err := h.svc.CreateManual(r.Context(), wc.WorkspaceID, wc.UserID, in); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			sharedhttp.NotFound(w, r)
			return
		}
		if errors.Is(err, ErrProjectArchived) {
			form.Error = "That project is archived."
			h.renderEntries(w, r, form)
			return
		}
		if code, _, msg := taxonomyResponse(err); code != "" {
			h.logTaxonomy(r, code, "project_id", in.ProjectID.String())
			form.Error = msg
			h.renderEntries(w, r, form)
			return
		}
		h.logUnmapped(r, err, "project_id", in.ProjectID.String())
		form.Error = "Could not create entry."
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
	ErrorCode string // stable taxonomy code for the shared tracking_error partial
	Projects  []projects.Project
}

func (h *Handler) entryRow(w http.ResponseWriter, r *http.Request) {
	wc := authz.MustFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	e, err := h.svc.Get(r.Context(), wc.WorkspaceID, id)
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	_ = h.tpls.RenderPartial(w, http.StatusOK, "time.index", "entry_row", entryRowView{CSRFToken: csrf.Token(r), Entry: e})
}

func (h *Handler) editEntry(w http.ResponseWriter, r *http.Request) {
	wc := authz.MustFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	e, err := h.svc.Get(r.Context(), wc.WorkspaceID, id)
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	ps, _ := h.projectsSvc.ListActive(r.Context(), wc.WorkspaceID)
	_ = h.tpls.RenderPartial(w, http.StatusOK, "time.index", "entry_row", entryRowView{
		CSRFToken: csrf.Token(r), Entry: e, Edit: true, Projects: ps,
	})
}

func (h *Handler) updateEntry(w http.ResponseWriter, r *http.Request) {
	wc := authz.MustFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		sharedhttp.NotFound(w, r)
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
	e, err := h.svc.Edit(r.Context(), wc.WorkspaceID, wc.UserID, id, in)
	if err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			sharedhttp.NotFound(w, r)
			return
		}
		// Re-fetch the existing entry so the edit form can re-render with
		// the user's in-flight values preserved (project_id, timestamps).
		// If Get fails, fall back to an empty entry.
		existing, _ := h.svc.Get(r.Context(), wc.WorkspaceID, id)
		if existing.ID == uuid.Nil {
			existing = Entry{ID: id, WorkspaceID: wc.WorkspaceID, ProjectID: in.ProjectID, StartedAt: in.StartedAt, EndedAt: &in.EndedAt, IsBillable: in.IsBillable, Description: in.Description}
		}
		ps, _ := h.projectsSvc.ListActive(r.Context(), wc.WorkspaceID)
		if code, status, msg := taxonomyResponse(err); code != "" {
			h.logTaxonomy(r, code, "entry_id", id.String(), "project_id", in.ProjectID.String())
			_ = h.tpls.RenderPartial(w, status, "time.index", "entry_row", entryRowView{
				CSRFToken: csrf.Token(r), Entry: existing, Edit: true, Error: msg, ErrorCode: code, Projects: ps,
			})
			return
		}
		h.logUnmapped(r, err, "entry_id", id.String(), "project_id", in.ProjectID.String())
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	sharedhttp.TriggerEvent(w, "entries-changed")
	_ = h.tpls.RenderPartial(w, http.StatusOK, "time.index", "entry_row", entryRowView{CSRFToken: csrf.Token(r), Entry: e})
}

func (h *Handler) deleteEntry(w http.ResponseWriter, r *http.Request) {
	wc := authz.MustFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	if err := h.svc.Delete(r.Context(), wc.WorkspaceID, wc.UserID, id); err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			sharedhttp.NotFound(w, r)
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
