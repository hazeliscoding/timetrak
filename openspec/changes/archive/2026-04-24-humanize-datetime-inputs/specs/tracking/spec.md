## ADDED Requirements

### Requirement: Datetime input parse and display is workspace-timezone-aware

The tracking HTTP handlers that accept start/end datetimes (`POST /time-entries` for manual create and `PATCH /time-entries/{id}` for edit) SHALL accept split `date + time` inputs — one `type="date"` input yielding `YYYY-MM-DD` and one `type="time"` input yielding `HH:MM` — and SHALL parse each pair in the active workspace's `ReportingTimezone`. The resulting `time.Time` SHALL be stored in UTC; persistence in `timestamptz` is unchanged. Raw ISO 8601 strings (e.g. `2026-04-24T09:15:30Z`) submitted in a single text field are NOT an accepted input format on these surfaces.

Display of persisted `started_at` / `ended_at` values in the entry-edit form SHALL convert from UTC to the workspace's `ReportingTimezone` before splitting into the prefilled `date` + `time` values, so the form reads back consistently with its write path.

Invalid date strings, invalid time strings, intervals where end precedes or equals start, and any other parse failure SHALL surface via the existing `tracking_error` partial path and the existing `tracking.invalid_interval` / handler-validation taxonomy; no new error code is introduced.

#### Scenario: Entry edit prefills date + time in the workspace timezone

- **GIVEN** workspace `W1` has `ReportingTimezone = "America/New_York"` and entry `TE1` is stored with `started_at = 2026-04-24T14:00:00Z`
- **WHEN** a member of `W1` opens the entry-edit form for `TE1`
- **THEN** the form's `start_date` input value MUST be `2026-04-24` and the `start_time` input value MUST be `10:00` (the New York local equivalent of the stored UTC)
- **AND** the form MUST NOT render a raw ISO 8601 string in any field

#### Scenario: Entry edit parses date + time as workspace-local

- **GIVEN** workspace `W1` has `ReportingTimezone = "America/New_York"`
- **WHEN** a member of `W1` submits the edit form with `start_date = 2026-04-24` and `start_time = 10:00`
- **THEN** the handler MUST compute the `started_at` as `2026-04-24T10:00:00-04:00` and store it as `2026-04-24T14:00:00Z`
- **AND** subsequent reads of the entry MUST display the same `10:00` in the edit form under `ReportingTimezone = "America/New_York"`

#### Scenario: Manual entry create parses date + time as workspace-local

- **GIVEN** workspace `W1` has `ReportingTimezone = "America/Los_Angeles"`
- **WHEN** a member of `W1` submits a manual entry with `date = 2026-04-24`, `start_time = 09:00`, `end_time = 10:00`
- **THEN** the stored `started_at` MUST equal `2026-04-24T16:00:00Z` (the Los Angeles 09:00 local converted to UTC)
- **AND** the stored `ended_at` MUST equal `2026-04-24T17:00:00Z`
- **AND** `duration_seconds` MUST equal `3600`

#### Scenario: Default UTC workspace produces stable values with raw clock inputs

- **GIVEN** workspace `W1` has `ReportingTimezone = "UTC"` (the default)
- **WHEN** a member of `W1` submits a manual entry with `date = 2026-04-24`, `start_time = 09:00`, `end_time = 10:00`
- **THEN** the stored `started_at` MUST equal `2026-04-24T09:00:00Z`
- **AND** the stored `ended_at` MUST equal `2026-04-24T10:00:00Z`

#### Scenario: Invalid date or time string is rejected with the existing error surface

- **WHEN** a member submits the edit or manual-create form with a malformed `date` (e.g. `"not-a-date"`) or `time` (e.g. `"25:99"`) value
- **THEN** the handler MUST respond with HTTP 422
- **AND** the response MUST render the existing `tracking_error` partial with a clear per-field error
- **AND** no `time_entries` row MUST be created or updated

#### Scenario: End precedes start after timezone conversion

- **GIVEN** workspace `W1` has `ReportingTimezone = "America/New_York"`
- **WHEN** a member of `W1` submits a manual entry with `date = 2026-04-24`, `start_time = 10:00`, `end_time = 09:00`
- **THEN** the handler MUST respond with HTTP 422 and `tracking.invalid_interval`
- **AND** the validation MUST operate on the TZ-converted UTC values (not on the raw clock-time strings), so the same rule applies across every supported tz

#### Scenario: Raw ISO 8601 text fields are not accepted on edit

- **GIVEN** the entry-edit form surface as specified
- **WHEN** a legacy client submits `started_at=2026-04-24T14:00:00Z` in place of the `start_date` + `start_time` pair
- **THEN** the handler MUST respond with HTTP 422 (missing required `start_date` / `start_time` fields)
- **AND** the response MUST NOT silently fall back to parsing a single-field ISO input
