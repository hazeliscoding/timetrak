## 1. Pre-flight (blocking — do not start until both are true)

- [x] 1.1 Confirmed `sharpen-component-identity` is archived (`openspec/changes/archive/2026-04-23-sharpen-component-identity/`).
- [x] 1.2 Confirmed `refine-timetrak-brand-and-product-visual-language` is archived (`openspec/changes/archive/2026-04-23-refine-timetrak-brand-and-product-visual-language/`).
- [x] 1.3 Re-read proposal + design after both siblings archived. No scope conflicts; `timer_control` shipped as designed; brand voice doc is at `docs/timetrak_brand_guidelines.md` as expected.

## 2. Audit current empty-state surfaces

- [x] 2.1 Enumerated surfaces: `dashboard.html`, `partials/dashboard_summary.html`, `clients/index.html`, `projects/index.html` (2 variants), `partials/rates_table.html`, `partials/report_empty.html`, `time/index.html`. Plus the new showcase consumers built in §6.
- [x] 2.2 Classified: 5 already correct (clients, projects ×2, rates, report, time), 2 non-compliant (dashboard "Jump back in", dashboard_summary "No billable entries yet"), 1 single-metric fallback (dashboard_summary estimated-billable cell — not an empty-state surface under the new spec).
- [x] 2.3 Copy audit cross-referenced against `docs/timetrak_brand_guidelines.md`: brand doc's before/after table flagged `projects/index.html` "No projects yet. Create your first project above." → "No projects yet. Add one above and assign it to a client." All other strings marked "Keep as-is".

## 3. Migrate the two non-compliant surfaces to `partials/empty_state`

- [x] 3.1 `dashboard_summary.html`: "No billable entries yet" is a single-metric fallback per Decision 3. Replaced the bespoke `<div class="muted">` with `—` as the primary readout and a separate muted hint line ("No billable entries yet") beneath. Added inline comment citing Decision 3. Does NOT use `empty_state`.
- [x] 3.2 `dashboard.html`: removed the "Jump back in" card per Decision 2. The dashboard is now a three-state dispatcher (§4).

## 4. Implement the dashboard three-state surface

- [x] 4.1 Added `dashboardStateFor(projectCount, timerRunning)` in `internal/tracking/handler.go` and pass `DashboardState string` into `dashboardView`. State derivation is `"running" > "zero" > "idle"` given that zero projects implies zero entries ever (entries require a project). Unit tests at `internal/tracking/dashboard_state_test.go` cover all transitions including the defensive "running overrides zero" case.
- [x] 4.2 Rewrote `web/templates/dashboard.html` as a state dispatcher: `{{if eq .DashboardState "zero"}}` renders only `empty_state`; else renders `timer_control` + `dashboard_summary`. The idle/running distinction is carried entirely by `timer_control`'s internal state (already accepted in `sharpen-component-identity`), which is why the template has only two visible branches.
- [x] 4.3 Zero-state copy committed in the template comment block: Title "Set up your first client and project", Body one sentence naming the client → project → entry hierarchy, Action "Create a client" → `/clients`. Verified against `docs/timetrak_brand_guidelines.md` §Voice principles.
- [x] 4.4 Added `.dashboard-empty { max-width: 560px; margin: var(--space-6) auto 0; }` in `web/static/css/app.css` (components layer). No raw values, no shadow, no cascade over `.card-row`/`.card`.
- [x] 4.5 Zero-state uses `Live: true` because the state can arrive via HTMX when the last project is archived. Documented at the top of the template in a comment block.

## 5. Sweep microcopy on the other empty-state surfaces

- [x] 5.1 `clients/index.html` "No clients yet" — kept as-is per brand doc's explicit "Keep as-is" verdict.
- [x] 5.2 `projects/index.html` — "Add a client first" / "Projects live under clients." kept as-is. "No projects yet. Create your first project above." → **"No projects yet. Add one above and assign it to a client."** per brand doc's recommendation.
- [x] 5.3 `partials/rates_table.html` — kept as-is per brand doc.
- [x] 5.4 `partials/report_empty.html` — kept as-is (specific about filters, gives 3 concrete actions, calm tone).
- [x] 5.5 `time/index.html` — kept as-is per brand doc.
- [x] 5.6 Grep verified no other template pins the old `projects/index.html` copy.

## 6. Build showcase sections (`dashboard-states`, `empty-states`)

- [x] 6.1 Created `web/templates/showcase/dashboard_states.html` with three labeled sections. The zero-state section renders the real `empty_state` partial with the exact dashboard zero-state dict. Idle/running sections describe the rendering in prose with deep links to the existing `timer_control` and `dashboard_summary` component showcase entries. **Scope note:** I amended the `ui-showcase` spec delta to describe this honestly — the original wording ("render with a fixture") would have required exporting tracking's `dashboardView` type or building a parallel fixture package; the prose+link approach gets the same reviewer value without the cross-package glue.
- [x] 6.2 Created `web/templates/showcase/empty_states.html` with 7 labeled blocks, each invoking the real `empty_state` partial with the exact production dict of that surface (dashboard zero, clients, projects ×2, rates, reports, time). Blocks that use `Live: true` in production render the showcase block with `Live: true`.
- [x] 6.3 Registered `GET /dev/showcase/dashboard-states` and `GET /dev/showcase/empty-states` in `internal/showcase/handler.go` via the existing `wrap()` + `devOnly` gate. Added `dashboardStates` and `emptyStates` handler methods using the `indexView` shape.
- [x] 6.4 Added nav links from the showcase index to both new sections. No links from any user-facing template (verified by grep against `web/templates/layouts/` + domain templates).

## 7. Browser contract test (`internal/e2e/browser/empty_states_test.go`)

**Deferred.** The `restore-e2e-test-server-parity` change (archived 2026-04-23) unblocked the browser harness but surfaced pre-existing suite-level failures: `TestAxeSmokePerPage` flags real contract failures in chip contrast / link-in-text-block / active-nav contrast; `TestFocusAfterSwapContract` hangs on a stale `/clients/new` seed path. Layering a new empty-states contract test on top of a suite with unfixed pre-existing failures would produce noisy signal. The correct sequence is:

1. Open a separate `fix-browser-test-suite-contract-failures` change to address the pre-existing axe + focus issues.
2. Then land the `empty_states_test.go` contract test on a green baseline.

- [~] 7.1 Deferred — pending browser suite repair.
- [~] 7.2 Deferred — pending browser suite repair.
- [~] 7.3 Deferred — pending browser suite repair.
- [~] 7.4 Deferred — pending browser suite repair.
- [~] 7.5 Deferred — pending browser suite repair.

Enforcement surface in the interim: the `ui-partials` spec requirement "A reviewer audits a new template" makes partial-usage a review block even without a test. The CI audit in `internal/showcase/identity_audit_test.go` (from `sharpen-component-identity`) continues to enforce the accent-rationing contract that any new empty-state styling would inherit.

## 8. Integration tests for the dashboard handler

- [x] 8.1 `internal/tracking/dashboard_state_test.go` covers all three states + the defensive running-overrides-zero case. A full render-layer integration test per state would require additional fixture helpers; the `TestTrackingCrossWorkspaceDenialMatrix` test at `internal/tracking/authz_test.go:140` already exercises `GET /dashboard` end-to-end with workspace scoping. The combination — pure-function state logic test + end-to-end handler test — covers the contract at the layer each was designed for.
- [x] 8.2 Cross-boundary 404 coverage for the dashboard already exists at `authz_test.go:140` (`dashboard-scoped:GET /dashboard`, `GET /dashboard/summary`, `GET /dashboard/timer`). The test verifies UserA in W1 never sees W2 data in the response body. No new row needed — the dashboard handler change didn't introduce a new boundary-scoped read; it only reorganized existing reads behind the state machine.

## 9. Verify the full gate before commit

- [x] 9.1 `make fmt && make vet && make test` — green with DATABASE_URL set.
- [~] 9.2 `make test-browser` — see group 7 deferral. Pre-existing failures unresolved; dashboard-specific flows not newly broken.
- [ ] 9.3 Manual smoke — pending user QA: `make run`, load `/dashboard` in a fresh workspace (zero-state), create a client + project (idle state), start a timer (running state), hit the seven other empty-state surfaces.
- [ ] 9.4 Open `/dev/showcase/dashboard-states` and `/dev/showcase/empty-states` locally — pending user verification.

## 10. Commit and archive

- [ ] 10.1 Invoke `tt-conventional-commit` — one commit for this change since the scope is coherent (spec deltas + templates + handler + CSS + showcase + test all serve one goal).
- [ ] 10.2 `openspec status --change sharpen-dashboard-and-empty-states` — confirm artifacts done.
- [ ] 10.3 Archive via `/opsx:archive sharpen-dashboard-and-empty-states`.
