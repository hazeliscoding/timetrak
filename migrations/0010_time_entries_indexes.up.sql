CREATE INDEX ix_time_entries_workspace_started_desc
    ON time_entries (workspace_id, started_at DESC);

CREATE INDEX ix_time_entries_workspace_project_started_desc
    ON time_entries (workspace_id, project_id, started_at DESC);
