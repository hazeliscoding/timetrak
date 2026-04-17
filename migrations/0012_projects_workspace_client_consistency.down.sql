ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_client_workspace_fk;
ALTER TABLE clients DROP CONSTRAINT IF EXISTS clients_id_workspace_uniq;
