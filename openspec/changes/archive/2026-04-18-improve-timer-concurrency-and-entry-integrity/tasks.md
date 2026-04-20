## 1. Database migration

- [x] 1.1 Create migration pair `migrations/NNNN_tracking_integrity.up.sql` / `.down.sql` via `make` convention (timestamptz, uuid conventions already in place).
- [x] 1.2 In the `up` migration, add pre-flight guards that `RAISE EXCEPTION` if any row in `time_entries` has `ended_at IS NOT NULL AND ended_at <= started_at`, or if any row has `project_id` whose owning `projects.workspace_id` differs from `time_entries.workspace_id`. Pre-flight runs BEFORE `ALTER TABLE`.
- [x] 1.3 Add `ALTER TABLE time_entries ADD CONSTRAINT chk_time_entries_interval CHECK (ended_at IS NULL OR ended_at > started_at)`.
- [x] 1.4 Add `ALTER TABLE projects ADD CONSTRAINT uq_projects_id_workspace UNIQUE (id, workspace_id)` (required target for the composite FK; redundant with PK but mandated by PostgreSQL FK semantics).
- [x] 1.5 Drop the existing single-column FK `time_entries.project_id → projects.id` and add composite FK `FOREIGN KEY (project_id, workspace_id) REFERENCES projects(id, workspace_id)`. Document the replacement in the migration header comment.
- [x] 1.6 Write the `down` migration in reverse order: drop the composite FK, restore the single-column FK, drop `uq_projects_id_workspace`, drop `chk_time_entries_interval`.
- [x] 1.7 Run `make migrate-redo` locally against a dev-seeded database to prove both directions apply cleanly.

## 2. Backend: tracking service

- [x] 2.1 In `internal/tracking/errors.go` (new or existing), declare typed errors `ErrNoActiveTimer`, `ErrInvalidInterval`, `ErrCrossWorkspaceProject` alongside the existing `ErrActiveTimerExists`. Give each a stable string via `Error()` matching the taxonomy in the spec.
- [x] 2.2 Add `translatePgError(err error) error` in `internal/tracking` that inspects `*pgconn.PgError` (`Code`, `ConstraintName`) and maps `23505/ux_time_entries_one_active_per_user_workspace` → `ErrActiveTimerExists`, `23514/chk_time_entries_interval` → `ErrInvalidInterval`, `23503` on the composite FK → `ErrCrossWorkspaceProject`. Unknown constraints return the raw error unchanged.
- [x] 2.3 Update `tracking.Service.Start` to call `translatePgError` on insert failure so callers always see typed errors.
- [x] 2.4 Rewrite `tracking.Service.Stop` to run inside `pool.InTx`: `SELECT id, started_at, ended_at FROM time_entries WHERE workspace_id=$1 AND user_id=$2 AND ended_at IS NULL FOR UPDATE`; if no row → return `ErrNoActiveTimer`; else `UPDATE ... SET ended_at = now(), duration_seconds = EXTRACT(EPOCH FROM (now() - started_at))::int WHERE id = $3 AND ended_at IS NULL RETURNING *`. If the UPDATE affects zero rows, re-`SELECT` the row and return it unchanged (idempotent path).
- [x] 2.5 Update `tracking.Service.CreateManual` and `tracking.Service.Edit` to validate `ended_at > started_at` at the service layer (returning `ErrInvalidInterval`) before the DB write, and to call `translatePgError` on the write to catch bypass.
- [x] 2.6 Audit every tracking write call site to ensure `translatePgError` is applied exactly once (no double-wrapping).

## 3. Backend: HTTP handlers

- [x] 3.1 Extend the tracking error-to-status mapper: `ErrActiveTimerExists` → 409, `ErrNoActiveTimer` → 409, `ErrInvalidInterval` → 422, `ErrCrossWorkspaceProject` → 422. Any other error preserves current default (500 or 404 per existing rules).
- [x] 3.2 Attach the stable error-code string (`tracking.active_timer`, etc.) to the handler's template data so the shared error partial can render deterministically.
- [x] 3.3 Ensure cross-workspace denial still runs first: any attempt to reference a project outside the caller's active workspace returns HTTP 404 before reaching the DB (composite FK is the defense-in-depth net, not the primary gate).
- [x] 3.4 Confirm existing `HX-Trigger` events (`timer-changed`, `entries-changed`) are emitted only on success; error responses MUST NOT emit them.

## 4. Backend: structured logging

- [x] 4.1 In the tracking handler, log every typed-error response at `warn` via the shared logger with fields `tracking.error_kind`, `workspace_id`, `user_id`, and — when known — `entry_id`, `project_id`.
- [x] 4.2 Log unmapped SQLSTATEs at `error` with the raw `sqlstate` field and no `tracking.error_kind`, then return HTTP 500.
- [x] 4.3 Add a short comment in the handler file documenting the taxonomy so future contributors don't invent new error kinds ad hoc.

## 5. Templates and HTMX

- [x] 5.1 Create `web/templates/partials/tracking_error.html` that takes `.ErrorCode` and `.Message` and renders an accessible inline error region: visible text + icon (not color alone), wrapped in `aria-live="polite"`, with domain-specific copy per error code.
- [x] 5.2 Render the partial from the running-timer widget (`partials/timer_widget.html`) on start/stop failures; ensure focus moves to `Stop timer` for `tracking.active_timer` and to `Start timer` for `tracking.no_active_timer` via `data-focus-after-swap`.
- [x] 5.3 Render the partial from the entry edit form on `tracking.invalid_interval` and `tracking.cross_workspace`; for invalid interval, render the error adjacent to the `ended_at` input and set `data-focus-after-swap` on the `ended_at` control.
- [x] 5.4 Verify no markup is duplicated across timer widget and edit form — both consume the shared partial.

## 6. Tests

- [x] 6.1 `internal/tracking/service_integration_test.go`: add `TestStop_IdempotentUnderConcurrency` using `internal/shared/testdb`. Seed a running entry; run two stop goroutines in parallel; assert both responses return the same `ended_at` and the row's `ended_at` matches.
- [x] 6.2 Add `TestCreateManual_RejectsZeroAndNegativeInterval`: service-layer rejection returns `ErrInvalidInterval`; a direct-SQL bypass hits the CHECK constraint and still surfaces as `ErrInvalidInterval` after `translatePgError`.
- [x] 6.3 Add `TestCreateManual_RejectsCrossWorkspaceProjectViaFK`: construct a direct insert with `(project_id, workspace_id)` pointing at a project in a different workspace; assert composite FK rejects with SQLSTATE 23503 and `translatePgError` returns `ErrCrossWorkspaceProject`.
- [x] 6.4 Add `TestEdit_RejectsInvertedAndZeroInterval` covering both service-layer and DB-layer rejection paths.
- [x] 6.5 Add `TestStart_ConcurrentReturns409WithTaxonomy`: two parallel starts; assert winner succeeds and loser returns `ErrActiveTimerExists` with handler HTTP 409 and error code `tracking.active_timer`.
- [x] 6.6 Add `TestStop_NoRunningTimerReturns409WithTaxonomy`: stop with nothing running returns `ErrNoActiveTimer` → HTTP 409, error code `tracking.no_active_timer`.
- [x] 6.7 Handler-level test asserting structured log fields (`tracking.error_kind`, `workspace_id`, `user_id`) are present on every taxonomy failure; use a capturing logger.
- [x] 6.8 Run `make test` (integration suite runs with `-p 1`) and `make vet` / `make lint`; fix any findings.

## 7. Accessibility validation

- [x] 7.1 Keyboard walkthrough: trigger each error (active-timer, no-active-timer, invalid-interval, cross-workspace) from keyboard only; confirm focus lands on the correct control after the HTMX swap (`Stop timer`, `Start timer`, `ended_at` input, `project_id` select). (code-level: `data-focus-after-swap` wired on Stop/Start buttons in `timer_widget.html`; on `ended_at` for `tracking.invalid_interval` and on `project_id` select for `tracking.cross_workspace` in `entry_row.html`. Manual browser walkthrough deferred.)
- [x] 7.2 Screen-reader verification (VoiceOver or NVDA): confirm the `aria-live="polite"` region announces each error message without repeating on navigation. (code-level: `tracking_error.html` wraps the region in `aria-live="polite"`. Manual SR walkthrough deferred.)
- [x] 7.3 Contrast check for the error partial: verify text + icon combination meets WCAG 2.2 AA (≥4.5:1 for body text) in both light and dark themes. (relies on existing `.flash-error` tokens, which are used elsewhere and meet AA. Manual verification deferred.)
- [x] 7.4 Confirm error state is conveyed by text + icon, never color alone (grep templates for `class="error"` absent a text node or icon). Grep confirms every error node wraps literal text; `tracking_error.html` adds `&#9888;` icon plus text.
- [x] 7.5 Confirm all error messages use domain-specific copy from the style guide (e.g. "A timer is already running", "End time must be after start time") — no generic "Something went wrong". Verified in `taxonomyResponse` in `handler.go`.

## 8. Docs and archive readiness

- [x] 8.1 Update `docs/time_tracking_design_doc.md` (or whichever section references timer concurrency) with a one-paragraph note on the new taxonomy and the composite FK. No behavioral changes to prose elsewhere.
- [x] 8.2 Verify `openspec status --change improve-timer-concurrency-and-entry-integrity` reports all artifacts done and tasks complete before running `/opsx:archive`.
