## Why

Workspace is the sole authorization boundary for TimeTrak, and the MVP landed the core invariants (every repository method takes `workspaceID` explicitly, cross-workspace access returns HTTP 404, and the active-timer unique index is scoped to `(workspace_id, user_id)`). These invariants are currently upheld by convention: there is no systematic audit, no typed context guarantee that a handler has a verified active workspace, and no integration suite that actively attempts cross-workspace access against every handler. Stage 2's #1 priority is to harden these boundaries before we layer additional behavior (timer integrity, rate edits, reporting) on top of them.

## What Changes

- Introduce a typed `WorkspaceContext` (carrying authenticated `userID` + verified active `workspaceID`) that handlers MUST obtain from a single middleware rather than reading it ad hoc from the session.
- Require every domain handler (clients, projects, tracking, rates, reporting) to receive `WorkspaceContext` via middleware; handlers MUST NOT read `workspace_id` from request form/body/query params for authorization purposes.
- Add a repository-level audit checklist and a lint/test harness: every public repository method that accepts a `workspace_id` parameter MUST include `workspace_id = $N` in its WHERE clause (verified by a scripted static check over SQL strings in `internal/*/repo*.go`).
- Add a cross-workspace authorization integration test matrix: for every mutating and reading handler across clients, projects, tracking, rates, and reporting, a test attempts access with a user whose active workspace differs from the target resource and asserts HTTP 404 with no information disclosure in the body.
- Formalize the "return 404 never 403" rule and the "no resource-existence disclosure" rule as first-class requirements in the workspace spec, with scenarios that apply to each domain handler family.
- Add a `projects.workspace_id` / `projects.client_id` referential integrity check at the database level via a composite foreign key or trigger, closing the gap where a project row could be inserted with a mismatched `workspace_id` through a buggy service path.
- Document the authorization model in `openspec/specs/workspace/spec.md` so the contract is explicit, testable, and discoverable by future agents.

## Capabilities

### New Capabilities
<!-- None. This change hardens existing behavior rather than introducing a new capability. -->

### Modified Capabilities
- `workspace`: Expand the "Workspace is the authorization boundary" requirement with explicit, testable rules for middleware-provided workspace context, response-body information disclosure, and enumeration of the handler families this applies to. Add a new requirement covering repository-level enforcement obligations.
- `clients`: Add cross-workspace denial scenarios covering every mutating and reading handler (list, detail, create, edit, archive, delete) so the 404 contract is exhaustive rather than illustrative.
- `projects`: Strengthen the `projects.workspace_id` / `projects.client_id` consistency invariant with a database-level enforcement requirement, and add exhaustive cross-workspace denial scenarios for list/detail/create/edit/archive/delete.
- `tracking`: Add cross-workspace denial scenarios for timer start, timer stop, entry list, entry edit, entry delete, and the dashboard summary, and require that the active-timer invariant is evaluated strictly within the caller's workspace.
- `reporting`: Add exhaustive cross-workspace denial scenarios covering every reporting endpoint (dashboard summary, billable totals, entry list filters).

## Impact

- **Code**: `internal/shared/http` (middleware, request context types), all domain handler packages (`internal/clients`, `internal/projects`, `internal/tracking`, `internal/rates`, `internal/reporting`), all domain repository packages (for the repository audit), and the integration test tree under each domain. Expect a small set of handler signatures to change as they move from reading session state directly to consuming the typed `WorkspaceContext`.
- **Database**: One migration to add a composite foreign key or check constraint enforcing `projects.workspace_id` matches the referenced client's `workspace_id`.
- **APIs**: No external API surface changes. Response codes for cross-workspace access remain HTTP 404; this change makes that contract exhaustive and tested.
- **Dependencies**: None added. This is pure Go + PostgreSQL hardening.
- **Risk**: The repository audit may surface latent bugs; each finding is resolved within this change (they are in-scope). Handler signature changes are mechanical but touch every domain package.

## Scope

**In scope**
- Typed workspace request context and middleware-enforced handler entrypoints
- Repository-level WHERE-clause audit and a repeatable check
- Exhaustive cross-workspace integration test matrix across all existing handler families
- Database-level `projects.workspace_id` ↔ `projects.client_id` consistency enforcement
- Spec updates in `workspace`, `clients`, `projects`, `tracking`, `reporting`

**Out of scope**
- Role-based distinctions beyond the existing `owner`/`admin`/`member` (deferred until team invitations land in Stage 3)
- Multi-workspace invitations, membership changes, or workspace deletion flows
- Audit logging of denied requests (future observability change)
- Rate-limit tightening on auth endpoints (separate Stage 2 item)
- Any UI visual changes beyond error page copy consistency

## Assumptions and Risks

- Assumes the current session layer correctly resolves the user's active workspace; this change depends on that behavior but does not re-architect it.
- Assumes all existing handlers can be retrofitted to the typed context without breaking existing HTMX partial-refresh triggers.
- Risk: adding a composite foreign key on `projects` may require a data backfill if any existing rows are inconsistent. The migration MUST verify consistency before applying the constraint and fail loudly otherwise.
- Risk: the repository audit may surface a handler that currently leaks resource existence via differing response codes (e.g., 400 vs 404). Those are resolved in this change.

## Likely Follow-ups

- Audit logging of denied cross-workspace access (observability change)
- Role-based authorization (Stage 3, when team features land)
- A CI-enforced lint rule that fails the build if a new repository method omits `workspace_id` from its WHERE clause
