## ADDED Requirements

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

The system SHALL allow a workspace member to stop their running timer. Stopping MUST set `ended_at = now()`, MUST compute `duration_seconds` from `ended_at - started_at`, and MUST run within a database transaction that locks the running row.

#### Scenario: Successful stop
- **GIVEN** Alice has a running timer `TE1` in workspace `W1`
- **WHEN** Alice stops her timer
- **THEN** `TE1.ended_at = now()`
- **AND** `TE1.duration_seconds = ended_at - started_at` (in whole seconds, non-negative)
- **AND** the timer widget shows the idle state
- **AND** today's total on the dashboard is updated

#### Scenario: Stop with no running timer
- **GIVEN** Alice has no running timer
- **WHEN** Alice submits a stop request
- **THEN** the system MUST respond with HTTP 409 and a clear message
- **AND** no row is modified

### Requirement: Manual time entry create

The system SHALL allow creating a completed time entry with explicit `started_at` and `ended_at`, referencing a non-archived project in the active workspace. `ended_at` MUST be greater than or equal to `started_at`, and `duration_seconds` MUST be derived at write time. Manual entries MUST NOT violate the single-active-timer rule because they are not running.

#### Scenario: Successful manual entry
- **WHEN** Alice creates an entry for `P1` with `started_at = 2026-04-17T09:00Z` and `ended_at = 2026-04-17T10:30Z`
- **THEN** a `time_entries` row is stored with `duration_seconds = 5400`
- **AND** the entry appears in the entries list

#### Scenario: Invalid time range rejected
- **WHEN** a manual entry submission has `ended_at < started_at`
- **THEN** the system MUST reject with a validation error
- **AND** the database check constraint MUST reject the row if the validation is somehow bypassed

### Requirement: Edit a time entry

The system SHALL allow editing an existing time entry's description, project, task, billable flag, started_at, and ended_at. Edits MUST preserve the single-active-timer invariant if the edit causes the entry to transition to/from `running`.

#### Scenario: Editing description
- **WHEN** Alice edits the description of `TE1`
- **THEN** the updated description is persisted and displayed

#### Scenario: Edit cannot create a second running timer
- **GIVEN** Alice has a running timer `TE2` and a completed entry `TE1`
- **WHEN** Alice edits `TE1` to set `ended_at = NULL` (making it running)
- **THEN** the system MUST reject the edit with HTTP 409
- **AND** both entries remain unchanged

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
