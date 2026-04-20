## ADDED Requirements

### Requirement: Rate rules UI uses HTMX partials for inline edit

The rate rules management page MUST support creating, editing, and deleting rate rules through server-rendered HTMX partials so that each mutation updates only the affected row or table region without a full page reload. The page MUST continue to function when HTMX is unavailable or JavaScript is disabled.

The system MUST expose:

- `GET /rates/{id}/edit` — returns the rate row in edit mode as a partial.
- `GET /rates/{id}/row` — returns the rate row in display mode as a partial (used to cancel an edit).
- `POST /rates`, `POST /rates/{id}`, `POST /rates/{id}/delete` — when the request includes the `HX-Request: true` header, the server MUST respond with the relevant partial(s) instead of an HTTP redirect. When the header is absent, the server MUST continue to respond with an HTTP 303 redirect to `/rates` as today.

All HTMX edit, create, and delete flows MUST carry CSRF protection via the signed double-submit cookie, exactly as non-HTMX POSTs do.

#### Scenario: Edit end date without full page reload
- **GIVEN** a rate rule exists in the current workspace
- **WHEN** the user clicks "Edit end date" on that row
- **THEN** the server returns the edit-mode rate row partial
- **AND** only that table row is replaced in the DOM
- **AND** submitting a valid new `effective_to` replaces the row with its display-mode partial
- **AND** the rest of the page, including scroll position, is preserved

#### Scenario: Cancel edit restores display row
- **GIVEN** the user has opened the edit form on a rate row
- **WHEN** the user clicks Cancel
- **THEN** the server returns the display-mode rate row partial for that same rule
- **AND** the edit form is removed from the DOM

#### Scenario: Create rule re-renders table without page reload
- **WHEN** the user submits a valid new rate rule via HTMX
- **THEN** the server responds with the rates table partial and a reset rate-form partial
- **AND** the new rule appears in the table without a full page reload

#### Scenario: Delete unreferenced rule re-renders table without page reload
- **GIVEN** a rate rule has no referenced time entries
- **WHEN** the user confirms deletion via HTMX
- **THEN** the server responds with the rates table partial
- **AND** the deleted rule's row is no longer present

#### Scenario: No-JS fallback still works
- **GIVEN** a browser with JavaScript disabled
- **WHEN** the user submits the "New rate rule" form
- **THEN** the server responds with HTTP 303 and the browser navigates to `/rates`
- **AND** the newly created rule is visible on the reloaded page

### Requirement: Rate rule writes emit `rates-changed` peer-refresh event

On every successful create, update, or delete of a rate rule initiated via HTMX, the system MUST set the `HX-Trigger` response header to include `rates-changed`, so that other partials in the page (and future views) can subscribe via `hx-trigger="rates-changed from:body"` without polling. The event MUST NOT be emitted when a mutation fails.

#### Scenario: Successful create emits `rates-changed`
- **WHEN** an HTMX `POST /rates` succeeds
- **THEN** the response includes an `HX-Trigger` header whose value contains `rates-changed`

#### Scenario: Successful update emits `rates-changed`
- **WHEN** an HTMX `POST /rates/{id}` succeeds
- **THEN** the response includes an `HX-Trigger` header whose value contains `rates-changed`

#### Scenario: Successful delete emits `rates-changed`
- **WHEN** an HTMX `POST /rates/{id}/delete` succeeds
- **THEN** the response includes an `HX-Trigger` header whose value contains `rates-changed`

#### Scenario: Failed mutation does not emit `rates-changed`
- **WHEN** an HTMX mutation is rejected for validation, overlap, or historical-safety reasons
- **THEN** the response MUST NOT include `rates-changed` in `HX-Trigger`

### Requirement: Rate rule partial preserves historical-safety semantics

The HTMX edit flow MUST continue to enforce that rules referenced by historical time entries can only have their `effective_to` date extended; every other column (`scope`, `client_id`, `project_id`, `currency_code`, `hourly_rate_minor`, `effective_from`) MUST remain immutable regardless of what the client submits. Tampered hidden fields MUST NOT cause a referenced rule to mutate.

#### Scenario: Attempt to change currency on a referenced rule fails inline
- **GIVEN** a rate rule is referenced by at least one time entry
- **WHEN** the user (or a crafted request) submits an HTMX update with a different `currency_code`
- **THEN** the server returns HTTP 409 with the rate row partial in edit mode
- **AND** an inline error is shown via `aria-describedby` on the edit form
- **AND** the stored rule is unchanged

#### Scenario: Delete of a referenced rule fails inline without removing the row
- **GIVEN** a rate rule is referenced by at least one time entry
- **WHEN** the user confirms an HTMX delete
- **THEN** the server returns HTTP 409
- **AND** the row remains in the rates table
- **AND** the row shows "Referenced by N entries" and the Delete button is disabled
- **AND** an inline error describes why deletion is blocked

### Requirement: Rate rule inline edit preserves keyboard focus

The HTMX rates UI MUST preserve sensible keyboard focus across swaps so keyboard-only users are not dropped to the top of the document after each action. The system MUST:

- Focus the `Effective to` input after entering edit mode.
- Focus the row's "Edit end date" disclosure control after a successful update.
- Focus the "New rate rule" scope select after a successful create.
- Focus the "New rate rule" scope select after a successful delete (since the deleted row no longer exists).

Focus behavior MUST use the existing `data-focus-after-swap` convention wired in `web/static/js/app.js`.

#### Scenario: Focus lands on the end-date input after opening edit
- **WHEN** the user activates "Edit end date" on a rate row
- **THEN** after the HTMX swap completes, keyboard focus is on the `Effective to` input for that row

#### Scenario: Focus lands on the row's disclosure after a successful save
- **WHEN** the user submits a valid updated end date
- **THEN** after the swap, keyboard focus is on the "Edit end date" disclosure within the replaced row

#### Scenario: Focus lands on the new-rule form after create or delete
- **WHEN** an HTMX create or delete succeeds
- **THEN** after the swap, keyboard focus is on the "New rate rule" scope select

### Requirement: Rate rule partial meets WCAG 2.2 AA

The rate row and rate form partials MUST continue to meet WCAG 2.2 AA. Validation and conflict errors MUST be rendered as text adjacent to the offending field and associated via `aria-describedby`, MUST have `role="alert"` and `aria-live="assertive"`, and MUST NOT rely on color alone. All interactive controls within the partials MUST have visible labels and visible keyboard focus. Destructive delete MUST use a confirmation affordance (e.g., `hx-confirm`) before issuing the delete request.

#### Scenario: Overlap error on inline edit references the offending field
- **WHEN** an HTMX update is rejected for overlap
- **THEN** the response includes the rate row partial in edit mode
- **AND** the error text is adjacent to the date fields
- **AND** the date inputs reference the error via `aria-describedby`
- **AND** the error has `role="alert"` and is not conveyed by color alone

#### Scenario: Delete requires confirmation
- **WHEN** the user clicks Delete on an unreferenced rate row
- **THEN** a confirmation prompt is shown before the HTMX request is issued
- **AND** declining the prompt cancels the request without any server call
