## Why

TimeTrak now has a codified CSS foundation (`web/static/css/README.md` + `openspec/specs/ui-foundation/spec.md`) and a reusable partials catalogue (`web/templates/partials/README.md` + `openspec/specs/ui-partials/spec.md`), but the only way a contributor can see what exists today is to read prose READMEs and grep the templates directory. There is no single place to view every reusable partial rendered in every documented state, no single place to inspect every semantic token with its computed value, and no copy-ready snippet surface to shortcut adoption. That gap invites drift: new pages re-implement markup the foundation already covers, and token usage diverges because computed values are invisible until the component ships.

This change is the third of four follow-ups queued from the polish-mvp work — the MVP is still the priority and the showcase is the developer-facing reference that keeps the component library honest as the next Stage 2 UI work lands.

## What Changes

- Add a new **developer-only showcase surface** mounted at `/dev/showcase`, registered ONLY when `APP_ENV=dev`. The route MUST return 404 in any non-dev environment; it is belt-and-suspenders-guarded both at registration time (`cmd/web/main.go`) and at handler entry.
- Add a new `internal/showcase/` package that owns the route's handlers, the catalogue metadata (`[]ComponentEntry`, `[]TokenEntry`), and the fixture snippets. The package depends on the existing template loader — it does NOT ship a second renderer.
- Add a **token catalogue page** that enumerates every primitive ramp, every semantic alias, and every scale token (spacing, radius, typography, motion, elevation, z-index, breakpoint) currently declared in `tokens.css`. Each entry renders the token's computed value, a visible sample (swatch, sizing bar, sample text, etc.), and its documented semantic role. The existing `data-theme` toggle switches light/dark in place.
- Add a **component catalogue page** (plus anchor-linked sub-sections) that documents every partial enumerated in `web/templates/partials/README.md`. Each entry renders the **real partial** via `template.ExecuteTemplate`, shows a copy-ready `<pre><code>` snippet drawn from a colocated fixture file, documents the `dict` keys it consumes, and shows documented state / variant permutations where they exist in code (e.g. `flash` severities, `btn` variants, form-field invalid state, `empty_state` with/without action). No synthetic states.
- Add a short **contribution guide** appendix rendered on the showcase index describing how to add a new token or component entry.
- Cross-link both existing READMEs → showcase URL, and showcase → READMEs + the relevant requirement in `ui-foundation` / `ui-partials` specs.
- Add browser contract coverage under `internal/e2e/browser/` (new file) asserting the showcase index loads, every anchor resolves, axe smoke passes on `wcag2a` / `wcag2aa` / `wcag22aa` with zero `serious` / `critical` violations, and a coverage test asserts every partial in `web/templates/partials/` has a showcase entry.
- **Out of scope (explicit):** production exposure; Storybook / MDX / Figma sync; in-page live code editing; new components or new tokens (gaps surface as follow-up changes); brand refresh (that is the next follow-up); i18n; component versioning / changelog; automated screenshot generation.

## Capabilities

### New Capabilities

- `ui-showcase`: A dev-only, server-rendered showcase surface that catalogues every reusable partial and every design token living in the repo, with live example renderings and copy-ready template snippets, and enforces that the documented surface stays in sync with the actual partials and tokens.

### Modified Capabilities

<!-- No existing spec requirements change. The showcase cross-links ui-foundation and ui-partials requirements but does not amend them. -->

## Impact

- **New code**: `internal/showcase/` (handlers, catalogue metadata, fixture snippets loader), `web/templates/showcase/` (index + token catalogue + component catalogue templates + colocated snippet fixtures), a new dev-only route group in `cmd/web/main.go`.
- **New tests**: `internal/e2e/browser/showcase_test.go` (reachability + axe smoke), a unit-level coverage test inside `internal/showcase/` that enumerates partials from `web/templates/partials/` and asserts each non-grandfathered one appears exactly once in the component catalogue.
- **Docs**: `web/static/css/README.md` and `web/templates/partials/README.md` gain a one-line pointer to `/dev/showcase`. No other documentation moves or is rewritten.
- **No new runtime dependencies**: no Storybook, no Node build step, no markdown renderer, no syntax-highlighting library, no SPA framework, no client-side state library.
- **No production surface change**: the route is unreachable in production; the only artifact a production build gains is unreferenced Go + template source compiled into the binary. No routes are registered, no links to `/dev/showcase` appear in any user-facing template.
- **No domain model change**: the showcase exists outside `auth` / `workspace` / `clients` / `projects` / `tracking` / `rates` / `reporting`. It requires a session (reuses `authz.RequireAuth`) but does NOT require a workspace.
- **Risks**: snippet / render drift between copy-ready snippet and real partial (mitigated: both go through the same template loader so a mismatch fails at render time in dev, plus the coverage test); route leaking into prod (mitigated: `APP_ENV` gate at registration + runtime handler check); partial coverage gap (mitigated: enumerating coverage test).
- **Follow-up**: the fourth and final polish-mvp follow-up (`refine-timetrak-brand-and-product-visual-language`) will consume the showcase as the canonical surface against which brand changes are reviewed.
