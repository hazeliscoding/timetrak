## MODIFIED Requirements

<!--
  NO-OP DELTA. This change is prose-only: it rewrites the non-normative
  `## Purpose` section of `openspec/specs/projects/spec.md` and does NOT
  alter any requirement. The requirement block below is copied verbatim
  from the baseline so the OpenSpec validator recognises a MODIFIED delta.
  See proposal.md and design.md (Decision 1).
-->

### Requirement: Create a project under a client

The system SHALL allow a workspace member to create a project under an existing, non-archived client in the active workspace. A project MUST have a non-empty `name` and MUST reference a `client_id` whose `workspace_id` matches the active workspace. A project MAY carry a `default_billable` flag (default `true`) and an optional short `code`.

#### Scenario: Successful project creation
- **GIVEN** Alice's active workspace is `W1` and non-archived client `C1` belongs to `W1`
- **WHEN** Alice submits the create-project form with name `Website redesign` under client `C1`
- **THEN** a `projects` row is inserted with `workspace_id = W1`, `client_id = C1`, `name = 'Website redesign'`, and `default_billable = true`

#### Scenario: Cross-workspace client rejected
- **GIVEN** client `C2` belongs to workspace `W2`, and Alice is not a member of `W2`
- **WHEN** Alice attempts to create a project under `C2`
- **THEN** the system MUST respond with HTTP 404
- **AND** no project is created

#### Scenario: Archived client cannot be parent of a new project
- **GIVEN** client `C1` in `W1` is archived
- **WHEN** a create-project request targets `C1`
- **THEN** the request MUST be rejected with a validation error
- **AND** no project is created
