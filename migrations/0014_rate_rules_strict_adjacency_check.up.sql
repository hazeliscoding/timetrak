-- Diagnostic-only probe. This migration does NOT mutate data.
--
-- The service-level overlap check tightens from "effective_to >= next.effective_from"
-- to rejecting shared boundaries (A.effective_to = B.effective_from). Existing data
-- might already contain such same-level adjacencies; this probe raises loudly so an
-- operator can shift A.effective_to back by one day before the new app code is rolled.
--
-- Rule-pair scope: same workspace, same precedence tier (workspace-default, same
-- client, same project).

DO $$
DECLARE
    conflict_count int;
BEGIN
    SELECT count(*)
      INTO conflict_count
      FROM rate_rules a
      JOIN rate_rules b
        ON a.workspace_id = b.workspace_id
       AND a.id <> b.id
       AND a.effective_to IS NOT NULL
       AND a.effective_to = b.effective_from
       AND COALESCE(a.client_id,  '00000000-0000-0000-0000-000000000000'::uuid)
         = COALESCE(b.client_id,  '00000000-0000-0000-0000-000000000000'::uuid)
       AND COALESCE(a.project_id, '00000000-0000-0000-0000-000000000000'::uuid)
         = COALESCE(b.project_id, '00000000-0000-0000-0000-000000000000'::uuid);

    IF conflict_count > 0 THEN
        RAISE EXCEPTION
            'rate_rules adjacency conflict: % pair(s) share a boundary date (A.effective_to = B.effective_from). Shift A.effective_to back by one day, then re-run migrations.',
            conflict_count;
    END IF;
END $$;
