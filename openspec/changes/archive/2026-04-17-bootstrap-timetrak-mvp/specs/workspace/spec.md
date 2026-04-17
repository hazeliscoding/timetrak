## ADDED Requirements

### Requirement: Default personal workspace on signup

The system SHALL automatically create a default personal workspace for each new user during registration and add that user as the workspace `owner` in the same transaction as user creation.

#### Scenario: First workspace created on signup
- **GIVEN** a user completes signup
- **THEN** exactly one `workspaces` row is created for that user
- **AND** a `workspace_members` row exists with role `owner`
- **AND** the session's active workspace is set to that workspace

### Requirement: Workspace is the authorization boundary

All domain data (clients, projects, tasks, time entries, rate rules, reports) MUST be scoped to a workspace. Every read and write operation MUST verify that the current user is a member of the workspace that owns the data. Operations that attempt to access data in a workspace the user is not a member of MUST return HTTP 404.

#### Scenario: Member reads own workspace data
- **GIVEN** user Alice is a member of workspace `W1`
- **WHEN** Alice requests clients for `W1`
- **THEN** the system returns clients whose `workspace_id = W1`

#### Scenario: Non-member is blocked from another workspace's data
- **GIVEN** user Bob is NOT a member of workspace `W1`
- **WHEN** Bob requests a client, project, or time entry that belongs to `W1`
- **THEN** the system MUST respond with HTTP 404
- **AND** the response MUST NOT disclose whether the resource exists

#### Scenario: Cross-workspace write is rejected
- **GIVEN** user Alice's active workspace is `W1`
- **WHEN** a mutating request attempts to modify a resource in `W2` (where Alice is not a member)
- **THEN** the request MUST be rejected with HTTP 404
- **AND** no database mutation occurs

### Requirement: Workspace membership roles

The system SHALL support at least three membership roles: `owner`, `admin`, `member`. The creator of a workspace MUST be its `owner`. For MVP, all three roles grant full read/write access to workspace data; role distinctions are reserved for later changes (team invitations, approvals).

#### Scenario: Owner role assigned on creation
- **WHEN** a workspace is created
- **THEN** the creating user's `workspace_members.role` is `owner`

### Requirement: Active workspace switching

A user who is a member of more than one workspace SHALL be able to switch the active workspace. The switch MUST persist for the duration of the session. When the user has only one workspace, the switcher control MAY be hidden to reduce visual noise, but the session MUST still record an active workspace.

#### Scenario: User with multiple memberships switches workspace
- **GIVEN** Alice is a member of `W1` and `W2`, with `W1` active
- **WHEN** Alice selects `W2` from the workspace switcher
- **THEN** the session's `active_workspace_id` is updated to `W2`
- **AND** subsequent pages show data scoped to `W2`

#### Scenario: Solo-workspace user has workspace set
- **GIVEN** Alice is a member of exactly one workspace `W1`
- **WHEN** Alice logs in
- **THEN** the session's `active_workspace_id` equals `W1`
- **AND** the workspace switcher control MAY be hidden

### Requirement: Workspace switcher UI accessibility

When the workspace switcher is rendered, it MUST be a native `<select>` or an equivalent ARIA-conformant listbox, MUST have a visible label, MUST be keyboard operable, and MUST convey the currently active workspace by text (not color alone).

#### Scenario: Keyboard switch
- **GIVEN** Alice has the switcher focused
- **WHEN** she opens it with the keyboard and selects a workspace
- **THEN** the selection is applied without requiring a pointer
- **AND** the active workspace text is announced by assistive technology
