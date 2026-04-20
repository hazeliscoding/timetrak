## ADDED Requirements

### Requirement: Workspace reporting timezone

Each workspace SHALL carry a `reporting_timezone` attribute: a non-empty IANA timezone name (e.g., `UTC`, `America/New_York`, `Europe/Berlin`). The attribute SHALL default to `'UTC'` for new workspaces and for all existing workspaces at migration time. The reporting domain SHALL consume this attribute to bucket entries by local calendar day; no other behavior in the workspace domain itself depends on it.

#### Scenario: Default value on new workspace

- **WHEN** a workspace is created without specifying a timezone
- **THEN** its `reporting_timezone` SHALL equal `'UTC'`

#### Scenario: Backfill preserves existing behavior

- **GIVEN** workspaces existed before this change
- **WHEN** the migration runs
- **THEN** every pre-existing workspace SHALL have `reporting_timezone = 'UTC'`
- **AND** report output for those workspaces SHALL be unchanged relative to the pre-change UTC baseline

### Requirement: Reporting timezone settings

The system SHALL provide a workspace-scoped settings control that lets an authorized user change the workspace `reporting_timezone` to any value in Postgres's `pg_timezone_names`. Submitting an unrecognized or empty value SHALL be rejected with a form-level error and SHALL NOT persist. A change SHALL take effect on the next report request with no data migration.

#### Scenario: Valid timezone persists

- **GIVEN** Alice is on the workspace settings page
- **WHEN** she selects `America/New_York` and submits
- **THEN** the workspace's `reporting_timezone` SHALL be updated
- **AND** the next `/reports` request SHALL bucket entries by `America/New_York` local date

#### Scenario: Invalid timezone is rejected

- **WHEN** the settings endpoint receives a tz name not present in `pg_timezone_names`
- **THEN** the system SHALL respond with a validation error
- **AND** the stored `reporting_timezone` SHALL be unchanged

#### Scenario: Setting change is workspace-scoped

- **GIVEN** Alice is a member of workspaces `W1` and `W2`
- **WHEN** she changes `W1`'s `reporting_timezone`
- **THEN** `W2`'s `reporting_timezone` SHALL be unaffected
- **AND** any attempt to mutate `W2`'s timezone without membership SHALL return HTTP 404
