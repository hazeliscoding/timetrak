## MODIFIED Requirements

<!--
  NO-OP DELTA. This change is prose-only: it rewrites the non-normative
  `## Purpose` section of `openspec/specs/reporting/spec.md` and does NOT
  alter any requirement. The requirement block below is copied verbatim
  from the baseline so the OpenSpec validator recognises a MODIFIED delta.
  See proposal.md and design.md (Decision 1).
-->

### Requirement: Workspace-scoped reports

All reports MUST be scoped to the active workspace. The system MUST never include time entries, clients, or projects from a workspace the current user is not a member of.

#### Scenario: Other workspace data is excluded
- **GIVEN** Alice is a member of `W1` only
- **WHEN** Alice opens any report
- **THEN** only data with `workspace_id = W1` SHALL be aggregated and displayed
