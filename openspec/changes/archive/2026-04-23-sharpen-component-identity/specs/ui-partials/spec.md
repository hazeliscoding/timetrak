## ADDED Requirements

### Requirement: Status chip partial

The system SHALL provide a canonical `partials/status_chip` that renders a status or metadata chip with the following contract:

- `dict` context: `{kind, label, variant, glyph?}`
  - `kind`: required semantic key, one of the enumerated values documented in `web/templates/partials/README.md` (initial set: `billable`, `non-billable`, `running`, `draft`, `archived`, `warning`).
  - `label`: required human-readable text rendered inside the chip.
  - `variant`: required, one of `filled` (accent-soft fill for billable/running; warning-soft for warning) or `outlined` (neutral border for non-billable/draft/archived). Enumerated — free-form values are prohibited.
  - `glyph`: optional leading glyph or unicode symbol; REQUIRED when `kind` conveys a state that must not rely on color alone (e.g. `running`, `draft`, `archived`).
- Rendered element: rectangular (`var(--radius-sm)`), height 20px, 6px horizontal padding, `text-xs`, medium weight.
- The chip MUST NOT be pill-shaped.
- The chip MUST pair its color signal with either a `glyph` or a distinct shape/position cue so state is never conveyed by color alone.

#### Scenario: Billable chip renders with filled variant

- **WHEN** `partials/status_chip` is invoked with `{kind: "billable", label: "Billable", variant: "filled"}`
- **THEN** the rendered element has `border-radius: var(--radius-sm)`, a `--color-accent-soft` fill, and the text `Billable`.

#### Scenario: Running chip omits a glyph

- **WHEN** `partials/status_chip` is invoked with `{kind: "running", label: "Running", variant: "filled"}` and no `glyph`
- **THEN** the partial MUST render a default indicator glyph (or the template system MUST fail review) so the state is not conveyed by color alone.

#### Scenario: Contributor adds a new `kind`

- **WHEN** a contributor proposes a new `kind` value (e.g. `overdue`)
- **THEN** the proposal MUST amend the enumerated `kind` list in this requirement and the partials README before the new kind can be used.

#### Scenario: Chip is authored as a pill

- **WHEN** a domain template invokes `partials/status_chip` with a `class` override attempting to set pill radius
- **THEN** the review blocks the change (see `ui-component-identity`: Shape language taxonomy).

### Requirement: Timer control partial

The system SHALL provide a canonical `partials/timer_control` that renders the workspace's time-tracking control with the following contract:

- `dict` context: `{state, running?}` where `state` is `idle` or `running`, and `running` is the running time entry (project, client, elapsed start timestamp) when `state == "running"`.
- Idle rendering: a start-entry form (project picker + optional description + submit) whose primary submit button is a pill (inherits `.btn-primary` pill styling) labelled `Start timer`. HTMX attributes post to `/timer/start` and swap the partial target.
- Running rendering: a single pill container (no form peers), `var(--radius-pill)`, `var(--color-accent-soft)` fill, 2px `var(--color-accent)` border, leading pulsing accent dot (static under `prefers-reduced-motion: reduce`), project name, tabular-nums elapsed `HH:MM:SS`, and a distinct `Stop` affordance rendered as a secondary/ghost button (NOT another `.btn-primary` pill).
- Emits `HX-Trigger: timer-changed, entries-changed` on state transitions, consistent with the existing HTMX event contract.
- `data-focus-after-swap` SHALL be applied to the primary actionable control in both states (idle: the Start pill; running: the Stop affordance) so keyboard users retain focus across swaps.

#### Scenario: Timer renders in idle state

- **WHEN** `partials/timer_control` is invoked with `{state: "idle"}`
- **THEN** the rendered element is a `--radius-pill` button with neutral styling, a neutral leading dot, and the label `Start timer`.

#### Scenario: Timer renders in running state

- **WHEN** `partials/timer_control` is invoked with `{state: "running", running: <entry>}`
- **THEN** the rendered element is a `--radius-pill` container with `--color-accent-soft` fill, a 2px `--color-accent` border, an accent leading dot, the project name, and an elapsed-time readout with `font-variant-numeric: tabular-nums`; a distinct `Stop` control is also rendered.

#### Scenario: Focus after HTMX swap on start

- **WHEN** the user starts a timer and the partial is swapped to running state
- **THEN** focus lands on the `Stop` affordance via `data-focus-after-swap`.

#### Scenario: Reduced-motion user views running timer

- **WHEN** the running timer renders under `@media (prefers-reduced-motion: reduce)`
- **THEN** the accent dot is static (no animation) and all other running-state styling is preserved.

## MODIFIED Requirements

### Requirement: Table shell and empty state partials

The system SHALL provide a canonical CSS contract on the shared `.table` class that every domain table (entries, clients, projects, rate rules, reports, showcase) consumes, enforcing TimeTrak's data-table visual identity. The system SHALL provide a canonical `partials/empty_state` that renders a copy-first empty block (title, body, optional action) and SHALL NOT rely on color or iconography to convey meaning. Domain tables that have no rows SHALL render `partials/empty_state` in place of their tbody rows and MUST preserve the table's accessible name.

The `.table` CSS contract MUST render tables with:

- hairline horizontal dividers only (`border-bottom: 1px solid var(--color-border)` on rows), no vertical dividers, no zebra striping;
- a body-row cell padding that yields a visual row height of approximately 40px, achieved via padding so content reflows accessibly (no fixed pixel height on `tr`);
- a hover state of `background: var(--color-surface-alt)` with no border shift;
- a selected/focused row state rendered as a 2px `var(--color-accent)` inside-left edge, flush to the cell padding — NOT as a full border or background fill;
- `<th>` elements styled in uppercase with letter-spacing `+0.04em`, `text-xs`, `var(--color-text-muted)`; column headers are the only uppercase text in the application;
- numeric columns, marked via `data-col-kind="numeric"` on `<th>` and `<td>` or via a `.col-num` class, rendered with `font-variant-numeric: tabular-nums` and `text-align: right`.

The previously-specified `partials/table_shell` wrapper partial SHALL NOT be provided; Go `html/template` cannot slot HTML blocks cleanly without an additional template helper, and the per-domain `<thead>` markup is intentionally co-located with each domain template. Introducing a slot-helper is architecture work that belongs in its own change.

#### Scenario: Domain table renders with zero rows

- **WHEN** a domain list template is rendered with an empty row set
- **THEN** it MUST render `partials/empty_state` in place of the tbody rows and MUST preserve the table's accessible name.

#### Scenario: Domain table renders with rows

- **WHEN** a domain list template is rendered with one or more rows
- **THEN** it MUST render the rows inside `<tbody>` and MUST NOT render the empty-state block.

#### Scenario: Table renders numeric column

- **WHEN** a column is marked `data-col-kind="numeric"` (or class `.col-num`)
- **THEN** each cell in that column MUST have `font-variant-numeric: tabular-nums` and `text-align: right`.

#### Scenario: Row is selected via keyboard or HTMX

- **WHEN** a row receives the selected/focused state (via `aria-selected="true"` or an equivalent focused affordance)
- **THEN** a 2px `var(--color-accent)` left-edge rule renders inside the row, with no background fill change and no border shift on other edges.

#### Scenario: Column header renders

- **WHEN** a `<th>` is rendered inside any `.table`
- **THEN** its text is uppercase with `letter-spacing: 0.04em`, `text-xs`, and `var(--color-text-muted)`.

#### Scenario: Row hover

- **WHEN** a pointer hovers a row inside `.table`
- **THEN** the row background changes to `var(--color-surface-alt)` and no border on the row shifts or grows.
