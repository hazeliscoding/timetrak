package tracking_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"timetrak/internal/shared/testdb"
)

// TestTimezoneRoundTrip_LosAngeles covers the full write/read cycle for
// a non-UTC workspace:
//
//   - Create a manual entry with local clock times in America/Los_Angeles.
//   - Confirm the stored UTC is shifted by the PDT offset.
//   - Open the edit form; confirm the prefilled date+time inputs render
//     the original local clock times, not UTC.
//
// This is the flagship scenario the humanize-datetime-inputs change
// exists to make correct. See
// openspec/specs/tracking/spec.md (Datetime input parse and display is
// workspace-timezone-aware).
func TestTimezoneRoundTrip_LosAngeles(t *testing.T) {
	pool := testdb.Open(t)
	f := testdb.SeedAuthzFixture(t, pool)
	mux, _ := buildTrackingTestHandler(t, pool)

	// Set WorkspaceA's reporting tz to America/Los_Angeles.
	if _, err := pool.Exec(context.Background(),
		`UPDATE workspaces SET reporting_timezone = $1 WHERE id = $2`,
		"America/Los_Angeles", f.WorkspaceA); err != nil {
		t.Fatalf("set tz: %v", err)
	}

	// Create a manual entry with local clock times.
	form := url.Values{
		"project_id":  {f.ProjectA.String()},
		"date":        {"2026-04-24"},
		"start_time":  {"09:00"},
		"end_time":    {"10:00"},
		"is_billable": {"on"},
	}
	req := buildFormPost(t, "/time-entries", form)
	req = f.AttachAsUserA(req)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create manual: want 303, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Confirm the stored UTC is PDT-shifted. 2026-04-24 09:00 PDT = 16:00Z.
	var startedAt, endedAt time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT started_at, ended_at FROM time_entries WHERE workspace_id = $1 ORDER BY started_at DESC LIMIT 1`,
		f.WorkspaceA).Scan(&startedAt, &endedAt); err != nil {
		t.Fatalf("fetch stored entry: %v", err)
	}
	wantStart, _ := time.Parse(time.RFC3339, "2026-04-24T16:00:00Z")
	wantEnd, _ := time.Parse(time.RFC3339, "2026-04-24T17:00:00Z")
	if !startedAt.Equal(wantStart) {
		t.Fatalf("started_at: got %s, want %s", startedAt.Format(time.RFC3339), wantStart.Format(time.RFC3339))
	}
	if !endedAt.Equal(wantEnd) {
		t.Fatalf("ended_at: got %s, want %s", endedAt.Format(time.RFC3339), wantEnd.Format(time.RFC3339))
	}

	// Find the entry id for the edit-form read-back step.
	var entryID uuid.UUID
	if err := pool.QueryRow(context.Background(),
		`SELECT id FROM time_entries WHERE workspace_id = $1 ORDER BY started_at DESC LIMIT 1`,
		f.WorkspaceA).Scan(&entryID); err != nil {
		t.Fatalf("fetch entry id: %v", err)
	}

	// Open the edit form and confirm the prefilled values render in LA local.
	editReq, _ := http.NewRequest(http.MethodGet, "/time-entries/"+entryID.String()+"/edit", nil)
	editReq = f.AttachAsUserA(editReq)
	editRec := httptest.NewRecorder()
	mux.ServeHTTP(editRec, editReq)
	if editRec.Code != http.StatusOK {
		t.Fatalf("edit form: want 200, got %d", editRec.Code)
	}
	body := editRec.Body.String()

	// Prefilled date must be 2026-04-24 in LA local; prefilled time 09:00.
	// We assert on the value="..." attribute the template emits for each
	// of the four new inputs.
	for _, want := range []string{
		`name="start_date"`, `value="2026-04-24"`,
		`name="start_time"`, `value="09:00"`,
		`name="end_date"`,
		`name="end_time"`, `value="10:00"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("edit form missing %q; body=%s", want, body)
		}
	}

	// And — critically — must NOT contain the old raw-ISO readback.
	if strings.Contains(body, "2026-04-24T16:00:00Z") {
		t.Fatalf("edit form leaked raw UTC ISO into prefill; body=%s", body)
	}
}

// TestTimezoneRoundTrip_UTCDefault confirms the UTC-default workspace
// behaves exactly as it always has: local clock == stored UTC.
func TestTimezoneRoundTrip_UTCDefault(t *testing.T) {
	pool := testdb.Open(t)
	f := testdb.SeedAuthzFixture(t, pool)
	mux, _ := buildTrackingTestHandler(t, pool)

	form := url.Values{
		"project_id":  {f.ProjectA.String()},
		"date":        {"2026-04-24"},
		"start_time":  {"09:00"},
		"end_time":    {"10:00"},
		"is_billable": {"on"},
	}
	req := buildFormPost(t, "/time-entries", form)
	req = f.AttachAsUserA(req)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create manual: want 303, got %d body=%s", rec.Code, rec.Body.String())
	}

	var startedAt, endedAt time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT started_at, ended_at FROM time_entries WHERE workspace_id = $1 ORDER BY started_at DESC LIMIT 1`,
		f.WorkspaceA).Scan(&startedAt, &endedAt); err != nil {
		t.Fatalf("fetch stored: %v", err)
	}
	wantStart, _ := time.Parse(time.RFC3339, "2026-04-24T09:00:00Z")
	wantEnd, _ := time.Parse(time.RFC3339, "2026-04-24T10:00:00Z")
	if !startedAt.Equal(wantStart) || !endedAt.Equal(wantEnd) {
		t.Fatalf("UTC default drift: got %s..%s", startedAt.Format(time.RFC3339), endedAt.Format(time.RFC3339))
	}
}

// TestTimezoneInvalidDateRejected covers the missing-required-field path
// the new spec mandates.
func TestTimezoneInvalidDateRejected(t *testing.T) {
	pool := testdb.Open(t)
	f := testdb.SeedAuthzFixture(t, pool)
	mux, _ := buildTrackingTestHandler(t, pool)

	// Seed a completed entry to edit.
	entryID := uuid.New()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO time_entries (id, workspace_id, user_id, project_id, started_at, ended_at, duration_seconds, is_billable)
		VALUES ($1, $2, $3, $4, $5, $6, 3600, true)
	`, entryID, f.WorkspaceA, f.UserA, f.ProjectA,
		time.Now().UTC().Add(-2*time.Hour), time.Now().UTC().Add(-time.Hour)); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Submit the edit form with a malformed date.
	form := url.Values{
		"project_id": {f.ProjectA.String()},
		"start_date": {"not-a-date"},
		"start_time": {"09:00"},
		"end_date":   {"2026-04-24"},
		"end_time":   {"10:00"},
	}
	req := httptest.NewRequest(http.MethodPatch,
		"/time-entries/"+entryID.String(),
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = f.AttachAsUserA(req)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422 on invalid date; got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `data-tracking-error-code="tracking.invalid_interval"`) {
		t.Fatalf("missing tracking.invalid_interval in body: %s", rec.Body.String())
	}
}

// TestTimezoneLegacyISOFieldsRejected — the handler MUST NOT silently
// fall back to accepting raw ISO `started_at` / `ended_at` when the
// new split fields are absent. Per spec "Raw ISO 8601 text fields are
// not accepted on edit."
func TestTimezoneLegacyISOFieldsRejected(t *testing.T) {
	pool := testdb.Open(t)
	f := testdb.SeedAuthzFixture(t, pool)
	mux, _ := buildTrackingTestHandler(t, pool)

	entryID := uuid.New()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO time_entries (id, workspace_id, user_id, project_id, started_at, ended_at, duration_seconds, is_billable)
		VALUES ($1, $2, $3, $4, $5, $6, 3600, true)
	`, entryID, f.WorkspaceA, f.UserA, f.ProjectA,
		time.Now().UTC().Add(-2*time.Hour), time.Now().UTC().Add(-time.Hour)); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Submit with only the legacy fields.
	form := url.Values{
		"project_id": {f.ProjectA.String()},
		"started_at": {time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)},
		"ended_at":   {time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)},
	}
	req := httptest.NewRequest(http.MethodPatch,
		"/time-entries/"+entryID.String(),
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = f.AttachAsUserA(req)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("legacy-only submit: want 422 (missing start_date), got %d body=%s", rec.Code, rec.Body.String())
	}
}
