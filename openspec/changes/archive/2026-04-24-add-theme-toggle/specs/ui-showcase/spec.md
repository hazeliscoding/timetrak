## ADDED Requirements

### Requirement: Theme switch catalogue entry documents every selected state

The component catalogue SHALL include a `theme_switch` entry that renders the real `partials/theme_switch` partial three times, once per selected state (`light-selected`, `dark-selected`, `system-selected`), so a reviewer can verify the sharpened selected-segment treatment without toggling the live control. Each example MUST invoke the partial with an `InitialSelected` dict key matching the example's name, MUST cite the `ui-partials` and `ui-component-identity` requirements that govern the partial, and MUST include an accessibility note documenting the `role="radiogroup"` + `aria-checked` contract.

#### Scenario: Showcase entry renders three selected states

- **WHEN** a reader opens the `theme_switch` entry in `/dev/showcase/components`
- **THEN** exactly three live renderings are present, one per selected state
- **AND** each rendering invokes the real `partials/theme_switch` partial (not a re-implementation)
- **AND** the selected segment in each rendering carries `aria-pressed="true"` and `aria-checked="true"`

#### Scenario: Showcase entry cross-references spec

- **WHEN** a reader opens the `theme_switch` entry
- **THEN** the entry's SpecRef MUST point at `openspec/specs/ui-partials/spec.md` (Theme switch partial)
- **AND** the entry's A11yNotes MUST mention the radiogroup role, the dual `aria-pressed` + `aria-checked` attribute contract, and the accent-rationing allow-list entry that governs the selected segment.
