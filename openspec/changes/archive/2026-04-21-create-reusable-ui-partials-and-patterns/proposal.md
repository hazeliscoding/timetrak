## Why

During MVP build-out, recurring UI patterns (row partials, forms with inline errors, flash messages, spinners, empty states, confirm prompts, timer widget, filter bars, paginated tables) emerged as copy-pasted markup across domain templates (`clients/`, `projects/`, `rates/`, `tracking/`, `reporting/`). Divergence is already visible in inline-error formatting, empty-state copy, table shells, and HTMX event wiring — which risks inconsistent focus behavior after swaps, uneven WCAG 2.2 AA compliance, and a fragile base for the Stage 2 component library foundation. Consolidating now, before more domains land, keeps the server-rendered + HTMX model coherent.

## What Changes

- Inventory existing partials under `web/templates/partials/` and catalogue duplicated markup across domain templates.
- Define a documented naming and slot convention for partials (file naming, `{{define}}` block names, expected `.` shape, optional slots via `dict`).
- Extract shared building blocks as canonical partials, consolidating existing copies:
  - `form_field` (label + input + inline error + hint, honoring native control semantics)
  - `form_errors` (summary block near top of forms for screen readers)
  - `table_shell` (consistent table head, sort affordance, empty state, loading state)
  - `row` convention (domain row partials follow one shape for HTMX OOB swaps)
  - `flash`, `spinner`, `confirm`, `pagination`, `filter_bar`, `empty_state`
- Document HTMX event contracts each partial participates in: `hx-swap-oob`, `data-focus-after-swap`, `*-changed` triggers (`timer-changed`, `entries-changed`, `clients-changed`, `projects-changed`, `rates-changed`).
- Document per-partial WCAG 2.2 AA expectations (labels, focus, contrast, target size, non-color status conveyance).
- Migrate existing domain templates (`clients/`, `projects/`, `rates/`, `tracking/`, `reporting/`) to compose the canonical partials; remove now-duplicate markup.
- Add a short partials README under `web/templates/partials/README.md` that codifies conventions for future changes.

Out of scope: brand or visual redesign, design-token changes, introducing a CSS framework, new JS dependencies, a separate "component library package" (that is a later follow-up), and any new product features.

## Capabilities

### New Capabilities

- `ui-partials`: codifies TimeTrak's reusable server-rendered partial conventions — naming, slot shape, HTMX event contracts (`*-changed`, `hx-swap-oob`, `data-focus-after-swap`), and per-partial WCAG 2.2 AA obligations. This becomes the accepted baseline future UI changes cite.

### Modified Capabilities

_None. Domain-level behavior (auth, workspace, clients, projects, tracking, rates, reporting) is unchanged. This change is confined to template composition and does not alter accepted requirements in the existing domain specs._

## Impact

- **Templates**: Files under `web/templates/partials/` are reorganized; domain templates under `web/templates/{clients,projects,rates,tracking,reporting,dashboard}/` are edited to compose canonical partials. No route or handler changes.
- **Template funcs**: May add one or two small helpers (e.g. `slot` defaulting) only if existing `dict`/`seq` are insufficient; avoid func bloat.
- **HTMX contracts**: Event names and `data-focus-after-swap` behavior preserved exactly; any partial that emits or listens for an event is re-verified.
- **Docs**: New `web/templates/partials/README.md` documenting conventions. `docs/timetrak_ui_style_guide.md` is unchanged in this change (a later follow-up may cross-link).
- **Risks**: over-abstraction (partials with too many slots), partial proliferation (many near-identical partials), template-func bloat, and silently breaking HTMX event contracts during migration. Mitigated by a conservative extraction bar (only patterns used in 2+ domains) and a migration checklist that re-verifies each swap target and trigger.
- **Follow-ups**: separate Stage 2 changes for a token/theming audit and for a formal component library package.
