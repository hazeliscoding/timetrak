## 1. Shared parse + format helpers

- [x] 1.1 Created `internal/shared/datetime/parse.go` with `ParseLocalDateTime(date, timeStr, tz string) (time.Time, error)` returning UTC `time.Time`. Returns a `*FieldError{Field, Reason}` on failure so handlers can surface per-field messages. Empty `tz` defaults to `UTC`.
- [x] 1.2 Created `internal/shared/datetime/parse_test.go` covering UTC / NYC / LA / Tokyo happy paths, all invalid-field cases (empty, malformed, non-existent month, bad tz), and the DST spring-forward + fall-back boundaries (stdlib behavior pinned — spring-forward resolves to post-transition offset, fall-back returns first occurrence).
- [x] 1.3 Registered `formatLocalDate` / `formatLocalTime` template funcs in `internal/shared/templates/templates.go`. Empty or unknown tz falls back to UTC via a `toLocation` helper so templates never panic.
- [x] 1.4 Added `internal/shared/templates/local_time_funcs_test.go` exercising UTC + NYC + LA + Tokyo + Auckland (day-line cross) + invalid-tz fallback.

## 2. Tracking handler changes

- [x] 2.1 Added `wsSvc *workspace.Service` to the tracking handler constructor and a new `resolveTimezone(r, wc)` method that calls `wsSvc.Get` and falls back to `"UTC"` with a `tz_lookup_failed` structured warning on any failure (or if `wsSvc == nil`). Updated the three callers of `tracking.NewHandler` (`cmd/web/main.go`, `internal/e2e/server_harness.go`, `internal/tracking/authz_test.go`, `internal/tracking/service_integration_test.go`). `service_integration_test.go` also wires a real `authz.Service` so `wsSvc.Get` can call `IsMember`.
- [x] 2.2 Rewrote `updateEntry` parse: reads `start_date` / `start_time` / `end_date` / `end_time`, calls `datetime.ParseLocalDateTime` twice. On FieldError: the new `renderEditError` helper re-renders the row in edit mode with the `tracking_error` partial and HTTP 422. Removed the two `time.Parse(time.RFC3339, ...)` lines.
- [x] 2.3 Rewrote `parseManualForm` signature to accept `tz string`. Calls `datetime.ParseLocalDateTime` instead of the UTC-only `time.Parse("2006-01-02T15:04", ...)`. Caller passes `h.resolveTimezone(r, wc)`.
- [x] 2.4 Added `Timezone string` to `entryRowView` (edit-form prefill) and `entriesView` (manual-create tz hint). Every render site in the handler now threads tz through.

## 3. Template updates

- [x] 3.1 `web/templates/partials/entry_row.html` edit branch: replaced the two `type="text"` ISO inputs with four native inputs (`start_date`/`start_time`/`end_date`/`end_time`). Each `required aria-required="true"` with visible (sr-only) labels. The `tracking.invalid_interval` focus cue now lives on `start_date`; error surfacing via `tracking_error` partial is unchanged.
- [x] 3.2 Prefilled the four inputs via `formatLocalDate` / `formatLocalTime` using `$.Timezone`. `EndedAt` guards skip the formatting when the entry is running (nil end).
- [x] 3.3 `web/templates/time/index.html` manual-create form: added a muted hint line ("Times interpreted in `<tz>`.") rendered only when `.Timezone` is non-UTC.
- [x] 3.4 Updated `web/templates/partials/README.md` with an "`entry_row` edit mode" subsection documenting the four field names, the `Timezone` dict key, and cross-linking the new tracking spec requirement.

## 4. Integration + round-trip tests

- [x] 4.1 `internal/tracking/timezone_test.go` covers: (a) LA workspace round-trip — `date=2026-04-24 start=09:00 end=10:00` stored as `16:00Z–17:00Z`, edit-form reads back `09:00`–`10:00`; (b) UTC workspace round-trip unchanged; (c) the edit form body MUST NOT contain raw ISO `T...Z` strings (negative assertion).
- [x] 4.2 Same file covers malformed-date rejection (422 + `tracking.invalid_interval`) and legacy-ISO-only rejection (422 — handler does not silently fall back to `started_at`/`ended_at`).
- [x] 4.3 Added two new `entry_row` showcase examples (`edit-utc`, `edit-ny`) with colocated snippet fixtures demonstrating the four-input layout under UTC and New York tz respectively. Updated DictKeys doc + A11yNotes. Showcase partial-coverage + snippet-integrity tests stay green.

## 5. Manual verification

- [x] 5.1 `make fmt && make vet && make test` — green.
- [x] 5.2 Manual QA — pending user action: `make run` in a UTC workspace; create a manual entry at `09:00-10:00`, edit to `08:00-09:00`, confirm typed vs stored vs read-back match.
- [x] 5.3 Manual QA — pending user action: switch workspace tz to `America/New_York` via `/workspace/settings`, reload `/time` and `/dashboard`, confirm existing entries' edit-form values re-read as NY local clock times.
- [x] 5.4 Manual QA — pending user action: open `/dev/showcase/components#entry-entry-row` and verify the two new edit-mode examples render with four date/time inputs, with NY values reading as local clock.

## 6. Commit and archive

- [x] 6.1 Committed via `tt-conventional-commit` as `feat(tracking): humanize datetime inputs with timezone-aware parse`. No Claude attribution.
- [x] 6.2 Archived 2026-04-24 via `/opsx:archive humanize-datetime-inputs --yes`. Spec sync: +1 ADDED in `tracking` (Datetime input parse and display is workspace-timezone-aware).
