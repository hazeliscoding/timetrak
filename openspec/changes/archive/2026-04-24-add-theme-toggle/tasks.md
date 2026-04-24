## 1. Partial

- [x] 1.1 Created `web/templates/partials/theme_switch.html` with block `theme_switch`. Root `<div role="radiogroup" aria-label="Theme" class="tt-theme-switch">`. Three child `<button role="radio">` segments for `light` / `dark` / `system`, each carrying `data-theme-set`, `aria-pressed`, `aria-checked`, an `aria-hidden` glyph, and a visible label.
- [x] 1.2 `InitialSelected` dict key supported. When set, the matching segment renders `aria-pressed="true"` / `aria-checked="true"`; otherwise all three render `"false"` and the client JS synchronizes post-parse.
- [x] 1.3 Documented the partial in `web/templates/partials/README.md` with context keys, accessibility obligations, and cross-links to the new `ui-partials` + `ui-component-identity` requirements.

## 2. CSS

- [x] 2.1 Added `.tt-theme-switch` rules in `web/static/css/app.css` (components layer): pill-radius outer border, flex row, per-segment 1px `var(--color-border)` dividers.
- [x] 2.2 Segment at-rest styling: neutral background, muted text color, `min-height: 32px`. Hover swaps to `var(--color-surface-alt)`. Focus-visible inherits the accepted accent focus ring.
- [x] 2.3 `[aria-pressed="true"]` selected state: `var(--color-accent-soft)` fill, `var(--color-accent)` text, `box-shadow: inset 0 0 0 2px var(--color-accent)` edge — the 2px state weight from the two-weight border contract.
- [x] 2.4 Glyph (1em, `aria-hidden`, `var(--space-1)` gap to label) + label (normal weight at rest, medium when selected) styled per the partial.
- [x] 2.5 `@media (max-width: 720px)` collapses the visible labels to `sr-only`, shrinking segments to icon-only while preserving the accessible name via per-segment `aria-label`.

## 3. FOUC prevention

- [x] 3.1 Added inline IIFE `<script>` at the top of `<head>` in `web/templates/layouts/base.html` before any `<link rel="stylesheet">`. Reads `localStorage.timetrak.theme` (defaults to `"system"`), applies to `<html data-theme=...>` synchronously. Wrapped in `try { ... } catch {}`. Inline comment cites the `ui-foundation` single-sanctioned-inline-script requirement.
- [ ] 3.2 Manual QA pending — user verification that no theme flash occurs on first paint for a `dark` preference.

## 4. Swap the header control

- [x] 4.1 Replaced the three `<button>` theme controls in `web/templates/layouts/app.html` with `{{template "theme_switch" (dict)}}`. The old `<nav aria-label="Theme">` wrapper was dropped because the partial's root now carries `role="radiogroup"` + `aria-label="Theme"` directly — no redundant landmark. **Surfaced fix:** `nil` is not a valid Go-template command; `(dict)` is the correct way to pass an empty context. The partial's `{{if .}}` guard makes an empty map falsy and skips the `InitialSelected` branch cleanly.
- [x] 4.2 Existing `app.js` click handler works unchanged — it binds on `[data-theme-set]` which the new segments emit identically.

## 5. Accent-rationing allow-list amendment

- [x] 5.1 Added `.tt-theme-switch-segment[aria-pressed="true"]` to the allow-list slice in `internal/showcase/identity_audit_test.go` with a short comment citing the amended `ui-component-identity` requirement (item 8).
- [x] 5.2 `go test ./internal/showcase/... -run TestAccentRationingAudit` passes with the new allow-list entry and the new `.tt-theme-switch` CSS rules.

## 6. Showcase entry + snippets

- [x] 6.1 Added a `theme_switch` `ComponentEntry` to `internal/showcase/catalogue.go` with three `Example`s (`light-selected`, `dark-selected`, `system-selected`). SpecRef → `openspec/specs/ui-partials/spec.md`; A11yNotes enumerates the radiogroup + dual aria contract + accent-rationing allow-list entry.
- [x] 6.2 Added three snippet fixtures: `internal/showcase/snippets/theme_switch.{light,dark,system}_selected.tmpl`, each invoking the real partial with a matching `InitialSelected` dict key.
- [x] 6.3 `go test ./internal/showcase/...` — partial-coverage + snippet-integrity tests green.

## 7. Verification and archival

- [x] 7.1 `make fmt && make vet && make test` — all green (including the two `reporting.TestReportsPartial*` tests that surfaced the `nil` template-argument bug mid-implementation).
- [ ] 7.2 Manual smoke pending — user verification: `make run`, confirm a single segmented control in the header (not three loose buttons), click each segment, reload to verify persistence, verify no FOUC. Open `/dev/showcase/components#entry-theme-switch` and verify all three selected-state renderings.
- [ ] 7.3 Commit via `tt-conventional-commit` skill — one commit covering partial + CSS + layout swap + head-script + allow-list amendment + showcase entry. No Claude attribution.
- [ ] 7.4 Archive via `/opsx:archive add-theme-toggle`.
