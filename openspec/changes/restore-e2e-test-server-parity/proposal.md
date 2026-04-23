## Why

The accepted `ui-browser-tests` capability requires the browser harness to reuse the existing server bootstrap — specifically, the requirement "Harness MUST reuse the existing server bootstrap and testdb fixtures." In practice, `internal/e2e/server_harness.go::BuildServer` diverges from `cmd/web/main.go` in two ways:

1. **No `/static/*` route is mounted** — `/static/css/tokens.css`, `/static/css/app.css`, `/static/js/app.js`, and `/static/vendor/htmx.min.js` all return 404 against the test server. Browser tests therefore render pages with no CSS, which breaks any contract that depends on rendered styles (target-size, focus-ring contrast, tabular-nums rendering, accent-border treatment).
2. **No `/settings` handler is registered** — `/settings` returns 404, causing `TestAxeSmokePerPage/settings` to fail on `document-title` and `html-has-lang` because the 404 page lacks a proper shell.

This is a straightforward bug in the test harness, not a feature. It has been latent since the browser harness landed: `testdata/browser-artifacts/TestFocusAfterSwapContract/failure.png` has been in the working tree across recent commits, and `TestAxeSmokePerPage` flags `target-size` violations that only exist because CSS isn't applied. The recent `sharpen-component-identity` change had to defer its browser contract extensions (tasks 9.1–9.5) because the harness can't render CSS-dependent assertions today.

## What Changes

- **`BuildServer`** in `internal/e2e/server_harness.go` mounts `/static/*` using the same `http.FileServer` + `http.StripPrefix` pattern as `cmd/web/main.go`, pointed at the repo's `web/static/` directory resolved via `FindRepoRoot`.
- **`BuildServer`** registers the existing `settings` handler (already imported elsewhere in the codebase) so `/settings` returns a real page under the shared layout.
- No new middleware, no new routes beyond what `cmd/web` exposes. The harness becomes byte-for-byte identical to `cmd/web` in route surface except for the fixed test-only session secret and the test-only `APP_ENV=dev` flag for showcase routes.
- No API or behavior changes to product code. No DB migrations.

Out of scope: extending browser tests to exercise the now-working CSS contracts. That work is queued in the archived `sharpen-component-identity` tasks.md (group 9) and will be picked up by the next UI change that needs it.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `ui-browser-tests`: No requirement-level behavior changes — the existing "Harness MUST reuse the existing server bootstrap" requirement is already correct. This change adds two new **clarifying scenarios** under that requirement to pin what "reuse" means concretely (static assets served at `/static/*`; `/settings` registered) so regressions are caught at review time.

## Impact

- `internal/e2e/server_harness.go` — imports `path/filepath` and `timetrak/internal/settings` (already a package in the repo), adds ~10 lines to register the static handler and the settings handler.
- No change to `cmd/web/main.go`, no change to handler packages, no change to templates or CSS.
- Browser tests that were previously failing due to missing CSS — `TestAxeSmokePerPage`, `TestFocusRingContract`, and `TestBrandSurfaceAxeSmoke` in particular — will now run against a real rendered page. Some may surface legitimate contract failures they couldn't expose before; those are separate follow-up work, not part of this change.
- `testdata/browser-artifacts/TestFocusAfterSwapContract/failure.png` and related `.json` snapshots will regenerate on the next `make test-browser` run; leaving their disposition to the user.
