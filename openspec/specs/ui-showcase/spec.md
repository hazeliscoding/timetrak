# ui-showcase Specification

## Purpose
The ui-showcase capability defines a dev-only `/dev/showcase` surface —
mounted only when `APP_ENV=dev`, reachable by any authenticated session
(workspace optional), and never linked from user-facing templates —
that hosts TimeTrak's token and component catalogues. The catalogues
cover every public semantic alias, scale token, and canonical partial,
rendering real partials rather than re-implementations, colocating
copy-ready snippets with live examples, documenting variant and state
permutations, and cross-linking to source and specs. A contribution
guide accompanies the catalogues, the showcase passes a WCAG 2.2 AA
smoke on every page, and it MUST NOT introduce any new runtime
dependency.
## Requirements
### Requirement: Dev-only showcase surface

TimeTrak SHALL expose a developer-facing showcase surface mounted at `/dev/showcase` (plus sub-routes for the token catalogue and component catalogue) that is reachable ONLY when the application is running with `APP_ENV=dev`. In any non-development environment, the route MUST NOT be registered at server startup AND the handler MUST return HTTP 404 if invoked. The showcase MUST NOT be linked from any user-facing template, navigation, or footer.

#### Scenario: Showcase reachable in dev environment

- **WHEN** the server is started with `APP_ENV=dev` and an authenticated session requests `GET /dev/showcase`
- **THEN** the response status is 200 and the showcase index is rendered

#### Scenario: Showcase unreachable in production environment

- **WHEN** the server is started with a non-dev `APP_ENV` (for example `prod` or `staging`) and any client requests `GET /dev/showcase` or any `/dev/showcase/*` sub-route
- **THEN** the response status is 404
- **AND** the route MUST NOT appear in the registered route table

#### Scenario: Showcase is not linked from user-facing templates

- **WHEN** any shipped user-facing template (layouts, navigation, footer, dashboards, domain pages) is rendered
- **THEN** it MUST NOT contain a hyperlink whose href points at `/dev/showcase` or any sub-route

### Requirement: Showcase requires authenticated session but no workspace

The showcase SHALL require an authenticated session (reusing the existing `authz.RequireAuth` middleware) but MUST NOT require the session to be scoped to a workspace. An authenticated user without a workspace MUST be able to view every showcase page.

#### Scenario: Unauthenticated request is redirected

- **WHEN** an unauthenticated client requests `GET /dev/showcase` in a dev environment
- **THEN** the response redirects to the login flow consistent with other authenticated routes

#### Scenario: Authenticated user without a workspace views showcase

- **WHEN** an authenticated user who has not created or joined a workspace requests `GET /dev/showcase` in a dev environment
- **THEN** the response status is 200 and the showcase renders without referencing workspace data

### Requirement: Component catalogue covers every reusable partial

The component catalogue SHALL contain exactly one entry per reusable partial documented in `web/templates/partials/README.md`. For each partial the entry MUST render:

- the partial's display name,
- a one-paragraph prose description of its purpose,
- the documented `dict` context keys (name, required/optional, default where applicable),
- at least one live rendering of the partial invoked through the real template loader,
- a copy-ready template snippet displayed inside a `<pre><code>` block,
- the partial's documented accessibility obligations (label source, focus target, non-color status conveyance, target-size notes),
- a link to the partial's source file under `web/templates/partials/<name>.html`,
- a link to the relevant requirement in `openspec/specs/ui-partials/spec.md` or `openspec/specs/ui-foundation/spec.md`.

A partial-coverage test SHALL enumerate files under `web/templates/partials/` and fail the build when any non-grandfathered partial lacks a showcase entry or when a partial has more than one entry.

#### Scenario: Every documented partial has an entry

- **WHEN** the partial-coverage test enumerates `.html` files under `web/templates/partials/`
- **THEN** for every non-grandfathered partial there is exactly one `ComponentEntry` whose `PartialName` matches the file stem

#### Scenario: New partial shipped without a showcase entry

- **WHEN** a contributor adds `web/templates/partials/<new_name>.html` but does NOT add a corresponding `ComponentEntry`
- **THEN** the partial-coverage test fails with a message naming the missing partial

#### Scenario: Entry documents partial context contract

- **WHEN** a reader opens the showcase entry for a partial
- **THEN** the entry lists every `dict` key that partial consumes, marking required vs optional and the default value for optional keys

### Requirement: Token catalogue covers every semantic alias and scale token

The token catalogue SHALL contain exactly one entry per semantic alias and per scale token enumerated in `web/static/css/README.md` and declared in `web/static/css/tokens.css`. Each entry MUST render:

- the token's CSS custom property name,
- a visible sample appropriate to the token family (swatch for color, sizing bar for spacing, sample text for typography, motion demo for duration/easing, labeled preview for radius / elevation / z-index / breakpoint),
- the documented semantic role or usage guidance.

Primitive ramp tokens MUST be rendered in their own catalogue section, clearly separated from semantic aliases, with a visible note that primitive ramps MUST NOT be consumed directly by components.

The catalogue MUST honor the existing `data-theme` toggle so that switching between light and dark theme updates every sample in place.

#### Scenario: Every semantic alias has an entry

- **WHEN** the token catalogue page is rendered
- **THEN** every semantic alias enumerated in `web/static/css/README.md` (`--color-bg`, `--color-surface`, `--color-surface-alt`, `--color-text`, `--color-text-muted`, `--color-border`, `--color-border-strong`, `--color-accent`, `--color-accent-hover`, `--color-accent-soft`, `--color-focus`, and the severity pairs) appears exactly once

#### Scenario: Every scale token has an entry

- **WHEN** the token catalogue page is rendered
- **THEN** every enumerated spacing, radius, typography, motion, elevation, z-index, and breakpoint token appears exactly once with a visible sample appropriate to its family

#### Scenario: Primitive ramp section is clearly marked

- **WHEN** the token catalogue page is rendered
- **THEN** primitive ramp tokens appear in a dedicated section with a visible note that components MUST NOT consume them directly

#### Scenario: Theme toggle updates samples live

- **WHEN** a viewer toggles `data-theme` between light and dark on the token catalogue page
- **THEN** every color swatch, text sample, and border preview reflects the resolved value under the active theme without a full page reload

### Requirement: Showcase renders real partials, never re-implementations

Every live example on the component catalogue SHALL be produced by invoking the real partial through the application's template loader (for example via `template.ExecuteTemplate(w, "<partial-name>", <dict>)`). The showcase MUST NOT define, embed, or duplicate the markup of any documented partial. If a partial's `dict` contract drifts from the showcase example, the showcase page MUST fail to render in dev.

#### Scenario: Live example is rendered via the template loader

- **WHEN** a component catalogue entry renders its live example
- **THEN** the rendered HTML is produced by the same template loader that serves product pages, invoked against the block name documented in `web/templates/partials/README.md`

#### Scenario: Dict contract drift surfaces immediately

- **WHEN** a partial's required `dict` keys change and the showcase example is not updated to match
- **THEN** the showcase page fails to render in dev with a template-execution error naming the missing key

### Requirement: Copy-ready snippets are colocated with live examples

Each component catalogue entry SHALL display a copy-ready template snippet (inside a `<pre><code>` block) that is loaded from a fixture file colocated with the showcase source. The fixture payload referenced by a snippet MUST be the same `dict` payload used to render the live example; the two MUST NOT be authored independently. A contract test SHALL assert that every snippet references a `PartialName` that resolves against the template loader.

#### Scenario: Snippet and live example share the same dict payload

- **WHEN** a component catalogue entry renders an example labeled "Success flash"
- **THEN** the copy-ready snippet displayed in the same entry calls the same block name and passes the same `dict` keys as the fixture used to render the live example

#### Scenario: Snippet references a template that does not exist

- **WHEN** a fixture references a template block name that is not registered in the template loader
- **THEN** the snippet-integrity contract test fails with a message naming the missing block

### Requirement: Component catalogue documents variants and state permutations

For every partial that ships documented variants (for example `flash` severities `success` / `info` / `warn` / `error`; button variants `primary` / `secondary` / `ghost` / `danger`; badges `running` / `billable` / `archived` / `warning`; form-field states default / focused / invalid / disabled; `empty_state` with and without action), the showcase entry SHALL render one live example per documented variant and one live example per documented state. The showcase MUST NOT invent states or variants that do not exist in the partial's documented contract or in the CSS authoring contract.

#### Scenario: Flash severity variants are rendered

- **WHEN** a reader opens the `flash` catalogue entry
- **THEN** exactly four live examples are rendered, one each for `success`, `info`, `warn`, and `error`, with the corresponding ARIA role visible in the markup

#### Scenario: Undocumented variant is attempted

- **WHEN** a contributor proposes a showcase entry that adds a button variant not present in `openspec/specs/ui-foundation/spec.md`
- **THEN** the proposal is rejected until the variant is first amended into the foundation spec

### Requirement: Showcase cross-links to source and specs

Each showcase entry SHALL include a hyperlink to the source file it documents (partial file or `tokens.css`) and a hyperlink or stable reference to the corresponding requirement in `openspec/specs/ui-partials/spec.md` or `openspec/specs/ui-foundation/spec.md`. The authoring READMEs (`web/static/css/README.md` and `web/templates/partials/README.md`) SHALL each contain a short pointer to `/dev/showcase` identifying it as the browser-visible reference.

#### Scenario: Component entry links to partial source

- **WHEN** a reader views a component catalogue entry
- **THEN** the entry includes a visible link to `web/templates/partials/<name>.html`

#### Scenario: Token entry links to token declaration

- **WHEN** a reader views a token catalogue entry
- **THEN** the entry includes a visible reference to the token's declaration in `web/static/css/tokens.css`

#### Scenario: READMEs point at the showcase

- **WHEN** a reader opens `web/static/css/README.md` or `web/templates/partials/README.md`
- **THEN** the document contains a short pointer that names `/dev/showcase` as the browser-visible reference surface

### Requirement: Contribution guide accompanies the catalogue

The showcase SHALL include a contribution guide section (either on the index or as a dedicated sub-route) that describes how to add a new component entry and how to add a new token entry, citing the coverage test that enforces completeness.

#### Scenario: Contributor looks up how to add an entry

- **WHEN** a contributor opens `/dev/showcase` looking for guidance on documenting a new component
- **THEN** they find a section describing how to add a `ComponentEntry`, a fixture snippet, and satisfy the partial-coverage test

### Requirement: Showcase passes WCAG 2.2 AA smoke

Showcase pages SHALL pass an axe-core smoke test under the `wcag2a`, `wcag2aa`, and `wcag22aa` tag sets with zero violations at impact `serious` or `critical`. The browser contract test under `internal/e2e/browser/` MUST cover the showcase index and at least one component catalogue page.

#### Scenario: Axe smoke on showcase index

- **WHEN** the browser contract test navigates to `/dev/showcase`
- **THEN** axe-core reports no `serious` or `critical` violations for `wcag2a`, `wcag2aa`, or `wcag22aa`

#### Scenario: Axe smoke on component catalogue page

- **WHEN** the browser contract test navigates to `/dev/showcase/components`
- **THEN** axe-core reports no `serious` or `critical` violations for `wcag2a`, `wcag2aa`, or `wcag22aa`

### Requirement: Showcase introduces zero new runtime dependencies

The showcase implementation SHALL NOT introduce a new Go module dependency, a Node-based build step, a markdown renderer, a syntax-highlighting library, a SPA framework, or a client-side state library. It MUST render with the stack already present: Go stdlib `net/http`, `html/template`, HTMX, and the existing CSS token / component surface.

#### Scenario: Proposal attempts to add a build tool

- **WHEN** a contributor proposes wiring Storybook, MDX, esbuild, or a similar tool as part of showcase implementation
- **THEN** the proposal is rejected; snippets render as plain `<pre><code>` text and live examples render via `template.ExecuteTemplate`

#### Scenario: Go module graph is unchanged by the showcase

- **WHEN** the showcase change lands
- **THEN** `go.mod` and `go.sum` contain no new direct dependencies attributable to the showcase implementation

### Requirement: Component identity states rendered per component

For every component governed by `ui-component-identity` (timer control, data table, status chip, and any additional components added to the identity contract in future changes), the showcase component catalogue SHALL render one live example per documented identity state. At minimum:

- **Timer control:** `idle`, `running`, and — when the reduced-motion media query is simulated via a toggle or an in-page control group — `running` with the static-dot fallback. Each rendering MUST use the real `partials/timer_control`.
- **Data table (via `.table` CSS contract):** a dedicated gallery section renders a live `<table class="table">` demonstrating `default row`, `hover row` (via a CSS `:hover` demonstration note or an `is-hover` simulation class), `selected/focused row` (`aria-selected="true"`), and `empty` (using `partials/empty_state`). Numeric-column treatment (`.col-num` / `tabular-nums`, right-aligned) MUST be visible in at least one row.
- **Status chip (via `partials/status_chip`):** one rendering per enumerated `kind` × documented `variant`, plus one example that demonstrates non-color-only conveyance (glyph + label).

These identity-state renderings are additive to the existing variant-permutation coverage and do not replace it.

#### Scenario: Timer shows idle and running

- **WHEN** a reader opens the timer entry in `/dev/showcase`
- **THEN** at least two live renderings are present — one for `idle` (neutral pill, `Start timer` label) and one for `running` (accent-soft fill, 2px accent border, tabular-nums elapsed time, distinct `Stop` affordance).

#### Scenario: Table selected-row state is rendered

- **WHEN** a reader opens the table-states gallery section in `/dev/showcase/components`
- **THEN** a live rendering of a row with the selected/focused state (2px accent inside-left edge via `box-shadow: inset 2px 0 0 0 var(--color-accent)`, driven by `aria-selected="true"`) is present and visually distinguishable from the default and hover rows, and at least one column is marked `.col-num` demonstrating `tabular-nums` right-alignment.

#### Scenario: Chip state conveyance is shown

- **WHEN** a reader opens the status-chip entry in `/dev/showcase`
- **THEN** each enumerated `kind` renders once per documented `variant`, and at least one example explicitly demonstrates a glyph-plus-label combination to satisfy the non-color-alone rule.

### Requirement: Component identity checklist rendered on showcase index

The `/dev/showcase` index page SHALL render the component-identity review checklist defined in `ui-component-identity` above the component catalogue listing. The checklist MUST be sourced from the same canonical document (`docs/timetrak_ui_style_guide.md` or the spec) rather than re-authored in the showcase templates, so the two cannot drift.

Each checklist item MUST link or cross-reference the corresponding requirement in `openspec/specs/ui-component-identity/spec.md`.

#### Scenario: Reader opens showcase index

- **WHEN** an authenticated user opens `/dev/showcase` in a dev environment
- **THEN** the page renders the component-identity checklist (shape language, two-weight borders, tabular-nums, accent rationing, state coverage) above the component catalogue.

#### Scenario: Checklist item links to its requirement

- **WHEN** a reader clicks a checklist item
- **THEN** the link navigates to or stably references the matching requirement in `ui-component-identity`.

#### Scenario: Checklist drifts from the spec

- **WHEN** the showcase checklist text differs from the canonical source (style guide or spec)
- **THEN** the showcase test suite MUST fail until the two are reconciled.

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

