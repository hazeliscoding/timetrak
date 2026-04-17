# projects Specification

## Purpose
TBD - created by archiving change bootstrap-timetrak-mvp. Update Purpose after archive.
## Requirements
### Requirement: Create a project under a client

The system SHALL allow a workspace member to create a project under an existing, non-archived client in the active workspace. A project MUST have a non-empty `name` and MUST reference a `client_id` whose `workspace_id` matches the active workspace. A project MAY carry a `default_billable` flag (default `true`) and an optional short `code`.

#### Scenario: Successful project creation
- **GIVEN** Alice's active workspace is `W1` and non-archived client `C1` belongs to `W1`
- **WHEN** Alice submits the create-project form with name `Website redesign` under client `C1`
- **THEN** a `projects` row is inserted with `workspace_id = W1`, `client_id = C1`, `name = 'Website redesign'`, and `default_billable = true`

#### Scenario: Cross-workspace client rejected
- **GIVEN** client `C2` belongs to workspace `W2`, and Alice is not a member of `W2`
- **WHEN** Alice attempts to create a project under `C2`
- **THEN** the system MUST respond with HTTP 404
- **AND** no project is created

#### Scenario: Archived client cannot be parent of a new project
- **GIVEN** client `C1` in `W1` is archived
- **WHEN** a create-project request targets `C1`
- **THEN** the request MUST be rejected with a validation error
- **AND** no project is created

### Requirement: Edit a project

A workspace member SHALL be able to edit a project's editable fields (at minimum `name`, `code`, `default_billable`). Edits MUST be scoped to the active workspace.

#### Scenario: Successful edit
- **GIVEN** project `P1` in `W1`, Alice is a member of `W1`
- **WHEN** Alice edits `P1.default_billable` to `false`
- **THEN** the stored `projects.default_billable` equals `false`

### Requirement: Archive and unarchive a project

The system SHALL support archiving and unarchiving a project via `is_archived`. Archiving MUST NOT delete the project or its time entries. Archived projects MUST be excluded from default lists and from timer-start selection, and MUST remain referenced by historical reports.

#### Scenario: Archived project excluded from timer selection
- **GIVEN** project `P1` in `W1` is archived
- **WHEN** Alice opens the timer-start project picker
- **THEN** `P1` MUST NOT appear in the default selection list

#### Scenario: Archived project remains in reports
- **GIVEN** `P1` has historical time entries
- **WHEN** reports are generated for a date range that contains those entries
- **THEN** `P1`'s entries MUST continue to be included in report totals

### Requirement: Project workspace/client consistency invariant

The system MUST enforce that `projects.workspace_id` equals the `workspace_id` of the referenced client. This invariant MUST be enforced at the database layer via a composite foreign key from `projects (client_id, workspace_id)` to `clients (id, workspace_id)` (backed by a unique constraint on `clients (id, workspace_id)`), such that any attempt to insert or update a project row with a mismatched `workspace_id` fails with a referential integrity error independent of application code. The service layer MAY additionally reject the operation earlier with a user-facing 404, but the database constraint is the ultimate enforcement point.

#### Scenario: Mismatched insert rejected at database layer
- **WHEN** a project create/edit would set `workspace_id = W1` and `client_id = C` where `C.workspace_id = W2`
- **THEN** the INSERT or UPDATE MUST fail with a referential integrity error
- **AND** no `projects` row exists with that mismatch

#### Scenario: Service layer returns 404 before reaching database
- **GIVEN** Alice's active workspace is `W1` and client `C2` belongs to `W2`
- **WHEN** Alice submits a project-create form referencing `client_id = C2`
- **THEN** the system MUST respond with HTTP 404
- **AND** no INSERT is attempted

### Requirement: Projects list view

The system SHALL present a list view of projects in the active workspace showing, at minimum: name, client, default-billable flag, entry count (or recent activity indicator), and archived status. The list MUST be a semantic HTML table and MUST support an "Include archived" toggle.

#### Scenario: Default list excludes archived
- **GIVEN** `W1` contains three active and one archived project
- **WHEN** Alice opens the projects list
- **THEN** three rows are shown by default

#### Scenario: Empty state
- **GIVEN** `W1` has no projects
- **WHEN** Alice opens the projects list
- **THEN** the page renders an empty state with a primary action `New project`

### Requirement: Projects UI accessibility

The projects list and form MUST meet WCAG 2.2 AA: semantic `<table>`, programmatically associated labels on all form controls, visible keyboard focus, non-color status indicators (e.g., archived shown as text), and focus preservation after HTMX row swaps.

#### Scenario: Keyboard-only project creation
- **WHEN** a keyboard-only user opens the `New project` form, fills all fields, and submits
- **THEN** the project is created
- **AND** focus returns to a defined target (e.g., the new row or a confirmation message) announced via `aria-live`

### Requirement: Exhaustive cross-workspace denial for every projects handler

Every read and write handler in the `projects` family MUST return HTTP 404 with the shared not-found response body when invoked by a user whose active workspace does not own the referenced project or referenced client. This rule applies without exception to: list view, detail view, create, edit, archive, unarchive, and delete. The response body MUST NOT disclose the existence, name, or owning workspace of the target resource.

#### Scenario: List view is scoped to active workspace
- **GIVEN** Alice's active workspace is `W1` and projects exist in both `W1` and `W2`
- **WHEN** Alice requests the projects list
- **THEN** only projects with `workspace_id = W1` are returned

#### Scenario: Detail view for other-workspace project returns 404
- **GIVEN** project `P2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice requests `GET /projects/P2`
- **THEN** the system MUST respond with HTTP 404
- **AND** the response body MUST be the shared not-found template with no mention of `P2`

#### Scenario: Create referencing other-workspace client returns 404
- **GIVEN** client `C2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice submits a project-create form with `client_id = C2`
- **THEN** the system MUST respond with HTTP 404

#### Scenario: Edit across workspaces returns 404
- **GIVEN** project `P2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice POSTs an edit to `P2`
- **THEN** the system MUST respond with HTTP 404
- **AND** no row in `projects` is modified

#### Scenario: Archive across workspaces returns 404
- **GIVEN** project `P2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice POSTs an archive request for `P2`
- **THEN** the system MUST respond with HTTP 404

#### Scenario: Delete across workspaces returns 404
- **GIVEN** project `P2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice POSTs a delete request for `P2`
- **THEN** the system MUST respond with HTTP 404
- **AND** the row for `P2` still exists in the database

