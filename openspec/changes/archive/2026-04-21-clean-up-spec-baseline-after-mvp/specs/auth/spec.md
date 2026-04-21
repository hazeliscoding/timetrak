## MODIFIED Requirements

<!--
  NO-OP DELTA. This change is prose-only: it rewrites the non-normative
  `## Purpose` section of `openspec/specs/auth/spec.md` and does NOT
  alter any requirement. The requirement block below is copied verbatim
  from the baseline so the OpenSpec validator recognises a MODIFIED delta.
  See proposal.md and design.md (Decision 1).
-->

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
