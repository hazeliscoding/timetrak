## ADDED Requirements

### Requirement: Brand sub-surface in the component catalogue

The dev-only showcase (mounted at `/dev/showcase` only when `APP_ENV=dev`) SHALL expose the `brandmark` partial as a documented entry in its component catalogue, either as a dedicated sub-surface at `/dev/showcase/brand` or as an anchored section of the existing component catalogue page. The brand entry MUST render the real `brandmark` partial — not a re-implementation — for every documented `Size` variant (at minimum `md` and `sm`) and for both `Decorative: false` and `Decorative: true` states, colocated with copy-ready snippets matching the existing showcase-snippet convention. The brand entry MUST also display the shipped `web/static/favicon.svg` as a live preview with an accompanying note explaining that the favicon follows the OS `prefers-color-scheme` preference rather than the app's in-tab `data-theme` toggle. The brand entry MUST cross-link to its spec reference (this requirement and the `ui-partials` "Brand mark partial" requirement) and to the partial source. The brand surface MUST remain under the existing showcase authentication, environment, non-linking, accessibility, and zero-new-runtime-dependency requirements — in particular it MUST NOT be reachable outside `APP_ENV=dev`, MUST NOT be linked from any user-facing template, MUST pass the WCAG 2.2 AA axe-core smoke on every page that hosts it, and MUST NOT introduce any new runtime dependency (no icon library, no SVG optimizer, no client-side renderer).

#### Scenario: Brand entry reachable only in dev

- **WHEN** the server runs with `APP_ENV=dev` and an authenticated session requests the brand surface (either `/dev/showcase/brand` or the brand anchor inside `/dev/showcase`)
- **THEN** the response status is 200
- **AND** the rendered output contains the live `brandmark` partial output for each documented variant
- **AND** an `<img>` preview of `/static/favicon.svg` is rendered on the same surface

#### Scenario: Brand entry unreachable in non-dev environments

- **WHEN** the server runs with a non-dev `APP_ENV` (for example `prod` or `staging`) and any client requests `/dev/showcase/brand`
- **THEN** the response status is 404
- **AND** the route MUST NOT appear in the registered route table
- **AND** no user-facing template contains a link whose href resolves to the brand surface

#### Scenario: Brand entry renders the real partial, not a copy

- **WHEN** the brand entry renders each `Size` × `Decorative` variant
- **THEN** the rendered HTML is produced by invoking `{{template "brandmark" ...}}` against the live template loader
- **AND** any divergence between the brand entry's output and the partial's own output for the same `dict` is a failing assertion in the showcase snippet-integrity test

#### Scenario: Brand entry passes the existing accessibility smoke

- **WHEN** axe-core runs against the brand surface with the rulesets `wcag2a`, `wcag2aa`, and `wcag22aa`
- **THEN** zero `serious` or `critical` violations are reported
- **AND** the brandmark's accessible name is "TimeTrak" on non-decorative variants
- **AND** the favicon preview carries a text alternative or is marked `aria-hidden`, per the showcase's existing decorative-image convention

#### Scenario: Brand entry introduces no new runtime dependency

- **WHEN** the brand entry, the `brandmark` partial, and the favicon ship
- **THEN** `go.mod`, `go.sum`, and `web/static/vendor/` are unchanged relative to the pre-change baseline
- **AND** no icon library, SVG optimiser, or client-side renderer is added
