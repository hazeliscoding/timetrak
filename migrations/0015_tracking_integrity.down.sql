-- Reverse 0013 in reverse order: drop composite FK, restore single-column FK,
-- drop the projects composite UNIQUE, drop the interval CHECK.

ALTER TABLE time_entries
    DROP CONSTRAINT IF EXISTS time_entries_project_workspace_fk;

ALTER TABLE time_entries
    ADD CONSTRAINT time_entries_project_id_fkey
    FOREIGN KEY (project_id)
    REFERENCES projects (id)
    ON DELETE RESTRICT;

ALTER TABLE projects
    DROP CONSTRAINT IF EXISTS uq_projects_id_workspace;

ALTER TABLE time_entries
    DROP CONSTRAINT IF EXISTS chk_time_entries_interval;
