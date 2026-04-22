## Context

TimeTrak's visual system is locked. `web/static/css/tokens.css` codifies a two-layer token model, `openspec/specs/ui-foundation/spec.md` enforces the authoring contract, and the Stage 2 polish/foundation/showcase changes have landed. What remains is the product-identity layer — the bits that tell a user "this is TimeTrak" before they read a single row of data.

Today's identity surface:

- **Wordmark:** `web/templates/layouts/app.html:4` renders `<strong>TimeTrak</strong>`. It has no custom typography, no accent treatment, no accessible mark semantics beyond the default text node.
- **Browser-tab affordance:** none. `web/templates/layouts/base.html:7-8` loads the stylesheets but does not reference a favicon. Browsers render the generic default icon.
- **`<title>`:** `{{block "title" .}}TimeTrak{{end}}` at `base.html:4`. Every non-overriding page tab reads "TimeTrak" identically.
- **Voice / microcopy:** `docs/timetrak_ui_style_guide.md:408-426` lists good vs. bad microcopy patterns at the paragraph level, but there is no longer-form guidance doc. `CLAUDE.md` UI Direction section says "use domain-specific copy" but does not enumerate the rules.
- **Brand marks:** `web/static/` contains `css/`, `js/`, `vendor/`. No `favicon.*`, no `logo.*`, no `brand/` directory. No image assets of any kind.

The gap is narrow and well-defined: a wordmark mark, a favicon, a title convention, and a voice/microcopy reference document. Everything else — accent hue, type scale, motion — is already covered by the foundation spec and is out of scope.

## Goals / Non-Goals

**Goals:**

- Give TimeTrak a recognizable product-identity surface: wordmark, favicon, tab title, and voice rules — in that order of user visibility.
- Keep the brand consumable by the existing token contract. The wordmark and favicon reference only `--color-text`, `--color-accent`, and `currentColor`. No raw hex values.
- Make voice and microcopy reviewable at PR time via a short prose doc alongside `docs/timetrak_ui_style_guide.md`.
- Preserve WCAG 2.2 AA across wordmark, favicon, and any new template fragment.
- Integrate into the existing showcase so the brand surface is reviewable in the same place every other partial is.

**Non-Goals:**

- No new semantic aliases. The token spec is not amended by this change.
- No change to accent hue, type scale, spacing, radius, motion, or elevation.
- No full logo system: no icon-only mark, no app-icon tile, no OG/social-share card, no email-signature artefact.
- No PNG or ICO favicon fallback. SVG-only; legacy browser fallback is a deferred change.
- No copy audit. The guidelines doc ships with illustrative before/after examples, not a sweep of every existing template.
- No tagline, no product rename, no marketing surface.
- No new runtime dependency. No icon library. No SVG optimiser in the build. No client-side renderer.
- No animated wordmark / transitions. The mark is static.

## Decisions

### Decision 1 — Scope boundary: "brand" vs "visual language"

- **Decision:** "brand" in this change means four things: (1) the wordmark as an inline-SVG partial, (2) a single SVG favicon, (3) the `<title>` composition convention (`<page> · TimeTrak`, see Decision 5), and (4) a prose voice/microcopy guidelines doc. "Visual language" — accent hue, type scale, motion tokens, elevation — is OUT of scope and is already covered by `ui-foundation`.
- **Rationale:** the one-change-per-unit rule rejects umbrella proposals. Brand marks are a cohesive, user-visible, low-risk unit. Re-opening the token contract for a hue tweak is a separately-justifiable change with a separate risk profile and a separate spec amendment.
- **Alternative considered:** bundle in an accent-hue refinement. Rejected — would require amending `ui-foundation`'s Two-Layer Token Taxonomy, touch every surface through `--color-accent`, and balloon the blast radius. Split into a later change if it ever becomes necessary.

### Decision 2 — No new semantic aliases; brand consumes existing tokens only

- **Decision:** the wordmark SVG and favicon reference only `currentColor`, `var(--color-text)`, and `var(--color-accent)`. No new `--color-brand-*` aliases. No new primitive ramps.
- **Rationale:** `openspec/specs/ui-foundation/spec.md` requires a spec amendment to add a semantic alias. The existing `--color-accent` is the product's identity colour by construction; a parallel `--color-brand` would duplicate it with no contrast-role story. Components must consume aliases, and the wordmark is a component.
- **Alternative considered:** introduce `--color-brand-mark` as its own alias for "the wordmark renders in this colour." Rejected — it would be indistinguishable from `--color-accent` and would violate the "new alias requires spec amendment with a distinct contrast role" rule in the foundation spec.

### Decision 3 — This change DOES require requirement deltas in two specs

- **Decision:** `ui-partials` gets a requirement delta adding `brandmark` to the enumerated partial catalogue. `ui-showcase` gets a requirement delta acknowledging the brand sub-section of the showcase. `ui-foundation` is NOT amended.
- **Rationale:** `ui-partials` explicitly enumerates the canonical partial set; adding a partial without updating that list would let the coverage test lag the spec. `ui-showcase` enforces partial-coverage; the brand partial must be under that coverage from the moment it ships. `ui-foundation` is untouched because no token, scale, or variant changes.
- **Alternative considered:** ship purely as prose/doc with no requirement delta (the pattern used by `2026-04-22-transition-to-stage-3`). Rejected — this change introduces a production-rendered partial and a new shipped asset. That is behavior, not prose.

### Decision 4 — Asset strategy: inline SVG for the wordmark, file SVG for the favicon

- **Decision:** the wordmark ships as an inline `<svg>` rendered by `web/templates/partials/brandmark.html`. The favicon ships as `web/static/favicon.svg`, referenced from `<link rel="icon" type="image/svg+xml" href="/static/favicon.svg">` in `base.html`.
- **Rationale:**
  - **Inline wordmark** — lets the mark adopt `currentColor` and `var(--*)` tokens natively, follows the app's theme toggle without a second asset, and needs no HTTP request. Footprint is small (single-path SVG in a layout that already loads on every page).
  - **File favicon** — browsers fetch favicons via `<link rel="icon">`, not inline. SVG favicons are supported by every modern browser TimeTrak targets. Shipping as a static file under `web/static/` stays inside the existing static-asset pipeline; no new mux route needed.
- **Alternative considered:** icon font or `@font-face` wordmark (rejected — a new runtime dependency and a new asset format for one glyph). Inline favicon via data URI (rejected — no caching, no external cacheability, and a longer `<head>`). PNG favicon ships alongside SVG (rejected — adds two artefacts for marginal legacy-browser coverage; deferred to a follow-up change with an explicit justification).

### Decision 5 — `<title>` convention: `<page> · TimeTrak` with middle-dot (amended 2026-04-22)

- **Decision:** pages override `{{block "title" .}}` with their specific page name followed by ` · TimeTrak` (U+00B7 MIDDLE DOT, surrounded by single spaces). The base template composes `{{block "title" .}}TimeTrak{{end}}` so that an override like `{{define "title"}}Clients · TimeTrak{{end}}` produces the full tab string. Pages that do not override the block render just `TimeTrak`. The guidelines doc codifies the pattern so future page authors converge on it.
- **Rationale:** middle-dot is the separator every existing page title already uses (`Dashboard · TimeTrak`, `Clients · TimeTrak`, `Sign in · TimeTrak`, etc.), so codifying it matches reality and requires no follow-up copy pass. Middle-dot reads as a brand separator — not a compound like hyphen, not noise like pipe — and keeps the specific page name leftmost so the page context is scanable and the product name sits closest to the favicon.
- **Amendment history:** the original Decision 5 prescribed em-dash without evaluating the middle-dot that every live page already used. Amended 2026-04-22 after implementation-phase audit confirmed zero pages used em-dash and that converting them was explicitly out-of-scope per the proposal's "no copy audit" clause. Middle-dot was the obvious incumbent and satisfies every criterion em-dash did.
- **Alternatives considered:** em-dash `—` (rejected on amendment — no live page uses it, would create immediate inconsistency with ten existing pages, and its "brand separator" advantage applies equally to middle-dot). Hyphen `-` (rejected — reads as a compound word). Pipe `|` (rejected — visual clutter). Branded-first `TimeTrak · <page>` (rejected — buries page context behind the brand, harms browser-tab scanability).

### Decision 6 — Voice / microcopy doc lives at `docs/timetrak_brand_guidelines.md`

- **Decision:** a new prose doc in `docs/`, cross-linked from `docs/timetrak_ui_style_guide.md` and `CLAUDE.md`. It is a companion, not a replacement. The existing "Microcopy" section (style guide lines 408-426) stays; the brand guidelines doc expands it.
- **Rationale:** `docs/` is the documented home for long-form narrative reference per `CLAUDE.md`. Splitting voice into its own file keeps the style guide focused on visual tokens and keeps the guidelines doc focused on language. The two docs have different audiences (visual vs. editorial) even when contributors overlap.
- **Alternative considered:** append a "Voice" section to the existing style guide (rejected — the style guide is already 569 lines and mixing visual and editorial rules dilutes both). Ship as a section of `web/static/css/README.md` (rejected — that README is scoped to the CSS authoring contract).

### Decision 7 — Accessibility model for the wordmark

- **Decision:** `brandmark.html` accepts a `Decorative` boolean in its `dict`. When `Decorative` is `false` (the default used in the app header), the rendered SVG carries `role="img"` and a `<title>TimeTrak</title>` child, so screen readers announce "TimeTrak" as a graphic. When `Decorative` is `true` (reserved for cases where the mark sits next to a text "TimeTrak" already), the SVG carries `aria-hidden="true"` and no `<title>`. The wrapping `<a>` anchor in `app.html` carries an accessible name via the mark.
- **Rationale:** doubling up "TimeTrak" to both a screen reader and a sighted reader is noise; the `Decorative` flag exists to handle that. The 3:1 non-text contrast rule in `ui-foundation`'s Focus-Indicator Contract applies to the mark's stroke/fill against header surfaces — verified in a task.
- **Alternative considered:** always `role="img"` + `<title>` (rejected — forces a repetition in any surface that already names the product in adjacent text). Always `aria-hidden` (rejected — the app-header wordmark is the only product-identity signal an AT user gets).

### Decision 8 — Favicon contrast & theming strategy

- **Decision:** the favicon ships as a monochrome SVG using `fill="currentColor"` with an inline `<style>` that sets `:root { color: <value matching --color-text light ramp>; }` and a `@media (prefers-color-scheme: dark) { :root { color: <value matching --color-text dark ramp> } }` pair. Because the SVG is a separate resource fetched by the browser (not part of the app's CSS cascade), it cannot reference the app's CSS custom properties. The inline media query is the supported hook.
- **Rationale:** this is the standard cross-browser pattern for themed SVG favicons. It avoids a second asset and still flips correctly under light/dark system preferences.
- **Trade-off:** the favicon does NOT follow the app's in-tab `data-theme` toggle (that only affects the app's own DOM). It follows the OS / browser-level colour-scheme preference only. Accepted — browser-tab favicons are a system-level surface.

### Decision 9 — How this change stays WCAG 2.2 AA

- **Wordmark:** fill/stroke reference `currentColor` or `var(--color-accent)`, both of which meet ≥4.5:1 against `--color-surface` in both themes by construction (already verified by `ui-foundation`). Any decorative element must meet 3:1 non-text contrast. A task verifies both with axe-core.
- **Favicon:** monochrome against a transparent or neutral square. The browser's tab chrome sets the background; the glyph must stay recognizable against both a light and a dark tab. Verified visually and called out in tasks.
- **Title convention:** purely textual, no contrast concern.
- **Voice guidelines doc:** editorial, not visual; accessibility concern is limited to the doc rendering correctly in browsers / editors and having an unambiguous heading structure.
- **Focus:** the anchor wrapping the wordmark in the app header inherits the global `:focus-visible` outline from `app.css`; no override.
- **Reduced motion:** the mark is static. No motion concern.

### Rendering flow (no server-side change)

```mermaid
flowchart LR
    A[Browser request: any app page] --> B[base.html loads tokens + app css + favicon link]
    B --> C[app.html shell renders]
    C --> D[brandmark.html partial executes]
    D --> E[inline SVG emitted using currentColor + var(--color-accent)]
    E --> F[theme toggle in app.js flips data-theme attribute]
    F --> G[SVG re-paints via CSS custom property cascade]
    B --> H[Browser fetches /static/favicon.svg]
    H --> I[SVG prefers-color-scheme media query selects glyph fill]
```

## Risks / Trade-offs

- **Risk: visual churn** — contributors may read "brand refinement" as license to tweak visual tokens. **Mitigation:** the proposal's Out-of-scope list is explicit and the design doc's Decision 1 draws the line. PR review rejects any token change under this change's name.
- **Risk: scope creep into a full logo system** — once a wordmark lands, pressure to ship an app icon, social-share card, and email-signature mark grows. **Mitigation:** the proposal enumerates these as explicit out-of-scope follow-ups. Each gets its own change proposal if ever prioritised.
- **Risk: accessibility regression** — a mark with insufficient contrast or a missing accessible name would degrade the app's current AA posture. **Mitigation:** tasks include axe-core smoke coverage on the showcase `brand` page, a contrast check for the wordmark against `--color-surface` in both themes, and an assertion that the wordmark carries an accessible name when non-decorative.
- **Risk: generic-AI-SaaS visual language sneaking in** — "brand refinement" is the exact prompt that invites floating blob decorations, gradient lockups, oversized hero headlines, and vague copy. **Mitigation:** Decision 1 locks the surface to four narrow artefacts, Decision 2 forbids new visual tokens, and the guidelines doc explicitly enumerates anti-patterns copied from `CLAUDE.md` UI Direction (no gradients on the mark, no glow, no blob art, no decorative hero copy, no marketing verbs).
- **Risk: favicon theming is inconsistent across browsers** — not every browser honours `prefers-color-scheme` inside an SVG favicon. **Mitigation:** the fallback path (single mid-tone glyph visible on both light and dark tabs) is the default; the media query is a progressive enhancement. Documented in Decision 8 and in the guidelines doc.
- **Risk: title convention drifts because future pages forget to define the `title` block** — pages without an override render as just "TimeTrak" in the tab. **Mitigation:** the guidelines doc codifies the `<page> · TimeTrak` convention and enumerates the ten existing pages that already conform as the live reference. Because Decision 5 was amended to match incumbent practice (middle-dot), there is no pre-existing drift to sweep — drift can only appear in pages added after this change, and the guidelines doc gives authors a one-line example to copy.
- **Trade-off: no PNG/ICO favicon fallback** — very old browsers see no icon. Accepted; TimeTrak's browser-support bar is modern evergreen browsers. If a contract ever requires broader support, a follow-up change adds PNG + ICO side-by-side.
- **Trade-off: no animated or interactive wordmark** — the mark is static SVG. Accepted; motion would conflict with the "calm, tool-like" brand posture codified in `docs/timetrak_ui_style_guide.md`.
- **Open question:** should the header wordmark link to `/dashboard` (the signed-in landing page) or be a non-link display element? Deferred to implementation — both satisfy the spec. The implementer picks the shape that matches existing in-app navigation expectations and documents the choice in the partials README.
