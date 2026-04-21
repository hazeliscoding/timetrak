# auth Specification

## Purpose
TBD - created by archiving change bootstrap-timetrak-mvp. Update Purpose after archive.
## Requirements
### Requirement: User registration with email and password

The system SHALL allow a new user to register with a unique email address, a password, and a display name. Passwords MUST be stored only as cryptographic hashes using Argon2id or bcrypt; plaintext passwords MUST NOT be written to any persistent store or log. On successful registration the system SHALL create the user record, provision a default personal workspace, add the user to that workspace as `owner`, and establish an authenticated session — all in a single database transaction.

#### Scenario: Successful signup provisions a personal workspace
- **GIVEN** no account exists for `alice@example.com`
- **WHEN** the user submits the signup form with `alice@example.com`, a compliant password, and display name `Alice`
- **THEN** a `users` row is created with a hashed password
- **AND** a `workspaces` row is created for Alice's personal workspace
- **AND** a `workspace_members` row is created with role `owner`
- **AND** an authenticated session is established with that workspace as the active workspace
- **AND** the user is redirected to `/dashboard`

#### Scenario: Duplicate email is rejected
- **GIVEN** an account already exists for `alice@example.com`
- **WHEN** a signup is submitted with `alice@example.com`
- **THEN** the system MUST reject the submission with a validation error visible on the signup form
- **AND** no new user, workspace, or membership is created
- **AND** the error message does not disclose whether the password met complexity rules

#### Scenario: Weak password is rejected before persistence
- **GIVEN** the signup form is open
- **WHEN** the user submits a password that does not meet the documented minimum length
- **THEN** the system MUST reject the submission with a validation error
- **AND** no `users` row is created

### Requirement: Email and password login with session cookie

The system SHALL authenticate an existing user using email and password, and on success issue a signed, HttpOnly, SameSite=Lax, Secure session cookie bound to a server-side session record. Failed logins MUST NOT disclose whether the email exists.

#### Scenario: Successful login
- **GIVEN** a user account exists for `alice@example.com` with a matching password
- **WHEN** the user submits the login form with correct credentials
- **THEN** the system SHALL create a session record and set the session cookie
- **AND** redirect to `/dashboard`

#### Scenario: Wrong password
- **WHEN** the user submits an incorrect password for an existing email
- **THEN** the system MUST display a generic authentication failure message
- **AND** no session is created
- **AND** the response MUST NOT reveal whether the email was registered

#### Scenario: Unknown email
- **WHEN** the user submits an email that has no account
- **THEN** the system MUST display the same generic authentication failure message used for a wrong password
- **AND** no session is created

### Requirement: Logout invalidates the session

The system SHALL provide a logout action that invalidates the current session on the server and clears the session cookie on the client.

#### Scenario: Logout clears session
- **GIVEN** the user has an active session
- **WHEN** the user submits a logout request
- **THEN** the session record is deleted or marked expired
- **AND** the session cookie is cleared
- **AND** the user is redirected to `/login`

### Requirement: CSRF protection on mutating requests

All state-changing requests (POST, PATCH, DELETE) MUST be protected against cross-site request forgery. Requests lacking a valid CSRF token MUST be rejected with HTTP 403 and MUST NOT mutate state.

#### Scenario: Missing CSRF token is rejected
- **GIVEN** an authenticated session
- **WHEN** a POST request is made without a valid CSRF token
- **THEN** the system MUST respond with HTTP 403
- **AND** no database mutation occurs

#### Scenario: Forged CSRF token is rejected
- **WHEN** a POST request includes a CSRF token that does not match the session-bound value
- **THEN** the system MUST respond with HTTP 403
- **AND** no database mutation occurs

### Requirement: Rate limiting on authentication endpoints

The system SHALL rate-limit login and signup endpoints per client identifier (IP for MVP) to slow credential-stuffing and enumeration attacks. Limits MUST be documented and enforced server-side.

#### Scenario: Excessive login attempts are throttled
- **WHEN** more than the documented threshold of login attempts occurs from the same IP within the documented window
- **THEN** subsequent attempts from that IP MUST receive HTTP 429 until the window elapses
- **AND** legitimate attempts from other IPs MUST NOT be affected

### Requirement: Authentication and signup UI accessibility

The login and signup pages MUST meet WCAG 2.2 AA. Every form control MUST have a visible, programmatically associated label; keyboard focus MUST be visible on all interactive elements; validation errors MUST be conveyed by text (not color alone) and associated with the relevant field via `aria-describedby`; the submit button MUST have a visible accessible name.

#### Scenario: Keyboard-only signup
- **GIVEN** a keyboard-only user on the signup page
- **WHEN** the user tabs through the form
- **THEN** each control receives a visible focus ring
- **AND** the user can complete and submit the form without using a pointer

#### Scenario: Validation error is announced
- **WHEN** signup fails validation on the password field
- **THEN** the error text MUST appear adjacent to the password field
- **AND** the password input MUST reference the error via `aria-describedby`
- **AND** the error MUST NOT be conveyed only by color

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

