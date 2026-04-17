// Package layout assembles the common data every app-shell page needs.
//
// Domain handlers pass their specific view struct plus BaseView (embedded)
// so templates can reliably access CurrentUser, Workspaces, ActivePage, etc.
package layout

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"timetrak/internal/shared/csrf"
	"timetrak/internal/shared/db"
	"timetrak/internal/shared/session"
	"timetrak/internal/workspace"
)

// BaseView carries the fields required by layouts/app.html.
type BaseView struct {
	CurrentUser         *User
	ActivePage          string
	Workspaces          []workspace.Workspace
	ActiveWorkspaceID   uuid.UUID
	ActiveWorkspaceName string
	CSRFToken           string
	Flash               []FlashMessage
}

// User is the subset of user data the layout needs.
type User struct {
	ID          uuid.UUID
	Email       string
	DisplayName string
}

// FlashMessage is a one-shot status message.
type FlashMessage struct {
	Kind    string // "success" | "error" | "info"
	Message string
}

// Builder loads layout data for a request.
type Builder struct {
	pool *db.Pool
	wsvc *workspace.Service
}

// New returns a layout builder.
func New(pool *db.Pool, wsvc *workspace.Service) *Builder {
	return &Builder{pool: pool, wsvc: wsvc}
}

// Base returns the layout data for a request. Safe to call from any authenticated handler.
func (b *Builder) Base(r *http.Request, activePage string) (BaseView, error) {
	bv := BaseView{ActivePage: activePage, CSRFToken: csrf.Token(r)}
	sess, ok := session.FromContext(r.Context())
	if !ok {
		return bv, nil
	}
	user, err := b.loadUser(r.Context(), sess.UserID)
	if err != nil {
		return bv, err
	}
	bv.CurrentUser = user
	ws, err := b.wsvc.ListForUser(r.Context(), sess.UserID)
	if err != nil {
		return bv, err
	}
	bv.Workspaces = ws
	if sess.ActiveWorkspaceID != nil {
		bv.ActiveWorkspaceID = *sess.ActiveWorkspaceID
		for _, w := range ws {
			if w.ID == *sess.ActiveWorkspaceID {
				bv.ActiveWorkspaceName = w.Name
				break
			}
		}
	}
	return bv, nil
}

func (b *Builder) loadUser(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := b.pool.QueryRow(ctx, `SELECT id, email, display_name FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Email, &u.DisplayName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
