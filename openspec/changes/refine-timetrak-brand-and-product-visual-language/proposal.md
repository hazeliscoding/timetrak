## Why

TimeTrak has a codified visual foundation (`openspec/specs/ui-foundation/spec.md`, `web/static/css/tokens.css`), a reusable partial catalogue (`openspec/specs/ui-partials/spec.md`), and a dev-only showcase (`openspec/specs/ui-showcase/spec.md`). What it does NOT have is a product-identity surface. The wordmark is a bare `<strong>TimeTrak</strong>` in `web/templates/layouts/app.html:4`, the `<title>` is a static literal in `web/templates/layouts/base.html:4`, there is no favicon referenced in `<head>` (browser tab renders as a generic globe), and there is no `docs/` artifact that codifies voice, microcopy rules, or mark-usage rules. The UI style guide covers visual tokens; nothing covers brand.

Stage 3 is the right time to close that gap. The visual system is locked, the showcase exists as a review surface, and the next Stage 3 feature candidates (CSV export, invoices, team workspaces) all benefit from a settled brand identity before more surfaces are added. This change is intentionally narrow: it formalizes a bounded brand-identity surface and a voice/microcopy reference doc. It does NOT touch the token contract, introduce new semantic aliases, or rework any visual component.

## What Changes

- Add a single inline-SVG wordmark partial at `web/templates/partials/brandmark.html` that renders the TimeTrak wordmark using existing semantic aliases (`--color-text`, `--color-accent`) and no raw colour values. The partial accepts a `dict` with `Size` (one of `sm`, `md`) and `Decorative` (bool controlling whether the SVG carries `role="img"` + `<title>` or `aria-hidden="true"`).
- Replace the bare `<strong>TimeTrak</strong>` in `web/templates/layouts/app.html` with a call to the new `brandmark` partial, sized `md`, carrying an accessible name.
- Add a single-file favicon at `web/static/favicon.svg` (monochrome SVG, themed via `currentColor` against a neutral square) and wire `<link rel="icon" type="image/svg+xml" href="/static/favicon.svg">` into `web/templates/layouts/base.html`. No PNG fallback ships in this change (documented as a deferred follow-up).
- Codify the existing `<title>` convention in `base.html` so pages that define their own title compose as `<page> · TimeTrak` with a middle-dot (U+00B7) separator; pages that do not define one render just `TimeTrak`. This is a microcopy convention, not a new template mechanism — the existing `{{block "title" .}}` is reused, and every live page already follows the middle-dot pattern. See design.md Decision 5.
- Add `docs/timetrak_brand_guidelines.md`: a prose companion to `docs/timetrak_ui_style_guide.md` covering (1) wordmark usage rules (clear-space, min size, permitted/prohibited treatments, the explicit "no gradients, no glow, no lockup with taglines" list), (2) voice principles (calm, specific, billing-aware; domain nouns over generic productivity verbs), (3) microcopy patterns for empty states, confirmations, validation errors, and loading labels with 10–15 concrete before/after examples sourced from the existing templates. Cross-linked from the style guide and `CLAUDE.md` UI Direction section.
- Add a new showcase section at `/dev/showcase/brand` that renders the `brandmark` partial in each documented size and a rendered favicon preview, matching the existing showcase contract (live partial rendering, copy-ready snippet, spec ref).
- Add a browser contract test asserting the wordmark renders with an accessible name and the favicon link resolves with `Content-Type: image/svg+xml`.
- **Out of scope (explicit):**
  - No new semantic aliases. Brand colours are the existing `--color-accent` and `--color-text`.
  - No change to the accent hue, type scale, spacing scale, motion tokens, radius, or elevation. Visual language stays put.
  - No full logo system (icon-only mark, app-icon variants, social-share card, email-signature marks, OG image). Tracked as future follow-ups if needed.
  - No PNG / ICO favicon fallback. Modern browsers support SVG favicons; legacy fallback is a separate change if ever justified.
  - No marketing site, landing page, or signup-screen redesign.
  - No copy audit of every existing template. The guidelines doc ships with illustrative before/after pairs; a full sweep is a separate change.
  - No rename of the product, no tagline, no brand-colour re-tune.
  - No new runtime dependency. SVG is inline; no icon library, no build step.

## Capabilities

### New Capabilities

- _None._ Brand-identity surface is narrow enough to live inside existing capabilities. A new `brand` capability would be overfit for one partial, one favicon, one showcase section, and one doc.

### Modified Capabilities

- `ui-partials` — adds the `brandmark` partial to the enumerated catalogue; the requirement that lists canonical partials must extend by one entry. This is a requirement delta, not prose.
- `ui-showcase` — adds the `brand` section to the dev-only showcase's coverage obligation; the partial-coverage enforcement requirement must recognise `brandmark` as a documented entry (will happen automatically under the existing coverage test once the partial ships, but the spec text should call out the brand section as a documented sub-surface).
- `ui-foundation` — NOT modified. No new semantic alias, no new scale token, no new primitive, no new `tt-<component>`. The `brandmark` partial consumes existing aliases only.

## Impact

- **New files:** `web/templates/partials/brandmark.html`, `web/static/favicon.svg`, `docs/timetrak_brand_guidelines.md`, `web/templates/showcase/brand.html` (or an addition to `web/templates/showcase/components.html` — decided in design), snippet fixture under `internal/showcase/snippets/`, `internal/e2e/browser/brand_test.go`.
- **Modified files:** `web/templates/layouts/app.html` (wordmark swap), `web/templates/layouts/base.html` (favicon link + title convention comment), `internal/showcase/catalogue.go` (new `ComponentEntry` for `brandmark`), `web/templates/partials/README.md` (catalogue entry), `docs/timetrak_ui_style_guide.md` (one-line cross-link to the new brand guidelines), `CLAUDE.md` UI Direction section (one-line cross-link to the new brand guidelines).
- **Code / runtime:** no Go service logic touched; no migrations; no new dependency.
- **Risk profile:** low. The only user-visible production change is a wordmark swap and a favicon appearing. Visual tokens untouched.
- **Accessibility:** wordmark must carry an accessible name (`role="img"` + `<title>`) when used non-decoratively, honour `:focus-visible` through its anchor wrapper, meet ≥4.5:1 contrast of mark against header surface in both themes.
- **Follow-ups (not part of this change):** PNG/ICO favicon fallback, OG/social-share image, email-signature mark, marketing-surface brand kit, full copy audit.
