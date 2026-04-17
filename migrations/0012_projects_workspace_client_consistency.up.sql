-- Enforce projects.workspace_id == referenced client's workspace_id at the
-- database layer. Service code already validates this; this migration makes
-- the invariant a property of the schema, so any buggy INSERT/UPDATE path
-- fails immediately rather than silently writing inconsistent rows.
--
-- Steps:
--   1. Pre-check: refuse to apply if any existing project is inconsistent.
--   2. Add a unique constraint on clients(id, workspace_id) so the composite
--      foreign key has a valid target. (clients.id is already PRIMARY KEY,
--      so this UNIQUE is redundant for uniqueness but required by Postgres
--      as the FK target.)
--   3. Add the composite foreign key on projects(client_id, workspace_id).

DO $$
DECLARE
    inconsistent_count integer;
BEGIN
    SELECT count(*) INTO inconsistent_count
    FROM projects p
    JOIN clients c ON c.id = p.client_id
    WHERE c.workspace_id <> p.workspace_id;

    IF inconsistent_count > 0 THEN
        RAISE EXCEPTION
            'cannot add composite FK: % projects rows have workspace_id that disagrees with their client''s workspace_id. Resolve these rows before applying migration 0012.',
            inconsistent_count;
    END IF;
END $$;

ALTER TABLE clients
    ADD CONSTRAINT clients_id_workspace_uniq UNIQUE (id, workspace_id);

ALTER TABLE projects
    ADD CONSTRAINT projects_client_workspace_fk
    FOREIGN KEY (client_id, workspace_id)
    REFERENCES clients (id, workspace_id);
