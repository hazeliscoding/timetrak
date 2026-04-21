# CSS Authoring Contract

**Browser-visible reference:** `/dev/showcase/tokens` (dev-only) renders every
semantic alias, scale token, and primitive ramp with live previews under the
current theme. See `internal/showcase/` for the catalogue definition.

This is the authoring contract for TimeTrak's stylesheet. It governs
`web/static/css/tokens.css` and `web/static/css/app.css` — the only two
stylesheet entry points in the app.

**Authoritative source.** Until `docs/timetrak_ui_style_guide.md` is
updated to cite the codified tokens (a scheduled follow-up), this
README + `web/static/css/tokens.css` + `openspec/specs/ui-foundation/spec.md`
are authoritative. Where the style guide disagrees with the codified
tokens, the codified tokens win.

**Sibling doc.** Server-rendered partial conventions and the HTMX event
contract live in [`web/templates/partials/README.md`](../../templates/partials/README.md).

---

## 1. Two-layer token model

TimeTrak tokens split into **primitive ramps** (palette anchors, private
to the foundation) and **semantic aliases** (public tokens consumed by
components).

- **Primitive ramps** — `--neutral-0`…`--neutral-900`, `--accent-50`…`--accent-900`,
  plus severity anchors `--red-500`/`--red-600`/`--red-soft`,
  `--amber-500`/`--amber-600`/`--amber-soft`,
  `--green-500`/`--green-600`/`--green-soft`. Components **MUST NOT**
  reference these directly.
- **Semantic aliases** — the only colour tokens components may consume:

  | Alias                     | Role                                                           |
  | ------------------------- | -------------------------------------------------------------- |
  | `--color-bg`              | Page background.                                               |
  | `--color-surface`         | Cards, tables, inputs, buttons at rest.                        |
  | `--color-surface-alt`     | Table headers, hover rows, muted surfaces.                     |
  | `--color-text`            | Body text on any surface. Target ≥4.5:1.                       |
  | `--color-text-muted`      | Secondary / helper text on `--color-surface`. Target ≥4.5:1.   |
  | `--color-border`          | Default 1px separators. Target ≥3:1 non-text.                  |
  | `--color-border-strong`   | Input, button, and emphatic borders.                            |
  | `--color-accent`          | Primary brand signal. Non-text ≥3:1; `#fff` text on it ≥4.5:1. |
  | `--color-accent-hover`    | Hover state for accent fills.                                   |
  | `--color-accent-soft`     | Low-emphasis accent fill (billable badge, nav-current).         |
  | `--color-focus`           | The single focus-ring colour. ≥3:1 on every surface.            |
  | `--color-success` / `-soft` | Confirmed success status (paired with text or icon).          |
  | `--color-warning` / `-soft` | Warning status (paired with text or icon).                    |
  | `--color-danger`  / `-soft` | Destructive / error status (paired with text or icon).        |
  | `--color-info`    / `-soft` | Neutral informational status.                                 |

**Contract.** Components reference semantic aliases only. A raw hex,
`rgb(`, or primitive-ramp reference in a component rule is a review
block — either the alias exists and you forgot it, or the alias needs
to be proposed via a foundation change.

The one documented exception today: `.btn-primary` / `.btn-danger` use
`color: #fff` for text on filled accent/danger surfaces. That is the
documented "accent-on-text" pairing. If a second consumer appears, add
a `--color-on-accent` alias in a foundation change.

## 2. Scale tokens

Components **MUST** consume scale tokens instead of raw numeric values.

- **Spacing** — `--space-1` (4px) through `--space-8` (48px).
- **Radius** — `--radius-sm` (controls), `--radius-md` (cards).
- **Typography** — `--font-sans`, `--font-mono`. Static size scale in
  rem (`1rem`, `1.175rem`, `1.5rem`, `1.75rem`). No clamp-based fluid
  scales.
- **Motion** — `--motion-duration-fast` (120ms), `--motion-duration-normal`
  (200ms), `--motion-easing-standard`. All motion **MUST** collapse to
  instant under `prefers-reduced-motion: reduce`.
- **Elevation** — `--shadow-none`, `--shadow-sm`, `--shadow-md`. Cards
  default to `--shadow-none` + `1px solid var(--color-border)`. Shadows
  above `--shadow-md` require a foundation change.
- **Z-index** — `--z-base`, `--z-sticky`, `--z-dropdown`, `--z-modal`,
  `--z-toast`. Raw z-index integers in component CSS are prohibited.
- **Breakpoints** — `--bp-sm` (640px), `--bp-md` (960px), `--bp-lg`
  (1280px). For use in `@media` queries only.

## 3. `@layer` order

`app.css` declares exactly this order:

```css
@layer reset, tokens, base, components, utilities, overrides;
```

- `reset` — box-sizing, margin resets.
- `tokens` — the entire contents of `tokens.css` belong here (imported
  separately via `<link>`; precedence is still governed by the declared
  layer order thanks to native `@layer`).
- `base` — element defaults, `:focus-visible`, `.sr-only`, `.muted`,
  `.tabular`.
- `components` — `.app-shell`, `.nav`, `.btn`, `.field`, `.table`,
  `.card`, `.badge`, `.flash`, `.timer`, `.empty`, and all new `tt-*`
  components.
- `utilities` — `.num`, `.stack`, `.row`, `.row-between`, `.mt-0`,
  `.mb-0`.
- `overrides` — reserved, empty at foundation landing.

Token definitions **MUST NOT** be redefined inside `components`,
`utilities`, or `base`.

**Documented exception.** The `@media (prefers-reduced-motion: reduce)`
rule lives outside the layered cascade so it beats every component
layer regardless of source order. The `!important` on
`transition`/`animation` is the one approved use.

## 4. Component authoring convention

### 4.1 Naming

- **New components** — class prefix `tt-<component>` (`tt-button`,
  `tt-field`, `tt-toggle`).
- **Legacy selectors** — `.btn`, `.field`, `.table`, `.card`, `.badge`,
  `.timer`, `.flash`, `.empty`, `.nav`, `.app-shell`, `.app-header`,
  `.app-sidebar`, `.app-main` — are grandfathered and **MUST NOT** be
  renamed by this foundation change. They are the existing surface that
  partials already call; renaming forces handler + partial churn for no
  behavioural win.

### 4.2 State

Represent stateful variants with ARIA / native attributes where one
exists, or `is-<state>` classes otherwise.

- Preferred: `[aria-current]`, `[aria-invalid]`, `[aria-expanded]`,
  `[aria-disabled]`, `[disabled]`, `[data-theme]`.
- Allowed when no attribute fits: `is-loading`, `is-active`,
  `is-disabled` (purely visual, where the semantic attribute is
  already present elsewhere).
- **Prohibited**: ad-hoc state classes like `.disabled`, `.active`,
  `.selected`, `.open`.

### 4.3 Variants

Components that express emphasis use this vocabulary only:

- `primary` — one per page region; the main action.
- `secondary` — default bordered button. Use when emphasis is not needed.
- `ghost` — lowest-emphasis interactive element. No border, no fill at rest.
- `danger` — destructive. Pair with destructive copy; never the only
  non-text signal.

Adding a new variant (e.g. `success`, `warning` on buttons) requires a
foundation change that amends the spec. Severity / status belongs on
badges or flash, not on buttons.

### 4.4 Sizes

A `sm` / `md` / `lg` size scale **MAY** be introduced only when at least
one production surface requires it. MVP components ship `md` only.

### 4.5 Focus indicator

Exactly one focus-ring token (`--color-focus`) and exactly one
`:focus-visible` rule exist in `app.css`:

```css
:focus-visible {
  outline: 3px solid var(--color-focus);
  outline-offset: 2px;
  border-radius: 2px;
}
```

The token must achieve ≥3:1 contrast against every surface it can
appear on (`--color-surface`, `--color-surface-alt`, `--color-bg`,
`--color-accent`) in both light and dark themes.

Components **MUST NOT** disable `:focus-visible`. If a specific
component needs a different ring colour on its own surface, override
`outline-color` on that selector using an existing documented token;
do NOT introduce a second focus primitive.

**Contract test.** `internal/e2e/browser/focus_ring_test.go` (gated by
`//go:build browser`; run via `make test-browser`) drives every
interactive primitive in both `[data-theme="light"]` and
`[data-theme="dark"]` and asserts computed `outline-width` / `outline-offset`
plus live-resolved `--color-focus`. Adding a new interactive primitive
means adding a row to that test's `focusRingRows()` table. The companion
`reduced_motion_test.go` asserts that transitions collapse to `0s` under
`prefers-reduced-motion: reduce`; add a target there when you introduce
a new transition.

### 4.6 Target size

Every interactive element renders with at least 24×24 CSS pixels of hit
area (WCAG 2.2 SC 2.5.8). The existing `.btn` (36px tall, ≥44px wide)
and `.field` inputs (36px tall) satisfy this. Icon-only controls added
in future changes **MUST** meet this bar.

### 4.7 Status never conveyed by colour alone

Any state or status that carries meaning (success, warning, error,
running, archived, billable, disabled, loading) **MUST** be
communicated by text, icon, or shape in addition to colour. The
severity tokens are supporting signals, never the sole signal.

Disabled state uses at least one non-colour cue (reduced opacity plus
`cursor: not-allowed`, or a textual "(disabled)" label where opacity
alone would not read as disabled).

## 5. Token deprecation and migration

When a token is renamed (for example the current rename of `--surface`
to `--color-surface`), the old name **MUST** continue to resolve as a
deprecation alias for at least one subsequent foundation change. The
token file carries a `/* deprecated: use --color-foo */` comment next
to each alias. Components migrate to the new name in the same change
that introduces it. Old aliases are removed only by a later change that
explicitly enumerates them.

New tokens, scales, or ramps are added by a change proposal that amends
the relevant spec requirement and documents the rationale, contrast
role (for colour tokens), and affected components.

## 6. Authoring a new component — checklist

1. **Name** — prefix with `tt-` (`tt-toggle`, `tt-drawer`).
2. **Layer** — wrap rules in `@layer components { ... }` or keep them
   inside the `components` layer block.
3. **Tokens** — consume only semantic aliases + scale tokens. No raw
   hex, no `rgb(`, no raw px for spacing / radius / shadow / motion.
   If the token you need does not exist, propose a foundation change —
   don't reach for a ramp.
4. **State** — ARIA / native attributes first, `is-<state>` classes
   otherwise. No ad-hoc `.disabled` / `.active`.
5. **Variants** — stay within `primary` / `secondary` / `ghost` /
   `danger`. Severity lives on badges / flash.
6. **Focus** — rely on the global `:focus-visible` rule; do not disable
   it. If a variant needs a different ring colour, override
   `outline-color` with a documented token.
7. **Target size** — ≥24×24 CSS px hit area for every interactive
   element.
8. **Status** — never colour alone. Pair with text or icon.
9. **Motion** — use `--motion-duration-*` / `--motion-easing-*`; verify
   `prefers-reduced-motion: reduce` collapses the motion.
10. **Cross-doc** — if the component ships a partial, add it to
    [`web/templates/partials/README.md`](../../templates/partials/README.md).

## 7. Proposing a new token or alias

- **New primitive ramp step** — propose via a foundation change. Ramps
  are palette anchors; additions are rare and must document which
  semantic alias will consume them.
- **New semantic alias** — propose via a foundation change that amends
  the "Two-Layer Token Taxonomy" requirement in
  `openspec/specs/ui-foundation/spec.md`. Document the contrast role
  (what surface it pairs with, what ratio target), the at-least-one
  current consumer, and the dark-theme mirror.
- **New scale token** (motion, shadow, z, breakpoint, spacing, radius)
  — propose via a foundation change that amends the "Scale Tokens"
  requirement. Include the dark-theme value if applicable.
- **Component-scoped token** — allowed inside a specific component's
  selector scope (e.g. `.tt-toggle { --tt-toggle-knob: var(--color-surface); }`).
  Do NOT add component-scoped tokens to the global `tokens.css`.
