## Context

TimeTrak's UI is server-rendered Go `html/template` plus HTMX. During MVP, each domain (`clients/`, `projects/`, `rates/`, `tracking/`, `reporting/`, `dashboard/`) shipped its own forms, tables, row partials, flash, empty-state, and filter-bar markup. A handful of partials already live under `web/templates/partials/` (`client_row.html`, `project_row.html`, `entry_row.html`, `rate_row.html`, `flash.html`, `spinner.html`, `confirm_dialog.html`, `pagination.html`, `timer_widget.html`, `rate_form.html`, `report_*`, `tracking_error.html`, `dashboard_summary.html`), but the naming, slot shape, and HTMX event wiring are not uniform.

This change consolidates those patterns before additional Stage 2 and Stage 3 work compounds the divergence. It is a template refactor only — no handlers, routes, or specs change.

Binding constraints carried in from the project:

- `html/template` at startup (layouts + partials auto-included by `internal/shared/templates`).
- HTMX event contracts already in flight: `timer-changed`, `entries-changed`, `clients-changed`, `projects-changed`, `rates-changed`.
- `data-focus-after-swap` convention handled by `web/static/js/app.js` on `htmx:afterSwap`.
- WCAG 2.2 AA is the enforced baseline.
- No SPA frameworks, no client-state libraries, no new CSS frameworks, no new JS deps.

## Goals / Non-Goals

**Goals:**

- Produce a documented, stable set of canonical partials under `web/templates/partials/` that domain templates compose.
- Establish a single naming and slot convention so future partials slot in predictably.
- Preserve existing HTMX event contracts and focus behavior exactly (no silent regressions).
- Codify per-partial WCAG 2.2 AA expectations so accessibility is not re-derived per domain.
- Reduce copy-pasted markup in `clients/`, `projects/`, `rates/`, `tracking/`, `reporting/`, and `dashboard/` templates.

**Non-Goals:**

- No brand or visual redesign.
- No design-token / color-system changes.
- No new CSS framework, component library package, or JS dependency.
- No changes to handlers, routes, or the accepted specs in `openspec/specs/`.
- No new product features. If a partial exposes a capability no template currently needs, it is not extracted.

## Decisions

### 1. Convention: one partial = one `{{define}}` block, one expected `.` shape

Each partial file under `web/templates/partials/<name>.html` defines a single block `{{define "partials/<name>"}}`. Templates invoke it with `{{template "partials/<name>" .Context}}` where `.Context` is a documented struct or `dict` with a fixed key set. Optional slots are passed via `dict` keys with documented defaults.

Rationale: matches existing repo idiom, keeps static analysis of call sites grep-able, avoids the ambiguity of anonymous partials. Alternatives considered: one-partial-per-file with implicit root name (rejected — ambiguous when a file defines multiple blocks); Go-side view-model structs per partial (rejected for this refactor — too invasive; can be layered later without breaking the template convention).

### 2. Extraction bar: only patterns used in 2+ domains become canonical partials

A partial is promoted to canonical only when its markup currently appears in two or more domains. One-offs stay in their domain template.

Rationale: prevents partial proliferation and over-abstraction. Alternatives: extract every visually similar block (rejected — breeds near-identical variants and slot explosion).

### 3. Canonical partial set

After the inventory, the target canonical set is:

- `partials/form_field` — label + native control + hint + inline error. One input per invocation.
- `partials/form_errors` — top-of-form error summary, `role="alert"`, links to fields via `aria-describedby`.
- `partials/table_shell` — thead + tbody wrapper with consistent empty state and optional sort affordances.
- `partials/empty_state` — domain-agnostic empty block (icon-free, copy-first), accepts a title, body, and optional primary action slot.
- `partials/filter_bar` — form wrapper with `hx-trigger="change delay:200ms"` convention for filterable tables.
- `partials/pagination` — already exists; normalize `.` shape and aria-labeling.
- `partials/flash` — already exists; normalize severity levels (`info`, `success`, `warn`, `error`) and `role="status"`/`role="alert"` mapping.
- `partials/spinner` — already exists; normalize `aria-live` and `data-focus-after-swap` pairing.
- `partials/confirm_dialog` — already exists as `<dialog>`; document when to use it vs. `hx-confirm`.
- `partials/timer_widget` — already exists; treat as domain partial but document its event contract.
- Row partials (`client_row`, `project_row`, `entry_row`, `rate_row`) — keep per-domain, but standardize: every row renders with stable `id="<domain>-row-<uuid>"`, supports OOB swap, and advertises the `*-changed` event it triggers.

Rationale: matches actual observed duplication; each item either consolidates existing copy-paste or normalizes an existing partial.

### 4. HTMX event contract documentation

A new `web/templates/partials/README.md` documents:

- Naming: `partials/<name>` block names, file naming matches block name.
- Slot convention: keys passed via `dict`, defaults, required vs optional.
- Event names: the authoritative list of `*-changed` events and which partials emit / listen for them.
- Focus convention: when to set `data-focus-after-swap`, and the rule that any modal/dialog swap MUST set it on a reasonable target.
- OOB swap rules: row partials use `hx-swap-oob="true"` with stable ids; `flash` partial is the canonical OOB target for user-facing messages.

Rationale: the contract is the point of this change. Without a written convention, divergence returns. Alternatives: inline comments at each partial (rejected — discoverability is worse).

### 5. Accessibility bar documented per partial

Each partial in the README lists its WCAG 2.2 AA obligations (label source, focus target, contrast requirement, non-color status cue). The README is the reference that future changes cite instead of re-deriving.

### 6. No new template funcs unless required

The existing template func set (`dict`, `seq`, `formatDate`, `formatTime`, `formatDuration`, `formatMinor`, `iso`, `add`, `sub`) is sufficient. Adding funcs is a last resort; if any partial forces a new helper, that decision is called out in the migration PR.

### 7. Migration sequence: consolidate first, then migrate per domain

The implementation order is: (1) inventory and write README, (2) extract/normalize canonical partials, (3) migrate domains one at a time (`clients` → `projects` → `rates` → `tracking` → `reporting` → `dashboard`), re-verifying events and focus after each. Rationale: small, reviewable steps; each domain migration can be reverted independently if an HTMX contract regresses.

## Risks / Trade-offs

- **Over-abstraction via too many slots** → Mitigation: the extraction bar (§ Decision 2); slot keys are documented and capped; if a partial needs more than four optional slots, split it instead of adding a fifth.
- **Partial proliferation** → Mitigation: canonical set enumerated up front (§ Decision 3); anything outside the list stays in the domain template.
- **Template func bloat** → Mitigation: § Decision 6; any new func requires an explicit note in the migration PR.
- **Silent HTMX contract regression during migration** → Mitigation: per-domain migration task includes a checklist (event names emitted, swap targets, focus-after-swap targets) and a manual smoke of start/stop timer, entry CRUD, client/project/rate CRUD, and report filter.
- **Accessibility regression** → Mitigation: each migrated template gets an a11y pass (keyboard, focus after swap, screen-reader labels, contrast spot-check) before moving to the next domain.
- **Review size** → Mitigation: README + canonical partials can ship in one PR; each domain migration ships in its own PR.

## Migration Plan

1. Land the README and canonical partial set (no domain templates changed yet). Existing partials remain; new ones are additive.
2. Migrate domains one at a time, deleting now-dead inline markup as each domain switches over.
3. When all domains are on canonical partials, remove any deprecated partial files and close the change.

Rollback: each per-domain migration is independently revertable. The README and canonical partials are additive and safe to leave in place even if a specific domain migration is reverted.

## Open Questions

- Should row partials share a single `partials/row` base, or stay per-domain? Current lean: stay per-domain (§ Decision 3). Revisit if all four row partials converge on identical structure after migration.
- Should `confirm_dialog` replace `hx-confirm` for destructive deletes project-wide, or remain an opt-in for flows that need focus trapping? Current lean: opt-in; document the decision criteria in the README.
