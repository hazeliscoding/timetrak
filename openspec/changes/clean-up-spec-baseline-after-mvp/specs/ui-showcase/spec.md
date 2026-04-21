## MODIFIED Requirements

<!--
  NO-OP DELTA. This change is prose-only: it rewrites the non-normative
  `## Purpose` section of `openspec/specs/ui-showcase/spec.md` and does NOT
  alter any requirement. The requirement block below is copied verbatim
  from the baseline so the OpenSpec validator recognises a MODIFIED delta.
  See proposal.md and design.md (Decision 1).
-->

### Requirement: Dev-only showcase surface

TimeTrak SHALL expose a developer-facing showcase surface mounted at `/dev/showcase` (plus sub-routes for the token catalogue and component catalogue) that is reachable ONLY when the application is running with `APP_ENV=dev`. In any non-development environment, the route MUST NOT be registered at server startup AND the handler MUST return HTTP 404 if invoked. The showcase MUST NOT be linked from any user-facing template, navigation, or footer.

#### Scenario: Showcase reachable in dev environment

- **WHEN** the server is started with `APP_ENV=dev` and an authenticated session requests `GET /dev/showcase`
- **THEN** the response status is 200 and the showcase index is rendered

#### Scenario: Showcase unreachable in production environment

- **WHEN** the server is started with a non-dev `APP_ENV` (for example `prod` or `staging`) and any client requests `GET /dev/showcase` or any `/dev/showcase/*` sub-route
- **THEN** the response status is 404
- **AND** the route MUST NOT appear in the registered route table

#### Scenario: Showcase is not linked from user-facing templates

- **WHEN** any shipped user-facing template (layouts, navigation, footer, dashboards, domain pages) is rendered
- **THEN** it MUST NOT contain a hyperlink whose href points at `/dev/showcase` or any sub-route
