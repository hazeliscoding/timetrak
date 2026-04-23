## 1. Audit and scope confirmation

- [x] 1.1 Confirm no existing brand assets ship today: verify `web/static/` contains no `favicon.*`, no `logo.*`, and no `brand/` directory; verify `web/templates/layouts/base.html` does not reference a favicon; verify `web/templates/layouts/app.html` renders the wordmark as `<strong>TimeTrak</strong>`.
- [x] 1.2 Confirm no semantic alias or scale token needs to change by listing every `var(--color-*)` the new wordmark and favicon will consume — expected set is `currentColor`, `var(--color-text)`, `var(--color-accent)`. If any other token is required, STOP and amend the proposal to include a `ui-foundation` spec delta.
- [x] 1.3 Enumerate every existing page template that overrides `{{block "title" .}}` today, and list pages that do NOT, so the guidelines doc can document the "current status / follow-up" table without a full copy audit under this change.

## 2. Brandmark partial

- [x] 2.1 Create `web/templates/partials/brandmark.html` defining a `{{define "brandmark"}}` block that accepts a `dict` with keys `Size` (string, one of `sm`, `md`; defaults to `md` when empty) and `Decorative` (bool; defaults to `false` when missing).
- [x] 2.2 When `Decorative` is `false`, render the SVG with `role="img"` and a child `<title>TimeTrak</title>` so screen readers announce the mark.
- [x] 2.3 When `Decorative` is `true`, render the SVG with `aria-hidden="true"` and omit the `<title>`.
- [x] 2.4 Size the SVG via CSS custom properties on the parent or via inline `width`/`height` that references pixel values sourced from the scale tokens' spacing ladder (e.g. `md` = `var(--space-5)` tall, `sm` = `var(--space-4)` tall); do NOT use raw pixel literals inside the component's `style`.
- [x] 2.5 Use only `currentColor`, `var(--color-text)`, and `var(--color-accent)` for SVG fill/stroke. No raw hex values, no new aliases.
- [x] 2.6 Add a one-paragraph header comment in the partial documenting the `dict` contract and the decorative/non-decorative rule.

## 3. Favicon asset

- [x] 3.1 Create `web/static/favicon.svg` as a monochrome square SVG using `fill="currentColor"`.
- [x] 3.2 Embed an inline `<style>` block in the favicon SVG that sets a mid-contrast colour by default and flips under `@media (prefers-color-scheme: dark)` to a lighter tone that is legible on dark browser-tab chrome.
- [x] 3.3 Wire a `<link rel="icon" type="image/svg+xml" href="/static/favicon.svg">` into the `<head>` of `web/templates/layouts/base.html`, placed between the existing stylesheet links and the script tags.
- [x] 3.4 Manually verify the favicon renders in Firefox, Chromium, and Safari under both light and dark OS colour-scheme preferences. Record results in the task checkoff note.

## 4. Layout integration

- [x] 4.1 Replace `<strong>TimeTrak</strong>` in `web/templates/layouts/app.html` with a call to the `brandmark` partial: `{{template "brandmark" (dict "Size" "md" "Decorative" false)}}`, wrapped in an anchor (target resolved per the open question in Decision 9 — document the chosen target in the partials README entry).
- [x] 4.2 Ensure the anchor inherits the global `:focus-visible` outline; do NOT introduce a component-scoped focus override.
- [x] 4.3 Leave `{{block "title" .}}TimeTrak{{end}}` in `base.html` unchanged structurally; add an HTML comment above it documenting the "`<page> · TimeTrak`" composition convention (middle-dot `·`, U+00B7, single spaces either side) for future authors. See design.md Decision 5 (amended 2026-04-22).

## 5. Brand guidelines document

- [x] 5.1 Create `docs/timetrak_brand_guidelines.md` with four top-level sections: `Wordmark usage`, `Favicon and browser-tab identity`, `Title convention`, `Voice and microcopy`.
- [x] 5.2 Under `Wordmark usage`, document clear-space (at least `--space-3` on all sides), minimum rendered size (the `sm` variant), permitted contexts (app header, dev showcase, guidelines doc), and an explicit prohibited-treatment list copied from `CLAUDE.md` UI Direction: no gradients, no glow/shadow lockup, no accent-hue recolour outside the documented tokens, no taglines, no animated variants.
- [x] 5.3 Under `Favicon`, document the monochrome-glyph rule, the `prefers-color-scheme` behaviour from Decision 8, and the explicit "no PNG/ICO fallback in this change" note with a link to the deferred follow-up.
- [x] 5.4 Under `Title convention`, document the `<page> · TimeTrak` pattern with middle-dot (U+00B7, single spaces either side) per design.md Decision 5 (amended 2026-04-22). Every existing user-facing page already conforms; the ten conforming pages (`dashboard.html`, `time/index.html`, `clients/index.html`, `projects/index.html`, `rates/index.html`, `reports/index.html`, `workspace/settings.html`, `auth/login.html`, `auth/signup.html`, `errors/not_found.html`) are documented as the convention's live reference. No "follow-up" table is needed.
- [x] 5.5 Under `Voice and microcopy`, document the three voice principles (calm, specific, billing-aware) and ship 10–15 before/after examples sourced from existing TimeTrak templates (e.g. confirmations, validation messages, empty states, loading labels). Every example pair MUST cite the file path it was drawn from.
- [x] 5.6 Cross-link `docs/timetrak_brand_guidelines.md` from `docs/timetrak_ui_style_guide.md` (one line under Status) and from the `## UI Direction` section of `CLAUDE.md` (one line).

## 6. Showcase entry

- [x] 6.1 Decide whether to render the brandmark inside the existing `web/templates/showcase/components.html` as a new anchored section, or to create a separate `web/templates/showcase/brand.html` page accessible from the showcase index. Document the choice in this task's checkoff note.
- [x] 6.2 Add a `ComponentEntry` for `brandmark` to `internal/showcase/catalogue.go` with `PartialName: "brandmark"`, a `SourcePath` pointing at `web/templates/partials/brandmark.html`, a `SpecRef` pointing at the `ui-partials` catalogue requirement, documented `DictKeys` for `Size` and `Decorative`, and at least two `Examples`: one `Size: "md", Decorative: false` and one `Size: "sm", Decorative: true`.
- [x] 6.3 Add a snippet fixture per example under `internal/showcase/snippets/` (matching the existing convention from `2026-04-21-create-component-library-showcase-and-usage-docs`).
- [x] 6.4 Render the favicon preview somewhere on the chosen showcase page — an `<img src="/static/favicon.svg">` alongside a note that it follows OS colour-scheme preference, not the app's `data-theme` toggle.
- [x] 6.5 Add an entry to `web/templates/partials/README.md` for `brandmark` documenting its `dict` contract, its accessibility model, and a pointer to the new guidelines doc.

## 7. Tests

- [x] 7.1 Extend the existing partial-coverage test in `internal/showcase/coverage_test.go` (or rely on its enumeration, if it already scans `web/templates/partials/`) to include the new `brandmark.html`. If the test enumerates automatically, no change is needed beyond verifying it still passes.
- [x] 7.2 Add a snippet-integrity assertion for the new `brandmark` examples (`PartialName` resolves against the live template loader), matching the existing snippet-integrity test pattern.
- [x] 7.3 Add `internal/e2e/browser/brand_test.go` gated by `//go:build browser` that asserts:
  - the app header (on an authenticated page, e.g. `/dashboard`) contains an element matching the brandmark SVG with an accessible name "TimeTrak";
  - `GET /static/favicon.svg` returns 200 with `Content-Type: image/svg+xml`;
  - `<head>` on every public page contains the `<link rel="icon" type="image/svg+xml" href="/static/favicon.svg">`;
  - axe-core on the showcase brand surface passes `wcag2a`, `wcag2aa`, `wcag22aa` with zero `serious` / `critical` violations.

## 8. Accessibility and token-contract validation

- [x] 8.1 Verify the wordmark SVG's fill and stroke reference only the documented tokens (`currentColor`, `var(--color-text)`, `var(--color-accent)`) and contain no raw hex, rgb, or named colour values. Grep the partial for raw colour patterns as a check.
- [x] 8.2 Verify the wordmark meets ≥4.5:1 contrast against `--color-surface` (the header surface) in both light and dark themes. Record measured values in the task checkoff.
- [x] 8.3 Verify the wordmark's focusable anchor shows the global `:focus-visible` ring; keyboard-tab from the skip link through the header and confirm focus is visible.
- [x] 8.4 Verify screen-reader output: with `Decorative: false`, the wordmark is announced as "TimeTrak" (graphic); with `Decorative: true`, it is skipped silently.
- [x] 8.5 Verify the favicon is legible against both a light and a dark browser-tab chrome. Record screenshots in the checkoff.
- [x] 8.6 Verify no new semantic alias and no new scale token were introduced — diff `web/static/css/tokens.css` and confirm zero changes under this branch.
- [x] 8.7 Run the existing `:focus-visible` contract test in `internal/e2e/browser/` (or equivalent) and confirm no regression.
- [x] 8.8 Verify `prefers-reduced-motion: reduce` leaves the wordmark and favicon untouched — both are static by construction; this task is a smoke check, not a fix.

## 9. Spec deltas

- [x] 9.1 Author the `specs/ui-partials/spec.md` delta that extends the partial-catalogue requirement to enumerate `brandmark` alongside the existing entries.
- [x] 9.2 Author the `specs/ui-showcase/spec.md` delta that acknowledges the brand sub-section of the showcase and confirms `brandmark` is under the partial-coverage enforcement requirement.
- [x] 9.3 Confirm `specs/ui-foundation/` is NOT touched by this change.

## 10. Build gates

- [x] 10.1 Run `make fmt`.
- [x] 10.2 Run `make vet`.
- [x] 10.3 Run `make test`.
- [x] 10.4 Run `make test-browser` (after `make browser-install` if needed). Attach failure artefacts per the existing browser-contract convention if anything regresses.
- [x] 10.5 Run `openspec validate refine-timetrak-brand-and-product-visual-language` and confirm the change is archive-ready.

## 11. Commit and archive

- [x] 11.1 Mark every task in this file complete.
- [x] 11.2 Committed retroactively as `c7f7421 feat(ui): refine brand and product visual language`. No Claude attribution.
- [x] 11.3 Archive in progress via `/opsx:archive refine-timetrak-brand-and-product-visual-language --yes` (2026-04-23).
