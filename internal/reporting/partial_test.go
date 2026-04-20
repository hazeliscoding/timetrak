package reporting_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"timetrak/internal/clients"
	"timetrak/internal/projects"
	"timetrak/internal/reporting"
	"timetrak/internal/shared/authz"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/testdb"
	"timetrak/internal/web/layout"
	"timetrak/internal/workspace"
)

// TestReportsPartialMatchesFullPageSlot verifies that the body of
// GET /reports/partial is exactly the content of the #report-results
// region in GET /reports for the same query string. The UI relies on this
// for byte-identical swaps.
func TestReportsPartialMatchesFullPageSlot(t *testing.T) {
	pool := testdb.Open(t)
	tpls := testdb.LoadTemplates(t)
	notFound := sharedhttp.NewNotFoundRenderer(tpls)
	sharedhttp.SetGlobalNotFound(notFound.Render)
	authz.SetNotFoundRenderer(notFound.Render)

	f := testdb.SeedAuthzFixture(t, pool)

	authzSvc := authz.NewService(pool.Pool)
	wsSvc := workspace.NewService(pool, authzSvc, nil)
	clientsSvc := clients.NewService(pool)
	projectsSvc := projects.NewService(pool)
	repSvc := reporting.NewService(pool)
	lay := layout.New(pool, wsSvc)
	h := reporting.NewHandler(repSvc, clientsSvc, projectsSvc, wsSvc, tpls, lay)

	mux := http.NewServeMux()
	h.Register(mux, func(next http.Handler) http.Handler { return next })

	// Full page.
	rFull := httptest.NewRequest(http.MethodGet, "/reports?preset=this_week", nil)
	rFull = f.AttachAsUserA(rFull)
	wFull := httptest.NewRecorder()
	mux.ServeHTTP(wFull, rFull)
	if wFull.Result().StatusCode != http.StatusOK {
		t.Fatalf("full page: got %d", wFull.Result().StatusCode)
	}
	fullBody := wFull.Body.String()

	// Partial.
	rPart := httptest.NewRequest(http.MethodGet, "/reports/partial?preset=this_week", nil)
	rPart = f.AttachAsUserA(rPart)
	wPart := httptest.NewRecorder()
	mux.ServeHTTP(wPart, rPart)
	if wPart.Result().StatusCode != http.StatusOK {
		t.Fatalf("partial: got %d", wPart.Result().StatusCode)
	}
	partialBody := strings.TrimSpace(wPart.Body.String())

	// Extract the contents of #report-results from the full page.
	slot := extractSlot(t, fullBody, `id="report-results"`)
	slot = strings.TrimSpace(slot)

	if slot != partialBody {
		t.Fatalf("full-page #report-results slot does not match partial body.\nSLOT:\n%s\n\nPARTIAL:\n%s", slot, partialBody)
	}
}

// TestReportsPartialEmptyHasAriaLive asserts the partial response contains
// the empty-state partial when filters return no rows, and the wrapping
// region (written by the full page) uses aria-live="polite".
func TestReportsPartialEmptyHasAriaLive(t *testing.T) {
	pool := testdb.Open(t)
	tpls := testdb.LoadTemplates(t)
	notFound := sharedhttp.NewNotFoundRenderer(tpls)
	sharedhttp.SetGlobalNotFound(notFound.Render)
	authz.SetNotFoundRenderer(notFound.Render)
	f := testdb.SeedAuthzFixture(t, pool)

	authzSvc := authz.NewService(pool.Pool)
	wsSvc := workspace.NewService(pool, authzSvc, nil)
	clientsSvc := clients.NewService(pool)
	projectsSvc := projects.NewService(pool)
	repSvc := reporting.NewService(pool)
	lay := layout.New(pool, wsSvc)
	h := reporting.NewHandler(repSvc, clientsSvc, projectsSvc, wsSvc, tpls, lay)

	mux := http.NewServeMux()
	h.Register(mux, func(next http.Handler) http.Handler { return next })

	// Full page (empty DB) MUST wrap the region in aria-live.
	r := httptest.NewRequest(http.MethodGet, "/reports?preset=this_week", nil)
	r = f.AttachAsUserA(r)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, `aria-live="polite"`) {
		t.Fatalf("full page empty state missing aria-live=\"polite\" region. Body:\n%s", body)
	}
	if !strings.Contains(body, "No entries match these filters") {
		t.Fatalf("full page empty state missing unified empty partial copy")
	}

	// Partial MUST render the empty-state partial directly.
	rp := httptest.NewRequest(http.MethodGet, "/reports/partial?preset=this_week", nil)
	rp = f.AttachAsUserA(rp)
	wp := httptest.NewRecorder()
	mux.ServeHTTP(wp, rp)
	if !strings.Contains(wp.Body.String(), "No entries match these filters") {
		t.Fatalf("partial empty state missing unified empty partial copy. Body:\n%s", wp.Body.String())
	}
}

// extractSlot returns the inner content of the first <div> whose opening tag
// contains `marker` (e.g. `id="report-results"`). It walks balanced <div>
// tags so nested divs do not truncate the slot.
func extractSlot(t *testing.T, body, marker string) string {
	t.Helper()
	idx := strings.Index(body, marker)
	if idx < 0 {
		t.Fatalf("marker %q not found in body", marker)
	}
	// End of opening tag.
	open := strings.Index(body[idx:], ">")
	if open < 0 {
		t.Fatalf("opening tag end not found after marker")
	}
	start := idx + open + 1
	depth := 1
	cur := start
	for {
		nextOpen := strings.Index(body[cur:], "<div")
		nextClose := strings.Index(body[cur:], "</div>")
		if nextClose < 0 {
			t.Fatalf("closing </div> not found")
		}
		if nextOpen >= 0 && nextOpen < nextClose {
			depth++
			// advance past the opening <div...>
			after := strings.Index(body[cur+nextOpen:], ">")
			if after < 0 {
				t.Fatalf("malformed <div> tag")
			}
			cur = cur + nextOpen + after + 1
			continue
		}
		depth--
		if depth == 0 {
			return body[start : cur+nextClose]
		}
		cur = cur + nextClose + len("</div>")
	}
}
