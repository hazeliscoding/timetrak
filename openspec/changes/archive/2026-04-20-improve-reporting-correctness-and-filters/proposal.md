## Why

Reporting is snapshot-only for closed entries (good), but correctness around day boundaries and filter ergonomics are still weak. All aggregation compares `started_at` against UTC-bounded ranges and ignores the user's local day, which means an entry started at 23:30 local time but 06:30 UTC is bucketed against the wrong calendar day. The handler also accepts `client_id` and `project_id` but applies them by filtering an already-computed in-memory result set rather than narrowing the SQL, which (a) inflates the grand total / non-billable figures shown above filtered tables, (b) makes `By Day` grouping ignore the filter entirely, and (c) yields inconsistent "Entries without a rate" counts. Multi-currency totals render correctly row-by-row but the summary has no stable display order, and there is no grand-total per-currency breakdown when the range contains entries in more than one currency. Empty / loading states are partially inconsistent across the three grouping modes. Stage 2 requires we fix these before any new reporting surface lands.

## What Changes

- Introduce a **workspace-local reporting timezone** so date-range filters and day bucketing match the user's wall clock, not UTC. Persist the timezone on the workspace and pass it through to every reporting query via `AT TIME ZONE`.
- Make the `client_id` and `project_id` filters **narrow aggregation at the SQL layer** for every grouping (`day`, `client`, `project`) and for the `TotalsBlock`, `NoRateCount`, and per-currency amount — not post-filter in Go. Filter-with-no-matches must render the same empty state as no-data.
- Add **billable-only / non-billable-only / all** toggle (`billable=`) to the filter set. Totals and amounts respect the toggle; `NoRateCount` only counts closed billable entries that match the non-billable-exclusive slice.
- Return a **grand-total estimated billable block per currency** (not just per grouped row), ordered deterministically by currency code. Render it above the grouping table.
- Normalize every **empty / loading / error state** across the three groupings to a single partial with an `aria-live="polite"` region; HTMX swaps preserve focus on the changed filter control via `data-focus-after-swap`.
- Convert the report filter form to **HTMX-driven submission** (`hx-get` to `/reports/partial`) so changing a filter swaps only the results partial — the surrounding page chrome and filter controls keep focus. Full-page GET remains supported for deep links and no-JS.
- Tighten snapshot-only invariants: add an explicit test that the reporting package imports from `rates` in no file on the read path, and that `Report`, `Dashboard`, and the new partial endpoint all return zero-amount + `NoRateCount += 1` when a closed billable entry's snapshot is NULL.
- Explicitly out of scope (deferred to follow-up changes): CSV / Excel export, PDF invoices, saved report presets, shared report links, scheduled emails, cross-workspace roll-ups, and any write path through reporting.

## Capabilities

### New Capabilities

_None._ This change hardens existing reporting behavior; no new domain is introduced.

### Modified Capabilities

- `reporting`: date-range semantics, client/project/billable filter application, grand-total per-currency block, unified empty state, HTMX partial endpoint, and the snapshot-only structural invariant are all tightened via delta.
- `workspace`: a single new field (`reporting_timezone`) is added so reporting can bucket days in the user's local time; no other workspace behavior changes.

## Impact

- **Code**: `internal/reporting/service.go` (SQL rewrites for scoped filters, timezone), `internal/reporting/handler.go` (HTMX partial route, billable filter), `internal/workspace/service.go` + settings template (add `reporting_timezone`), new reports partial template.
- **Database**: one migration adds `workspaces.reporting_timezone text not null default 'UTC'` with a CHECK validating an IANA tz name via a whitelist view or a startup-time validation pass.
- **Templates**: `web/templates/reports/index.html` is split into `index.html` + `partials/report_results.html` + `partials/report_summary.html`; empty state consolidated.
- **HTMX**: new `GET /reports/partial` endpoint; filter form gains `hx-get`, `hx-target`, `hx-push-url="true"`, and `data-focus-after-swap` hooks.
- **Tests**: new integration tests for timezone bucketing across DST, SQL-layer filter narrowing, multi-currency grand totals, and a compile-time-ish import check that `internal/reporting` does not reference `internal/rates` on the read path.
- **Risk**: adding `AT TIME ZONE` to aggregation queries can defeat existing indexes on `time_entries(started_at)`. Design doc covers the index strategy (either a functional index on `(started_at AT TIME ZONE <workspace_tz>)` — impractical since tz is per-row via join — or keeping the existing index and ensuring the planner uses it by bounding `started_at` on both sides with the tz-converted range). Benchmarked via `EXPLAIN ANALYZE` on the dev-seed dataset before merge.
- **Backfill**: all existing workspaces receive `reporting_timezone = 'UTC'` on migration, preserving current behavior. Users can change it in settings.
