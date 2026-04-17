// Package clients implements the clients domain: CRUD, archive, and listing,
// all scoped to the active workspace.
package clients

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"timetrak/internal/shared/db"
)

// Client is one row in clients.
type Client struct {
	ID           uuid.UUID
	WorkspaceID  uuid.UUID
	Name         string
	ContactEmail string
	IsArchived   bool
	ProjectCount int
}

// Errors the domain returns.
var (
	ErrNotFound  = errors.New("clients: not found")
	ErrEmptyName = errors.New("clients: name must not be empty")
)

// Service exposes the clients use cases.
type Service struct {
	pool *db.Pool
}

// NewService constructs the service.
func NewService(pool *db.Pool) *Service { return &Service{pool: pool} }

// List returns clients in the workspace (archived excluded unless includeArchived).
func (s *Service) List(ctx context.Context, workspaceID uuid.UUID, includeArchived bool) ([]Client, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.workspace_id, c.name, COALESCE(c.contact_email, ''), c.is_archived,
		       (SELECT count(*) FROM projects p WHERE p.workspace_id = c.workspace_id AND p.client_id = c.id)
		FROM clients c
		WHERE c.workspace_id = $1
		  AND ($2 OR c.is_archived = false)
		ORDER BY c.name ASC
	`, workspaceID, includeArchived)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Client{}
	for rows.Next() {
		var c Client
		if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.Name, &c.ContactEmail, &c.IsArchived, &c.ProjectCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListActive returns non-archived clients, used for project pickers.
func (s *Service) ListActive(ctx context.Context, workspaceID uuid.UUID) ([]Client, error) {
	return s.List(ctx, workspaceID, false)
}

// Get returns a single client in the workspace, or ErrNotFound.
func (s *Service) Get(ctx context.Context, workspaceID, clientID uuid.UUID) (Client, error) {
	var c Client
	err := s.pool.QueryRow(ctx, `
		SELECT id, workspace_id, name, COALESCE(contact_email, ''), is_archived
		FROM clients WHERE id = $1 AND workspace_id = $2
	`, clientID, workspaceID).Scan(&c.ID, &c.WorkspaceID, &c.Name, &c.ContactEmail, &c.IsArchived)
	if errors.Is(err, pgx.ErrNoRows) {
		return Client{}, ErrNotFound
	}
	return c, err
}

// Create inserts a new client in the workspace.
func (s *Service) Create(ctx context.Context, workspaceID uuid.UUID, name, contactEmail string) (Client, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Client{}, ErrEmptyName
	}
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO clients (workspace_id, name, contact_email)
		VALUES ($1, $2, NULLIF($3, ''))
		RETURNING id
	`, workspaceID, name, strings.TrimSpace(contactEmail)).Scan(&id)
	if err != nil {
		return Client{}, err
	}
	return s.Get(ctx, workspaceID, id)
}

// Update edits a client in the workspace.
func (s *Service) Update(ctx context.Context, workspaceID, clientID uuid.UUID, name, contactEmail string) (Client, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Client{}, ErrEmptyName
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE clients SET name = $3, contact_email = NULLIF($4, ''), updated_at = now()
		WHERE id = $1 AND workspace_id = $2
	`, clientID, workspaceID, name, strings.TrimSpace(contactEmail))
	if err != nil {
		return Client{}, err
	}
	if tag.RowsAffected() == 0 {
		return Client{}, ErrNotFound
	}
	return s.Get(ctx, workspaceID, clientID)
}

// SetArchived flips is_archived for a client in the workspace.
func (s *Service) SetArchived(ctx context.Context, workspaceID, clientID uuid.UUID, archived bool) (Client, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE clients SET is_archived = $3, updated_at = now()
		WHERE id = $1 AND workspace_id = $2
	`, clientID, workspaceID, archived)
	if err != nil {
		return Client{}, err
	}
	if tag.RowsAffected() == 0 {
		return Client{}, ErrNotFound
	}
	return s.Get(ctx, workspaceID, clientID)
}
