## Context

The snapshot columns on `time_entries` (`rate_rule_id`, `hourly_rate_minor`, `currency_code`) were introduced by migration 0013 specifically so closed-period reports stop being affected by later edits to `rate_rules`. The reporting service currently reads from the snapshot in its main aggregation SQL, but also carries a transitional branch gated by `REPORTING_RESOLVE_FALLBACK` that issues `rates.Service.Resolve` for any closed entry whose snapshot is NULL. That branch was always advertised as temporary:

- The package doc in `internal/reporting/service.go` names this change (`tighten-reporting-snapshot-only`) as the place it will be removed.
- The backfill subcommand (`go run ./cmd/migrate backfill-rate-snapshots`) exists precisely to eliminate the need for resolve-at-read.

Keeping the fallback in production has real cost:
- A retroactive rule edit for an un-snapshotted entry changes past totals.
- Two callers can see different numbers depending on whether the env flag is set.
- The `rates` service leaks into the reporting dependency graph for a code path that should not exist at steady state.

## Goals / Non-Goals

**Goals:**
- Make the snapshot the *only* source of truth for closed-entry estimated billable amounts.
- Remove the `REPORTING_RESOLVE_FALLBACK` flag and every branch it controls.
- Sever the compile-time dependency from `reporting.Service` to `rates.Service`.
- Ship a CI-friendly command that fails when closed billable entries have a NULL snapshot, so operators cannot silently regress into the hidden-`No rate` state.

**Non-Goals:**
- Adding a DB-level NOT NULL / partial CHECK constraint on the snapshot columns. That is a strictly stronger invariant and deserves its own change once the check gate has been green in all environments for a full billing cycle.
- Changing the snapshot write path in tracking, or adding a repair UI for already-NULL rows.
- Changing dashboard / reports markup. The existing `Entries without a rate` surface is sufficient.

## Decisions

### 1. Delete the fallback branch outright — no soft-landing flag

Alternatives considered:
- Leave the branch but default the env flag to off. Rejected: keeps the rates coupling, keeps a foot-gun in place, and keeps two possible aggregation paths for the same call.
- Delete immediately. Chosen: the fallback has a clear replacement (backfill, then a hard check), and leaving it in a "default-off" state has none of the benefits of removal.

### 2. Drop the `*rates.Service` field from `reporting.Service`

The field is currently documented as "retained for future live-preview use cases". That's speculation, and speculation shouldn't survive this cleanup. If a future change needs a live preview, it can wire the dependency back in with a targeted justification. Leaving it in after this change would be a lie: the comment says "not used on the read path", but the type signature says "may be used". Align the two by removing both.

### 3. Add a dedicated pre-flight command: `migrate check-rate-snapshots`

A simple SQL check run under the existing migrate binary:

```sql
SELECT count(*)
FROM time_entries
WHERE ended_at IS NOT NULL
  AND is_billable = true
  AND rate_rule_id IS NULL
```

Non-zero result exits non-zero, prints a per-workspace count, and reminds the operator to run `migrate backfill-rate-snapshots`. Reuses the existing `db.Pool` wiring in `cmd/migrate`. No new dependencies.

Alternatives:
- A spec-level requirement with no check command. Rejected: specs don't stop deploys.
- A boot-time check inside `cmd/web`. Rejected: webserver start is the wrong moment to discover data debt; it makes rollbacks painful.

### 4. Keep "NULL snapshot" a recoverable state at the DB level

The snapshot columns stay nullable. Reasons:
- Rollback safety — if this change needs a hot revert, no schema change needs to be undone.
- The backfill command and future repair tooling need to distinguish "snapshot has never been written" from "snapshot is deliberately zero". A NOT NULL constraint forces us to invent a sentinel rule, which we don't have.
- The check command already handles the "you forgot to backfill" case loudly.

A follow-up change can harden this to a partial CHECK once deploy gates have been stable for a full billing cycle. This change doesn't preempt that.

### 5. Tests: narrow to two clear contracts

- **Contract A**: For a closed billable entry with a snapshot, reporting uses the snapshot — not the current rate rule — even after the underlying rule is edited.
- **Contract B**: For a closed billable entry without a snapshot, reporting contributes zero and increments `EntriesWithoutRate`. No environment variable affects this.

Any test exercising `REPORTING_RESOLVE_FALLBACK` is deleted outright. A new integration test covers `check-rate-snapshots`: it fails when a NULL-snapshot closed billable entry exists and passes when none do.

## Risks / Trade-offs

- **[Risk] An un-backfilled environment upgrades and sees a sudden jump in `Entries without a rate`.** → Mitigation: CI wires `make check-rate-snapshots` as a deploy prerequisite; release notes call it out as a hard prerequisite, not a recommendation. A visible count is better than the fallback's silent "looks right but secretly re-resolves".
- **[Risk] Some future live-preview feature wishes `reporting.Service` still had `*rates.Service`.** → Mitigation: accept the minor inconvenience; that feature re-wires the dependency with a focused proposal. The current field is not load-bearing.
- **[Trade-off] Not adding a NOT NULL constraint now.** → Downside: the "NULL snapshot" state remains possible in production. Upside: rollback stays trivial, and the hardening is clean, one-purpose, later. The check command is the interim enforcement point.
- **[Trade-off] The check is gate-only, not self-healing.** → The backfill command already exists; duplicating it into an auto-repair path would hide data debt behind an opaque "we'll fix it for you" step, which is exactly the failure mode this change is removing.

## Migration Plan

1. Operators run `make backfill-rate-snapshots` (or `go run ./cmd/migrate backfill-rate-snapshots`) against every environment that has time entries predating migration 0013 — this step was already expected during the 0013 rollout.
2. Operators run `make check-rate-snapshots`; a zero exit code is the green light.
3. Deploy this change. The `REPORTING_RESOLVE_FALLBACK` env var becomes ignored; if any runbook or CI still sets it, remove the setting but there is no failure if it lingers.
4. Rollback: revert the code change; no schema rollback is needed because no schema changed.
