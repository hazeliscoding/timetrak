## ADDED Requirements

### Requirement: Theme switch partial

The system SHALL provide a canonical `partials/theme_switch` that renders a single segmented control for the product's three themes (`light`, `dark`, `system`). The partial's root element SHALL be `<div role="radiogroup" aria-label="Theme">`. Each theme option SHALL render as a child `<button type="button" role="radio">` carrying `data-theme-set="<value>"`, both `aria-pressed="true|false"` and `aria-checked="true|false"` set consistently, and a visible `<span>` label paired with an `aria-hidden` leading glyph.

The partial's `dict` context is optional. When invoked without context (the production header case) it renders all three segments with `aria-pressed="false"` / `aria-checked="false"` — the existing client JS (`web/static/js/app.js`) synchronizes the active segment from `localStorage.timetrak.theme` after the FOUC head-script sets `data-theme`. When invoked with an explicit `InitialSelected` key (the showcase case) it renders with the matching segment pre-set, for documenting each of the three selected states statically.

The partial MUST NOT introduce any new runtime dependency, MUST NOT ship inline JS, and MUST reuse the existing `data-theme-set` click contract.

#### Scenario: Production header renders all three segments, none pre-selected

- **WHEN** `{{template "theme_switch" (dict)}}` is invoked from the app shell (no `InitialSelected` key)
- **THEN** the partial renders exactly one `role="radiogroup"` element containing exactly three `role="radio"` `<button>` children whose `data-theme-set` values are `light`, `dark`, and `system` respectively
- **AND** every child button initially carries `aria-pressed="false"` and `aria-checked="false"` (client JS synchronizes the active state post-parse)
- **AND** every child button has a visible label element and an `aria-hidden` leading glyph

#### Scenario: Showcase rendering pre-selects a segment

- **WHEN** `{{template "theme_switch" (dict "InitialSelected" "dark")}}` is invoked
- **THEN** the `dark` segment renders with `aria-pressed="true"` and `aria-checked="true"`
- **AND** the other two segments render with `aria-pressed="false"` and `aria-checked="false"`

#### Scenario: Keyboard focus lands on the active segment

- **WHEN** a keyboard user tabs into the theme switch
- **THEN** focus lands on the currently-active segment (the one with `aria-pressed="true"`), matching the standard radiogroup keyboard contract
- **AND** the visible focus ring on that segment satisfies the accepted `ui-foundation` focus-indicator contract
