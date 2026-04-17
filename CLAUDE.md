# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Status

TimeTrak is **pre-code**. No application source exists yet — the repo currently contains only planning material:

- `docs/` — narrative design and UI reference docs
- `openspec/` — spec-driven change workflow (OpenSpec / OPSX)
- `.claude/` — slash commands and skills for the OpenSpec workflow

Treat this as Stage 0/Stage 1 of the flow described in `docs/openspec_flow_nothing_to_mvp_to_real_product.md`: the first real unit of work should be a single MVP bootstrap change under `openspec/changes/`, not speculative files in `openspec/specs/` or production code without a change backing it.

## Stack (Target)

- **Backend:** Go (modular monolith)
- **UI:** Go HTML templates rendered server-side + HTMX for partial updates; minimal custom JS
- **Database:** PostgreSQL (system of record)
- **Auth boundary:** Workspace (all authorization scopes to workspace)

Do not introduce SPA frameworks, client-side state libraries, or ORMs that fight the server-rendered + HTMX model without an explicit change proposal.

## Domain Model

Main hierarchy: **Workspace → Client → Project → Time Entry**. Rate resolution precedence: **project rate → client rate → workspace default**. Accepted MVP domains are `auth`, `workspace`, `clients`, `projects`, `tracking`, `rates`, `reporting` (see `openspec/config.yaml`).

## Data Conventions (binding)

- UUID primary keys
- `timestamptz` for all persisted timestamps
- Money stored as **integer minor units** — never floats
- Transactional tables normalized to 3NF unless there is a documented read-model reason

## Workflow: OpenSpec is the Source of Truth

This project uses OpenSpec (`/opsx:*` / `/openspec-*` skills). The canonical flow:

1. `openspec/specs/` = accepted behavior (source of truth once MVP lands)
2. `openspec/changes/<name>/` = active proposed deltas with proposal, specs, design, tasks
3. `docs/` = long-form narrative reference; **not** the behavioral baseline

Rules enforced by `openspec/config.yaml`:

- Proposals: MVP-first scope, explicit in/out of scope, call out assumptions and risks
- Specs: MUST/SHALL language, GIVEN/WHEN/THEN scenarios, organized by domain, include empty/loading/success/error/destructive states and accessibility requirements when UI is involved
- Design docs: include Mermaid diagrams when relevant; reflect Go + HTMX + PostgreSQL constraints; explain tradeoffs
- Tasks: small and implementation-ready; group by backend / database / templates / HTMX; include accessibility validation tasks for UI work

After MVP: **one change per meaningful unit of work** (e.g. `add-csv-export`), never umbrella changes like `phase-2` or `misc-cleanup`. Archive changes quickly once implemented.

### Useful slash commands

- `/opsx:explore` — think through a fuzzy idea before proposing
- `/opsx:propose <change-name>` — create a change with proposal/specs/design/tasks
- `/opsx:apply <change-name>` — implement from `tasks.md`
- `/opsx:archive <change-name>` — move completed change into baseline

## UI Direction (binding for any template work)

- Calm, trustworthy, tool-like; data-first, medium-density layouts; tables are first-class
- One restrained accent color, strong neutral system; prefer borders + spacing over heavy shadows
- Avoid generic AI-SaaS visual language: no blob art, no oversized hero sections, no random gradients, no vague productivity copy
- Use domain-specific copy (`Start timer`, `Billable this week`, `Client rate`, `Running entry`)
- Server-rendered flows over SPA patterns; HTMX for timers, inline edits, filtering, pagination, modals
- Preserve sensible focus behavior after HTMX swaps; prefer native controls before custom widgets

## Accessibility (binding)

Target **WCAG 2.2 AA**. Visible labels, visible keyboard focus, sufficient contrast, comfortable target sizes. Color must never be the sole means of conveying status. Include accessibility validation tasks in any UI-affecting change.

## Build / Test Commands

None yet — the Go project has not been scaffolded. When adding the first Go code (as part of the MVP bootstrap change), add the build/test/lint commands to this file.
