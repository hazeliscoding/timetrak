## Why

The MVP shipped a flat set of design tokens in `web/static/css/tokens.css` and a single `app.css` of base styles, and the prior change (`create-reusable-ui-partials-and-patterns`) codified server-rendered partial conventions. What is still missing is an intentional **foundation layer** between those two things: a formal token taxonomy (primitives → semantic aliases), a documented CSS layering order, and an authoring contract (naming, states, focus ring, target size, variants) that future component work can cite instead of re-arguing per change. Without it, the next two roadmap changes — the component library showcase and the brand refinement — will relitigate foundation decisions inline, and new components will silently reach for raw color values or invent ad-hoc state classes.

This change establishes that foundation. It is **scaffolding + conventions**, not a component rollout and not a visual redesign.

## What Changes

- **Audit** the current `tokens.css` + `app.css` + component partial CSS surface for ad-hoc values (raw colors, radii, shadows, magic numbers), duplicated tokens, and gaps.
- **Token taxonomy**: introduce a two-layer model — primitive ramps (e.g. `--neutral-50`…`--neutral-900`, `--accent-50`…`--accent-900`) and semantic aliases (`--color-text`, `--color-text-muted`, `--color-surface`, `--color-surface-alt`, `--color-border`, `--color-border-strong`, `--color-focus`, severity tokens for info/success/warn/error). Components reference **only** the semantic aliases.
- **Scale tokens**: codify spacing, radius, typography (font-family, size, weight, line-height), motion (duration, easing) guarded by `prefers-reduced-motion`, elevation (borders-first per CLAUDE.md; shadows used sparingly), z-index layers, and breakpoints.
- **CSS organization**: adopt a single documented layer order (`@layer reset, tokens, base, components, utilities, overrides`) and make the token file the single source of truth. Components must not redefine tokens.
- **Component authoring contract**: settle one naming convention (`tt-<component>` with state classes like `is-disabled`, `is-loading` and ARIA-driven state where possible), one focus-ring token meeting WCAG 2.2 SC 1.4.11 (3:1 non-text contrast), a minimum target size of 24×24 (SC 2.5.8), a size-scale policy (`sm`/`md`/`lg` only when justified), and a variant vocabulary (`primary` / `secondary` / `ghost` / `danger`) with when-to-use rules.
- **Accessibility obligations**: every defined semantic color pair carries a documented contrast role; focus indicators survive Windows high-contrast mode; disabled state never relies on color alone; motion respects `prefers-reduced-motion`.
- **Contribution rules**: how a future change adds a new token (extend a ramp vs. add a semantic alias) or a new component without breaking the foundation; deprecation-alias rule for renamed tokens (old names kept for one change cycle).
- **Authoritative-source rule**: after this change lands, the CSS token file + the `ui-foundation` spec are authoritative. `docs/timetrak_ui_style_guide.md` is advisory until updated in a follow-up.

### In scope

- Token taxonomy, CSS layer order, authoring contract, variant vocabulary, accessibility rules baked into tokens, contribution / migration rules.
- Re-pointing any existing component CSS at semantic aliases where it currently reaches for raw values (mechanical token renames only; no visual changes beyond fixing accessibility regressions surfaced by the audit).
- Keeping deprecated token names as aliases for one change cycle where needed.

### Out of scope (called out explicitly)

- Adding new components (buttons, inputs, dialogs, toasts, etc.). The foundation defines *how* to build them; the next change (`create-component-library-showcase-and-usage-docs`) catalogues them.
- Brand redesign or a new accent color — that is `refine-timetrak-brand-and-product-visual-language`.
- JS component framework, Web Components, or build-step tooling. TimeTrak stays plain CSS + `html/template` + HTMX.
- CSS-in-JS, Tailwind, PostCSS, Sass, CSS modules — not introduced.
- Full dark-mode palette implementation. The foundation documents dark mode as a constraint (semantic aliases MUST be theme-swappable) but does not overhaul the existing dark ramp.
- Icon system.
- Fluid/responsive typography (clamp-based scales). Static scale only.

### Assumptions

- All components remain server-rendered (`html/template`) + HTMX. No SPA framework.
- `app.css` + `tokens.css` remain the only stylesheet entry points; no preprocessor or build step added.
- Tokens live as CSS custom properties on `:root` (light) with overrides on `[data-theme="dark"]` and the `prefers-color-scheme` branch (already established in `tokens.css`).
- WCAG 2.2 AA remains the enforced accessibility target.
- The existing HTMX event contract, `data-focus-after-swap` helper, and partial naming conventions from the prior change are preserved untouched.

### Risks

- **Token sprawl** — too many semantic aliases become unmaintainable. Mitigation: the spec caps semantic aliases to the enumerated set; new aliases require justification.
- **Over-abstraction before demand** — inventing tokens no component consumes. Mitigation: aliases land only if at least one current partial needs them; ramps are primitives even without current consumers because they anchor future work.
- **Breaking existing partials during token rename** — the partials catalogue just stabilized. Mitigation: old token names are kept as aliases (pointing at the new semantic names) for one change cycle, with a deprecation comment.
- **Focus-ring regression** — changing the focus token without contrast verification breaks SC 1.4.11. Mitigation: a dedicated accessibility validation task covers focus ring, disabled, and all documented color pairs against 3:1 / 4.5:1 as applicable.
- **Divergence between style guide and codified tokens** — `docs/timetrak_ui_style_guide.md` currently quotes the old flat token set. Mitigation: the spec names the CSS file + `ui-foundation` spec as authoritative in the interim and flags the doc update as a required follow-up.

### Likely follow-ups

- `create-component-library-showcase-and-usage-docs` — consumes this foundation to catalogue button/input/table/etc. components.
- A dedicated documentation update to `docs/timetrak_ui_style_guide.md` to quote the codified tokens rather than its own table.
- `refine-timetrak-brand-and-product-visual-language` — consumes the ramp/alias split to swap the accent ramp without touching component code.

## Capabilities

### New Capabilities

- `ui-foundation`: the authoritative contract for TimeTrak's design tokens (primitive ramps, semantic aliases, scales), CSS layer order, component authoring conventions (naming, state, focus, target size, variants), accessibility obligations of the token layer, and contribution / migration rules. This is the baseline future UI changes cite before adding or modifying components.

### Modified Capabilities

_None._ `ui-partials` (the prior change's capability) is unchanged — it describes partial conventions and HTMX event contracts, which this change does not touch. Domain-level capabilities (`auth`, `workspace`, `clients`, `projects`, `tracking`, `rates`, `reporting`) are also unchanged; this is purely a CSS + conventions layer.

## Impact

- **CSS**: `web/static/css/tokens.css` gains the primitive-ramp + semantic-alias split and the documented scale tokens; `web/static/css/app.css` gains the `@layer` order declaration and migrates component selectors to semantic aliases. No file is renamed or moved unless the design doc specifies it.
- **Templates**: no markup changes. Existing partials in `web/templates/partials/` continue to render identically; only the CSS values behind them are reorganized.
- **Handlers / routes / schema**: unchanged.
- **JS**: unchanged. `web/static/js/app.js` (theme toggle + focus helper) is untouched.
- **Documentation**: new authoring-contract documentation lives alongside the CSS (exact location — dedicated `web/static/css/README.md` vs. appendix to `web/templates/partials/README.md` — decided in the design doc). `docs/timetrak_ui_style_guide.md` is **not** edited in this change; it is flagged as requiring a follow-up.
- **Tests**: no new Go tests. Accessibility validation is manual (contrast verification, focus-ring visibility against light/dark/high-contrast, reduced-motion verification) and is explicitly tasked.
- **Dependencies**: none added. No frameworks, no preprocessors, no JS libraries.
- **Stage fit**: Stage 2 (Stabilize). No new domain capability; this hardens and formalizes an existing foundation.
