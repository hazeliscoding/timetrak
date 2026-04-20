DROP INDEX IF EXISTS ix_time_entries_workspace_rate_rule;

ALTER TABLE time_entries
    DROP CONSTRAINT IF EXISTS ck_time_entries_rate_snapshot_amount,
    DROP CONSTRAINT IF EXISTS ck_time_entries_rate_snapshot_currency,
    DROP CONSTRAINT IF EXISTS ck_time_entries_rate_snapshot_atomic;

ALTER TABLE time_entries
    DROP COLUMN IF EXISTS currency_code,
    DROP COLUMN IF EXISTS hourly_rate_minor,
    DROP COLUMN IF EXISTS rate_rule_id;
