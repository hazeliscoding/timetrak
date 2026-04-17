## ADDED Requirements

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
