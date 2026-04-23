## ADDED Requirements

### Requirement: Shape language taxonomy

TimeTrak SHALL enforce a three-shape taxonomy across every component, where each shape has a fixed semantic role:

- **Pill** (fully rounded, `var(--radius-pill)`): SHALL be used for interactive actions — buttons and the timer control. MUST NOT be used for status, metadata, or presence indicators.
- **Rectangle** (`var(--radius-sm)` = 4px): SHALL be used for status and metadata — chips, badges, tags. MUST NOT be used for actions.
- **Circle** (`50%`): SHALL be used for presence dots and avatar placeholders — the running-timer indicator, user avatar fallback. MUST NOT be used for actions or status labels.

New shapes MUST NOT be introduced without a change proposal that amends this taxonomy.

#### Scenario: Chip is authored as a pill

- **WHEN** a reviewer sees a chip or badge component using `border-radius: 999px` or `var(--radius-pill)`
- **THEN** the review blocks the change until the chip uses `var(--radius-sm)`.

#### Scenario: Button is authored as a rectangle

- **WHEN** a reviewer sees a button or timer control using `var(--radius-sm)` or any radius other than the pill
- **THEN** the review blocks the change until the control uses `var(--radius-pill)`.

#### Scenario: Contributor proposes a fourth shape

- **WHEN** a contributor proposes a new shape (e.g. a trapezoid, a tab shape) for a component
- **THEN** the proposal MUST first amend this requirement's taxonomy before the component can land.

### Requirement: Two-weight border contract

TimeTrak SHALL use exactly two border weights across the app to encode structure vs state:

- **1px solid `var(--color-border)`** — structure, at-rest surfaces: card perimeters, inputs at rest, table horizontal dividers.
- **2px solid `var(--color-accent)` or `var(--color-danger)`** — state: focus ring, selected/focused row edge, running timer border, validation error edge. The danger variant SHALL use `--color-danger` but keep the 2px weight.

Dashed, double, inset, and outset border styles are prohibited in component CSS. Shadow elevation (`--shadow-*`) MUST NOT substitute for a border to convey state.

#### Scenario: Component uses a 1.5px or 3px border

- **WHEN** a reviewer sees a component CSS rule with `border-width` other than 1px or 2px
- **THEN** the review blocks the change until the border conforms to the two-weight system.

#### Scenario: Component uses a shadow to indicate focus or selection

- **WHEN** a reviewer sees a component CSS rule applying `box-shadow` to convey focused, selected, running, or error state
- **THEN** the review blocks the change until state is conveyed via the 2px-accent (or 2px-danger) border contract.

#### Scenario: Error state is authored with a red background fill

- **WHEN** a reviewer sees a form input with `background: var(--color-danger-soft)` used as the sole error cue
- **THEN** the review blocks the change until the 2px `var(--color-danger)` edge is present (the soft fill MAY coexist but is not sufficient alone).

### Requirement: Numeric text contract

Every element that renders a duration, monetary amount, hourly rate, or integer count SHALL apply `font-variant-numeric: tabular-nums`. In table cells, numeric columns SHALL be right-aligned via a documented class (`.col-num`) or `data-col-kind="numeric"` attribute. In summary cards and inline contexts, numeric figures MAY be left-aligned but MUST retain tabular numerals.

#### Scenario: Duration column in the entries table

- **WHEN** the entries list renders the `Duration` column for any row
- **THEN** the `<td>` MUST have `font-variant-numeric: tabular-nums` and `text-align: right`.

#### Scenario: Running timer elapsed time

- **WHEN** the timer is in `running` state and renders elapsed `HH:MM:SS`
- **THEN** the element containing the elapsed time MUST have `font-variant-numeric: tabular-nums` so digits do not reflow as seconds advance.

#### Scenario: Summary card figure

- **WHEN** a dashboard summary card renders a headline figure (e.g. `Billable this week`)
- **THEN** the figure element MUST have `font-variant-numeric: tabular-nums`.

### Requirement: Accent rationing

The accent color (`var(--color-accent)`, `var(--color-accent-soft)`, `var(--color-accent-hover)`, and the equivalent `--accent*` legacy aliases) SHALL appear only on surfaces that answer a "which one?" question for the user. The permitted surfaces are:

1. The running-timer fill, 2px border, leading dot, and elapsed-time readout.
2. The focus ring (as defined in `ui-foundation`).
3. The selected/focused table-row 2px inside-left edge rule.
4. The primary button fill, border, and hover (`.btn-primary`).
5. Link text and link hover (`a`, `a:hover`) — the universal "click here" signal.
6. The active/current navigation item — `[aria-current="page"]` in `.nav` uses accent-soft fill + accent text + accent left-edge rule to say "you are here."
7. Billable and running status chips — `.tt-chip-billable` and `.tt-chip-running` use accent-soft fill + accent text + accent border to say "this entry / this timer is the billable / running one."
8. The running-entry card top border (reserved — introduced by the follow-on `sharpen-dashboard-and-empty-states` change).

Accent usage outside this list is prohibited. Secondary buttons (`.btn`, `.btn-ghost`), at-rest inputs, non-billable / archived / draft chips, hovered rows (background change only — no accent), at-rest cards, `<thead>` headers, and any other chrome MUST NOT use the accent color. The underlying principle: accent answers "which one?" — spread it across generic chrome and it stops answering anything.

An automated CSS audit SHALL enforce this list: the audit MUST enumerate every selector in compiled CSS that references `var(--color-accent)` or `var(--color-accent-soft)` and fail the build when a selector is not on the allow-list.

#### Scenario: Contributor adds accent to a hover state

- **WHEN** a contributor adds `background: var(--color-accent-soft)` to a table row's `:hover` rule
- **THEN** the CSS audit test MUST fail naming the offending selector, and the rule MUST be removed or the proposal MUST amend this requirement's allow-list.

#### Scenario: Secondary button uses accent fill

- **WHEN** a reviewer sees a `.btn-ghost` or non-primary `.btn` variant with `background: var(--color-accent)`
- **THEN** the review blocks the change; non-primary buttons MUST use neutral tokens.

#### Scenario: Permitted accent usage passes audit

- **WHEN** the CSS audit runs over compiled CSS and finds `var(--color-accent*)` tokens only within the allow-listed selectors (running timer; focus ring; selected/focused table row; `.btn-primary`; `a` / `a:hover`; `.nav a[aria-current="page"]`; `.tt-chip-billable`; `.tt-chip-running`)
- **THEN** the audit passes.

### Requirement: Timer is a first-class signature component

TimeTrak's timer control SHALL be authored as its own component (not a button variant) with two documented states — `idle` and `running` — and explicit visual rules for each:

- **Idle state:** a start-entry form whose signature pill is the primary submit button labelled `Start timer` (pill shape via `var(--radius-pill)`, primary button fill). Project picker and optional description fields are peers of the pill; the pill is the terminal action. A leading neutral circle dot on the pill is OPTIONAL in the idle form to keep the inline row dense, but the pill MUST inherit the shared `.btn-primary` pill styling so it conforms to the shape-language contract.
- **Running state:** a single pill container (not a form with peers) with `var(--radius-pill)`, `var(--color-accent-soft)` fill, 2px `var(--color-accent)` border, a leading pulsing accent circle dot, the elapsed `HH:MM:SS` in tabular-nums at a weight and size equal to or greater than the project name, and a separate `Stop` control that is NOT visually identical to the idle `.btn-primary` pill (it MUST use a distinct button variant — ghost or secondary).

The pulsing dot MUST collapse to a static filled dot when `@media (prefers-reduced-motion: reduce)` is in effect.

The timer control is the only surface in the app that uses accent as a *fill*; other accent usages are edges or rings.

#### Scenario: Timer in idle state

- **WHEN** a user views a page with no running entry and the timer control is rendered
- **THEN** a start-entry form is rendered whose primary submit button is a pill (inheriting `.btn-primary` pill styling) labelled `Start timer`.

#### Scenario: Timer transitions to running via HTMX

- **WHEN** the user starts a timer and the HTMX swap replaces the control
- **THEN** the new control is a pill with `--color-accent-soft` fill, a 2px `--color-accent` border, a pulsing accent dot, an elapsed-time readout in `tabular-nums`, and a visually distinct `Stop` affordance.

#### Scenario: User has reduced-motion preference

- **WHEN** the running timer renders under `prefers-reduced-motion: reduce`
- **THEN** the accent dot is static (no pulse animation) and all other running-state styling is preserved.

#### Scenario: Stop button is styled identically to Start pill

- **WHEN** a reviewer sees the running-state `Stop` affordance rendered as the same neutral pill as the idle `Start timer`
- **THEN** the review blocks the change until `Stop` has a distinct visual treatment (e.g. ghost/secondary style) from the idle start pill.

### Requirement: Component identity review checklist

TimeTrak SHALL publish a component-identity review checklist in `docs/timetrak_ui_style_guide.md` and render it at the top of the `/dev/showcase` index page. The checklist MUST enumerate at minimum:

1. Does the component use the correct shape from the shape-language taxonomy?
2. Does every border conform to the two-weight contract (1px structure / 2px state)?
3. Does every number use `tabular-nums`?
4. Does the component consume the accent color only on an allow-listed surface?
5. Does every state (default, hover, focused, selected, error, empty, running where applicable) render in `/dev/showcase`?

UI-affecting proposals and pull requests SHALL cite the checklist as review criteria; cite the specific item being addressed or consciously waived rather than re-deriving the rule.

#### Scenario: Reviewer opens showcase index before approving a UI PR

- **WHEN** a reviewer opens `/dev/showcase`
- **THEN** the checklist is rendered above the component gallery so the reviewer can cross-reference live components against the contract.

#### Scenario: UI PR lacks a showcase entry for a new state

- **WHEN** a PR adds a new component state (e.g. `selected` for a new list) without a corresponding showcase entry
- **THEN** the review blocks the change citing checklist item 5.

#### Scenario: UI PR proposes a component that breaks shape language

- **WHEN** a PR proposes a pill-shaped chip or a rectangle-shaped button
- **THEN** the review blocks the change citing checklist item 1 and the shape-language taxonomy requirement.
