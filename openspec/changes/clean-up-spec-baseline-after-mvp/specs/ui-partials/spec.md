## MODIFIED Requirements

<!--
  NO-OP DELTA. This change is prose-only: it rewrites the non-normative
  `## Purpose` section of `openspec/specs/ui-partials/spec.md` and does NOT
  alter any requirement. The requirement block below is copied verbatim
  from the baseline so the OpenSpec validator recognises a MODIFIED delta.
  See proposal.md and design.md (Decision 1).
-->

### Requirement: Canonical partial location and naming

The system SHALL host reusable UI partials under `web/templates/partials/` where each partial is one file named `<name>.html` containing exactly one block defined as `{{define "partials/<name>"}}`. Domain templates SHALL invoke partials via `{{template "partials/<name>" .Context}}`.

#### Scenario: A new shared block is added
- **WHEN** a markup pattern is used by two or more domain templates and is promoted to a canonical partial
- **THEN** it MUST live at `web/templates/partials/<name>.html` with block name `partials/<name>` and MUST be listed in `web/templates/partials/README.md`

#### Scenario: A domain-only block is proposed for extraction
- **WHEN** a markup pattern is used by only one domain template
- **THEN** it SHALL remain in the domain template and SHALL NOT be extracted into `web/templates/partials/`
