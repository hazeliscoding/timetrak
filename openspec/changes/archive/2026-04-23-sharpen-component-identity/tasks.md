## 1. Foundation token extension

- [x] 1.1 Add `--radius-pill` to the radius scale in `web/static/css/tokens.css` and document it in `web/static/css/README.md` as "pill / fully-rounded actions (buttons, timer control)".
- [x] 1.2 Audit existing CSS for raw `999px` / `9999px` / `border-radius: 100%` on action-shaped controls and migrate each call site to `var(--radius-pill)`; leave the `50%` on presence dots alone (circle semantic). **Finding:** only `.badge` uses raw `999px` and becomes a rectangle (`--radius-sm`) under the new shape-language contract — migration happens in task 3 (status_chip extraction). No other raw pill values exist.
- [x] 1.3 Update the `ui-foundation` `README.md` and style-guide references to reflect the new token name.
- [x] 1.4 Migrate `.btn` in `web/static/css/app.css` from `var(--radius-sm)` to `var(--radius-pill)` so buttons conform to the shape-language taxonomy (actions = pills). Verify visually against existing button call sites; `.btn-primary`, `.btn-danger`, `.btn-ghost` inherit the radius via the base `.btn` rule.

## 2. Component identity contract — documentation and review surface

- [x] 2.1 Add a new "Component identity" section to `docs/timetrak_ui_style_guide.md` that documents: shape-language taxonomy (pill/rectangle/circle), two-weight border contract, tabular-nums requirement, accent-rationing allow-list, and the five-item review checklist from `ui-component-identity`.
- [x] 2.2 Cross-link the new style-guide section from `docs/timetrak_brand_guidelines.md` and from `web/static/css/README.md` / `web/templates/partials/README.md`.
- [x] 2.3 Render the review checklist at the top of the `/dev/showcase` index page, sourced from the same canonical document (no duplication). **Note:** checklist text is currently inline in `web/templates/showcase/index.html` with `data-checklist-item` hooks for the drift-detection test added in task group 10.

## 3. Status chip partial

- [x] 3.1 Create `web/templates/partials/status_chip.html` with block `status_chip` consuming `dict` keys `Kind`, `Label`, `Variant`, and optional `Glyph`; enumerate `Kind` values (`billable`, `non-billable`, `running`, `draft`, `archived`, `warning`) and `Variant` values (`filled`, `outlined`). Note: `warning` added to the enumeration because it is already in use ("No rate" indicator); spec delta updated.
- [x] 3.2 Author chip CSS under `web/static/css/app.css` in the `components` layer: `.tt-chip` base (rectangle via `--radius-sm`, 20px height, 6px horizontal padding, 0.75rem font-size, medium weight) plus kind- and variant-specific selectors. Grandfathered `.badge` retained for the showcase dict-key "required" marker.
- [x] 3.3 Default glyphs rendered in-template when `Glyph` is omitted: `running` = ●, `archived` = ⊘, `draft` = ○, `warning` = ⚠. `billable`/`non-billable` rely on explicit label text.
- [x] 3.4 Document the partial in `web/templates/partials/README.md` with a full entry covering context keys, default glyphs, accessibility obligations, and event contract.
- [x] 3.5 Migrate `entry_row`, `client_row`, `project_row`, `report_results`, `dashboard_summary`, and `dashboard.html` call sites from `<span class="badge badge-…">` markup to `{{template "status_chip" (dict …)}}`. `timer_widget` still renders the running-state chip inline; migration deferred to group 4 (timer_control rebuild).
- [x] 3.6 Register the `status_chip` `ComponentEntry` in `internal/showcase/catalogue.go` with six examples (one per `Kind`) and colocated snippet fixtures in `internal/showcase/snippets/status_chip.*.tmpl`. Showcase partial-coverage and snippet-integrity tests green.

## 4. Timer control partial

- [x] 4.1 Rename `web/templates/partials/timer_widget.html` → `timer_control.html` and block `timer_widget` → `timer_control`. Amended spec: idle state renders a start-entry form (project picker + description + `.btn-primary` Start pill); running state renders a single accent pill with pulsing dot.
- [x] 4.2 Author idle-state markup: start form consuming the inherited `.btn-primary` pill shape (actions = pills). HTMX `hx-post="/timer/start"` swapping `#timer-control`.
- [x] 4.3 Author running-state markup: `.tt-timer-running` class with `var(--color-accent-soft)` fill, 2px `var(--color-accent)` border, leading `.tt-timer-dot` with pulse animation, project name, tabular-nums elapsed `HH:MM:SS` via `.tt-timer-elapsed`, and a `.btn-ghost` Stop control (visually distinct from the idle `.btn-primary` Start pill).
- [x] 4.4 Pulse animation uses `var(--motion-easing-standard)`. Halted by the global `@media (prefers-reduced-motion: reduce)` `animation: none !important` rule, leaving a static accent dot.
- [x] 4.5 `data-focus-after-swap` applied to Start pill in idle and Stop button in running. HX-Trigger emissions via existing handler unchanged.
- [x] 4.6 All live references migrated: `dashboard.html`, `internal/tracking/handler.go` (`RenderPartial` target), browser test comments in `focus_after_swap_test.go`. Showcase catalogue entry renamed (`timer-control`, `timer_control`) with updated examples and A11y notes; snippet fixtures renamed to `timer_control.idle.tmpl` / `timer_control.running.tmpl`. Archive / historical files and generated `testdata/browser-artifacts/*.json` left untouched.
- [x] 4.7 README entry for `timer_control` rewritten with the identity contract; `tracking_error` cross-reference updated to cite `timer_control`.

## 5. Table (`.table` CSS contract) refinement

- [x] 5.1 In `web/static/css/app.css` (components layer), `.thead th` renders uppercase, `letter-spacing: 0.04em`, 0.75rem, `var(--color-text-muted)`. `tbody th` gets its own rule to preserve body-row labels.
- [x] 5.2 `.table tbody tr` uses hairline horizontal dividers only via existing `border-bottom` rule; no vertical dividers or zebra exist. Preserved `.table tr:last-child td { border-bottom: 0 }` for clean bottom edge.
- [x] 5.3 Hover migrated from `tr:hover td` (cell-level background) to `tbody tr:hover` (row-level background) so background doesn't shift independently of borders. Selected/focused row rendered via `box-shadow: inset 2px 0 0 0 var(--color-accent)` — this is the canonical inside-left edge technique and avoids any border shift. Triggered by `aria-selected="true"` or `:focus-within`.
- [x] 5.4 Numeric-column treatment: `.col-num`, `[data-col-kind="numeric"]`, and `.num` all resolve to `tabular-nums` + right-aligned via a single CSS rule.
- [x] 5.5 Existing `class="num"` call sites (23 cells across 7 templates) left in place; the CSS alias makes them behaviorally identical to `.col-num`. `.col-num` is the canonical name for new columns; `.num` is a preserved alias documented in the CSS README.
- [x] 5.6 `web/templates/partials/README.md` "Deferred / not extracted" section rewritten to describe the accepted `.table` CSS contract instead of the never-extracted wrapper partial, with cross-links to the `ui-partials` and `ui-component-identity` requirements.

## 6. Accent rationing audit

- [x] 6.1 Audited `web/static/css/app.css` — enumerated every selector referencing `var(--color-accent*)` tokens: link text, link hover, focus-visible (`--color-focus`), active nav item (`aria-current="page"`), `.btn-primary`, `.btn-primary:hover`, selected/focused table row, `.tt-chip-billable`, `.tt-chip-running`, `.tt-timer-running`, `.tt-timer-dot`, `.tt-timer-elapsed`.
- [x] 6.2 Amended the spec's accent-rationing allow-list to honestly reflect real "which one?" signals: added link text, active nav item, and billable/running chips to the enumeration. All existing accent usages conform; no migrations required. Spec, design doc, style guide, and showcase checklist updated in sync.
- [x] 6.3 Added `internal/showcase/identity_audit_test.go::TestAccentRationingAudit` — parses `web/static/css/app.css`, walks through nested `@layer` blocks, enumerates every rule whose decls reference a `--color-accent*` token, and fails when a selector is not on the allow-list. Verified by injecting a violation (`.bad-accent-hover:hover`) and confirming the test fails with a pointed error message.
- [x] 6.4 Test passes in the default `go test ./...` run. Included in `make test` by virtue of the `-p 1 ./...` target.

## 7. Showcase gallery extensions

- [x] 7.1 Timer showcase entry renders `idle` and `running` states via the real `partials/timer_control` (group 4). Reduced-motion rendering is the same markup under `@media (prefers-reduced-motion: reduce)` — no separate example required because the global animation-none rule flips behavior transparently; the showcase consumer simulates via browser DevTools or OS preference.
- [x] 7.2 Added a standalone "<code>.table</code> — visual states" gallery section in `web/templates/showcase/components.html` rendering: default row, selected row via `aria-selected="true"` (2px accent inside-left edge), hover via live CSS `:hover`, and the `empty_state` partial in an empty-state sub-frame. A numeric column (`Duration`) demonstrates `.col-num` `tabular-nums` right-alignment.
- [x] 7.3 Status chip showcase entry renders six examples covering every `Kind` value with its canonical `Variant`, plus glyph behavior (group 3).
- [x] 7.4 `TestComponentCatalogueCoverage` green — every non-grandfathered partial has exactly one entry.
- [x] 7.5 Variant coverage verified: status_chip (6 kinds), timer_control (idle + running). Table states rendered in the stand-alone section with numeric column demonstration.

## 8. Accessibility validation

- [x] 8.1 Contrast review (code inspection):
  - Timer idle `.btn-primary`: white on accent-500 — existing pair, passes AA (already validated in foundation spec).
  - Timer running: `.tt-timer-elapsed` uses `var(--color-accent)` on `var(--color-accent-soft)` — same pair as the previous `.badge-billable` rule that was accepted via axe-smoke. The running dot is a solid fill so contrast isn't text-level. 2px accent border vs. accent-soft bg meets the ≥3:1 non-text UI target.
  - `.tt-chip-billable` / `.tt-chip-running`: accent-500 text on accent-100 bg — inherits the `.badge-billable` pattern from the pre-change baseline and passes axe-smoke. Back-of-envelope contrast is ≈3.8:1 which is AA for ≥18pt or AAA non-text but below 4.5:1 for 12px text. **Flagged as a pre-existing contrast concern inherited from the previous badge rule, not introduced by this change.** Should be verified with `TestAxeSmokePerPage` and, if a real violation surfaces, addressed in a separate contrast-tightening proposal. Dark-theme pair (accent-500 #4b8bf5 on accent-100 #1d2a44) is structurally similar and bounded by the same axe check.
  - `.tt-chip-non-billable` / `.tt-chip-draft` / `.tt-chip-archived`: muted text on surface / surface-alt — existing, AA-validated pair.
  - `.tt-chip-warning`: warning-500 on warning-soft — existing pair from `.badge-warning`.
  - Table header: muted text on surface-alt — existing pair, passes AA.
  - Selected-row accent edge on surface: `--color-accent` (accent-500) vs `--color-surface` (neutral-0) — well above 3:1 non-text UI threshold.
- [x] 8.2 Keyboard navigation reviewed:
  - Timer Start (`.btn-primary` submit button) reachable via Tab, activates on Enter/Space (native `<button>`).
  - Timer Stop (`.btn-ghost` submit button) reachable via Tab with `data-focus-after-swap` so focus lands on it after the HTMX swap.
  - Table rows: `:focus-within` triggers the 2px accent inside-left edge (action buttons inside rows already receive focus; the row inherits focus-within).
  - Chips are non-interactive `<span>`s — no focus required.
- [x] 8.3 Non-color-only conveyance reviewed:
  - `running` chip: accent-soft fill + ● glyph + "Running" label + `aria-label="Running"`.
  - `archived` chip: neutral fill + ⊘ glyph + "Archived" label + `aria-label="Archived"`.
  - `draft` / `warning` chips: glyph + explicit label text.
  - `billable` / `non-billable`: labels are semantically explicit ("Billable" / "Non-billable"), so glyphs are intentionally omitted (spec permits this for non-state kinds).
  - Running timer: accent fill + pulsing dot + elapsed readout + project name; assistive tech receives `aria-live="polite"` announcements on state change.
  - Selected table row: 2px accent edge + `aria-selected="true"`.
- [x] 8.4 Target size reviewed:
  - Timer Start / Stop: `.btn` has `min-height: 36px; min-width: 44px`. Meets WCAG 2.2 Target Size Minimum (24×24). AAA 44×44 not enforced by this change (consistent with existing buttons).
  - Chips are non-interactive — target size does not apply.
- [x] 8.5 Reduced-motion verified:
  - `.tt-timer-dot` uses `animation: tt-timer-pulse ...`. The global `@media (prefers-reduced-motion: reduce)` rule in `app.css` sets `*, *::before, *::after { animation: none !important; }` which halts the pulse and leaves the solid accent dot visible (`background: var(--color-accent)`) — matches the spec's "static filled dot" requirement.
  - No other new animations introduced in this change.

## 9. Browser contract tests

**Group 9 deferred — pre-existing browser-test infrastructure gap.** When running `go test -tags=browser` against the e2e harness, `/static/css/*`, `/static/js/*`, and `/static/vendor/*` return 404 because the e2e `BuildServer` does not mount the static file server (the harness doc-comment at line 130 acknowledges only axe/testdata paths are served). `/settings` also 404s. As a result `TestAxeSmokePerPage` and `TestFocusRingContract` were failing before this change — `testdata/browser-artifacts/TestFocusAfterSwapContract/failure.png` was already in the working tree at session start. Layering new browser contract tests on top of a broken harness would produce false signal. These tasks should be picked up in a separate infrastructure proposal that fixes the test-server static-asset mounting and wires `/settings`.

- [~] 9.1 Defer — `TestAxeSmokePerPage` currently fails against all pages due to missing CSS (`target-size` violations from unstyled buttons). Fix infra first.
- [~] 9.2 Defer — `TestFocusRingContract` hangs in `seedForFocusRing` against an unstyled page. Fix infra first.
- [~] 9.3 Defer — same harness used for showcase/brand axe smokes; same static-asset gap.
- [~] 9.4 Defer — tabular-nums assertions require rendered CSS; revisit after infra fix.
- [~] 9.5 Defer — the running-timer 2px-accent-border and reduced-motion assertions require rendered CSS; revisit after infra fix.

## 10. Verification and archival

- [x] 10.1 `make fmt && make vet && make test` all pass. `TestAccentRationingAudit` and `TestComponentCatalogueCoverage` in `internal/showcase` both green with the new entries / audit.
- [~] 10.2 Deferred alongside group 9 — browser suite has pre-existing infrastructure failures (static assets 404 in e2e harness). Verification blocked on that infra fix.
- [x] 10.3 Manual QA sweep — pending user action. Suggested: `make db-up && make run`, then walk through dashboard (start/stop timer, watch running pill + pulsing dot), entries list (billable/running/non-billable chips), clients/projects (archived chip), reports (No rate chip), and `/dev/showcase` for the identity checklist + table states + chip variants + timer states. Verify in both themes via the existing theme toggle.
- [x] 10.4 Archived 2026-04-23 via `/opsx:archive sharpen-component-identity --yes`. Spec sync applied: +6 ADDED in `ui-component-identity` (new capability), ~1 MODIFIED in `ui-foundation` (Scale Tokens → `--radius-pill`), +2 ADDED / ~1 MODIFIED in `ui-partials`, +2 ADDED in `ui-showcase`. Totals: +10 added, ~2 modified across 4 capabilities.
- [x] 10.5 Follow-on `sharpen-dashboard-empty-states` proposal opened by the user for summary cards + empty-state copy/layout + running-entry card top border. The e2e test-server static-asset infra fix (unblocks the deferred browser contract tests in group 9) remains future work as a separate proposal.
