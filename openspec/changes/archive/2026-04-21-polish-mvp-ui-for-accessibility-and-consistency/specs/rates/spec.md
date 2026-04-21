## ADDED Requirements

### Requirement: Rate form SHALL meet WCAG 2.2 AA accessibility

The rate create/edit form SHALL:

- Pair every input with a visible `<label>` and mark required fields with visible text plus `aria-required="true"`.
- Wire inline validation errors via `aria-describedby` pointing at an element with id `#rate-form-<field>-error`, and mark the input with `aria-invalid="true"`.
- Render a top-of-form error summary element with `role="alert"`, `tabindex="-1"`, and `data-focus-after-swap` on validation failure; the summary SHALL list each error as a link to its field.
- Announce changes to the scope selector (`project` / `client` / `workspace default`) via text — hidden scope-specific field groups SHALL use `hidden` and the visible group SHALL retain a readable label. The no-JS fallback (all groups visible) SHALL remain functional.
- On successful save, return focus to the `Add rule` button via `data-focus-after-swap` on the re-rendered form. On validation failure, focus SHALL land on the error summary.

#### Scenario: Submitting with an invalid amount

- **GIVEN** a user submits the rate form with an invalid amount
- **WHEN** the server re-renders the form via HTMX
- **THEN** the amount input MUST carry `aria-invalid="true"` and `aria-describedby` pointing at a visible error element
- **AND** a top-of-form `role="alert"` summary MUST list the error
- **AND** focus MUST land on the summary after the swap

#### Scenario: Scope toggle shows the correct group

- **GIVEN** a user changes the scope selector from `workspace default` to `project`
- **WHEN** the change event fires
- **THEN** only the project-scope field group MUST be visible
- **AND** all hidden groups MUST carry the `hidden` attribute

#### Scenario: Successful save returns focus to Add rule

- **GIVEN** a rate rule is saved successfully via HTMX
- **WHEN** the form re-renders
- **THEN** focus MUST land on the `Add rule` button via `data-focus-after-swap`

### Requirement: Rates table SHALL present accessible table semantics and non-color-only status

The rates table SHALL include a `<caption>`, `<th scope="col">` on header cells, right-aligned numeric columns via a CSS utility class, and rate-scope labels (`project`, `client`, `workspace default`) SHALL render as text — color SHALL NOT be the sole differentiator between scopes. All scope pill variants SHALL meet 4.5:1 contrast against their background.

Empty-state rendering SHALL use a partial inside an `aria-live="polite"` region so async changes triggered by the `rates-changed` event are announced.

#### Scenario: Scope is distinguishable without color

- **GIVEN** a workspace has rate rules across multiple scopes
- **WHEN** the rates table renders
- **THEN** each row MUST include a visible scope label as text
- **AND** the text-on-background contrast of each scope pill MUST be at least 4.5:1

#### Scenario: Empty rates table announces via live region

- **GIVEN** a workspace has no rate rules
- **WHEN** the page renders
- **THEN** the empty-state container MUST carry `aria-live="polite"`
- **AND** the copy MUST be domain-specific (e.g. "No rate rules yet — start with a workspace default")

### Requirement: Deleting a rate rule SHALL use native hx-confirm

Deleting a rate rule from the list SHALL use native `hx-confirm`. After a successful delete, focus SHALL return to the `Add rule` button via `data-focus-after-swap`.

#### Scenario: Delete a rate rule

- **GIVEN** a rate rule exists
- **WHEN** a user clicks `Delete` on that row
- **THEN** a native `confirm()` dialog MUST appear
- **AND** on confirmation, the row MUST be removed and focus MUST land on the `Add rule` button
