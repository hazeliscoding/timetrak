## MODIFIED Requirements

<!--
  NO-OP DELTA. This change is prose-only: it rewrites the non-normative
  `## Purpose` section of `openspec/specs/rates/spec.md` and does NOT
  alter any requirement. The requirement block below is copied verbatim
  from the baseline so the OpenSpec validator recognises a MODIFIED delta.
  See proposal.md and design.md (Decision 1).
-->

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
