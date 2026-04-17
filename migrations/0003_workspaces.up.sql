CREATE TABLE workspaces (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text        NOT NULL CHECK (length(trim(name)) > 0),
    slug        text        NOT NULL UNIQUE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- Now that workspaces exists, wire the session.active_workspace_id FK.
ALTER TABLE sessions
    ADD CONSTRAINT fk_sessions_active_workspace
    FOREIGN KEY (active_workspace_id) REFERENCES workspaces(id) ON DELETE SET NULL;
