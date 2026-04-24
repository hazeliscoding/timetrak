## MODIFIED Requirements

### Requirement: Accent rationing

The accent color (`var(--color-accent)`, `var(--color-accent-soft)`, `var(--color-accent-hover)`, and the equivalent `--accent*` legacy aliases) SHALL appear only on surfaces that answer a "which one?" question for the user. The permitted surfaces are:

1. The running-timer fill, 2px border, leading dot, and elapsed-time readout.
2. The focus ring (as defined in `ui-foundation`).
3. The selected/focused table-row 2px inside-left edge rule.
4. The primary button fill, border, and hover (`.btn-primary`).
5. Link text and link hover (`a`, `a:hover`) — the universal "click here" signal.
6. The active/current navigation item — `[aria-current="page"]` in `.nav` uses accent-soft fill + accent text + accent left-edge rule to say "you are here."
7. Billable and running status chips — `.tt-chip-billable` and `.tt-chip-running` use accent-soft fill + accent text + accent border to say "this entry / this timer is the billable / running one."
8. The selected segment of the theme switch — `.tt-theme-seg[aria-pressed="true"]` uses accent-soft fill + accent text + a 2px accent inset edge to say "this theme is active."
9. The running-entry card top border (reserved — introduced by the follow-on `sharpen-dashboard-and-empty-states` change).

Accent usage outside this list is prohibited. Secondary buttons (`.btn`, `.btn-ghost`), at-rest inputs, non-billable / archived / draft chips, hovered rows (background change only — no accent), at-rest cards, `<thead>` headers, unselected theme-switch segments, and any other chrome MUST NOT use the accent color. The underlying principle: accent answers "which one?" — spread it across generic chrome and it stops answering anything.

An automated CSS audit SHALL enforce this list: the audit MUST enumerate every selector in compiled CSS that references `var(--color-accent)` or `var(--color-accent-soft)` and fail the build when a selector is not on the allow-list.

#### Scenario: Contributor adds accent to a hover state

- **WHEN** a contributor adds `background: var(--color-accent-soft)` to a table row's `:hover` rule
- **THEN** the CSS audit test MUST fail naming the offending selector, and the rule MUST be removed or the proposal MUST amend this requirement's allow-list.

#### Scenario: Secondary button uses accent fill

- **WHEN** a reviewer sees a `.btn-ghost` or non-primary `.btn` variant with `background: var(--color-accent)`
- **THEN** the review blocks the change; non-primary buttons MUST use neutral tokens.

#### Scenario: Permitted accent usage passes audit

- **WHEN** the CSS audit runs over compiled CSS and finds `var(--color-accent*)` tokens only within the allow-listed selectors (running timer; focus ring; selected/focused table row; `.btn-primary`; `a` / `a:hover`; `.nav a[aria-current="page"]`; `.tt-chip-billable`; `.tt-chip-running`; `.tt-theme-seg[aria-pressed="true"]`)
- **THEN** the audit passes.

#### Scenario: Unselected theme segment uses accent

- **WHEN** a reviewer sees a `.tt-theme-seg` segment rule applying `var(--color-accent)` to the at-rest (unpressed) state
- **THEN** the review blocks the change; only the `aria-pressed="true"` segment may consume accent.
