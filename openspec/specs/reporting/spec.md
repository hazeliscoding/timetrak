# reporting Specification

## Purpose
TBD - created by archiving change bootstrap-timetrak-mvp. Update Purpose after archive.
## Requirements
### Requirement: Workspace-scoped reports

All reports MUST be scoped to the active workspace. The system MUST never include time entries, clients, or projects from a workspace the current user is not a member of.

#### Scenario: Other workspace data is excluded
- **GIVEN** Alice is a member of `W1` only
- **WHEN** Alice opens any report
- **THEN** only data with `workspace_id = W1` SHALL be aggregated and displayed

### Requirement: Date-range filter

Every report SHALL support a date-range filter with `from` (inclusive) and `to` (inclusive) dates. Entries are included in a report when `started_at::date` falls within the selected range. The UI MUST provide sensible preset ranges (e.g., `This week`, `Last week`, `This month`) and a custom range.

#### Scenario: Preset `This week`
- **WHEN** Alice selects `This week` on a Friday
- **THEN** the report includes entries whose `started_at::date` is within the current Monday–Sunday (or locale equivalent) range

#### Scenario: Custom range inclusive of boundaries
- **WHEN** Alice sets from=2026-04-01 and to=2026-04-17
- **THEN** entries on 2026-04-01 and 2026-04-17 are included

### Requirement: Totals by day and by week

The system SHALL compute totals by day and by week within the selected date range: total duration (seconds), billable duration, and non-billable duration.

#### Scenario: Daily totals
- **WHEN** Alice views a report grouped by day
- **THEN** each day in range has a row with total, billable, and non-billable duration
- **AND** days with zero entries are either omitted or explicitly shown as zero (implementation MAY choose, but MUST be consistent)

### Requirement: Totals by client and by project

The system SHALL compute totals by client and by project within the selected date range: total duration, billable duration, non-billable duration, and estimated billable amount (see below). Archived clients and archived projects MUST still appear in historical reports.

#### Scenario: Client grouping
- **WHEN** Alice views a report grouped by client
- **THEN** each client with at least one entry in range is shown with its totals
- **AND** the row label includes the client name, and archived clients are labeled with `Archived` text (not color alone)

#### Scenario: Project grouping
- **WHEN** Alice views a report grouped by project
- **THEN** each project with at least one entry in range is shown with its client name, total duration, billable duration, and estimated billable amount

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

The `reporting` service MUST NOT hold or consume a reference to the rate-resolution service (`rates.Service.Resolve`) on any read path. Construction of the reporting service MUST NOT require a rates dependency. This is a structural invariant intended to prevent the fallback path from silently returning.

#### Scenario: Reporting service constructor has no rates dependency
- **WHEN** the reporting service is constructed at application startup
- **THEN** the constructor signature MUST NOT take a `*rates.Service` (or equivalent) parameter
- **AND** the reporting package MUST NOT import the rates package from its read-path files

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

Reports MUST meet WCAG 2.2 AA. Tables MUST use semantic `<table>` markup with `<th scope>` headers; totals MUST be conveyed as text (not color alone); filter controls MUST have visible labels and visible keyboard focus; HTMX filter swaps MUST preserve or explicitly move focus; an empty-state region MUST be announced via `aria-live` when filters produce no results.

#### Scenario: Keyboard-only filtering
- **WHEN** a keyboard-only user changes the date range and grouping
- **THEN** the report partial updates
- **AND** focus remains on or near the filter control
- **AND** the new totals are programmatically associated with the table caption

#### Scenario: Empty report result
- **WHEN** the current filters match zero entries
- **THEN** an empty-state message is rendered
- **AND** the message is announced via `aria-live`
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

