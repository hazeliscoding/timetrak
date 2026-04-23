## ADDED Requirements

### Requirement: Brand mark partial

The system SHALL expose a canonical brand-mark partial at `web/templates/partials/brandmark.html` that renders TimeTrak's wordmark as an inline SVG. The partial MUST consume only semantic-alias and `currentColor` values for fill and stroke — specifically `currentColor`, `var(--color-text)`, and `var(--color-accent)` — and MUST NOT reference primitive ramps, raw hex or rgb values, or any new semantic alias. The partial MUST accept a `dict` with two keys: `Size` (string, one of `sm` or `md`; defaults to `md` when empty) and `Decorative` (bool; defaults to `false`). When `Decorative` is `false` the rendered SVG MUST carry `role="img"` and a child `<title>TimeTrak</title>` so assistive technology announces the mark as a graphic named "TimeTrak". When `Decorative` is `true` the SVG MUST carry `aria-hidden="true"` and MUST NOT emit a `<title>` element. The partial MUST be listed in `web/templates/partials/README.md` alongside the other canonical partials and MUST NOT be duplicated or re-implemented in any domain template.

#### Scenario: Default non-decorative render from the app header

- **WHEN** `web/templates/layouts/app.html` invokes `{{template "brandmark" (dict "Size" "md" "Decorative" false)}}`
- **THEN** the rendered SVG carries `role="img"` and contains a `<title>TimeTrak</title>` child
- **AND** the SVG's fill and stroke reference only `currentColor`, `var(--color-text)`, or `var(--color-accent)`
- **AND** no raw hex, rgb, hsl, or named colour value appears in the rendered output

#### Scenario: Decorative render adjacent to text that already names the product

- **WHEN** a surface invokes `{{template "brandmark" (dict "Size" "sm" "Decorative" true)}}`
- **THEN** the rendered SVG carries `aria-hidden="true"`
- **AND** the SVG does NOT contain a `<title>` child
- **AND** assistive technology skips the mark silently

#### Scenario: Token-contract compliance is enforced at authoring time

- **WHEN** a contributor adds or modifies `web/templates/partials/brandmark.html`
- **THEN** code review MUST reject any fill or stroke that references a primitive ramp, a raw colour value, or a new semantic alias
- **AND** adding a new semantic alias for brand purposes requires amending `openspec/specs/ui-foundation/spec.md` under its existing amendment rule, not this requirement

#### Scenario: Focus behavior when wrapped in an anchor

- **WHEN** the brandmark partial is rendered inside an anchor (e.g. the app header link)
- **THEN** the anchor SHALL inherit the global `:focus-visible` outline documented in `ui-foundation`
- **AND** the partial MUST NOT introduce a component-scoped focus override
