## Why

TimeTrak has completed MVP bootstrap and every change on the Stage 2 (Stabilize) roadmap: workspace authorization hardening, timer integrity, rate resolution, reporting correctness, UI/accessibility polish, reusable UI partials, the component library foundation, the dev showcase, and — just now — the spec baseline cleanup. `openspec/changes/` currently holds only `archive/` with no active changes, and `openspec/specs/` is in good shape (11 capabilities, every Purpose real). The project is ready to shift out of "stabilize what shipped" mode and into Stage 3 — continuous feature delivery against the accepted baseline, per `docs/openspec_flow_nothing_to_mvp_to_real_product.md#stage-3--real-product`.

The framing surfaces in the repo haven't caught up: `CLAUDE.md` still declares "Post-MVP — Stage 2 (Stabilize)", `openspec/config.yaml` still describes Stage 2 priorities, `CLAUDE.md:24` enumerates only the seven MVP-era domains (current baseline has eleven), and `CLAUDE.md:9` points at `docs/timetrak_post_mvp_openspec_roadmap.md` which does not exist. Future contributors — human or AI — will read those as authoritative and scope-shape Stage 3 work against a Stage 2 picture. This change makes the repo's self-description match where the product actually is before any Stage 3 feature lands.

## What Changes

- Update the `## Project Status` block in `CLAUDE.md` from "Post-MVP — Stage 2 (Stabilize)" to "Stage 3 — Continuous Delivery" and rewrite the paragraph that enumerates Stage 2 priorities to describe Stage 3 operating posture (per-change evolution of the baseline, no umbrella scopes, archive often).
- Replace the broken `docs/timetrak_post_mvp_openspec_roadmap.md` reference in `CLAUDE.md:9` with a new `docs/timetrak_stage_3_roadmap.md` that names 3–5 candidate Stage 3 initiatives at one-paragraph depth each (no commitment, no ordering beyond "likely-next vs later").
- Refresh `CLAUDE.md:24` to reference the accepted baseline domains as the full current set (auth, clients, projects, rates, reporting, tracking, ui-browser-tests, ui-foundation, ui-partials, ui-showcase, workspace), or — preferred — point the line at `openspec/config.yaml` as the source of truth and remove the inline enumeration so it cannot drift again.
- Update the `context.Stage` paragraph in `openspec/config.yaml` (lines 6–8) from Stage 2 priority language to a Stage 3 operating description aligned with the flow doc.
- Create `docs/timetrak_stage_3_roadmap.md`. This is a narrative doc, not a behavioral baseline. Explicit non-goals: it is not a spec, not a backlog, not a commitment; it is a short orientation for contributors picking their first Stage 3 change.
- **Out of scope:** proposing, designing, or implementing any actual Stage 3 feature (CSV export, invoices, team workspaces, mobile, notifications, etc.); adding or modifying any requirement in `openspec/specs/`; touching any Go source, migrations, templates, or tests; renaming or reorganizing `openspec/specs/` domains; fixing the latent `Rules for 'specs' must be an array of strings` OpenSpec CLI warning.

## Capabilities

### New Capabilities

- _None._ Stage 3 is an operating posture, not a product capability. No spec folder is created.

### Modified Capabilities

- _None._ No requirement is added, removed, modified, or renamed. All edits are to non-spec prose (`CLAUDE.md`, `docs/`, `openspec/config.yaml`).

## Impact

- **Files:** `CLAUDE.md`, `openspec/config.yaml`, `docs/timetrak_stage_3_roadmap.md` (new).
- **Code / runtime:** none. No Go, migrations, templates, or tests touched.
- **Specs:** none. `openspec/specs/` is untouched.
- **Tooling / CI:** none. Build, test, and deploy paths unchanged.
- **Follow-ups:** the first actual Stage 3 feature change (proposed separately via `/opsx:propose <name>`). Candidates listed in the new roadmap doc are the user's to pick from and re-order.
- **Risks:** low. Main risk is the roadmap doc being read as a commitment rather than orientation; mitigated by an explicit "not a commitment / not a backlog" header on the doc and in `CLAUDE.md`.
- **Assumptions:** the OpenSpec validator will reject the change for having no requirement deltas. This change reuses the documented no-op verbatim-copy pattern from `openspec/changes/archive/2026-04-21-clean-up-spec-baseline-after-mvp/design.md` Decision 1 to satisfy the validator without touching any requirement. If that pattern is ever retired, this change's deltas would need to be regenerated accordingly.
