## ADDED Requirements

### Requirement: `partials/empty_state` is mandatory for collection zero-rows views

Every in-product surface that renders the zero-rows view of a list, table, filtered-collection result, or otherwise-bounded collection SHALL delegate its empty variant to `partials/empty_state`. Ad-hoc empty messaging — including inline `<p class="muted">` paragraphs, bespoke `<div class="muted">` blocks inside a card, or hand-rolled "no rows" markup — is non-compliant. Single-metric fallbacks (e.g. a summary cell with no computable value, rendered as an em-dash with a muted hint) are NOT collection zero-rows views and are therefore out of scope for this requirement.

#### Scenario: A domain list template renders its empty variant

- **WHEN** a domain list, table, or filtered-collection template renders its zero-rows view
- **THEN** it MUST invoke `{{template "partials/empty_state" (dict ...) }}` with at minimum `Title` and `Body` context keys
- **AND** it MUST NOT render any other text content inside the collection's container in the empty state

#### Scenario: An HTMX-delivered empty view replaces a populated one

- **WHEN** a filter change or peer-refresh event causes a previously-populated collection to render zero rows via an HTMX swap
- **THEN** the response MUST use `partials/empty_state` with `Live: true` so the partial emits `aria-live="polite"` on its wrapper

#### Scenario: A surface renders a single-metric fallback

- **WHEN** a non-collection surface (e.g. one cell in a multi-metric summary card) has no computable value
- **THEN** it MAY render `—` with a muted hint line instead of `partials/empty_state`
- **AND** the surface MUST NOT carry the visual weight of an `empty_state` (no heading, no primary action)

#### Scenario: A reviewer audits a new template

- **WHEN** a new UI-affecting change adds a collection surface to the product
- **THEN** the reviewer MUST verify the surface's empty variant uses `partials/empty_state`
- **AND** the browser contract suite MUST include coverage for that surface's empty variant

### Requirement: Dashboard surface renders three documented states

The dashboard page (`GET /dashboard`) SHALL render exactly one of three mutually exclusive states, selected by the workspace's data, and SHALL NOT render any state not listed here:

1. **Zero state** — no projects, no time entries, no running timer. The surface MUST render a single `partials/empty_state` card titled to orient the user toward the first setup step and carrying exactly one primary action linking to the next sensible route (the clients page under the current domain hierarchy). The timer control and summary card-row MUST NOT render in this state.
2. **Idle state** — one or more projects or entries exist and no timer is currently running. The surface MUST render the canonical timer control (in its idle identity, as accepted under `ui-component-identity`) and the summary card-row. No generic "Jump back in" or equivalent project-list card shall appear; the timer control's own project picker is the accepted affordance for starting a new timer.
3. **Running state** — a timer is currently running. The surface MUST render the canonical timer control (in its running identity) and the summary card-row. The summary card-row renders live values; the running-entry metadata is carried by the timer control, not duplicated.

This requirement binds the *surface*, not the partial structure: the timer control and the summary card-row are accepted elsewhere. What this requirement fixes is *which cards appear in which state* so a future change cannot silently reintroduce the three-always-on layout.

#### Scenario: A fresh workspace loads the dashboard

- **WHEN** an authenticated user with no projects, no time entries, and no running timer requests `GET /dashboard`
- **THEN** the response MUST render exactly one `partials/empty_state` card as the dashboard's sole content block
- **AND** the response MUST NOT contain a timer control, a summary card-row, or a "Jump back in" card
- **AND** the empty-state MUST carry exactly one primary action link

#### Scenario: A workspace with data loads the dashboard, no timer running

- **WHEN** an authenticated user with at least one project or entry and no running timer requests `GET /dashboard`
- **THEN** the response MUST render the canonical timer control in its idle identity and the summary card-row
- **AND** the response MUST NOT render a "Jump back in" card, a recent-projects list, or any other auxiliary project-selection surface

#### Scenario: A timer is running when the dashboard loads

- **WHEN** an authenticated user with a currently-running timer requests `GET /dashboard`
- **THEN** the response MUST render the canonical timer control in its running identity and the summary card-row
- **AND** the running-entry metadata MUST be carried by the timer control and MUST NOT be duplicated as a separate card

#### Scenario: A peer-refresh event transitions the dashboard between states

- **WHEN** `timer-changed` or `entries-changed` fires and the resulting state crosses a boundary (zero → idle, idle → running, running → idle, idle → zero if a user archives all projects mid-session)
- **THEN** the server-rendered response for the affected region MUST match the target state exactly as specified above
- **AND** no card from a different state may linger in the response
