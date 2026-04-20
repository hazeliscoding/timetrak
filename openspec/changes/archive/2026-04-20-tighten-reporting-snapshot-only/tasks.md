## 1. Backend: remove fallback from reporting

- [x] 1.1 Delete the `REPORTING_RESOLVE_FALLBACK` env-flag branch in `internal/reporting/service.go` (the `resolveFallbackEnabled()` helper and the fallback block inside `estimateScoped`)
- [x] 1.2 Remove the `rates *rates.Service` field from `reporting.Service` and the `rs *rates.Service` argument from `NewService`
- [x] 1.3 Remove the `timetrak/internal/rates` import from `internal/reporting/service.go`
- [x] 1.4 Update the package-level doc comment in `internal/reporting/service.go` to state the snapshot-only invariant as steady-state (not transitional) and drop any mention of `REPORTING_RESOLVE_FALLBACK`
- [x] 1.5 Update the `estimateScoped` doc comment so it matches the new single-path behavior

## 2. Wiring

- [x] 2.1 Update `cmd/web/main.go` so `reporting.NewService` is called without a `*rates.Service` argument
- [x] 2.2 `go vet ./...` and `make lint` to confirm no stale references to the removed argument or env var
- [x] 2.3 Grep the repo for `REPORTING_RESOLVE_FALLBACK` and delete any remaining references (docs, `.env.example`, READMEs, comments)

## 3. Deploy gate: `check-rate-snapshots`

- [x] 3.1 Add `cmd/migrate/check.go` with a `CheckRateSnapshots(ctx, pool)` function that runs `SELECT workspace_id, count(*) FROM time_entries WHERE ended_at IS NOT NULL AND is_billable = true AND rate_rule_id IS NULL GROUP BY workspace_id`
- [x] 3.2 Wire a `check-rate-snapshots` subcommand in `cmd/migrate/main.go` that prints the summary and exits non-zero when the total is > 0
- [x] 3.3 Add `make check-rate-snapshots` target in `Makefile` mirroring `make backfill-rate-snapshots`
- [x] 3.4 Running `make check-rate-snapshots` on a clean dev DB exits 0; seeding a NULL-snapshot closed billable entry makes it exit non-zero with the workspace listed (manual verification — requires running Postgres; covered by integration test `TestCheckRateSnapshotsFlagsNullSnapshotEntry`)

## 4. Tests

- [x] 4.1 In `internal/reporting/snapshot_test.go`, delete any test that sets/unsets `REPORTING_RESOLVE_FALLBACK`
- [x] 4.2 Keep (or add if missing) Contract A: closed billable entry with snapshot is unaffected by a later rate-rule edit — assert the estimated amount is the snapshot value
- [x] 4.3 Keep (or add if missing) Contract B: closed billable entry with NULL snapshot contributes zero and increments `EntriesWithoutRate` — and the rates service is not consulted
- [x] 4.4 Add a test that a running timer (`ended_at IS NULL`) never contributes to estimated billable amounts or `EntriesWithoutRate`, even when its rate-rule snapshot columns are NULL
- [x] 4.5 Add an integration test for `CheckRateSnapshots`: returns a non-zero offender count for a seeded NULL-snapshot closed billable entry, and zero when all such entries have snapshots
- [x] 4.6 Run `make test` and confirm all reporting and migrate tests pass with no `REPORTING_RESOLVE_FALLBACK` in the environment

## 5. Documentation

- [x] 5.1 Remove the `REPORTING_RESOLVE_FALLBACK` bullet from `CLAUDE.md` (the "reporting" or "implementation choices" section) if present (no bullet was present; the `Rate resolution` bullet was updated to describe the new snapshot-only invariant and deploy gate)
- [x] 5.2 Update the reporting section of `docs/time_tracking_design_doc.md` to state snapshot-only reads as the read-path invariant
- [x] 5.3 Update `cmd/migrate/backfill.go` doc comment to note that the reporting fallback has been removed and the backfill is a hard prerequisite before deploying this change
- [x] 5.4 If `.env.example` mentions `REPORTING_RESOLVE_FALLBACK`, remove the line (no such line existed)
- [x] 5.5 Add a short CHANGELOG-style note (in the proposal or a release-notes file the project already uses) flagging `REPORTING_RESOLVE_FALLBACK` as ignored/removed so operators know to unset it

## 6. Verification

- [x] 6.1 Manually run `make backfill-rate-snapshots` then `make check-rate-snapshots` against a local DB to confirm the green path
- [x] 6.2 Spot-check dashboard and reports pages render `Entries without a rate` correctly when a NULL-snapshot closed billable entry is seeded
- [x] 6.3 Confirm a retroactive edit to a `rate_rules` row does not change any report figure in a range covered by snapshots
