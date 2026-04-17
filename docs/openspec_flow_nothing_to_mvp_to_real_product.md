# OpenSpec Flow for TimeTrak: Nothing → MVP → Real Product

## Purpose

This document describes a practical OpenSpec workflow for a project moving through three stages:

1. **Nothing** — no real product yet, only ideas and design direction
2. **MVP** — the first buildable and shippable version
3. **Real Product** — an actively evolving application with ongoing features, fixes, and refactors

This is written for **TimeTrak**, but the overall flow works well for similar products.

---

## Core Idea

Use OpenSpec in two layers:

- **`openspec/specs/`** = accepted product behavior and current source of truth
- **`openspec/changes/<change-name>/`** = active proposed work that has not been merged into the baseline yet

Your long-form docs in `docs/` are still useful, but they are **reference material**, not the canonical accepted behavior once OpenSpec is in motion.

---

## Stage 0 — Nothing Yet

At this stage, you do not really have a product. You have:

- an idea
- architecture thoughts
- product goals
- maybe a design doc
- maybe a UI style guide

### Goal
Turn fuzzy product thinking into a structured starting point without pretending the app already exists.

### What to keep in the repo
- `docs/time_tracking_design_doc.md`
- `docs/time_trak_mvp_ui_style_guide.md`
- `openspec/config.yaml`

### What NOT to do yet
- do not overfill `openspec/specs/` with giant speculative specs
- do not write dozens of future features before the MVP shape is clear
- do not treat OpenSpec like a giant static requirements dump

### Best flow here
1. write or refine the long-form product/design docs in `docs/`
2. keep `openspec/config.yaml` concise and opinionated
3. create your **first real MVP change** in `openspec/changes/`
4. let that first change generate the planning artifacts
5. implement the MVP from those artifacts
6. archive into `openspec/specs/` once accepted

### Recommended command pattern
- use `/opsx:explore` when the idea is still fuzzy
- use `/opsx:propose <change-name>` when you are ready to turn it into actual work

### Good first change names
- `bootstrap-timetrak-mvp`
- `build-mvp-foundation`
- `create-initial-timetrak-domains`

### Deliverable mindset
At this stage, your output is:
- one serious MVP change
- not a huge permanent spec tree yet

---

## Stage 1 — Build the MVP

At this stage, OpenSpec should help you define and implement the first real version of the product.

### Goal
Get from “concept” to “working system” with enough structure to guide implementation, but not so much ceremony that you stall.

### What the MVP should produce
By the end of MVP, you should have:
- accepted baseline specs in `openspec/specs/`
- archived MVP work in `openspec/changes/archive/`
- code that matches the baseline closely enough that the specs are useful going forward

### Suggested MVP domain shape
Your accepted baseline will likely end up with domains like:
- `auth`
- `workspace`
- `clients`
- `projects`
- `tracking`
- `rates`
- `reporting`

### Recommended MVP flow
1. create one main MVP change
2. generate proposal, specs, design, and tasks
3. review and tighten the artifacts
4. implement from `tasks.md`
5. verify behavior against specs
6. archive the change
7. treat the archived result as the start of the product baseline

### MVP operating rule
During MVP, OpenSpec is mostly for:
- clarifying scope
- shaping implementation
- avoiding drift between idea and build

It is **not** yet about lots of parallel changes.

### MVP command pattern
For a simple solo workflow:
- `/opsx:propose bootstrap-timetrak-mvp`
- `/opsx:apply bootstrap-timetrak-mvp`
- `/opsx:archive bootstrap-timetrak-mvp`

If you want more control:
- `/opsx:new bootstrap-timetrak-mvp`
- `/opsx:continue bootstrap-timetrak-mvp`
- `/opsx:ff bootstrap-timetrak-mvp`
- `/opsx:verify bootstrap-timetrak-mvp`
- `/opsx:archive bootstrap-timetrak-mvp`

### When MVP is “done enough”
The MVP phase is complete when:
- the app works end to end
- the accepted behavior is represented in `openspec/specs/`
- the OpenSpec baseline is trustworthy enough to build on

---

## Stage 2 — Stabilize After MVP

This is the transition from “we built version one” to “we are now maintaining a product.”

### Goal
Make OpenSpec a lightweight operating system for product change instead of a one-time bootstrap tool.

### What changes now
After MVP:
- `openspec/specs/` becomes the real baseline
- `openspec/changes/` becomes a stream of small feature/refactor/fix changes
- `docs/` continues to hold higher-level reference material
- changes become narrower and more frequent

### New rule of thumb
After MVP, create **one change per meaningful unit of work**, not one giant umbrella change.

Good examples:
- `add-csv-export`
- `add-running-timer-reminders`
- `support-project-archival`
- `add-project-budget-alerts`
- `improve-rate-resolution-history`
- `refactor-timer-concurrency-guard`

Bad examples:
- `phase-2`
- `make-product-better`
- `all-reporting-work`
- `misc-cleanup`

### Typical flow after MVP
1. identify a feature, bug fix, or refactor
2. create a small change in `openspec/changes/`
3. generate or refine only the artifacts you need
4. implement the change
5. verify the implementation matches the change
6. archive quickly so the baseline stays current

### Operating principle
Archive often.
A stale change folder is less useful than a small archived change merged back into the baseline.

---

## Stage 3 — Real Product

At this stage, the app is a living product with repeated improvements.

### Goal
Use OpenSpec continuously to:
- evolve behavior safely
- keep product expectations visible
- reduce drift between product, UX, and implementation
- make refactors and new features easier to reason about

### What OpenSpec becomes now
OpenSpec becomes your change-management layer for:
- new features
- user-facing UX changes
- behavior changes
- domain rule updates
- important refactors
- cross-cutting architectural shifts

### What belongs in specs now
Specs should describe:
- current accepted user-visible behavior
- important business rules
- important domain expectations
- key workflow and state behavior

Specs should NOT become:
- a duplicate of code
- giant implementation notes
- a dumping ground for every idea
- a substitute for tickets or TODO lists

### What belongs in docs now
Keep `docs/` for:
- architecture overviews
- UI style guides
- technical reference docs
- diagrams and rationale
- onboarding material

### What belongs in changes now
Use `openspec/changes/` for:
- active proposed deltas from the current product baseline

---

## Recommended Long-Term Workflow

## 1. Use docs for big-picture reference
Keep:
- architecture docs
- UI style guide
- infra notes
- major design rationale

These help the AI think, but they are not the final accepted behavioral baseline.

## 2. Use specs for accepted behavior
Use `openspec/specs/` as the canonical source for:
- what the product currently does
- what users can expect
- what business rules are accepted

## 3. Use changes for active deltas
Every meaningful feature, refactor, or fix starts as a change.

## 4. Keep changes small
Smaller changes are easier to:
- review
- implement
- verify
- archive
- trust later

## 5. Verify before archive for important work
For anything non-trivial:
- compare implementation to specs
- confirm UI/UX behavior
- check important edge cases
- then archive

## 6. Archive quickly
Archive once the implementation and accepted behavior line up.
Do not leave old “active” changes hanging around forever.

---

## Suggested Phase-by-Phase Command Use

## Nothing → Early MVP
Use when the idea is not fully formed.

Good commands:
- `/opsx:explore`
- `/opsx:propose <change-name>`

Use this phase to:
- clarify the product
- shape MVP scope
- generate first artifacts

---

## Active MVP Build
Use when you are building the first real version.

Good commands:
- `/opsx:propose <change-name>`
- `/opsx:apply <change-name>`
- `/opsx:verify <change-name>` if available in your installed profile
- `/opsx:archive <change-name>`

Use this phase to:
- turn proposal/specs/design/tasks into implementation
- establish the initial accepted baseline

---

## Real Product / Ongoing Changes
Use when the product already exists and you are evolving it.

Good commands:
- `/opsx:propose <small-change>`
- `/opsx:new <small-change>`
- `/opsx:continue <small-change>`
- `/opsx:ff <small-change>`
- `/opsx:verify <small-change>`
- `/opsx:sync <small-change>`
- `/opsx:archive <small-change>`

Use this phase to:
- handle small features
- control refactors
- keep accepted behavior up to date

---

## Profile Recommendation

## For MVP
Use the default/core setup if you want the least friction.

That is usually enough for:
- one main MVP change
- solo development
- straightforward propose → apply → archive flow

## After MVP
Move to the expanded/custom command profile if you want:
- finer control over artifact generation
- explicit verification
- sync/preview workflows
- multiple concurrent changes
- a more granular product-change rhythm

---

## TimeTrak-Specific Recommendation

## Nothing
Use:
- `docs/time_tracking_design_doc.md`
- `docs/time_trak_mvp_ui_style_guide.md`
- `openspec/config.yaml`

Create:
- one MVP bootstrap change

## MVP
Likely accepted baseline domains:
- auth
- workspace
- clients
- projects
- tracking
- rates
- reporting

Use OpenSpec primarily to:
- shape scope
- guide implementation
- build a trustworthy starting baseline

## Real Product
After MVP, create small changes such as:
- `add-csv-export`
- `add-invoice-draft-generation`
- `support-project-archiving`
- `add-team-workspaces`
- `add-timesheet-approval-flow`
- `improve-report-filtering`
- `refactor-rate-resolution-service`

---

## Practical Rules That Keep OpenSpec Useful

### Rule 1
Do not let `docs/` and `openspec/specs/` fight each other.
Use docs for narrative reference.
Use specs for accepted behavior.

### Rule 2
Do not create giant umbrella changes after MVP.

### Rule 3
If a change meaningfully affects user behavior, domain rules, or major architecture, model it in OpenSpec.

### Rule 4
If a change is tiny and purely local, you may not need full ceremony.
Use judgment.

### Rule 5
Keep `config.yaml` concise.
Only include context and rules the AI truly needs repeatedly.

### Rule 6
The more mature the product gets, the more OpenSpec should feel like a lightweight delta system, not a wall of documentation.

---

## Simple Mental Model

### Nothing
“Help me figure out what I’m building.”

### MVP
“Help me define and build the first real version.”

### Real Product
“Help me manage continuous change without losing the plot.”

---

## Short Version

### Nothing → MVP
- keep product/design docs in `docs/`
- keep `config.yaml` concise
- create one serious MVP change
- implement it
- archive it into the baseline

### MVP → Real Product
- treat `openspec/specs/` as the source of truth
- create one change per meaningful feature/fix/refactor
- verify non-trivial work
- archive quickly
- keep docs as reference, not the canonical behavior layer

### Long-term
- docs = narrative reference
- specs = accepted behavior
- changes = active deltas

---

## Reference Notes

This workflow is aligned with current OpenSpec guidance that:
- OPSX is the standard workflow
- the system is fluid and iterative rather than phase-locked
- `openspec/specs/` is the source of truth
- `openspec/config.yaml` injects context and per-artifact rules into planning
- command availability depends on profile, with `core` and expanded/custom options
