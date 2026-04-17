package projects_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"timetrak/internal/shared/db"
	"timetrak/internal/shared/testdb"
)

// TestProjectsClientWorkspaceFK_RejectsMismatch verifies migration 0012:
// a raw INSERT into projects whose workspace_id disagrees with the
// referenced client's workspace_id MUST fail with a foreign-key
// referential-integrity error.
func TestProjectsClientWorkspaceFK_RejectsMismatch(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()

	w1 := uuid.New()
	w2 := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO workspaces (id, name, slug) VALUES
			($1, 'W1', $3),
			($2, 'W2', $4)
	`, w1, w2, w1.String(), w2.String()); err != nil {
		t.Fatalf("insert workspaces: %v", err)
	}

	// Client lives in W1.
	clientID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO clients (id, workspace_id, name) VALUES ($1, $2, 'Acme')
	`, clientID, w1); err != nil {
		t.Fatalf("insert client: %v", err)
	}

	// Attempt to insert a project that points at the W1 client but claims
	// to live in W2. The composite FK MUST reject this.
	_, err := pool.Exec(ctx, `
		INSERT INTO projects (id, workspace_id, client_id, name)
		VALUES (gen_random_uuid(), $1, $2, 'Bad Project')
	`, w2, clientID)
	if err == nil {
		t.Fatalf("expected referential integrity error, got nil — composite FK is missing or broken")
	}
	if !db.IsForeignKeyViolation(err) {
		t.Fatalf("expected foreign_key_violation (SQLSTATE 23503), got: %v", err)
	}
}

// TestProjectsClientWorkspaceFK_AcceptsMatch sanity-checks that a
// consistent insert still succeeds.
func TestProjectsClientWorkspaceFK_AcceptsMatch(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()

	w1 := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id, name, slug) VALUES ($1, 'W1', $2)`,
		w1, w1.String()); err != nil {
		t.Fatal(err)
	}
	clientID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO clients (id, workspace_id, name) VALUES ($1, $2, 'Acme')`,
		clientID, w1); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO projects (id, workspace_id, client_id, name)
		VALUES (gen_random_uuid(), $1, $2, 'Good Project')
	`, w1, clientID); err != nil {
		t.Fatalf("expected consistent insert to succeed, got: %v", err)
	}
}
