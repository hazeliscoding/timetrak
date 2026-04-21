## Why

The MVP shipped with inconsistent focus handling, partial WCAG 2.2 AA coverage, and pattern drift across domains: destructive delete flows differ between Clients, Projects, and Rates; several HTMX swap targets lack `data-focus-after-swap`; status pills (`Archived`, `Running`, `No rate`, `Billable`) rely on color alone in a few places; forms mix inline-error placements and missing `aria-describedby` wiring; and tables lack consistent `<caption>`, `scope`, and numeric-alignment semantics. Closing these gaps now — before we introduce reusable partials (`create-reusable-ui-partials-and-patterns`) and a component library foundation (`establish-custom-component-library-foundation`) — prevents us from baking the drift into shared components.

This change is strictly polish and consistency against the accepted Stage 2 baseline. It does not add new domain capability, does not introduce new pages, and does not overhaul tokens or theming.

## What Changes

- **Focus management after HTMX swaps**: audit every `hx-target` on the dashboard, time, clients, projects, rates, and reports pages; ensure swap targets that receive user intent (inline-edit rows, rate form, timer widget, filter bar) carry `data-focus-after-swap` on the right element so the existing `web/static/js/app.js` helper lands focus correctly.
- **Destructive confirmation consistency**: standardize the two patterns — native `hx-confirm` for list-row deletes (Clients, Projects, Rates rows) and `partials/confirm_dialog.html` for flows that warn about side effects (stopping a running timer that would discard an unsaved description, archiving a client with active projects). Document which pattern each surface uses.
- **Form field consistency**: every input has a visible `<label>`, required fields are marked in both text and `aria-required`, inline validation errors are wired via `aria-describedby` pointing at a predictable `#<field>-error` node, and each form renders a top-of-form error summary with `role="alert"` when server validation fails.
- **Table semantics**: Clients, Projects, Rates, Reports, and the Time entries list each gain a `<caption>` (visually hidden where the page title already covers it), `<th scope="col">`/`<th scope="row">` where applicable, numeric columns aligned via a utility class not inline styles, and sortable columns — where present — expose state via `aria-sort`.
- **Status communication without color**: `Archived`, `Running`, `No rate`, `Billable`, and rate-scope pills gain a text label or icon in addition to color, and token-level contrast for all pill variants is verified against 4.5:1.
- **Empty / loading / error states**: every HTMX-swapped region has an explicit empty-state partial (where missing) and a polite `aria-live` region for asynchronous status changes; `partials/spinner.html` is audited for `aria-hidden` / `role="status"` usage.
- **Skip link + landmarks + titles**: the skip-to-content link is verified against the first focusable element on every page; landmark roles (`banner`, `navigation`, `main`, `contentinfo`) are complete and unique; every page sets a meaningful `<title>`.
- **Keyboard navigability**: the timer widget, the entry inline editor, the rate form scope-toggle, and the report filter bar are walked keyboard-only and gaps are closed (missing `Enter`/`Escape` handling, tab traps, invisible focus).
- **Reduced motion**: any CSS transition introduced to achieve focus/hover/state polish is wrapped in `@media (prefers-reduced-motion: reduce)` to collapse to instant state changes.

## Capabilities

### New Capabilities

_None._ This change is a polish pass against accepted behavior; no new capability is introduced.

### Modified Capabilities

Only capabilities whose behavioral contract (not just markup) tightens:

- `tracking`: tighten accessibility requirements on the timer widget and the entry inline editor (focus-after-swap target, keyboard contract, destructive-confirm pattern for stop/discard).
- `rates`: tighten accessibility requirements on the rate form (scope toggle announced, inline errors wired via `aria-describedby`, focus returns to a predictable anchor after save/cancel) and on the rates table (status pills non-color-only).
- `reporting`: tighten accessibility requirements on the filter bar and the results table (caption, `scope`, `aria-sort` where applicable, empty/loading/error states announced via `aria-live`).
- `clients`: tighten accessibility requirements on the clients list (table semantics, destructive-confirm pattern, archived-status non-color-only).
- `projects`: tighten accessibility requirements on the projects list (same shape as `clients`).
- `auth`: tighten accessibility requirements on the login and signup forms (label + `aria-describedby` error wiring, error-summary with `role="alert"`, focus moved to first invalid field on server-side validation).

`workspace` is intentionally excluded — the settings page is a single form whose contract is already covered by the form-consistency requirements landing in the other capabilities, and tightening it does not require a new MUST/SHALL.

## Impact

- **Code / markup**: `web/templates/layouts/{base,app}.html`, `web/templates/partials/*.html`, every per-domain `index.html` under `web/templates/`, and the auth templates. Minor additions to `web/static/css/app.css` (numeric-alignment utility, reduced-motion wrapper, pill icon slot) and `web/static/js/app.js` (no new behaviors; possibly a dialog-focus-trap helper only if the `<dialog>`-based confirm pattern is expanded beyond the existing partial).
- **Backend / handlers**: no route or schema changes. Handler changes are limited to passing through field-level error keys so templates can render `aria-describedby` targets, and setting an explicit page `<title>` per route where missing.
- **Tests**: no new integration tests are required by the spec deltas themselves, but each UI-affecting task group ends with an explicit accessibility validation task (keyboard-only walkthrough, screen-reader spot check with VoiceOver or NVDA, automated axe / Lighthouse pass, contrast verification against 4.5:1).
- **Dependencies**: none added. No new JS libraries, no CSS frameworks.
- **Out of scope** (call-outs to prevent scope creep — each is a separate Stage 2 change already on the roadmap):
  - Extracting shared markup into reusable partials → `create-reusable-ui-partials-and-patterns`.
  - Introducing design tokens, a base component set, or product-specific widgets → `establish-custom-component-library-foundation`.
  - Internal showcase + usage docs → `create-component-library-showcase-and-usage-docs`.
  - Brand and visual-language refinement (logo, illustration, marketing tone) → `refine-timetrak-brand-and-product-visual-language`.
  - Any new feature, new page, or change to the domain model.
  - Dashboard redesign or addition of charts.
- **Assumptions**:
  - The existing `data-focus-after-swap` convention and `partials/confirm_dialog.html` are the right primitives — this change spreads their usage rather than replacing them.
  - Tokens in `web/static/css/tokens.css` already meet 4.5:1 for body text and primary-on-surface combinations; this change verifies that claim and adjusts only pill / muted-text variants that fail.
  - No backend schema migration is required.
- **Risks**:
  - Audit can surface a finding (e.g. a pill variant that structurally cannot hit 4.5:1 without a token change) that pushes into the token-overhaul territory reserved for `establish-custom-component-library-foundation`. Mitigation: if that happens, narrow the fix to the offending variant and note the broader token work as a follow-up in the design doc.
  - Spreading `data-focus-after-swap` too aggressively can cause unwanted focus jumps on passive peer-refresh swaps (e.g. dashboard summary refreshing on `entries-changed`). Mitigation: the design doc enumerates which swaps are user-intent swaps vs. passive refreshes and only the former get focus anchors.
- **Likely follow-up changes**: `create-reusable-ui-partials-and-patterns` consumes the conventions established here and extracts them into shared partials.
