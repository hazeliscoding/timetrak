# rates Specification

## Purpose
TBD - created by archiving change bootstrap-timetrak-mvp. Update Purpose after archive.
## Requirements
### Requirement: Rate rule storage with effective date windows

The system SHALL store billing rates as `rate_rules` with `workspace_id`, optional `client_id`, optional `project_id`, `currency_code`, a non-negative `hourly_rate_minor` (money as integer minor units), `effective_from` (date), and `effective_to` (nullable date; NULL means open-ended). A workspace-default rule has `client_id IS NULL AND project_id IS NULL`. A client-level override has `client_id` set and `project_id IS NULL`. A project-level override has `project_id` set.

#### Scenario: Workspace-default rule creation
- **WHEN** Alice creates a workspace default of 10000 minor units effective from 2026-01-01 with no end date
- **THEN** a `rate_rules` row is stored with `workspace_id = W1`, `client_id = NULL`, `project_id = NULL`, `hourly_rate_minor = 10000`, `effective_from = 2026-01-01`, `effective_to = NULL`

#### Scenario: Money stored as integer minor units
- **WHEN** any rate rule is created or edited
- **THEN** the persisted `hourly_rate_minor` MUST be an integer
- **AND** the system MUST NOT persist or compute rates using floating-point types

#### Scenario: Non-negative rate enforced
- **WHEN** a create/edit attempts `hourly_rate_minor < 0`
- **THEN** the system MUST reject at application validation
- **AND** the database check constraint MUST reject the row if validation is bypassed

### Requirement: No overlapping date windows at the same level

The system MUST reject a rate-rule create or edit that would cause two rules at the same precedence level (workspace-default, same client, or same project) to overlap in their effective date ranges.

#### Scenario: Overlapping workspace-default rejected
- **GIVEN** a workspace default exists with `effective_from = 2026-01-01`, `effective_to = NULL`
- **WHEN** a second workspace default is submitted with `effective_from = 2026-06-01`, `effective_to = NULL`
- **THEN** the submission MUST be rejected with a validation error
- **AND** the new row is not inserted

#### Scenario: Adjacent, non-overlapping windows are allowed
- **GIVEN** a workspace default with `effective_from = 2026-01-01`, `effective_to = 2026-06-30`
- **WHEN** a new workspace default is submitted with `effective_from = 2026-07-01`, `effective_to = NULL`
- **THEN** the submission MUST be accepted

### Requirement: Centralized rate resolution with precedence

The system SHALL provide a single rate-resolution function `Resolve(workspaceID, projectID, at)` that returns the applicable rate by consulting rules in this precedence order:

1. project-level rule for `project_id = projectID` active at `at`
2. client-level rule for `client_id = project's client_id` active at `at`
3. workspace-default rule for `workspace_id = workspaceID` active at `at`
4. no-rate sentinel

A rule is "active at `at`" when `effective_from <= at::date AND (effective_to IS NULL OR at::date <= effective_to)`. All rate reads (reporting, future invoicing) MUST go through this function.

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
- **AND** downstream billable-value computation MUST treat this entry as contributing zero billable amount
- **AND** the UI MUST visibly distinguish `No rate` from `0.00` in rate columns

#### Scenario: Resolution respects historical entries
- **GIVEN** a time entry started on 2026-02-15 and the workspace-default rate changed effective 2026-04-01
- **WHEN** `Resolve(W1, P1, 2026-02-15)` is called for that entry
- **THEN** the rule active on 2026-02-15 is returned (not the later one)

### Requirement: Rate management UI accessibility

The rate rules management UI MUST meet WCAG 2.2 AA. Money inputs MUST clearly indicate the currency and the unit (e.g., a label like `Rate (USD per hour)` and a helper describing that amounts are entered as decimals and stored as minor units by the system). Effective-date inputs MUST use native `<input type="date">`. Validation errors (overlaps, negative rate) MUST be conveyed by text next to the offending field and associated via `aria-describedby`.

#### Scenario: Keyboard-only rate rule creation
- **WHEN** a keyboard-only user creates a workspace-default rate rule
- **THEN** they can complete and submit the form without a pointer
- **AND** all fields have visible labels and visible focus rings

#### Scenario: Overlap error is announced
- **WHEN** an overlap validation error occurs
- **THEN** the error text appears adjacent to the effective-date field
- **AND** the date inputs reference the error via `aria-describedby`
- **AND** the error is not conveyed by color alone

