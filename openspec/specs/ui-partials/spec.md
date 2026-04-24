# ui-partials Specification

## Purpose
The ui-partials capability defines how reusable template fragments are
authored, invoked, and documented in TimeTrak. It fixes the canonical
location and naming (`web/templates/partials/<name>.html` with block
name `partials/<name>`), the slot-and-context convention using the
`dict` template func, the HTMX event-name contract for peer-refresh
events, row partial conventions for out-of-band swaps, focus management
after HTMX swaps, and the documented set of canonical partials (form
field, error summary, table shell, empty state, flash, spinner,
pagination, filter bar). Each canonical partial carries per-partial
accessibility obligations enumerated in `web/templates/partials/README.md`.
## Requirements
### Requirement: Canonical partial location and naming

The system SHALL host reusable UI partials under `web/templates/partials/` where each partial is one file named `<name>.html` containing exactly one block defined as `{{define "partials/<name>"}}`. Domain templates SHALL invoke partials via `{{template "partials/<name>" .Context}}`.

#### Scenario: A new shared block is added
- **WHEN** a markup pattern is used by two or more domain templates and is promoted to a canonical partial
- **THEN** it MUST live at `web/templates/partials/<name>.html` with block name `partials/<name>` and MUST be listed in `web/templates/partials/README.md`

#### Scenario: A domain-only block is proposed for extraction
- **WHEN** a markup pattern is used by only one domain template
- **THEN** it SHALL remain in the domain template and SHALL NOT be extracted into `web/templates/partials/`

### Requirement: Partial slot and context convention

Each canonical partial SHALL document the shape of `.` it expects (required keys, optional keys, defaults) in `web/templates/partials/README.md`. Optional slots SHALL be passed via the existing `dict` template func using documented keys. A partial SHALL NOT require more than four optional slot keys; exceeding that threshold requires splitting the partial.

#### Scenario: Caller passes optional slot
- **WHEN** a domain template invokes a canonical partial with a documented optional key via `dict`
- **THEN** the partial SHALL render using the provided value

#### Scenario: Caller omits optional slot
- **WHEN** a domain template invokes a canonical partial without an optional key
- **THEN** the partial SHALL render using the documented default and SHALL NOT error

### Requirement: HTMX event contract documentation

The system SHALL maintain a single authoritative list in `web/templates/partials/README.md` of HTMX peer-refresh event names (`timer-changed`, `entries-changed`, `clients-changed`, `projects-changed`, `rates-changed`) and, for each canonical partial, document which of these events the partial emits via `HX-Trigger` and which it listens for via `hx-trigger="... from:body"`.

#### Scenario: A partial changes its emitted event set
- **WHEN** a canonical partial is modified to emit or stop emitting an `*-changed` event
- **THEN** the partials README MUST be updated in the same change and existing listeners MUST be audited

#### Scenario: A new domain reuses the contract
- **WHEN** a new domain template composes a canonical partial
- **THEN** the domain SHALL rely on the documented event names without inventing domain-specific variants

### Requirement: Row partial conventions for OOB swap

Row partials (`client_row`, `project_row`, `entry_row`, `rate_row`, and future domain equivalents) SHALL render a stable root element with `id="<domain>-row-<uuid>"` so that server responses MAY target them via `hx-swap-oob="true"`. Each row partial SHALL document the `*-changed` event that MUST be emitted by handlers mutating that row.

#### Scenario: Server returns an updated row out-of-band
- **WHEN** a handler mutates a domain record and returns the re-rendered row with `hx-swap-oob="true"`
- **THEN** the browser SHALL replace the element whose id matches `<domain>-row-<uuid>` and peer lists MUST refresh via the documented `*-changed` event

### Requirement: Focus management after HTMX swaps

Canonical partials that are swap targets SHALL either carry `data-focus-after-swap` on a sensible focus target (primary input, first actionable control, or the swapped container with `tabindex="-1"`) or document in the README why no explicit focus target is required. Modal and dialog swaps MUST set `data-focus-after-swap`.

#### Scenario: Form partial is swapped in after validation error
- **WHEN** a form partial re-renders with inline errors via HTMX
- **THEN** the first invalid control (or the form errors summary) MUST carry `data-focus-after-swap` so keyboard and screen-reader users land on the error

#### Scenario: Modal or dialog partial is swapped in
- **WHEN** a modal or `<dialog>` partial is rendered into the page via HTMX
- **THEN** the partial MUST set `data-focus-after-swap` on an element inside the dialog

### Requirement: Form field and error summary partials

The system SHALL provide a canonical `partials/form_field` that renders a visible `<label>` bound to exactly one native control, an optional hint, and an inline error region linked via `aria-describedby`. The system SHALL provide a canonical `partials/form_errors` that renders a top-of-form summary with `role="alert"` when the form has one or more validation errors.

#### Scenario: Field renders with a validation error
- **WHEN** `partials/form_field` is rendered with an error value
- **THEN** the control MUST have `aria-invalid="true"` and the error region MUST be associated via `aria-describedby`

#### Scenario: Form submits with multiple errors
- **WHEN** server-side validation fails with more than one error
- **THEN** `partials/form_errors` MUST render the list with `role="alert"` and MUST be focusable so `data-focus-after-swap` can target it

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

### Requirement: Flash, spinner, pagination, and filter bar partials

The system SHALL provide canonical `partials/flash`, `partials/spinner`, `partials/pagination`, and `partials/filter_bar` partials. `flash` SHALL map severity levels `info`, `success`, `warn`, and `error` to appropriate ARIA roles (`role="status"` for non-urgent, `role="alert"` for urgent). `spinner` SHALL carry `aria-live="polite"` and SHALL NOT be the sole cue for completion. `pagination` SHALL expose prev/next controls with accessible names. `filter_bar` SHALL use native form controls and SHALL debounce change-driven HTMX requests.

#### Scenario: Success flash after create
- **WHEN** a handler returns a flash message with severity `success` via OOB swap
- **THEN** `partials/flash` MUST render with `role="status"` and MUST NOT steal focus

#### Scenario: Error flash after destructive failure
- **WHEN** a handler returns a flash message with severity `error`
- **THEN** `partials/flash` MUST render with `role="alert"` so assistive tech announces it immediately

### Requirement: Per-partial accessibility obligations documented

`web/templates/partials/README.md` SHALL document, for each canonical partial, its WCAG 2.2 AA obligations — label source, focus target, non-color status conveyance, and target-size expectations. Future UI changes SHALL cite the README instead of re-deriving these obligations.

#### Scenario: New UI change references a canonical partial
- **WHEN** a later change composes a canonical partial
- **THEN** its tasks.md SHALL reference the README's accessibility entry for that partial rather than re-specifying it

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

### Requirement: Brand mark partial

The system SHALL expose a canonical brand-mark partial at `web/templates/partials/brandmark.html` that renders TimeTrak's wordmark as an inline SVG. The partial MUST consume only semantic-alias and `currentColor` values for fill and stroke — specifically `currentColor`, `var(--color-text)`, and `var(--color-accent)` — and MUST NOT reference primitive ramps, raw hex or rgb values, or any new semantic alias. The partial MUST accept a `dict` with two keys: `Size` (string, one of `sm` or `md`; defaults to `md` when empty) and `Decorative` (bool; defaults to `false`). When `Decorative` is `false` the rendered SVG MUST carry `role="img"` and a child `<title>TimeTrak</title>` so assistive technology announces the mark as a graphic named "TimeTrak". When `Decorative` is `true` the SVG MUST carry `aria-hidden="true"` and MUST NOT emit a `<title>` element. The partial MUST be listed in `web/templates/partials/README.md` alongside the other canonical partials and MUST NOT be duplicated or re-implemented in any domain template.

#### Scenario: Default non-decorative render from the app header

- **WHEN** `web/templates/layouts/app.html` invokes `{{template "brandmark" (dict "Size" "md" "Decorative" false)}}`
- **THEN** the rendered SVG carries `role="img"` and contains a `<title>TimeTrak</title>` child
- **AND** the SVG's fill and stroke reference only `currentColor`, `var(--color-text)`, or `var(--color-accent)`
- **AND** no raw hex, rgb, hsl, or named colour value appears in the rendered output

#### Scenario: Decorative render adjacent to text that already names the product

- **WHEN** a surface invokes `{{template "brandmark" (dict "Size" "sm" "Decorative" true)}}`
- **THEN** the rendered SVG carries `aria-hidden="true"`
- **AND** the SVG does NOT contain a `<title>` child
- **AND** assistive technology skips the mark silently

#### Scenario: Token-contract compliance is enforced at authoring time

- **WHEN** a contributor adds or modifies `web/templates/partials/brandmark.html`
- **THEN** code review MUST reject any fill or stroke that references a primitive ramp, a raw colour value, or a new semantic alias
- **AND** adding a new semantic alias for brand purposes requires amending `openspec/specs/ui-foundation/spec.md` under its existing amendment rule, not this requirement

#### Scenario: Focus behavior when wrapped in an anchor

- **WHEN** the brandmark partial is rendered inside an anchor (e.g. the app header link)
- **THEN** the anchor SHALL inherit the global `:focus-visible` outline documented in `ui-foundation`
- **AND** the partial MUST NOT introduce a component-scoped focus override

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

### Requirement: Theme switch partial

The system SHALL provide a canonical `partials/theme_switch` that renders a single segmented control for the product's three themes (`light`, `dark`, `system`). The partial's root element SHALL be `<div role="radiogroup" aria-label="Theme">`. Each theme option SHALL render as a child `<button type="button" role="radio">` carrying `data-theme-set="<value>"`, both `aria-pressed="true|false"` and `aria-checked="true|false"` set consistently, and a visible `<span>` label paired with an `aria-hidden` leading glyph.

The partial's `dict` context is optional. When invoked without context (the production header case) it renders all three segments with `aria-pressed="false"` / `aria-checked="false"` — the existing client JS (`web/static/js/app.js`) synchronizes the active segment from `localStorage.timetrak.theme` after the FOUC head-script sets `data-theme`. When invoked with an explicit `InitialSelected` key (the showcase case) it renders with the matching segment pre-set, for documenting each of the three selected states statically.

The partial MUST NOT introduce any new runtime dependency, MUST NOT ship inline JS, and MUST reuse the existing `data-theme-set` click contract.

#### Scenario: Production header renders all three segments, none pre-selected

- **WHEN** `{{template "theme_switch" (dict)}}` is invoked from the app shell (no `InitialSelected` key)
- **THEN** the partial renders exactly one `role="radiogroup"` element containing exactly three `role="radio"` `<button>` children whose `data-theme-set` values are `light`, `dark`, and `system` respectively
- **AND** every child button initially carries `aria-pressed="false"` and `aria-checked="false"` (client JS synchronizes the active state post-parse)
- **AND** every child button has a visible label element and an `aria-hidden` leading glyph

#### Scenario: Showcase rendering pre-selects a segment

- **WHEN** `{{template "theme_switch" (dict "InitialSelected" "dark")}}` is invoked
- **THEN** the `dark` segment renders with `aria-pressed="true"` and `aria-checked="true"`
- **AND** the other two segments render with `aria-pressed="false"` and `aria-checked="false"`

#### Scenario: Keyboard focus lands on the active segment

- **WHEN** a keyboard user tabs into the theme switch
- **THEN** focus lands on the currently-active segment (the one with `aria-pressed="true"`), matching the standard radiogroup keyboard contract
- **AND** the visible focus ring on that segment satisfies the accepted `ui-foundation` focus-indicator contract

