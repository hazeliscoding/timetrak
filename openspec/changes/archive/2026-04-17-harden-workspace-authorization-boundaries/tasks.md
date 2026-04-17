## 1. Backend — Typed workspace context and middleware

- [x] 1.1 Create `internal/shared/authz/workspace_context.go` with `WorkspaceContext{UserID, WorkspaceID, Role}`, an unexported context key, `FromContext`, and `MustFromContext` accessors
- [x] 1.2 Add `RequireWorkspace` middleware in `internal/shared/http/middleware` that resolves active workspace from the session, verifies `workspace_members` for (user, workspace), injects `WorkspaceContext`, or renders the shared 404 on failure
- [x] 1.3 Wire `RequireWorkspace` into the route registration for every authenticated domain route in `cmd/web`
- [x] 1.4 Add unit tests for `RequireWorkspace` covering: no session, valid session but no membership, valid session with membership, tampered session values
- [x] 1.5 Add a forbid-list lint test (simple `go test` walking `internal/{clients,projects,tracking,rates,reporting}/handlers*.go`) that fails if a handler body references `r.FormValue("workspace_id")`, `r.URL.Query().Get("workspace_id")`, or equivalent input-derived workspace reads

## 2. Backend — Handler migration to typed context

- [x] 2.1 Migrate `internal/clients` handlers to read `WorkspaceContext` from `r.Context()` and pass `wsCtx.WorkspaceID` to every repository call
- [x] 2.2 Migrate `internal/projects` handlers likewise
- [x] 2.3 Migrate `internal/tracking` handlers likewise, confirming the active-timer uniqueness check uses `wsCtx.WorkspaceID`
- [x] 2.4 Migrate `internal/rates` handlers likewise
- [x] 2.5 Migrate `internal/reporting` handlers likewise, including the dashboard summary partial
- [x] 2.6 Remove the old session-reading helper (or mark it internal-only to auth flows) once all domain handlers migrate
- [x] 2.7 Run `make test` and `make vet` and confirm a green build after each domain migration

## 3. Backend — Repository audit harness

- [x] 3.1 Create `internal/shared/authz/audit_test.go` that walks repository source files, identifies public methods accepting `workspaceID uuid.UUID`, and asserts each SQL string in the method body references `workspace_id`
- [x] 3.2 Support an `//authz:ok: <reason>` inline allowlist for confirmed-safe exceptions; fail if the comment has no reason
- [x] 3.3 Run the audit and resolve any findings in-scope for this change (fix the SQL or document the exception)
- [x] 3.4 Add a short section to `docs/time_tracking_design_doc.md` pointing at the audit as the canonical workspace-scope enforcement check

## 4. Database — projects/clients composite FK consistency

- [x] 4.1 Add migration `migrations/NNNN_projects_workspace_client_consistency.up.sql` that: (a) verifies no existing `projects` rows violate the invariant and aborts with a descriptive error if any do, (b) adds `UNIQUE (id, workspace_id)` on `clients`, (c) adds composite FK `projects (client_id, workspace_id) REFERENCES clients (id, workspace_id)`
- [x] 4.2 Add the matching `.down.sql` that drops the composite FK and the unique constraint cleanly
- [x] 4.3 Run `make migrate-up`, `make migrate-down`, `make migrate-redo`, and `make dev-seed` end-to-end to confirm the migration is reversible and seed-compatible
- [x] 4.4 Add an integration test that attempts a raw INSERT with a mismatched `workspace_id` via `pgxpool.Pool` and asserts a referential integrity error

## 5. Templates — Shared not-found partial

- [x] 5.1 Create `web/templates/errors/not_found.html` with calm, tool-like copy that does not name the requested resource type or echo URL identifiers
- [x] 5.2 Route every cross-workspace 404 and every "row not found" 404 through this template via a `http.NotFound`-style helper in `internal/shared/http`
- [x] 5.3 Audit existing error templates and delete or redirect any resource-specific 404 pages so response bodies are byte-identical across resource types
- [x] 5.4 Confirm keyboard focus lands on the page's primary heading after the 404 is rendered (no HTMX swap context — full page render)
- [x] 5.5 Verify WCAG 2.2 AA on the not-found page: visible heading, sufficient contrast, visible focus on the "Back to dashboard" link, target size at least 24x24 CSS pixels
- [x] 5.6 Screen-reader spot-check with a landmark role on the error region and a clear `<h1>` announcement

## 6. HTMX — Consistent partial responses on cross-workspace denial

- [x] 6.1 For HTMX-initiated requests that trip a cross-workspace 404, return the shared not-found template fragment (no layout) and set `HX-Retarget` to a global error region if one is in scope; otherwise full-page 404 with `HX-Refresh: true`
- [x] 6.2 Ensure no `HX-Trigger` header is emitted on denied mutations (no `timer-changed`, no `entries-changed`, no `clients-changed`, no `projects-changed`)
- [x] 6.3 Manual test: timer start against a cross-workspace project does not update the dashboard summary widget

## 7. Testing — Cross-workspace denial integration matrix

- [x] 7.1 Add `internal/clients/authz_test.go` with a table-driven test covering list, detail, create, edit, archive, unarchive, delete — each asserting HTTP 404 with the shared not-found body
- [x] 7.2 Add `internal/projects/authz_test.go` covering list, detail, create (including cross-workspace `client_id`), edit, archive, unarchive, delete
- [x] 7.3 Add `internal/tracking/authz_test.go` covering timer start, timer stop, active-timer read, entry list, entry detail, entry edit, entry delete; include the "W1 running does not block W2 start" scenario
- [x] 7.4 Add `internal/rates/authz_test.go` covering rate-rule list, create, edit, delete, and the `Resolve` service path against other-workspace IDs
- [x] 7.5 Add `internal/reporting/authz_test.go` covering dashboard summary, filtered entries list, filter-by-other-workspace-client, filter-by-other-workspace-project
- [x] 7.6 Add a route-coverage test in `internal/shared/http` that enumerates registered authenticated routes under the covered families and fails if any route has no corresponding authz row in the domain test tables
- [x] 7.7 Run `make test` (which uses `-p 1` due to shared Postgres) and confirm a green build with all authz tests passing

## 8. Documentation and archive prep

- [x] 8.1 Update `openspec/specs/workspace/spec.md` purpose section to reference the hardened contract (replace the bootstrap "TBD" placeholder)
- [x] 8.2 Add a short "Authorization" section to `docs/time_tracking_design_doc.md` describing the `WorkspaceContext` contract, the repository audit, and the shared 404 template
- [x] 8.3 Confirm `make fmt`, `make vet`, `make lint`, `make test` all pass locally
- [x] 8.4 Prepare archive checklist: verify all tasks are checked, all spec deltas apply cleanly against `openspec/specs/`, and the change is ready for `/opsx:archive`
