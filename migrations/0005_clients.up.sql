CREATE TABLE clients (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name            text        NOT NULL CHECK (length(trim(name)) > 0),
    contact_email   text        NULL,
    is_archived     boolean     NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ix_clients_workspace ON clients (workspace_id);
CREATE INDEX ix_clients_workspace_archived ON clients (workspace_id, is_archived);
