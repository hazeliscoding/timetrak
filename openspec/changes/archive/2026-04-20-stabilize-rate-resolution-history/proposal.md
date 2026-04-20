## Why

TimeTrak's rate resolver already accepts a point-in-time `at` argument and reporting already passes each entry's `started_at` to it — so on paper, historical rates are respected. In practice there are still three correctness hazards that Stage 2 must close before invoicing (Stage 3) can be built on top:

1. **Retroactive mutation of history.** `rate_rules` rows can be edited or deleted freely. Because reports resolve rates on every read, any edit to an old rule silently rewrites past billable totals. Nothing pins the rule that was in force when an entry was tracked.
2. **Ambiguous same-day boundaries.** Windows are compared as dates against a `timestamptz` started_at truncated to UTC midnight. The overlap check allows `effective_to = next.effective_from`, and Resolve's inclusive-both-ends window means both rules are "active at" the boundary date — the tie-breaker (`ORDER BY effective_from DESC`) is implicit and untested.
3. **No historical-correctness test.** There is no integration test that asserts "changing a rule after the entry was recorded does not alter that entry's estimated billable amount." Without it, future refactors can regress history silently.

Closing these gaps is a Stage 2 (Stabilize) concern: we are hardening existing behavior, not expanding the domain model.

## What Changes

- **Snapshot the resolved rate onto the time entry at stop/save time.** Persist `rate_rule_id`, `hourly_rate_minor`, and `currency_code` on `time_entries` as a historical snapshot. Reports read the snapshot instead of re-resolving on every read.
- **Make historical rules effectively immutable.** Rules whose window has started and that are referenced by at least one `time_entries.rate_rule_id` MAY NOT be deleted, and their `effective_from`, `effective_to`, `hourly_rate_minor`, `currency_code`, `client_id`, or `project_id` MAY NOT be edited in a way that changes what past entries saw. Edits that only extend an open-ended `effective_to` into the future remain allowed.
- **Tighten boundary semantics in `Resolve`.** Define `effective_to` as exclusive-end-of-day in UTC, adjacency (`old.effective_to = new.effective_from`) as non-overlap, and the tie-breaker at boundaries as "latest `effective_from` wins" — and cover all three in spec scenarios.
- **Backfill snapshots for existing entries.** The migration that adds the snapshot columns runs a one-shot backfill that calls the same resolver against each existing entry's `started_at`. Entries with no resolvable rate are left with NULLs and counted in the existing `No rate` bucket. Running entries (`ended_at IS NULL`) are snapshotted on stop, never retroactively.
- **Change the reporting read path.** `reporting.Service` reads `hourly_rate_minor` / `currency_code` / `rate_rule_id` directly from `time_entries` for closed entries. `rates.Service.Resolve` is still the single source of truth — it is invoked exactly once per entry, at stop/save time, not on every report render.
- **Add integration tests** that prove (a) a rule edit after-the-fact does not change a historical entry's billable amount; (b) same-day boundary between an old and new rule is resolved deterministically; (c) cross-workspace rules never leak into another workspace's snapshots.

## Capabilities

### New Capabilities
_None._ This change stabilizes two existing capabilities; it does not introduce a new domain.

### Modified Capabilities
- `rates`: add immutability rules for referenced rate_rules, tighten the boundary semantics of `Resolve`, and formalize the snapshot contract produced at stop/save time.
- `reporting`: require reports to read persisted rate snapshots from `time_entries` for closed entries rather than re-resolving on each render, while preserving the existing `No rate` surface for entries without a snapshot.

## Impact

- **Database**
  - New migration pair adding `rate_rule_id uuid NULL`, `hourly_rate_minor bigint NULL`, `currency_code char(3) NULL` to `time_entries`, with a FK to `rate_rules(id) ON DELETE RESTRICT`.
  - A one-shot backfill (idempotent, re-runnable) that populates the snapshot for entries where `ended_at IS NOT NULL AND rate_rule_id IS NULL`.
  - No change to `rate_rules` shape; only new constraints on edit/delete enforced in the service.

- **Backend**
  - `tracking` domain: on timer stop and on entry save, call `rates.Service.Resolve(ctx, workspaceID, projectID, started_at)` and persist the snapshot columns in the same transaction that closes/edits the entry.
  - `rates` domain: `Update` and `Delete` gain a reference-check that rejects mutation with a new typed error (`ErrRuleReferenced`) when any `time_entries.rate_rule_id` points at the rule and the mutation would alter the historical view.
  - `reporting` domain: `estimateBillable`, `estimateByClient`, `estimateByProject` switch to reading `hourly_rate_minor` / `currency_code` from `time_entries` and accumulate directly. The `rates` service remains injected only for future per-entry-preview use cases (e.g., UI hint on the edit form).

- **Templates / HTMX**
  - Entry edit and list partials: `No rate` surface remains unchanged (it already reads from the entry + resolver result; now it reads from the entry snapshot columns instead). No visible change for the user on existing flows.
  - Rate rules list/edit: surface a non-destructive hint ("Referenced by N entries — historical edits disabled") when a rule is immutable. Must use text + icon, never color alone (WCAG 2.2 AA).

- **Out of scope (explicit)**
  - Invoicing: this change does _not_ introduce invoicing, line-item locking, or PDF generation. It only makes the billable figures stable enough that invoicing can be built on top in Stage 3.
  - Multi-currency FX conversion for totals across currencies. We continue to emit a per-currency map.
  - Cursor-based pagination for entries. Still offset-based.
  - CSV export of reports. Separate Stage 2 change.
  - User-facing UI for migrating a rule's historical window (e.g. "split this rule at 2026-06-01"). If it is needed it will be a follow-up change.

- **Assumptions**
  - All `time_entries.started_at` are `timestamptz` and can be safely interpreted against a UTC-date window. Local-time billing boundaries are a Stage 3 concern.
  - Every workspace has at most a few hundred rate rules and a few hundred thousand entries; the backfill can run in a single transaction per workspace during the migration window. A batched-backfill tool is not needed at MVP-adjacent scale.
  - `rates.Service.Resolve` is currently correct for single-point-in-time queries; we are hardening its contract, not rewriting its logic.

- **Risks**
  - **Backfill resolves to a different rate than a human operator expects** for a legacy entry (e.g. because a rule was edited after the fact). Mitigation: emit a structured log line per backfilled entry and a summary (`backfilled=N, no_rate=N`) so operators can audit before marking the change archived.
  - **Admins complain that historical rules are "stuck"** once referenced. Mitigation: the typed error message is specific, the UI surfaces the reason, and a follow-up change can add a "supersede and close" workflow (create a new rule that takes effect _from today_, leaving history intact).
  - **Double-source-of-truth drift**: snapshot on the entry vs. live resolution from rules. Mitigation: Resolve is invoked exactly once, at the transaction that creates or closes the entry; there is no resynchronization path, and any UI preview that shows "current rate" is labeled as such rather than mixed into historical reports.

- **Likely follow-up changes**
  - `add-rate-rule-supersede-flow`: UX for closing an old rule on a date and starting a new one on the next day, with live preview of which future entries will be affected.
  - `add-csv-export` in reporting (independent).
  - Invoicing capability (Stage 3), which can now rely on stable historical billable amounts.
