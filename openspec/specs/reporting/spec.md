# reporting Specification

## Purpose
The reporting capability provides workspace-scoped read-only aggregations
of time entries for the user interface. It covers date-range filtering
with workspace-local day bucketing, totals by day, week, client, and
project, billable vs non-billable breakdowns, estimated-billable grand
totals per currency, the dashboard at-a-glance summary, and the HTMX
filter partial. Reporting MUST NOT call rate resolution for closed
entries: totals are computed exclusively from the per-entry rate
snapshot, and completeness of those snapshots is enforced by a deploy
gate. Every reporting handler performs exhaustive cross-workspace
denial, every surface has a unified empty-state region with announced
empty, loading, and error states, and the UI meets WCAG 2.2 AA.
## Requirements
### Requirement: Workspace-scoped reports

All reports MUST be scoped to the active workspace. The system MUST never include time entries, clients, or projects from a workspace the current user is not a member of.

#### Scenario: Other workspace data is excluded
- **GIVEN** Alice is a member of `W1` only
- **WHEN** Alice opens any report
- **THEN** only data with `workspace_id = W1` SHALL be aggregated and displayed

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

### Requirement: Billable vs non-billable breakdown

Every report view MUST display billable and non-billable totals separately, not only a combined total.

#### Scenario: Breakdown present
- **WHEN** any summary report is rendered
- **THEN** the UI MUST show both billable and non-billable totals as distinct values

### Requirement: Estimated billable amount

The system SHALL compute an estimated billable amount per grouping by summing, for each closed billable entry (`ended_at IS NOT NULL`), `duration_seconds * hourly_rate_minor / 3600` using the per-entry **rate snapshot** columns (`rate_rule_id`, `hourly_rate_minor`, `currency_code`) persisted at stop/save time. The reporting read path MUST treat the snapshot as the sole source of truth for closed entries and MUST NOT call the rate-resolution function (`rates.Service.Resolve`) for any closed entry, regardless of environment configuration. Amounts MUST be accumulated as integer minor units and displayed formatted by the entry's snapshot currency. A closed billable entry whose snapshot columns are NULL MUST contribute zero to the estimated billable amount and MUST be counted in the `Entries without a rate` aggregate (exposed as `EntriesWithoutRate` on the dashboard summary and as `NoRateCount` on reports). Entries with `ended_at IS NULL` (running timers) MUST NOT contribute to estimated billable amounts and MUST NOT be counted in `Entries without a rate`.

#### Scenario: Amount respects historical rate via snapshot
- **GIVEN** a closed billable entry of 60 minutes with a rate snapshot of 10000 minor units/hour in USD
- **AND** the underlying `rate_rules` row is later edited to 12000 minor units/hour
- **WHEN** the estimated billable amount is computed for that entry
- **THEN** its contribution is 10000 minor units in USD
- **AND** the edit to the rate rule has no effect on the report

#### Scenario: Closed entry without a snapshot is flagged, not resolved
- **GIVEN** a closed billable entry whose `rate_rule_id`, `hourly_rate_minor`, and `currency_code` are all NULL
- **WHEN** any report, dashboard summary, or grouped total is computed
- **THEN** the entry contributes zero to the estimated billable amount
- **AND** the entry increments the `Entries without a rate` count surfaced to the user
- **AND** the reporting read path MUST NOT invoke `rates.Service.Resolve` for the entry
- **AND** no environment variable or configuration flag changes this behavior

#### Scenario: Running timer is never counted
- **GIVEN** a billable entry with `ended_at IS NULL`
- **WHEN** any report or dashboard summary is computed
- **THEN** the entry contributes zero to the estimated billable amount
- **AND** the entry is NOT included in the `Entries without a rate` count

#### Scenario: Non-billable entries excluded from amount
- **GIVEN** a closed non-billable entry
- **WHEN** reports are computed
- **THEN** its billable-amount contribution is 0
- **AND** it is not counted in `Entries without a rate`

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

### Requirement: Deploy gate for snapshot completeness

The system SHALL provide an operator-invokable command (`migrate check-rate-snapshots`, exposed via `make check-rate-snapshots`) that exits with a non-zero status when any closed billable entry has a NULL `rate_rule_id`. The command's output MUST name the affected workspaces and include the count per workspace so an operator can take corrective action via the existing `backfill-rate-snapshots` command. The command MUST exit zero when no such entries exist.

#### Scenario: Check passes when every closed billable entry has a snapshot
- **GIVEN** every closed billable time entry has `rate_rule_id IS NOT NULL`
- **WHEN** an operator runs `make check-rate-snapshots`
- **THEN** the command prints a zero-count summary
- **AND** the process exit code is 0

#### Scenario: Check fails when any closed billable entry lacks a snapshot
- **GIVEN** at least one closed billable time entry has `rate_rule_id IS NULL`
- **WHEN** an operator runs `make check-rate-snapshots`
- **THEN** the command prints the affected workspace IDs and per-workspace counts
- **AND** the process exit code is non-zero

#### Scenario: Running entries never cause the check to fail
- **GIVEN** a running timer entry exists (`ended_at IS NULL`) with no snapshot columns
- **AND** every closed billable entry has a snapshot
- **WHEN** an operator runs `make check-rate-snapshots`
- **THEN** the process exit code is 0

### Requirement: Dashboard at-a-glance summary

The dashboard SHALL display, for the active workspace and current user: the running timer (if any), today's total duration (billable and non-billable), this week's total duration (billable and non-billable), and this week's estimated billable amount. These widgets MUST refresh via HTMX when the timer starts or stops and when entries are created, edited, or deleted.

#### Scenario: Dashboard updates after timer stop
- **GIVEN** Alice stops a running timer
- **WHEN** the stop response swaps in the timer widget partial
- **THEN** today's total and this week's total MUST reflect the new duration
- **AND** the dashboard SHOULD NOT require a full page reload

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

### Requirement: Exhaustive cross-workspace denial for every reporting handler

Every read handler in the `reporting` family MUST return HTTP 404 with the shared not-found response body when invoked by a user whose active workspace does not own a referenced filter target (client, project, or entry), and MUST scope all aggregations strictly to the caller's active workspace when no specific target is referenced. This rule applies without exception to: dashboard summary, today/week totals, billable totals, and any entries-list filter pages. No reporting response may aggregate or display data from a workspace other than the caller's active workspace.

#### Scenario: Dashboard summary is scoped to active workspace
- **GIVEN** Alice's active workspace is `W1` and entries exist in both `W1` and `W2`
- **WHEN** Alice loads the dashboard
- **THEN** the running-timer widget, today's total, this-week's total, and this-week's billable amount MUST reflect only entries with `workspace_id = W1`
- **AND** no data from `W2` influences any displayed figure

#### Scenario: Report filter by other-workspace project returns 404
- **GIVEN** project `P2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice requests a report filtered by `project_id = P2`
- **THEN** the system MUST respond with HTTP 404
- **AND** no aggregation is performed

#### Scenario: Report filter by other-workspace client returns 404
- **GIVEN** client `C2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice requests a report filtered by `client_id = C2`
- **THEN** the system MUST respond with HTTP 404

#### Scenario: Entries-list filter is scoped to active workspace
- **GIVEN** Alice's active workspace is `W1`
- **WHEN** Alice requests the entries list with any combination of filters
- **THEN** every returned row MUST have `workspace_id = W1`
- **AND** pagination counts MUST reflect only `W1` entries

### Requirement: Report filter bar SHALL meet WCAG 2.2 AA accessibility

The report filter bar SHALL:

- Pair every filter control (date range, client, project, billable toggle) with a visible `<label>`.
- Group related controls inside a `<fieldset>` with a `<legend>` where the grouping is semantic (e.g. date-range start/end).
- Be fully operable by keyboard — every control reachable via `Tab`, submit via `Enter`, clear/reset via a visible button.
- After submitting filters via HTMX, move focus to the results `<table>` via `data-focus-after-swap` on an element with `tabindex="-1"`.

#### Scenario: Submitting filters moves focus to results

- **GIVEN** a user applies a date range and submits
- **WHEN** the results partial is swapped into the target
- **THEN** focus MUST land on the results table (or its heading) via `data-focus-after-swap`

#### Scenario: Every filter control has a visible label

- **GIVEN** the reports page renders
- **WHEN** the filter bar is inspected
- **THEN** every interactive control MUST have a visible `<label>`
- **AND** date-range start/end MUST be grouped inside a `<fieldset>` with a `<legend>`

### Requirement: Report results table SHALL present accessible semantics

The results table SHALL include a `<caption>` describing the filter applied (e.g. "Billable hours by project — Apr 1 to Apr 14"), `<th scope="col">` on header cells, right-aligned numeric columns via a CSS utility class with `font-variant-numeric: tabular-nums`, and — if any column is sortable — the currently-sorted column SHALL expose `aria-sort="ascending"` or `aria-sort="descending"` and sortable headers SHALL be buttons inside their `<th>`.

#### Scenario: Caption reflects applied filter

- **GIVEN** a user applies a filter and submits
- **WHEN** the results render
- **THEN** the table `<caption>` MUST summarize the filter in human-readable text

#### Scenario: Numeric columns are right-aligned via utility class

- **GIVEN** results contain hours and money columns
- **WHEN** the table renders
- **THEN** those columns MUST be right-aligned via a CSS utility class
- **AND** MUST NOT use inline `style` attributes
- **AND** MUST apply `font-variant-numeric: tabular-nums`

### Requirement: Report empty, loading, and error states SHALL be announced

An empty results set SHALL render via a dedicated partial inside an `aria-live="polite"` container whose text explains the empty condition in domain-specific copy (e.g. "No billable time in the selected range"). A loading state (where rendered) SHALL use `role="status"` with accessible text. An error state SHALL use `role="alert"` with `tabindex="-1"` and receive focus via `data-focus-after-swap`.

#### Scenario: Empty results after filter change

- **GIVEN** a user applies a filter that returns no rows
- **WHEN** the empty-state partial is swapped in
- **THEN** the container MUST carry `aria-live="polite"`
- **AND** the text MUST explain the empty condition in domain-specific copy

#### Scenario: Server error during report generation

- **GIVEN** the server returns an error during report generation
- **WHEN** the error partial is swapped in
- **THEN** the container MUST carry `role="alert"` and `tabindex="-1"`
- **AND** focus MUST land on the error container via `data-focus-after-swap`

