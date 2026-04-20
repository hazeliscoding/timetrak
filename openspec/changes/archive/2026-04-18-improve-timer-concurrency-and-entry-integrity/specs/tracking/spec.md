## MODIFIED Requirements

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

## ADDED Requirements

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
