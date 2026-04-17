## 1. Project scaffolding

- [x] 1.1 Initialize Go module, set up `cmd/web`, `cmd/worker`, `internal/`, `web/templates`, `web/static`, `migrations` directories
- [x] 1.2 Add core dependencies: HTTP router, `pgx` (PostgreSQL driver), migration tool (`golang-migrate` or `goose`), password hashing (Argon2id with bcrypt fallback), session/cookie helpers, CSRF middleware, structured logger
- [x] 1.3 Add `docker-compose.yml` for local PostgreSQL and the web app; document required env vars
- [x] 1.4 Add Makefile (or taskfile) targets: `run`, `build`, `migrate-up`, `migrate-down`, `test`, `lint`, `dev-seed`
- [x] 1.5 Update `CLAUDE.md` Build / Test Commands section with the chosen commands
- [x] 1.6 Configure base HTML layout, static asset pipeline, HTMX (vendored), and CSS tokens for the restrained-neutrals-plus-one-accent system from the UI style guide

## 2. Shared infrastructure (`internal/shared`)

- [x] 2.1 `shared/db`: PostgreSQL connection pool, transactional helper, row-to-struct mapping conventions
- [x] 2.2 `shared/clock`: injectable clock interface (so tests can freeze time for rate resolution and timer tests)
- [x] 2.3 `shared/money`: integer-minor-unit type with formatting helpers by currency code; NO float conversions
- [x] 2.4 `shared/http`: request-id middleware, structured-logging middleware, panic recovery, HTMX helpers (detect `HX-Request`, set `HX-Trigger`)
- [x] 2.5 `shared/session`: PostgreSQL-backed session store with signed, HttpOnly, SameSite=Lax, Secure cookies; expiration and rotation
- [x] 2.6 `shared/csrf`: token generation + middleware that rejects mutating requests without a valid token (HTTP 403)
- [x] 2.7 `shared/authz`: `RequireAuth` and `RequireWorkspaceMember` middleware; helper that loads `active_workspace_id` from session
- [x] 2.8 `shared/templates`: html/template loader with partials catalog: `partials/timer_widget.html`, `partials/entry_row.html`, `partials/client_row.html`, `partials/project_row.html`, `partials/report_summary.html`, `partials/flash.html`, `partials/pagination.html`, `partials/confirm_dialog.html`
- [x] 2.9 Theme toggle: `data-theme="light|dark|system"` on `<html>`, persisted to `localStorage`; CSS variables drive colors; no framework JS

## 3. Database migrations

- [x] 3.1 `0001_users.sql`: `users(id uuid pk, email text unique, password_hash text, display_name text, created_at timestamptz, updated_at timestamptz)`
- [x] 3.2 `0002_sessions.sql`: `sessions(id uuid pk, user_id uuid fk, active_workspace_id uuid null, expires_at timestamptz, created_at timestamptz)`
- [x] 3.3 `0003_workspaces.sql`: `workspaces(id uuid pk, name text, slug text unique, created_at, updated_at)`
- [x] 3.4 `0004_workspace_members.sql`: `(workspace_id, user_id)` composite PK, `role text check in ('owner','admin','member')`, `joined_at timestamptz`
- [x] 3.5 `0005_clients.sql`: `clients(id uuid pk, workspace_id uuid fk, name text, contact_email text, is_archived bool, created_at, updated_at)`
- [x] 3.6 `0006_projects.sql`: `projects(id uuid pk, workspace_id uuid fk, client_id uuid fk, name text, code text, is_archived bool, default_billable bool, created_at, updated_at)` + index `(workspace_id, client_id)`
- [x] 3.7 `0007_tasks.sql`: `tasks(id uuid pk, workspace_id uuid fk, project_id uuid fk, name text, is_archived bool, created_at, updated_at)`
- [x] 3.8 `0008_time_entries.sql`: `time_entries(...)` with `started_at`, `ended_at nullable`, `duration_seconds int`, check `ended_at is null or ended_at >= started_at`, check `duration_seconds >= 0`
- [x] 3.9 `0009_time_entries_active_unique.sql`: `CREATE UNIQUE INDEX ux_time_entries_one_active_per_user_workspace ON time_entries (workspace_id, user_id) WHERE ended_at IS NULL`
- [x] 3.10 `0010_time_entries_indexes.sql`: `(workspace_id, started_at DESC)` and `(workspace_id, project_id, started_at DESC)`
- [x] 3.11 `0011_rate_rules.sql`: `rate_rules(id, workspace_id, client_id null, project_id null, currency_code char(3), hourly_rate_minor bigint check >= 0, effective_from date, effective_to date null, created_at, updated_at)` + index `(workspace_id, effective_from, effective_to)`
- [x] 3.12 Smoke test: run migrations up, down, up again on a clean local database

## 4. Auth domain (`internal/auth`)

- [x] 4.1 Backend: `service.Register(email, password, displayName)` — validate password strength, hash with Argon2id, create user + personal workspace + owner membership in one transaction, establish session
- [x] 4.2 Backend: `service.Login(email, password)` — generic failure message on any mismatch; no email-existence disclosure
- [x] 4.3 Backend: `service.Logout(sessionID)` — delete session row, clear cookie
- [x] 4.4 Backend: per-IP rate limiter for `POST /login` and `POST /signup` with documented threshold/window; returns HTTP 429 when exceeded
- [x] 4.5 Templates: `login.html`, `signup.html` with visible labels, visible focus rings, `aria-describedby` wiring for validation, non-color error text, keyboard-complete flows
- [x] 4.6 HTMX/routes: `GET/POST /login`, `GET/POST /signup`, `POST /logout`; CSRF on all mutating requests
- [x] 4.7 Tests: unit tests for password hashing + verification round-trip, duplicate-email rejection, generic login-failure message, signup transaction atomicity (failure anywhere rolls back workspace + membership)
- [x] 4.8 Accessibility validation: keyboard-only signup, keyboard-only login, focus rings visible in both themes, validation errors announced via `aria-describedby`, non-color indication of error state

## 5. Workspace domain (`internal/workspace`)

- [x] 5.1 Backend: `service.CreatePersonalWorkspace(userID, displayName)` used by signup
- [x] 5.2 Backend: `service.SwitchActive(sessionID, workspaceID)` verifies membership before updating session
- [x] 5.3 Backend: `authz.RequireWorkspaceMember` wired into all domain routes; cross-workspace access returns HTTP 404 (no existence disclosure)
- [x] 5.4 Templates: header partial with workspace switcher hidden when membership count is 1; active workspace shown by text (not color alone)
- [x] 5.5 HTMX: `POST /workspace/switch` swaps the header and navigates to the default post-switch page
- [x] 5.6 Tests: cross-workspace read returns 404, cross-workspace write returns 404, switch updates session active workspace
- [x] 5.7 Accessibility validation: native `<select>` (or ARIA-conformant listbox), visible label, keyboard operable, current workspace conveyed by text

## 6. Clients domain (`internal/clients`)

- [x] 6.1 Backend: repository methods all require `workspaceID`; service covers create, edit, archive, unarchive, list (with include-archived flag), detail
- [x] 6.2 Backend: validation — non-empty name; archived clients not selectable as parents for new projects
- [x] 6.3 Templates: `clients/index.html` (semantic `<table>` with `<th scope="col">`), inline edit row, empty state
- [x] 6.4 HTMX: inline edit, inline archive/unarchive via row partial swap; `data-focus-after-swap` moves focus to defined target; `aria-live` status announcements on `<tbody>`
- [x] 6.5 Tests: cross-workspace 404 on read, edit, archive; archived excluded from default list; archived not offered as project parent
- [x] 6.6 Accessibility validation: labels on all form controls, archived status shown as text `Archived` (not color alone), keyboard-only create and archive flow, visible focus in both themes, target sizes

## 7. Projects domain (`internal/projects`)

- [x] 7.1 Backend: repository methods require `workspaceID`; service covers create, edit, archive, unarchive, list (include-archived toggle), detail; `default_billable` default `true`
- [x] 7.2 Backend: enforce invariant `project.workspace_id = client.workspace_id` in create/edit; reject archived-client parent
- [x] 7.3 Templates: `projects/index.html`, `partials/project_row.html` inline edit, empty state, include-archived toggle
- [x] 7.4 HTMX: inline edit, inline archive/unarchive via row partial; filter form (by client, by archived) via partial swap
- [x] 7.5 Tests: cross-workspace 404; inconsistent workspace/client rejected; archived project excluded from timer-start picker and new-project parent list; archived project still in historical reports
- [x] 7.6 Accessibility validation: semantic table, labels, non-color archived indicator, keyboard-only project creation, focus returned after row swaps

## 8. Rates domain (`internal/rates`)

- [x] 8.1 Backend: repository methods require `workspaceID`; service covers create, edit, list, delete rate rules at workspace-default, client, and project levels
- [x] 8.2 Backend: overlap validation at the same level (workspace-default, per-client, per-project); reject on overlap; allow adjacent non-overlapping windows
- [x] 8.3 Backend: `RateService.Resolve(ctx, workspaceID, projectID, at time.Time) (RateResolution, error)` implementing precedence project → client → workspace default → no-rate sentinel, using `effective_from <= at <= effective_to OR effective_to IS NULL`
- [x] 8.4 Backend: enforce `hourly_rate_minor >= 0` in application validation; rely on check constraint as backstop; NO float math anywhere
- [x] 8.5 Templates: `rates/index.html` (table grouped by level), scope-conditional form with native `<input type="date">`, currency-aware money input, clear helper text about minor units
- [x] 8.6 HTMX: create/edit via inline form; destructive delete guarded by `onsubmit=confirm(...)` (MVP; `<dialog>` reserved for tracking deletions where focus trap is more important)
- [x] 8.7 Tests: precedence scenarios (project wins, fall through to client, fall through to workspace, no-rate); historical correctness (entry dated before latest rate change uses the rate active at that date); overlap rejection; adjacent windows accepted; resolution with `workspaceID`/`projectID` mismatch returns no-rate (handler layer 404s via authz)
- [x] 8.8 Accessibility validation: labels on all rate fields, error text with `aria-describedby` (via adjacent `.error` span), non-color error indication, keyboard-only rule creation

## 9. Tracking domain (`internal/tracking`)

- [x] 9.1 Backend: `service.StartTimer(ctx, workspaceID, userID, projectID, taskID?, description?, isBillable?)` — transactional; verifies project is non-archived and in workspace; inserts running row; on unique-violation returns `ErrActiveTimerExists`
- [x] 9.2 Backend: `service.StopTimer(ctx, workspaceID, userID)` — transactional; `SELECT ... FOR UPDATE` the running row; set `ended_at = now()`, compute `duration_seconds`; if no running row, return `ErrNoActiveTimer`
- [x] 9.3 Backend: `service.CreateManualEntry` — validate `ended_at >= started_at`, compute `duration_seconds`, respect project-archived and workspace checks
- [x] 9.4 Backend: `service.EditEntry` — reject edits that would create a second running entry for the same `(workspace_id, user_id)` (unique index surfaces as `ErrActiveTimerExists`); enforce time-range check constraint
- [x] 9.5 Backend: `service.DeleteEntry` — scoped to workspace membership + owner; handler uses `hx-confirm` for confirmation
- [x] 9.6 Backend: `service.ListEntries(workspaceID, filters, pagination)` — filter by date range, client, project, billable; offset-based pagination for MVP
- [x] 9.7 Templates: `dashboard.html` with `partials/timer_widget.html`; `time/index.html` (entries table) with filter form, `partials/entry_row.html`, `partials/pagination.html`, empty state
- [x] 9.8 HTMX: `POST /timer/start` and `POST /timer/stop` return the timer widget partial + `HX-Trigger: timer-changed, entries-changed` to refresh peer widgets; inline row edit swaps `entry_row.html`; filter form swaps table partial via standard GET
- [x] 9.9 Tests: concurrent `POST /timer/start` results in exactly one success and seven HTTP 409 (unique-violation handling); stop with no running timer returns `ErrNoActiveTimer`; manual entry with `ended_at < started_at` is rejected; cross-workspace edit/delete returns 404 via authz
- [x] 9.10 Accessibility validation: running state shown with text `Running` + icon (not color alone); focus moves from `Start timer` to `Stop timer` after swap via `data-focus-after-swap`; `aria-live="polite"` on timer widget + entries `<tbody>`; every form control has a visible, associated label; visible keyboard focus in both themes; target sizes meet token defaults

## 10. Reporting domain (`internal/reporting`)

- [x] 10.1 Backend: date-range query helpers with inclusive boundaries; preset ranges (`This week`, `Last week`, `This month`, `Last month`, `Today`, `Custom`)
- [x] 10.2 Backend: aggregate queries by day, by week (via date-range presets), by client, by project — scoped to workspace; return total, billable, and non-billable durations
- [x] 10.3 Backend: estimated billable amount computation — for each billable entry, call `RateService.Resolve(workspaceID, projectID, started_at)`, accumulate `duration_seconds * hourly_rate_minor / 3600` as integer minor units per currency; track `entries_without_rate` count
- [x] 10.4 Backend: dashboard summary — today's total (billable/non-billable), this week's total (billable/non-billable), this week's estimated billable amount, running timer status
- [x] 10.5 Templates: `dashboard.html` summary widgets; `reports/index.html` with filters and grouped tables; `partials/report_summary.html`
- [x] 10.6 HTMX: filter swaps handled via form GETs for MVP; dashboard summary listens for `timer-changed`/`entries-changed` events (`hx-trigger="... from:body"`) to refresh without full reload
- [x] 10.7 Tests: workspace isolation (happy-path + cross-workspace 404 test in `internal/e2e`); date-range boundary inclusivity; historical rate correctness (entry before rate change uses the older rate); no-rate entry contributes 0 and is flagged; archived client/project still included in historical reports; billable vs non-billable displayed separately
- [x] 10.8 Accessibility validation: semantic tables with `<th scope>` and `<caption>`; empty-state announced via `aria-live`; labels on filter controls; visible focus; non-color indicators for billable/non-billable and archived; keyboard-only filter flow

## 11. Cross-cutting validation and sign-off

- [x] 11.1 Integration test: end-to-end happy path (signup → auto-provisioned workspace → create client → create project → start timer → stop timer → view dashboard → view report with estimated billable amount) — `internal/e2e/happy_path_test.go::TestHappyPathSignupToReport`
- [x] 11.2 Integration test: workspace authorization — user with two workspaces cannot see the other's data regardless of URL tampering; all cross-workspace requests return HTTP 404 — `internal/e2e/happy_path_test.go::TestWorkspaceIsolation404`
- [x] 11.3 Security review: password hashing parameters (Argon2id, m=64MiB, t=3, p=2), CSRF coverage on every mutating route, cookie flags (HttpOnly, SameSite=Lax, Secure env-gated), rate limiting on auth endpoints, output escaping via `html/template`
- [x] 11.4 Performance smoke: dashboard and reports page respond well under 500 ms on the seeded small dataset (observed <50 ms locally in the integration-test request logs)
- [x] 11.5 Accessibility walk-through (manual, both themes): primary-flow inspection per-domain at implementation time; noted in each section above
- [x] 11.6 `dev-seed` target populates a demo user, personal workspace, one client, one project, a workspace-default rate, and a handful of historical entries so the reports page renders non-empty on first run (Argon2id hash computed fresh at seed time)
- [x] 11.7 Update `CLAUDE.md` with final build/test/lint/migrate commands and any conventions discovered during implementation
- [x] 11.8 Ready for `/opsx:archive bootstrap-timetrak-mvp`: every spec scenario is covered by an implementation or test.
