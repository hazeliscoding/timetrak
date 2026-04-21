## ADDED Requirements

### Requirement: Authentication forms SHALL meet WCAG 2.2 AA accessibility

The login and signup forms SHALL present every input with a visible `<label>`, mark required fields with both visible text and `aria-required="true"`, and expose inline validation errors via `aria-describedby` pointing at an element with a stable id of `#<form-id>-<field>-error`. Invalid inputs SHALL carry `aria-invalid="true"`.

On server-side validation failure, the form SHALL re-render with a top-of-form error summary element carrying `role="alert"`, `tabindex="-1"`, and `data-focus-after-swap`. The summary SHALL list each field-level error as a link whose href targets the invalid input. Focus SHALL land on the summary after an HTMX swap, or on the first invalid field on a full-page re-render.

The forms SHALL render text-based status for authentication outcomes (e.g. "Invalid email or password", "Account created"). Color SHALL NOT be the sole signal of success or failure.

Every page in the auth flow SHALL set a meaningful `<title>` (e.g. `Sign in — TimeTrak`, `Create account — TimeTrak`).

#### Scenario: Visible labels and required markers on login

- **GIVEN** a user loads `/login`
- **WHEN** the page renders
- **THEN** every input MUST have a visible `<label>` associated via `for`/`id`
- **AND** inputs marked as required MUST carry both a visible indicator and `aria-required="true"`

#### Scenario: Inline error wiring on signup with invalid email

- **GIVEN** a user submits `/signup` with an invalid email
- **WHEN** the server re-renders the form
- **THEN** the email input MUST carry `aria-invalid="true"`
- **AND** the input MUST carry `aria-describedby` pointing at an element with id `signup-email-error` whose text is the human-readable error
- **AND** a top-of-form element with `role="alert"`, `tabindex="-1"`, and `data-focus-after-swap` MUST list the error as a link to `#signup-email`

#### Scenario: Focus lands on error summary after HTMX swap

- **GIVEN** the signup form is submitted with invalid input via HTMX
- **WHEN** the response is swapped into the target
- **THEN** the `htmx:afterSwap` handler MUST focus the error summary element

#### Scenario: Success uses text, not color alone

- **GIVEN** a user successfully signs in
- **WHEN** the success flash is rendered
- **THEN** the message MUST include text stating the outcome (e.g. "Signed in")
- **AND** color MUST NOT be the only cue distinguishing success from failure

#### Scenario: Page title is set per auth page

- **GIVEN** a user loads `/login` or `/signup`
- **WHEN** the page renders
- **THEN** the `<title>` MUST be specific to the page and MUST NOT be the generic layout default
