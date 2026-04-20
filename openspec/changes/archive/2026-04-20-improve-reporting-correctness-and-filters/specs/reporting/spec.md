## ADDED Requirements

### Requirement: Workspace-local day bucketing

Reports SHALL bucket time entries into calendar days using the active workspace's `reporting_timezone` (IANA tz name), not UTC. The `from` and `to` date-range parameters SHALL be interpreted as calendar dates in the workspace's local timezone. Day bucketing, ISO-week presets, and the `started_at::date` predicate MUST use `(started_at AT TIME ZONE <workspace_tz>)::date` semantics so that an entry's calendar day matches what the user saw in the timer UI.

#### Scenario: Entry started late at night in non-UTC workspace

- **GIVEN** workspace `W` has `reporting_timezone = 'America/New_York'`
- **AND** Alice starts an entry at `2026-04-17 23:30 America/New_York` (`2026-04-18 03:30 UTC`)
- **AND** the entry is 60 minutes long and billable
- **WHEN** Alice runs a report for `from = 2026-04-17, to = 2026-04-17`
- **THEN** the day bucket for `2026-04-17` SHALL include the entry's 60 minutes
- **AND** the day bucket for `2026-04-18` SHALL NOT include the entry

#### Scenario: DST spring-forward day

- **GIVEN** workspace `W` has `reporting_timezone = 'America/New_York'`
- **AND** Alice records four one-hour entries across `2026-03-08` local time (a 23-hour day due to DST)
- **WHEN** Alice runs a report for `from = 2026-03-08, to = 2026-03-08`
- **THEN** the day bucket for `2026-03-08` SHALL total exactly 4 * 3600 = 14400 seconds
- **AND** no portion of any entry SHALL be attributed to `2026-03-09`

#### Scenario: Default timezone is UTC

- **GIVEN** a workspace whose `reporting_timezone` has never been changed
- **WHEN** the workspace row is inspected
- **THEN** `reporting_timezone` SHALL equal `'UTC'`
- **AND** existing report behavior SHALL be unchanged relative to the UTC-only baseline

### Requirement: SQL-layer filter narrowing

When the handler receives `client_id`, `project_id`, or `billable` query parameters, the reporting service SHALL apply each filter as a WHERE-clause predicate on every aggregation query — `Totals`, `EstimatedByCurrency`, `NoRateCount`, and the selected grouping (`day`, `client`, `project`). Post-query filtering of the result set in application code is prohibited for these parameters. `TotalsBlock`, `NoRateCount`, and each grouping's rows MUST be mutually consistent for any combination of filters.

#### Scenario: Client filter narrows every aggregate

- **GIVEN** workspace `W` has entries against clients `C1` and `C2` in the range
- **WHEN** Alice submits `client_id = C1` with `grouping = day`
- **THEN** `TotalsBlock.TotalSeconds` SHALL equal the sum of `C1`'s entries only
- **AND** `TotalsBlock.BillableSeconds` SHALL reflect only `C1`'s billable entries
- **AND** `TotalsBlock.EstimatedByCurrency` SHALL reflect only `C1`'s closed billable entries with a snapshot
- **AND** `NoRateCount` SHALL count only `C1`'s closed billable entries missing a snapshot
- **AND** each `ByDay` row SHALL contain only `C1`'s seconds for that day

#### Scenario: Billable tri-state

- **WHEN** Alice submits `billable = yes`
- **THEN** every aggregate SHALL include only entries with `is_billable = true`
- **AND** `TotalsBlock.NonBillableSeconds` SHALL equal 0
- **WHEN** Alice submits `billable = no`
- **THEN** every aggregate SHALL include only entries with `is_billable = false`
- **AND** `TotalsBlock.BillableSeconds` SHALL equal 0
- **AND** `TotalsBlock.EstimatedByCurrency` SHALL be empty
- **AND** `NoRateCount` SHALL equal 0
- **WHEN** `billable` is omitted or empty
- **THEN** every aggregate SHALL include both billable and non-billable entries

#### Scenario: Project filter applies to day grouping

- **GIVEN** the current handler behavior post-migration
- **WHEN** Alice submits `grouping = day` with `project_id = P1`
- **THEN** each `ByDay` row SHALL contain only seconds attributable to `P1`
- **AND** `TotalsBlock` SHALL agree with the sum of `ByDay` rows

### Requirement: Grand-total estimated billable per currency

When a report has at least one closed billable entry with a non-NULL rate snapshot in the selected range (and filter combination), the UI SHALL render a grand-total block that lists the estimated billable amount once per distinct `currency_code`, ordered ascending by currency code. Amounts from different currencies MUST NOT be summed together. When the range produces no such entries, the block SHALL be omitted.

#### Scenario: Multi-currency grand total

- **GIVEN** the selected range contains closed billable entries in `USD` and `EUR`
- **WHEN** the report is rendered
- **THEN** the grand-total block SHALL show one labeled row for `EUR` and one for `USD`, in that alphabetical order
- **AND** no combined cross-currency total SHALL be rendered

#### Scenario: Single currency

- **GIVEN** all closed billable entries in the range share `currency_code = 'USD'`
- **WHEN** the report is rendered
- **THEN** the grand-total block SHALL show exactly one row labeled `USD`

#### Scenario: No billable entries in range

- **GIVEN** the range contains only non-billable entries
- **WHEN** the report is rendered
- **THEN** the grand-total per-currency block SHALL be omitted from the DOM

### Requirement: HTMX filter partial endpoint

The system SHALL expose `GET /reports/partial` that returns only the report results fragment (totals, grouping table or empty state, grand-total currency block, no-rate flash). The `/reports` filter form SHALL use HTMX attributes (`hx-get`, `hx-target`, `hx-push-url`) to swap the fragment when any filter control changes. The full-page `GET /reports` SHALL continue to render the entire page for deep links and no-JS clients. Both endpoints SHALL parse query parameters identically and SHALL render byte-identical markup for the swapped region given the same query string.

#### Scenario: Partial swap preserves focus

- **GIVEN** Alice is focused on the `Preset` select
- **WHEN** she changes the preset from `This week` to `Last month`
- **THEN** the browser SHALL issue a `GET /reports/partial` with the new query string
- **AND** only the `#report-results` region SHALL swap
- **AND** focus SHALL remain on the `Preset` select
- **AND** the URL SHALL be updated via `hx-push-url` so the view is deep-linkable

#### Scenario: Deep link via full page

- **WHEN** Alice opens `/reports?preset=last_month&group=project&client_id=<C1>` in a new tab
- **THEN** the server SHALL respond with the complete page
- **AND** the rendered results SHALL match what `/reports/partial` would produce for the same query string

#### Scenario: Invalid workspace-scoped filter returns 404 on the partial

- **GIVEN** project `P2` does not belong to Alice's active workspace
- **WHEN** Alice requests `/reports/partial?project_id=P2`
- **THEN** the system SHALL respond with HTTP 404
- **AND** the response body SHALL be the shared not-found body, not aggregation markup

### Requirement: Unified empty-state region

Every report grouping (`day`, `client`, `project`) SHALL share one accessible empty-state partial rendered when the filtered result set is empty. The partial MUST use semantic messaging (not color alone) and MUST be wrapped in an `aria-live="polite"` region so assistive tech announces the change when HTMX swaps replace populated results with the empty state.

#### Scenario: Filter produces zero results

- **WHEN** Alice applies filters that match no entries
- **THEN** the response SHALL render the shared empty partial with a domain-specific message (e.g., "No entries match these filters")
- **AND** the partial SHALL be inside a region with `aria-live="polite"`
- **AND** no grouping table SHALL be rendered in the DOM

#### Scenario: Swapping from populated to empty announces

- **GIVEN** Alice has a populated report on screen
- **WHEN** she narrows filters so the result is empty
- **THEN** the HTMX swap SHALL replace the table with the empty partial
- **AND** assistive tech SHALL be notified via the `aria-live` region

## MODIFIED Requirements

### Requirement: Date-range filter

Every report SHALL support a date-range filter with `from` (inclusive) and `to` (inclusive) dates interpreted in the workspace's `reporting_timezone`. Entries are included in a report when `(started_at AT TIME ZONE <workspace_tz>)::date` falls within the selected range. The UI MUST provide sensible preset ranges (e.g., `This week`, `Last week`, `This month`) and a custom range. Preset ranges SHALL be computed against the current wall-clock time in `<workspace_tz>`.

#### Scenario: Preset `This week` in non-UTC workspace

- **GIVEN** `reporting_timezone = 'America/New_York'`
- **WHEN** Alice selects `This week` at `2026-04-17 22:00 America/New_York` (a Friday)
- **THEN** the report includes entries whose local `started_at` date falls within the current Monday–Sunday range of `America/New_York`
- **AND** the range is NOT shifted by the UTC offset

#### Scenario: Custom range inclusive of boundaries

- **WHEN** Alice sets from=2026-04-01 and to=2026-04-17
- **THEN** entries whose local `started_at` date is `2026-04-01` through `2026-04-17` are included
- **AND** entries whose local date is `2026-03-31` or `2026-04-18` are excluded

### Requirement: Totals by day and by week

The system SHALL compute totals by day and by week within the selected date range, using `(started_at AT TIME ZONE <workspace_tz>)::date` as the bucketing key: total duration (seconds), billable duration, and non-billable duration. When any `client_id`, `project_id`, or `billable` filter is active, each bucket SHALL reflect only entries that satisfy the filters.

#### Scenario: Daily totals respect timezone and filters

- **GIVEN** `reporting_timezone = 'America/New_York'` and `project_id = P1` is filtered
- **WHEN** Alice views a report grouped by day
- **THEN** each day in range has a row containing only `P1`'s seconds, bucketed by local date
- **AND** days with zero matching entries are either omitted or shown as zero (implementation MAY choose, but MUST be consistent)

### Requirement: Totals by client and by project

The system SHALL compute totals by client and by project within the selected date range: total duration, billable duration, non-billable duration, and estimated billable amount. Archived clients and archived projects MUST still appear in historical reports. When any `client_id`, `project_id`, or `billable` filter is active, each row SHALL reflect only matching entries. When `client_id` is active, the `by client` grouping SHALL return at most one row; when `project_id` is active, the `by project` grouping SHALL return at most one row.

#### Scenario: Client grouping with project filter

- **GIVEN** `project_id = P1` is filtered and `P1` belongs to client `C1`
- **WHEN** Alice views a report grouped by client
- **THEN** only `C1` is listed
- **AND** `C1`'s totals reflect only `P1`'s entries, not `C1`'s other projects

#### Scenario: Project grouping

- **WHEN** Alice views a report grouped by project
- **THEN** each project with at least one matching entry in range is shown with its client name, total duration, billable duration, and estimated billable amount
- **AND** archived projects are labeled with `Archived` text (not color alone)

### Requirement: Reporting service must not depend on rate resolution at read time

The `reporting` service MUST NOT hold or consume a reference to the rate-resolution service (`rates.Service.Resolve`) on any read path. Construction of the reporting service MUST NOT require a rates dependency. No file in the `internal/reporting` package (including the new partial handler and any helpers) SHALL import `timetrak/internal/rates`. This structural invariant is verified by an automated test that parses the package's Go files and inspects their import declarations.

#### Scenario: Reporting service constructor has no rates dependency

- **WHEN** the reporting service is constructed at application startup
- **THEN** the constructor signature MUST NOT take a `*rates.Service` (or equivalent) parameter

#### Scenario: No file in the reporting package imports rates

- **GIVEN** every `.go` file under `internal/reporting`
- **WHEN** the structural test runs
- **THEN** no file SHALL declare `"timetrak/internal/rates"` in its import block
- **AND** the test SHALL fail loudly if a future contributor adds such an import

#### Scenario: Removing the rate-rules table would not change reporting behavior

- **GIVEN** closed entries already carry their rate snapshot
- **WHEN** the `rate_rules` table is hypothetically unavailable at read time
- **THEN** reporting estimated billable amounts MUST still be computable from `time_entries` alone

### Requirement: Reporting UI accessibility

Reports MUST meet WCAG 2.2 AA. Tables MUST use semantic `<table>` markup with `<th scope>` headers; totals MUST be conveyed as text (not color alone); filter controls MUST have visible labels and visible keyboard focus; HTMX filter swaps MUST preserve focus on the control that triggered the swap or explicitly move focus to a sensible landmark; the unified empty-state partial MUST be announced via `aria-live="polite"` when filters produce no results; the grand-total per-currency block MUST render each currency as text with its ISO code visible.

#### Scenario: Keyboard-only filter change via HTMX

- **WHEN** a keyboard-only user tabs to the `Group by` control and changes its value
- **THEN** the HTMX partial swap executes
- **AND** focus remains on the `Group by` control after the swap
- **AND** the new totals and grand-total currency block are programmatically associated with the table caption or the live region

#### Scenario: Empty report result announces

- **WHEN** the current filters match zero entries
- **THEN** the shared empty partial is rendered
- **AND** the partial is inside a region with `aria-live="polite"`
- **AND** the emptiness is not conveyed by color alone
