## ADDED Requirements

### Requirement: Showcase covers the dashboard surface in every documented state

The `/dev/showcase` catalogue SHALL include a `dashboard-states` section at `/dev/showcase/dashboard-states` that documents the dashboard surface's three documented states (zero, idle, running) side-by-side on one page. Each state MUST appear as a distinct labeled block, MUST cite the `ui-partials` requirement it satisfies, and MUST either render the state's canonical markup inline (for states renderable without domain fixtures, like the zero state which is just the `empty_state` partial) or cross-link to the component showcase entries that compose the state (for the idle/running states whose rendering is driven by the `timer_control` and `dashboard_summary` partials which already have their own showcase entries).

#### Scenario: Dashboard states section renders in dev

- **WHEN** an authenticated developer requests `GET /dev/showcase/dashboard-states` with `APP_ENV=dev`
- **THEN** the response status is 200
- **AND** the page renders three labeled blocks: zero state, idle state, running state
- **AND** the zero-state block renders a live `empty_state` partial matching the dashboard template's zero-state copy and action
- **AND** the idle-state and running-state blocks describe the rendering in prose with deep links to the `timer_control` and `dashboard_summary` component showcase entries that compose those states
- **AND** each block cites the `ui-partials` requirement that governs its rendering

#### Scenario: A future change modifies the dashboard surface without updating the showcase

- **WHEN** a proposed change modifies the dashboard's state rendering (e.g. introduces a fourth state or restructures an existing one)
- **THEN** the change MUST update `web/templates/showcase/dashboard_states.html` in the same change
- **AND** the showcase-axe-smoke test MUST exercise the new or modified block

### Requirement: Showcase covers every live empty-state consumer

The `/dev/showcase` catalogue SHALL include an `empty-states` section that renders every live `partials/empty_state` consumer in the product (every surface whose zero-rows view delegates to the partial) as a distinct labeled block on one page. For each consumer, the block MUST show the partial with the exact context keys the consumer uses in production, MUST label the surface (e.g. "clients/index — no clients yet"), and MUST cite the template file. When a consumer uses the `Live` flag, the showcase block MUST render with `Live: true` so the `aria-live="polite"` wrapper is visible.

#### Scenario: Empty states section renders in dev

- **WHEN** an authenticated developer requests `GET /dev/showcase/empty-states` with `APP_ENV=dev`
- **THEN** the response status is 200
- **AND** the page renders one labeled block per live empty-state consumer
- **AND** each block uses the exact production context keys for that consumer

#### Scenario: A new empty-state consumer is added

- **WHEN** a new change adds a collection surface whose zero-rows view delegates to `partials/empty_state`
- **THEN** the change MUST add a corresponding block to `web/templates/showcase/empty_states.html`
- **AND** the showcase-axe-smoke test MUST exercise the new block

#### Scenario: A future automated coverage test enforces one-to-one mapping

- **WHEN** the showcase-coverage enforcement is mechanized (a follow-up to this change)
- **THEN** the test SHALL enumerate every template that invokes `{{template "empty_state" ...}}` and assert each appears as a labeled block in `web/templates/showcase/empty_states.html`
- **AND** a production surface without a matching showcase block SHALL cause the test to fail
- **NOTE:** This scenario captures the target enforcement state. In this change's scope, coverage is maintained by review; the enforcement test is queued as a follow-up in the archived tasks.md.
