## MODIFIED Requirements

<!--
  NO-OP DELTA. This change is prose-only: it rewrites the non-normative
  `## Purpose` section of `openspec/specs/clients/spec.md` and does NOT
  alter any requirement. The requirement block below is copied verbatim
  from the baseline so the OpenSpec validator recognises a MODIFIED delta.
  See proposal.md and design.md (Decision 1).
-->

### Requirement: Create a client in the active workspace

The system SHALL allow a workspace member to create a client with at least a non-empty `name`. The client MUST be scoped to the active workspace (`clients.workspace_id = active workspace`).

#### Scenario: Successful client creation
- **GIVEN** Alice's active workspace is `W1`
- **WHEN** Alice submits the create-client form with name `Acme Co.`
- **THEN** a `clients` row is inserted with `workspace_id = W1` and `name = 'Acme Co.'`
- **AND** Alice is returned to the clients list with the new client visible

#### Scenario: Empty name is rejected
- **WHEN** the create-client form is submitted with an empty name
- **THEN** the system MUST reject the submission with a validation error
- **AND** no `clients` row is created
