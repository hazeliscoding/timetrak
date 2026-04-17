CREATE TABLE sessions (
    id                      uuid        PRIMARY KEY,
    user_id                 uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    active_workspace_id     uuid        NULL,  -- FK added in 0003 once workspaces exists.
    expires_at              timestamptz NOT NULL,
    created_at              timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ix_sessions_user ON sessions (user_id);
CREATE INDEX ix_sessions_expires ON sessions (expires_at);
