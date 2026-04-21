## Why

The recent UI/a11y hardening cycle (polish-mvp-ui, create-reusable-ui-partials, establish-custom-component-library-foundation) left several durable regression checks unchecked because they require a human walkthrough: a11y keyboard walks, focus-visible visual verification, reduced-motion behavior, focus-after-swap targets, and per-theme focus rings. Any future edit to a CSS token, template partial, or the HTMX focus handler can silently break these contracts without any signal from `make test`. We need an automated harness that fails fast when these contracts drift.

## What Changes

- Introduce a browser-driven UI contract test harness, gated behind a `//go:build browser` tag so the default `make test` stays hermetic and fast.
- Standardize on **Playwright-Go** so tests stay inside the Go test runner and reuse the existing `internal/e2e/` server bootstrap and `internal/shared/testdb` fixtures.
- Add three contract test families against the already-shipped UI specs:
  - Focus-ring contract (per `ui-foundation`): for every interactive primitive across light + dark themes, assert computed `:focus-visible` outline width/color/offset match the live CSS custom properties.
  - Reduced-motion contract: under `prefers-reduced-motion: reduce`, assert deterministic transitions collapse to effectively zero duration.
  - `data-focus-after-swap` contract (per `ui-partials`): for each documented HTMX interaction, assert the post-swap active element matches the expected selector after `htmx:afterSettle`.
- Add an axe-core smoke pass per top-level page (login, signup, dashboard, entries, clients, projects, rates, reports, settings) failing on `serious`+`critical` findings.
- Add `make test-browser` and `make browser-install` targets; leave `make test` unchanged.
- Document how to add a new contract test in `web/static/css/README.md` and `web/templates/partials/README.md`.

## Capabilities

### New Capabilities
- `ui-browser-tests`: Automated browser-driven UI contract tests covering focus-ring, reduced-motion, focus-after-swap, and baseline axe-core assertions; defines the harness shape, gating, and the contracts that MUST fail the test suite when violated.

### Modified Capabilities
<!-- None. This change is additive; existing ui-foundation and ui-partials specs remain authoritative for the contract values themselves. -->

## Impact

- Adds a Go module dependency on `github.com/playwright-community/playwright-go` (pinned).
- Adds a new test tree at `internal/e2e/browser/` gated by `//go:build browser`.
- Adds `make test-browser` and `make browser-install` to `Makefile`; `make test` is unchanged.
- Extracts the server bootstrap helper currently inlined in `internal/e2e/happy_path_test.go` into a shared helper that both the existing e2e test and the new browser harness consume.
- CI gains an opt-in browser-test stage (Chromium-only, Linux, single viewport). No cross-OS/cross-browser matrix.
- No changes to production code paths, no new migrations, no new domain logic.
- Explicitly out of scope: pixel/visual regression, cross-browser matrix, mobile viewports, Lighthouse budgets, Storybook/component showcase, and any replacement of existing Go integration tests.
