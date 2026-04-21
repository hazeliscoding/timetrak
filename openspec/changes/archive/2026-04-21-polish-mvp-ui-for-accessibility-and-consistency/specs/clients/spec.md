## ADDED Requirements

### Requirement: Clients list SHALL present accessible table semantics

The clients list table SHALL include a `<caption>` element (visually hidden where the page header already conveys the same label), `<th scope="col">` on every header cell, and `<th scope="row">` on the first cell of each row when that cell identifies the client. Columns containing numeric data SHALL be aligned via a CSS utility class, not inline `style` attributes, and SHALL use `font-variant-numeric: tabular-nums`.

Empty-state rendering (no clients yet, or all clients filtered out) SHALL render a dedicated empty-state partial inside an element with `aria-live="polite"` so screen readers hear the transition after HTMX-driven filter changes.

#### Scenario: Table has caption and column scopes

- **GIVEN** a workspace has at least one client
- **WHEN** a user loads `/clients`
- **THEN** the clients `<table>` MUST contain a `<caption>` (sr-only is acceptable)
- **AND** every `<th>` in `<thead>` MUST carry `scope="col"`
- **AND** the first cell of each `<tr>` in `<tbody>` that identifies the client MUST be a `<th scope="row">`

#### Scenario: Empty state announces via live region

- **GIVEN** a workspace has no clients
- **WHEN** `/clients` renders
- **THEN** the empty-state container MUST carry `aria-live="polite"`
- **AND** the text MUST state the empty condition in domain-specific copy (e.g. "No clients yet")

### Requirement: Client status pills SHALL NOT rely on color alone

The `Archived` pill and any other client-status indicator SHALL render a text label in addition to any color treatment. The pill's text-on-background combination SHALL pass a 4.5:1 contrast ratio against its background.

#### Scenario: Archived pill carries text

- **GIVEN** a client is archived
- **WHEN** the client row renders
- **THEN** the archived pill MUST contain the visible text `Archived`
- **AND** the text-on-background contrast MUST be at least 4.5:1

### Requirement: Destructive client actions SHALL use the documented confirmation pattern

Deleting a client that has no projects and no time entries SHALL use native `hx-confirm` confirmation. Archiving a client that has active projects or historical time entries SHALL use the `<dialog>`-based confirmation partial that enumerates the side effects before confirming.

After a successful delete, focus SHALL land on a stable anchor on the page (the `New client` button) via `data-focus-after-swap`. After a cancelled delete, focus SHALL return to the row's action button that invoked the confirmation.

#### Scenario: Delete a client with no projects

- **GIVEN** a client has no projects and no time entries
- **WHEN** a user clicks `Delete` on that row
- **THEN** a native `confirm()` dialog MUST appear (via `hx-confirm`)
- **AND** on confirmation, the row MUST be removed and focus MUST land on the `New client` button

#### Scenario: Archive a client with active projects shows side effects

- **GIVEN** a client has at least one active project or historical time entry
- **WHEN** a user clicks `Archive`
- **THEN** the `<dialog>`-based confirmation MUST open
- **AND** the dialog MUST enumerate the number of projects and entries affected in plain text
- **AND** focus MUST be trapped inside the dialog while it is open
- **AND** on close (confirm or cancel), focus MUST return to the `Archive` button that opened it

### Requirement: Client form SHALL meet WCAG 2.2 AA accessibility

The client create/edit form SHALL wire inline validation errors via `aria-describedby`, mark invalid inputs with `aria-invalid="true"`, render a top-of-form error summary with `role="alert"` and `data-focus-after-swap` on validation failure, and mark required fields with visible text and `aria-required="true"`.

#### Scenario: Creating a client with missing name

- **GIVEN** a user submits the client create form with an empty name
- **WHEN** the server re-renders the form via HTMX
- **THEN** the name input MUST carry `aria-invalid="true"` and `aria-describedby` pointing at a visible error element
- **AND** a top-of-form `role="alert"` summary MUST list the error
- **AND** focus MUST land on the summary after the swap
