## 1. Restore static-asset parity

- [x] 1.1 Added `GET /static/` handler in `internal/e2e/server_harness.go` — `http.FileServer(http.Dir(...))` rooted at `FindRepoRoot(t)/web/static/`, matching `cmd/web/main.go:161-162`.
- [x] 1.2 Verified via `TestAxeSmokePerPage` run — styled pages now render; `color-contrast` assertions now fire (they couldn't before because no CSS was applied).

## 2. Restore settings-route parity

- [x] 2.1 Imported `timetrak/internal/settings`. Snapshot `tzList` via `wsSvc.ListTimezones(context.Background())` inside `BuildServer`, mirroring `cmd/web/main.go:129-136`.
- [x] 2.2 Constructed `settings.NewHandler(wsSvc, tpls, lay, tzList).Register(mux, protect)` in the same position as the production server.
- [x] 2.3 `GET /workspace/settings` (the real route — see note below) returns 200 on the test server.
- [x] 2.4 Fixed `internal/e2e/browser/axe_smoke_test.go` — replaced two stale test paths: `/entries` → `/time` and `/settings` → `/workspace/settings`. These paths never existed at their old names; the test was aspirational. Parity means the tests hit real routes.

## 3. Verify browser tests now render styled pages

- [x] 3.1 `DATABASE_URL=... go test -tags=browser -run TestAxeSmokePerPage` now exercises real CSS. `TestAxeSmokePerPage/settings` no longer appears in the fail list (title + lang now present).
- [x] 3.2 Remaining axe failures recorded as follow-ups, NOT fixed here:
  - **Chip contrast** (`color-contrast` on `.tt-chip-billable`) — the pre-existing accent-500 on accent-100 ≈3.8:1 concern flagged in `sharpen-component-identity` group 8. Axe's surfacing of this was masked by the missing CSS.
  - **Active nav contrast** (`color-contrast` on `a[aria-current="page"]`) — accent text on accent-soft fill has the same ~3.8:1 issue.
  - **Link-in-text-block** (`<a>` inside `<p>`) — links rely on color alone at rest (no underline). Pre-existing pattern, now visible.
  - **`TestFocusAfterSwapContract`** hangs on `/clients/new` — that route doesn't exist (clients uses an inline-form pattern at `/clients`). This is a test-code bug, not a harness issue.
  - None of the above are `restore-e2e-test-server-parity`'s scope — they are real product / test-suite contract failures that were hidden by the missing CSS. Pick up in a `fix-browser-test-suite-contract-failures` follow-on or as part of the next UI change that touches those surfaces.

## 4. Verification and archival

- [x] 4.1 `make fmt && make vet && make test` — non-browser suite still green.
- [x] 4.2 Browser suite summary: `TestBrandmarkInAppHeader` ✓, `TestFaviconLinkPresentOnPublicPages` ✓, `TestFaviconResourceServes` ✓, `TestAxeSmokePerPage/settings` ✓ (was failing), 8 other axe sub-pages flag real contract failures, `TestFocusAfterSwapContract` hangs on a test-code bug, `TestFocusRingContract` hangs on the same `/clients/new` seed issue. Tests that exercised CSS-dependent contracts (target-size, focus-ring appearance) can now run meaningfully.
- [x] 4.3 Committed via `tt-conventional-commit` as `fix(e2e): restore test-server route parity with cmd/web`. No Claude attribution.
- [x] 4.4 Archived 2026-04-23 via `/opsx:archive restore-e2e-test-server-parity --yes`. Spec sync: ~1 MODIFIED in `ui-browser-tests` (added scenarios for static assets + settings route).
