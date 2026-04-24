package tracking_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"timetrak/internal/clients"
	"timetrak/internal/projects"
	"timetrak/internal/rates"
	"timetrak/internal/reporting"
	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/clock"
	"timetrak/internal/shared/db"
	"timetrak/internal/shared/testdb"
	"timetrak/internal/tracking"
	"timetrak/internal/web/layout"
	"timetrak/internal/workspace"
)

// TestStop_IdempotentUnderConcurrency drives two concurrent StopTimer calls
// against the same running entry. Neither may overwrite the other's ended_at;
// stored row MUST equal the value returned to the successful caller(s).
func TestStop_IdempotentUnderConcurrency(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()

	f := testdb.SeedAuthzFixture(t, pool)
	svc := tracking.NewService(pool, clock.System{}, nil)

	if _, err := svc.StartTimer(ctx, f.WorkspaceA, f.UserA, tracking.StartInput{ProjectID: f.ProjectA}); err != nil {
		t.Fatalf("seed start: %v", err)
	}
	// Back-date started_at so duration_seconds is obviously positive.
	if _, err := pool.Exec(ctx, `UPDATE time_entries SET started_at = now() - interval '1 hour' WHERE workspace_id = $1 AND user_id = $2 AND ended_at IS NULL`, f.WorkspaceA, f.UserA); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	var wg sync.WaitGroup
	results := make([]tracking.Entry, 2)
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			e, err := svc.StopTimer(ctx, f.WorkspaceA, f.UserA)
			results[i] = e
			errs[i] = err
		}()
	}
	wg.Wait()

	var successEnded *time.Time
	successes := 0
	for i := 0; i < 2; i++ {
		if errs[i] == nil {
			successes++
			if results[i].EndedAt == nil {
				t.Fatalf("caller %d nil ended_at on success", i)
			}
			if successEnded == nil {
				successEnded = results[i].EndedAt
			} else if !successEnded.Equal(*results[i].EndedAt) {
				t.Fatalf("divergent ended_at: %v vs %v", *successEnded, *results[i].EndedAt)
			}
		} else if !errors.Is(errs[i], tracking.ErrNoActiveTimer) {
			t.Fatalf("caller %d unexpected err: %v", i, errs[i])
		}
	}
	if successes < 1 {
		t.Fatalf("expected >=1 success, got 0")
	}

	var stored time.Time
	if err := pool.QueryRow(ctx, `SELECT ended_at FROM time_entries WHERE workspace_id = $1 AND user_id = $2 ORDER BY started_at DESC LIMIT 1`, f.WorkspaceA, f.UserA).Scan(&stored); err != nil {
		t.Fatalf("select ended_at: %v", err)
	}
	if !stored.Equal(*successEnded) {
		t.Fatalf("stored %v != returned %v", stored, *successEnded)
	}
}

// TestCreateManual_RejectsZeroAndNegativeInterval covers service-layer
// rejection (fast path) plus direct-SQL bypass (CHECK constraint at 23514).
func TestCreateManual_RejectsZeroAndNegativeInterval(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)
	svc := tracking.NewService(pool, clock.System{}, nil)

	now := time.Now().UTC()

	if _, err := svc.CreateManual(ctx, f.WorkspaceA, f.UserA, tracking.ManualInput{
		ProjectID: f.ProjectA, StartedAt: now, EndedAt: now.Add(-time.Hour), IsBillable: true,
	}); !errors.Is(err, tracking.ErrInvalidInterval) {
		t.Fatalf("negative: want ErrInvalidInterval, got %v", err)
	}
	if _, err := svc.CreateManual(ctx, f.WorkspaceA, f.UserA, tracking.ManualInput{
		ProjectID: f.ProjectA, StartedAt: now, EndedAt: now, IsBillable: true,
	}); !errors.Is(err, tracking.ErrInvalidInterval) {
		t.Fatalf("zero: want ErrInvalidInterval, got %v", err)
	}

	// Direct-SQL bypass hits CHECK constraint 23514.
	_, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id, user_id, project_id, started_at, ended_at, duration_seconds, is_billable)
		VALUES ($1, $2, $3, $4, $4, 0, true)
	`, f.WorkspaceA, f.UserA, f.ProjectA, now)
	if err == nil {
		t.Fatalf("direct zero-duration insert unexpectedly succeeded")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Fatalf("want 23514, got %v", err)
	}
	if pgErr.ConstraintName != "chk_time_entries_interval" && pgErr.ConstraintName != "ck_time_entries_range" {
		t.Fatalf("unexpected constraint name %q", pgErr.ConstraintName)
	}
}

// TestCreateManual_RejectsCrossWorkspaceProjectViaFK forges a direct insert
// with (project_id, workspace_id) pointing at a project in another workspace.
// The composite FK must reject with SQLSTATE 23503.
func TestCreateManual_RejectsCrossWorkspaceProjectViaFK(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)

	start := time.Now().UTC()
	end := start.Add(time.Hour)
	_, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id, user_id, project_id, started_at, ended_at, duration_seconds, is_billable)
		VALUES ($1, $2, $3, $4, $5, 3600, true)
	`, f.WorkspaceA, f.UserA, f.ProjectB /* lives in W2 */, start, end)
	if err == nil {
		t.Fatalf("cross-workspace insert unexpectedly succeeded")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("want 23503, got %v", err)
	}
	if pgErr.ConstraintName != "time_entries_project_workspace_fk" {
		t.Fatalf("unexpected constraint name %q", pgErr.ConstraintName)
	}
}

// TestEdit_RejectsInvertedAndZeroInterval exercises the service-layer
// validation path on Edit.
func TestEdit_RejectsInvertedAndZeroInterval(t *testing.T) {
	pool := testdb.Open(t)
	ctx := context.Background()
	f := testdb.SeedAuthzFixture(t, pool)
	svc := tracking.NewService(pool, clock.System{}, nil)

	start := time.Now().UTC().Add(-2 * time.Hour)
	end := start.Add(time.Hour)
	entry, err := svc.CreateManual(ctx, f.WorkspaceA, f.UserA, tracking.ManualInput{
		ProjectID: f.ProjectA, StartedAt: start, EndedAt: end, IsBillable: true,
	})
	if err != nil {
		t.Fatalf("seed manual: %v", err)
	}

	if _, err := svc.Edit(ctx, f.WorkspaceA, f.UserA, entry.ID, tracking.ManualInput{
		ProjectID: f.ProjectA, StartedAt: end, EndedAt: start, IsBillable: true,
	}); !errors.Is(err, tracking.ErrInvalidInterval) {
		t.Fatalf("inverted: want ErrInvalidInterval, got %v", err)
	}
	if _, err := svc.Edit(ctx, f.WorkspaceA, f.UserA, entry.ID, tracking.ManualInput{
		ProjectID: f.ProjectA, StartedAt: start, EndedAt: start, IsBillable: true,
	}); !errors.Is(err, tracking.ErrInvalidInterval) {
		t.Fatalf("zero: want ErrInvalidInterval, got %v", err)
	}
}

// TestStart_ConcurrentReturns409WithTaxonomy drives a sequential
// start-then-start via the HTTP handler. The second call MUST return 409
// with data-tracking-error-code="tracking.active_timer" and MUST NOT emit
// HX-Trigger events.
func TestStart_ConcurrentReturns409WithTaxonomy(t *testing.T) {
	pool := testdb.Open(t)
	f := testdb.SeedAuthzFixture(t, pool)
	mux, _ := buildTrackingTestHandler(t, pool)

	req1 := buildFormPost(t, "/timer/start", url.Values{"project_id": {f.ProjectA.String()}})
	req1 = f.AttachAsUserA(req1)
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first start: want 200, got %d body=%s", rec1.Code, rec1.Body.String())
	}

	req2 := buildFormPost(t, "/timer/start", url.Values{"project_id": {f.ProjectA.String()}})
	req2 = f.AttachAsUserA(req2)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("second start: want 409, got %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), `data-tracking-error-code="tracking.active_timer"`) {
		t.Fatalf("missing taxonomy code in body: %s", rec2.Body.String())
	}
	if rec2.Header().Get("HX-Trigger") != "" {
		t.Fatalf("error response emitted HX-Trigger: %q", rec2.Header().Get("HX-Trigger"))
	}
}

// TestStop_NoRunningTimerReturns409WithTaxonomy sends a stop with nothing
// running; handler must return 409 + tracking.no_active_timer.
func TestStop_NoRunningTimerReturns409WithTaxonomy(t *testing.T) {
	pool := testdb.Open(t)
	f := testdb.SeedAuthzFixture(t, pool)
	mux, _ := buildTrackingTestHandler(t, pool)

	req := buildFormPost(t, "/timer/stop", url.Values{})
	req = f.AttachAsUserA(req)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `data-tracking-error-code="tracking.no_active_timer"`) {
		t.Fatalf("missing taxonomy code: %s", rec.Body.String())
	}
}

// TestHandler_LogsStructuredErrorKindOnTaxonomyFailure confirms every taxonomy
// response emits a warn line with tracking.error_kind, workspace_id, user_id.
func TestHandler_LogsStructuredErrorKindOnTaxonomyFailure(t *testing.T) {
	pool := testdb.Open(t)
	f := testdb.SeedAuthzFixture(t, pool)

	var buf bytes.Buffer
	capLogger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	mux, th := buildTrackingTestHandler(t, pool)
	th.SetLogger(capLogger)

	req := buildFormPost(t, "/timer/stop", url.Values{})
	req = f.AttachAsUserA(req)
	mux.ServeHTTP(httptest.NewRecorder(), req)

	out := buf.String()
	for _, want := range []string{
		`"tracking.error_kind":"tracking.no_active_timer"`,
		`"workspace_id":"` + f.WorkspaceA.String() + `"`,
		`"user_id":"` + f.UserA.String() + `"`,
		`"level":"WARN"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("log missing %q; got: %s", want, out)
		}
	}
}

// buildTrackingTestHandler wires a real tracking handler against the real
// pool plus in-process templates. Tests attach the workspace context via
// f.AttachAsUserA before invoking the returned http.Handler; auth middleware
// is bypassed (the tracking handler still reads authz.MustFromContext).
func buildTrackingTestHandler(t *testing.T, pool *db.Pool) (http.Handler, *tracking.Handler) {
	t.Helper()
	tpls := testdb.LoadTemplates(t)
	clientsSvc := clients.NewService(pool)
	projectsSvc := projects.NewService(pool)
	ratesSvc := rates.NewService(pool)
	reportSvc := reporting.NewService(pool)
	trackingSvc := tracking.NewService(pool, clock.System{}, ratesSvc)
	// wsSvc now receives a real authz service because tracking handlers
	// look up ReportingTimezone via wsSvc.Get, which invokes authz.IsMember.
	// See humanize-datetime-inputs change.
	authzSvc := authz.NewService(pool.Pool)
	wsSvc := workspace.NewService(pool, authzSvc, nil)
	lay := layout.New(pool, wsSvc)
	th := tracking.NewHandler(trackingSvc, projectsSvc, clientsSvc, reportSvc, wsSvc, tpls, lay)

	mux := http.NewServeMux()
	th.Register(mux, func(next http.Handler) http.Handler { return next })
	return mux, th
}

func buildFormPost(_ *testing.T, path string, form url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}
