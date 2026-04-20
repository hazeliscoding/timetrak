## MODIFIED Requirements

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

## ADDED Requirements

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
