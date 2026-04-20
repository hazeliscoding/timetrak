-- Adds a rate snapshot to time_entries so historical billable figures are stable
-- even when the underlying rate_rules row is later edited. All three columns
-- travel together (all-set or all-null) enforced by a CHECK constraint.
--
-- Backfill for existing rows is performed by the companion Go sub-command
-- `go run ./cmd/migrate backfill-rate-snapshots` (or `make backfill-rate-snapshots`),
-- which uses the same precedence logic as `rates.Service.Resolve` and is idempotent.

ALTER TABLE time_entries
    ADD COLUMN rate_rule_id      uuid      NULL REFERENCES rate_rules(id) ON DELETE RESTRICT,
    ADD COLUMN hourly_rate_minor bigint    NULL,
    ADD COLUMN currency_code     char(3)   NULL;

ALTER TABLE time_entries
    ADD CONSTRAINT ck_time_entries_rate_snapshot_atomic
    CHECK (
        (rate_rule_id IS NULL AND hourly_rate_minor IS NULL AND currency_code IS NULL)
        OR
        (rate_rule_id IS NOT NULL AND hourly_rate_minor IS NOT NULL AND currency_code IS NOT NULL)
    );

ALTER TABLE time_entries
    ADD CONSTRAINT ck_time_entries_rate_snapshot_currency
    CHECK (currency_code IS NULL OR currency_code = upper(currency_code));

ALTER TABLE time_entries
    ADD CONSTRAINT ck_time_entries_rate_snapshot_amount
    CHECK (hourly_rate_minor IS NULL OR hourly_rate_minor >= 0);

CREATE INDEX ix_time_entries_workspace_rate_rule
    ON time_entries (workspace_id, rate_rule_id)
    WHERE rate_rule_id IS NOT NULL;
