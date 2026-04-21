## 1. Package scaffolding and dev-only gating

- [x] 1.1 Create `internal/showcase/` package with `handler.go`, `catalogue.go`, and `snippets.go` files and a package doc comment describing the dev-only scope.
- [x] 1.2 Read `APP_ENV` from the existing config surface (same accessor `cmd/web/main.go` uses) and expose a predicate (e.g. `IsDev(cfg)`) in `internal/showcase` for both registration-time and request-time gating.
- [x] 1.3 In `cmd/web/main.go`, register the showcase route group (`/dev/showcase`, `/dev/showcase/tokens`, `/dev/showcase/components`, `/dev/showcase/components/{slug}`) ONLY when `IsDev(cfg)` returns true; document the gate with an inline comment.
- [x] 1.4 In each showcase handler, add a defense-in-depth early return that responds with 404 when `APP_ENV` is not dev, matching the existing not-found renderer used for cross-workspace access.
- [x] 1.5 Wrap the registered route group with the existing `authz.RequireAuth` middleware; do NOT require a workspace. Document in-code that the showcase is the one authenticated surface that does not require workspace scoping.

## 2. Catalogue metadata (components and tokens)

- [x] 2.1 In `internal/showcase/catalogue.go`, declare the `ComponentEntry`, `ComponentExample`, `DictKeyDoc`, and `TokenEntry` types per the design doc.
- [x] 2.2 Declare the `ComponentEntries` slice and populate one entry per partial enumerated in `web/templates/partials/README.md`: `flash`, `spinner`, `empty_state`, `form_errors`, `form_field`, `pagination`, `confirm_dialog`, `client_row`, `project_row`, `entry_row`, `rate_row`, `rate_form`, `rates_table`, `timer_widget`, `tracking_error`, `dashboard_summary`, `report_summary`, `reports.partial.results`, `reports.partial.empty`. Each entry MUST set `PartialName`, `SourcePath`, `SpecRef`, `Purpose`, `DictKeys`, and at least one `Example`.
- [x] 2.3 Declare the `TokenEntries` slice and populate one entry per semantic alias and per scale token currently in `web/static/css/tokens.css` (colors, spacing, radius, typography, motion, elevation, z-index, breakpoints) plus a clearly-separated section for primitive ramps (`--neutral-*`, `--accent-*`, severity anchors).
- [x] 2.4 Add a documented grandfather list (code-level constant + README-level note inside the showcase package) for any partial whose PartialName MUST NOT be enumerated by the coverage test (initially empty; add entries only when a partial is intentionally excluded).

## 3. Snippet fixtures

- [x] 3.1 Create `internal/showcase/snippets/` (or an `//go:embed` block colocated with the catalogue) and add one fixture file per `ComponentExample`, containing the copy-ready `{{template "..."}}` snippet text.
- [x] 3.2 Wire `snippets.go` to load fixtures at package init (embed.FS) and expose a `LookupSnippet(id string) (string, error)` used by the handler.
- [x] 3.3 Ensure every `ComponentExample` references the same `dict` payload that the showcase uses to render the live example — one struct, two consumers, no drift.

## 4. Showcase templates

- [x] 4.1 Create `web/templates/showcase/index.html` rendering the catalogue navigation, the contribution guide section, and links to the tokens and components sub-pages. Use the existing `layouts/app.html` shell so the theme toggle and nav behave identically.
- [x] 4.2 Create `web/templates/showcase/tokens.html` rendering semantic aliases first, scale tokens next, and primitive ramps last with a visible note that components MUST NOT reference primitive ramps directly. Color entries render as swatches using `var(--<token>)`; spacing as sized bars; typography as sample text; motion as a short keyframed demo; radius / elevation / z-index / breakpoint as labeled previews.
- [x] 4.3 Create `web/templates/showcase/components.html` rendering each `ComponentEntry` as an anchored section. Every live example calls `{{template "<PartialName>" .Dict}}` against the real template loader. Every snippet is rendered as a `<pre><code>` block.
- [x] 4.4 Render per-entry cross-links: source file link (to `web/templates/partials/<name>.html`) and spec reference link (to the relevant requirement anchor in `openspec/specs/ui-partials/spec.md` or `openspec/specs/ui-foundation/spec.md`).
- [x] 4.5 Render per-entry accessibility notes pulled from `ComponentEntry.A11yNotes`, keeping the existing visible-label / focus-target / non-color-status copy verbatim from the partials README.

## 5. Cross-linking documentation

- [x] 5.1 Add a one-line pointer at the top of `web/static/css/README.md` naming `/dev/showcase/tokens` as the browser-visible reference for the token catalogue.
- [x] 5.2 Add a one-line pointer at the top of `web/templates/partials/README.md` naming `/dev/showcase/components` as the browser-visible reference for the partial catalogue.
- [x] 5.3 Ensure no user-facing template, nav, or footer contains a link to `/dev/showcase` or any sub-route (verify via a grep-based guard in tests).

## 6. Partial-coverage and snippet-integrity tests

- [x] 6.1 Add `internal/showcase/coverage_test.go` that enumerates `.html` files under `web/templates/partials/` and asserts each non-grandfathered file stem appears exactly once as a `ComponentEntry.PartialName`.
- [x] 6.2 Add a snippet-integrity unit test that loads the template set (via the existing loader) and asserts every `ComponentEntry.PartialName` and every `Example.PartialName` resolves against the loader (no references to missing block names).
- [x] 6.3 Add a "no production link" test that greps shipped templates under `web/templates/` (excluding `web/templates/showcase/`) and asserts no reference to `/dev/showcase` appears.
- [x] 6.4 Add a "dev-only registration" test that constructs the router with `APP_ENV=prod` and asserts `/dev/showcase` returns 404.

## 7. Browser contract coverage

- [x] 7.1 Add `internal/e2e/browser/showcase_test.go` gated by `//go:build browser`, reusing the existing harness (server bootstrap + `internal/shared/testdb`). Run every scenario with `APP_ENV=dev` for the server under test.
- [x] 7.2 Assert `GET /dev/showcase`, `GET /dev/showcase/tokens`, and `GET /dev/showcase/components` all return 200 with `Content-Type: text/html`.
- [x] 7.3 Assert every in-page anchor on `/dev/showcase/components` (`#entry-<slug>`) resolves to an element present in the DOM.
- [x] 7.4 Inject axe-core on both catalogue pages and assert zero violations at `impact: serious` or `impact: critical` across `wcag2a`, `wcag2aa`, `wcag22aa`.
- [x] 7.5 Assert the theme toggle still flips `data-theme` on the showcase surface and at least one token swatch resolves to a different `background-color` value across themes.

## 8. Accessibility and UI polish validation

- [x] 8.1 Verify every interactive element on showcase pages (nav anchors, theme toggle, source links, spec links, copy-button if added) has a visible label, meets the 24×24 target size bar, and shows the global focus ring.
- [x] 8.2 Verify all color samples render a visible label or token name — color alone never conveys meaning.
- [x] 8.3 Verify all sample text for typography tokens meets ≥4.5:1 contrast against the sample surface in both light and dark theme.
- [x] 8.4 Verify `prefers-reduced-motion: reduce` collapses the motion-demo token previews, matching the existing foundation contract.

## 9. Build gates and cleanup

- [x] 9.1 Run `make fmt` and commit the formatted result.
- [x] 9.2 Run `make vet` and resolve any issues surfaced by the new package.
- [x] 9.3 Run `make test` and confirm the new coverage, snippet-integrity, "no production link", and "dev-only registration" tests pass.
- [ ] 9.4 Run `make test-browser` (after `make browser-install` if needed) and confirm the new `showcase_test.go` scenarios pass; attach any failure artifacts per the existing browser-contract convention.
- [x] 9.5 Run `openspec validate` and confirm the change is archive-ready.
- [x] 9.6 Update `openspec/changes/create-component-library-showcase-and-usage-docs/tasks.md` to mark all tasks complete before archiving.
