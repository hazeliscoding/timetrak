CREATE TABLE projects (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    client_id           uuid        NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    name                text        NOT NULL CHECK (length(trim(name)) > 0),
    code                text        NULL,
    is_archived         boolean     NOT NULL DEFAULT false,
    default_billable    boolean     NOT NULL DEFAULT true,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ix_projects_workspace_client ON projects (workspace_id, client_id);
CREATE INDEX ix_projects_workspace_archived ON projects (workspace_id, is_archived);
