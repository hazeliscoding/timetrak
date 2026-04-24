## Context

TimeTrak ships a working but visually ungainly theme toggle today. The pieces:

- **Tokens** — `tokens.css` has `:root` (light), `[data-theme="dark"]`, and `@media (prefers-color-scheme: dark) { [data-theme="system"] { ... } }`. The token split is correct; it produces consistent light, dark, and system-following surfaces.
- **Control** — three `.btn-ghost` buttons in `layouts/app.html:9-11` with `data-theme-set="light|dark|system"` and `aria-pressed` wired by `app.js`.
- **JS** — `app.js:3-27` reads `localStorage.timetrak.theme` on DOMContentLoaded, applies to `<html>` via `data-theme`, toggles `aria-pressed`, and handles clicks by persisting + re-applying.
- **Flash** — `base.html:2` sets `data-theme="system"` as the initial attribute. If the user's stored preference is `dark`, the document paints with `system` (which may be light) for the brief window between parse and script execution.

This change does not redesign the token layer or the core behavior. It consolidates the three-button surface into one segmented control (the standard pattern for a three-way enum setting), eliminates the first-paint flash, and threads the control through the sharpened component-identity contracts — accent rationing, shape language, focus ring.

## Goals / Non-Goals

**Goals:**

- Replace the three loose buttons with one cohesive segmented control that occupies one header slot instead of three.
- Eliminate the flash of default-theme paint when a non-default stored preference exists.
- Sharpen the *selected-segment* visual so the user's current choice is unambiguously signaled via the project's two-weight border + accent-soft fill idiom.
- Keep the existing `data-theme-set` + localStorage + `data-theme` plumbing exactly as-is. The new control is an extraction, not a rewrite.

**Non-Goals:**

- Per-user or per-workspace backend persistence. Out of scope; localStorage remains the sole persistence layer.
- Any token change, any new color, any new radius scale. The two-weight border and accent-rationing contracts from `sharpen-component-identity` govern this component.
- Automatic scheduled switching (time-of-day, location). Different design problem.
- Icons from a library or SVG sprites. Unicode glyphs only — consistent with the rest of the product (e.g. `●` on running timer, `⊘` on archived chip).
- Reduced-motion handling unique to this control. The component has no animations; the global `prefers-reduced-motion` rule catches anything inherited.

## Decisions

### D1. Segmented control with three segments, not a cycle button or dropdown

**Chosen:** A `<div role="radiogroup" aria-label="Theme">` wrapper with three `<button type="button" role="radio">` segments. Each segment shows an icon + visible label (label becomes `sr-only` under a breakpoint). Selected segment has `aria-pressed="true"` + `aria-checked="true"`.

**Alternatives considered:**

- *Single cycle button.* One button that cycles Light → Dark → System → Light with a label showing the current state. Rejected because it hides two of the three options behind a click each, violating the "calm tool" heuristic — a user scanning for "where is dark mode?" should see it immediately.
- *Icon + dropdown.* Compact, but adds a modal/menu surface that would need its own focus contract. Overkill for a 3-option enum.
- *Two-state (light/dark) toggle with implicit system default.* Rejected because "System" is a real, valuable option (ships an explicit preference that follows the OS), and hiding it costs discoverability for no real space savings.

### D2. ARIA contract — `radiogroup` + `role="radio"`, with both `aria-pressed` and `aria-checked`

**Chosen:** The wrapper is `role="radiogroup"` with `aria-label="Theme"`. Each segment is a `<button>` carrying `role="radio"`, `aria-checked="true|false"`, and `aria-pressed="true|false"` (the latter for the existing JS hook + for backwards-compat with a11y tests that already assert on `aria-pressed`).

**Why both:** the existing `app.js` code reads `aria-pressed` to synchronize state across segments. Keeping `aria-pressed` lets the existing JS work unchanged. Adding `aria-checked` is the more semantically correct signal for a radio-group member. They do not conflict.

**Alternative considered:** a `<fieldset>` with three `<input type="radio">` children. Rejected because real radios submit with a form and the theme toggle must not round-trip to the server.

### D3. FOUC prevention via a synchronous inline head-script

**Chosen:** Add ~15 lines of inline `<script>` at the top of `<head>` in `base.html` (before any `<link rel="stylesheet">`):

```html
<script>
  (function () {
    try {
      var t = localStorage.getItem("timetrak.theme") || "system";
      document.documentElement.setAttribute("data-theme", t);
    } catch (e) {
      // Private mode / denied: fall through with default data-theme.
    }
  })();
</script>
```

Why synchronous and inline:

- Parser reaches the script before any stylesheet is applied, so `data-theme` is set before first paint.
- An external script would need network + parse time, re-introducing FOUC.
- A `<script async>` or `<script defer>` misses the first-paint window.

**Trade-off:** one inline script. This is the ONLY sanctioned inline script in the product; the `ui-foundation` spec delta codifies that. The `try` guard prevents localStorage-denied browsers (private mode in some configurations) from breaking render.

### D4. Selected-segment styling consumes accent — allow-list amendment required

**Chosen:** The selected segment renders with `var(--color-accent-soft)` background, `var(--color-accent)` text color, and a 2px `var(--color-accent)` inset border on the segment (matching the two-weight contract's "state" rule). Unselected segments are neutral with 1px borders.

**Why accent is appropriate here:** the theme switch answers "which one?" — which theme is active. That's the exact principle governing the current allow-list entries (primary button, active nav, selected row, billable/running chips). Adding `.tt-theme-switch [aria-pressed="true"]` to the allow-list is consistent with the principle, not an exception to it.

**Amendment:** the `ui-component-identity` "Accent rationing" requirement's enumerated allow-list grows from 8 to 9 items. The CI audit test's allow-list slice also grows by one line.

### D5. Icons via Unicode, not SVG sprites or an icon library

**Chosen:** `☀` (Light), `☾` (Dark), `⌘` (System). Rendered as a `.tt-theme-switch-glyph` span with `aria-hidden="true"`. The visible label (or `sr-only` label on narrow viewports) carries the accessible name.

**Why Unicode:** consistent with the project's existing precedent (`●` running-timer dot, `⊘` archived chip, `⚠` warning chip, `○` draft chip). Adding an SVG sprite system for three glyphs would be scope creep and a new dependency on maintenance discipline.

**Trade-off:** glyph rendering varies slightly across OS/font stacks. Acceptable — the visible text label is always present, and the glyph is supplementary under WCAG "never color/icon-only."

### D6. Partial lives at `web/templates/partials/theme_switch.html`, block name `theme_switch`

**Chosen:** One canonical partial following the project's convention (bare block name, file in `partials/`, invoked via `{{template "theme_switch" .}}`). Context is empty — the partial reads no runtime data. `base.html`'s inline script + `app.js` hold all state.

**Why a partial for one call site:** the partial's real consumer is also the **showcase catalogue** (`internal/showcase/catalogue.go` will render three live examples, one per selected state). Extracting the partial means the showcase renders the *actual* control, not a re-implementation, which is the `ui-showcase` spec's explicit rule.

### D7. Selected-state demo in showcase uses inline `data-theme-set` overrides on the snippet

**Chosen:** Three showcase examples (`light-selected`, `dark-selected`, `system-selected`). Each renders the real `partials/theme_switch` with a per-example `InitialSelected` dict key that pre-sets `aria-pressed` / `aria-checked` / `data-active` on the matching segment so reviewers see the three selected states side-by-side without toggling. The showcase page does NOT call the actual localStorage/JS path — it just renders the static HTML with the attributes set.

## Risks / Trade-offs

- **[Risk]** Inline head-script is the first inline script in the app, breaking a tidy "no inline JS" convention. → *Mitigation:* the `ui-foundation` delta codifies this as the **single sanctioned exception**; any future attempt to add another inline script fails review by citing the spec.
- **[Risk]** Unicode glyph rendering inconsistency across platforms. → *Mitigation:* labels carry meaning; the glyph is `aria-hidden` supplementary.
- **[Risk]** The segmented control competes with the workspace switcher + sign-out for header space on narrow viewports. → *Mitigation:* the `sr-only` label collapse under a breakpoint shrinks the control to three icon-only segments, recovering ~80px; a second follow-up could move all three header controls into a hamburger menu, but that's out of scope.
- **[Risk]** A user with localStorage denied (strict privacy mode) sees the default `system` theme on every load, with no persistence. → *Mitigation:* acceptable; the `try { ... } catch` prevents render errors, and `system` is a reasonable default for a user who refuses persistence.
- **[Trade-off]** `aria-pressed` + `aria-checked` redundancy. Kept for compatibility with existing JS hooks; removing `aria-pressed` is a follow-up if the JS is ever rewritten to use `aria-checked` as its source of truth.
- **[Trade-off]** Allow-list grows from 8 to 9 entries. Small, principled; each entry answers "which one?"

## Migration Plan

- Land partial + CSS + template swap + head-script + allow-list amendment + showcase entry + spec deltas together in one PR. No feature flag, no phased rollout — the old three-button markup is deleted in the same commit the new partial lands.
- No data migration. No user action required.
- Rollback: revert the commit. Storage key and CSS tokens are unchanged.

## Open Questions

- Whether to auto-shrink the visible label to `sr-only` via a container query or a fixed breakpoint. Leaning fixed breakpoint (`@media (max-width: 720px)`) because container queries are overkill for a single-known-container control. Resolved during implementation; captured in CSS.
- Whether the showcase entry should toggle *live* (clicking actually changes theme) or show only static states. Leaning static — the live toggle is available in the actual header on the same page, and the showcase's job is to document states side-by-side, not re-demonstrate the click path. Resolved during implementation.
