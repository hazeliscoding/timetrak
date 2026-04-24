## Why

The time-entry edit form is the last datetime-input UX that still demands raw ISO 8601 UTC strings from the user. At `web/templates/partials/entry_row.html:21-25` the start/end inputs are `<input type="text">` with the value prefilled as `2026-04-24T09:15:30Z` — a format humans cannot read at a glance, cannot type correctly without referencing the current value, and that implicitly asks the user to do mental timezone math when their workspace reports in `America/New_York`. Every other datetime surface in the product is already humane: manual-entry create uses `type="date"` + `type="time"` (`web/templates/time/index.html:53-62`), rate rules use `type="date"` (`web/templates/partials/rate_form.html:50-55`), and filter bars use `type="date"` (time, reports). The edit form is the odd one out, and it is also the most-used path — every correction to a stopped timer goes through it.

A second, more subtle problem hides inside the already-humane manual-create path: the server parses `date + start_time + end_time` via `time.Parse("2006-01-02T15:04", ...)` (`internal/tracking/handler.go:544`), which produces a `time.Time` in **UTC regardless of the workspace's `ReportingTimezone`**. A freelancer in `America/Los_Angeles` who records "Monday 9–10am" today has that entry stored as `09:00–10:00 UTC` (01:00–02:00 local), skewing every subsequent report by 8 hours. The workspace's reporting timezone is already persisted (shipped by the earlier reporting change; validated in `internal/workspace/timezone_test.go`) — the tracking parse path just never consumed it.

This change makes both paths honestly TZ-aware and replaces the last raw-ISO text input with date+time pair inputs that behave consistently with the rest of the product.

## What Changes

- **Entry-edit form** (`web/templates/partials/entry_row.html`): replace the two `type="text"` ISO inputs (`started_at`, `ended_at`) with four native inputs — `start_date` + `start_time` + `end_date` + `end_time` — each `type="date"` or `type="time"`. Prefilled values are formatted in the workspace's reporting timezone, not UTC. The existing per-entry row contract (`id="entry-row-<uuid>"`, HTMX swap target) is preserved.
- **Tracking handler parse** (`internal/tracking/handler.go`): `updateEntry` switches from `time.Parse(time.RFC3339, ...)` over `started_at`/`ended_at` to a TZ-aware parse over `start_date`+`start_time` / `end_date`+`end_time` using the workspace's `ReportingTimezone`. `createManual` switches from `time.Parse("2006-01-02T15:04", ...)` (UTC) to the same TZ-aware parse — fixing the pre-existing bug whereby manual entries ignored the workspace tz.
- **Shared helper** (`internal/shared/clock` or a new `internal/shared/datetime`): introduce a small `ParseLocalDateTime(date, timeStr, tzName string) (time.Time, error)` that loads the named `*time.Location`, parses the two strings together, and returns a `time.Time` in UTC for storage. Used by both tracking paths so the parse contract is single-sourced.
- **Template helper**: add `formatLocalDate(t time.Time, tz string) string` and `formatLocalTime(t time.Time, tz string) string` — used by the entry-edit form to prefill the date/time pair in the workspace tz, and cited in the partials README.
- **Validation & error surfacing**: an invalid date or time string, or an impossible tz (shouldn't happen post-workspace-settings-validation, but defensive), returns `tracking.invalid_interval` via the existing `tracking_error` partial path. The error taxonomy does not grow.
- **Out of scope (explicit):**
  - Natural-language input ("today 9am", "2h ago", "yesterday 15:00"). That is a real follow-up but a much bigger design space (parser, testability, a11y for assistive tech). Separate proposal if/when demand surfaces.
  - The running-timer's `data-timer-started-at` attribute (`web/templates/partials/timer_control.html:11`). It is consumed by `web/static/js/app.js` for the live elapsed clock, not by humans; ISO 8601 Z is correct there.
  - Rate-rule effective dates, reports filter bar, manual-create form structure — all already use `type="date"` / `type="time"` and do not need humanizing.
  - Any DB migration. `started_at` and `ended_at` remain `timestamptz` in UTC. Only the input/parse boundary changes.
  - Workspace timezone UX, picker widget, or tz validation — all shipped by the prior reporting change. This change only *consumes* the accepted `ReportingTimezone` field.

## Capabilities

### New Capabilities

- *None.* The scope lives entirely inside `tracking` (input contract + parse behavior) and touches `ui-partials` (the entry-edit form is a partial). No new capability warranted for a TZ-aware input sweep on one handler pair.

### Modified Capabilities

- `tracking` — ADDS a requirement that datetime inputs on the entry-edit and manual-create paths are split `date + time` pairs parsed in the workspace's `ReportingTimezone`, and MODIFIES (if a matching requirement exists, otherwise ADDS) the interval-validation requirement to operate post-tz-conversion rather than on raw RFC3339 strings. The existing active-timer invariant, cross-workspace 404 contract, and rate-snapshot behavior are untouched.
- `ui-partials` — MODIFIES the `entry_row` partial's documented context contract (if any fields are enumerated in the spec) to reflect the split inputs; otherwise this is a purely markup-level change and no `ui-partials` delta is needed. Auditing required before writing the delta.

## Impact

- **Templates modified:** `web/templates/partials/entry_row.html` (edit-mode form layout), possibly a small helper partial if the date+time pair is reused on the manual-create form.
- **Go code modified:** `internal/tracking/handler.go` (`updateEntry`, `createManual`, and the view struct / form-error plumbing for the new field names). Possibly `internal/tracking/service.go` only if a signature changes (should not — service already takes `time.Time`).
- **New Go file:** `internal/shared/datetime/parse.go` (or an extension to `internal/shared/clock`) exposing `ParseLocalDateTime`. Colocated unit tests.
- **Template funcs:** extend `internal/shared/templates` to register `formatLocalDate` / `formatLocalTime` alongside the existing `formatDate` / `formatTime` helpers.
- **Tests:** unit tests for the new parse helper (happy path + DST boundary + invalid tz + invalid date + invalid time + mismatch where end precedes start); handler-level integration test updates for `updateEntry` + `createManual` reflecting the new field names and the tz-aware parse; add an `internal/tracking/timezone_test.go` covering the "Los Angeles user records 9am Monday → stored as 16:00 UTC → reads back as 9am local" round-trip that today silently fails.
- **Specs:** delta under `openspec/changes/humanize-datetime-inputs/specs/tracking/spec.md`; optional `ui-partials` delta if the partial's documented context contract mentions input field names.
- **No DB, no migration, no new dependency.** No new route, no new handler.
- **Risk:** the tz-aware parse will cause a behavioral shift for existing users whose workspaces have a non-UTC `ReportingTimezone` AND who have previously edited entries under the old raw-ISO form. The shift is *correct* (their inputs now match their local clock), but it will look like "my old entries moved". Mitigated by: (a) display-side conversion also moves to workspace tz so the edit form re-reads consistently; (b) stored data in `timestamptz` is unchanged, so historical totals are stable; (c) documented in the tasks.md verification section.
- **Assumptions:**
  1. Workspace `ReportingTimezone` is reliably populated (default `UTC`) on every accessible workspace — validated by the prior reporting change's spec and tests.
  2. The browser's native `type="date"` / `type="time"` widgets are acceptable UX. They are, per every other humane datetime surface already shipped; we do not introduce a custom widget.
- **Follow-ups (not part of this change):** natural-language input ("today 9am"); a shared `form_datetime_pair` partial if the date+time layout needs sharper reuse; a timezone-display cue on the edit form ("Times shown in America/New_York") if user testing shows the conversion is surprising.
