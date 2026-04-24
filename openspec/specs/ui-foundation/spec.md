# ui-foundation Specification

## Purpose
The ui-foundation capability defines the design-system primitives every
TimeTrak component is built on: a strict two-layer token taxonomy
(private primitive ramps fronted by a bounded set of public semantic
aliases), scale tokens for spacing, radius, type, and motion, a
cascade-aware CSS layer order, and the component authoring contract
that forbids primitive or raw values in component CSS. It also enforces
the visible focus-indicator contract in both light and dark themes, the
invariant that status is never conveyed by color alone, the
deprecation-and-migration rule for evolving tokens, and the
authoring-contract documentation that makes these rules reviewable at
PR time.
## Requirements
### Requirement: Two-Layer Token Taxonomy

TimeTrak's CSS token system SHALL be organized into two layers: **primitive ramps** (palette anchors, private to the foundation) and **semantic aliases** (public tokens consumed by components). Components SHALL consume only semantic aliases, never primitive ramps or raw values.

The primitive ramp layer MUST include at minimum: a neutral ramp (`--neutral-0` through `--neutral-900`), an accent ramp (`--accent-50` through `--accent-900`), and severity anchors for red, amber, and green (at least a 500-weight step per hue for both light and dark themes).

The semantic alias layer MUST include at minimum: `--color-bg`, `--color-surface`, `--color-surface-alt`, `--color-text`, `--color-text-muted`, `--color-border`, `--color-border-strong`, `--color-accent`, `--color-accent-hover`, `--color-accent-soft`, `--color-focus`, and the severity pairs `--color-success` / `--color-success-soft`, `--color-warning` / `--color-warning-soft`, `--color-danger` / `--color-danger-soft`, `--color-info` / `--color-info-soft`.

New semantic aliases MUST NOT be added without a change proposal that extends this requirement. Component-scoped tokens (e.g. a hypothetical `--btn-primary-bg`) are NOT part of the public semantic layer and, if introduced, MUST be declared locally within the component's CSS scope rather than in the global token file.

#### Scenario: Component references a semantic alias

- **WHEN** a component CSS rule needs a surface colour
- **THEN** it references `var(--color-surface)` (or another documented semantic alias) and does NOT reference `var(--neutral-0)` or a raw hex value.

#### Scenario: Token file is rebuilt from primitives

- **WHEN** the token file is re-generated or reviewed
- **THEN** every semantic alias is defined as a `var(--<primitive>)` expression (or as a `var()` to another alias), and every primitive ramp step is defined as a raw color value.

#### Scenario: Contributor attempts to add a new semantic alias

- **WHEN** a proposal extends the semantic alias list
- **THEN** the proposal explicitly amends this requirement's enumeration; adding an alias without a spec update is rejected in review.

### Requirement: Scale Tokens

TimeTrak SHALL define scale tokens for spacing, radius, typography, motion, elevation, z-index layers, and breakpoints as CSS custom properties. Components SHALL reference scale tokens and MUST NOT use raw numeric values for these concerns.

- **Spacing** MUST be an 8px-based scale (`--space-1` = 4px through `--space-8` = 48px or equivalent named set documented in the token file).
- **Radius** MUST provide at minimum `--radius-sm` (small controls, inputs, chips, badges), `--radius-md` (larger cards, modals), and `--radius-pill` (fully rounded actions — buttons, timer control). Components authoring pill-shaped actions MUST reference `var(--radius-pill)` and MUST NOT use raw `999px` or equivalent literal values.
- **Typography** MUST define a font-family token pair (`--font-sans`, `--font-mono`) AND a codified size / weight / line-height scale. The scale tokens are:
  - **Size:** `--text-xs` (0.75rem) / `--text-sm` (0.8125rem) / `--text-md` (0.875rem) / `--text-base` (0.9375rem — the root body size) / `--text-lg` (1rem) / `--text-xl` (1.175rem) / `--text-2xl` (1.5rem) / `--text-3xl` (1.75rem). Component CSS MUST reference these tokens and MUST NOT declare raw `font-size: <n>rem` or `<n>px` values. Relative-to-parent sizes (`em` units on decorative glyphs, etc.) are permitted.
  - **Weight:** `--weight-regular` (400) / `--weight-medium` (500) / `--weight-semibold` (600) / `--weight-bold` (700). Component CSS MUST reference these tokens for any `font-weight` declaration.
  - **Line height:** `--leading-none` (1) / `--leading-tight` (1.1) / `--leading-snug` (1.25) / `--leading-normal` (1.5). Component CSS MUST reference these tokens for any `line-height` declaration.

  Fluid or clamp-based scales are out of scope. Additions to any of the three sub-scales require their own change proposal that amends this enumeration.
- **Motion** MUST define at least one duration token (e.g. `--motion-duration-fast`) and one easing token (e.g. `--motion-easing-standard`). All motion-using components MUST be collapsed to instant transitions under `@media (prefers-reduced-motion: reduce)`.
- **Elevation** MUST define `--shadow-none`, `--shadow-sm`, `--shadow-md`. Cards default to `--shadow-none` with a border; shadows above `--shadow-md` require a change proposal.
- **Z-index** MUST define an enumerated stack (at minimum `--z-base`, `--z-sticky`, `--z-dropdown`, `--z-modal`, `--z-toast`). Raw z-index integers in component CSS are prohibited.
- **Breakpoints** MUST define at least `--bp-sm`, `--bp-md`, `--bp-lg` for reference in media queries.

#### Scenario: Component uses a spacing value

- **WHEN** a component needs padding or gap
- **THEN** it references a `--space-*` token and does NOT use raw pixel or rem values.

#### Scenario: User prefers reduced motion

- **WHEN** the user agent reports `prefers-reduced-motion: reduce`
- **THEN** all transitions and animations defined anywhere in the foundation or component CSS collapse to instant state changes.

#### Scenario: Component defines a custom shadow

- **WHEN** a reviewer sees a component CSS rule using a raw `box-shadow` value instead of `var(--shadow-*)`
- **THEN** the review blocks the change and requires either adopting an existing shadow token or proposing a new one via a foundation change.

#### Scenario: Component authors a pill-shaped action

- **WHEN** a button or the timer control is authored with `border-radius: 999px` or `border-radius: 9999px`
- **THEN** the review blocks the change until the rule references `var(--radius-pill)`.

#### Scenario: Component uses a font-size value

- **WHEN** a component CSS rule declares `font-size: <n>rem` or `font-size: <n>px` with a literal value
- **THEN** the review MUST block the change until the rule references one of the enumerated `--text-*` tokens, or until a proposal amends this requirement to add a new size step.

#### Scenario: Component uses a font-weight value

- **WHEN** a component CSS rule declares `font-weight: <numeric>` with a raw literal (`font-weight: 500`, `font-weight: 700`, etc.)
- **THEN** the review MUST block the change until the rule references one of the `--weight-*` tokens.

#### Scenario: Component uses a line-height value

- **WHEN** a component CSS rule declares `line-height: <value>` with a raw literal
- **THEN** the review MUST block the change until the rule references one of the `--leading-*` tokens. Relative-to-parent `em` values on decorative glyph spans are permitted; absolute values are not.

### Requirement: CSS Layer Order

TimeTrak's stylesheet SHALL declare a canonical `@layer` order of `reset, tokens, base, components, utilities, overrides`. All rules MUST be authored inside the layer appropriate to their role. The `overrides` layer MUST exist but SHALL remain empty at foundation landing (reserved for future hot-fixes).

Token definitions MUST live in the `tokens` layer and MUST NOT be redefined inside `components`, `utilities`, or `base`.

The `@media (prefers-reduced-motion: reduce)` cross-cutting rule is the one approved exception to layer-scoping and MAY live outside the declared layer order.

#### Scenario: New component CSS is added

- **WHEN** a contributor adds CSS for a new component
- **THEN** the rules are wrapped in `@layer components { ... }` (or the file is structured so they fall into that layer), not placed in `base` or `utilities`.

#### Scenario: Component tries to redefine a token

- **WHEN** a component rule contains `--color-surface: ...` intended to override the token globally
- **THEN** the change is rejected; tokens are edited only in the `tokens` layer.

### Requirement: Component Authoring Contract

Any new component added to TimeTrak's CSS SHALL follow a single authoring contract.

- **Naming**: new components MUST use the `tt-<component>` class prefix (e.g. `tt-button`, `tt-field`). Legacy selectors in use at foundation landing (`.btn`, `.field`, `.table`, `.card`, `.badge`, `.timer`, `.flash`, `.empty`, `.nav`, `.app-shell`, `.app-header`, `.app-sidebar`, `.app-main`) are grandfathered and MUST NOT be renamed by this change.
- **State**: components MUST represent stateful variants using either ARIA / native attributes (`[aria-current]`, `[aria-invalid]`, `[aria-expanded]`, `[aria-disabled]`, `[disabled]`, `[data-theme]`) or `is-<state>` classes (`is-loading`, `is-active`). Ad-hoc state classes such as `.disabled` or `.active` MUST NOT be introduced.
- **Variants**: components that express emphasis SHALL use the variant vocabulary `primary`, `secondary`, `ghost`, `danger` only. Introducing a new variant (e.g. `success`, `warning` as a button variant) requires a change proposal that extends this requirement. Severity / status is expressed on badges or flash, not on buttons.
- **Sizes**: a size scale (`sm` / `md` / `lg`) MAY be introduced only when at least one production surface requires it. MVP components ship `md` only.
- **Target size**: every interactive element MUST render with at least 24×24 CSS pixels of hit area (WCAG 2.2 SC 2.5.8). The existing `.btn` and `.field` minimums satisfy this.

#### Scenario: New component introduced

- **WHEN** a contributor adds CSS for a new `tt-toggle` component
- **THEN** its class is prefixed `tt-`, its disabled state is expressed via `[aria-disabled="true"]` or `[disabled]`, and its variants are limited to `primary` / `secondary` / `ghost` / `danger` (or a documented subset) with no ad-hoc severity variant.

#### Scenario: Icon-only button is added

- **WHEN** an icon-only control is added to a partial
- **THEN** its rendered hit area is at least 24×24 CSS pixels and an accessible name is provided via visible label, `aria-label`, or `aria-labelledby`.

#### Scenario: Component introduces a new variant

- **WHEN** a change proposes a `tt-button.tt-button--success` variant
- **THEN** the proposal either amends this requirement's variant vocabulary or is rejected.

### Requirement: Focus-Indicator Contract

TimeTrak SHALL define exactly one focus-ring token (`--color-focus`) and exactly one `:focus-visible` rule in the `base` layer. The focus token MUST achieve a non-text contrast ratio of at least 3:1 (WCAG 2.2 SC 1.4.11) against every surface it can appear on in both light and dark themes, including `--color-surface`, `--color-surface-alt`, `--color-bg`, and `--color-accent`.

Components MUST NOT disable `:focus-visible` outlines. If a component needs a variant-specific ring colour (e.g. a different ring on top of an accent-filled surface), it SHALL override `outline-color` on the specific selector referencing a documented token, and MUST NOT introduce a second focus primitive.

#### Scenario: User tabs through the app

- **WHEN** a keyboard user focuses any interactive control via Tab, Shift+Tab, or Enter
- **THEN** a visible outline rendered with `var(--color-focus)` appears with at least 3:1 contrast against the surface the control sits on, in both light and dark themes.

#### Scenario: Component overrides focus outline colour

- **WHEN** a component needs a different focus-ring colour on its own surface
- **THEN** it overrides `outline-color` on its own selector using a documented token, does NOT define a new `--*-focus` primitive, and its chosen colour meets the same 3:1 contrast bar.

### Requirement: Status Never Conveyed By Colour Alone

Any component state or status that carries meaning (success, warning, error, running, archived, billable, disabled, loading) MUST be communicated by text, icon, or shape in addition to colour. Tokens in the severity pair (`--color-success`, `--color-warning`, `--color-danger`, `--color-info`) are supporting signals, never the sole signal.

Disabled state MUST be expressed with at least one non-colour cue (reduced opacity plus `cursor: not-allowed`, or a textual "(disabled)" label where opacity alone would not read as disabled).

#### Scenario: Archived badge is rendered

- **WHEN** a row shows a status badge for an archived record
- **THEN** the badge contains the word "Archived" (or an equivalent icon with an accessible name), not only a muted colour.

#### Scenario: Button enters disabled state

- **WHEN** a button has the `[disabled]` attribute or `is-disabled` class
- **THEN** the component CSS reduces opacity and sets `cursor: not-allowed`, and handlers communicate *why* the button is unavailable either via visible copy, tooltip, or adjacent helper text.

### Requirement: Token Deprecation and Migration Rule

When a token is renamed (for example, the foundation rename from `--surface` to `--color-surface`), the old name MUST continue to resolve to the new name as a deprecation alias for at least one subsequent foundation change. The deprecation alias MUST carry a comment in the token file identifying it as deprecated and naming the replacement. Old aliases are removed only by a later change that explicitly enumerates them.

New tokens, scales, or ramps SHALL be added by a change proposal that amends the relevant requirement (Two-Layer Token Taxonomy, Scale Tokens) and documents the rationale, the contrast role (for colour tokens), and any affected components.

#### Scenario: Token is renamed

- **WHEN** the foundation introduces `--color-surface` to replace `--surface`
- **THEN** `--surface` remains defined in the token file as `var(--color-surface)` with a `/* deprecated */` comment, and components migrate to the new name in the same change.

#### Scenario: Deprecated token is removed

- **WHEN** a subsequent foundation change proposes removing a deprecation alias
- **THEN** the proposal lists every alias being removed, and CI / review verifies no component CSS still references them.

### Requirement: Authoring Contract Documentation

TimeTrak SHALL ship a developer-facing document at `web/static/css/README.md` that codifies the authoring contract: the two-layer token model, the enumerated semantic aliases, the scale token set, the `@layer` order, the component naming convention, the variant vocabulary, state-class rules, the focus-indicator rule, the target-size rule, and the deprecation-alias rule. The document MUST cross-link to `web/templates/partials/README.md` and be cited from any future change proposal that adds a component or token.

The document MUST state explicitly that during any transition period where `docs/timetrak_ui_style_guide.md` has not yet been updated to match, the codified CSS tokens and this specification are authoritative.

#### Scenario: Contributor adds a new component

- **WHEN** a contributor opens `web/static/css/README.md`
- **THEN** they find the naming convention, variant vocabulary, state-class rules, and the enumerated semantic aliases they must use, without needing to read the CSS source.

#### Scenario: Style guide and codified tokens disagree

- **WHEN** `docs/timetrak_ui_style_guide.md` quotes a token name or rule that differs from `web/static/css/tokens.css`
- **THEN** the codified CSS + this spec win, and the style guide MUST be updated in a follow-up change.

### Requirement: FOUC-prevention head script is the single sanctioned inline script

The `base.html` layout SHALL carry exactly one inline `<script>` element, placed before any `<link rel="stylesheet">` in `<head>`, whose sole purpose is to read the user's stored theme preference from `localStorage` under the key `timetrak.theme` and apply it to `<html>` as a `data-theme` attribute before first paint. The script MUST be ≤30 lines, MUST be wrapped in a `try { ... } catch (e) { }` so a localStorage-denied environment falls through cleanly, and MUST NOT reference any symbol outside its own IIFE scope.

No other inline `<script>` element is permitted anywhere in the product's template tree. All other client-side behavior lives in `web/static/js/app.js` (or a successor external script) and is subject to the normal caching / CSP story.

#### Scenario: First paint renders the stored theme

- **WHEN** a user who previously selected `dark` returns to any page
- **THEN** the inline head-script reads `localStorage.timetrak.theme` and sets `<html data-theme="dark">` before the browser paints the first frame
- **AND** the user MUST NOT see a flash of the default `system` theme

#### Scenario: localStorage is denied

- **WHEN** the user's browser denies `localStorage.getItem` (strict privacy modes, private-browsing flavors)
- **THEN** the inline head-script's `try { ... } catch` swallows the error
- **AND** the document falls through to its default `data-theme` attribute (`system`) without a render error

#### Scenario: Contributor attempts to add another inline script

- **WHEN** a proposed change adds an inline `<script>` anywhere in `web/templates/`
- **THEN** the review MUST block the change unless the change also amends this requirement
- **AND** the acceptable alternative is to extend `web/static/js/app.js` or ship a new external script

