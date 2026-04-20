package rates_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"timetrak/internal/clients"
	"timetrak/internal/projects"
	"timetrak/internal/rates"
	"timetrak/internal/shared/authz"
	"timetrak/internal/shared/db"
	sharedhttp "timetrak/internal/shared/http"
	"timetrak/internal/shared/testdb"
	"timetrak/internal/web/layout"
	"timetrak/internal/workspace"
)

// setupHTMX wires the full rates handler against the shared authz fixture
// without middleware (protect is a no-op), mirroring authz_test.go. CSRF
// middleware is not applied in this test harness.
func setupHTMX(t *testing.T) (*http.ServeMux, testdb.AuthzFixture, *rates.Service, *db.Pool) {
	t.Helper()
	pool := testdb.Open(t)
	tpls := testdb.LoadTemplates(t)
	notFound := sharedhttp.NewNotFoundRenderer(tpls)
	sharedhttp.SetGlobalNotFound(notFound.Render)
	authz.SetNotFoundRenderer(notFound.Render)

	f := testdb.SeedAuthzFixture(t, pool)

	clientsSvc := clients.NewService(pool)
	projectsSvc := projects.NewService(pool)
	ratesSvc := rates.NewService(pool)
	authzSvc := authz.NewService(pool.Pool)
	wsSvc := workspace.NewService(pool, authzSvc, nil)
	lay := layout.New(pool, wsSvc)
	h := rates.NewHandler(ratesSvc, clientsSvc, projectsSvc, tpls, lay)

	mux := http.NewServeMux()
	h.Register(mux, func(next http.Handler) http.Handler { return next })

	return mux, f, ratesSvc, pool
}

// hxRequest builds an HTMX-flagged form POST/GET request.
func hxRequest(t *testing.T, method, path string, body url.Values) *http.Request {
	t.Helper()
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, strings.NewReader(body.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.Header.Set("HX-Request", "true")
	return r
}

func TestHX_EditRowAndRow_NotFoundCrossWorkspace(t *testing.T) {
	mux, f, ratesSvc, _ := setupHTMX(t)

	// Seed a rule in W2.
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ruleB, err := ratesSvc.Create(context.Background(), f.WorkspaceB, rates.Input{
		ClientID: f.ClientB, CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: from,
	})
	if err != nil {
		t.Fatalf("seed W2 rule: %v", err)
	}
	// And one in W1 so we have an in-workspace happy path.
	ruleA, err := ratesSvc.Create(context.Background(), f.WorkspaceA, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 5000, EffectiveFrom: from,
	})
	if err != nil {
		t.Fatalf("seed W1 rule: %v", err)
	}

	run := func(name, path string) {
		t.Run(name+"-ws", func(t *testing.T) {
			r := hxRequest(t, http.MethodGet, strings.ReplaceAll(path, "{id}", ruleA.String()), nil)
			r = f.AttachAsUserA(r)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Result().StatusCode != http.StatusOK {
				t.Fatalf("%s in-workspace: got %d want 200", name, w.Result().StatusCode)
			}
			if !strings.Contains(w.Body.String(), `id="rate-`+ruleA.String()+`"`) {
				t.Fatalf("%s in-workspace: response missing rate_row for own rule", name)
			}
		})
		t.Run(name+"-cross", func(t *testing.T) {
			r := hxRequest(t, http.MethodGet, strings.ReplaceAll(path, "{id}", ruleB.String()), nil)
			r = f.AttachAsUserA(r)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Result().StatusCode != http.StatusNotFound {
				t.Fatalf("%s cross-workspace: got %d want 404", name, w.Result().StatusCode)
			}
		})
	}
	run("edit", "/rates/{id}/edit")
	run("row", "/rates/{id}/row")
}

func TestHX_CreateSuccess_EmitsRatesChangedAndRefreshedForm(t *testing.T) {
	mux, f, _, _ := setupHTMX(t)

	body := url.Values{
		"scope":          {"workspace"},
		"currency_code":  {"USD"},
		"hourly_decimal": {"100"},
		"effective_from": {time.Now().UTC().Format("2006-01-02")},
	}
	r := hxRequest(t, http.MethodPost, "/rates", body)
	r = f.AttachAsUserA(r)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want 200", w.Result().StatusCode)
	}
	trigger := w.Result().Header.Get("HX-Trigger")
	if !strings.Contains(trigger, "rates-changed") {
		t.Fatalf("HX-Trigger: got %q want to contain rates-changed", trigger)
	}
	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, `id="rates-table"`) {
		t.Fatalf("body missing rates_table partial")
	}
	if !strings.Contains(bodyStr, `id="rate-form"`) || !strings.Contains(bodyStr, `hx-swap-oob="true"`) {
		t.Fatalf("body missing OOB rate_form reset")
	}
}

func TestHX_UpdateSuccess_ReturnsRowDisplayAndEmitsRatesChanged(t *testing.T) {
	mux, f, ratesSvc, _ := setupHTMX(t)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ruleID, err := ratesSvc.Create(context.Background(), f.WorkspaceA, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: from,
	})
	if err != nil {
		t.Fatal(err)
	}
	newTo := from.Add(180 * 24 * time.Hour).Format("2006-01-02")
	body := url.Values{
		"scope":          {"workspace"},
		"currency_code":  {"USD"},
		"hourly_decimal": {"100"},
		"effective_from": {from.Format("2006-01-02")},
		"effective_to":   {newTo},
	}
	r := hxRequest(t, http.MethodPost, "/rates/"+ruleID.String(), body)
	r = f.AttachAsUserA(r)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want 200", w.Result().StatusCode)
	}
	if trig := w.Result().Header.Get("HX-Trigger"); !strings.Contains(trig, "rates-changed") {
		t.Fatalf("HX-Trigger: got %q want rates-changed", trig)
	}
	body2 := w.Body.String()
	if !strings.Contains(body2, `id="rate-`+ruleID.String()+`"`) {
		t.Fatalf("body missing rate_row for updated rule")
	}
	// Display mode has no hx-post="/rates/{id}" form for edit body.
	if strings.Contains(body2, `hx-post="/rates/`+ruleID.String()+`"`) {
		t.Fatalf("body should be in display mode, but contains the edit form hx-post")
	}
}

func TestHX_UpdateReferencedRule_Conflict_NoRatesChanged(t *testing.T) {
	mux, f, ratesSvc, pool := setupHTMX(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ruleID, err := ratesSvc.Create(ctx, f.WorkspaceA, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: from,
	})
	if err != nil {
		t.Fatal(err)
	}
	feb := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable,rate_rule_id,hourly_rate_minor,currency_code)
		VALUES ($1,$2,$3,$4,$5,3600,true,$6,10000,'USD')
	`, f.WorkspaceA, f.UserA, f.ProjectA, feb, feb.Add(time.Hour), ruleID); err != nil {
		t.Fatal(err)
	}

	// Attempt to change the amount (disallowed for referenced rule).
	body := url.Values{
		"scope":          {"workspace"},
		"currency_code":  {"USD"},
		"hourly_decimal": {"200"}, // changed from 100
		"effective_from": {from.Format("2006-01-02")},
	}
	r := hxRequest(t, http.MethodPost, "/rates/"+ruleID.String(), body)
	r = f.AttachAsUserA(r)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusConflict {
		t.Fatalf("status: got %d want 409", w.Result().StatusCode)
	}
	if trig := w.Result().Header.Get("HX-Trigger"); strings.Contains(trig, "rates-changed") {
		t.Fatalf("HX-Trigger should NOT contain rates-changed, got %q", trig)
	}
	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, `id="rate-`+ruleID.String()+`"`) {
		t.Fatalf("body missing rate_row for conflicting rule")
	}
	if !strings.Contains(bodyStr, `hx-post="/rates/`+ruleID.String()+`"`) {
		t.Fatalf("body should be rate_row in edit mode (has hx-post form)")
	}
	if !strings.Contains(bodyStr, `role="alert"`) {
		t.Fatalf("body should include inline error with role=alert")
	}

	// Verify the stored rule amount is unchanged.
	var storedMinor int64
	if err := pool.QueryRow(ctx, `SELECT hourly_rate_minor FROM rate_rules WHERE id=$1`, ruleID).Scan(&storedMinor); err != nil {
		t.Fatal(err)
	}
	if storedMinor != 10000 {
		t.Fatalf("rule was mutated: got %d want 10000", storedMinor)
	}
}

func TestHX_DeleteUnreferenced_RefreshTableEmitsRatesChanged(t *testing.T) {
	mux, f, ratesSvc, _ := setupHTMX(t)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ruleID, err := ratesSvc.Create(context.Background(), f.WorkspaceA, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: from,
	})
	if err != nil {
		t.Fatal(err)
	}

	r := hxRequest(t, http.MethodPost, "/rates/"+ruleID.String()+"/delete", url.Values{})
	r = f.AttachAsUserA(r)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want 200", w.Result().StatusCode)
	}
	if trig := w.Result().Header.Get("HX-Trigger"); !strings.Contains(trig, "rates-changed") {
		t.Fatalf("HX-Trigger: got %q want rates-changed", trig)
	}
	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, `id="rates-table"`) {
		t.Fatalf("body missing rates_table partial")
	}
	if strings.Contains(bodyStr, `id="rate-`+ruleID.String()+`"`) {
		t.Fatalf("body should not contain the deleted row")
	}
}

func TestHX_DeleteReferenced_Conflict_NoRatesChanged(t *testing.T) {
	mux, f, ratesSvc, pool := setupHTMX(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ruleID, err := ratesSvc.Create(ctx, f.WorkspaceA, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: from,
	})
	if err != nil {
		t.Fatal(err)
	}
	feb := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable,rate_rule_id,hourly_rate_minor,currency_code)
		VALUES ($1,$2,$3,$4,$5,3600,true,$6,10000,'USD')
	`, f.WorkspaceA, f.UserA, f.ProjectA, feb, feb.Add(time.Hour), ruleID); err != nil {
		t.Fatal(err)
	}

	r := hxRequest(t, http.MethodPost, "/rates/"+ruleID.String()+"/delete", url.Values{})
	r = f.AttachAsUserA(r)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusConflict {
		t.Fatalf("status: got %d want 409", w.Result().StatusCode)
	}
	if trig := w.Result().Header.Get("HX-Trigger"); strings.Contains(trig, "rates-changed") {
		t.Fatalf("HX-Trigger should NOT contain rates-changed, got %q", trig)
	}
	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, `id="rate-`+ruleID.String()+`"`) {
		t.Fatalf("body missing rate_row for referenced rule")
	}
	// Display mode — the referenced rule's row must not include the edit form.
	if strings.Contains(bodyStr, `hx-post="/rates/`+ruleID.String()+`"`) {
		t.Fatalf("body should be in display mode for referenced delete")
	}
	if !strings.Contains(bodyStr, `role="alert"`) {
		t.Fatalf("body should include inline error with role=alert")
	}

	// Rule still exists.
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM rate_rules WHERE id=$1`, ruleID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("referenced rule was deleted: count=%d", n)
	}
}

func TestNonHX_Regression_303Redirect(t *testing.T) {
	mux, f, ratesSvc, _ := setupHTMX(t)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ruleID, err := ratesSvc.Create(context.Background(), f.WorkspaceA, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: from,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create (second rule, different level to avoid overlap).
	body := url.Values{
		"scope":          {"client"},
		"client_id":      {f.ClientA.String()},
		"currency_code":  {"USD"},
		"hourly_decimal": {"200"},
		"effective_from": {from.Format("2006-01-02")},
	}
	r := httptest.NewRequest(http.MethodPost, "/rates", strings.NewReader(body.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r = f.AttachAsUserA(r)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusSeeOther {
		t.Fatalf("non-HX create: got %d want 303", w.Result().StatusCode)
	}
	if loc := w.Result().Header.Get("Location"); loc != "/rates" {
		t.Fatalf("non-HX create: Location=%q want /rates", loc)
	}

	// Update (extend effective_to — no referenced entries so anything goes).
	newTo := from.Add(180 * 24 * time.Hour).Format("2006-01-02")
	ubody := url.Values{
		"scope":          {"workspace"},
		"currency_code":  {"USD"},
		"hourly_decimal": {"100"},
		"effective_from": {from.Format("2006-01-02")},
		"effective_to":   {newTo},
	}
	r2 := httptest.NewRequest(http.MethodPost, "/rates/"+ruleID.String(), strings.NewReader(ubody.Encode()))
	r2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r2 = f.AttachAsUserA(r2)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, r2)
	if w2.Result().StatusCode != http.StatusSeeOther {
		t.Fatalf("non-HX update: got %d want 303", w2.Result().StatusCode)
	}

	// Delete (unreferenced).
	r3 := httptest.NewRequest(http.MethodPost, "/rates/"+ruleID.String()+"/delete", strings.NewReader(""))
	r3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r3 = f.AttachAsUserA(r3)
	w3 := httptest.NewRecorder()
	mux.ServeHTTP(w3, r3)
	if w3.Result().StatusCode != http.StatusSeeOther {
		t.Fatalf("non-HX delete: got %d want 303", w3.Result().StatusCode)
	}
}

func TestHX_TamperedImmutableFieldOnReferencedRule_Rejected(t *testing.T) {
	mux, f, ratesSvc, pool := setupHTMX(t)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ruleID, err := ratesSvc.Create(ctx, f.WorkspaceA, rates.Input{
		CurrencyCode: "USD", HourlyRateMinor: 10000, EffectiveFrom: from,
	})
	if err != nil {
		t.Fatal(err)
	}
	feb := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO time_entries (workspace_id,user_id,project_id,started_at,ended_at,duration_seconds,is_billable,rate_rule_id,hourly_rate_minor,currency_code)
		VALUES ($1,$2,$3,$4,$5,3600,true,$6,10000,'USD')
	`, f.WorkspaceA, f.UserA, f.ProjectA, feb, feb.Add(time.Hour), ruleID); err != nil {
		t.Fatal(err)
	}

	// Submit an HX update with a tampered currency_code.
	body := url.Values{
		"scope":          {"workspace"},
		"currency_code":  {"EUR"}, // tampered
		"hourly_decimal": {"100"},
		"effective_from": {from.Format("2006-01-02")},
	}
	r := hxRequest(t, http.MethodPost, "/rates/"+ruleID.String(), body)
	r = f.AttachAsUserA(r)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Result().StatusCode != http.StatusConflict {
		t.Fatalf("status: got %d want 409", w.Result().StatusCode)
	}
	var storedCurrency string
	if err := pool.QueryRow(ctx, `SELECT currency_code FROM rate_rules WHERE id=$1`, ruleID).Scan(&storedCurrency); err != nil {
		t.Fatal(err)
	}
	if storedCurrency != "USD" {
		t.Fatalf("currency was mutated: got %q want USD", storedCurrency)
	}
}
