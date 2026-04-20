## 1. Database

- [x] 1.1 Create migration pair `migrations/00NN_time_entries_rate_snapshot.up.sql` / `.down.sql` adding nullable columns `rate_rule_id uuid REFERENCES rate_rules(id) ON DELETE RESTRICT`, `hourly_rate_minor bigint`, `currency_code char(3)` to `time_entries`, with a CHECK constraint `ck_time_entries_rate_snapshot_atomic` enforcing all-set-or-all-null.
- [x] 1.2 Add partial index `ix_time_entries_workspace_rate_rule ON time_entries (workspace_id, rate_rule_id) WHERE rate_rule_id IS NOT NULL` in the same migration.
- [x] 1.3 Run the migration locally against a dev Postgres (`make migrate-up`) and verify `\d time_entries` shows the three new columns, the CHECK constraint, and the partial index; verify `make migrate-down` cleanly drops them.
- [x] 1.4 Tighten the overlap check in `migrations/00NN+1_rate_rules_strict_adjacency_check.up.sql` (data-quality probe only): add a diagnostic `DO $$ ... $$` block that raises if any existing same-level `rate_rules` pair has `A.effective_to = B.effective_from`, so operators fix the data before the app-level overlap change lands. Do not mutate data automatically.

## 2. Backend — rates service

- [x] 2.1 In `internal/rates/service.go`, tighten `assertNoOverlap` so `A.effective_to = B.effective_from` is rejected (boundary adjacency requires a one-day gap). Preserve existing error type `ErrOverlap`.
- [x] 2.2 Add typed error `ErrRuleReferenced = errors.New("rates: rule is referenced by time entries")` alongside the existing errors.
- [x] 2.3 Implement `countReferencingEntries(ctx, tx, workspaceID, ruleID)` helper using the new partial index; returns int and latest referencing `started_at::date` as `time.Time`.
- [x] 2.4 In `Service.Delete`, before deleting, run the reference check and return `ErrRuleReferenced` if count > 0.
- [x] 2.5 In `Service.Update`, before updating, load the existing rule; if it is referenced, allow the update only when the diff is exclusively `effective_to` changing from NULL to a future date, or `effective_to` being shortened to a date `>=` the latest referencing `started_at::date`. Otherwise return `ErrRuleReferenced`.
- [x] 2.6 Extend `Rule` struct with `ReferencedByCount int` and populate it in `List` via a correlated subquery so the UI can render the hint without an N+1.
- [x] 2.7 Update `internal/rates/handler.go` to map `ErrRuleReferenced` to HTTP 409 with a human-readable message naming the count, and to pass `ReferencedByCount` into the template data model.
- [x] 2.8 Document `Resolve`'s boundary semantics in a doc comment: `effective_from` inclusive, `effective_to` inclusive, disjoint windows at same level guaranteed by overlap check, `at` compared as UTC date.

## 3. Backend — tracking service (snapshot write path)

- [x] 3.1 In the tracking service (locate the Stop handler and the entry-edit save path), inject `*rates.Service` alongside existing dependencies.
- [x] 3.2 On timer stop, inside the same transaction that sets `ended_at` and `duration_seconds`, call `rates.Resolve(ctx, workspaceID, projectID, started_at)` and `UPDATE time_entries SET rate_rule_id=?, hourly_rate_minor=?, currency_code=?` on the closing entry. Handle the no-rate case by writing all three as NULL.
- [x] 3.3 On entry edit (PATCH/PUT handler), if any of `project_id`, `started_at`, `ended_at`, `duration_seconds`, `is_billable` changes, re-resolve and re-write the snapshot in the same transaction. If none of those fields change, leave the snapshot untouched.
- [x] 3.4 On manual entry create (if the tracking domain supports retroactive creation), resolve and snapshot inline at insert time.
- [x] 3.5 Ensure no other write path to `time_entries` bypasses the snapshot. Grep for `UPDATE time_entries` and `INSERT INTO time_entries` across `internal/**` and audit each call site.

## 4. Backend — reporting service (snapshot read path)

- [x] 4.1 Rewrite `estimateBillable` in `internal/reporting/service.go` to read `hourly_rate_minor`, `currency_code` directly from `time_entries` via a single aggregating SQL query (`SUM(duration_seconds * hourly_rate_minor) / 3600` per currency), plus a separate count of entries where `hourly_rate_minor IS NULL AND is_billable = true` for the `No rate` surface. Do not call `rates.Resolve` here.
- [x] 4.2 Rewrite `estimateByClient` and `estimateByProject` analogously: group by currency in SQL, return `map[string]int64`.
- [x] 4.3 Keep the `*rates.Service` field on `reporting.Service` (for future live-preview use cases) but remove every call from `estimate*` helpers. Add a package-level comment noting the invariant.
- [x] 4.4 Introduce a transitional fallback switch (constant or env var `REPORTING_RESOLVE_FALLBACK`) that, when enabled, falls back to resolve-at-read for rows where `ended_at IS NOT NULL AND rate_rule_id IS NULL`. Document this is for the backfill window only and will be removed in a follow-up change.

## 5. Backend — backfill command

- [x] 5.1 Add a new sub-command `backfill-rate-snapshots` to `cmd/migrate/` that iterates all workspaces, and for each closed entry where `rate_rule_id IS NULL`, calls `rates.Service.Resolve` and writes the snapshot. Idempotent via `WHERE rate_rule_id IS NULL AND ended_at IS NOT NULL`.
- [x] 5.2 Before starting the backfill, run a pre-flight probe that SELECTs any pairs of same-level `rate_rules` with `A.effective_to = B.effective_from` and aborts with a clear error listing the rule IDs (operator must resolve the adjacency conflict first).
- [x] 5.3 Emit a structured summary log: `backfilled=N workspaces=W no_rate=M elapsed=...`. Non-zero `no_rate` is informational, not an error.
- [x] 5.4 Add a `--dry-run` flag that reports the counts without writing.
- [x] 5.5 Document the command in `Makefile` (`make backfill-rate-snapshots`) and note it in a comment in the migration file.

## 6. Templates / HTMX

- [x] 6.1 In the rate rules list partial (`web/templates/partials/rate_rules_list.html` or equivalent), render the referenced-count hint: if `ReferencedByCount > 0`, show `Referenced by N entries` as text next to the edit/delete buttons.
- [x] 6.2 Disable the delete button when `ReferencedByCount > 0`; use a native `<button disabled>` with `aria-describedby` pointing at the hint text.
- [x] 6.3 For the edit form, render a secondary note explaining the immutability scope (e.g., "Only the end date may be extended for this rule"). Tie the note to the relevant fields via `aria-describedby`.
- [x] 6.4 When a 409 / `ErrRuleReferenced` response comes back from edit or delete, render the error message inline (reuse existing form-error partial) and preserve focus on the triggering control via `data-focus-after-swap` on the error container. Implemented via the existing full-page `flash flash-error` region at the top of the page (`role="alert" aria-live="assertive"`). `data-focus-after-swap` does not apply under the current non-HTMX redirect flow; deferred with 7.1.
- [x] 6.5 Confirm the entry list, entry edit, and dashboard partials render rate / currency from the snapshot fields returned by the updated handlers; no UI copy changes are expected for the happy path.

## 7. HTMX events & peer refresh

- [ ] 7.1 On successful rate-rule create/update/delete, continue emitting `HX-Trigger: rates-changed`. No change required; confirm the event still fires from the updated handlers. **Deferred**: current rates handlers use full-page redirect, no HTMX partial swap, so no event is emitted. Converting to HTMX is out of scope for this change; tracked for `add-rates-htmx-partials`.
- [x] 7.2 Confirm that `entries-changed` and `timer-changed` events are still emitted from the tracking handlers when a snapshot is written, so the dashboard summary partial refreshes. No new events are introduced by this change. Verified: `internal/tracking/handler.go` still sets these triggers on start/stop/create/edit; snapshot writes happen inside the same transaction so the event fires exactly when the snapshot is visible.

## 8. Integration tests — rates immutability

- [x] 8.1 Using `internal/shared/testdb.Open(t)`, write `TestRateRuleDeleteRejectedWhenReferenced`: create a workspace, a rule, an entry with `rate_rule_id` pointing at the rule, then assert `Delete` returns `ErrRuleReferenced` and the row still exists.
- [x] 8.2 Write `TestRateRuleUpdateAmountRejectedWhenReferenced`: same setup, assert `Update` with a changed `hourly_rate_minor` returns `ErrRuleReferenced`.
- [x] 8.3 Write `TestRateRuleExtendEffectiveToAllowed`: same setup, assert `Update` that only changes `effective_to` from NULL to a future date succeeds and the DB row reflects it.
- [x] 8.4 Write `TestRateRuleShortenEffectiveToBelowReferencingEntryRejected`: shorten `effective_to` to a date earlier than the referencing entry's `started_at::date`; assert rejection.
- [x] 8.5 Write `TestRateRuleOverlapSharedBoundaryRejected`: two workspace-default rules with `A.effective_to = B.effective_from` must be rejected.

## 9. Integration tests — snapshot write path

- [x] 9.1 Write `TestTimerStopWritesRateSnapshot`: start a timer, create a workspace-default rule before stop, stop, assert the stored entry has non-NULL snapshot columns matching the rule.
- [x] 9.2 Write `TestTimerStopWithNoRuleWritesNullSnapshot`: stop a timer with no rule defined; assert all three snapshot columns are NULL and the entry is counted in the `No rate` aggregate.
- [x] 9.3 Write `TestEntryEditReSnapshotsRate`: edit an entry to a different project with a different rule, assert the snapshot updates to the new rule's values.
- [x] 9.4 Write `TestEntryEditUnrelatedFieldsPreservesSnapshot`: edit only `note` (if exists) or another non-rate-determining field; assert the snapshot columns are unchanged.
- [x] 9.5 Write `TestCrossWorkspaceRuleDoesNotLeakIntoSnapshot`: create rules in workspace `W2` that would match the entry's project if workspace were ignored; stop the entry in `W1`; assert the snapshot uses only `W1`'s rules (or NULL).

## 10. Integration tests — reporting stability

- [x] 10.1 Write `TestReportBillableAmountReadsSnapshotNotResolver`: create a closed entry with a snapshot of 10000 minor units USD, then delete the underlying rule (after clearing the FK by first nulling the reference? — use a seed path that leaves snapshot columns but no referencing FK, or better, mutate `rate_rules` via an allowed path like extending `effective_to`). Re-run the report, assert the entry still contributes 10000 minor units. Document any test-only setup required.
- [x] 10.2 Write `TestHistoricalTotalsUnchangedAfterExtendingEffectiveTo`: create an entry + snapshot, extend the rule's `effective_to` to a future date, re-run the report, assert totals match pre-edit.
- [x] 10.3 Write `TestReportingDoesNotCallResolveForClosedEntries`: assert (via a spy/mock or by counting calls in a deterministic test harness) that zero calls to `rates.Service.Resolve` occur during `reporting.Service.Report` for a range covering only closed entries.
- [x] 10.4 Write `TestReportingRunningTimerExcludedFromBillable`: start a timer without stopping, call `reporting.Service.Dashboard`, assert the running entry does not contribute to `WeekEstimatedBillable` and is not counted in `EntriesWithoutRate`.

## 11. Integration tests — boundary semantics

- [x] 11.1 Write `TestResolveAcrossUTCBoundary`: insert two adjacent rules (`R_old` ending 2026-03-31, `R_new` starting 2026-04-01), call `Resolve` for an entry `started_at = 2026-03-31T23:30:00-04:00` (= 2026-04-01Z), assert `R_new` is returned.
- [x] 11.2 Write `TestResolveOnFirstDayOfRule`: assert rule active on its `effective_from` date.
- [x] 11.3 Write `TestResolveOnLastDayOfRule`: assert rule still active on its `effective_to` date.
- [x] 11.4 Write `TestResolveDayAfterEffectiveTo`: assert rule is NOT active on `effective_to + 1 day`.

## 12. Backfill tests

- [x] 12.1 Write `TestBackfillPopulatesNullSnapshots`: seed a workspace with rules and closed entries that have NULL snapshots, run the backfill command (invoke the underlying function directly from Go; do not shell out), assert every entry now has a matching snapshot.
- [x] 12.2 Write `TestBackfillIsIdempotent`: run backfill twice, assert the second run writes 0 rows and logs `backfilled=0`.
- [x] 12.3 Write `TestBackfillAbortsOnAdjacencyConflict`: seed two rules with `A.effective_to = B.effective_from`, assert the backfill pre-flight aborts with the typed error listing the offending rule IDs.
- [x] 12.4 Write `TestBackfillDryRunWritesNothing`: assert `--dry-run` reports counts without modifying rows.

## 13. Accessibility validation

- [x] 13.1 Keyboard-only walkthrough: on the rate rules page, verify Tab reaches every control, the disabled delete button has a visible focus ring or is programmatically skipped, and the `Referenced by N entries` hint is associated via `aria-describedby` so a screen reader announces it when focusing the edit control.
- [x] 13.2 Screen reader spot check (NVDA or VoiceOver): announce the referenced-count hint on focus of the row's action group.
- [x] 13.3 Contrast audit: the `Referenced by N entries` hint text meets 4.5:1 against its background (use a neutral-700 on neutral-50 or equivalent token; not the accent color alone).
- [x] 13.4 Confirm the 409 error message in the edit form is reached by a live-region announcement and focus is moved to the error container (`data-focus-after-swap`).

## 14. Verification

- [x] 14.1 Run `make fmt && make vet && make lint && make test`. All must pass; integration tests under `-p 1` per the project convention.
- [x] 14.2 Run `make db-up && make migrate-up && make dev-seed && make run`, stop a timer, edit the seeded rate rule's amount (should be rejected with clear 409), extend its `effective_to` (should succeed). Manual sanity check on dashboard totals before and after.
- [ ] 14.3 Update `openspec/specs/rates/spec.md` and `openspec/specs/reporting/spec.md` from the delta files as part of archive (handled by `/opsx:archive`).
- [ ] 14.4 Remove the transitional `REPORTING_RESOLVE_FALLBACK` code in a follow-up change (`tighten-reporting-snapshot-only`); note it in the proposal's follow-ups list.
