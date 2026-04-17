## ADDED Requirements

### Requirement: Exhaustive cross-workspace denial for every reporting handler

Every read handler in the `reporting` family MUST return HTTP 404 with the shared not-found response body when invoked by a user whose active workspace does not own a referenced filter target (client, project, or entry), and MUST scope all aggregations strictly to the caller's active workspace when no specific target is referenced. This rule applies without exception to: dashboard summary, today/week totals, billable totals, and any entries-list filter pages. No reporting response may aggregate or display data from a workspace other than the caller's active workspace.

#### Scenario: Dashboard summary is scoped to active workspace
- **GIVEN** Alice's active workspace is `W1` and entries exist in both `W1` and `W2`
- **WHEN** Alice loads the dashboard
- **THEN** the running-timer widget, today's total, this-week's total, and this-week's billable amount MUST reflect only entries with `workspace_id = W1`
- **AND** no data from `W2` influences any displayed figure

#### Scenario: Report filter by other-workspace project returns 404
- **GIVEN** project `P2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice requests a report filtered by `project_id = P2`
- **THEN** the system MUST respond with HTTP 404
- **AND** no aggregation is performed

#### Scenario: Report filter by other-workspace client returns 404
- **GIVEN** client `C2` belongs to `W2` and Alice is not a member of `W2`
- **WHEN** Alice requests a report filtered by `client_id = C2`
- **THEN** the system MUST respond with HTTP 404

#### Scenario: Entries-list filter is scoped to active workspace
- **GIVEN** Alice's active workspace is `W1`
- **WHEN** Alice requests the entries list with any combination of filters
- **THEN** every returned row MUST have `workspace_id = W1`
- **AND** pagination counts MUST reflect only `W1` entries
