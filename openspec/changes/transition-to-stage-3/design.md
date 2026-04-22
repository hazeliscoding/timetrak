## Context

The TimeTrak repo currently describes itself in three overlapping surfaces:

1. `CLAUDE.md` — canonical contributor guidance, loaded into every Claude Code session as project instructions.
2. `openspec/config.yaml` — the `context` block is loaded into every OpenSpec artifact's `instructions` output.
3. `docs/` — longer-form narrative reference (`time_tracking_design_doc.md`, `timetrak_ui_style_guide.md`, `openspec_flow_nothing_to_mvp_to_real_product.md`).

All three still describe a Stage 2 project: `CLAUDE.md:7` declares "Post-MVP — Stage 2 (Stabilize)", `config.yaml:6–8` lists Stage 2 priorities, `CLAUDE.md:9` points at a roadmap file (`docs/timetrak_post_mvp_openspec_roadmap.md`) that isn't in the tree, and `CLAUDE.md:24` enumerates seven "Accepted MVP domains" even though the baseline now has eleven (the four `ui-*` capabilities were promoted in Stage 2).

The Stage 2 roadmap — as reconstructed from `openspec/changes/archive/` commit history — has fully landed. Archived changes include: bootstrap MVP (2026-04-17), harden workspace authorization, improve timer concurrency, stabilize rate resolution history, tighten reporting snapshot-only, add rates HTMX partials, improve reporting correctness and filters, polish MVP UI, establish component library foundation, create reusable UI partials, create component library showcase, add browser UI contract tests, and the just-landed spec baseline cleanup. There is no outstanding Stage 2 hardening item identified anywhere in `openspec/` or `docs/`.

Per `docs/openspec_flow_nothing_to_mvp_to_real_product.md#stage-3--real-product`, Stage 3 is an operating posture rather than a set of features: OpenSpec becomes the change-management layer for new features, UX changes, domain rule updates, and cross-cutting refactors, while `docs/` holds architecture and UX reference. That same doc is what this change aligns the repo's self-description with.

## Goals / Non-Goals

**Goals:**

- `CLAUDE.md` and `openspec/config.yaml` describe Stage 3 as the current operating posture and no longer mention Stage 2 priorities as live.
- The dangling `docs/timetrak_post_mvp_openspec_roadmap.md` reference is replaced by a real file that orients a new contributor on candidate Stage 3 work without committing to it.
- The "Accepted MVP domains" line in `CLAUDE.md` either stops enumerating domains inline, or enumerates the current eleven — whichever is harder to drift from.
- `openspec validate` continues to pass for both the baseline and this change with no new warnings.
- Contributors can run `/opsx:propose <name>` for a Stage 3 feature immediately after this lands, with no further framing work needed.

**Non-Goals:**

- Writing any actual Stage 3 feature proposal (CSV export, invoices, teams, mobile, notifications, etc.). Those are independent changes.
- Deciding Stage 3 ordering or priority beyond "likely-next vs later" orientation.
- Touching `openspec/specs/` — no requirement is added, removed, or modified.
- Editing Go source, migrations, templates, static assets, or tests.
- Renaming or re-homing `openspec/specs/` capabilities.
- Fixing the latent `Rules for 'specs' must be an array of strings` OpenSpec CLI warning — separate hygiene change, already scoped out in the previous cleanup.
- Creating a formal backlog or ticket system in-repo. Tickets belong in whatever tracker you use, not under `docs/` or `openspec/`.

## Decisions

**Decision 1: Treat Stage 3 as an operating posture in prose, not as a new spec capability.**

Stage 3 is not a product capability, it is a phase label. Creating a `stage-3/` folder under `openspec/specs/` would be category-error — specs describe accepted user-visible behavior, not project management state. The change therefore edits narrative surfaces (`CLAUDE.md`, `docs/`, config `context`) and leaves `openspec/specs/` untouched.

_Alternative considered:_ Add a `meta/` or `project-state/` spec capability that encodes phase history as requirements. Rejected because it conflates phase metadata with normative product behavior; the flow doc already warns against turning specs into "a dumping ground for every idea".

**Decision 2: Remove the inline "Accepted MVP domains" enumeration from `CLAUDE.md` and point at `openspec/config.yaml` instead.**

The `CLAUDE.md:24` line currently duplicates the domain list from `config.yaml:20`. We just fixed the config line in the previous change; we'd rather not have two sources of truth to keep in sync. `CLAUDE.md` should defer to `openspec/config.yaml` for the enumeration and describe only the relationship (e.g. "accepted baseline capabilities are enumerated in `openspec/config.yaml`").

_Alternative considered:_ Refresh the inline list to the current eleven domains. Rejected because it will drift again the next time a capability is added or retired; removing the inline list kills the drift class entirely.

**Decision 3: The Stage 3 roadmap doc is narrative orientation, not a commitment or backlog.**

`docs/timetrak_stage_3_roadmap.md` will enumerate 3–5 candidate Stage 3 initiatives at one-paragraph depth each, labelled "likely-next" and "later / exploratory". The doc's preamble MUST state explicitly that it is not a plan, not a commitment, and not a replacement for per-change proposals. Each candidate must still go through `/opsx:propose` before any implementation begins.

Candidate shortlist (for the roadmap doc itself to present — order is not a commitment):

- _Likely-next:_ CSV export for time entries and reporting; invoice generation from rate snapshots; team workspaces (multi-member, non-owner roles).
- _Later / exploratory:_ native mobile or PWA timer; email/webhook notifications; audit log; data import from competitor tools.

_Alternative considered:_ Embed the roadmap directly in `CLAUDE.md`. Rejected because `CLAUDE.md` should stay short and binding, not carry exploratory prose; the flow doc also says narrative reference lives in `docs/`.

**Decision 4: Reuse the no-op verbatim-copy delta pattern from the spec baseline cleanup.**

The change has no requirement deltas, but `openspec validate` requires at least one parseable delta. We will create `specs/<capability>/spec.md` no-op deltas — each a verbatim copy of the first `### Requirement:` block from the corresponding baseline spec under a `## MODIFIED Requirements` header, with a comment explaining the no-op nature. This is exactly the pattern documented in `openspec/changes/archive/2026-04-21-clean-up-spec-baseline-after-mvp/design.md` Decision 1 and proven to pass the validator without altering baseline behavior.

The delta files will cover only as many capabilities as the validator requires (minimum one). We will create exactly one, under `specs/workspace/` — `workspace` is the oldest, most stable capability and the least likely to churn while this change is in-flight.

_Alternative considered:_ Create deltas for all 11 capabilities as in the cleanup change. Rejected because this change does not edit any baseline `spec.md` file, so pairing every capability with a no-op is ceremony. One delta is enough to satisfy the validator.

**Decision 5 (amendment, 2026-04-22): edit two additional Stage-2-phrase lines in `openspec/config.yaml` beyond the `context.Stage` paragraph.**

Original task 4.2 forbade touching any part of `openspec/config.yaml` except the `context.Stage` paragraph. Implementation surfaced two concrete Stage-2-phrase residues outside that paragraph, both inconsistent with the Stage 3 framing this change is landing:

- `openspec/config.yaml:43` (UI-direction bullet) — "Component library foundation is being established in Stage 2 — prefer reusable partials over bespoke markup." Now factually stale: the foundation, partials, and showcase are all archived baseline capabilities.
- `openspec/config.yaml:67` (rules.proposal bullet) — "Stage 2 changes should harden, polish, or stabilize existing behavior — not expand the domain model." Redundant with and contradicted by the adjacent line 68 "Stage 3 changes may introduce new capability…"

Keeping these intact would leave visible Stage-2 framing in the live config while claiming the project is in Stage 3, which is exactly the self-description drift this change exists to kill. The amendment permits editing these two lines (and only these two) in `openspec/config.yaml`. Line 43 is rewritten to describe the current state (foundation exists, partials preferred over bespoke markup). Line 67 is deleted — line 68 now carries the live Stage 3 guidance and adding a matching Stage-3-restriction line would re-introduce the same drift risk we're fixing.

_Alternative considered:_ Leave lines 43 and 67 alone and scope the 5.3 grep to the `Stage:` paragraph only, logging the residue as follow-up. Rejected because two documented-stale lines in a file that every Claude Code session reads as ground truth is not a follow-up — it's the failure mode this change exists to prevent.

The `Accepted domains:` line (line 20), the `context.Documentation` block, and the rest of the `rules` block are still out of scope; the amendment is narrowly scoped to lines 43 and 67.

## Risks / Trade-offs

- **[Risk]** The Stage 3 roadmap doc is read as a plan and someone builds a candidate without running `/opsx:propose`. **Mitigation:** the doc leads with a bold "This is orientation, not a plan — every candidate MUST go through `/opsx:propose` before implementation" preamble; `CLAUDE.md` echoes the same in the paragraph that links to the doc.
- **[Risk]** Removing the inline `CLAUDE.md:24` domain enumeration loses quick-scan visibility into which capabilities exist. **Mitigation:** replace it with a one-liner like "Accepted baseline capabilities are enumerated in `openspec/config.yaml`; run `ls openspec/specs/` for a live list." The cost of the extra command is tiny compared to the drift cost of maintaining a second copy.
- **[Risk]** A single-capability no-op delta under `specs/workspace/` tricks a future reviewer into thinking this change modifies workspace behavior. **Mitigation:** the delta file carries the same explicit "NO-OP DELTA" HTML-comment header used by the cleanup change, pointing at this `design.md` Decision 4 for rationale.
- **[Risk]** The new roadmap doc's candidate list dates quickly. **Mitigation:** the doc header notes its snapshot nature and that candidates are confirmed or killed via `/opsx:propose`; any candidate that becomes a real change is removed from the roadmap at archive time, and any candidate abandoned is removed in whichever change happens to touch the doc next. The doc is allowed to shrink over time.
- **[Trade-off]** Introducing a new doc adds a file the team must remember to update. Accepted because the alternative (no forward-looking surface at all) leaves a dangling `CLAUDE.md` reference we're forced to either delete or point somewhere.
