## ADDED Requirements

### Requirement: Timer widget SHALL meet WCAG 2.2 AA accessibility

The timer widget SHALL:

- Expose the current running state in text — not color alone. A `Running` indicator MUST include visible text; the elapsed time MUST be readable and SHALL use `font-variant-numeric: tabular-nums` so digits do not shift.
- Be fully keyboard-operable: `Start timer` and `Stop timer` reachable via `Tab`, activated with `Enter` or `Space`.
- After `POST /timer/start` or `POST /timer/stop` swaps the widget via HTMX, focus SHALL land on the opposite-action button via `data-focus-after-swap` so the user can continue keyboard-only.
- On error (e.g. HTTP 409 `ErrActiveTimerExists`), the error partial SHALL render with `role="alert"`, `tabindex="-1"`, and `data-focus-after-swap`; focus SHALL land on the error container.

#### Scenario: Starting the timer moves focus to Stop

- **GIVEN** a user has no active timer
- **WHEN** the user presses `Enter` on the `Start timer` button
- **THEN** after the HTMX swap, focus MUST land on the `Stop timer` button

#### Scenario: 409 error renders an accessible alert

- **GIVEN** a concurrent `Start timer` has already created an active entry
- **WHEN** a second start request returns HTTP 409
- **THEN** the error partial MUST carry `role="alert"` and `tabindex="-1"`
- **AND** focus MUST land on the error container via `data-focus-after-swap`

#### Scenario: Running indicator carries text

- **GIVEN** a timer is running
- **WHEN** the widget renders
- **THEN** the `Running` indicator MUST include visible text
- **AND** MUST NOT rely on color alone

### Requirement: Entry inline editor SHALL meet WCAG 2.2 AA accessibility

The entry inline editor (opened via `GET /entries/<id>/edit` and replacing the entry row) SHALL:

- Render every input with a visible `<label>` (sr-only is acceptable where the column header already conveys the label and the row's column headers are reachable via `<th scope="col">` / `<th scope="row">`).
- Mark invalid inputs with `aria-invalid="true"` and wire `aria-describedby` to a visible error element on validation failure.
- Render a row-level error summary with `role="alert"` and `data-focus-after-swap` on validation failure; focus SHALL land on the summary.
- On open, focus SHALL land on the description field via `data-focus-after-swap`.
- On cancel (`GET /entries/<id>`) or successful save, focus SHALL return to the `Edit` button on the re-rendered row.

#### Scenario: Opening the editor focuses the description field

- **GIVEN** a user clicks `Edit` on an entry row
- **WHEN** the editor partial is swapped in
- **THEN** focus MUST land on the description input via `data-focus-after-swap`

#### Scenario: Canceling the editor returns focus to Edit

- **GIVEN** the entry editor is open
- **WHEN** the user clicks `Cancel` and the row partial is swapped back in
- **THEN** focus MUST land on the `Edit` button

#### Scenario: Saving with invalid data shows row-level alert

- **GIVEN** the editor is open with invalid data (e.g. negative duration)
- **WHEN** the user submits
- **THEN** the invalid input MUST carry `aria-invalid="true"` and `aria-describedby`
- **AND** a row-level `role="alert"` summary MUST list the error
- **AND** focus MUST land on the summary after swap

### Requirement: Time entries list SHALL present accessible table semantics

The time entries list table SHALL include a `<caption>` (sr-only where the page header already conveys it), `<th scope="col">` on header cells, and numeric columns (duration, billable amount) right-aligned via a CSS utility class with `font-variant-numeric: tabular-nums`. Billable/non-billable status SHALL be indicated by text, not color alone.

Pagination controls SHALL expose the current page and total pages as text (e.g. "Page 2 of 7") and SHALL be keyboard-operable. After a pagination click, focus SHALL land on the first row's cell via `data-focus-after-swap`.

An empty-state partial SHALL render inside an `aria-live="polite"` region when the list is empty.

#### Scenario: Billable indicator carries text

- **GIVEN** an entry is marked billable
- **WHEN** the row renders
- **THEN** the indicator MUST contain the visible text `Billable` or `Non-billable`
- **AND** MUST NOT rely on color alone

#### Scenario: Pagination click focuses first result row

- **GIVEN** the entries list spans multiple pages
- **WHEN** a user clicks `Next`
- **THEN** after the HTMX swap, focus MUST land on the first row cell via `data-focus-after-swap`

#### Scenario: Empty state announces via live region

- **GIVEN** no time entries match the current filter
- **WHEN** the empty-state partial is rendered
- **THEN** its container MUST carry `aria-live="polite"`

### Requirement: Destructive tracking actions SHALL use the documented confirmation pattern

Deleting a time entry SHALL use native `hx-confirm`. Stopping a running timer SHALL use native `hx-confirm` only when no description or unsaved state is in flight; if the timer widget contains an unsaved description buffer, the stop action SHALL use the `<dialog>`-based confirmation partial and state that the description will be preserved on the saved entry.

After a successful destructive action, focus SHALL land on a stable anchor (`New entry` button after delete, `Start timer` button after stop).

#### Scenario: Deleting a time entry

- **GIVEN** a time entry exists
- **WHEN** a user clicks `Delete` on that row
- **THEN** a native `confirm()` dialog MUST appear
- **AND** on confirmation, the row MUST be removed and focus MUST land on a stable page anchor
