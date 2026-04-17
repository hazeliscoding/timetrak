## ADDED Requirements

### Requirement: Exhaustive cross-workspace denial for every tracking handler

Every read and write handler in the `tracking` family MUST return HTTP 404 with the shared not-found response body when invoked by a user whose active workspace does not own the referenced time entry or referenced project. This rule applies without exception to: timer start, timer stop, active-timer read, entry list, entry detail, entry edit, and entry delete. The response body MUST NOT disclose the existence, project, client, or owning workspace of the target resource.

#### Scenario: Timer start against other-workspace project returns 404
- **GIVEN** project `P2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice POSTs a timer-start request targeting `P2`
- **THEN** the system MUST respond with HTTP 404
- **AND** no `time_entries` row is inserted
- **AND** no `HX-Trigger` header is emitted

#### Scenario: Timer stop against other-workspace entry returns 404
- **GIVEN** a running entry `E2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice POSTs a timer-stop request for `E2`
- **THEN** the system MUST respond with HTTP 404
- **AND** `E2.ended_at` remains NULL

#### Scenario: Entry list is scoped to active workspace
- **GIVEN** Alice's active workspace is `W1` and time entries exist in both `W1` and `W2`
- **WHEN** Alice requests the entries list
- **THEN** only entries with `workspace_id = W1` are returned
- **AND** no entries from `W2` appear in the rendered table

#### Scenario: Entry edit across workspaces returns 404
- **GIVEN** entry `E2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice POSTs an edit to `E2`
- **THEN** the system MUST respond with HTTP 404
- **AND** no row in `time_entries` is modified

#### Scenario: Entry delete across workspaces returns 404
- **GIVEN** entry `E2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice POSTs a delete request for `E2`
- **THEN** the system MUST respond with HTTP 404
- **AND** the row for `E2` still exists in the database

### Requirement: Active-timer invariant MUST be evaluated strictly within the caller's workspace

The active-timer uniqueness check (enforced by the partial unique index `ux_time_entries_one_active_per_user_workspace`) MUST be evaluated using the caller's verified active `workspace_id` from the typed request context, never a `workspace_id` drawn from request input. A user with a running timer in `W1` who switches active workspace to `W2` MUST be able to start a timer in `W2` without being blocked by the `W1` timer, because the uniqueness constraint is scoped per `(workspace_id, user_id)`.

#### Scenario: Running timer in W1 does not block start in W2
- **GIVEN** Alice is a member of both `W1` and `W2`
- **AND** Alice has a running `time_entries` row in `W1`
- **WHEN** Alice switches active workspace to `W2` and starts a timer in `W2`
- **THEN** the timer start MUST succeed with HTTP 200 (or 201)
- **AND** Alice now has two running entries: one in `W1`, one in `W2`

#### Scenario: Concurrent start in same workspace returns 409
- **GIVEN** Alice has a running entry in `W1`
- **WHEN** Alice (or a concurrent tab) POSTs another timer-start in `W1`
- **THEN** the system MUST respond with HTTP 409
- **AND** the error message MUST be the domain-specific "A timer is already running" copy
