# Tasks — improve-reporting-correctness-and-filters

Tasks are implementation-ready and grouped by layer. Keep commits small; run `make test`, `make vet`, and `make fmt` locally before each commit.

## Database

- [x] Add migration `migrations/00NN_add_workspaces_reporting_timezone.up.sql` that adds `reporting_timezone text NOT NULL DEFAULT 'UTC'` to `workspaces`, with `CHECK (length(reporting_timezone) > 0)`.
- [x] Add the paired `.down.sql` that drops the column.
- [x] Backfill verification query in the migration (or a separate seed script): confirm every row has a value; assert `pg_timezone_names` contains it for a sampled subset (informational only; no hard CHECK against `pg_timezone_names` because it is a view, not a constraint target).
- [x] Re-run `make migrate-redo` locally to confirm up/down symmetry.

## Backend — Workspace domain

- [x] Extend `internal/workspace.Service` (read + update) so `reporting_timezone` is returned on `Get` and settable via a new `UpdateReportingTimezone(ctx, workspaceID, tz)` method.
- [x] Validate tz at write time by querying `SELECT 1 FROM pg_timezone_names WHERE name = $1`; return a typed `ErrInvalidTimezone` when absent.
- [x] Wire the new method into the workspace handler on `POST /workspaces/<id>/settings/timezone` (or the existing settings endpoint — confirm the current route name and reuse).
- [x] Write a workspace-scoped integration test using `internal/shared/testdb`: one workspace, change tz, round-trip through `Get`; assert cross-workspace write returns 404 not 403.

## Backend — Reporting service

- [x] Replace the three `groupBy*` functions and `estimateScoped` with one `reportQuery` struct + builder that accepts `{workspaceID, userID, from, to, tz, grouping, clientID, projectID, billable}` and emits a shared WHERE skeleton.
- [x] Use `(te.started_at AT TIME ZONE $tz)::date BETWEEN $from AND $to` as the bucketing/range predicate; additionally bound `te.started_at` with an inclusive ±1-day envelope to keep the existing `(workspace_id, started_at)` index usable.
- [x] Apply `billable` tri-state (`""` / `"yes"` / `"no"`) uniformly across `totals`, `estimateScoped`, `noRate`, and each grouping query.
- [x] Apply `client_id` via `AND p.client_id = $N` and `project_id` via `AND te.project_id = $N` on every aggregation query, never in Go.
- [x] Load `reporting_timezone` by joining `workspaces w ON w.id = te.workspace_id` in each SQL (or passing it in once from the handler after a single `Get`). Pick one approach; document it in the service doc comment.
- [x] Update `PresetRange` to accept a `*time.Location` (or tz name) and compute ISO weeks, `this_month`, `last_month`, etc. against that location — not UTC.
- [x] Update `Dashboard` the same way so today/week totals respect the workspace tz. Confirm `DashboardSummary` field meanings are unchanged.
- [x] Benchmark the new queries against dev-seed with `EXPLAIN ANALYZE` before merging; attach results to the PR description. (Results in `design.md` → "Measured plans": inflated to 202k rows; all four queries use `Bitmap Index Scan on ix_time_entries_workspace_started_desc`, 26–37 ms exec time.)
- [x] Add integration tests under `internal/reporting`:
  - DST spring-forward: four 1-hour entries on `2026-03-08 America/New_York`; assert day total = 14400s.
  - DST fall-back: four 1-hour entries on `2026-11-01 America/New_York`; assert day total = 14400s, nothing attributed to `2026-10-31` or `2026-11-02`.
  - Cross-midnight local entry attributed to correct local date (the spec's "Entry started late at night in non-UTC workspace" scenario).
  - Client filter reduces `TotalsBlock`, `NoRateCount`, `EstimatedByCurrency`, and every grouping by the same amount.
  - `billable=yes` gives `NonBillableSeconds = 0`; `billable=no` gives `BillableSeconds = 0` and empty `EstimatedByCurrency` and `NoRateCount = 0`.
  - Multi-currency grand total: seed entries in USD and EUR; assert two keys, deterministic ordering by template render.

## Backend — Reporting handler

- [x] Extract a shared `parseFilters(r *http.Request, ws workspace.Workspace) (filters, error)` helper used by both `/reports` and `/reports/partial`.
- [x] Extract a shared `renderResults(w, r, filters, report)` helper that writes either the full page (via `tpls.Render("reports.index", …)`) or the partial (via `tpls.Render("reports.partial.results", …)`).
- [x] Add route `GET /reports/partial` in `Handler.Register`; reuse the same authz middleware.
- [x] Keep the existing cross-workspace 404 checks for `client_id` / `project_id`; apply the same checks on the partial endpoint.
- [x] Delete the `filterGrouped` post-filter helper — SQL does the work now.

## Templates — server rendering

- [x] Split `web/templates/reports/index.html`:
  - `index.html`: shell + filter form + `{{template "reports.partial.results" .}}` mount point wrapped in `<section id="report-results" aria-live="polite">`.
  - `web/templates/reports/partials/results.html`: per-grouping tables, grand-total per-currency block, no-rate flash.
  - `web/templates/reports/partials/empty.html`: the unified empty state with `role="status"`.
- [x] Register the new partials in `internal/shared/templates` so they are auto-included by both the full-page and partial-endpoint renders.
- [x] Add the grand-total per-currency block above the grouping table. Render each currency on its own `<div>` using `formatMinor` and the ISO code. Sort keys ascending by currency code (add a template func or sort in Go before passing).
- [x] Add the `Billable` filter control to the form: `<select name="billable">` with options `All` (empty) / `Billable only` (`yes`) / `Non-billable only` (`no`).
- [x] Add a workspace timezone `<select>` to the workspace settings template (if a settings page exists; otherwise create a minimal one under `web/templates/workspace/settings.html`). Options populated from a Go slice derived from `pg_timezone_names` at startup (cache it).

## HTMX wiring

- [x] On the report filter form, set `hx-get="/reports/partial"`, `hx-target="#report-results"`, `hx-swap="innerHTML"`, `hx-push-url="true"`, `hx-trigger="change from:find select, change from:find input[type=date], submit"`.
- [x] Add `data-focus-after-swap` to each filter control so `app.js` restores focus to the triggering control after swap (re-use existing helper).
- [x] Add `hx-indicator="#report-loading"` and a small spinner partial `web/templates/partials/spinner.html` shown while a swap is in flight. Confirm it does not block screen-reader announcements of the new results.
- [x] Ensure the no-JS submit path still posts the same URL with the same query string (plain `<button type="submit">`).
- [x] On workspace timezone save, emit `HX-Trigger: workspace-changed` and have the reports page listen via `hx-trigger="workspace-changed from:body"` to refresh the results region.

## Accessibility

- [x] Every filter control has a visible `<label for>` and visible focus ring (reuse design-token styles).
- [x] Tables keep semantic `<caption>`, `<thead>`, `<th scope="col">`, and numeric columns use the `num tabular` class for right-aligned tabular figures.
- [x] Grand-total per-currency block uses semantic markup (a `<dl>` or `<ul>` with ISO codes as text, not color).
- [x] Empty partial is inside `aria-live="polite"`; assert via a manual screen-reader pass (VoiceOver on macOS + NVDA on Windows if available) and document in the PR.
- [x] Contrast audit the `Archived` badge against background tokens; ensure ≥ 4.5:1 for the label text.
- [x] Keyboard-only walk-through: Tab through filter controls, change each, confirm focus persists and the live region announces empty→populated transitions.
- [x] Automated check: add an integration test that asserts the partial response body contains `aria-live="polite"` when the result set is empty.

## Structural / safety tests

- [x] Add `internal/reporting/structure_test.go` that uses `go/parser` to parse every `.go` file under `internal/reporting` and asserts none imports `timetrak/internal/rates`.
- [x] Add a handler-level test: issue `GET /reports?preset=this_week` and `GET /reports/partial?preset=this_week` with the same session; assert the `#report-results` slot in the first response is byte-identical to the entire body of the second response.
- [x] Extend `make check-rate-snapshots` CI gate run to also run the new structure test in the same job (no new target required; `go test ./internal/reporting/...` already covers it).

## Documentation

- [x] Update `docs/time_tracking_design_doc.md` reporting section: document the tz-aware bucketing, the SQL-layer filter invariant, the new partial endpoint, and the grand-total per-currency block.
- [x] No README changes required.

## Release

- [x] `make test` green locally with `DATABASE_URL` set.
- [x] `make vet` + `make fmt` clean.
- [x] `make check-rate-snapshots` passes on the deploy target before merge.
- [x] Archive via `/opsx:archive improve-reporting-correctness-and-filters` once shipped.
