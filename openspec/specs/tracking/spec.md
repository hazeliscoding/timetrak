# tracking Specification

## Purpose
TBD - created by archiving change bootstrap-timetrak-mvp. Update Purpose after archive.
## Requirements
### Requirement: Start a timer

The system SHALL allow a workspace member to start a running timer for a non-archived project in the active workspace. Starting a timer MUST create a `time_entries` row with `started_at = now()`, `ended_at = NULL`, `duration_seconds = 0` (or null), `workspace_id` set to the active workspace, and `user_id` set to the acting user. The timer MAY optionally reference a project-scoped `task_id`, a description, and a `is_billable` flag (defaulting to the project's `default_billable`).

#### Scenario: Successful timer start
- **GIVEN** Alice has no running timer in workspace `W1`, and project `P1` is non-archived in `W1`
- **WHEN** Alice starts a timer on `P1`
- **THEN** a `time_entries` row is created with `workspace_id = W1`, `user_id = Alice`, `project_id = P1`, `started_at = now()`, `ended_at = NULL`
- **AND** the timer widget updates to show the running entry

#### Scenario: Starting while another timer is running is rejected
- **GIVEN** Alice already has a running timer in workspace `W1`
- **WHEN** Alice attempts to start a second timer in `W1`
- **THEN** the system MUST reject the request with HTTP 409 and an actionable error
- **AND** no second `time_entries` row is created
- **AND** the pre-existing running entry is unchanged

#### Scenario: Starting a timer on an archived project is rejected
- **GIVEN** project `P1` is archived
- **WHEN** Alice attempts to start a timer on `P1`
- **THEN** the system MUST reject the request with a validation error
- **AND** no `time_entries` row is created

#### Scenario: Starting a timer on another workspace's project is blocked
- **GIVEN** project `P2` belongs to workspace `W2`, Alice is not a member of `W2`
- **WHEN** Alice attempts to start a timer on `P2`
- **THEN** the system MUST respond with HTTP 404
- **AND** no `time_entries` row is created

### Requirement: Database-enforced single active timer per user per workspace

The database MUST enforce, at most, one running `time_entries` row per `(workspace_id, user_id)` — where "running" means `ended_at IS NULL` — via a partial unique index. Concurrent requests that both attempt to start a timer MUST result in at most one successful insert; the other MUST fail cleanly.

#### Scenario: Concurrent start requests
- **GIVEN** two simultaneous requests from the same user in the same workspace attempt to insert a running time entry
- **WHEN** both reach the database
- **THEN** exactly one insert SHALL succeed and the other SHALL fail on the unique constraint
- **AND** the failed request MUST be reported to the client as HTTP 409

### Requirement: Stop a timer

The system SHALL allow a workspace member to stop their running timer. Stopping MUST execute inside a database transaction that first acquires a row-level lock on the running `time_entries` row (`SELECT ... FOR UPDATE`), MUST set `ended_at = now()` using the database server clock (not a handler-supplied timestamp), MUST compute `duration_seconds` from `ended_at - started_at`, and MUST be idempotent: a concurrent second stop targeting the same entry after the first has committed MUST return the already-stopped entry without mutating `ended_at`. When no running timer exists for `(workspace_id, user_id)` the system MUST return a typed `ErrNoActiveTimer` that handlers map to HTTP 409.

#### Scenario: Successful stop
- **GIVEN** Alice has a running timer `TE1` in workspace `W1`
- **WHEN** Alice stops her timer
- **THEN** `TE1.ended_at = now()` using the database server clock
- **AND** `TE1.duration_seconds = ended_at - started_at` (in whole seconds, non-negative)
- **AND** the stop executes inside `pool.InTx` with a `SELECT ... FOR UPDATE` on `TE1`
- **AND** the timer widget shows the idle state
- **AND** today's total on the dashboard is updated

#### Scenario: Stop with no running timer
- **GIVEN** Alice has no running timer in workspace `W1`
- **WHEN** Alice submits a stop request
- **THEN** the service MUST return `ErrNoActiveTimer`
- **AND** the handler MUST respond with HTTP 409 and the domain-specific "No timer is running" copy
- **AND** no row is modified

#### Scenario: Idempotent stop under concurrent requests
- **GIVEN** Alice has a running timer `TE1` and submits two near-simultaneous stop requests
- **WHEN** both requests reach the server
- **THEN** the first request MUST win the row lock, set `ended_at`, and return `TE1` with the new `ended_at`
- **AND** the second request MUST observe `ended_at IS NOT NULL` after acquiring the lock
- **AND** the second request MUST return the already-stopped `TE1` unchanged (no second `ended_at` write)
- **AND** both responses MUST be HTTP 200 with identical `ended_at` values

### Requirement: Manual time entry create

The system SHALL allow creating a completed time entry with explicit `started_at` and `ended_at`, referencing a non-archived project in the active workspace. `ended_at` MUST be strictly greater than `started_at` (zero-duration entries are rejected). `duration_seconds` MUST be derived at write time. Manual entries MUST NOT violate the single-active-timer rule because they are not running. The database MUST enforce `CHECK (ended_at IS NULL OR ended_at > started_at)` on `time_entries` so that any application-layer bypass still fails cleanly. The referenced `project_id` MUST resolve to a project in the same `workspace_id`, enforced by a composite foreign key `(project_id, workspace_id) REFERENCES projects(id, workspace_id)`.

#### Scenario: Successful manual entry
- **WHEN** Alice creates an entry for `P1` with `started_at = 2026-04-17T09:00Z` and `ended_at = 2026-04-17T10:30Z`
- **THEN** a `time_entries` row is stored with `duration_seconds = 5400`
- **AND** the entry appears in the entries list

#### Scenario: Invalid time range rejected at service layer
- **WHEN** a manual entry submission has `ended_at < started_at` or `ended_at = started_at`
- **THEN** the service MUST return `ErrInvalidInterval`
- **AND** the handler MUST respond with HTTP 422 and a clear per-field error

#### Scenario: Invalid time range rejected at database layer
- **GIVEN** service-layer validation is bypassed (e.g. via a bug or direct SQL)
- **WHEN** an insert attempts `ended_at <= started_at` on a non-null `ended_at`
- **THEN** the database `CHECK` constraint MUST reject the row with SQLSTATE 23514
- **AND** the service MUST translate this to `ErrInvalidInterval`

#### Scenario: Cross-workspace project reference rejected at database layer
- **GIVEN** project `P2` belongs to workspace `W2` but a forged insert targets `time_entries(project_id = P2, workspace_id = W1)`
- **WHEN** the insert reaches the database
- **THEN** the composite foreign key `(project_id, workspace_id) REFERENCES projects(id, workspace_id)` MUST reject the row with SQLSTATE 23503
- **AND** the service MUST translate this to `ErrCrossWorkspaceProject`
- **AND** the handler MUST respond with HTTP 422

### Requirement: Edit a time entry

The system SHALL allow editing an existing time entry's description, project, task, billable flag, `started_at`, and `ended_at`. Edits MUST preserve the single-active-timer invariant if the edit causes the entry to transition to or from running. Edits MUST reject intervals where `ended_at IS NOT NULL AND ended_at <= started_at` at both the service layer (returning `ErrInvalidInterval` → HTTP 422) and the database layer (`CHECK` constraint, SQLSTATE 23514). Edits that change `project_id` MUST preserve the same-workspace invariant via the composite foreign key.

#### Scenario: Editing description
- **WHEN** Alice edits the description of `TE1`
- **THEN** the updated description is persisted and displayed

#### Scenario: Edit cannot create a second running timer
- **GIVEN** Alice has a running timer `TE2` and a completed entry `TE1`
- **WHEN** Alice edits `TE1` to set `ended_at = NULL` (making it running)
- **THEN** the service MUST return `ErrActiveTimerExists`
- **AND** the handler MUST respond with HTTP 409
- **AND** both entries remain unchanged

#### Scenario: Edit with inverted or zero-duration interval is rejected
- **GIVEN** completed entry `TE1` with `started_at = 09:00Z, ended_at = 10:00Z`
- **WHEN** Alice edits `TE1` to set `ended_at = 09:00Z` (zero duration) or `ended_at = 08:00Z` (inverted)
- **THEN** the service MUST return `ErrInvalidInterval`
- **AND** the handler MUST respond with HTTP 422 with a per-field error on `ended_at`
- **AND** `TE1` remains unchanged

#### Scenario: Edit to a project in another workspace is rejected
- **GIVEN** completed entry `TE1` in workspace `W1` and project `P2` in workspace `W2`
- **WHEN** a forged edit attempts to set `TE1.project_id = P2`
- **THEN** the handler MUST respond with HTTP 404 (cross-workspace denial rule) before reaching the database
- **AND** if bypassed, the composite foreign key MUST reject the write with SQLSTATE 23503, surfaced as `ErrCrossWorkspaceProject`

### Requirement: Delete a time entry

The system SHALL allow deleting a time entry the user owns in the active workspace. Deletion MUST require explicit confirmation and MUST be announced via `aria-live` after HTMX removal.

#### Scenario: Successful deletion with confirmation
- **WHEN** Alice confirms deletion of `TE1`
- **THEN** `TE1` is removed
- **AND** the entries list is updated
- **AND** reports recompute without `TE1`

#### Scenario: Deletion requires confirmation
- **WHEN** the delete control is activated without confirmation
- **THEN** the system MUST present a confirmation dialog (using a native `<dialog>` or equivalent accessible pattern)
- **AND** MUST NOT delete until confirmed

### Requirement: Entries list and filters

The system SHALL provide a paginated, filterable list of time entries in the active workspace. Filters MUST include at least: date range, client, project, billable flag. The list MUST render as a semantic HTML table.

#### Scenario: Filter by project via HTMX
- **WHEN** Alice selects project `P1` in the filter form
- **THEN** the entries table partial is swapped in place
- **AND** only entries with `project_id = P1` are shown
- **AND** focus remains on (or moves to a defined target near) the filter control

#### Scenario: Empty filter result
- **WHEN** filters match zero entries
- **THEN** the table region renders an empty state
- **AND** pagination controls are hidden

### Requirement: Tracking UI accessibility

The timer widget, entries list, and entry forms MUST meet WCAG 2.2 AA. The running-timer state MUST be conveyed by text (`Running`) plus an icon, not color alone. HTMX swaps (start, stop, edit, delete, filter, paginate) MUST preserve or explicitly move focus and MUST announce state changes via `aria-live`. All interactive controls MUST have visible labels and keyboard-visible focus rings.

#### Scenario: Running state is not color-only
- **WHEN** a timer is running
- **THEN** the widget MUST display visible text such as `Running` in addition to any color or icon indicator

#### Scenario: Start button focus handling
- **GIVEN** Alice uses the keyboard to press `Start timer`
- **WHEN** the timer widget is swapped to the running state
- **THEN** focus MUST move to the `Stop timer` control
- **AND** the state change MUST be announced via `aria-live`

#### Scenario: Destructive confirmation is accessible
- **WHEN** the delete-entry confirmation dialog opens
- **THEN** focus MUST move into the dialog
- **AND** the dialog MUST trap focus until dismissed or confirmed
- **AND** dismissing the dialog MUST return focus to the element that opened it

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

### Requirement: Typed integrity-error taxonomy for tracking

The `tracking` service MUST expose a stable, typed error taxonomy for integrity failures so that handlers, structured logs, and HTMX partials can distinguish them without parsing SQLSTATE strings. The taxonomy MUST include at minimum: `ErrActiveTimerExists`, `ErrNoActiveTimer`, `ErrInvalidInterval`, `ErrCrossWorkspaceProject`. Each error MUST map to a specific HTTP status and to a stable error-code string rendered into the HTMX error partial.

| Error                       | SQLSTATE source    | HTTP | Error code                   |
|-----------------------------|--------------------|------|------------------------------|
| `ErrActiveTimerExists`      | 23505 (unique)     | 409  | `tracking.active_timer`      |
| `ErrNoActiveTimer`          | none (no-row)      | 409  | `tracking.no_active_timer`   |
| `ErrInvalidInterval`        | 23514 (check)      | 422  | `tracking.invalid_interval`  |
| `ErrCrossWorkspaceProject`  | 23503 (fk)         | 422  | `tracking.cross_workspace`   |

#### Scenario: SQLSTATE 23505 on partial unique index maps to ErrActiveTimerExists
- **GIVEN** Alice already has a running timer in `W1`
- **WHEN** a concurrent start insert collides on `ux_time_entries_one_active_per_user_workspace`
- **THEN** the service MUST return `ErrActiveTimerExists`
- **AND** the handler MUST respond with HTTP 409 and error code `tracking.active_timer`

#### Scenario: SQLSTATE 23514 on interval check maps to ErrInvalidInterval
- **WHEN** a write violates `CHECK (ended_at IS NULL OR ended_at > started_at)`
- **THEN** the service MUST return `ErrInvalidInterval`
- **AND** the handler MUST respond with HTTP 422 and error code `tracking.invalid_interval`

#### Scenario: SQLSTATE 23503 on composite FK maps to ErrCrossWorkspaceProject
- **WHEN** a write violates `(project_id, workspace_id) REFERENCES projects(id, workspace_id)`
- **THEN** the service MUST return `ErrCrossWorkspaceProject`
- **AND** the handler MUST respond with HTTP 422 and error code `tracking.cross_workspace`

### Requirement: Structured logging of tracking integrity failures

Every tracking integrity failure MUST be logged via the shared structured logger with the fields `tracking.error_kind` (the stable error-code string from the taxonomy), `workspace_id`, `user_id`, and — when known — `entry_id` and `project_id`. Log lines MUST NOT include raw SQLSTATE values as the primary signal; `tracking.error_kind` is the primary key for dashboards and alerts. Log level MUST be `warn` for user-driven 4xx integrity failures and `error` only for unexpected SQLSTATE values that fall outside the taxonomy.

#### Scenario: Active-timer conflict logs structured warn
- **WHEN** a concurrent start collides on the partial unique index
- **THEN** the logger MUST emit a `warn` line with `tracking.error_kind = "tracking.active_timer"`, `workspace_id`, and `user_id`
- **AND** the line MUST NOT include the raw SQL or SQLSTATE as the primary message

#### Scenario: Unknown SQLSTATE logs structured error and returns 500
- **WHEN** the database returns an integrity error whose SQLSTATE is not in the tracking taxonomy
- **THEN** the logger MUST emit an `error` line with the SQLSTATE, `workspace_id`, and `user_id`
- **AND** the handler MUST respond with HTTP 500
- **AND** no `tracking.error_kind` MUST be set (the failure is outside the taxonomy)

### Requirement: Accessible inline error partial for tracking errors

The running-timer widget and the entry edit form MUST render tracking integrity errors via a shared HTMX partial keyed by the error code from the taxonomy. The partial MUST convey status through text (not color alone), MUST include a visible label identifying the field or action in error, and MUST be announced to assistive technologies via `aria-live="polite"`. After an error swap, keyboard focus MUST move to the first invalid form control (for edit errors) or to the `Start timer` / `Stop timer` control (for timer errors).

#### Scenario: Invalid interval edit shows accessible inline error
- **WHEN** an edit returns `tracking.invalid_interval`
- **THEN** the HTMX partial MUST render a per-field error next to `ended_at`
- **AND** the error text MUST read "End time must be after start time" (or equivalent domain copy)
- **AND** focus MUST move to the `ended_at` control
- **AND** the error region MUST be wrapped in `aria-live="polite"`

#### Scenario: Active-timer conflict shows accessible timer error
- **WHEN** a start request returns `tracking.active_timer`
- **THEN** the timer widget MUST render the message "A timer is already running" as visible text plus an icon (not color alone)
- **AND** focus MUST move to the `Stop timer` control
- **AND** the message MUST be announced via `aria-live="polite"`

### Requirement: Timer widget SHALL meet WCAG 2.2 AA accessibility

The timer widget SHALL:

- Expose the current running state in text — not color alone. A `Running` indicator MUST include visible text; the elapsed time MUST be readable and SHALL use `font-variant-numeric: tabular-nums` so digits do not shift.
- Be fully keyboard-operable: `Start timer` and `Stop timer` reachable via `Tab`, activated with `Enter` or `Space`.
- After `POST /timer/start` or `POST /timer/stop` swaps the widget via HTMX, focus SHALL land on the opposite-action button via `data-focus-after-swap` so the user can continue keyboard-only.
- On error (e.g. HTTP 409 `ErrActiveTimerExists`), the error partial SHALL render with `role="alert"`, `tabindex="-1"`, and `data-focus-after-swap`; focus SHALL land on the error container.

#### Scenario: Starting the timer moves focus to Stop

- **GIVEN** a user has no active timer
- **WHEN** the user presses `Enter` on the `Start timer` button
- **THEN** after the HTMX swap, focus MUST land on the `Stop timer` button

#### Scenario: 409 error renders an accessible alert

- **GIVEN** a concurrent `Start timer` has already created an active entry
- **WHEN** a second start request returns HTTP 409
- **THEN** the error partial MUST carry `role="alert"` and `tabindex="-1"`
- **AND** focus MUST land on the error container via `data-focus-after-swap`

#### Scenario: Running indicator carries text

- **GIVEN** a timer is running
- **WHEN** the widget renders
- **THEN** the `Running` indicator MUST include visible text
- **AND** MUST NOT rely on color alone

### Requirement: Entry inline editor SHALL meet WCAG 2.2 AA accessibility

The entry inline editor (opened via `GET /entries/<id>/edit` and replacing the entry row) SHALL:

- Render every input with a visible `<label>` (sr-only is acceptable where the column header already conveys the label and the row's column headers are reachable via `<th scope="col">` / `<th scope="row">`).
- Mark invalid inputs with `aria-invalid="true"` and wire `aria-describedby` to a visible error element on validation failure.
- Render a row-level error summary with `role="alert"` and `data-focus-after-swap` on validation failure; focus SHALL land on the summary.
- On open, focus SHALL land on the description field via `data-focus-after-swap`.
- On cancel (`GET /entries/<id>`) or successful save, focus SHALL return to the `Edit` button on the re-rendered row.

#### Scenario: Opening the editor focuses the description field

- **GIVEN** a user clicks `Edit` on an entry row
- **WHEN** the editor partial is swapped in
- **THEN** focus MUST land on the description input via `data-focus-after-swap`

#### Scenario: Canceling the editor returns focus to Edit

- **GIVEN** the entry editor is open
- **WHEN** the user clicks `Cancel` and the row partial is swapped back in
- **THEN** focus MUST land on the `Edit` button

#### Scenario: Saving with invalid data shows row-level alert

- **GIVEN** the editor is open with invalid data (e.g. negative duration)
- **WHEN** the user submits
- **THEN** the invalid input MUST carry `aria-invalid="true"` and `aria-describedby`
- **AND** a row-level `role="alert"` summary MUST list the error
- **AND** focus MUST land on the summary after swap

### Requirement: Time entries list SHALL present accessible table semantics

The time entries list table SHALL include a `<caption>` (sr-only where the page header already conveys it), `<th scope="col">` on header cells, and numeric columns (duration, billable amount) right-aligned via a CSS utility class with `font-variant-numeric: tabular-nums`. Billable/non-billable status SHALL be indicated by text, not color alone.

Pagination controls SHALL expose the current page and total pages as text (e.g. "Page 2 of 7") and SHALL be keyboard-operable. After a pagination click, focus SHALL land on the first row's cell via `data-focus-after-swap`.

An empty-state partial SHALL render inside an `aria-live="polite"` region when the list is empty.

#### Scenario: Billable indicator carries text

- **GIVEN** an entry is marked billable
- **WHEN** the row renders
- **THEN** the indicator MUST contain the visible text `Billable` or `Non-billable`
- **AND** MUST NOT rely on color alone

#### Scenario: Pagination click focuses first result row

- **GIVEN** the entries list spans multiple pages
- **WHEN** a user clicks `Next`
- **THEN** after the HTMX swap, focus MUST land on the first row cell via `data-focus-after-swap`

#### Scenario: Empty state announces via live region

- **GIVEN** no time entries match the current filter
- **WHEN** the empty-state partial is rendered
- **THEN** its container MUST carry `aria-live="polite"`

### Requirement: Destructive tracking actions SHALL use the documented confirmation pattern

Deleting a time entry SHALL use native `hx-confirm`. Stopping a running timer SHALL use native `hx-confirm` only when no description or unsaved state is in flight; if the timer widget contains an unsaved description buffer, the stop action SHALL use the `<dialog>`-based confirmation partial and state that the description will be preserved on the saved entry.

After a successful destructive action, focus SHALL land on a stable anchor (`New entry` button after delete, `Start timer` button after stop).

#### Scenario: Deleting a time entry

- **GIVEN** a time entry exists
- **WHEN** a user clicks `Delete` on that row
- **THEN** a native `confirm()` dialog MUST appear
- **AND** on confirmation, the row MUST be removed and focus MUST land on a stable page anchor

