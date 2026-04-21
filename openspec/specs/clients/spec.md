# clients Specification

## Purpose
The clients capability governs workspace-scoped lifecycle management of
the billable client entity: creation, editing, archival and unarchival,
and the list view from which other domains (projects, entries, rates)
hang. Every clients handler MUST enforce workspace membership and return
HTTP 404 for cross-workspace access, verified by exhaustive per-handler
denial tests. Every clients UI surface MUST meet WCAG 2.2 AA, present
accessible table semantics, never convey status with color alone, and
use the documented confirmation pattern for destructive actions.
## Requirements
### Requirement: Create a client in the active workspace

The system SHALL allow a workspace member to create a client with at least a non-empty `name`. The client MUST be scoped to the active workspace (`clients.workspace_id = active workspace`).

#### Scenario: Successful client creation
- **GIVEN** Alice's active workspace is `W1`
- **WHEN** Alice submits the create-client form with name `Acme Co.`
- **THEN** a `clients` row is inserted with `workspace_id = W1` and `name = 'Acme Co.'`
- **AND** Alice is returned to the clients list with the new client visible

#### Scenario: Empty name is rejected
- **WHEN** the create-client form is submitted with an empty name
- **THEN** the system MUST reject the submission with a validation error
- **AND** no `clients` row is created

### Requirement: Edit a client

A workspace member SHALL be able to edit an existing client's editable fields (at minimum `name` and `contact_email`). Edits MUST only apply to clients in the member's active workspace.

#### Scenario: Successful edit
- **GIVEN** a client `C1` in workspace `W1`, and Alice is a member of `W1`
- **WHEN** Alice edits `C1.name` to `Acme LLC.`
- **THEN** the stored `clients.name` equals `Acme LLC.`

#### Scenario: Edit attempt across workspaces is blocked
- **GIVEN** client `C1` belongs to workspace `W2`, and Alice is not a member of `W2`
- **WHEN** Alice attempts to edit `C1`
- **THEN** the system MUST respond with HTTP 404
- **AND** `C1` is unchanged

### Requirement: Archive and unarchive a client

The system SHALL support archiving a client by setting `is_archived = true`, and unarchiving by setting it back to `false`. Archiving MUST NOT delete the client or any related projects or time entries. Archived clients MUST be excluded from default lists and from timer-start selection, but MUST remain visible in historical reports.

#### Scenario: Archive hides client from default list
- **WHEN** Alice archives client `C1`
- **THEN** `C1.is_archived` becomes `true`
- **AND** `C1` no longer appears in the default clients list
- **AND** `C1` still appears when "Show archived" is enabled
- **AND** projects under `C1` remain intact
- **AND** existing time entries and reports still reference `C1`

#### Scenario: Archived client is not selectable for new projects or timers
- **GIVEN** `C1` is archived
- **WHEN** Alice opens the create-project form or the timer-start project picker
- **THEN** `C1` and its non-archived projects MUST NOT appear in the default selection list

### Requirement: Clients list view

The system SHALL present a list view of clients in the active workspace with, at minimum: name, contact email, project count, and archived status. The list MUST support an "Include archived" toggle and MUST render as a semantic HTML table.

#### Scenario: Default list excludes archived clients
- **GIVEN** workspace `W1` has three active clients and one archived client
- **WHEN** Alice opens the clients list
- **THEN** three rows are shown
- **AND** archived clients are shown only when "Include archived" is enabled

#### Scenario: Empty state
- **GIVEN** workspace `W1` has no clients
- **WHEN** Alice opens the clients list
- **THEN** the page MUST render an empty state with a clear primary action `New client`
- **AND** the empty state MUST NOT display a skeleton/loading state indefinitely

### Requirement: Clients UI accessibility

The clients list and client form MUST meet WCAG 2.2 AA. The list MUST be a semantic `<table>` with `<th scope="col">` headers; the archived indicator MUST use text and/or an icon, not color alone; all actions MUST be keyboard operable; focus MUST be preserved or explicitly moved after HTMX row swaps (e.g., after inline edit or archive).

#### Scenario: Archive action preserves focus context
- **GIVEN** Alice archives a client inline via HTMX
- **WHEN** the row partial is swapped
- **THEN** focus MUST move to a defined target (e.g., the Undo control or the next row)
- **AND** a status message MUST be announced via `aria-live`

#### Scenario: Archived status not color-only
- **WHEN** an archived client is displayed
- **THEN** its archived state MUST be indicated by visible text (for example `Archived`) in addition to any color treatment

### Requirement: Exhaustive cross-workspace denial for every clients handler

Every read and write handler in the `clients` family MUST return HTTP 404 with the shared not-found response body when invoked by a user whose active workspace does not own the referenced client. This rule applies without exception to: list view, detail view, create, edit, archive, unarchive, and delete. The response body MUST NOT disclose the existence, name, or owning workspace of the target resource.

#### Scenario: List view is scoped to active workspace
- **GIVEN** Alice's active workspace is `W1` and clients exist in both `W1` and `W2`
- **WHEN** Alice requests the clients list
- **THEN** only clients with `workspace_id = W1` are returned
- **AND** no clients from `W2` appear in the rendered table

#### Scenario: Detail view for other-workspace client returns 404
- **GIVEN** client `C2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice requests `GET /clients/C2`
- **THEN** the system MUST respond with HTTP 404
- **AND** the response body MUST be the shared not-found template with no mention of `C2`

#### Scenario: Create cannot target another workspace
- **GIVEN** Alice's active workspace is `W1`
- **WHEN** Alice submits a client-create form whose body attempts to set `workspace_id = W2`
- **THEN** the resulting `clients` row MUST have `workspace_id = W1`
- **AND** no row with `workspace_id = W2` is created

#### Scenario: Edit across workspaces returns 404
- **GIVEN** client `C2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice POSTs an edit to `C2`
- **THEN** the system MUST respond with HTTP 404
- **AND** no row in `clients` is modified

#### Scenario: Archive across workspaces returns 404
- **GIVEN** client `C2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice POSTs an archive request for `C2`
- **THEN** the system MUST respond with HTTP 404
- **AND** `C2.archived_at` is unchanged

#### Scenario: Delete across workspaces returns 404
- **GIVEN** client `C2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice POSTs a delete request for `C2`
- **THEN** the system MUST respond with HTTP 404
- **AND** the row for `C2` still exists in the database

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

