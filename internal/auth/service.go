package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"timetrak/internal/shared/db"
)

// Errors returned by the auth service.
var (
	ErrInvalidCredentials = errors.New("auth: invalid email or password")
	ErrEmailExists        = errors.New("auth: email already registered")
	ErrInvalidEmail       = errors.New("auth: invalid email address")
	ErrDisplayName        = errors.New("auth: display name must not be empty")
)

var emailRE = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// Service handles user registration, login, and logout.
type Service struct {
	pool *db.Pool
}

// NewService constructs the auth service.
func NewService(pool *db.Pool) *Service { return &Service{pool: pool} }

// RegisterResult describes what Register produced for the caller.
type RegisterResult struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
}

// Register creates a user, a personal workspace, owner membership, and returns the new IDs.
// The cookie/session is established by the handler after Register returns.
func (s *Service) Register(ctx context.Context, email, password, displayName string) (RegisterResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)
	if !emailRE.MatchString(email) {
		return RegisterResult{}, ErrInvalidEmail
	}
	if displayName == "" {
		return RegisterResult{}, ErrDisplayName
	}
	if err := ValidatePassword(password); err != nil {
		return RegisterResult{}, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return RegisterResult{}, err
	}

	var userID, workspaceID uuid.UUID
	err = s.pool.InTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, display_name)
			VALUES ($1, $2, $3)
			RETURNING id
		`, email, hash, displayName).Scan(&userID); err != nil {
			if db.IsUniqueViolation(err) {
				return ErrEmailExists
			}
			return fmt.Errorf("insert user: %w", err)
		}
		slug, err := uniqueWorkspaceSlug(ctx, tx, displayName)
		if err != nil {
			return err
		}
		wsName := displayName + "'s workspace"
		if err := tx.QueryRow(ctx, `
			INSERT INTO workspaces (name, slug)
			VALUES ($1, $2)
			RETURNING id
		`, wsName, slug).Scan(&workspaceID); err != nil {
			return fmt.Errorf("insert workspace: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO workspace_members (workspace_id, user_id, role)
			VALUES ($1, $2, 'owner')
		`, workspaceID, userID); err != nil {
			return fmt.Errorf("insert membership: %w", err)
		}
		return nil
	})
	if err != nil {
		return RegisterResult{}, err
	}
	return RegisterResult{UserID: userID, WorkspaceID: workspaceID}, nil
}

// Login verifies credentials and returns the user id on success.
// Callers MUST display a generic failure message regardless of which error is returned.
func (s *Service) Login(ctx context.Context, email, password string) (uuid.UUID, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var id uuid.UUID
	var hash string
	err := s.pool.QueryRow(ctx, `
		SELECT id, password_hash FROM users WHERE lower(email) = $1
	`, email).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrInvalidCredentials
	}
	if err != nil {
		return uuid.Nil, err
	}
	if err := VerifyPassword(password, hash); err != nil {
		return uuid.Nil, ErrInvalidCredentials
	}
	return id, nil
}

// FirstWorkspaceForUser returns the user's first workspace id (used to seed session active workspace on login).
func (s *Service) FirstWorkspaceForUser(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT workspace_id FROM workspace_members WHERE user_id = $1
		ORDER BY joined_at ASC
		LIMIT 1
	`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil
	}
	return id, err
}

// slugRE strips everything that isn't lowercase alphanumeric or dash.
var slugRE = regexp.MustCompile(`[^a-z0-9-]+`)

func uniqueWorkspaceSlug(ctx context.Context, tx pgx.Tx, displayName string) (string, error) {
	base := slugRE.ReplaceAllString(strings.ToLower(strings.ReplaceAll(displayName, " ", "-")), "")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "workspace"
	}
	for i := 0; i < 50; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i+1)
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE slug = $1)`, candidate).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	// Fallback with a random suffix.
	return fmt.Sprintf("%s-%s", base, uuid.New().String()[:8]), nil
}
