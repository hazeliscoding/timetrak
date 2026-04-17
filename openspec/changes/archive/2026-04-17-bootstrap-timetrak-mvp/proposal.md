## Why

TimeTrak is pre-code: it has design docs and a UI style guide but no running system and no accepted behavioral baseline in `openspec/specs/`. Contractors, freelancers, and small agencies need a focused, trustworthy tool that replaces spreadsheets for tracking billable time — not another generic SaaS app. This change bootstraps the one MVP foundation: the minimum behavior set required for a solo freelancer to sign up, configure a workspace, track time against a client's project, and see what they've billed this week. Everything downstream (invoicing, approvals, exports, team features) needs this baseline to exist first.

## What Changes

- Introduce a Go + HTMX + PostgreSQL modular monolith scaffold with the seven MVP domains.
- Add email/password authentication with hashed passwords, session cookies, and CSRF protection on mutating actions.
- Auto-provision a default personal workspace for new users and establish workspace as the authorization boundary (every domain query scopes to workspace membership).
- Add client CRUD and archival within a workspace.
- Add project CRUD and archival under a client, with `default_billable` flag and workspace_id invariant matching the parent client.
- Add time entry lifecycle: start timer, stop timer, manual entry create/edit/delete, billable flag, description, project association, optional task association.
- Enforce "one active timer per (workspace, user)" at the database level via a partial unique index.
- Add rate rules with effective date windows and a centralized `RateService` that resolves rates by precedence: project → client → workspace default → none.
- Add MVP reporting: totals by day/week, totals by client, totals by project, billable vs non-billable, estimated billable amount, date-range filter — all scoped to the active workspace.
- Add server-rendered HTML templates (login, dashboard with running-timer widget, clients list/detail, projects list/detail, time entries list, reports) with HTMX partials for timer start/stop, inline edits, filters, and pagination.
- Apply the UI style guide: restrained accent, data-first tables, domain-specific copy, WCAG 2.2 AA (visible labels, keyboard focus, non-color status cues, sufficient contrast), light/dark theme support.
- Add migrations for `users`, `workspaces`, `workspace_members`, `clients`, `projects`, `tasks`, `time_entries`, `rate_rules` with UUID PKs, `timestamptz` timestamps, money in integer minor units, and the recommended indexes/check constraints.

## Capabilities

### New Capabilities

- `auth`: User registration, email/password login, logout, session management, password hashing, CSRF protection, rate-limited auth endpoints.
- `workspace`: Workspace creation (including auto-provisioned personal workspace on signup), workspace membership with roles (owner/admin/member), active-workspace switching, workspace as the authorization boundary for all domain data.
- `clients`: Client CRUD, archival, list/detail views within a workspace, required `workspace_id` scoping.
- `projects`: Project CRUD under a client, archival, `default_billable` flag, `workspace_id`/`client_id` consistency invariant, optional project-scoped tasks.
- `tracking`: Start/stop running timer, manual time entry creation and edit, delete, billable flag, description, project (and optional task) association, duration calculation on stop, single-active-timer enforcement per (workspace, user).
- `rates`: Rate rule storage with effective date windows at workspace-default, client, and project levels; centralized rate resolution with precedence project → client → workspace default → none; non-negative `hourly_rate_minor`.
- `reporting`: Workspace-scoped summaries by day, week, client, and project; billable vs non-billable totals; estimated billable value using resolved rates; date-range and workspace filtering.

### Modified Capabilities

None. This is the initial bootstrap; no accepted baseline exists yet in `openspec/specs/`.

## Impact

- **New code**: Go module layout under `/cmd/web`, `/cmd/worker`, `/internal/{auth,workspace,clients,projects,tracking,rates,reporting,shared}`; HTML templates under `/web/templates`; static assets under `/web/static`; SQL migrations under `/migrations`.
- **Database**: Eight new tables, partial unique index for active timer integrity, reporting indexes, check constraints on durations/dates/rates.
- **Dependencies introduced**: Go HTTP router, HTML template engine, PostgreSQL driver, HTMX (vendored/CDN), password hashing (Argon2id or bcrypt), CSRF middleware, SQL migration tool.
- **Out of scope (deferred to later changes)**: invoice generation, PDF/CSV export, approval workflows, team invitations beyond personal workspace, multi-currency conversion, expense tracking, public API, OAuth/SSO, per-user rate overrides, audit log, recurring timer reminders, materialized views.
- **Assumptions**:
  1. MVP is solo-freelancer-first; multi-member workspaces are modeled in schema but team invitation UI is deferred.
  2. Tasks are project-scoped (not workspace-level) for MVP to keep the tracking flow simple.
  3. A time entry MUST belong to a project (no client-only tracking in MVP).
  4. Reports compute rate values live from `rate_rules` for MVP; invoice-time rate snapshots are deferred with invoicing.
  5. Deployment target is a single Go web container, a single PostgreSQL instance, and (optionally) a worker process; no background-queue infrastructure is required for MVP.
- **Risks**:
  1. **Timer race conditions** — double-submit or concurrent tabs could attempt to start two timers. Mitigated by the `(workspace_id, user_id) WHERE ended_at IS NULL` partial unique index and transactional start/stop handlers.
  2. **Rate resolution correctness** — effective-date windows are easy to misquery. Mitigated by centralizing in `RateService` with explicit unit tests covering all precedence paths and boundary dates.
  3. **Workspace authorization leaks** — modular monolith boundaries blur if handlers query other workspaces' data. Mitigated by requiring all repositories to accept `workspace_id` as a mandatory parameter and by authz-helper review.
  4. **Template sprawl** — server-rendered + HTMX can grow unwieldy without discipline. Mitigated by defining reusable partials for timer widget, entry row, client row, project row, and report summary card up front.
  5. **Accessibility drift** — easy to ship tables/forms that fail WCAG 2.2 AA. Mitigated by explicit accessibility validation tasks (keyboard nav, focus on HTMX swap, label association, contrast, non-color status indicators) on every UI task group.
- **Follow-up phases** (post-MVP, each its own small change): `add-csv-export`, `add-invoice-draft-generation`, `support-project-archiving-ux`, `add-team-workspace-invitations`, `add-timesheet-approval-flow`, `improve-report-filtering`, `add-running-timer-reminders`, `refactor-rate-resolution-snapshot-for-invoicing`.
