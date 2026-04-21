## ADDED Requirements

### Requirement: Projects list SHALL present accessible table semantics

The projects list table SHALL include a `<caption>` element (visually hidden where the page header already conveys the label), `<th scope="col">` on every header cell, and `<th scope="row">` on the first cell of each row when that cell identifies the project. Numeric columns SHALL be right-aligned via a CSS utility class with `font-variant-numeric: tabular-nums`, not inline styles.

Empty-state rendering SHALL use a dedicated partial inside an `aria-live="polite"` region so async HTMX filter changes are announced.

#### Scenario: Table has caption and column scopes

- **GIVEN** a workspace has at least one project
- **WHEN** a user loads `/projects`
- **THEN** the projects `<table>` MUST contain a `<caption>` (sr-only acceptable)
- **AND** every `<th>` in `<thead>` MUST carry `scope="col"`
- **AND** the project-name cell MUST be a `<th scope="row">`

#### Scenario: Empty state announces via live region

- **GIVEN** a workspace has no projects
- **WHEN** the page renders
- **THEN** the empty-state container MUST carry `aria-live="polite"`

### Requirement: Project status pills SHALL NOT rely on color alone

`Archived`, `No rate`, and any other project-status pill SHALL include visible text. Text-on-background contrast SHALL meet 4.5:1.

#### Scenario: No-rate pill carries text

- **GIVEN** a project has no resolved rate (neither project-level, client-level, nor workspace default)
- **WHEN** the project row renders
- **THEN** the indicator MUST contain the visible text `No rate`
- **AND** the text-on-background contrast MUST be at least 4.5:1

### Requirement: Destructive project actions SHALL use the documented confirmation pattern

Deleting a project with no time entries SHALL use native `hx-confirm`. Deleting a project with historical time entries SHALL use the `<dialog>`-based confirmation partial that states how many entries will be affected before confirming.

After a successful delete, focus SHALL land on a stable anchor (the `New project` button) via `data-focus-after-swap`. After cancellation, focus SHALL return to the row's action button.

#### Scenario: Delete a project with no entries

- **GIVEN** a project has no time entries
- **WHEN** a user clicks `Delete`
- **THEN** a native `confirm()` dialog MUST appear
- **AND** on confirmation, focus MUST land on the `New project` button

#### Scenario: Delete a project with historical entries shows count

- **GIVEN** a project has N historical time entries
- **WHEN** a user clicks `Delete`
- **THEN** the `<dialog>`-based confirmation MUST open stating the number of entries affected
- **AND** focus MUST be trapped inside the dialog
- **AND** on close, focus MUST return to the `Delete` button

### Requirement: Project form SHALL meet WCAG 2.2 AA accessibility

The project create/edit form SHALL wire inline errors via `aria-describedby`, set `aria-invalid="true"` on invalid inputs, render a top-of-form error summary with `role="alert"` and `data-focus-after-swap`, and mark required fields with visible text and `aria-required="true"`.

#### Scenario: Creating a project with missing required field

- **GIVEN** a user submits the form with a missing required field
- **WHEN** the server re-renders via HTMX
- **THEN** the invalid input MUST carry `aria-invalid="true"` and `aria-describedby`
- **AND** a top-of-form `role="alert"` summary MUST list the error
- **AND** focus MUST land on the summary after swap
