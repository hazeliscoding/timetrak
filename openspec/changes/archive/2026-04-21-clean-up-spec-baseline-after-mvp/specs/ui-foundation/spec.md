## MODIFIED Requirements

<!--
  NO-OP DELTA. This change is prose-only: it rewrites the non-normative
  `## Purpose` section of `openspec/specs/ui-foundation/spec.md` and does NOT
  alter any requirement. The requirement block below is copied verbatim
  from the baseline so the OpenSpec validator recognises a MODIFIED delta.
  See proposal.md and design.md (Decision 1).
-->

### Requirement: Two-Layer Token Taxonomy

TimeTrak's CSS token system SHALL be organized into two layers: **primitive ramps** (palette anchors, private to the foundation) and **semantic aliases** (public tokens consumed by components). Components SHALL consume only semantic aliases, never primitive ramps or raw values.

The primitive ramp layer MUST include at minimum: a neutral ramp (`--neutral-0` through `--neutral-900`), an accent ramp (`--accent-50` through `--accent-900`), and severity anchors for red, amber, and green (at least a 500-weight step per hue for both light and dark themes).

The semantic alias layer MUST include at minimum: `--color-bg`, `--color-surface`, `--color-surface-alt`, `--color-text`, `--color-text-muted`, `--color-border`, `--color-border-strong`, `--color-accent`, `--color-accent-hover`, `--color-accent-soft`, `--color-focus`, and the severity pairs `--color-success` / `--color-success-soft`, `--color-warning` / `--color-warning-soft`, `--color-danger` / `--color-danger-soft`, `--color-info` / `--color-info-soft`.

New semantic aliases MUST NOT be added without a change proposal that extends this requirement. Component-scoped tokens (e.g. a hypothetical `--btn-primary-bg`) are NOT part of the public semantic layer and, if introduced, MUST be declared locally within the component's CSS scope rather than in the global token file.

#### Scenario: Component references a semantic alias

- **WHEN** a component CSS rule needs a surface colour
- **THEN** it references `var(--color-surface)` (or another documented semantic alias) and does NOT reference `var(--neutral-0)` or a raw hex value.

#### Scenario: Token file is rebuilt from primitives

- **WHEN** the token file is re-generated or reviewed
- **THEN** every semantic alias is defined as a `var(--<primitive>)` expression (or as a `var()` to another alias), and every primitive ramp step is defined as a raw color value.

#### Scenario: Contributor attempts to add a new semantic alias

- **WHEN** a proposal extends the semantic alias list
- **THEN** the proposal explicitly amends this requirement's enumeration; adding an alias without a spec update is rejected in review.
