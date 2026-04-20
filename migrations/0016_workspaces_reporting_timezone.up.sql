-- Add reporting_timezone to workspaces so reports can bucket entries by the
-- user's local calendar day (not UTC). IANA tz name; defaults to 'UTC' so
-- existing workspaces preserve their current UTC bucketing behavior.
--
-- The CHECK enforces a non-empty string. We deliberately do NOT reference
-- pg_timezone_names in a CHECK — it is a view, not a constraint target.
-- Validation against pg_timezone_names happens at write time in the workspace
-- service (see ErrInvalidTimezone).

ALTER TABLE workspaces
    ADD COLUMN reporting_timezone text NOT NULL DEFAULT 'UTC'
    CHECK (length(reporting_timezone) > 0);

-- Backfill verification (informational only). Assert every row has a value
-- and at least every distinct stored value is recognized by Postgres. Fails
-- loudly if somehow a bogus value slipped in before NOT NULL took effect.
DO $$
DECLARE
    null_count        integer;
    unknown_count     integer;
BEGIN
    SELECT count(*) INTO null_count
    FROM workspaces WHERE reporting_timezone IS NULL OR reporting_timezone = '';
    IF null_count > 0 THEN
        RAISE EXCEPTION 'workspaces.reporting_timezone backfill left % empty rows', null_count;
    END IF;

    SELECT count(*) INTO unknown_count
    FROM (
        SELECT DISTINCT reporting_timezone FROM workspaces
    ) d
    LEFT JOIN pg_timezone_names tz ON tz.name = d.reporting_timezone
    WHERE tz.name IS NULL;
    IF unknown_count > 0 THEN
        RAISE EXCEPTION 'workspaces.reporting_timezone contains % unknown IANA names', unknown_count;
    END IF;
END $$;
