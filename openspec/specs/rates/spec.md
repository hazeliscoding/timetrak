# rates Specification

## Purpose
TBD - created by archiving change bootstrap-timetrak-mvp. Update Purpose after archive.
## Requirements
### Requirement: Rate rule storage with effective date windows

The system SHALL store billing rates as `rate_rules` with `workspace_id`, optional `client_id`, optional `project_id`, `currency_code`, a non-negative `hourly_rate_minor` (money as integer minor units), `effective_from` (date), and `effective_to` (nullable date; NULL means open-ended). A workspace-default rule has `client_id IS NULL AND project_id IS NULL`. A client-level override has `client_id` set and `project_id IS NULL`. A project-level override has `project_id` set.

#### Scenario: Workspace-default rule creation
- **WHEN** Alice creates a workspace default of 10000 minor units effective from 2026-01-01 with no end date
- **THEN** a `rate_rules` row is stored with `workspace_id = W1`, `client_id = NULL`, `project_id = NULL`, `hourly_rate_minor = 10000`, `effective_from = 2026-01-01`, `effective_to = NULL`

#### Scenario: Money stored as integer minor units
- **WHEN** any rate rule is created or edited
- **THEN** the persisted `hourly_rate_minor` MUST be an integer
- **AND** the system MUST NOT persist or compute rates using floating-point types

#### Scenario: Non-negative rate enforced
- **WHEN** a create/edit attempts `hourly_rate_minor < 0`
- **THEN** the system MUST reject at application validation
- **AND** the database check constraint MUST reject the row if validation is bypassed

### Requirement: No overlapping date windows at the same level

The system MUST reject a rate-rule create or edit that would cause two rules at the same precedence level (workspace-default, same client, or same project) to overlap in their effective date ranges. `effective_from` is inclusive and `effective_to` is inclusive; adjacency (`A.effective_to + 1 day = B.effective_from`) is the required pattern for a handoff. Two rules at the same level MUST NOT share any UTC date, including sharing the boundary date (`A.effective_to = B.effective_from` is rejected as overlap).

#### Scenario: Overlapping workspace-default rejected

- **GIVEN** a workspace default exists with `effective_from = 2026-01-01`, `effective_to = NULL`
- **WHEN** a second workspace default is submitted with `effective_from = 2026-06-01`, `effective_to = NULL`
- **THEN** the submission MUST be rejected with a validation error
- **AND** the new row is not inserted

#### Scenario: Adjacent, non-overlapping windows are allowed

- **GIVEN** a workspace default with `effective_from = 2026-01-01`, `effective_to = 2026-06-30`
- **WHEN** a new workspace default is submitted with `effective_from = 2026-07-01`, `effective_to = NULL`
- **THEN** the submission MUST be accepted

#### Scenario: Shared boundary date is rejected as overlap

- **GIVEN** a workspace default with `effective_from = 2026-01-01`, `effective_to = 2026-06-30`
- **WHEN** a new workspace default is submitted with `effective_from = 2026-06-30`, `effective_to = NULL`
- **THEN** the submission MUST be rejected as overlapping
- **AND** the error message MUST reference both windows' date ranges

### Requirement: Centralized rate resolution with precedence

The system SHALL provide a single rate-resolution function `Resolve(workspaceID, projectID, at)` that returns the applicable rate by consulting rules in this precedence order:

1. project-level rule for `project_id = projectID` active at `at`
2. client-level rule for `client_id = project's client_id` active at `at`
3. workspace-default rule for `workspace_id = workspaceID` active at `at`
4. no-rate sentinel

A rule is "active at `at`" when, with `date = at.UTC()::date`: `effective_from <= date AND (effective_to IS NULL OR date <= effective_to)`. Because the system rejects overlapping windows at the same level (see overlap requirement), at most one rule per precedence tier is ever active on a given date. `Resolve` MUST be invoked exactly once per entry at the moment the entry is created, stopped, or edited (see snapshot requirement) and MUST NOT be invoked on the reporting read path. Live-preview features that display a rate for a yet-unsaved entry MAY call `Resolve` but MUST label the result as a preview, not a historical figure.

#### Scenario: Project rule wins over client and workspace

- **GIVEN** a workspace default, a client rule, and a project rule all active on 2026-04-17
- **WHEN** `Resolve(W1, P1, 2026-04-17)` is called
- **THEN** the project rule is returned

#### Scenario: Fall through to client when no project rule

- **GIVEN** no project rule active on 2026-04-17 for `P1`, but a client rule and workspace default are active
- **WHEN** `Resolve(W1, P1, 2026-04-17)` is called
- **THEN** the client rule is returned

#### Scenario: Fall through to workspace default when no project or client rule

- **GIVEN** only the workspace default is active on 2026-04-17
- **WHEN** `Resolve(W1, P1, 2026-04-17)` is called
- **THEN** the workspace default is returned

#### Scenario: No rule anywhere returns no-rate sentinel

- **GIVEN** no active rule exists at any level for 2026-04-17
- **WHEN** `Resolve(W1, P1, 2026-04-17)` is called
- **THEN** a no-rate sentinel is returned
- **AND** the resulting entry snapshot columns MUST all be NULL
- **AND** the UI MUST visibly distinguish `No rate` from `0.00` in rate columns

#### Scenario: Resolution uses the entry's started_at UTC date

- **GIVEN** a time entry with `started_at = 2026-03-31T23:30:00-04:00` (local) which is `2026-04-01T03:30:00Z` UTC
- **AND** rule `R1` is effective through 2026-03-31 and rule `R2` is effective from 2026-04-01
- **WHEN** the entry is stopped and `Resolve` is called
- **THEN** the resolver MUST return `R2` (because `at.UTC()::date = 2026-04-01`)

#### Scenario: Historical entry's snapshot is stable across subsequent rule edits

- **GIVEN** a closed entry snapshotted against rule `R1` at 10000 minor units
- **WHEN** `R1` is edited in any way the system still permits (e.g., `effective_to` extended into the future)
- **THEN** the entry's `hourly_rate_minor` MUST remain 10000
- **AND** subsequent reports over that entry MUST show the original figure

### Requirement: Rate snapshot produced at entry close/save time

The system SHALL, within the same database transaction that closes a running timer or persists an edit to a time entry's `project_id`, `started_at`, `ended_at`, `duration_seconds`, or `is_billable`, invoke `rates.Service.Resolve(ctx, workspaceID, projectID, started_at)` exactly once and persist the result as a snapshot on the time entry (`rate_rule_id`, `hourly_rate_minor`, `currency_code`). Running entries (`ended_at IS NULL`) MUST NOT carry a snapshot. Reports MUST read the snapshot for closed entries instead of re-resolving at read time.

#### Scenario: Stopping a timer writes the snapshot

- **GIVEN** a running entry for project `P1` started at 2026-04-17T09:00Z in workspace `W1`
- **AND** a workspace-default rate rule `R1` of 10000 minor units USD effective from 2026-01-01 with no end
- **WHEN** Alice stops the timer at 2026-04-17T10:00Z
- **THEN** the entry row MUST have `rate_rule_id = R1`, `hourly_rate_minor = 10000`, `currency_code = 'USD'`
- **AND** the snapshot MUST be written in the same transaction that set `ended_at` and `duration_seconds`

#### Scenario: Editing an entry re-evaluates the snapshot

- **GIVEN** a closed entry previously snapshotted against rule `R1`
- **WHEN** Alice edits the entry's `project_id` to a project whose client has an active client-level rule `R2`
- **THEN** the saved entry MUST carry `rate_rule_id = R2` and `R2`'s rate and currency
- **AND** the snapshot update MUST occur in the same transaction as the entry edit

#### Scenario: No resolvable rate produces NULL snapshot columns

- **GIVEN** no rate rule is active for the entry's project, client, or workspace on the entry's `started_at` date
- **WHEN** the entry is stopped or saved
- **THEN** `rate_rule_id`, `hourly_rate_minor`, and `currency_code` MUST all be NULL
- **AND** the entry MUST still be surfaced in the existing `Entries without a rate` aggregate

#### Scenario: Running timer has no snapshot

- **GIVEN** a running entry (`ended_at IS NULL`)
- **WHEN** any code reads the entry
- **THEN** `rate_rule_id`, `hourly_rate_minor`, and `currency_code` MUST be NULL
- **AND** reporting MUST NOT include the running entry in billable amounts until it is stopped

#### Scenario: Snapshot columns are atomic (all-set or all-null)

- **WHEN** the system writes a snapshot on a time entry
- **THEN** either all three of `rate_rule_id`, `hourly_rate_minor`, `currency_code` are NULL
- **OR** all three are non-NULL
- **AND** a database check constraint MUST enforce this invariant

### Requirement: Rate rules referenced by time entries are immutable on the historical axis

The system MUST reject any mutation of a `rate_rules` row that would alter the historical figure seen by at least one `time_entries.rate_rule_id = R`. Specifically, when any entry references rule `R`, the system MUST reject a delete of `R`, and MUST reject an update that changes `hourly_rate_minor`, `currency_code`, `client_id`, `project_id`, `effective_from`, or that shortens `effective_to` to a date earlier than the latest `started_at::date` among referencing entries. The system SHALL allow updates that only extend an open-ended `effective_to` from NULL to a future date, or that shorten `effective_to` to a date on or after the latest referencing entry's `started_at::date`. Rejection MUST return a typed error distinguishable from `ErrNotFound` and `ErrOverlap`, and the UI MUST surface a text explanation (not color alone) that names the number of referencing entries.

#### Scenario: Delete of referenced rule is rejected

- **GIVEN** rate rule `R1` is referenced by three closed time entries in workspace `W1`
- **WHEN** Alice attempts to delete `R1`
- **THEN** the system MUST reject the delete with a typed error meaning "rule is referenced by historical entries"
- **AND** the database row MUST remain unchanged

#### Scenario: Edit of rate amount on a referenced rule is rejected

- **GIVEN** rate rule `R1` is referenced by at least one closed time entry
- **WHEN** Alice attempts to update `R1.hourly_rate_minor` from 10000 to 12000
- **THEN** the system MUST reject the update with a typed error
- **AND** the database row MUST remain unchanged
- **AND** the UI MUST display a text hint naming the number of referencing entries and suggesting creating a successor rule

#### Scenario: Closing an open-ended rule on a future date is allowed

- **GIVEN** rate rule `R1` has `effective_from = 2026-01-01`, `effective_to = NULL`, and is referenced by entries dated 2026-01-10 through 2026-04-17
- **WHEN** Alice sets `R1.effective_to = 2026-06-30` (a future date)
- **THEN** the system MUST accept the update
- **AND** the database row's `effective_to` MUST be `2026-06-30`

#### Scenario: Shortening effective_to below latest referencing entry is rejected

- **GIVEN** rate rule `R1` has `effective_to = 2026-12-31` and is referenced by an entry dated 2026-04-17
- **WHEN** Alice attempts to set `R1.effective_to = 2026-03-31`
- **THEN** the system MUST reject the update with the typed "referenced" error
- **AND** the database row MUST remain unchanged

#### Scenario: Rate rules UI shows a referenced-count hint

- **GIVEN** rate rule `R1` is referenced by 3 time entries
- **WHEN** the rate rules list is rendered
- **THEN** the row for `R1` MUST display a textual indicator such as `Referenced by 3 entries` next to the edit/delete controls
- **AND** the indicator MUST NOT rely on color alone
- **AND** destructive controls MUST be disabled with a `aria-describedby`-associated explanation

### Requirement: Rate management UI accessibility

The rate rules management UI MUST meet WCAG 2.2 AA. Money inputs MUST clearly indicate the currency and the unit (e.g., a label like `Rate (USD per hour)` and a helper describing that amounts are entered as decimals and stored as minor units by the system). Effective-date inputs MUST use native `<input type="date">`. Validation errors (overlaps, negative rate) MUST be conveyed by text next to the offending field and associated via `aria-describedby`.

#### Scenario: Keyboard-only rate rule creation
- **WHEN** a keyboard-only user creates a workspace-default rate rule
- **THEN** they can complete and submit the form without a pointer
- **AND** all fields have visible labels and visible focus rings

#### Scenario: Overlap error is announced
- **WHEN** an overlap validation error occurs
- **THEN** the error text appears adjacent to the effective-date field
- **AND** the date inputs reference the error via `aria-describedby`
- **AND** the error is not conveyed by color alone


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
