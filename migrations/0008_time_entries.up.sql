CREATE TABLE time_entries (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        uuid        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id             uuid        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    project_id          uuid        NOT NULL REFERENCES projects(id) ON DELETE RESTRICT,
    task_id             uuid        NULL REFERENCES tasks(id) ON DELETE SET NULL,
    description         text        NULL,
    started_at          timestamptz NOT NULL,
    ended_at            timestamptz NULL,
    duration_seconds    integer     NOT NULL DEFAULT 0,
    is_billable         boolean     NOT NULL DEFAULT true,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_time_entries_range    CHECK (ended_at IS NULL OR ended_at >= started_at),
    CONSTRAINT ck_time_entries_duration CHECK (duration_seconds >= 0)
);
