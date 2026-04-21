---
name: tt-write-e2e-test
description: Author an end-to-end test for TimeTrak — either an in-process HTTP e2e test under internal/e2e/ (cookie jar, CSRF, HTMX endpoints) or a Playwright+Chromium browser contract test under internal/e2e/browser/ (focus, tokens, accessibility, deterministic waits). Use when a task demands coverage that the service/handler layer cannot express.
license: MIT
compatibility: TimeTrak. Requires DATABASE_URL for HTTP e2e; additionally requires `make browser-install` and the `browser` build tag for the browser suite.
metadata:
  author: timetrak
  version: "1.0"
---

Sibling skill to `tt-write-integration-test`. Integration tests own service/handler correctness; e2e tests own **flows** (signup → CSRF → HTMX → reports) and **browser-observable contracts** (focus-after-swap, token-backed styling, axe smoke, reduced-motion).

**Pick the right layer — if the assertion fits at a lower layer, write it there instead.**

| Layer | When to use | Package |
| --- | --- | --- |
| Integration (service / handler) | Logic, authz boundaries, SQL behavior | `internal/<domain>/` (see `tt-write-integration-test`) |
| HTTP e2e | Multi-step flows, CSRF, cookie-jar sessions, HX-Trigger chains | `internal/e2e/` (no build tag) |
| Browser e2e | Anything only observable in a real browser: focus, computed styles, axe, `prefers-reduced-motion` | `internal/e2e/browser/` (`//go:build browser`) |

---

## Path A — HTTP e2e (`internal/e2e/`)

1. **Read the peer**
   Open `internal/e2e/happy_path_test.go` for the canonical shape: cookie-jar `client`, `c.post` that warms `GET /` to obtain `tt_csrf`, `extractFirst` for row-ID regex, `StatusSeeOther` for redirect endpoints.

2. **Boot the server**
   ```go
   ts := e2e.BuildServer(t)   // uses testdb.Open → skips if DATABASE_URL unset
   c := newClient(t, ts)
   ```
   `BuildServer` truncates domain tables on entry. Do NOT call `t.Parallel()` — all e2e packages share one Postgres and `make test` runs `-p 1`.

3. **Drive the flow through real HTTP**
   - Use `c.post(path, url.Values{...})` — it sets `Content-Type: application/x-www-form-urlencoded` and injects `csrf_token` from the cookie jar.
   - Follow redirects manually: `CheckRedirect` returns `ErrUseLastResponse`. Assert `StatusSeeOther` (303) for form posts, `StatusOK` for HTMX endpoints.
   - For HTMX endpoints that return partials (e.g. `/timer/start`, `/timer/stop`), assert `StatusOK` and (when relevant) inspect the body or `HX-Trigger` header.

4. **Extract IDs the way peers do**
   The happy-path test parses IDs from the rendered row markup:
   ```go
   clientID := extractFirst(t, `id="client-row-([0-9a-f-]+)"`, body(t, resp))
   ```
   Mirror that — it couples the test to the partial contract, which is the point.

5. **Cross-workspace isolation**
   Use two independent `*client` instances (two cookie jars = two sessions). Assert HTTP 404 on cross-workspace reads and mutations — never 403. See `TestWorkspaceIsolation404` for the pattern.

6. **Run**
   ```bash
   go test ./internal/e2e/... -run TestYourFlow -count=1 -v
   make test   # full gate
   ```

---

## Path B — Browser contract (`internal/e2e/browser/`)

1. **Every file starts with the build tag**
   ```go
   //go:build browser

   package browser
   ```
   Non-negotiable. It keeps the default `make test` build hermetic for contributors without Chromium.

2. **Start the harness**
   ```go
   h := StartHarness(t)
   if h == nil { return } // skipped cleanly: missing driver/Chromium
   h.SignupFreshWorkspace("My Scenario")
   h.GotoPath("/dashboard")
   ```
   `StartHarness` boots `e2e.BuildServer`, launches Chromium, opens a traced context, and parks the page on `about:blank`. It calls `t.Skipf` with a pointer to `make browser-install` when binaries are absent — **never** make that a hard failure.

3. **Synchronize on deterministic events only**
   Legal waits:
   - `WaitForHTMXSettle(h.Page)` — wraps `htmx:afterSettle`.
   - `page.WaitForSelector(...)` / `WaitForLoadState(...)`.
   - Locators' built-in auto-wait on `Click`, `Fill`, etc.

   **Prohibited:** `time.Sleep`, `page.WaitForTimeout`, any arbitrary sleep. Flakes caught here MUST be fixed at the source.

4. **Read tokens live; never hardcode**
   Focus-ring, color, spacing, and motion assertions MUST read from `getComputedStyle(document.documentElement)` (or the element under test):
   ```go
   val, _ := h.Page.Evaluate(`() => getComputedStyle(document.documentElement).getPropertyValue('--focus-ring-color').trim()`)
   ```
   A hardcoded `#2563eb` or `3px` in a browser test is a bug — it defeats the one regression class this suite exists to catch (token renamed in one place, missed in another).

5. **Focus-after-swap contract**
   Mirror `assertFocusedHasFocusAfterSwapAttr` from `focus_after_swap_test.go`: after an HTMX swap, `document.activeElement` MUST carry `[data-focus-after-swap]`. Read `web/templates/partials/README.md` for which partials participate; if your new swap target does not yet document a focus behavior, update the README **first** and cite it in the test.

6. **Accessibility (axe) smoke**
   For new UI surfaces, add a scenario to the axe smoke suite rather than spawning a parallel axe harness. The vendored axe-core bundle is resolved via `h.RepoRoot`.

7. **Artifacts on failure**
   Traces land under `internal/e2e/browser/testdata/browser-artifacts/`. Inspect with `npx playwright show-trace <path>`. CI is expected to upload this directory on failure — see `docs/ci/browser-tests.yml.example`.

8. **Run**
   ```bash
   make browser-install   # one-time
   go test -tags=browser ./internal/e2e/browser/... -run TestYourContract -count=1 -v
   make test-browser      # full browser suite
   ```

---

## Guardrails (both paths)

- Never call `t.Parallel()` — e2e packages share one Postgres.
- Never assert 403 for cross-workspace access. The contract is 404.
- Never bump the pinned `PlaywrightVersion` — that requires its own OpenSpec change (see header of `internal/e2e/browser/harness.go`).
- Never add `time.Sleep` to a browser test.
- Never hardcode tokenized values in a browser assertion.
- Never leave a flow under-asserted with a lone `StatusOK` — also inspect the body, `HX-Trigger`, or the resulting DOM so a regression that silently drops content is caught.
- If the test requires a new domain table truncation, add it in `internal/shared/testdb/testdb.go` first.

## When NOT to write an e2e test

- Pure service/handler correctness → `tt-write-integration-test`.
- Money math, rate resolution, workspace authz SQL → integration layer.
- Template rendering unit (tag present / field interpolated) → a plain `html/template` test is cheaper than booting Chromium.

E2e tests are load-bearing but slow and heavy. Add one only when the behavior is genuinely end-to-end (flow, cookie jar, browser-observable contract). Otherwise, push the assertion down.

**Fluid Workflow Integration**
Invoke from an `openspec-apply-change` task that says "add e2e coverage for X", "assert the focus contract holds for Y", "verify token Z drives the rendered style", or "cover the cross-workspace 404 for the new route". Tick the task `- [x]` only after the targeted test **and** `make test` (for HTTP e2e) or `make test-browser` (for browser) pass.
