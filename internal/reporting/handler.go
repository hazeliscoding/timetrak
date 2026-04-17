package reporting

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/clients"
	"timetrak/internal/projects"
	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/session"
	"timetrak/internal/shared/templates"
	"timetrak/internal/web/layout"
)

// Handler serves /reports.
type Handler struct {
	svc         *Service
	clientsSvc  *clients.Service
	projectsSvc *projects.Service
	tpls        *templates.Registry
	lay         *layout.Builder
}

// NewHandler constructs the handler.
func NewHandler(svc *Service, cs *clients.Service, ps *projects.Service, tpls *templates.Registry, lay *layout.Builder) *Handler {
	return &Handler{svc: svc, clientsSvc: cs, projectsSvc: ps, tpls: tpls, lay: lay}
}

// Register mounts /reports.
func (h *Handler) Register(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.Handle("GET /reports", protect(http.HandlerFunc(h.list)))
}

type reportView struct {
	layout.BaseView
	Report         Report
	Preset         string
	From           string
	To             string
	Grouping       string
	ClientID       string
	ProjectID      string
	ActiveClients  []clients.Client
	ActiveProjects []projects.Project
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	wsID := authz.ActiveWorkspace(r.Context())
	sess, _ := session.FromContext(r.Context())

	q := r.URL.Query()
	preset := q.Get("preset")
	grouping := q.Get("group")
	if grouping == "" {
		grouping = "day"
	}
	var rng Range
	if preset != "" && preset != "custom" {
		rng = PresetRange(time.Now(), preset)
	} else if q.Get("from") != "" && q.Get("to") != "" {
		from, errF := time.Parse("2006-01-02", q.Get("from"))
		to, errT := time.Parse("2006-01-02", q.Get("to"))
		if errF == nil && errT == nil {
			rng = Range{From: from.UTC(), To: to.UTC()}
			preset = "custom"
		}
	}
	if rng.From.IsZero() {
		rng = PresetRange(time.Now(), "this_week")
		if preset == "" {
			preset = "this_week"
		}
	}

	rep, err := h.svc.Report(r.Context(), wsID, sess.UserID, rng, grouping)
	if err != nil {
		http.Error(w, "report failed", http.StatusInternalServerError)
		return
	}

	// Optional client/project filters: apply to report by re-calling the aggregation
	// path with filtered scope isn't supported in MVP; we just filter grouped rows in-place.
	clientFilter := q.Get("client_id")
	projectFilter := q.Get("project_id")
	if clientFilter != "" {
		if id, err := uuid.Parse(clientFilter); err == nil {
			rep.ByClient = filterGrouped(rep.ByClient, func(g GroupedBucket) bool { return g.ID == id })
			rep.ByProject = filterGrouped(rep.ByProject, func(g GroupedBucket) bool {
				// Filter projects whose client matches: reload with project list.
				return true
			})
		}
	}
	if projectFilter != "" {
		if id, err := uuid.Parse(projectFilter); err == nil {
			rep.ByProject = filterGrouped(rep.ByProject, func(g GroupedBucket) bool { return g.ID == id })
		}
	}

	acs, _ := h.clientsSvc.List(r.Context(), wsID, true)
	aps, _ := h.projectsSvc.List(r.Context(), wsID, projects.Filters{IncludeArchived: true})
	base, _ := h.lay.Base(r, "reports")

	_ = h.tpls.Render(w, http.StatusOK, "reports.index", reportView{
		BaseView:       base,
		Report:         rep,
		Preset:         preset,
		From:           rng.From.Format("2006-01-02"),
		To:             rng.To.Format("2006-01-02"),
		Grouping:       grouping,
		ClientID:       clientFilter,
		ProjectID:      projectFilter,
		ActiveClients:  acs,
		ActiveProjects: aps,
	})
}

func filterGrouped(in []GroupedBucket, keep func(GroupedBucket) bool) []GroupedBucket {
	out := in[:0]
	for _, g := range in {
		if keep(g) {
			out = append(out, g)
		}
	}
	return out
}
