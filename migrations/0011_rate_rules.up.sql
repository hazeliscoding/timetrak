CREATE TABLE rate_rules (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    client_id           uuid        NULL REFERENCES clients(id) ON DELETE CASCADE,
    project_id          uuid        NULL REFERENCES projects(id) ON DELETE CASCADE,
    currency_code       char(3)     NOT NULL CHECK (currency_code = upper(currency_code)),
    hourly_rate_minor   bigint      NOT NULL CHECK (hourly_rate_minor >= 0),
    effective_from      date        NOT NULL,
    effective_to        date        NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_rate_rules_window
        CHECK (effective_to IS NULL OR effective_to >= effective_from),
    -- A rule is exactly one level: workspace-default, client, or project.
    CONSTRAINT ck_rate_rules_level
        CHECK (
            (client_id IS NULL AND project_id IS NULL)                 -- workspace default
            OR (client_id IS NOT NULL AND project_id IS NULL)          -- per client
            OR (project_id IS NOT NULL)                                 -- per project (client_id optional)
        )
);

CREATE INDEX ix_rate_rules_workspace_window
    ON rate_rules (workspace_id, effective_from, effective_to);
CREATE INDEX ix_rate_rules_workspace_project ON rate_rules (workspace_id, project_id);
CREATE INDEX ix_rate_rules_workspace_client  ON rate_rules (workspace_id, client_id);
