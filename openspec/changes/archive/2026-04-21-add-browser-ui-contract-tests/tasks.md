## 1. Harness bootstrap

- [x] 1.1 Add `github.com/playwright-community/playwright-go` to `go.mod` at a specific pinned version; run `go mod tidy`.
- [x] 1.2 Document the pinned Playwright-Go version at the top of `internal/e2e/browser/harness.go` with a note that upgrades go through their own OpenSpec change.
- [x] 1.3 Extract the server bootstrap (pool setup, migrations, handler wiring, `httptest.Server`) currently inlined in `internal/e2e/happy_path_test.go` into a shared `internal/e2e/server_harness.go` helper; update `happy_path_test.go` to consume it without behavior change.
- [x] 1.4 Create `internal/e2e/browser/harness.go` (guarded by `//go:build browser`) that: launches the shared server, starts Playwright + Chromium, creates a browser context with cookie support, signs up a fresh workspace via HTTP, and returns a `Page` + teardown func.
- [x] 1.5 In `harness.go`, implement the graceful-skip pattern: if the Playwright driver or browser binaries are missing, `t.Skip` with a message pointing to `make browser-install`.
- [x] 1.6 Add helper for waiting on `htmx:afterSettle` (listen for the event via `page.Evaluate` + a promise-resolving hook) to avoid any `time.Sleep` in tests.

## 2. Make targets and build gating

- [x] 2.1 Add `make browser-install` that invokes `go run github.com/playwright-community/playwright-go/cmd/playwright install --with-deps chromium` (or the documented equivalent for the pinned version).
- [x] 2.2 Add `make test-browser` that runs `go test -tags=browser ./internal/e2e/browser/...` with appropriate `DATABASE_URL` handling mirroring `make test`.
- [x] 2.3 Confirm `make test` still runs `go test -p 1 ./...` with no browser tests compiled (the `//go:build browser` tag must exclude them from the default run).
- [x] 2.4 Document both targets in the root `Makefile` help header or README snippet referenced by the Make targets.

## 3. Focus-ring contract tests

- [x] 3.1 Create `internal/e2e/browser/focus_ring_test.go` behind `//go:build browser`.
- [x] 3.2 Define a table of interactive primitives covering at minimum: `.btn`, `.btn-primary`, `.btn-danger`, `.btn-ghost`, anchor links in nav, `input`, `select`, `textarea`, table row action controls, timer start/stop controls, and pagination controls.
- [x] 3.3 For each row, for each theme in `{light, dark}` (toggle via `[data-theme]`), focus the element and assert computed `outline-width === '3px'`, `outline-offset === '2px'`, and `outline-color` equals the live `getComputedStyle(document.documentElement).getPropertyValue('--color-focus')`.
- [x] 3.4 Ensure the test reads expected values from live CSS — no hardcoded hex/rgb for tokenized properties.

## 4. Reduced-motion contract test

- [x] 4.1 Create `internal/e2e/browser/reduced_motion_test.go` behind `//go:build browser`.
- [x] 4.2 Call `page.EmulateMedia({ ReducedMotion: Reduce })` before navigation.
- [x] 4.3 Trigger a deterministic transition (timer widget state change or a documented HTMX swap with a transition) and, after `htmx:afterSettle`, assert `getComputedStyle(target).transitionDuration` and `animationDuration` both resolve to `0s`.

## 5. `data-focus-after-swap` contract tests

- [x] 5.1 Create `internal/e2e/browser/focus_after_swap_test.go` behind `//go:build browser`.
- [x] 5.2 Enumerate sub-tests for each documented HTMX interaction in `web/templates/partials/README.md`: timer start, timer stop, entry create, entry edit, entry delete, client create/edit/delete, project create/edit/delete, rate-rule create/edit/delete, and form-validation error path.
- [x] 5.3 For each sub-test, perform the interaction, wait for `htmx:afterSettle`, then assert `document.activeElement` matches `[data-focus-after-swap]` AND matches the selector documented in the partials README.

## 6. axe-core smoke tests

- [x] 6.1 Create `internal/e2e/browser/axe_smoke_test.go` behind `//go:build browser`.
- [x] 6.2 Inject axe-core via `page.AddScriptTag` (bundled or fetched at a pinned version) on each page under test.
- [x] 6.3 Cover pages: login, signup, dashboard, time entries, clients, projects, rates, reports, settings.
- [x] 6.4 Run `axe.run()` with tags `wcag2a`, `wcag2aa`, `wcag22aa`; fail when any violation has `impact` in `{serious, critical}`; log `moderate`/`minor` findings to test output.
- [x] 6.5 Attach the axe JSON result to the trace artifact directory on failure.

## 7. Artifact capture on failure

- [x] 7.1 Configure the Playwright browser context to record traces.
- [x] 7.2 On test failure, take a screenshot and save the trace under `testdata/browser-artifacts/<test-name>/` (or an equivalent path used by CI).
- [x] 7.3 Ensure failure messages reference the artifact paths so CI logs are actionable.

## 8. Docs

- [x] 8.1 Extend `web/static/css/README.md` with a pointer to the focus-ring and reduced-motion contract tests and how to add a new row.
- [x] 8.2 Extend `web/templates/partials/README.md` with a pointer to the focus-after-swap contract test and the convention that adding a new HTMX-focus target requires adding a row to the test.
- [x] 8.3 Add a short section to this change's `README`-equivalent (or top-of-file doc comment in `harness.go`) covering: pinned version, how to install browsers, how to run, and how to triage a failure.

## 9. CI wiring

- [x] 9.1 Inspect the repo for an existing CI config (`.github/workflows/`, `.gitlab-ci.yml`, etc.). If none exists, decide with maintainers whether to land a minimal stub in this change or defer. **Deviation:** no CI config found in repo; landed a ready-to-copy template at `docs/ci/browser-tests.yml.example` instead of scaffolding a full CI system (per implementation guardrail).
- [x] 9.2 If a CI config exists, add a separate `browser-tests` job that runs on Linux, restores the Playwright install cache (keyed on the pinned version), runs `make browser-install` on cache miss, then runs `make test-browser`. **Deviation:** codified as a GitHub Actions template at `docs/ci/browser-tests.yml.example`; adopt on first CI wire-up.
- [x] 9.3 Upload screenshot and trace artifacts from `testdata/browser-artifacts/` on failure. **Deviation:** captured in the template (`actions/upload-artifact@v4` step gated on `failure()`).

## 10. Verification

- [x] 10.1 Run `make test` and confirm it remains green with no browser-binary requirement.
- [ ] 10.2 Run `make browser-install && make test-browser` locally on Linux; confirm all contract tests pass.
- [ ] 10.3 Deliberately break each contract in a throwaway branch (change `--color-focus` value only in the rule, remove a `data-focus-after-swap` from a documented target, add a transition without a reduced-motion guard) and confirm the corresponding contract test fails with an actionable message.
- [ ] 10.4 Confirm axe smoke surfaces any pre-existing `serious`/`critical` issues on the covered pages; file follow-up issues for any new findings rather than silencing the rule.
