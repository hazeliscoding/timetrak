package workspace

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/session"
)

// Handler exposes workspace switching.
type Handler struct {
	svc *Service
}

// NewHandler constructs the handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register mounts /workspace/switch.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /workspace/switch", h.postSwitch)
}

func (h *Handler) postSwitch(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.FromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	wsID, err := uuid.Parse(r.FormValue("workspace_id"))
	if err != nil {
		sharedhttp.NotFound(w, r)
		return
	}
	if err := h.svc.SwitchActive(r.Context(), sess.ID, sess.UserID, wsID); err != nil {
		if errors.Is(err, ErrForbidden) {
			sharedhttp.NotFound(w, r)
			return
		}
		http.Error(w, "failed to switch workspace", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
