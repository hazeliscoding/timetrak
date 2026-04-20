---
name: tt-add-migration
description: Create a numbered SQL migration pair under migrations/ following TimeTrak's conventions (timestamptz, uuid PKs, integer minor units, workspace_id FK). Use when implementing a task that adds, alters, or backfills a table.
license: MIT
compatibility: TimeTrak Postgres + in-repo migration runner at cmd/migrate.
metadata:
  author: timetrak
  version: "1.0"
---

Create a `NNNN_<name>.up.sql` / `NNNN_<name>.down.sql` pair under `/home/hazeliscoding/projects/timetrak/migrations/`. The runner lives at `cmd/migrate/`; there is no `golang-migrate` or `goose`. Safe to invoke mid-`openspec-apply-change` when `tasks.md` requires schema work.

**Steps**

1. **Pick the next number**
   ```bash
   ls /home/hazeliscoding/projects/timetrak/migrations/ | grep -oE '^[0-9]+' | sort -n | tail -1
   ```
   Increment by 1 and pad to 4 digits (e.g. `0013`). Migration names are `snake_case` and describe the change (`add_invoices`, `time_entries_add_note_index`).

2. **Write the `.up.sql`**

   Conventions (binding):
   - `id uuid PRIMARY KEY DEFAULT gen_random_uuid()` (the `pgcrypto` / `gen_random_uuid()` is already enabled by earlier migrations).
   - All timestamps: `timestamptz NOT NULL DEFAULT now()`.
   - Money columns: `<name>_minor bigint NOT NULL CHECK (<name>_minor >= 0)`. Never `numeric`/`float`.
   - Currency columns: `currency_code char(3) NOT NULL CHECK (currency_code = upper(currency_code))`.
   - Every domain table that holds tenant data MUST have `workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE`.
   - Indexes that filter by workspace MUST lead with `workspace_id`: `CREATE INDEX ix_<table>_workspace_<col> ON <table> (workspace_id, <col>);`.
   - Use named `CONSTRAINT ck_<table>_<rule>` for non-trivial CHECKs.
   - Soft-archive via `is_archived boolean NOT NULL DEFAULT false` rather than deleting.

   Reference: `migrations/0011_rate_rules.up.sql` for a feature-rich example.

3. **Write the `.down.sql`**

   - Must be the precise inverse of the up.
   - For `CREATE TABLE`: down is `DROP TABLE IF EXISTS <table>;` (FKs cascade).
   - For `ALTER TABLE ADD COLUMN`: down is `ALTER TABLE <t> DROP COLUMN IF EXISTS <c>;`.
   - For `CREATE INDEX`: down is `DROP INDEX IF EXISTS <name>;`.
   - For backfills: down should restore the prior value where feasible, or be an explicit no-op with a comment explaining why.

4. **Apply locally and verify**
   ```bash
   make db-up           # if Postgres isn't running
   make migrate-up
   make migrate-redo    # exercises down + up of the latest
   ```

   Then run `psql "$DATABASE_URL"` and `\d+ <table>` to confirm columns, indexes, and constraints landed.

5. **Run the test suite**
   ```bash
   make test
   ```
   Integration tests truncate domain tables; if you added a new domain table that needs truncation, append it to the `TRUNCATE` list in `internal/shared/testdb/testdb.go`.

**Guardrails**
- Never edit an already-applied migration in `main`. Add a new one.
- Never store money as `numeric` or `double precision`.
- Never omit `workspace_id` from a tenant table or its primary lookup index.
- Never drop a column in `up.sql` without a tested `down.sql` that recreates it.
- If the change requires data backfill that can't be expressed inversely, document the irreversibility at the top of the down file.

**Fluid Workflow Integration**
Invoke when `tasks.md` (in an OpenSpec change) lists "add migration for X" or "alter <table> to ...". After `make migrate-redo` and `make test` pass, tick the task `- [x]` and continue.
