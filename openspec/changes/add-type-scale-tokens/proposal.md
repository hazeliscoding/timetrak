## Why

The accepted `ui-foundation` Scale Tokens requirement says: *"Typography MUST define a font-family token pair (`--font-sans`, `--font-mono`) and a documented static size / weight / line-height set."* The font-family pair exists. The documented size / weight / line-height set does not — `tokens.css` stops at the family pair, and `app.css` has 10+ raw `font-size:` values, 6+ raw `font-weight:` values, and 5+ raw `line-height:` values scattered across components. The `docs/timetrak_ui_style_guide.md` §Type Hierarchy describes the intent in prose ("Page title: bold, large" … "Secondary text: muted but still readable") but never codifies it as tokens.

The practical consequence: every new component author has to invent a size (`0.75rem`? `0.8125rem`? `0.875rem`? — all three exist today), weight (500? 600?), and line-height (1, 1.1, 1.25, 1.5 — all in use). Drift is guaranteed. The `sharpen-component-identity` accent-rationing audit gave accent a single source of truth; this change does the analogous work for type.

The inventory found in a grep of `web/static/css/app.css`:

- Sizes in use: `0.75rem`, `0.8125rem`, `0.875rem`, `0.9375rem`, `1rem`, `1.175rem`, `1.5rem`, `1.75rem`, plus `15px` on `body` (= `0.9375rem` relative to a `16px` root).
- Weights in use: `400` (implicit default), `500`, `600`, `700`.
- Line heights in use: `1`, `1.1`, `1.25`, `1.5`.

This change codifies each of those real values as a named token, migrates every raw value in `app.css` to consume them, and amends the Scale Tokens requirement so "typography" matches the level of specification already in force for spacing / radius / motion / elevation / z-index / breakpoints.

## What Changes

- **Add a type-size scale** to `web/static/css/tokens.css` covering the 8 distinct values currently in use:
  - `--text-xs`   (`0.75rem`  / 12px)   — chip labels, uppercase table headers
  - `--text-sm`   (`0.8125rem` / 13px)  — nav, timer meta, "started at" line
  - `--text-md`   (`0.875rem` / 14px)   — hints, field errors, muted secondary
  - `--text-base` (`0.9375rem` / 15px)  — body default; table body strong cells
  - `--text-lg`   (`1rem` / 16px)       — h3, normal-weight emphasized
  - `--text-xl`   (`1.175rem` / ~19px)  — h2
  - `--text-2xl`  (`1.5rem` / 24px)     — h1
  - `--text-3xl`  (`1.75rem` / 28px)    — numeric summaries (timer elapsed, running card)
- **Add a weight scale:** `--weight-regular` (400), `--weight-medium` (500), `--weight-semibold` (600), `--weight-bold` (700).
- **Add a line-height scale:** `--leading-none` (1) for tight single-line chips/glyphs, `--leading-tight` (1.1) for large numeric display, `--leading-snug` (1.25) for headings, `--leading-normal` (1.5) for body prose.
- **Migrate every raw `font-size` / `font-weight` / `line-height` in `web/static/css/app.css`** to consume the new tokens. The `0.75em` / `0.9em` relative-to-parent values on chip/theme glyphs stay raw (they are *intentionally* relative, not fixed; the tokens are for absolute sizes).
- **Amend the `ui-foundation` Scale Tokens requirement** to enumerate the new sub-scales alongside the existing spacing / radius / etc. scales, and to require components to consume them (no raw `font-size` / `font-weight` / `line-height` values in component CSS).
- **Document the scale** in `web/static/css/README.md` under the existing Scale tokens section, and cross-link from `docs/timetrak_ui_style_guide.md` §Type Hierarchy so the prose-level hierarchy maps onto concrete tokens.
- **Out of scope (explicit):**
  - Migrating inline `style="font-size:..."` in templates. Those are the 15 raw-value inline styles flagged in the earlier audit; a separate `sweep-raw-inline-styles` change consumes this change's output.
  - Changing any actual size, weight, or line-height. The scale names new tokens for values already in use; no visual change is intended.
  - Introducing a fluid / clamp-based scale. `ui-foundation` already excludes that, and this change respects the exclusion.
  - Adding new sizes not currently in use (e.g. a `--text-4xl` for some hypothetical hero). Additions require their own change.
  - Changing the root body size (`15px` = `0.9375rem`). The `body` rule migrates to `font-size: var(--text-base)` to consume the token like any other CSS; the numeric value is unchanged.

## Capabilities

### New Capabilities

- *None.* The change extends the existing `ui-foundation` Scale Tokens requirement rather than introducing a new capability. A new capability for "typography tokens" would be overfit for a sub-scale that naturally belongs under Scale Tokens.

### Modified Capabilities

- `ui-foundation` — MODIFIES the Scale Tokens requirement to enumerate the typography sub-scales (size, weight, line-height) and require component CSS to consume them. Adds scenarios pinning the no-raw-value rule at review time. No other `ui-foundation` requirement is touched.

## Impact

- **CSS:** `web/static/css/tokens.css` grows by ~16 lines (new token declarations in the `:root` block; no dark-theme overrides needed since the scale is theme-invariant). `web/static/css/app.css` sees every raw `font-size`, `font-weight`, `line-height` replaced by a token reference — a mechanical pass across ~25 declarations.
- **Docs:** `web/static/css/README.md` gains a Typography sub-section under Scale tokens. `docs/timetrak_ui_style_guide.md` §Type Hierarchy gets a small code-reference block mapping each hierarchy item to a token pair.
- **Specs:** delta under `openspec/changes/add-type-scale-tokens/specs/ui-foundation/spec.md`.
- **Tests:** no new test; the `sharpen-component-identity` CSS audit test covers accent rationing, not type. If a future need emerges for an audit of raw-type values in component CSS, it can be a separate change — this one keeps scope tight to the token surface + migration.
- **No template changes. No backend, no DB, no migration, no new dependency.**
- **Risk:** a mechanical migration where one declaration is mistyped (e.g. `--text-xs` vs `--text-sm`) could produce a visible regression. Mitigated by (a) the inventory above pinning each call site to a specific token, (b) manual eyeball in `make run` on the dashboard and the showcase after migration, (c) no change to actual rendered size — a diff against pre-change computed styles should be zero.
- **Follow-ups:** `sweep-raw-inline-styles` (template-level inline styles referring to raw type / gap / margin values); a potential audit test analogous to `TestAccentRationingAudit` that fails the build on raw `font-size:` / `font-weight:` / `line-height:` in component CSS.
