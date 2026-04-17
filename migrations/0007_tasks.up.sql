CREATE TABLE tasks (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    project_id      uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            text        NOT NULL CHECK (length(trim(name)) > 0),
    is_archived     boolean     NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ix_tasks_workspace_project ON tasks (workspace_id, project_id);
