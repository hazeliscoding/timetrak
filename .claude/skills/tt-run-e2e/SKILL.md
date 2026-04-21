---
name: tt-run-e2e
description: Run TimeTrak's end-to-end test suites — the in-process HTTP e2e tests under internal/e2e/ and the opt-in Playwright+Chromium browser contract tests under internal/e2e/browser/. Use when verifying UI-affecting changes or triaging an e2e failure.
license: MIT
compatibility: TimeTrak. Requires DATABASE_URL for HTTP e2e; additionally requires `make browser-install` (Playwright driver + Chromium, ~200MB) for the browser suite.
metadata:
  author: timetrak
  version: "1.0"
---

Run the right e2e command for the task at hand. Sibling to `tt-run-tests`; that skill covers unit/integration. This one covers the two full-stack suites that boot a real `httptest.Server` via `internal/e2e.BuildServer`.

**Quick reference**

| Goal | Command |
| --- | --- |
| HTTP e2e (happy path + workspace isolation) | `go test ./internal/e2e/... -count=1` |
| One HTTP e2e test | `go test ./internal/e2e/... -run TestHappyPathSignupToReport -count=1 -v` |
| Browser contract suite | `make test-browser` |
| One browser test | `go test -tags=browser ./internal/e2e/browser/... -run TestFocusAfterSwapContract -count=1 -v` |
| First-time browser install | `make browser-install` |
| Full local gate including e2e | `make fmt && make vet && make test && make test-browser` |

**Steps**

1. **Confirm Postgres is up**
   ```bash
   docker compose ps | grep -q postgres || make db-up
   ```
   Both suites call `internal/shared/testdb.Open`; if `DATABASE_URL` is unset, every test in `internal/e2e/` and `internal/e2e/browser/` skips silently. Verify it is up before claiming green.

2. **HTTP e2e loop**
   ```bash
   go test ./internal/e2e/... -run <TestName> -count=1 -v
   ```
   These tests drive the server through a real `http.Client` with a cookie jar, exercising signup → CSRF → HTMX endpoints. They are part of the default `make test` suite; no extra tags.

3. **Browser e2e — first-time setup**
   ```bash
   make browser-install
   ```
   Downloads the Playwright-Go driver and Chromium (~200MB). Pinned to the version surfaced at `internal/e2e/browser/harness.PlaywrightVersion`. Do NOT upgrade the pin in passing — it requires its own OpenSpec change (see the harness file header).

4. **Browser e2e loop**
   ```bash
   make test-browser
   # or, targeted:
   go test -tags=browser -p 1 ./internal/e2e/browser/... -run <TestName> -count=1 -v
   ```
   The `browser` build tag keeps these files out of the default `make test` build, so developers without Chromium can still run the main suite.

5. **Pre-commit gate for UI work (binding)**
   ```bash
   make fmt && make vet && make test && make test-browser
   ```
   Any change that touches `web/templates/`, `web/static/css/`, the focus/HTMX partials, or design tokens SHOULD run `make test-browser` before commit. The browser suite is the only layer that catches token drift (focus ring, reduced-motion, axe smoke) and post-swap focus regressions.

6. **Triaging a failure**
   - **Browser skipped with "browser-install"**: driver/Chromium missing. Run `make browser-install` and retry.
   - **HTTP e2e skipped**: `DATABASE_URL` unset or Postgres down. `make db-up` and export the URL.
   - **Flaky browser test**: the harness forbids `time.Sleep`; the only legal waits are `htmx:afterSettle`, completed responses, `WaitForSelector`, `WaitForLoadState`. If a test is flaky, audit for a sneaky sleep or a race against an unsettled HTMX swap — do not add retries.
   - **Token assertion failing only in browser**: tokens MUST be read live via `getComputedStyle(document.documentElement)`. A hardcoded hex/rgb string in a browser test is a bug — it defeats the one regression class this suite exists to catch.
   - **Artifacts on failure**: traces land in `internal/e2e/browser/testdata/browser-artifacts/`. Open the Playwright trace with `npx playwright show-trace <path>` for a frame-by-frame replay.
   - **CSRF 403 in HTTP e2e**: the `client.post` helper warms the cookie jar with `GET /` before reading `tt_csrf`. If a new flow posts before any GET, warm it manually.

7. **When to NOT run the browser suite**
   - Backend-only changes with no template or token diff.
   - Doc-only changes.
   - Migration-only changes.
   In those cases, `make test` alone is sufficient.

**Guardrails**
- Never bypass `//go:build browser`. That tag exists so the default build stays hermetic. If you need a browser test to run by default, the premise is wrong — write the assertion at the HTTP or unit layer.
- Never bump the pinned Playwright version ad-hoc — see `harness.go` header for the OpenSpec-change requirement.
- Never add `time.Sleep` to a browser test. Use the deterministic synchronization primitives listed above.
- Never hardcode a tokenized color/px value in a browser assertion. Read it from `getComputedStyle`.
- Never claim the gate passed without running `make test-browser` when the change touches the UI layer described in step 5.

**Fluid Workflow Integration**
Invoke alongside `tt-run-tests` whenever an `openspec-apply-change` task ticks off UI-affecting work (templates, tokens, HTMX wiring, accessibility). Run step 5's gate before any commit or `openspec-archive-change`.
