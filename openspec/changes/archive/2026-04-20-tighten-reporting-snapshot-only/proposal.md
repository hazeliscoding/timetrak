## Why

Migration 0013 added a rate snapshot (`rate_rule_id`, `hourly_rate_minor`, `currency_code`) to `time_entries` so historical billable totals cannot silently shift when a rate rule is later edited. The reporting read path already prefers the snapshot, but still carries a transitional `REPORTING_RESOLVE_FALLBACK` escape hatch that calls `rates.Service.Resolve` at read time for any closed entry whose snapshot is NULL. That fallback existed only to bridge the backfill window for pre-0013 rows, and every week it stays enabled is another week where a retroactive rule edit can move closed-period totals. The backfill has now run; it is time to remove the fallback, make the snapshot the *only* source of truth for closed-entry reporting, and push the "no snapshot" case into a countable, visible data-quality signal instead of a silently patched number.

## What Changes

- **BREAKING (internal)**: remove the `REPORTING_RESOLVE_FALLBACK` environment flag and the resolve-at-read branch in `internal/reporting/service.go` — closed entries with a NULL snapshot will contribute zero to estimated billable amounts and increment `EntriesWithoutRate` instead of being silently resolved
- Remove the `*rates.Service` dependency from `reporting.Service` (reporting no longer consults rate rules on the read path at all — the field, constructor argument, and wiring go away)
- Keep the existing SQL aggregation that already reads from snapshot columns; no storage or index changes
- Update the reporting spec to state snapshot-only reads as a normative requirement, not a transitional preference
- Add a dashboard + reports surface line (already rendered as "Entries without a rate") that remains the single way an operator learns about un-snapshotted historical rows
- Ship a pre-flight check: `go run ./cmd/migrate check-rate-snapshots` (and `make check-rate-snapshots`) that fails non-zero when any closed billable entry has a NULL snapshot, so CI / deploy gates can block a release that would start hiding billable amounts
- Update `cmd/migrate/backfill.go` documentation to note the fallback is gone and backfill is a hard prerequisite, not a convenience

Out of scope:
- Adding a NOT NULL database constraint on the snapshot columns (deferred — see follow-ups). Closed entries can still exist with a NULL snapshot; they are simply reported as `No rate` and counted.
- Changing the `Resolve` function, rate-rule storage, or rate-resolution precedence.
- Changing dashboard or reports HTML/HTMX markup beyond the copy already in place.
- Retroactively re-snapshotting entries whose rate rule was later edited (the snapshot is by design frozen at stop/save time).

## Capabilities

### New Capabilities
<!-- none — this change hardens an existing capability -->

### Modified Capabilities
- `reporting`: strengthen the "Estimated billable amount" requirement to mandate snapshot-only reads for closed entries and forbid resolve-at-read; retire the transitional fallback as a spec-visible concept

## Impact

- Code: `internal/reporting/service.go` (delete fallback branch, drop rates field and constructor arg), `cmd/web/main.go` (stop wiring `rates.Service` into `reporting.NewService`), new `cmd/migrate/check.go` + subcommand, `Makefile` target
- Tests: `internal/reporting/snapshot_test.go` — keep the "closed entry with NULL snapshot contributes zero and increments `EntriesWithoutRate`" case as the sole behavior; remove any test that exercised the `REPORTING_RESOLVE_FALLBACK` branch
- Docs: spec update (snapshot-only), `CLAUDE.md` bullet referencing the fallback (remove), `docs/time_tracking_design_doc.md` reporting section (clarify snapshot-only invariant)
- Ops: any environment that still sets `REPORTING_RESOLVE_FALLBACK` should unset it; the new `check-rate-snapshots` step should be added to deploy CI
- Risk: if an un-backfilled environment upgrades without running the check, historical `No rate` counts will jump. Mitigation: the deploy gate check; a clear release note.
- Follow-ups (not in this change):
  - Enforce `rate_rule_id IS NOT NULL` for closed billable entries via a partial CHECK once all known environments are clean
  - Surface a per-entry "re-snapshot" action for the "correct an obvious data bug" case

## Release Notes

- **Removed**: the `REPORTING_RESOLVE_FALLBACK` environment variable is no longer read. Unset it in any runbook, CI config, or deploy environment — lingering values are silently ignored but should be cleaned up.
- **Added**: `make check-rate-snapshots` (exposed from `go run ./cmd/migrate check-rate-snapshots`). Exits non-zero when any closed billable `time_entries` row has a NULL `rate_rule_id`. Wire it into deploy CI after `make backfill-rate-snapshots`.
- **Behavior change**: reporting now reads exclusively from snapshot columns for closed entries. Any closed billable entry with a NULL snapshot will contribute zero to estimated billable amounts and will increment the `Entries without a rate` counter on dashboards and reports until operators run the backfill.
