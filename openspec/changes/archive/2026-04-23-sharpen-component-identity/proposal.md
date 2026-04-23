## Why

TimeTrak's visual system is technically correct — tokens, partials, focus contract, severity, reduced-motion — but the components built on top of it read as generic and interchangeable. A table looks like any table; a button looks like any button; the timer, which is the product's signature action, has no more presence than a filter dropdown. The app feels boring, and boring erodes the "trustworthy tool" register we want: calm should feel *crafted*, not *undifferentiated*.

This change sharpens component identity within the existing restrained brand: Linear/Stripe-level opinion, not Notion/ClickUp flourish. It does NOT introduce gradients, illustrations, new colors, hero sections, or animated micro-interactions. It does introduce a small set of shape-and-edge contracts that make each component recognizably itself across every screen.

## What Changes

- Introduce a **shape-language contract** that is load-bearing across the app: pills = actions (buttons, timer), rectangles with 4px radius = status/metadata (chips, badges), circles = presence dots. Components MUST honor this taxonomy; violations are a review block.
- Introduce a **two-weight border contract**: 1px neutral for structure (cards, inputs at rest, table dividers); 2px accent for state (focus, selection, running, validation error). Nothing in between. This becomes the single visual grammar for "this one is active."
- Introduce a **numeric-text contract**: `font-variant-numeric: tabular-nums` is required wherever durations, amounts, rates, or counts render; right-aligned in table cells, left-aligned in summary cards.
- Introduce an **accent-rationing contract**: the accent color is permitted only on (a) the running-timer fill, (b) the focus ring, (c) selected-row edge, (d) primary button, (e) running-entry card top border. Appearing elsewhere is a bug.
- Sharpen the **timer control** identity: idle state is a neutral pill with leading dot; running state inverts to an accent-tinted fill with a pulsing dot (respects reduced-motion), tabular-nums elapsed time taking visual priority, and a distinct `Stop` shape. The timer is the app's signature object, not a styled button.
- Sharpen the **data table** treatment: hairline horizontal dividers only, no zebra, no verticals; `tabular-nums` + right-aligned numeric columns; uppercase letter-spaced column headers (the only uppercase in the app); selected/focused row gets a 2px accent left-edge rule inside the row, not a full border or background.
- Sharpen the **status chip**: rectangular, 4px radius, 20px height, text-xs medium; chips are rectangles, buttons are pills — this separation is load-bearing. Every chip pairs color with a glyph or shape to satisfy "never color-alone."
- Add a **component identity review contract** — a documented checklist the `ui-showcase` gallery and PR review use to verify shape, border, numeric, accent-rationing, and focus compliance before merge.

Deferred to follow-on proposals (explicitly out of scope here):
- Summary cards, empty states, form groups, toasts, filter bar, dashboard refresh.
- Any changes to color tokens, new accent colors, or new severity hues.
- Dark/light theme toggle UX (separate proposal).
- Datetime input UX (separate proposal).

## Capabilities

### New Capabilities

- `ui-component-identity`: The design contracts that give TimeTrak components distinct identity within the calm/tool-like brand — shape-language taxonomy (pill/rectangle/circle), two-weight border system, numeric-text contract, accent-rationing rules, timer-as-signature-object contract, and the showcase/review checklist that enforces them. This is distinct from `ui-foundation` (which owns *tokens*) and `ui-partials` (which owns *partial structure and events*) — it owns the *visual grammar* that components MUST follow regardless of which tokens or partials they use.

### Modified Capabilities

- `ui-foundation`: Adds a named `--radius-pill` scale token so action-shape components (buttons, timer control) can reference the pill radius via a token instead of a raw `999px` value, preserving the "no raw numeric values for scale concerns" rule.
- `ui-partials`: Adds a new canonical `partials/status_chip`, renames `partials/timer_widget` → `partials/timer_control` with identity requirements (running-timer fill + pulse, tabular-nums elapsed time, distinct Stop affordance), and MODIFIES the accepted "Table shell and empty state partials" requirement to drop the never-extracted `partials/table_shell` wrapper in favor of a canonical `.table` CSS contract consumed by every domain list (hairline dividers, uppercase headers, `.col-num`, 2px accent selected edge). The partials README already documents the wrapper deferral; this change aligns the accepted spec with that reality. Existing slot/context/event contracts on other partials are unchanged.
- `ui-showcase`: The gallery MUST render each sharpened component in every documented state (idle, running, selected, focus, error) and MUST surface the component-identity checklist so reviewers can verify compliance visually.

## Impact

- **CSS:** `web/static/css/` — new component-scoped rules for timer, table, chip; no new semantic aliases in tokens. Any raw pixel values or shadows surfaced during the audit get migrated to existing scale tokens.
- **Templates:** `web/templates/partials/status_chip.html`, `partials/table_shell.html`, `partials/timer_control.html` (or the current timer markup location), and any domain templates that render ad-hoc versions of these patterns get migrated to the canonical partials.
- **Showcase:** `web/templates/showcase/` — new or expanded gallery entries for timer (idle + running), table (default + selected + focused row), status chip (all variants). Existing axe-smoke browser test coverage extends to cover new states.
- **Tests:** `internal/e2e/browser/` — extend focus-ring contract, brand-surface-axe-smoke, and showcase-axe-smoke tests to cover sharpened components; add contract tests asserting tabular-nums presence on numeric columns and the accent-rationing rule (no accent fills outside the permitted list).
- **Docs:** `docs/timetrak_ui_style_guide.md` gets a new "Component identity" section cross-linked from `docs/timetrak_brand_guidelines.md`. No docs/ replaces the spec — the spec in `openspec/specs/ui-component-identity/` is canonical.
- **No backend, DB, or domain-logic changes.** No new dependencies. No migrations. No breaking API changes.
- **Risk:** visual regressions on existing screens are likely during rollout — mitigated by the showcase gallery acting as the first consumer and by browser contract tests. The accent-rationing rule in particular may surface unintended accent usage in current templates that will need cleanup in this change.
