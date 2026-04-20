# workspace Specification

## Purpose
Workspace is the sole authorization boundary in TimeTrak. Every domain
entity (clients, projects, tasks, time entries, rate rules, reporting
aggregates) is owned by exactly one workspace, and every read or write
operation MUST verify the caller is a member of that workspace. Cross-
workspace access returns HTTP 404 with no information disclosure. The
authorization contract is upheld by typed request context, repository-
level WHERE-clause discipline (audited by `go test`), exhaustive per-
handler cross-workspace denial tests, and database-level invariants where
service code alone is insufficient.
## Requirements
### Requirement: Default personal workspace on signup

The system SHALL automatically create a default personal workspace for each new user during registration and add that user as the workspace `owner` in the same transaction as user creation.

#### Scenario: First workspace created on signup
- **GIVEN** a user completes signup
- **THEN** exactly one `workspaces` row is created for that user
- **AND** a `workspace_members` row exists with role `owner`
- **AND** the session's active workspace is set to that workspace

### Requirement: Workspace is the authorization boundary

All domain data (clients, projects, tasks, time entries, rate rules, reports) MUST be scoped to a workspace. Every read and write operation MUST verify that the current user is a member of the workspace that owns the data. Operations that attempt to access data in a workspace the user is not a member of MUST return HTTP 404 and MUST NOT disclose whether the resource exists, what workspace it belongs to, or any identifiers beyond those present in the request URL. Cross-workspace denials MUST NOT return HTTP 403, 400, or any status other than 404; the response body MUST be rendered from a single shared not-found template that does not name the requested resource type.

#### Scenario: Member reads own workspace data
- **GIVEN** user Alice is a member of workspace `W1`
- **WHEN** Alice requests clients for `W1`
- **THEN** the system returns clients whose `workspace_id = W1`

#### Scenario: Non-member is blocked from another workspace's data
- **GIVEN** user Bob is NOT a member of workspace `W1`
- **WHEN** Bob requests a client, project, or time entry that belongs to `W1`
- **THEN** the system MUST respond with HTTP 404
- **AND** the response body MUST be byte-identical to the response for a resource that does not exist
- **AND** the response MUST NOT include the requested resource name, type, or owning workspace identifier

#### Scenario: Cross-workspace write is rejected
- **GIVEN** user Alice's active workspace is `W1`
- **WHEN** a mutating request attempts to modify a resource in `W2` (where Alice is not a member)
- **THEN** the request MUST be rejected with HTTP 404
- **AND** no database mutation occurs

#### Scenario: Cross-workspace status parity across handler families
- **GIVEN** the handler families `clients`, `projects`, `tracking`, `rates`, and `reporting`
- **WHEN** any mutating or reading handler in any of these families receives a request referencing a resource in a workspace the caller is not a member of
- **THEN** the response status MUST be 404 regardless of family or verb
- **AND** the outcome MUST be identical to the outcome for a resource whose identifier does not exist at all

### Requirement: Handlers MUST receive the active workspace via a typed request context

Every domain handler (clients, projects, tracking, rates, reporting) MUST obtain the authenticated user identifier, the active workspace identifier, and the membership role from a single typed request-context value populated by authorization middleware. Handlers MUST NOT read `workspace_id` from form input, query parameters, URL path parameters, or request bodies for the purpose of authorization. The middleware MUST resolve the active workspace from the session, verify membership against the database, and either inject the typed context or respond with HTTP 404.

#### Scenario: Handler reads workspace from context only
- **GIVEN** a handler in any domain family is invoked
- **WHEN** the handler determines which workspace to scope its database queries to
- **THEN** it MUST read the workspace identifier from the typed request-context value
- **AND** it MUST NOT trust any `workspace_id` value arriving in form, query, path, or body input

#### Scenario: Missing or invalid workspace membership short-circuits at middleware
- **GIVEN** an authenticated user whose session active workspace is `W1`
- **WHEN** the middleware cannot confirm a `workspace_members` row for (user, `W1`)
- **THEN** the request MUST NOT reach the handler
- **AND** the response MUST be HTTP 404 using the shared not-found template

#### Scenario: Untrusted form input does not influence authorization
- **GIVEN** user Alice is a member only of `W1`
- **WHEN** Alice submits a form whose body contains `workspace_id=W2`
- **THEN** the handler MUST ignore that field for authorization
- **AND** the request MUST be scoped to Alice's active workspace `W1`

### Requirement: Repositories MUST constrain every query by workspace_id

Every public repository method accepting a `workspaceID` parameter MUST include `workspace_id = $N` (or an equivalent constrained predicate joined to the owning workspace) in the `WHERE` clause of every SQL statement it issues. A test harness SHALL inspect the repository source and fail the build if a method accepts `workspaceID` but issues a query that does not reference `workspace_id`. Exceptions MUST be declared inline with an `//authz:ok: <reason>` comment adjacent to the query and MUST be reviewable.

#### Scenario: Missing workspace predicate is detected
- **GIVEN** a new repository method `FindFoo(ctx, workspaceID, id)` is added to `internal/foo/repo.go`
- **WHEN** its SQL body is `SELECT * FROM foos WHERE id = $1`
- **THEN** `make test` MUST fail with a message identifying the file and method
- **AND** the failure MUST remain until the predicate is added or an `//authz:ok` exception with a reason is recorded

#### Scenario: Compliant repository method passes
- **GIVEN** a repository method `FindFoo(ctx, workspaceID, id)` whose SQL is `SELECT * FROM foos WHERE id = $1 AND workspace_id = $2`
- **WHEN** the audit runs
- **THEN** it MUST pass with no findings for that method

### Requirement: Cross-workspace denial coverage MUST be exhaustive per handler family

The test suite MUST include, for every registered HTTP route in the `clients`, `projects`, `tracking`, `rates`, and `reporting` handler families, at least one integration test that invokes the route as a user whose active workspace differs from the target resource's workspace and asserts HTTP 404 with a shared not-found response body. A test MUST fail if a new route is registered without a corresponding authz row.

#### Scenario: New handler without authz coverage fails build
- **WHEN** a developer registers a new route under any covered handler family
- **AND** no cross-workspace denial test is added alongside it
- **THEN** the route-coverage test MUST fail
- **AND** the failure MUST name the uncovered route

#### Scenario: All registered routes pass cross-workspace denial
- **GIVEN** every route in the covered handler families has an authz test row
- **WHEN** the integration suite runs
- **THEN** every row MUST receive HTTP 404 when invoked across workspaces
- **AND** every response body MUST match the shared not-found template exactly

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

### Requirement: Workspace reporting timezone

Each workspace SHALL carry a `reporting_timezone` attribute: a non-empty IANA timezone name (e.g., `UTC`, `America/New_York`, `Europe/Berlin`). The attribute SHALL default to `'UTC'` for new workspaces and for all existing workspaces at migration time. The reporting domain SHALL consume this attribute to bucket entries by local calendar day; no other behavior in the workspace domain itself depends on it.

#### Scenario: Default value on new workspace

- **WHEN** a workspace is created without specifying a timezone
- **THEN** its `reporting_timezone` SHALL equal `'UTC'`

#### Scenario: Backfill preserves existing behavior

- **GIVEN** workspaces existed before this change
- **WHEN** the migration runs
- **THEN** every pre-existing workspace SHALL have `reporting_timezone = 'UTC'`
- **AND** report output for those workspaces SHALL be unchanged relative to the pre-change UTC baseline

### Requirement: Reporting timezone settings

The system SHALL provide a workspace-scoped settings control that lets an authorized user change the workspace `reporting_timezone` to any value in Postgres's `pg_timezone_names`. Submitting an unrecognized or empty value SHALL be rejected with a form-level error and SHALL NOT persist. A change SHALL take effect on the next report request with no data migration.

#### Scenario: Valid timezone persists

- **GIVEN** Alice is on the workspace settings page
- **WHEN** she selects `America/New_York` and submits
- **THEN** the workspace's `reporting_timezone` SHALL be updated
- **AND** the next `/reports` request SHALL bucket entries by `America/New_York` local date

#### Scenario: Invalid timezone is rejected

- **WHEN** the settings endpoint receives a tz name not present in `pg_timezone_names`
- **THEN** the system SHALL respond with a validation error
- **AND** the stored `reporting_timezone` SHALL be unchanged

#### Scenario: Setting change is workspace-scoped

- **GIVEN** Alice is a member of workspaces `W1` and `W2`
- **WHEN** she changes `W1`'s `reporting_timezone`
- **THEN** `W2`'s `reporting_timezone` SHALL be unaffected
- **AND** any attempt to mutate `W2`'s timezone without membership SHALL return HTTP 404

### Requirement: Workspace switcher UI accessibility

When the workspace switcher is rendered, it MUST be a native `<select>` or an equivalent ARIA-conformant listbox, MUST have a visible label, MUST be keyboard operable, and MUST convey the currently active workspace by text (not color alone).

#### Scenario: Keyboard switch
- **GIVEN** Alice has the switcher focused
- **WHEN** she opens it with the keyboard and selects a workspace
- **THEN** the selection is applied without requiring a pointer
- **AND** the active workspace text is announced by assistive technology

