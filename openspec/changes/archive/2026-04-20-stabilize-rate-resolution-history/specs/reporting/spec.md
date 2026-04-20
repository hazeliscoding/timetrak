## MODIFIED Requirements

### Requirement: Estimated billable amount

The system SHALL compute an estimated billable amount per grouping by summing, for each billable time entry in the range, `duration_seconds / 3600 * hourly_rate_minor`, where `hourly_rate_minor` and `currency_code` are read directly from the entry's persisted rate snapshot (written when the entry was stopped or saved — see `rates` capability). Amounts MUST be accumulated as integer minor units and displayed formatted by the entry's snapshot `currency_code`. Entries whose snapshot columns are NULL (no rate resolvable at the time of stop/save) MUST contribute zero to the total and MUST be flagged as `No rate` in the row or in the aggregate `Entries without a rate` count. Reporting MUST NOT invoke `rates.Service.Resolve` on the read path for any closed time entry; the snapshot is the authoritative source for historical figures.

#### Scenario: Amount uses the persisted snapshot

- **GIVEN** a billable entry of 60 minutes whose snapshot is `hourly_rate_minor = 10000`, `currency_code = 'USD'`
- **WHEN** estimated billable amount is computed for that entry
- **THEN** its contribution is 10000 minor units to the `USD` bucket
- **AND** the reporting service MUST NOT call `rates.Service.Resolve` for this entry

#### Scenario: Entry without a snapshot is flagged as No rate

- **GIVEN** a billable entry whose snapshot columns are all NULL
- **WHEN** reports are computed
- **THEN** its billable-amount contribution is 0
- **AND** the UI surfaces a visible `No rate` indicator (text, not color alone) or increments the aggregate `Entries without a rate` count

#### Scenario: Non-billable entries excluded from amount

- **GIVEN** a non-billable entry, with or without a snapshot
- **WHEN** reports are computed
- **THEN** its billable-amount contribution is 0

#### Scenario: Retroactive rule edit does not move historical totals

- **GIVEN** a closed billable entry whose snapshot is `hourly_rate_minor = 10000`, `currency_code = 'USD'`, referencing rule `R1`
- **AND** an administrator subsequently performs any permitted edit to `R1` (e.g., extending `effective_to` into the future)
- **WHEN** the weekly, monthly, or ad-hoc report over the entry's date is recomputed
- **THEN** the entry's contribution MUST remain 10000 minor units in `USD`
- **AND** the total for the containing group MUST be identical to the total before the edit

#### Scenario: Running entries are excluded from billable totals

- **GIVEN** a running entry (`ended_at IS NULL`) with no snapshot
- **WHEN** reports or dashboard widgets are computed
- **THEN** the running entry MUST NOT contribute to any billable-amount total
- **AND** the entry MUST NOT be counted in the `Entries without a rate` aggregate
