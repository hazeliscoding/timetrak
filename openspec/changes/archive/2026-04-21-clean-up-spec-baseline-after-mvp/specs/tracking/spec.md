## MODIFIED Requirements

<!--
  NO-OP DELTA. This change is prose-only: it rewrites the non-normative
  `## Purpose` section of `openspec/specs/tracking/spec.md` and does NOT
  alter any requirement. The requirement block below is copied verbatim
  from the baseline so the OpenSpec validator recognises a MODIFIED delta.
  See proposal.md and design.md (Decision 1).
-->

### Requirement: Start a timer

The system SHALL allow a workspace member to start a running timer for a non-archived project in the active workspace. Starting a timer MUST create a `time_entries` row with `started_at = now()`, `ended_at = NULL`, `duration_seconds = 0` (or null), `workspace_id` set to the active workspace, and `user_id` set to the acting user. The timer MAY optionally reference a project-scoped `task_id`, a description, and a `is_billable` flag (defaulting to the project's `default_billable`).

#### Scenario: Successful timer start
- **GIVEN** Alice has no running timer in workspace `W1`, and project `P1` is non-archived in `W1`
- **WHEN** Alice starts a timer on `P1`
- **THEN** a `time_entries` row is created with `workspace_id = W1`, `user_id = Alice`, `project_id = P1`, `started_at = now()`, `ended_at = NULL`
- **AND** the timer widget updates to show the running entry

#### Scenario: Starting while another timer is running is rejected
- **GIVEN** Alice already has a running timer in workspace `W1`
- **WHEN** Alice attempts to start a second timer in `W1`
- **THEN** the system MUST reject the request with HTTP 409 and an actionable error
- **AND** no second `time_entries` row is created
- **AND** the pre-existing running entry is unchanged

#### Scenario: Starting a timer on an archived project is rejected
- **GIVEN** project `P1` is archived
- **WHEN** Alice attempts to start a timer on `P1`
- **THEN** the system MUST reject the request with a validation error
- **AND** no `time_entries` row is created

#### Scenario: Starting a timer on another workspace's project is blocked
- **GIVEN** project `P2` belongs to workspace `W2`, Alice is not a member of `W2`
- **WHEN** Alice attempts to start a timer on `P2`
- **THEN** the system MUST respond with HTTP 404
- **AND** no `time_entries` row is created
