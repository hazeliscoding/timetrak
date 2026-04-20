package workspace_test

import (
	"context"
	"errors"
	"testing"

	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/testdb"
	"timetrak/internal/workspace"
)

// TestUpdateReportingTimezoneRoundTrip asserts a tz change persists and is
// returned on the next Get. Also confirms cross-workspace write returns
// ErrForbidden (which handlers translate to HTTP 404).
func TestUpdateReportingTimezoneRoundTrip(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	authzSvc := authz.NewService(pool.Pool)
	svc := workspace.NewService(pool, authzSvc, nil)

	// Default is UTC.
	ws, err := svc.Get(ctx, f.UserA, f.WorkspaceA)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ws.ReportingTimezone != "UTC" {
		t.Fatalf("default tz: got %q want UTC", ws.ReportingTimezone)
	}

	// Valid tz round-trips.
	if err := svc.UpdateReportingTimezone(ctx, f.UserA, f.WorkspaceA, "America/New_York"); err != nil {
		t.Fatalf("update tz: %v", err)
	}
	ws, err = svc.Get(ctx, f.UserA, f.WorkspaceA)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if ws.ReportingTimezone != "America/New_York" {
		t.Fatalf("tz not persisted: got %q", ws.ReportingTimezone)
	}

	// Invalid tz is rejected with typed error.
	err = svc.UpdateReportingTimezone(ctx, f.UserA, f.WorkspaceA, "Not/A/Real/Zone")
	if !errors.Is(err, workspace.ErrInvalidTimezone) {
		t.Fatalf("invalid tz: got %v want ErrInvalidTimezone", err)
	}
	// Empty is also rejected.
	err = svc.UpdateReportingTimezone(ctx, f.UserA, f.WorkspaceA, "")
	if !errors.Is(err, workspace.ErrInvalidTimezone) {
		t.Fatalf("empty tz: got %v want ErrInvalidTimezone", err)
	}
	// Stored tz unchanged after rejected writes.
	ws, _ = svc.Get(ctx, f.UserA, f.WorkspaceA)
	if ws.ReportingTimezone != "America/New_York" {
		t.Fatalf("rejected write mutated stored tz: %q", ws.ReportingTimezone)
	}

	// Cross-workspace write MUST return ErrForbidden (handler maps this to 404).
	err = svc.UpdateReportingTimezone(ctx, f.UserA, f.WorkspaceB, "Europe/Berlin")
	if !errors.Is(err, workspace.ErrForbidden) {
		t.Fatalf("cross-ws tz update: got %v want ErrForbidden", err)
	}
	// W2's tz is unchanged.
	wsB, err := svc.Get(ctx, f.UserB, f.WorkspaceB)
	if err != nil {
		t.Fatalf("get W2: %v", err)
	}
	if wsB.ReportingTimezone != "UTC" {
		t.Fatalf("cross-ws write leaked into W2: %q", wsB.ReportingTimezone)
	}
}
