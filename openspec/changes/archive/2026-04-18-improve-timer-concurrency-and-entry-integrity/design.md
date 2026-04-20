## Context

The MVP bootstrap gave TimeTrak a working tracking domain: a single running timer per `(workspace_id, user_id)` enforced by the partial unique index `ux_time_entries_one_active_per_user_workspace`, a typed `ErrActiveTimerExists` for 409s, and workspace-scoped read/write enforcement in every repo query. What it did not give us:

- No database-level guarantee that `ended_at > started_at`. A buggy handler or direct SQL path can persist a zero- or negative-duration entry, which silently poisons reports (billable-seconds becomes zero or negative).
- No database-level guarantee that a time entry's `project_id` lives in the same `workspace_id` as the entry. Today this is enforced only by the service's `WHERE workspace_id = ?` filter in the projects repo; any future code path that bypasses the service and constructs a row directly would be free to forge a cross-workspace reference.
- Stop is not strictly idempotent. Two concurrent stops can both read `ended_at IS NULL`, both write `now()`, and the second write wins — producing a non-deterministic `ended_at` that differs between the response body and the stored row.
- The service exposes one typed error (`ErrActiveTimerExists`). Other integrity failures surface as `pgx.PgError` or generic `500`s, making dashboards and HTMX error copy imprecise.

Stage 2 is the stabilize stage; reporting and eventual invoicing both read these rows. Hardening them before building on top is the correct sequencing.

## Goals / Non-Goals

**Goals:**
- Close the remaining tracking integrity gaps (interval CHECK, composite cross-workspace FK) at the database layer so no application path can persist a bad row.
- Make stop deterministic and idempotent under concurrency.
- Replace the one-error taxonomy with a small, stable set of typed errors mapped cleanly to HTTP status codes and stable error-code strings.
- Give operators structured log signal (`tracking.error_kind`) for every integrity failure.
- Surface each failure kind with accessible, domain-specific copy in the HTMX partials.

**Non-Goals:**
- Offline or multi-device sync and reconciliation.
- Per-entry audit log / change history.
- Timer pause/resume semantics.
- Protecting against adversarial client clock skew beyond "server-side `now()` for stop/start".
- Replacing the partial unique index with advisory locks. The index is sufficient for the MVP workload; revisit only if contention is observed.
- Any changes to reporting or rate resolution behavior. Reporting continues to read through `rates.Service.Resolve`.

## Decisions

### Decision 1: Database CHECK constraint for interval validity

Add `CHECK (ended_at IS NULL OR ended_at > started_at)` to `time_entries`.

- **Alternatives considered**:
  - *App-layer validation only*: rejected. Any future code path that bypasses the service (migrations, batch jobs, direct SQL fixes) can persist bad rows. Defense in depth is cheap here.
  - *`ended_at >= started_at`* (allow zero-duration): rejected. A zero-duration entry has no product meaning and pollutes billable-seconds aggregates. Manual entry UX already requires a non-empty interval; making that explicit at the DB level is consistent.
- **Rationale**: The check is O(1) per write, catches all bypass paths, and translates to a single typed error via SQLSTATE 23514.

### Decision 2: Composite FK for same-workspace project reference

Ensure `projects` has a composite unique key on `(id, workspace_id)` (already implied by `id` being PK, but the FK target needs a matching unique constraint), then add a composite FK `time_entries(project_id, workspace_id) REFERENCES projects(id, workspace_id)`.

- **Alternatives considered**:
  - *Trigger checking workspace equality*: rejected. Triggers are harder to reason about and slower than a FK.
  - *Rely on service-layer `WHERE workspace_id = ?`*: rejected for the same defense-in-depth reason as Decision 1.
- **Rationale**: Composite FKs are a well-known pattern for enforcing tenant isolation at the schema level. PostgreSQL enforces them cheaply.

### Decision 3: Stop uses `SELECT ... FOR UPDATE` inside `pool.InTx` with server-side `now()`

```
BEGIN
  SELECT id, started_at, ended_at
    FROM time_entries
   WHERE workspace_id = $1 AND user_id = $2 AND ended_at IS NULL
   FOR UPDATE;
  -- if no row: return ErrNoActiveTimer
  -- if row with ended_at IS NOT NULL (impossible under the WHERE, but handled if WHERE is relaxed): return it unchanged
  UPDATE time_entries
     SET ended_at = now(), duration_seconds = EXTRACT(EPOCH FROM (now() - started_at))::int
   WHERE id = $3 AND ended_at IS NULL
  RETURNING *;
COMMIT
```

- **Alternatives considered**:
  - *Single `UPDATE ... WHERE ended_at IS NULL RETURNING`*: simpler and atomic, but doesn't let us distinguish "no active timer" from "already stopped" in an idempotent follow-up request that references a specific `entry_id`.
  - *Advisory lock per `(workspace_id, user_id)`*: stronger serialization, but overkill when the partial unique index already caps active rows at one per user per workspace.
- **Rationale**: The row-level lock plus server-side `now()` gives deterministic `ended_at` under concurrency and makes the handler trivially testable. The `UPDATE ... WHERE id = $3 AND ended_at IS NULL` guard makes the write itself idempotent even if the lock is somehow released early.

### Decision 4: Error taxonomy mapped to SQLSTATE and HTTP status

| Error                      | SQLSTATE | HTTP | Error code                  |
|----------------------------|----------|------|-----------------------------|
| `ErrActiveTimerExists`     | 23505    | 409  | `tracking.active_timer`     |
| `ErrNoActiveTimer`         | —        | 409  | `tracking.no_active_timer`  |
| `ErrInvalidInterval`       | 23514    | 422  | `tracking.invalid_interval` |
| `ErrCrossWorkspaceProject` | 23503    | 422  | `tracking.cross_workspace`  |

Translation happens in a small `tracking.translatePgError` helper that inspects `pgconn.PgError.Code` and `ConstraintName` so that (e.g.) a 23505 on an unrelated index does not silently become `ErrActiveTimerExists`.

### Decision 5: Structured log field `tracking.error_kind`

Every taxonomy failure logs at `warn` with `tracking.error_kind` equal to the stable error-code string. Unknown SQLSTATE values log at `error` without `tracking.error_kind` and return HTTP 500. This keeps dashboards clean: `tracking.error_kind` is a low-cardinality enum, perfect for a Grafana panel.

### Decision 6: Shared HTMX error partial keyed by error code

A single `web/templates/partials/tracking_error.html` partial receives the error code and renders domain-specific copy. The timer widget and entry edit form both render it via `{{template "partials/tracking_error.html" .}}`. This keeps copy consistent and gives the component-library foundation one more reusable piece.

## Sequence: start/stop race resolution

```mermaid
sequenceDiagram
    autonumber
    participant C1 as Client (Tab 1)
    participant C2 as Client (Tab 2)
    participant H as Tracking Handler
    participant S as Tracking Service
    participant DB as PostgreSQL

    C1->>H: POST /timer/start (P1)
    C2->>H: POST /timer/start (P1)
    H->>S: Start(ctx, W1, Alice, P1)
    H->>S: Start(ctx, W1, Alice, P1)
    S->>DB: INSERT ... (first)
    S->>DB: INSERT ... (second)
    DB-->>S: OK (first)
    DB-->>S: 23505 on ux_time_entries_one_active_per_user_workspace (second)
    S-->>H: running entry (first)
    S-->>H: ErrActiveTimerExists (second)
    H-->>C1: 200 OK + HX-Trigger: timer-changed,entries-changed
    H-->>C2: 409 + tracking_error partial (code: tracking.active_timer)

    Note over C1,DB: Later: two concurrent stops
    C1->>H: POST /timer/stop
    C2->>H: POST /timer/stop
    H->>S: Stop(ctx, W1, Alice)
    H->>S: Stop(ctx, W1, Alice)
    S->>DB: BEGIN; SELECT ... FOR UPDATE (first wins lock)
    S->>DB: BEGIN; SELECT ... FOR UPDATE (blocks)
    DB-->>S: row (running)
    S->>DB: UPDATE ended_at=now() WHERE id=? AND ended_at IS NULL
    DB-->>S: 1 row updated
    S->>DB: COMMIT (first)
    DB-->>S: lock released; row returned with ended_at NOT NULL
    S->>DB: UPDATE ... WHERE ended_at IS NULL
    DB-->>S: 0 rows updated (idempotent no-op)
    S-->>H: entry unchanged (first stop's ended_at)
    H-->>C1: 200 OK
    H-->>C2: 200 OK (same ended_at)
```

## Risks / Trade-offs

- **Latent bad rows block the migration.** → Mitigation: `up` migration runs a pre-flight `SELECT ... WHERE ended_at IS NOT NULL AND ended_at <= started_at` and a cross-workspace pre-flight; aborts with a clear message before `ALTER TABLE`. `make migrate-redo` remains safe.
- **Row lock contention on stop.** → Mitigation: the partial unique index caps active rows at one per `(workspace_id, user_id)`, so the `SELECT ... FOR UPDATE` effectively locks a single row. Pathological retry loops (hundreds of stops per second from one user) are out of scope.
- **SQLSTATE mapping drift.** A future migration that adds an unrelated unique index could collide with 23505 mapping. → Mitigation: `translatePgError` inspects `ConstraintName`, not just `Code`, and defaults to returning the raw error (logged at `error`) if the constraint is unknown.
- **Server clock skew on the DB host.** Using `now()` on the DB removes per-request handler skew but not DB host clock drift. → Mitigation: assume NTP; out of scope.
- **Migration ordering.** The composite FK requires a matching unique key on `projects(id, workspace_id)`. If `projects.id` is already globally unique (it is, since UUID PK), adding a `UNIQUE(id, workspace_id)` is redundant but required by PostgreSQL FK semantics. → Mitigation: add it explicitly in the migration; document the redundancy.

## Migration Plan

1. Ship the migration pair (up + down) gated behind a pre-flight integrity check.
2. Run `make migrate-up` in CI on a snapshot of production-shape data to confirm no latent violations.
3. Deploy backend with feature-flag-free rollout — the new errors are strictly tighter, so existing clients see the same 409 or a new 422 for already-invalid inputs.
4. Monitor the `tracking.error_kind` dashboard for 48 hours. A spike in `tracking.invalid_interval` would indicate a previously silent bug in an edit path.
5. Rollback: `make migrate-down` removes the CHECK and composite FK; the backend gracefully degrades because `translatePgError` falls back to the generic error path.

## Open Questions

- Should `ErrNoActiveTimer` return 409 or 404? Proposed: **409** for consistency with "conflict with current state" semantics; the resource (the user's timer slot) exists, it's just not running. Revisit if UX testing finds it confusing.
- Do we want a `tracking.error_kind = "tracking.unknown"` bucket for unmapped SQLSTATEs, or leave the field unset? Proposed: leave it unset so dashboards don't conflate known and unknown failures.
