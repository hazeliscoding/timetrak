# ui-partials Specification

## Purpose
TBD - created by archiving change create-reusable-ui-partials-and-patterns. Update Purpose after archive.
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

The system SHALL provide a canonical `partials/table_shell` that wraps a `<table>` with consistent head, body slot, and empty-state slot. The system SHALL provide a canonical `partials/empty_state` that renders a copy-first empty block (title, body, optional action) and SHALL NOT rely on color or iconography to convey meaning.

#### Scenario: Table renders with zero rows
- **WHEN** `partials/table_shell` is rendered with an empty row set
- **THEN** it MUST render `partials/empty_state` in place of the tbody rows and MUST preserve the table's accessible name

#### Scenario: Table renders with rows
- **WHEN** `partials/table_shell` is rendered with one or more rows
- **THEN** it MUST render the row slot inside `<tbody>` and MUST NOT render the empty-state slot

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
