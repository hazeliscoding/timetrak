---
name: tt-scaffold-domain
description: Scaffold or extend a workspace-scoped domain package in internal/<domain>/ following TimeTrak's bootstrap conventions (service + handler + authz test). Use when implementing a new domain feature, adding a new entity to an existing domain, or wiring a new HTTP route trio.
license: MIT
compatibility: TimeTrak Go monolith. Requires Go 1.22+.
metadata:
  author: timetrak
  version: "1.0"
---

Scaffold a workspace-scoped repo + service + handler trio under `internal/<domain>/` following the patterns landed in the bootstrap change. Safe to invoke mid-`openspec-apply-change` when a `tasks.md` step says "add service X" or "register handler Y".

**Input**: domain package name (e.g. `invoicing`), entity name (e.g. `Invoice`), and the operations needed (list/create/update/archive/etc.).

**Steps**

1. **Confirm scope**
   - Confirm the package: existing one under `internal/<domain>/` or a new directory.
   - Confirm whether routes are user-facing (mounted via `protect`) or admin-only.
   - If unclear, pause and ask.

2. **Read one existing peer for tone**
   Read `internal/clients/service.go` and `internal/clients/handler.go` as the reference pair. Match: receiver names, error sentinel naming (`Err...`), `NewService(pool *db.Pool)`, `NewHandler(svc, tpls, lay)`.

3. **Create / extend `service.go`**

   Required shape:
   ```go
   // Package <domain> implements the <domain> use cases, all scoped to the active workspace.
   package <domain>

   import (
       "context"
       "errors"

       "github.com/google/uuid"
       "github.com/jackc/pgx/v5"

       "timetrak/internal/shared/db"
   )

   var (
       ErrNotFound  = errors.New("<domain>: not found")
       ErrEmpty<X>  = errors.New("<domain>: <field> must not be empty")
   )

   type Service struct{ pool *db.Pool }

   func NewService(pool *db.Pool) *Service { return &Service{pool: pool} }
   ```

   **Workspace-authz invariant (binding)**:
   - Every method takes `workspaceID uuid.UUID` as a leading arg after `ctx`.
   - Every SQL statement includes `workspace_id = $N` in `WHERE`.
   - Cross-workspace lookup returns `ErrNotFound` (handler maps to HTTP 404, never 403).

   **Money** (binding): integer minor units only — `hourly_rate_minor bigint`, never floats.

   For multi-statement writes, wrap with `pool.InTx(ctx, func(tx pgx.Tx) error { ... })`.

4. **Create / extend `handler.go`**

   Required shape (mirror `internal/clients/handler.go`):
   ```go
   package <domain>

   import (
       "errors"
       "net/http"

       "github.com/google/uuid"

       "timetrak/internal/shared/authz"
       "timetrak/internal/shared/csrf"
       sharedhttp "timetrak/internal/shared/http"
       "timetrak/internal/shared/templates"
       "timetrak/internal/web/layout"
   )

   type Handler struct {
       svc  *Service
       tpls *templates.Registry
       lay  *layout.Builder
   }

   func NewHandler(svc *Service, tpls *templates.Registry, lay *layout.Builder) *Handler {
       return &Handler{svc: svc, tpls: tpls, lay: lay}
   }

   func (h *Handler) Register(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
       mux.Handle("GET /<resource>", protect(http.HandlerFunc(h.list)))
       mux.Handle("POST /<resource>", protect(http.HandlerFunc(h.create)))
       // ... per route
   }
   ```

   **Per-handler checklist**:
   - Pull `workspaceID` via `authz.WorkspaceFromContext(r.Context())` (or the helper used in peer handlers).
   - Validate CSRF on POST/PUT/PATCH/DELETE via `csrf.Validate(r)` — middleware already runs but explicit guards are common in peers; match what `internal/clients/handler.go` does.
   - On `errors.Is(err, ErrNotFound)`: render the global 404 (not 403).
   - Emit HTMX peer-refresh trigger via `HX-Trigger` response header where state changed:
     - tracking: `timer-changed, entries-changed`
     - entries CRUD: `entries-changed`
     - clients CRUD: `clients-changed`
     - projects CRUD: `projects-changed`
     - rates CRUD: `rates-changed`
   - For inline edits: return the row partial (not the full page) and use `data-focus-after-swap` on the focusable element to be restored.

5. **Wire the handler in `cmd/web`**

   Construct the service+handler in `cmd/web/main.go` (or wherever peers are wired) and call `Register(mux, protect)`. Verify the route prefix doesn't collide with existing handlers.

6. **Add `authz_test.go` (cross-workspace denial matrix)**

   Mirror `internal/clients/authz_test.go`. Every registered route MUST have a row that:
   - Authenticates as UserA in WorkspaceA.
   - Targets a resource owned by WorkspaceB.
   - Asserts HTTP 404.
   - Asserts the database row is unchanged.

   Use `testdb.SeedAuthzFixture(t, pool)` for the standard two-workspace fixture.

7. **Verify**
   ```bash
   make fmt && make vet && go build ./...
   go test ./internal/<domain>/...
   ```

**Guardrails**
- Never bypass `workspaceID` in SQL — even for "internal-only" reads.
- Never return 403 for a cross-workspace miss; always 404.
- Money is `*_minor bigint`. If you find yourself writing `float64`, stop.
- No ORM, no third-party router. Stdlib `net/http` + `pgx/v5` only.
- If the change spans multiple domains, propose a separate OpenSpec change rather than umbrellaing.

**Fluid Workflow Integration**
This skill can be invoked mid-`openspec-apply-change` when a task line in `tasks.md` reads like "scaffold service for X", "register handler Y", or "add cross-workspace test for Z". Mark the task `- [x]` only after `make fmt && make vet && go test` pass.
