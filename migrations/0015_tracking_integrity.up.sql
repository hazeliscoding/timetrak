-- Tighten time_entries integrity:
--   1. Pre-flight guards abort the migration if any existing row would violate
--      the new constraints (zero/negative interval or cross-workspace project).
--   2. Add CHECK (ended_at IS NULL OR ended_at > started_at) so zero-duration
--      entries are rejected at the database layer. This supersedes the looser
--      `ck_time_entries_range` (>=) added in 0008 — we keep that one in place
--      as a weaker redundant guard; the new `chk_time_entries_interval` is the
--      authoritative invariant for tracking taxonomy mapping (SQLSTATE 23514).
--   3. Add UNIQUE(id, workspace_id) on projects — required as the target of the
--      composite FK from time_entries (redundant with the PRIMARY KEY on id,
--      but mandated by PostgreSQL FK semantics).
--   4. Replace the single-column FK time_entries.project_id -> projects.id
--      with a composite FK (project_id, workspace_id) -> projects(id, workspace_id)
--      so cross-workspace project references are rejected with SQLSTATE 23503.

DO $$
DECLARE
    bad_interval_count integer;
    cross_workspace_count integer;
BEGIN
    SELECT count(*) INTO bad_interval_count
    FROM time_entries
    WHERE ended_at IS NOT NULL AND ended_at <= started_at;

    IF bad_interval_count > 0 THEN
        RAISE EXCEPTION
            'cannot add chk_time_entries_interval: % time_entries rows have ended_at <= started_at. Resolve before applying migration 0013.',
            bad_interval_count;
    END IF;

    SELECT count(*) INTO cross_workspace_count
    FROM time_entries te
    JOIN projects p ON p.id = te.project_id
    WHERE p.workspace_id <> te.workspace_id;

    IF cross_workspace_count > 0 THEN
        RAISE EXCEPTION
            'cannot add composite FK: % time_entries rows reference a project in a different workspace. Resolve before applying migration 0013.',
            cross_workspace_count;
    END IF;
END $$;

ALTER TABLE time_entries
    ADD CONSTRAINT chk_time_entries_interval
    CHECK (ended_at IS NULL OR ended_at > started_at);

ALTER TABLE projects
    ADD CONSTRAINT uq_projects_id_workspace UNIQUE (id, workspace_id);

-- The default FK name from 0008 is `time_entries_project_id_fkey`.
ALTER TABLE time_entries
    DROP CONSTRAINT time_entries_project_id_fkey;

ALTER TABLE time_entries
    ADD CONSTRAINT time_entries_project_workspace_fk
    FOREIGN KEY (project_id, workspace_id)
    REFERENCES projects (id, workspace_id)
    ON DELETE RESTRICT;
