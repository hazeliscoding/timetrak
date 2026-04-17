ALTER TABLE sessions DROP CONSTRAINT IF EXISTS fk_sessions_active_workspace;
DROP TABLE IF EXISTS workspaces;
