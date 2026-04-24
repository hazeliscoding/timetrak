## Why

TimeTrak already supports three themes — `light`, `dark`, and `system` — at the CSS token layer (`web/static/css/tokens.css`) and ships working toggle JS (`web/static/js/app.js`). What's missing is a **crafted, single-surface control** that honors the `sharpen-component-identity` contract.

Today the header carries three loose `.btn-ghost` buttons labeled "Light" / "Dark" / "System" (`web/templates/layouts/app.html:9-11`). They function, but they:

1. **Take three header slots for what is conceptually one setting.** The header is real estate — `brandmark`, workspace switcher, status region, theme, sign-out — and three buttons for a three-way toggle reads as disorganized.
2. **Don't signal "which one" in the sharpened visual grammar.** `aria-pressed="true"` currently gets default `.btn-ghost:hover` treatment when active — the user's current choice is indistinguishable at a glance from "this button is hovered."
3. **Flash the wrong theme on first paint.** The theme script runs at DOMContentLoaded, so a user whose stored preference is `dark` sees a brief flash of the light default while the document parses. This is the canonical client-side-theme flash-of-unstyled-content (FOUC) problem.
4. **Lack icons.** Three text buttons with identical visual weight feel undifferentiated.

This change consolidates the three buttons into **one segmented `tt-theme-switch` control** with icons + labels, sharpens the selected-state styling against the component-identity contract, and eliminates the first-paint flash via a synchronous head-script. No backend persistence; no schema changes; no new dependencies.

## What Changes

- **Replace the three loose `.btn-ghost` buttons** in `web/templates/layouts/app.html` with a single `partials/theme_switch` component. The partial renders three linked button-like `<button type="button" role="radio">` controls inside a `<div role="radiogroup" aria-label="Theme">` wrapper, styled as one segmented control.
- **Introduce a `.tt-theme-switch` CSS component** in `web/static/css/app.css` (components layer). It renders as a single rounded pill divided into three segments, 1px border around the group, segment dividers at 1px, the *selected* segment filled with `var(--color-accent-soft)` + `var(--color-accent)` text (consistent with `.nav a[aria-current="page"]`), and a 2px accent edge on the selected segment's start/end.
- **Icons via unicode glyphs.** Light = `☀` (U+2600), Dark = `☾` (U+263E), System = `⌘` (U+2318). Each glyph is paired with a visible label on wider viewports; on a narrow viewport the label becomes `sr-only` so the control shrinks gracefully.
- **Eliminate FOUC** by adding a synchronous inline script at the top of `<head>` (`web/templates/layouts/base.html`) that reads `localStorage.timetrak.theme` and sets `data-theme` on `<html>` before the document body parses. The existing `app.js` deferred listener still handles clicks + sync. The inline script is small (~15 lines), non-blocking in practice (synchronous head script that reads localStorage is fast), and documented in `web/static/css/README.md`.
- **Amend the accent-rationing allow-list** in `openspec/specs/ui-component-identity/spec.md` and the corresponding `internal/showcase/identity_audit_test.go` allow-list to include `.tt-theme-switch [aria-pressed="true"]` as a legitimate "which one?" consumer of accent. The theme switch is structurally parallel to the already-allow-listed `.nav a[aria-current="page"]` — both say "this option is active."
- **Showcase** gains a `theme_switch` entry in the components catalogue (default render, each of the three selected states), so the control is reviewable alongside other sharpened components.
- **Out of scope (explicit):**
  - Per-user or per-workspace backend theme persistence. The current reality is localStorage-scoped; Stage 3's solo-freelancer assumption means cross-device sync is not a user pain point worth the auth-schema churn. Noted as a follow-up if real cross-device friction surfaces.
  - Scheduled auto-switching (time-of-day, geolocation, etc.). Would require design decisions beyond this sharpening pass.
  - Contrast audit — separate issue. The sibling accent-rationing work already surfaced real contrast concerns (chip text at ~3.8:1); those are tracked in the deferred `fix-browser-test-suite-contract-failures` proposal.
  - Any change to the tokens themselves (`--color-*`, ramps). The existing three-theme token layer is accepted baseline.
  - Any new JavaScript dependency or framework. The existing ~15 lines of theme JS stay, plus ~15 new lines of head-script for FOUC.

## Capabilities

### New Capabilities

- *None.* The scope lives inside `ui-partials` (new `theme_switch` partial), `ui-foundation` (the small head-script contract — framed as a scale/CSS-layer-adjacent rule), `ui-component-identity` (allow-list amendment), and `ui-showcase` (catalogue entry). A new capability would be overfit for one component and one 15-line initialization script.

### Modified Capabilities

- `ui-partials` — ADDS a new canonical `partials/theme_switch` partial with enumerated context keys and accessibility obligations (`role="radiogroup"`, per-segment `aria-pressed`, visible keyboard focus).
- `ui-foundation` — ADDS a small requirement codifying the FOUC-prevention inline head-script: the script MUST read `localStorage.timetrak.theme`, set `data-theme` on `<html>` synchronously before any stylesheet paints, and be small enough (≤30 lines) that it can live inline in `base.html`. This is the only sanctioned inline `<script>` in the app.
- `ui-component-identity` — MODIFIES the "Accent rationing" requirement's enumerated allow-list to include the theme-switch selected-segment surface. The justification is that the theme switch is a "which one?" signal structurally parallel to the already-allow-listed active navigation item.
- `ui-showcase` — ADDS a `theme_switch` entry to the components catalogue (one example per selected variant: `light`, `dark`, `system`).

## Impact

- **Templates modified:** `web/templates/layouts/app.html` (swap three buttons for one `theme_switch` invocation), `web/templates/layouts/base.html` (add inline head-script), `web/templates/partials/README.md` (new entry).
- **Templates added:** `web/templates/partials/theme_switch.html`.
- **CSS:** `web/static/css/app.css` gains the `.tt-theme-switch` component rules. No new tokens. No new semantic aliases.
- **JS:** `web/static/js/app.js` stays (the click handler already toggles via `data-theme-set`; the partial emits the same hook so the existing listener works unchanged). The FOUC head-script is ~15 new lines of inline JS in `base.html`.
- **Audit test:** `internal/showcase/identity_audit_test.go` allow-list gains one line for `.tt-theme-switch [aria-pressed="true"]`. Spec comment updated to match.
- **Showcase:** `internal/showcase/catalogue.go` gains a `theme_switch` `ComponentEntry` with three `Example`s; three new snippet fixtures under `internal/showcase/snippets/theme_switch.*.tmpl`.
- **No backend, no DB, no new dependency, no migration.**
- **Risk:** the head-script introduces a tiny synchronous read from localStorage on every page load. Measured cost is sub-millisecond; the alternative (accepting FOUC) is worse UX. Documented as the single sanctioned inline script — the `ui-foundation` requirement explicitly forbids any other inline JS.
- **Follow-ups:** per-user backend theme persistence if cross-device sync becomes a real pain point; a compact "icon-only" variant of the control for further header consolidation if the segmented form takes more space than design tolerates.
