## Why

The MVP bootstrap established a partial unique index that prevents two concurrent running timers per `(workspace_id, user_id)`, surfacing conflicts as HTTP 409. That invariant is necessary but not sufficient: stops can race with stops, manual entry edits can produce zero- or negative-duration intervals, a time entry's `project_id` can theoretically point at a project from another workspace, and operators have no structured signal to distinguish "active timer exists" from "interval constraint violated". Stage 2 is the stabilize stage, and tracking correctness is the highest-leverage area to harden before reporting and invoicing build on top of it.

## What Changes

- Add a database `CHECK` constraint ensuring `ended_at IS NULL OR ended_at > started_at` on `time_entries` (strict greater-than — zero-duration entries are rejected).
- Add a database `CHECK` or FK-level guarantee that a time entry's `project_id` resolves to a project in the same `workspace_id` (via a composite FK `(project_id, workspace_id)` referencing a matching unique key on `projects`).
- Make timer **stop** idempotent at the service layer: stopping an already-stopped entry returns the existing entry without mutating `ended_at`; stopping when no active entry exists returns a typed `ErrNoActiveTimer` that the handler maps to HTTP 409 (not 500).
- Resolve stop races deterministically: the stop path takes a row-level lock on the active entry (`SELECT ... FOR UPDATE` inside `pool.InTx`) before writing `ended_at`, and uses server-side `now()` (monotonic on the DB) rather than trusting handler-supplied timestamps.
- Classify integrity failures distinctly in the tracking service: `ErrActiveTimerExists` (SQLSTATE 23505 on partial unique index), `ErrInvalidInterval` (CHECK violation on `ended_at > started_at`), `ErrCrossWorkspaceProject` (composite FK violation), each mapped to specific 4xx responses with stable error codes for HTMX partials.
- Add structured log fields (`tracking.error_kind`, `workspace_id`, `user_id`, `entry_id`) for every integrity failure so operators can triage without reading SQLSTATE.
- Surface clear, accessible inline error copy on the running-timer partial and the entry edit form for each failure kind; never rely on color alone.

## Capabilities

### New Capabilities

None. This change hardens existing accepted behavior.

### Modified Capabilities

- `tracking`: tightens requirements around active-timer concurrency, idempotent stop, interval validity (start strictly before end), same-workspace project reference, and the taxonomy of integrity errors returned to handlers and surfaced in the UI.

## Impact

- **Database**: new migration pair adding (a) `CHECK (ended_at IS NULL OR ended_at > started_at)` on `time_entries`, (b) a composite unique key on `projects(id, workspace_id)` if not already present, and (c) a composite FK from `time_entries(project_id, workspace_id)` to `projects(id, workspace_id)`. Backfill is a no-op under MVP invariants but the migration runs a pre-flight `SELECT` to fail loudly if any row would violate the new constraint.
- **Backend (`internal/tracking`)**: stop handler/service refactored to use `pool.InTx` + `SELECT ... FOR UPDATE`; new typed errors `ErrNoActiveTimer`, `ErrInvalidInterval`, `ErrCrossWorkspaceProject`; edit path validates interval before writing and maps PG error codes to the typed errors.
- **HTTP layer**: error-to-status mapping extended (`409` for active-timer and no-active-timer races, `422` for invalid interval and cross-workspace project); HTMX partials get a small `partials/timer_error.html` fragment keyed by error code.
- **Reporting**: no behavioral change. Reporting continues to read via `rates.Service.Resolve`; this change only strengthens the rows it reads.
- **Observability**: structured logs via the shared logger gain a `tracking.error_kind` field; no new dependencies.
- **Out of scope** (explicit, each would be a separate focused change):
  - Offline / multi-device timer reconciliation.
  - Per-entry audit log or change history.
  - Timer pause/resume semantics.
  - Clock-skew protection across clients (server-side `now()` is the MVP-appropriate fix).
  - Advisory-lock-based throttling of start attempts (the partial unique index plus typed error remains the MVP choice; revisit if contention is observed).
- **Assumptions**:
  - No production data currently violates `ended_at > started_at` or cross-workspace project references (pre-flight check will confirm).
  - Server wall-clock drift is within acceptable bounds; NTP is assumed on the DB host.
  - HTMX peer-refresh events (`timer-changed`, `entries-changed`) remain the sole coordination primitive for UI refresh.
- **Risks**:
  - A latent bad row would block the migration. Mitigation: pre-flight query in the `up` migration that aborts with a clear message before the constraint is added.
  - Row-level locking on stop marginally increases contention under pathological retry loops. Mitigation: the partial unique index already bounds active rows per user to one, so the lock is effectively single-row.
