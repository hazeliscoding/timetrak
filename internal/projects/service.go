// Package projects owns the projects domain: CRUD, archive, and listing.
package projects

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"timetrak/internal/shared/db"
)

// Project is a row in projects.
type Project struct {
	ID              uuid.UUID
	WorkspaceID     uuid.UUID
	ClientID        uuid.UUID
	ClientName      string
	Name            string
	Code            string
	IsArchived      bool
	DefaultBillable bool
	EntryCount      int
}

// Errors.
var (
	ErrNotFound        = errors.New("projects: not found")
	ErrEmptyName       = errors.New("projects: name must not be empty")
	ErrClientArchived  = errors.New("projects: cannot use an archived client as parent")
	ErrClientMismatch  = errors.New("projects: client does not belong to the workspace")
)

// Filters control the List call.
type Filters struct {
	IncludeArchived bool
	ClientID        uuid.UUID // zero = all
}

// Service exposes project use cases.
type Service struct{ pool *db.Pool }

// NewService constructs the service.
func NewService(pool *db.Pool) *Service { return &Service{pool: pool} }

// List returns projects in the workspace, optionally filtering.
func (s *Service) List(ctx context.Context, workspaceID uuid.UUID, f Filters) ([]Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.workspace_id, p.client_id, c.name, p.name,
		       COALESCE(p.code, ''), p.is_archived, p.default_billable,
		       (SELECT count(*) FROM time_entries te
		         WHERE te.workspace_id = p.workspace_id AND te.project_id = p.id)
		FROM projects p
		JOIN clients c ON c.id = p.client_id
		WHERE p.workspace_id = $1
		  AND ($2 OR p.is_archived = false)
		  AND ($3::uuid IS NULL OR p.client_id = $3)
		ORDER BY c.name ASC, p.name ASC
	`, workspaceID, f.IncludeArchived, nullUUID(f.ClientID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Project{}
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.ClientID, &p.ClientName, &p.Name,
			&p.Code, &p.IsArchived, &p.DefaultBillable, &p.EntryCount); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListActive returns non-archived projects (with non-archived clients), used for timer pickers.
func (s *Service) ListActive(ctx context.Context, workspaceID uuid.UUID) ([]Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.workspace_id, p.client_id, c.name, p.name,
		       COALESCE(p.code, ''), p.is_archived, p.default_billable, 0
		FROM projects p
		JOIN clients c ON c.id = p.client_id
		WHERE p.workspace_id = $1 AND p.is_archived = false AND c.is_archived = false
		ORDER BY c.name ASC, p.name ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Project{}
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.ClientID, &p.ClientName, &p.Name,
			&p.Code, &p.IsArchived, &p.DefaultBillable, &p.EntryCount); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Get returns a single project scoped to the workspace.
func (s *Service) Get(ctx context.Context, workspaceID, projectID uuid.UUID) (Project, error) {
	var p Project
	err := s.pool.QueryRow(ctx, `
		SELECT p.id, p.workspace_id, p.client_id, c.name, p.name,
		       COALESCE(p.code, ''), p.is_archived, p.default_billable, 0
		FROM projects p
		JOIN clients c ON c.id = p.client_id
		WHERE p.id = $1 AND p.workspace_id = $2
	`, projectID, workspaceID).Scan(&p.ID, &p.WorkspaceID, &p.ClientID, &p.ClientName, &p.Name,
		&p.Code, &p.IsArchived, &p.DefaultBillable, &p.EntryCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	return p, err
}

// CreateInput groups fields for Create.
type CreateInput struct {
	ClientID        uuid.UUID
	Name            string
	Code            string
	DefaultBillable bool
}

// Create inserts a project after verifying the client is in-workspace and non-archived.
func (s *Service) Create(ctx context.Context, workspaceID uuid.UUID, in CreateInput) (Project, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Code = strings.TrimSpace(in.Code)
	if in.Name == "" {
		return Project{}, ErrEmptyName
	}
	if err := s.verifyClient(ctx, workspaceID, in.ClientID); err != nil {
		return Project{}, err
	}
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, client_id, name, code, default_billable)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5)
		RETURNING id
	`, workspaceID, in.ClientID, in.Name, in.Code, in.DefaultBillable).Scan(&id)
	if err != nil {
		return Project{}, err
	}
	return s.Get(ctx, workspaceID, id)
}

// UpdateInput is the payload for Update.
type UpdateInput struct {
	Name            string
	Code            string
	DefaultBillable bool
}

// Update edits a project in the workspace.
func (s *Service) Update(ctx context.Context, workspaceID, projectID uuid.UUID, in UpdateInput) (Project, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Code = strings.TrimSpace(in.Code)
	if in.Name == "" {
		return Project{}, ErrEmptyName
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE projects SET name = $3, code = NULLIF($4, ''), default_billable = $5, updated_at = now()
		WHERE id = $1 AND workspace_id = $2
	`, projectID, workspaceID, in.Name, in.Code, in.DefaultBillable)
	if err != nil {
		return Project{}, err
	}
	if tag.RowsAffected() == 0 {
		return Project{}, ErrNotFound
	}
	return s.Get(ctx, workspaceID, projectID)
}

// SetArchived toggles the archive flag scoped to workspace.
func (s *Service) SetArchived(ctx context.Context, workspaceID, projectID uuid.UUID, archived bool) (Project, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE projects SET is_archived = $3, updated_at = now()
		WHERE id = $1 AND workspace_id = $2
	`, projectID, workspaceID, archived)
	if err != nil {
		return Project{}, err
	}
	if tag.RowsAffected() == 0 {
		return Project{}, ErrNotFound
	}
	return s.Get(ctx, workspaceID, projectID)
}

func (s *Service) verifyClient(ctx context.Context, workspaceID, clientID uuid.UUID) error {
	var isArchived bool
	var wsID uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT workspace_id, is_archived FROM clients WHERE id = $1`, clientID).
		Scan(&wsID, &isArchived)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrClientMismatch
	}
	if err != nil {
		return err
	}
	if wsID != workspaceID {
		return ErrClientMismatch
	}
	if isArchived {
		return ErrClientArchived
	}
	return nil
}

func nullUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
