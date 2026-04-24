## 1. Token declarations

- [x] 1.1 Added the eight `--text-*` size tokens (`--text-xs` 0.75rem → `--text-3xl` 1.75rem) to `:root` in `web/static/css/tokens.css` under the Typography comment block, with per-token inline use-case hints and a cross-reference to the amended `ui-foundation` Scale Tokens requirement.
- [x] 1.2 Added the four `--weight-*` tokens (`regular`/`medium`/`semibold`/`bold`).
- [x] 1.3 Added the four `--leading-*` tokens (`none`/`tight`/`snug`/`normal`).
- [x] 1.4 Tokens are theme-invariant; no dark-theme overrides added.

## 2. Migrate `app.css`

- [x] 2.1 Migrated every `font-size:` raw value to the matching `--text-*` token. `body` → `--text-base` (consuming the token, not the raw 15px). Full sweep covered `h1`/`h2`/`h3`, `.field .hint`/`.error`, `.table thead th`, `.table tbody th`, `.badge`, `.tt-chip`, `.tt-timer-meta`, `.tt-timer-elapsed`, `.tt-timer-started`, grandfathered `.timer .elapsed`.
- [x] 2.2 Migrated every `font-weight:` raw value to the matching `--weight-*` token. Call sites: `.nav a`, `.nav a[aria-current="page"]`, `.btn`, `.field label`, `.table caption`, `.table thead th`, `.table tbody th`, `h1`, `h2`, `h3`, `.badge`, `.tt-chip`, `.tt-theme-seg[aria-pressed="true"]`, `.tt-timer-elapsed`, `.timer .elapsed`.
- [x] 2.3 Migrated every `line-height:` raw value to the matching `--leading-*` token. Call sites: `body`, `h1, h2, h3, h4`, `.tt-chip`, `.tt-chip-glyph`, `.tt-chip-label`, `.tt-theme-seg-glyph`, `.tt-theme-seg-label`, `.tt-timer-elapsed`.
- [x] 2.4 Preserved relative-to-parent `em` sizes on decorative glyphs (`.tt-chip-glyph { font-size: 0.75em }` / `.tt-theme-seg-glyph { font-size: 0.9em }`). Verified via grep that no raw `<n>rem` / `<n>px` / `<n>%` / bare numeric `font-weight`/`line-height` values remain in `app.css`.

## 3. Documentation

- [x] 3.1 Extended `web/static/css/README.md` §Scale tokens → Typography with the three sub-scales, per-token use cases, and a cross-link to the amended `ui-foundation` spec.
- [x] 3.2 Rewrote `docs/timetrak_ui_style_guide.md` §Type Hierarchy as a Role → `--text-*` × `--weight-*` mapping table. Each prose role now has concrete tokens.

## 4. Verify

- [x] 4.1 `make fmt && make vet && make test` — all green.
- [ ] 4.2 Manual eyeball pending user action — `make run` in light AND dark themes to confirm no visual shift. Migration uses the same numeric values, just through tokens, so computed `font-size` / `font-weight` / `line-height` on every element should be identical to pre-change.
- [ ] 4.3 Manual spot check of `/dev/showcase/components` pending user action.

## 5. Commit and archive

- [ ] 5.1 Commit via `tt-conventional-commit`.
- [ ] 5.2 Archive via `/opsx:archive add-type-scale-tokens`.
