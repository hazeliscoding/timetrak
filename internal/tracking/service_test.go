package tracking_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/shared/clock"
	"timetrak/internal/shared/testdb"
	"timetrak/internal/tracking"
)

func TestConcurrentStartReturnsExactlyOneRunningEntry(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()

	var userID, workspaceID, clientID, projectID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('t@e','x','T') RETURNING id`).Scan(&userID)
	_ = pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug) VALUES ('W','w-track') RETURNING id`).Scan(&workspaceID)
	_, _ = pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID)
	_ = pool.QueryRow(ctx, `INSERT INTO clients (workspace_id, name) VALUES ($1, 'A') RETURNING id`, workspaceID).Scan(&clientID)
	_ = pool.QueryRow(ctx, `INSERT INTO projects (workspace_id, client_id, name) VALUES ($1,$2,'Web') RETURNING id`, workspaceID, clientID).Scan(&projectID)

	svc := tracking.NewService(pool, clock.System{}, nil)

	const goroutines = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	var successes, conflicts, others int
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.StartTimer(ctx, workspaceID, userID, tracking.StartInput{ProjectID: projectID})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				successes++
			case errors.Is(err, tracking.ErrActiveTimerExists):
				conflicts++
			default:
				others++
			}
		}()
	}
	wg.Wait()

	if successes != 1 {
		t.Fatalf("expected exactly one successful start, got %d (conflicts=%d others=%d)", successes, conflicts, others)
	}
	if others != 0 {
		t.Fatalf("unexpected errors: %d", others)
	}
	if conflicts != goroutines-1 {
		t.Fatalf("expected %d conflicts, got %d", goroutines-1, conflicts)
	}
}

func TestStopTimerComputesDuration(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()

	var userID, workspaceID, clientID, projectID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('s@e','x','S') RETURNING id`).Scan(&userID)
	_ = pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug) VALUES ('W','w-stop') RETURNING id`).Scan(&workspaceID)
	_, _ = pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID)
	_ = pool.QueryRow(ctx, `INSERT INTO clients (workspace_id, name) VALUES ($1, 'A') RETURNING id`, workspaceID).Scan(&clientID)
	_ = pool.QueryRow(ctx, `INSERT INTO projects (workspace_id, client_id, name) VALUES ($1,$2,'Web') RETURNING id`, workspaceID, clientID).Scan(&projectID)

	// StopTimer now uses the DB server clock (now()) to set ended_at and to
	// compute duration_seconds, so we seed started_at directly on the DB
	// to make the expected duration deterministic. The mutableClock is kept
	// in the ctor for parity with other tests even though its Now() is no
	// longer consulted during stop.
	svc := tracking.NewService(pool, &mutableClock{t: time.Now().UTC()}, nil)
	if _, err := svc.StartTimer(ctx, workspaceID, userID, tracking.StartInput{ProjectID: projectID}); err != nil {
		t.Fatal(err)
	}
	// Back-date the running entry by exactly 90 minutes against the DB clock.
	if _, err := pool.Exec(ctx, `UPDATE time_entries SET started_at = now() - interval '90 minutes' WHERE workspace_id = $1 AND user_id = $2 AND ended_at IS NULL`, workspaceID, userID); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	e, err := svc.StopTimer(ctx, workspaceID, userID)
	if err != nil {
		t.Fatal(err)
	}
	// Allow a small slack (DB query latency) around the expected 5400s.
	if e.DurationSeconds < 5398 || e.DurationSeconds > 5405 {
		t.Fatalf("expected ~5400s (+/-5), got %d", e.DurationSeconds)
	}

	// Stopping again within the 5s idempotency window returns the
	// already-stopped entry unchanged (idempotent path). Outside the window
	// a fresh stop would return ErrNoActiveTimer; we cover that separately
	// in TestStop_NoRunningTimerReturns409WithTaxonomy.
	e2, err := svc.StopTimer(ctx, workspaceID, userID)
	if err != nil {
		t.Fatalf("second stop expected idempotent success, got %v", err)
	}
	if e2.EndedAt == nil || !e2.EndedAt.Equal(*e.EndedAt) {
		t.Fatalf("second stop diverged: %v vs %v", e2.EndedAt, e.EndedAt)
	}
}

func TestManualEntryInvalidRange(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()

	var userID, workspaceID, clientID, projectID uuid.UUID
	_ = pool.QueryRow(ctx, `INSERT INTO users (email,password_hash,display_name) VALUES ('m@e','x','M') RETURNING id`).Scan(&userID)
	_ = pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug) VALUES ('W','w-m') RETURNING id`).Scan(&workspaceID)
	_, _ = pool.Exec(ctx, `INSERT INTO workspace_members (workspace_id,user_id,role) VALUES ($1,$2,'owner')`, workspaceID, userID)
	_ = pool.QueryRow(ctx, `INSERT INTO clients (workspace_id, name) VALUES ($1, 'A') RETURNING id`, workspaceID).Scan(&clientID)
	_ = pool.QueryRow(ctx, `INSERT INTO projects (workspace_id, client_id, name) VALUES ($1,$2,'Web') RETURNING id`, workspaceID, clientID).Scan(&projectID)

	svc := tracking.NewService(pool, clock.System{}, nil)
	start := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
	end := start.Add(-time.Hour)
	if _, err := svc.CreateManual(ctx, workspaceID, userID, tracking.ManualInput{
		ProjectID: projectID, StartedAt: start, EndedAt: end, IsBillable: true,
	}); !errors.Is(err, tracking.ErrInvalidRange) {
		t.Fatalf("expected ErrInvalidRange, got %v", err)
	}
}

type mutableClock struct{ t time.Time }

func (m *mutableClock) Now() time.Time { return m.t }
