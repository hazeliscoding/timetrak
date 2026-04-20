## ADDED Requirements

### Requirement: Rate snapshot produced at entry close/save time

The system SHALL, within the same database transaction that closes a running timer or persists an edit to a time entry's `project_id`, `started_at`, `ended_at`, `duration_seconds`, or `is_billable`, invoke `rates.Service.Resolve(ctx, workspaceID, projectID, started_at)` exactly once and persist the result as a snapshot on the time entry (`rate_rule_id`, `hourly_rate_minor`, `currency_code`). Running entries (`ended_at IS NULL`) MUST NOT carry a snapshot. Reports MUST read the snapshot for closed entries instead of re-resolving at read time.

#### Scenario: Stopping a timer writes the snapshot

- **GIVEN** a running entry for project `P1` started at 2026-04-17T09:00Z in workspace `W1`
- **AND** a workspace-default rate rule `R1` of 10000 minor units USD effective from 2026-01-01 with no end
- **WHEN** Alice stops the timer at 2026-04-17T10:00Z
- **THEN** the entry row MUST have `rate_rule_id = R1`, `hourly_rate_minor = 10000`, `currency_code = 'USD'`
- **AND** the snapshot MUST be written in the same transaction that set `ended_at` and `duration_seconds`

#### Scenario: Editing an entry re-evaluates the snapshot

- **GIVEN** a closed entry previously snapshotted against rule `R1`
- **WHEN** Alice edits the entry's `project_id` to a project whose client has an active client-level rule `R2`
- **THEN** the saved entry MUST carry `rate_rule_id = R2` and `R2`'s rate and currency
- **AND** the snapshot update MUST occur in the same transaction as the entry edit

#### Scenario: No resolvable rate produces NULL snapshot columns

- **GIVEN** no rate rule is active for the entry's project, client, or workspace on the entry's `started_at` date
- **WHEN** the entry is stopped or saved
- **THEN** `rate_rule_id`, `hourly_rate_minor`, and `currency_code` MUST all be NULL
- **AND** the entry MUST still be surfaced in the existing `Entries without a rate` aggregate

#### Scenario: Running timer has no snapshot

- **GIVEN** a running entry (`ended_at IS NULL`)
- **WHEN** any code reads the entry
- **THEN** `rate_rule_id`, `hourly_rate_minor`, and `currency_code` MUST be NULL
- **AND** reporting MUST NOT include the running entry in billable amounts until it is stopped

#### Scenario: Snapshot columns are atomic (all-set or all-null)

- **WHEN** the system writes a snapshot on a time entry
- **THEN** either all three of `rate_rule_id`, `hourly_rate_minor`, `currency_code` are NULL
- **OR** all three are non-NULL
- **AND** a database check constraint MUST enforce this invariant

### Requirement: Rate rules referenced by time entries are immutable on the historical axis

The system MUST reject any mutation of a `rate_rules` row that would alter the historical figure seen by at least one `time_entries.rate_rule_id = R`. Specifically, when any entry references rule `R`, the system MUST reject a delete of `R`, and MUST reject an update that changes `hourly_rate_minor`, `currency_code`, `client_id`, `project_id`, `effective_from`, or that shortens `effective_to` to a date earlier than the latest `started_at::date` among referencing entries. The system SHALL allow updates that only extend an open-ended `effective_to` from NULL to a future date, or that shorten `effective_to` to a date on or after the latest referencing entry's `started_at::date`. Rejection MUST return a typed error distinguishable from `ErrNotFound` and `ErrOverlap`, and the UI MUST surface a text explanation (not color alone) that names the number of referencing entries.

#### Scenario: Delete of referenced rule is rejected

- **GIVEN** rate rule `R1` is referenced by three closed time entries in workspace `W1`
- **WHEN** Alice attempts to delete `R1`
- **THEN** the system MUST reject the delete with a typed error meaning "rule is referenced by historical entries"
- **AND** the database row MUST remain unchanged

#### Scenario: Edit of rate amount on a referenced rule is rejected

- **GIVEN** rate rule `R1` is referenced by at least one closed time entry
- **WHEN** Alice attempts to update `R1.hourly_rate_minor` from 10000 to 12000
- **THEN** the system MUST reject the update with a typed error
- **AND** the database row MUST remain unchanged
- **AND** the UI MUST display a text hint naming the number of referencing entries and suggesting creating a successor rule

#### Scenario: Closing an open-ended rule on a future date is allowed

- **GIVEN** rate rule `R1` has `effective_from = 2026-01-01`, `effective_to = NULL`, and is referenced by entries dated 2026-01-10 through 2026-04-17
- **WHEN** Alice sets `R1.effective_to = 2026-06-30` (a future date)
- **THEN** the system MUST accept the update
- **AND** the database row's `effective_to` MUST be `2026-06-30`

#### Scenario: Shortening effective_to below latest referencing entry is rejected

- **GIVEN** rate rule `R1` has `effective_to = 2026-12-31` and is referenced by an entry dated 2026-04-17
- **WHEN** Alice attempts to set `R1.effective_to = 2026-03-31`
- **THEN** the system MUST reject the update with the typed "referenced" error
- **AND** the database row MUST remain unchanged

#### Scenario: Rate rules UI shows a referenced-count hint

- **GIVEN** rate rule `R1` is referenced by 3 time entries
- **WHEN** the rate rules list is rendered
- **THEN** the row for `R1` MUST display a textual indicator such as `Referenced by 3 entries` next to the edit/delete controls
- **AND** the indicator MUST NOT rely on color alone
- **AND** destructive controls MUST be disabled with a `aria-describedby`-associated explanation

## MODIFIED Requirements

### Requirement: No overlapping date windows at the same level

The system MUST reject a rate-rule create or edit that would cause two rules at the same precedence level (workspace-default, same client, or same project) to overlap in their effective date ranges. `effective_from` is inclusive and `effective_to` is inclusive; adjacency (`A.effective_to + 1 day = B.effective_from`) is the required pattern for a handoff. Two rules at the same level MUST NOT share any UTC date, including sharing the boundary date (`A.effective_to = B.effective_from` is rejected as overlap).

#### Scenario: Overlapping workspace-default rejected

- **GIVEN** a workspace default exists with `effective_from = 2026-01-01`, `effective_to = NULL`
- **WHEN** a second workspace default is submitted with `effective_from = 2026-06-01`, `effective_to = NULL`
- **THEN** the submission MUST be rejected with a validation error
- **AND** the new row is not inserted

#### Scenario: Adjacent, non-overlapping windows are allowed

- **GIVEN** a workspace default with `effective_from = 2026-01-01`, `effective_to = 2026-06-30`
- **WHEN** a new workspace default is submitted with `effective_from = 2026-07-01`, `effective_to = NULL`
- **THEN** the submission MUST be accepted

#### Scenario: Shared boundary date is rejected as overlap

- **GIVEN** a workspace default with `effective_from = 2026-01-01`, `effective_to = 2026-06-30`
- **WHEN** a new workspace default is submitted with `effective_from = 2026-06-30`, `effective_to = NULL`
- **THEN** the submission MUST be rejected as overlapping
- **AND** the error message MUST reference both windows' date ranges

### Requirement: Centralized rate resolution with precedence

The system SHALL provide a single rate-resolution function `Resolve(workspaceID, projectID, at)` that returns the applicable rate by consulting rules in this precedence order:

1. project-level rule for `project_id = projectID` active at `at`
2. client-level rule for `client_id = project's client_id` active at `at`
3. workspace-default rule for `workspace_id = workspaceID` active at `at`
4. no-rate sentinel

A rule is "active at `at`" when, with `date = at.UTC()::date`: `effective_from <= date AND (effective_to IS NULL OR date <= effective_to)`. Because the system rejects overlapping windows at the same level (see overlap requirement), at most one rule per precedence tier is ever active on a given date. `Resolve` MUST be invoked exactly once per entry at the moment the entry is created, stopped, or edited (see snapshot requirement) and MUST NOT be invoked on the reporting read path. Live-preview features that display a rate for a yet-unsaved entry MAY call `Resolve` but MUST label the result as a preview, not a historical figure.

#### Scenario: Project rule wins over client and workspace

- **GIVEN** a workspace default, a client rule, and a project rule all active on 2026-04-17
- **WHEN** `Resolve(W1, P1, 2026-04-17)` is called
- **THEN** the project rule is returned

#### Scenario: Fall through to client when no project rule

- **GIVEN** no project rule active on 2026-04-17 for `P1`, but a client rule and workspace default are active
- **WHEN** `Resolve(W1, P1, 2026-04-17)` is called
- **THEN** the client rule is returned

#### Scenario: Fall through to workspace default when no project or client rule

- **GIVEN** only the workspace default is active on 2026-04-17
- **WHEN** `Resolve(W1, P1, 2026-04-17)` is called
- **THEN** the workspace default is returned

#### Scenario: No rule anywhere returns no-rate sentinel

- **GIVEN** no active rule exists at any level for 2026-04-17
- **WHEN** `Resolve(W1, P1, 2026-04-17)` is called
- **THEN** a no-rate sentinel is returned
- **AND** the resulting entry snapshot columns MUST all be NULL
- **AND** the UI MUST visibly distinguish `No rate` from `0.00` in rate columns

#### Scenario: Resolution uses the entry's started_at UTC date

- **GIVEN** a time entry with `started_at = 2026-03-31T23:30:00-04:00` (local) which is `2026-04-01T03:30:00Z` UTC
- **AND** rule `R1` is effective through 2026-03-31 and rule `R2` is effective from 2026-04-01
- **WHEN** the entry is stopped and `Resolve` is called
- **THEN** the resolver MUST return `R2` (because `at.UTC()::date = 2026-04-01`)

#### Scenario: Historical entry's snapshot is stable across subsequent rule edits

- **GIVEN** a closed entry snapshotted against rule `R1` at 10000 minor units
- **WHEN** `R1` is edited in any way the system still permits (e.g., `effective_to` extended into the future)
- **THEN** the entry's `hourly_rate_minor` MUST remain 10000
- **AND** subsequent reports over that entry MUST show the original figure
