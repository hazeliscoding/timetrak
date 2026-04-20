# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Status

TimeTrak is **post-MVP — Stage 2 (Stabilize)**. The MVP bootstrap change has been implemented and archived under `openspec/changes/archive/2026-04-17-bootstrap-timetrak-mvp/`. The application source exists; `openspec/specs/` is the accepted behavioral baseline.

Active work follows the Stage 2 roadmap: hardening workspace authorization, timer integrity, rate resolution, reporting correctness, UI/accessibility polish, reusable UI partials, and component library foundation. See `docs/timetrak_post_mvp_openspec_roadmap.md` for the full roadmap and ordering.

New work must follow the one-change-per-unit rule: use `/opsx:propose <change-name>` to create a focused change before implementing. Never umbrella changes like `phase-2` or `misc-cleanup`.

## Stack (Target)

- **Backend:** Go (modular monolith)
- **UI:** Go HTML templates rendered server-side + HTMX for partial updates; minimal custom JS
- **Database:** PostgreSQL (system of record)
- **Auth boundary:** Workspace (all authorization scopes to workspace)

Do not introduce SPA frameworks, client-side state libraries, or ORMs that fight the server-rendered + HTMX model without an explicit change proposal.

## Domain Model

Main hierarchy: **Workspace → Client → Project → Time Entry**. Rate resolution precedence: **project rate → client rate → workspace default**. Accepted MVP domains are `auth`, `workspace`, `clients`, `projects`, `tracking`, `rates`, `reporting` (see `openspec/config.yaml`).

## Data Conventions (binding)

- UUID primary keys
- `timestamptz` for all persisted timestamps
- Money stored as **integer minor units** — never floats
- Transactional tables normalized to 3NF unless there is a documented read-model reason

## Workflow: OpenSpec is the Source of Truth

This project uses OpenSpec (`/opsx:*` / `/openspec-*` skills). The canonical flow:

1. `openspec/specs/` = accepted behavior (source of truth once MVP lands)
2. `openspec/changes/<name>/` = active proposed deltas with proposal, specs, design, tasks
3. `docs/` = long-form narrative reference; **not** the behavioral baseline

Rules enforced by `openspec/config.yaml`:

- Proposals: MVP-first scope, explicit in/out of scope, call out assumptions and risks
- Specs: MUST/SHALL language, GIVEN/WHEN/THEN scenarios, organized by domain, include empty/loading/success/error/destructive states and accessibility requirements when UI is involved
- Design docs: include Mermaid diagrams when relevant; reflect Go + HTMX + PostgreSQL constraints; explain tradeoffs
- Tasks: small and implementation-ready; group by backend / database / templates / HTMX; include accessibility validation tasks for UI work

After MVP: **one change per meaningful unit of work** (e.g. `add-csv-export`), never umbrella changes like `phase-2` or `misc-cleanup`. Archive changes quickly once implemented.

### Useful slash commands

- `/opsx:explore` — think through a fuzzy idea before proposing
- `/opsx:propose <change-name>` — create a change with proposal/specs/design/tasks
- `/opsx:apply <change-name>` — implement from `tasks.md`
- `/opsx:archive <change-name>` — move completed change into baseline

## UI Direction (binding for any template work)

- Calm, trustworthy, tool-like; data-first, medium-density layouts; tables are first-class
- One restrained accent color, strong neutral system; prefer borders + spacing over heavy shadows
- Avoid generic AI-SaaS visual language: no blob art, no oversized hero sections, no random gradients, no vague productivity copy
- Use domain-specific copy (`Start timer`, `Billable this week`, `Client rate`, `Running entry`)
- Server-rendered flows over SPA patterns; HTMX for timers, inline edits, filtering, pagination, modals
- Preserve sensible focus behavior after HTMX swaps; prefer native controls before custom widgets

## Accessibility (binding)

Target **WCAG 2.2 AA**. Visible labels, visible keyboard focus, sufficient contrast, comfortable target sizes. Color must never be the sole means of conveying status. Include accessibility validation tasks in any UI-affecting change.

## Build / Test Commands

Standard development loop is driven by `Makefile`; a running PostgreSQL (via `make db-up`) and a populated `.env` (copy `.env.example`) are prerequisites.

- `make db-up` / `make db-down` — start/stop the local Postgres via `docker-compose.yml`
- `make run` — run the web server (`go run ./cmd/web`)
- `make build` — produce `bin/web` and `bin/migrate`
- `make migrate-up` / `make migrate-down` / `make migrate-redo` — apply / roll back / re-apply the most recent migration
- `make dev-seed` — seed a demo user, workspace, client, project, rate, and historical entries
- `make test` — `go test ./...`
- `make lint` / `make vet` — `go vet ./...`
- `make fmt` — `gofmt -w .`
- `make tidy` — `go mod tidy`

Required env vars (see `.env.example`): `DATABASE_URL`, `SESSION_SECRET` (≥32 bytes), `COOKIE_SECURE`, `APP_ENV`, `HTTP_ADDR`.

## Implementation choices landed in the bootstrap change

- **HTTP router**: stdlib `net/http` (Go 1.22+ method+path patterns like `GET /static/`). No third-party router.
- **Template engine**: stdlib `html/template`. Templates live under `web/templates/` and are loaded at startup by `internal/shared/templates` (layouts + partials auto-included with every page). Extra template funcs: `dict`, `seq`, `formatDate`, `formatTime`, `formatDuration`, `formatMinor`, `iso`, `add`, `sub`.
- **DB driver**: `github.com/jackc/pgx/v5` (direct use; no ORM). Transactions use `pool.InTx` (internally `pgx.BeginFunc`) against a `pgxpool.Pool`.
- **Password hashing**: Argon2id via `golang.org/x/crypto/argon2`. Parameters: `m=64 MiB`, `t=3`, `p=2`, 16-byte salt, 32-byte key. Minimum password length: 10.
- **Migrations**: plain SQL under `migrations/` with a tiny in-repo runner at `cmd/migrate/` (`NNNN_name.up.sql` / `NNNN_name.down.sql`, tracked in a `schema_migrations` table). No `golang-migrate`/`goose` dependency. `go run ./cmd/migrate seed` hashes the demo password fresh (no pinned hash string in git).
- **Pagination for entries list**: offset-based for MVP (deferred cursor upgrade). 25 rows per page.
- **CSRF**: signed double-submit cookie (`tt_csrf`), validated on POST/PUT/PATCH/DELETE via form field `csrf_token` or header `X-CSRF-Token`.
- **Sessions**: Postgres-backed (`sessions` table), cookie is `tt_session`, HMAC-signed session id, HttpOnly + SameSite=Lax; `Secure` controlled by `COOKIE_SECURE`.
- **Rate limiter** (auth): in-memory per-IP token bucket, burst 10, refill 1 token/minute. Swap for a Redis-backed bucket when scaling out.
- **HTMX peer-refresh events**: timer start/stop emit `HX-Trigger: timer-changed, entries-changed`; entry CRUD emits `entries-changed`; client/project CRUD emit their own `-changed` events; rate rule CRUD emits `rates-changed`. Dashboard summary refreshes via `hx-trigger="timer-changed from:body, entries-changed from:body"`.
- **Focus after HTMX swap**: `data-focus-after-swap` on the target element; `web/static/js/app.js` focuses it on `htmx:afterSwap`. Destructive deletes use `hx-confirm` (native `confirm()`) for MVP; `partials/confirm_dialog.html` is available as an accessible `<dialog>` option for future flows where focus trapping matters.
- **Rate resolution**: `rates.Service.Resolve(ctx, workspaceID, projectID, at)` is the single source of truth at write time. Tracking calls it at stop/save to persist a per-entry snapshot (`rate_rule_id`, `hourly_rate_minor`, `currency_code`) on `time_entries`. Reporting's read path is snapshot-only for closed entries and MUST NOT call `Resolve` — this keeps historical totals stable across retroactive `rate_rules` edits. `make check-rate-snapshots` is a deploy gate that fails when any closed billable entry is missing a snapshot; `make backfill-rate-snapshots` is the remediation.
- **Money**: integer minor units everywhere. `DurationBillable(seconds, hourlyRateMinor) = (seconds * rate) / 3600` — no floats.
- **Workspace authz**: every repository method takes `workspaceID` explicitly and includes it in the `WHERE` clause. Cross-workspace access returns HTTP 404, never 403.
- **Active-timer invariant**: partial unique index `ux_time_entries_one_active_per_user_workspace` (on `(workspace_id, user_id) WHERE ended_at IS NULL`). Concurrent starts fail with SQLSTATE 23505 → handler returns HTTP 409 via `ErrActiveTimerExists`.
- **Integration tests**: `internal/shared/testdb.Open(t)` opens the pool from `$DATABASE_URL` and truncates domain tables; tests are skipped gracefully when the env var is absent. Because all integration-test packages share one Postgres, `make test` runs `go test -p 1 ./...` so the truncate steps don't race.

## Repository layout

```
cmd/
  web/      HTTP server
  migrate/  Migration runner + dev-seed
  worker/   Placeholder for background jobs
internal/
  shared/   Cross-cutting: db, clock, money, http (middleware/HTMX), session, csrf, authz, templates, logging
  auth/ workspace/ clients/ projects/ tracking/ rates/ reporting/   (per-domain, land incrementally)
web/
  templates/{layouts,partials,...}   Server-rendered HTML + HTMX partials
  static/{css,js,vendor}             Tokens, app.js (theme toggle + HTMX focus helper), vendored HTMX
migrations/ 0001_*.up.sql / 0001_*.down.sql ...
```
