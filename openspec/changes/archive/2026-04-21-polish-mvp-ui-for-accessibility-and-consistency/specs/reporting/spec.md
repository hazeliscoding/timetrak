## ADDED Requirements

### Requirement: Report filter bar SHALL meet WCAG 2.2 AA accessibility

The report filter bar SHALL:

- Pair every filter control (date range, client, project, billable toggle) with a visible `<label>`.
- Group related controls inside a `<fieldset>` with a `<legend>` where the grouping is semantic (e.g. date-range start/end).
- Be fully operable by keyboard — every control reachable via `Tab`, submit via `Enter`, clear/reset via a visible button.
- After submitting filters via HTMX, move focus to the results `<table>` via `data-focus-after-swap` on an element with `tabindex="-1"`.

#### Scenario: Submitting filters moves focus to results

- **GIVEN** a user applies a date range and submits
- **WHEN** the results partial is swapped into the target
- **THEN** focus MUST land on the results table (or its heading) via `data-focus-after-swap`

#### Scenario: Every filter control has a visible label

- **GIVEN** the reports page renders
- **WHEN** the filter bar is inspected
- **THEN** every interactive control MUST have a visible `<label>`
- **AND** date-range start/end MUST be grouped inside a `<fieldset>` with a `<legend>`

### Requirement: Report results table SHALL present accessible semantics

The results table SHALL include a `<caption>` describing the filter applied (e.g. "Billable hours by project — Apr 1 to Apr 14"), `<th scope="col">` on header cells, right-aligned numeric columns via a CSS utility class with `font-variant-numeric: tabular-nums`, and — if any column is sortable — the currently-sorted column SHALL expose `aria-sort="ascending"` or `aria-sort="descending"` and sortable headers SHALL be buttons inside their `<th>`.

#### Scenario: Caption reflects applied filter

- **GIVEN** a user applies a filter and submits
- **WHEN** the results render
- **THEN** the table `<caption>` MUST summarize the filter in human-readable text

#### Scenario: Numeric columns are right-aligned via utility class

- **GIVEN** results contain hours and money columns
- **WHEN** the table renders
- **THEN** those columns MUST be right-aligned via a CSS utility class
- **AND** MUST NOT use inline `style` attributes
- **AND** MUST apply `font-variant-numeric: tabular-nums`

### Requirement: Report empty, loading, and error states SHALL be announced

An empty results set SHALL render via a dedicated partial inside an `aria-live="polite"` container whose text explains the empty condition in domain-specific copy (e.g. "No billable time in the selected range"). A loading state (where rendered) SHALL use `role="status"` with accessible text. An error state SHALL use `role="alert"` with `tabindex="-1"` and receive focus via `data-focus-after-swap`.

#### Scenario: Empty results after filter change

- **GIVEN** a user applies a filter that returns no rows
- **WHEN** the empty-state partial is swapped in
- **THEN** the container MUST carry `aria-live="polite"`
- **AND** the text MUST explain the empty condition in domain-specific copy

#### Scenario: Server error during report generation

- **GIVEN** the server returns an error during report generation
- **WHEN** the error partial is swapped in
- **THEN** the container MUST carry `role="alert"` and `tabindex="-1"`
- **AND** focus MUST land on the error container via `data-focus-after-swap`
