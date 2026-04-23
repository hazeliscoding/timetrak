# ui-browser-tests Specification

## Purpose
The ui-browser-tests capability defines how TimeTrak exercises its
browser-facing UI contracts through a Playwright harness gated behind
the `browser` Go build tag. It covers the focus-ring contract in both
themes, the reduced-motion contract, the `data-focus-after-swap`
behavior documented in the partials README, token-to-computed-style
fidelity read from live CSS, and a per-page axe-core smoke test. The
harness reuses the existing server bootstrap and `internal/shared/testdb`
fixtures rather than standing up parallel infrastructure, synchronizes
on deterministic HTMX and DOM events rather than wall-clock sleeps, runs
on a single pinned browser configuration in CI, and emits actionable
artifacts on failure. `make test` and `go test ./...` MUST remain
hermetic with respect to this harness.
## Requirements
### Requirement: Browser-driven test harness MUST be gated behind a build tag

The repository SHALL provide a browser-driven UI contract test harness under `internal/e2e/browser/`, with every file carrying the `//go:build browser` build constraint. The default `go test ./...` and `make test` commands MUST NOT compile or execute these tests. A dedicated `make test-browser` target SHALL exist to run them, and a `make browser-install` target SHALL install the required browser binaries on demand.

#### Scenario: Default test run skips browser tests
- **WHEN** a developer runs `make test` or `go test ./...`
- **THEN** no files under `internal/e2e/browser/` are compiled or executed
- **AND** the test run remains hermetic with respect to browser binaries

#### Scenario: Opt-in browser test run
- **WHEN** a developer runs `make test-browser` after `make browser-install`
- **THEN** the browser-tagged tests under `internal/e2e/browser/...` are executed against a locally launched server

#### Scenario: Graceful skip when browser binaries are absent
- **WHEN** `make test-browser` is run without Playwright browser binaries installed
- **THEN** each browser test SHALL skip with a message instructing the developer to run `make browser-install`
- **AND** the test suite MUST NOT fail due to the missing binaries

### Requirement: Harness MUST reuse the existing server bootstrap and testdb fixtures

The browser harness SHALL launch the TimeTrak HTTP server via the same bootstrap used by `internal/e2e/happy_path_test.go` and SHALL reuse `internal/shared/testdb` for schema setup and per-test truncation. It MUST NOT introduce a parallel server, a parallel migration runner, or a parallel database lifecycle.

"Same bootstrap" means route parity with `cmd/web/main.go`: every route registered by the production server (including static assets at `/static/*` and the settings surface at `/settings`) SHALL be registered by the test bootstrap. The only permitted divergences are the fixed test-only session secret, the `APP_ENV=dev` flag that enables the developer showcase surface, and the use of `httptest.Server` as the network transport.

#### Scenario: Shared server bootstrap
- **WHEN** a browser test starts
- **THEN** the server is constructed through the shared `internal/e2e` helper used by the non-browser e2e test
- **AND** the database is prepared through `internal/shared/testdb`

#### Scenario: Cross-workspace isolation between tests
- **WHEN** two browser tests execute sequentially against the same database
- **THEN** each test sees a clean set of domain tables via the existing truncate fixtures
- **AND** neither test can observe state written by the other

#### Scenario: Static assets are served by the test server

- **WHEN** a browser test requests `/static/css/tokens.css`, `/static/css/app.css`, `/static/js/app.js`, or `/static/vendor/htmx.min.js`
- **THEN** the test server returns the file from `web/static/` with HTTP 200
- **AND** the rendered page has the real compiled CSS applied, so contract assertions that depend on computed styles (target-size, focus-ring contrast, tabular-nums, accent borders) have a valid DOM to inspect.

#### Scenario: Settings route is registered on the test server

- **WHEN** a browser test navigates an authenticated workspace-scoped session to `/settings`
- **THEN** the response status is 200
- **AND** the page renders under the standard app layout with a valid `<title>` and `lang` attribute, so `TestAxeSmokePerPage/settings` can assert WCAG 2.2 AA compliance against a real page.

### Requirement: Harness MUST synchronize on deterministic events, not wall-clock sleeps

Every browser test SHALL wait on deterministic signals (`htmx:afterSettle`, completed HTTP responses, `page.WaitForLoadState`, or equivalent) before asserting on DOM state after an HTMX interaction. Raw `time.Sleep` and raw `setTimeout` SHALL NOT be used to paper over async behavior.

#### Scenario: HTMX swap synchronization
- **WHEN** a test triggers an HTMX swap
- **THEN** the test waits on `htmx:afterSettle` (or an equivalent deterministic event) before asserting
- **AND** the assertion does not depend on a fixed sleep duration

### Requirement: Focus-ring contract MUST be enforced for every interactive primitive in both themes

A focus-ring contract test SHALL enumerate every interactive primitive in the UI foundation (at minimum: `.btn`, `.btn-primary`, `.btn-danger`, `.btn-ghost`, anchors, `input`, `select`, `textarea`, table row action controls, timer controls, nav links). For each primitive, under both `[data-theme="light"]` and `[data-theme="dark"]`, the test SHALL focus the element and assert:

- `outline-width` equals `3px`,
- `outline-offset` equals `2px`,
- the resolved `outline-color` equals the live value of the `--color-focus` CSS custom property on `:root`.

The expected `--color-focus` value MUST be read from the live stylesheet at test time via `getComputedStyle(document.documentElement)`. It MUST NOT be hardcoded as a hex or rgb string in the test.

#### Scenario: Focus ring matches the live token in light theme
- **WHEN** an interactive primitive receives focus while `html[data-theme="light"]`
- **THEN** its computed `outline-width` is `3px`, `outline-offset` is `2px`, and `outline-color` matches the computed value of `--color-focus`

#### Scenario: Focus ring matches the live token in dark theme
- **WHEN** an interactive primitive receives focus while `html[data-theme="dark"]`
- **THEN** its computed `outline-width` is `3px`, `outline-offset` is `2px`, and `outline-color` matches the computed value of `--color-focus` resolved under the dark theme

#### Scenario: Renaming the focus token without updating the rule fails the test
- **WHEN** a PR renames `--color-focus` or changes the focus-ring rule on one side only
- **THEN** the focus-ring contract test fails for the affected primitives

### Requirement: Reduced-motion contract MUST be enforced

A reduced-motion test SHALL emulate `prefers-reduced-motion: reduce` and trigger a deterministic UI transition (timer state change or an HTMX swap with a documented transition). After the transition, the target element's computed `transition-duration` and `animation-duration` MUST be effectively zero (`0s`).

#### Scenario: Transitions respect reduced-motion preference
- **WHEN** the browser emulates `prefers-reduced-motion: reduce`
- **AND** a known transitioning element is rendered or updated
- **THEN** the element's computed `transition-duration` and `animation-duration` resolve to `0s`

#### Scenario: Introducing a transition without a reduced-motion guard fails the test
- **WHEN** a PR adds a CSS transition or animation that does not honor `prefers-reduced-motion`
- **THEN** the reduced-motion contract test fails

### Requirement: `data-focus-after-swap` contract MUST match the partials README

A focus-after-swap test SHALL cover every HTMX interaction documented in `web/templates/partials/README.md`, including at minimum: timer start, timer stop, entry create, entry edit, entry delete, client create/edit/delete, project create/edit/delete, rate-rule create/edit/delete, and form validation error paths. For each scenario, after the swap settles, the test SHALL assert that `document.activeElement` matches a `[data-focus-after-swap]` element AND that the focused selector matches the target documented in the partials README.

#### Scenario: Focus lands on the documented target after swap
- **WHEN** an HTMX interaction documented in the partials README completes its swap
- **THEN** `document.activeElement` has the `data-focus-after-swap` attribute
- **AND** it matches the selector the README designates as the focus target for that interaction

#### Scenario: Removing a focus-after-swap target fails the test
- **WHEN** a PR removes `data-focus-after-swap` from a documented target without updating the README
- **THEN** the focus-after-swap contract test fails for that scenario

### Requirement: axe-core smoke test MUST run per top-level page

An axe-core smoke test SHALL be injected into each top-level page (at minimum: login, signup, dashboard, time entries, clients, projects, rates, reports, settings) and run against the rule tags `wcag2a`, `wcag2aa`, and `wcag22aa`. The test SHALL fail when any violation has `impact` of `serious` or `critical`. Violations at `moderate` or `minor` impact SHALL be logged as warnings and attached to the test artifacts but SHALL NOT fail the run.

#### Scenario: Critical axe violation fails the run
- **WHEN** axe-core reports a violation with impact `serious` or `critical` on any covered page
- **THEN** the browser test run fails for that page
- **AND** the failure message includes the axe rule id and the offending selector

#### Scenario: Moderate axe finding is logged only
- **WHEN** axe-core reports a violation with impact `moderate` or `minor`
- **THEN** the finding is attached to the test trace artifact
- **AND** the test run does not fail solely because of that finding

### Requirement: Expected token values MUST be read from live CSS

Any assertion that compares a computed style against a design-token value SHALL read the expected token value from the live stylesheet at test time (e.g. via `getComputedStyle(document.documentElement).getPropertyValue('--color-focus')`). Hardcoding expected hex or rgb strings in the test source is prohibited for tokens that exist as CSS custom properties.

#### Scenario: Token rename updated everywhere results in a passing run
- **WHEN** a CSS custom property is renamed and every consumer (including the focus-ring rule) is updated in the same change
- **THEN** the browser contract tests continue to pass without modification

#### Scenario: Token rename that misses a consumer fails the contract test
- **WHEN** a CSS custom property is renamed but one consumer of that token is not updated
- **THEN** the relevant contract test fails at the diverged assertion

### Requirement: Browser tests MUST run on a single pinned configuration

The browser test suite SHALL run on Chromium only, on a single desktop viewport, on Linux in CI. The Playwright-Go dependency SHALL be pinned in `go.mod` and the pin SHALL be documented in the harness source. Adding a second browser, a mobile viewport, or a cross-OS matrix requires a separate OpenSpec change.

#### Scenario: Single-browser, single-viewport run
- **WHEN** `make test-browser` runs in CI
- **THEN** tests execute against Chromium at a single desktop viewport
- **AND** no other browser or viewport is exercised in this change

#### Scenario: Dependency upgrade requires its own change
- **WHEN** a developer attempts to bump the Playwright-Go pin
- **THEN** that bump is proposed as its own OpenSpec change rather than bundled into unrelated work

### Requirement: Failing browser tests MUST emit actionable artifacts

On failure, each browser test SHALL attach at least a screenshot and a Playwright trace (or equivalent) to the test output directory so the failure can be diagnosed without re-running the suite locally.

#### Scenario: Failure produces a screenshot and a trace
- **WHEN** a browser test fails
- **THEN** a screenshot and a Playwright trace are written to the test artifacts directory
- **AND** the test failure message references those artifacts

