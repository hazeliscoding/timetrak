-- Enforce at-most-one running timer per (workspace, user).
-- A second concurrent POST /timer/start raises SQLSTATE 23505, which the
-- handler converts to a 409 Conflict.
CREATE UNIQUE INDEX ux_time_entries_one_active_per_user_workspace
    ON time_entries (workspace_id, user_id)
    WHERE ended_at IS NULL;
