## Why

The **first-impression surfaces** of TimeTrak — the dashboard and every in-product empty state — carry the highest brand gravity per square pixel. They are what a new user sees before any real data exists, and they are where a returning user lands. Today they are a mix of inline `<p class="muted">` blocks, ad-hoc card layouts with inline styles, and generic copy ("Add a client and a project to start tracking time.") that sits outside the canonical `empty_state` partial and outside any documented voice contract. The dashboard's "Jump back in" block uses inline muted text instead of the canonical partial (`web/templates/dashboard.html:19`), and `dashboard_summary` renders a bespoke "No billable entries yet" message (`web/templates/partials/dashboard_summary.html:26`) that no other surface shares.

Two sibling proposals are landing the contracts that make a sharpening pass tractable:

- **`sharpen-component-identity`** (active) locks down shape-language, two-weight borders, accent-rationing, `tabular-nums`, and the timer-as-signature-object contract — the visual grammar this change consumes on the dashboard.
- **`refine-timetrak-brand-and-product-visual-language`** (active) ships `docs/timetrak_brand_guidelines.md` with voice principles and empty-state / confirmation / error microcopy rules — the voice contract this change applies to every empty state.

This change is the follow-on those two proposals explicitly defer: apply the now-accepted component-identity and voice contracts to the dashboard's signature surface and to every empty state in the app, in one coordinated pass, so the "calm tool" register the brand work codifies is actually legible on the surfaces users see first.

Why one change, not two: the dashboard zero-state (no projects, no entries, no running timer) *is* an empty-state surface — the most visible one in the product. Splitting it would force arbitrary coordination between two changes touching the same `ui-partials` spec and the same `dashboard.html` template. The unifying thesis is zero-state and signature-state surfaces, not "dashboard stuff and empty-state stuff."

## What Changes

- Promote the canonical `empty_state` partial from "available" to **mandatory** for every list, table, or filtered-collection surface that renders a "no rows" view. No bespoke `<p class="muted">` empty messages, no inline muted cards. Currently seven surfaces; this change migrates the two non-compliant ones (dashboard "Jump back in", `dashboard_summary` "No billable entries yet") and re-audits the rest against the voice rules.
- Sharpen the **dashboard zero-state** as a first-class design surface, not a fallback. When the workspace has no projects, no entries, and no running timer, the dashboard renders a single cohesive `empty_state` that orients the user with one primary action ("Create your first client"), not three disjoint cards with inline copy. When there are entries but no running timer, the timer-control's idle identity (delivered by `sharpen-component-identity`) carries the surface; summary cards render with crisp zero values using the accepted `tabular-nums` contract instead of ad-hoc fallback strings.
- Align every empty-state copy string with `docs/timetrak_brand_guidelines.md` voice rules: domain-specific verbs and nouns ("Create your first client", "Widen the date range"), never generic productivity language, always an actionable next step when a sensible one exists, `aria-live="polite"` on HTMX-delivered empties. This is a microcopy sweep scoped to the seven live empty surfaces plus the two newly-migrated ones — not an app-wide copy audit.
- Introduce a **dashboard zero-state spec requirement** under `ui-partials` that codifies what the dashboard surface MUST render in each state (no projects / entries but no timer / running timer), so future timer or dashboard changes can't silently regress the first-impression surface.
- Extend the dev-only showcase with a `dashboard-states` section that renders the dashboard surface in every documented state side-by-side, and an `empty-states` section that renders every live empty-state variant (with and without action). This is the review surface the `sharpen-component-identity` change established for components — same contract, applied to surfaces.
- Add a browser contract test asserting: (a) every live empty-state surface uses `partials/empty_state` (no ad-hoc empties), (b) HTMX-delivered empties carry `aria-live="polite"`, (c) the dashboard zero-state renders exactly one primary action, (d) copy strings on empty states do not match a small deny-list of generic SaaS phrases ("Boost productivity", "Get started", "Welcome to…") sourced from the brand guidelines.
- **Out of scope (explicit):**
  - No change to any `empty_state` partial *structure* (context keys, CSS classes, focus behavior). The partial is already correct; only its *usage* becomes mandatory.
  - No new accent color, no new token, no new severity hue. The visual grammar comes from `sharpen-component-identity` verbatim.
  - No change to the timer control, status chip, or table treatment — those are owned by `sharpen-component-identity` and consumed as-is here.
  - No brand wordmark, favicon, or title-convention work — owned by `refine-timetrak-brand-and-product-visual-language`.
  - No app-wide copy audit beyond empty states. Validation errors, confirmation dialogs, and toast microcopy stay as-is; a future change can sweep them against the same voice doc.
  - No new route, no new handler, no new domain logic, no new migration. Templates and one CSS rule for the dashboard zero-state layout; everything else is copy and partial-usage.

## Capabilities

### New Capabilities

- _None._ The scope lives entirely inside `ui-partials` (empty-state usage contract + dashboard zero-state requirement) and `ui-showcase` (new gallery sections). A new capability would be overfit for a microcopy + partial-usage sweep.

### Modified Capabilities

- `ui-partials` — ADDS a requirement that every in-product list, table, or filtered-collection surface MUST use `partials/empty_state` for its zero-rows view (no inline `<p class="muted">`, no bespoke cards), and ADDS a requirement codifying the dashboard surface's three states (no projects / entries but no timer / running timer) and what each MUST render.
- `ui-showcase` — ADDS `dashboard-states` and `empty-states` gallery sections to the documented coverage obligation; existing partial-coverage enforcement extends to verify every empty-state consumer is represented.

## Impact

- **Templates modified:** `web/templates/dashboard.html` (zero-state restructure + "Jump back in" → `empty_state`), `web/templates/partials/dashboard_summary.html` ("No billable entries yet" migration), `web/templates/clients/index.html`, `web/templates/projects/index.html`, `web/templates/partials/rates_table.html`, `web/templates/partials/report_empty.html`, `web/templates/time/index.html` (copy sweep only; structure already canonical).
- **Templates added:** `web/templates/showcase/dashboard_states.html`, `web/templates/showcase/empty_states.html`.
- **CSS:** one new component-scoped rule in `web/static/css/app.css` for the dashboard zero-state layout (single primary-action card centered in the content area). No new tokens. No new semantic aliases.
- **Specs:** deltas under `openspec/changes/sharpen-dashboard-and-empty-states/specs/ui-partials/spec.md` and `.../specs/ui-showcase/spec.md`.
- **Tests:** `internal/e2e/browser/empty_states_test.go` (new) and an extension to an existing showcase-axe-smoke test to cover the two new showcase sections.
- **No backend, DB, or domain-logic changes.** No new dependencies. No migrations.
- **Risk:** copy changes may surface existing voice drift that needs fixing in this change's scope (e.g. `projects/index.html` "Projects live under clients." — currently correct but a re-read against the voice doc may tighten it further). Mitigated by the microcopy sweep being bounded to the seven live empty surfaces.
- **Assumptions (explicit dependencies on active changes):**
  1. `sharpen-component-identity` archives first. Design and specs here assume `tabular-nums`, the two-weight border, the status-chip contract, and the timer-control identity are accepted baseline. If that change is still active when this one is ready to implement, block on it.
  2. `refine-timetrak-brand-and-product-visual-language` archives first. The microcopy sweep cites `docs/timetrak_brand_guidelines.md` voice rules as its standard. If the brand doc is still in the active change folder when implementation starts, block on it.
- **Follow-ups (not part of this change):** voice sweep over validation errors, confirmation dialogs, and toast microcopy; a separate proposal if the browser test's deny-list approach to generic-SaaS phrases proves to need a positive spec (preferred-noun inventory) rather than a negative one.
