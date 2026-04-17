## ADDED Requirements

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
