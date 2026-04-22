## MODIFIED Requirements

<!--
  NO-OP DELTA. This change (transition-to-stage-3) is prose-only: it
  updates framing in CLAUDE.md, openspec/config.yaml, and adds a new
  narrative doc under docs/. No requirement under openspec/specs/ is
  added, modified, removed, or renamed.

  The requirement block below is copied verbatim from
  openspec/specs/workspace/spec.md so the OpenSpec validator recognises
  a parseable MODIFIED delta. See proposal.md and design.md Decision 4
  for rationale; the same pattern is used in the archived change
  2026-04-21-clean-up-spec-baseline-after-mvp.
-->

### Requirement: Default personal workspace on signup

The system SHALL automatically create a default personal workspace for each new user during registration and add that user as the workspace `owner` in the same transaction as user creation.

#### Scenario: First workspace created on signup
- **GIVEN** a user completes signup
- **THEN** exactly one `workspaces` row is created for that user
- **AND** a `workspace_members` row exists with role `owner`
- **AND** the session's active workspace is set to that workspace
