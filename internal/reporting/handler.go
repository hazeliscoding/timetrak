package reporting

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/clients"
	"timetrak/internal/projects"
	"timetrak/internal/shared/authz"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/templates"
	"timetrak/internal/web/layout"
	"timetrak/internal/workspace"
)

// Handler serves /reports.
type Handler struct {
	svc         *Service
	clientsSvc  *clients.Service
	projectsSvc *projects.Service
	workspace   *workspace.Service
	tpls        *templates.Registry
	lay         *layout.Builder
}

// NewHandler constructs the handler. The `workspace.Service` is optional for
// legacy tests that pass nil; when nil, the handler falls back to UTC and
// cannot render the partial endpoint's tz-aware bucketing. Production wiring
// in cmd/web/main.go MUST pass it.
func NewHandler(svc *Service, cs *clients.Service, ps *projects.Service, ws *workspace.Service, tpls *templates.Registry, lay *layout.Builder) *Handler {
	return &Handler{svc: svc, clientsSvc: cs, projectsSvc: ps, workspace: ws, tpls: tpls, lay: lay}
}

// Register mounts /reports.
func (h *Handler) Register(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.Handle("GET /reports", protect(http.HandlerFunc(h.list)))
	mux.Handle("GET /reports/partial", protect(http.HandlerFunc(h.partial)))
}

type reportView struct {
	layout.BaseView
	Report           Report
	Preset           string
	From             string
	To               string
	Grouping         string
	ClientID         string
	ProjectID        string
	Billable         string
	Timezone         string
	SortedCurrencies []string // grand-total currencies, ISO-ascending
	ActiveClients    []clients.Client
	ActiveProjects   []projects.Project
}

// filters is the parsed, workspace-scoped filter set shared by /reports and
// /reports/partial. Fields mirror the Filters struct but also include
// presentation state (preset/from/to strings) needed by templates.
type filters struct {
	Preset    string
	Grouping  string
	From      string // yyyy-mm-dd as displayed
	To        string
	ClientID  string
	ProjectID string
	Billable  string // "" | "yes" | "no"
	Range     Range
	Filters   Filters // struct consumed by the service
	Timezone  string  // IANA name; used for preset computation
}

// parseFilters parses the query string for /reports and /reports/partial,
// validates cross-workspace references, and returns the assembled filter
// set. Cross-workspace client_id or project_id yields `ok=false` and a 404
// has already been written to `w`.
func (h *Handler) parseFilters(w http.ResponseWriter, r *http.Request, ws workspace.Workspace) (filters, bool) {
	q := r.URL.Query()
	tz := ws.ReportingTimezone
	if tz == "" {
		tz = "UTC"
	}
	loc, err := loadLocation(tz)
	if err != nil {
		loc = time.UTC
		tz = "UTC"
	}

	preset := q.Get("preset")
	grouping := q.Get("group")
	if grouping != "day" && grouping != "client" && grouping != "project" {
		grouping = "day"
	}

	var rng Range
	if preset != "" && preset != "custom" {
		rng = PresetRange(time.Now(), preset, loc)
	} else if q.Get("from") != "" && q.Get("to") != "" {
		from, errF := time.ParseInLocation("2006-01-02", q.Get("from"), loc)
		to, errT := time.ParseInLocation("2006-01-02", q.Get("to"), loc)
		if errF == nil && errT == nil {
			rng = Range{From: from, To: to}
			preset = "custom"
		}
	}
	if rng.From.IsZero() {
		rng = PresetRange(time.Now(), "this_week", loc)
		if preset == "" {
			preset = "this_week"
		}
	}

	billable := q.Get("billable")
	if billable != "yes" && billable != "no" {
		billable = ""
	}

	clientFilter := q.Get("client_id")
	projectFilter := q.Get("project_id")
	var clientID, projectID uuid.UUID
	if clientFilter != "" {
		id, err := uuid.Parse(clientFilter)
		if err != nil {
			sharedhttp.NotFound(w, r)
			return filters{}, false
		}
		if _, err := h.clientsSvc.Get(r.Context(), ws.ID, id); err != nil {
			sharedhttp.NotFound(w, r)
			return filters{}, false
		}
		clientID = id
	}
	if projectFilter != "" {
		id, err := uuid.Parse(projectFilter)
		if err != nil {
			sharedhttp.NotFound(w, r)
			return filters{}, false
		}
		if _, err := h.projectsSvc.Get(r.Context(), ws.ID, id); err != nil {
			sharedhttp.NotFound(w, r)
			return filters{}, false
		}
		projectID = id
	}

	return filters{
		Preset:    preset,
		Grouping:  grouping,
		From:      rng.From.Format("2006-01-02"),
		To:        rng.To.Format("2006-01-02"),
		ClientID:  clientFilter,
		ProjectID: projectFilter,
		Billable:  billable,
		Range:     rng,
		Filters: Filters{
			ClientID:  clientID,
			ProjectID: projectID,
			Billable:  billable,
		},
		Timezone: tz,
	}, true
}

// loadWorkspace resolves the active workspace. Falls back to a UTC stub
// when the workspace service wasn't wired (legacy tests).
func (h *Handler) loadWorkspace(r *http.Request, wsID, userID uuid.UUID) workspace.Workspace {
	if h.workspace == nil {
		return workspace.Workspace{ID: wsID, ReportingTimezone: "UTC"}
	}
	ws, err := h.workspace.Get(r.Context(), userID, wsID)
	if err != nil {
		return workspace.Workspace{ID: wsID, ReportingTimezone: "UTC"}
	}
	return ws
}

// sortedCurrencyKeys returns the keys of an estimated-by-currency map sorted
// ascending (ISO alpha) for deterministic rendering.
func sortedCurrencyKeys(m map[string]int64) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// insertion sort — small N, no import needed; stable.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	wc := authz.MustFromContext(r.Context())
	ws := h.loadWorkspace(r, wc.WorkspaceID, wc.UserID)

	f, ok := h.parseFilters(w, r, ws)
	if !ok {
		return
	}

	rep, err := h.svc.ReportWithFilters(r.Context(), wc.WorkspaceID, wc.UserID, f.Range, f.Grouping, f.Filters)
	if err != nil {
		http.Error(w, "report failed", http.StatusInternalServerError)
		return
	}

	acs, _ := h.clientsSvc.List(r.Context(), wc.WorkspaceID, true)
	aps, _ := h.projectsSvc.List(r.Context(), wc.WorkspaceID, projects.Filters{IncludeArchived: true})
	base, _ := h.lay.Base(r, "reports")

	_ = h.tpls.Render(w, http.StatusOK, "reports.index", reportView{
		BaseView:         base,
		Report:           rep,
		Preset:           f.Preset,
		From:             f.From,
		To:               f.To,
		Grouping:         f.Grouping,
		ClientID:         f.ClientID,
		ProjectID:        f.ProjectID,
		Billable:         f.Billable,
		Timezone:         f.Timezone,
		SortedCurrencies: sortedCurrencyKeys(rep.Totals.EstimatedByCurrency),
		ActiveClients:    acs,
		ActiveProjects:   aps,
	})
}

// partial returns only the results fragment (#report-results contents).
// Used by HTMX swaps driven from the filter form on /reports.
func (h *Handler) partial(w http.ResponseWriter, r *http.Request) {
	wc := authz.MustFromContext(r.Context())
	ws := h.loadWorkspace(r, wc.WorkspaceID, wc.UserID)

	f, ok := h.parseFilters(w, r, ws)
	if !ok {
		return
	}

	rep, err := h.svc.ReportWithFilters(r.Context(), wc.WorkspaceID, wc.UserID, f.Range, f.Grouping, f.Filters)
	if err != nil {
		http.Error(w, "report failed", http.StatusInternalServerError)
		return
	}

	view := reportView{
		Report:           rep,
		Preset:           f.Preset,
		From:             f.From,
		To:               f.To,
		Grouping:         f.Grouping,
		ClientID:         f.ClientID,
		ProjectID:        f.ProjectID,
		Billable:         f.Billable,
		Timezone:         f.Timezone,
		SortedCurrencies: sortedCurrencyKeys(rep.Totals.EstimatedByCurrency),
	}
	_ = h.tpls.RenderPartial(w, http.StatusOK, "reports.index", "reports.partial.results", view)
}
