## Context

Two handlers own the tracking write path: `createManual` (POST `/time-entries`) and `updateEntry` (PATCH `/time-entries/{id}`). The former already takes a split `date + start_time + end_time` form, parsed via `time.Parse("2006-01-02T15:04", ...)` which produces a `time.Time` in UTC unconditionally. The latter takes two raw RFC 3339 strings (`started_at`, `ended_at`) and parses them via `time.Parse(time.RFC3339, ...)`, which respects the offset in the string.

The workspace has a persisted `ReportingTimezone` field (`internal/workspace/timezone_test.go` covers round-trip, invalid-tz rejection, and the default `UTC`). Reporting already consumes it for day/week bucketing. Tracking never has.

The user-visible consequence: the entry-edit form is the last raw-ISO input in the product, and a freelancer in `America/Los_Angeles` silently stores manual entries 7–8 hours shifted from local clock. Both problems share one fix: tz-aware parse + tz-aware display on both handler paths.

## Goals / Non-Goals

**Goals:**

- Replace the entry-edit form's two raw-ISO text inputs with four native `type="date"` + `type="time"` inputs — the same pattern already shipped on manual-create, rate rules, and filter bars.
- Parse date + time pairs in the workspace's `ReportingTimezone` on both `updateEntry` and `createManual`. Store UTC. Display back by converting UTC → workspace tz.
- Single-source the parse logic so the two handlers cannot drift.
- Preserve every accepted service-level invariant (active-timer, interval, cross-workspace, composite FK, rate-snapshot).

**Non-Goals:**

- Natural-language input. Separate proposal if a real need emerges; would need a parser, accessibility story, and testing strategy all its own.
- Custom datetime widgets, date-range pickers, or JS-driven inputs. The native browser `type="date"` / `type="time"` are already the product's accepted pattern.
- DST-aware business rules (e.g. "always schedule 9am local across the spring-forward boundary"). The parse contract is "interpret this clock time in this zone"; that covers both DST sides honestly but does not promise duration invariance across a transition.
- Changing the running-timer live clock (`data-timer-started-at`). That attribute is consumed by JS, not humans; ISO 8601 Z is correct.
- Workspace tz picker UX — shipped earlier, not touched here.

## Decisions

### D1. One shared `ParseLocalDateTime` helper, two consumers

**Chosen:** A new package `internal/shared/datetime` (or a new file in `internal/shared/clock`) exposes:

```go
// ParseLocalDateTime parses a YYYY-MM-DD date string and a HH:MM time string
// as a clock time in the named IANA timezone, returning the equivalent
// time.Time in UTC.
//
// Both inputs MUST be non-empty; empty strings yield a wrapped error with
// a named field so the handler can emit a per-field validation message.
//
// The tz MUST be an IANA zone name known to the runtime's tzdata; UTC is
// always accepted.
func ParseLocalDateTime(date, timeStr, tz string) (time.Time, error)
```

**Why not inline in each handler:** Two handlers parsing "the same thing" independently is exactly how `createManual` drifted from `updateEntry` in the first place (one in UTC, one in RFC3339). A single helper with one parse contract and one test surface is the durable fix.

**Why `internal/shared/datetime` vs extending `internal/shared/clock`:** `clock` today is `Clock` interface + `System{}` implementation — a pure now-abstraction. Dropping parse logic there mixes concerns. A small new package is clearer; alternately a `clock.Parse...` helper is acceptable. Final call during implementation; no architecture impact.

### D2. Display helpers are template funcs, not inline logic

**Chosen:** Add two template funcs to `internal/shared/templates`: `formatLocalDate(t time.Time, tz string) string` returns `YYYY-MM-DD` and `formatLocalTime(t time.Time, tz string) string` returns `HH:MM`, both in the named tz. The entry-edit form invokes them with the active workspace's `ReportingTimezone` passed through the view struct.

**Why not compute in the handler and pass formatted strings:** Templates already have `formatDate`, `formatTime`, `iso` — adding `formatLocalDate` / `formatLocalTime` keeps the family consistent and lets the form template read intention-revealingly. Handler stays focused on service calls.

### D3. View struct carries the workspace tz explicitly

**Chosen:** The entry-row view struct (passed to `entry_row.html` edit mode) gains a `Timezone string` field populated from `wc.WorkspaceID` → workspace lookup's `ReportingTimezone`. The template uses it with the helpers above. Similarly the manual-create form template reads the timezone from the page view.

**Alternative considered:** Use a template global via `layout.BaseView`. Rejected — not every page needs it, and the explicit field makes the data flow obvious at review time.

### D4. Parse validates inputs in the named tz, then re-validates interval on UTC values

**Chosen:** `updateEntry` and `createManual` compute `startedAt, err := ParseLocalDateTime(...)` then `endedAt, err := ParseLocalDateTime(...)`, then pass the resulting UTC `time.Time` values into the existing service methods. Interval validation (`ended > started`) happens inside the service, on UTC values — so a user who types `10:00` start and `09:00` end always gets `tracking.invalid_interval` regardless of tz.

**DST consequence:** on the spring-forward boundary, `02:30` does not exist in `America/New_York`. `time.ParseInLocation` treats it as the post-spring equivalent (03:30 local / 07:30 UTC). On fall-back, `01:30` is ambiguous; the Go stdlib consistently returns the first occurrence. Both behaviors are documented as known and deliberate; no bespoke DST handling is warranted for a time-tracking app.

### D5. Field-name migration is a hard break, not a compatibility shim

**Chosen:** The edit form's submitted fields become `start_date`, `start_time`, `end_date`, `end_time`. The handler does NOT accept the old `started_at` / `ended_at` fields as a fallback. Any legacy client (there are none — the HTMX forms submit whatever the current template emits) gets HTTP 422 with a missing-field error.

**Why:** Keeping a shim means maintaining two parse paths forever and documenting which one wins when both arrive. The handler is server-rendered with server-templated forms; the template and handler ship together. A shim is avoiding a problem that cannot actually happen.

### D6. `updateEntry` handler reads the workspace's ReportingTimezone at request time

**Chosen:** `updateEntry` already resolves `wc := authz.MustFromContext(r.Context())`. Extend to call `wsSvc.GetReportingTimezone(ctx, wc.WorkspaceID)` (or whatever the workspace service exposes) and pass it to `ParseLocalDateTime`. If the lookup fails, default to `"UTC"` and log a structured warning — the user's edit should not 500 because of a tz lookup glitch.

**Alternative considered:** Cache tz at session start. Rejected — adds cache-invalidation complexity for a ~sub-millisecond Postgres read; not worth it.

### D7. No new partial for the date+time pair

**Chosen:** Inline the four input fields in `entry_row.html`'s edit branch. They are co-located and identical to the pattern already inline on `time/index.html`'s manual-create form.

**Alternative considered:** Extract a `partials/datetime_pair` partial. Rejected as premature — two consumers (entry edit + manual create) with slight label-context differences; the shared extraction bar per `partials/README.md` is ≥2 consumers with genuinely identical context, and these differ enough that a partial would carry noisy optional slots.

## Risks / Trade-offs

- **[Risk]** A freelancer with a non-UTC workspace suddenly sees existing entries "shift" in the edit form (from UTC clock to local clock). The underlying storage is unchanged and reports already bucket in workspace tz, so the total hours remain identical — but the edit-form readout changes. → *Mitigation:* the tasks.md verification step explicitly exercises this round-trip; documentation cue is a small muted hint line on the edit form ("Times in <tz>") if reviewer testing shows the shift is surprising.
- **[Risk]** DST-boundary ambiguity (fall-back 01:30 exists twice, spring-forward 02:30 does not exist). → *Mitigation:* stdlib `time.ParseInLocation` behavior documented as accepted; out-of-scope to build explicit disambiguation UX for a single tracking form.
- **[Risk]** Timezone lookup failure at write time (extremely unlikely — workspace tz is a validated field) could 500 the edit. → *Mitigation:* D6 defaults to UTC + logs a structured warning rather than failing the user's write.
- **[Trade-off]** Two more template funcs (`formatLocalDate`, `formatLocalTime`) on the registry. Low cost, consistent naming with the existing family.
- **[Trade-off]** Field-name break on the edit handler. Acceptable because the submitting client is the template we ship with it; there is no external consumer.

## Migration Plan

- Ship handler, template, helper, and template-func changes together. No feature flag: the template change and the handler change must land atomically (D5).
- No data migration. Existing `timestamptz` columns are unchanged.
- Rollback: revert the commit. Storage is stable; only the input/display boundary changes.

## Open Questions

- Whether to render a small muted hint ("Times shown in America/New_York") below the edit form's date+time fields when the workspace's tz is non-UTC. Decide during implementation based on how the form reads in the `/dev/showcase` components entry for `entry_row`; the hint is a 2-line template addition if warranted.
- Whether `ParseLocalDateTime` lives at `internal/shared/datetime/` or as an additional file under `internal/shared/clock/`. Cosmetic; decide during implementation. No spec impact either way.
