## Why

The rates management page is the only core domain UI still using classic POST/redirect round-trips. Clients, projects, tracking, and reporting all lean on HTMX partials for inline edit, focus preservation, and peer-refresh events, so `/rates` now feels inconsistent, discards scroll position on every mutation, and cannot surface validation errors next to the offending field without a full reload. Stage 2's component-library foundation work requires a reusable `rate_row` partial and a standard `rates-changed` event before downstream features (reporting filters, invoicing, entry detail) can rely on rate edits without a full page reload.

## What Changes

- Add server-rendered HTMX partials for the rate rules management UI:
  - `rate_row` partial rendering a single rule row in display or edit mode
  - `rate_form` partial for the "New rate rule" card (create + inline error)
  - `rates_table` partial wrapping tbody for full re-render after create/delete
- Route additions (all workspace-scoped, CSRF-protected):
  - `GET /rates/{id}/edit` → returns `rate_row` in edit mode
  - `GET /rates/{id}/row` → returns `rate_row` in display mode (cancel)
  - Convert `POST /rates`, `POST /rates/{id}`, `POST /rates/{id}/delete` to return partials when `HX-Request: true`, preserving the current full-page flow for no-JS fallback
- Emit `HX-Trigger: rates-changed` on every successful create / update / delete so reporting and future invoicing views can refresh without polling
- Preserve focus after swaps via `data-focus-after-swap` on the edit row's first input and on the "New rate rule" form's scope select after successful create
- Replace the inline `onsubmit="return confirm(...)"` delete with `hx-confirm` (consistent with clients/projects)
- Replace the inline `onchange` scope toggle with an HTMX-driven re-render of the scope-dependent fields (or a minimal, accessible progressive-enhancement fallback) so no-JS continues to work
- Keep all existing validation semantics: overlap 409, referenced-rule 409, negative/invalid/currency 422; errors rendered via `aria-describedby` in the partial

Out of scope:
- Changing rate-resolution precedence, storage shape, or money math
- Bulk edit or CSV import of rules
- New columns (effective history timeline, audit trail)

## Capabilities

### New Capabilities
<!-- none — this change hardens an existing capability -->

### Modified Capabilities
- `rates`: add UI requirements for HTMX-driven inline edit, `rates-changed` peer-refresh event, focus-after-swap behavior, and no-JS fallback — the underlying rate-rule storage, overlap, precedence, and resolution requirements are unchanged

## Impact

- Code: `internal/rates/handler.go` (new HX-aware branches + row/edit routes), `web/templates/rates/index.html` (extract partials), new `web/templates/partials/rate_row.html`, `web/templates/partials/rate_form.html`, `web/templates/partials/rates_table.html`
- Templates: reuse existing `formatMinor`, `formatDate` helpers; no new template funcs
- Routes: two net-new GET routes (`/rates/{id}/edit`, `/rates/{id}/row`); existing POST routes gain an `HX-Request` branch
- Events: new `rates-changed` HX-Trigger — downstream reports/dashboard are free to subscribe in a later change but this proposal does not wire new listeners
- Accessibility: continues to target WCAG 2.2 AA; error messaging keeps `aria-describedby`; focus behavior standardized on `data-focus-after-swap`
- Risk: the edit-only-end-date historical-safety rule must still be enforced server-side — the partial form MUST continue to send hidden fields for the immutable columns. Covered by existing `ErrRuleReferenced` path.
- Follow-up (not in this change): subscribe reporting summary and dashboard widgets to `rates-changed`; extract a shared `confirm_dialog` usage once destructive-action UX is standardized across domains.
