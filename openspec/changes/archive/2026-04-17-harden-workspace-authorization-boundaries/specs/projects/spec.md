## MODIFIED Requirements

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

## ADDED Requirements

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
