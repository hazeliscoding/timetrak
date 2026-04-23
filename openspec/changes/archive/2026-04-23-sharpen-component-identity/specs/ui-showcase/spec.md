## ADDED Requirements

### Requirement: Component identity states rendered per component

For every component governed by `ui-component-identity` (timer control, data table, status chip, and any additional components added to the identity contract in future changes), the showcase component catalogue SHALL render one live example per documented identity state. At minimum:

- **Timer control:** `idle`, `running`, and — when the reduced-motion media query is simulated via a toggle or an in-page control group — `running` with the static-dot fallback. Each rendering MUST use the real `partials/timer_control`.
- **Data table (via `.table` CSS contract):** a dedicated gallery section renders a live `<table class="table">` demonstrating `default row`, `hover row` (via a CSS `:hover` demonstration note or an `is-hover` simulation class), `selected/focused row` (`aria-selected="true"`), and `empty` (using `partials/empty_state`). Numeric-column treatment (`.col-num` / `tabular-nums`, right-aligned) MUST be visible in at least one row.
- **Status chip (via `partials/status_chip`):** one rendering per enumerated `kind` × documented `variant`, plus one example that demonstrates non-color-only conveyance (glyph + label).

These identity-state renderings are additive to the existing variant-permutation coverage and do not replace it.

#### Scenario: Timer shows idle and running

- **WHEN** a reader opens the timer entry in `/dev/showcase`
- **THEN** at least two live renderings are present — one for `idle` (neutral pill, `Start timer` label) and one for `running` (accent-soft fill, 2px accent border, tabular-nums elapsed time, distinct `Stop` affordance).

#### Scenario: Table selected-row state is rendered

- **WHEN** a reader opens the table-states gallery section in `/dev/showcase/components`
- **THEN** a live rendering of a row with the selected/focused state (2px accent inside-left edge via `box-shadow: inset 2px 0 0 0 var(--color-accent)`, driven by `aria-selected="true"`) is present and visually distinguishable from the default and hover rows, and at least one column is marked `.col-num` demonstrating `tabular-nums` right-alignment.

#### Scenario: Chip state conveyance is shown

- **WHEN** a reader opens the status-chip entry in `/dev/showcase`
- **THEN** each enumerated `kind` renders once per documented `variant`, and at least one example explicitly demonstrates a glyph-plus-label combination to satisfy the non-color-alone rule.

### Requirement: Component identity checklist rendered on showcase index

The `/dev/showcase` index page SHALL render the component-identity review checklist defined in `ui-component-identity` above the component catalogue listing. The checklist MUST be sourced from the same canonical document (`docs/timetrak_ui_style_guide.md` or the spec) rather than re-authored in the showcase templates, so the two cannot drift.

Each checklist item MUST link or cross-reference the corresponding requirement in `openspec/specs/ui-component-identity/spec.md`.

#### Scenario: Reader opens showcase index

- **WHEN** an authenticated user opens `/dev/showcase` in a dev environment
- **THEN** the page renders the component-identity checklist (shape language, two-weight borders, tabular-nums, accent rationing, state coverage) above the component catalogue.

#### Scenario: Checklist item links to its requirement

- **WHEN** a reader clicks a checklist item
- **THEN** the link navigates to or stably references the matching requirement in `ui-component-identity`.

#### Scenario: Checklist drifts from the spec

- **WHEN** the showcase checklist text differs from the canonical source (style guide or spec)
- **THEN** the showcase test suite MUST fail until the two are reconciled.
