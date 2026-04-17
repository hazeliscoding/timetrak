package auth

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"timetrak/internal/shared/csrf"
	"timetrak/internal/shared/session"
	"timetrak/internal/shared/templates"
)

// Handler renders and processes the login/signup/logout endpoints.
type Handler struct {
	svc     *Service
	store   *session.Store
	tpls    *templates.Registry
	limiter *RateLimiter
}

// NewHandler constructs the handler.
func NewHandler(svc *Service, store *session.Store, tpls *templates.Registry, limiter *RateLimiter) *Handler {
	return &Handler{svc: svc, store: store, tpls: tpls, limiter: limiter}
}

// Register wires the auth routes onto the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", h.getLogin)
	mux.HandleFunc("POST /login", h.postLogin)
	mux.HandleFunc("GET /signup", h.getSignup)
	mux.HandleFunc("POST /signup", h.postSignup)
	mux.HandleFunc("POST /logout", h.postLogout)
}

type loginView struct {
	CSRFToken string
	Email     string
	Error     string
}

type signupView struct {
	CSRFToken   string
	Email       string
	DisplayName string
	Error       string
	FieldErrors map[string]string
}

func (h *Handler) getLogin(w http.ResponseWriter, r *http.Request) {
	// Already logged in? Bounce to dashboard.
	if _, ok := session.FromContext(r.Context()); ok {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	_ = h.tpls.Render(w, http.StatusOK, "auth.login", loginView{CSRFToken: csrf.Token(r)})
}

func (h *Handler) postLogin(w http.ResponseWriter, r *http.Request) {
	if !h.limiter.Allow(ClientIP(r)) {
		http.Error(w, "too many attempts, please try again later", http.StatusTooManyRequests)
		return
	}
	email := r.FormValue("email")
	password := r.FormValue("password")

	userID, err := h.svc.Login(r.Context(), email, password)
	if err != nil {
		// Always a generic message; never disclose which field failed.
		_ = h.tpls.Render(w, http.StatusUnauthorized, "auth.login", loginView{
			CSRFToken: csrf.Token(r),
			Email:     email,
			Error:     "Invalid email or password.",
		})
		return
	}
	sess, err := h.store.Create(r.Context(), w, userID)
	if err != nil {
		http.Error(w, "failed to start session", http.StatusInternalServerError)
		return
	}
	// Seed the active workspace from the user's first membership so the dashboard renders scoped data.
	if wsID, err := h.svc.FirstWorkspaceForUser(r.Context(), userID); err == nil && wsID != uuid.Nil {
		_ = h.store.SetActiveWorkspace(r.Context(), sess.ID, wsID)
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *Handler) getSignup(w http.ResponseWriter, r *http.Request) {
	if _, ok := session.FromContext(r.Context()); ok {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	_ = h.tpls.Render(w, http.StatusOK, "auth.signup", signupView{CSRFToken: csrf.Token(r)})
}

func (h *Handler) postSignup(w http.ResponseWriter, r *http.Request) {
	if !h.limiter.Allow(ClientIP(r)) {
		http.Error(w, "too many attempts, please try again later", http.StatusTooManyRequests)
		return
	}
	email := r.FormValue("email")
	displayName := r.FormValue("display_name")
	password := r.FormValue("password")

	res, err := h.svc.Register(r.Context(), email, password, displayName)
	if err != nil {
		view := signupView{
			CSRFToken:   csrf.Token(r),
			Email:       email,
			DisplayName: displayName,
			FieldErrors: map[string]string{},
		}
		switch {
		case errors.Is(err, ErrInvalidEmail):
			view.FieldErrors["email"] = "Enter a valid email address."
		case errors.Is(err, ErrDisplayName):
			view.FieldErrors["display_name"] = "Display name is required."
		case errors.Is(err, ErrWeakPassword):
			view.FieldErrors["password"] = "Use at least 10 characters."
		case errors.Is(err, ErrEmailExists):
			// Generic wording to avoid email-existence disclosure.
			view.Error = "We couldn't create that account. Try a different email or sign in."
		default:
			view.Error = "Something went wrong. Please try again."
		}
		_ = h.tpls.Render(w, http.StatusUnprocessableEntity, "auth.signup", view)
		return
	}
	sess, err := h.store.Create(r.Context(), w, res.UserID)
	if err != nil {
		http.Error(w, "failed to start session", http.StatusInternalServerError)
		return
	}
	_ = h.store.SetActiveWorkspace(r.Context(), sess.ID, res.WorkspaceID)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *Handler) postLogout(w http.ResponseWriter, r *http.Request) {
	if sess, ok := session.FromContext(r.Context()); ok {
		_ = h.store.Destroy(r.Context(), w, sess.ID)
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

