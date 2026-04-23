## MODIFIED Requirements

### Requirement: Scale Tokens

TimeTrak SHALL define scale tokens for spacing, radius, typography, motion, elevation, z-index layers, and breakpoints as CSS custom properties. Components SHALL reference scale tokens and MUST NOT use raw numeric values for these concerns.

- **Spacing** MUST be an 8px-based scale (`--space-1` = 4px through `--space-8` = 48px or equivalent named set documented in the token file).
- **Radius** MUST provide at minimum `--radius-sm` (small controls, inputs, chips, badges), `--radius-md` (larger cards, modals), and `--radius-pill` (fully rounded actions — buttons, timer control). Components authoring pill-shaped actions MUST reference `var(--radius-pill)` and MUST NOT use raw `999px` or equivalent literal values.
- **Typography** MUST define a font-family token pair (`--font-sans`, `--font-mono`) and a documented static size / weight / line-height set. Fluid or clamp-based scales are out of scope.
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
