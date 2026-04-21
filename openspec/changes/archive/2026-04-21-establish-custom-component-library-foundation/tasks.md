## 1. Audit & Inventory

- [x] 1.1 Walk `web/static/css/tokens.css` and list every current token, grouping by concern (colour, spacing, radius, typography). Note gaps against the target semantic alias set defined in the spec.
- [x] 1.2 Walk `web/static/css/app.css` and grep for raw colour values, raw radii, raw shadows, and raw pixel spacing in component selectors. Produce a short audit list (file, selector, raw value, proposed token) inside the design doc or as a PR comment.
- [x] 1.3 Grep component CSS for ad-hoc `.disabled`, `.active`, `.error` class names and other non-`is-` state classes. Note any that block the authoring-contract adoption.
- [x] 1.4 Confirm no existing partial in `web/templates/partials/` references CSS classes that would break under the planned token rename (rename is CSS-only, but verify).

## 2. Token Taxonomy (tokens.css)

- [x] 2.1 Add primitive ramps at the top of `tokens.css` under a clear comment block: `--neutral-0`…`--neutral-900`, `--accent-50`…`--accent-900`, and red/amber/green severity anchors for both light and dark themes.
- [x] 2.2 Add semantic aliases resolving to primitives: `--color-bg`, `--color-surface`, `--color-surface-alt`, `--color-text`, `--color-text-muted`, `--color-border`, `--color-border-strong`, `--color-accent`, `--color-accent-hover`, `--color-accent-soft`, `--color-focus`, severity pairs `--color-success`/`--color-success-soft`, `--color-warning`/`--color-warning-soft`, `--color-danger`/`--color-danger-soft`, `--color-info`/`--color-info-soft`.
- [x] 2.3 Redefine legacy token names (`--surface`, `--text`, `--border`, `--accent`, `--focus`, `--success`, `--warning`, `--danger`, the `*-soft` siblings) as `var(--color-*)` deprecation aliases, each with a `/* deprecated: remove next foundation change */` comment.
- [x] 2.4 Mirror the full two-layer + alias set in `[data-theme="dark"]` and in the `@media (prefers-color-scheme: dark) [data-theme="system"]` branch.
- [x] 2.5 Add scale tokens where missing: `--motion-duration-fast`, `--motion-duration-normal`, `--motion-easing-standard`, `--shadow-none`, `--shadow-sm`, `--shadow-md`, `--z-base`, `--z-sticky`, `--z-dropdown`, `--z-modal`, `--z-toast`, `--bp-sm`, `--bp-md`, `--bp-lg`. Keep the existing numeric `--space-*` and `--radius-*` scales as-is.
- [x] 2.6 Inline-comment each semantic colour alias with its documented contrast role (surface pairing, contrast ratio target) so future reviewers can audit without re-deriving.

## 3. CSS Layer Order (app.css)

- [x] 3.1 Declare `@layer reset, tokens, base, components, utilities, overrides;` at the top of `app.css`.
- [x] 3.2 Wrap the existing box-sizing / html/body reset rules in `@layer reset { ... }`.
- [x] 3.3 Wrap body typography, heading rules, `a`, `:focus-visible`, `.sr-only`, `.muted`, `.tabular` base rules in `@layer base { ... }` (and move the true utilities to `utilities`, see 3.5).
- [x] 3.4 Wrap `.app-shell`, `.app-header`, `.app-sidebar`, `.app-main`, `.nav`, `.btn`, `.btn-*`, `.field`, `.table`, `.card`, `.badge`, `.badge-*`, `.flash`, `.flash-*`, `.timer`, `.empty` in `@layer components { ... }`.
- [x] 3.5 Wrap `.num`, `.stack`, `.row`, `.row-between`, `.mt-0`, `.mb-0` in `@layer utilities { ... }`. Declare an empty `@layer overrides { }` for future hot-fixes.
- [x] 3.6 Keep the `@media (prefers-reduced-motion: reduce)` block outside the layer cascade (documented exception); verify it still flattens all transitions/animations to instant after the rewrite.

## 4. Component Re-Point (app.css)

- [x] 4.1 Mechanically replace `var(--surface)` → `var(--color-surface)`, `var(--surface-alt)` → `var(--color-surface-alt)`, `var(--text)` → `var(--color-text)`, `var(--text-muted)` → `var(--color-text-muted)`, `var(--border)` → `var(--color-border)`, `var(--border-strong)` → `var(--color-border-strong)`, `var(--accent*)` → `var(--color-accent*)`, `var(--focus)` → `var(--color-focus)`, `var(--success*)` / `var(--warning*)` / `var(--danger*)` → their `--color-*` equivalents throughout `app.css`.
- [x] 4.2 Verify every component selector now references only semantic aliases (no primitive ramp references outside `tokens.css`, no raw hex values).
- [x] 4.3 Replace any raw `box-shadow` values in component rules with `var(--shadow-*)` (cards use `--shadow-none` + border; modals/dropdowns future-only).
- [x] 4.4 Replace any raw motion durations or easings with the new motion tokens.
- [x] 4.5 Verify the focus-ring rule in `base` is `outline: 3px solid var(--color-focus); outline-offset: 2px;` and is the only `:focus-visible` rule in the stylesheet.

## 5. Authoring Contract Documentation

- [x] 5.1 Create `web/static/css/README.md`. Document: two-layer token model, enumerated semantic alias list, scale token set, `@layer` order, `tt-<component>` naming convention for new components, legacy-selector grandfathering, `is-<state>` vs ARIA-attribute state rules, variant vocabulary (`primary` / `secondary` / `ghost` / `danger`), size-scale policy (`sm`/`md`/`lg` only when justified), focus-indicator rule, 24×24 target-size rule, status-never-colour-alone rule, deprecation-alias rule, and the contribution process for adding a new token or component.
- [x] 5.2 In the README, explicitly state: until `docs/timetrak_ui_style_guide.md` is updated in a follow-up change, the CSS tokens + this README + `openspec/specs/ui-foundation/spec.md` are authoritative.
- [x] 5.3 Cross-link `web/templates/partials/README.md` ↔ `web/static/css/README.md` so contributors land on the right reference.
- [x] 5.4 Add a short "Authoring a new component" checklist in the CSS README: name with `tt-` prefix, declare in the `components` layer, consume only semantic aliases and scale tokens, use ARIA / `is-` for state, document variants used, verify focus visibility + 24×24 target size + reduced motion.

## 6. Accessibility Validation

- [x] 6.1 Contrast-check every semantic colour pair documented in the spec against its surface in both light and dark themes. Target 4.5:1 for body text pairings (`--color-text` on `--color-surface`, `--color-surface-alt`, `--color-bg`), 3:1 for non-text / large-text pairings and borders, 3:1 for `--color-focus` against every surface it can appear on including `--color-accent`. Record the values.
- [x] 6.2 If any pair fails, adjust the primitive ramp step the alias resolves to (not the alias itself) and re-measure. Do not introduce a bespoke one-off colour.
- [ ] 6.3 Visually verify focus ring on every interactive primitive (buttons, links, inputs, selects, table row actions, timer controls, nav items) in light and dark themes using keyboard Tab.
- [ ] 6.4 Verify `@media (prefers-reduced-motion: reduce)` still collapses motion: toggle the OS setting (or use devtools emulation) and confirm hover / state transitions render instantly.
- [x] 6.5 Spot-check that no badge, flash, or status indicator in the running app conveys meaning with colour alone (each carries text or icon alongside colour).
- [ ] 6.6 Keyboard walk: Tab through dashboard, Time entries, Clients, Projects, Rates, Reports. Confirm focus stays visible on every control and `data-focus-after-swap` targets still land focus correctly (regression check only — no contract change here).

## 7. Final Verification

- [x] 7.1 `make fmt` and `make vet` pass (no Go changes expected, but sanity).
- [ ] 7.2 `make run` renders every page with no visible regressions against a pre-change screenshot comparison (manual side-by-side for dashboard, time entries, a client page, a project page, rates, reports, settings, login, signup).
- [x] 7.3 Grep `web/static/css/app.css` for any surviving raw hex, raw `rgb(`, or raw pixel value in a component rule; each must resolve to a token or be explicitly justified in a code comment.
- [x] 7.4 Confirm `openspec/specs/ui-foundation/spec.md` (added by this change) validates via `openspec validate`.
- [x] 7.5 Open the follow-up tracking item to update `docs/timetrak_ui_style_guide.md` to cite the codified tokens (separate change, out of scope here).
