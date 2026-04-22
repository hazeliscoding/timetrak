## 1. Pre-flight

- [x] 1.1 Run `openspec validate --all` on a clean working tree and confirm the baseline is 11/11 pass, exit 0 with no new warnings. Capture the output so task 4.1 can compare against it.
- [x] 1.2 Run `ls openspec/changes/` and confirm only `archive/` plus this change's directory (`transition-to-stage-3/`) exist. If any other active change is present, pause and decide whether to land it first — Stage 3 framing should not ship on top of unresolved Stage 2 work.
- [x] 1.3 Read `docs/openspec_flow_nothing_to_mvp_to_real_product.md#stage-3--real-product` and use its description of Stage 3 as the source for the operating-posture wording in all edits below. Do not invent Stage 3 concepts that aren't in that doc.

## 2. Update CLAUDE.md

All edits in this group are scoped to the top of `CLAUDE.md` — specifically the `## Project Status` and `## Domain Model` sections. Do not touch any other section.

- [x] 2.1 Rewrite `CLAUDE.md:7` (the "TimeTrak is **post-MVP — Stage 2 (Stabilize)**..." sentence) to declare Stage 3 as the current operating posture. Keep it one sentence and keep the existing reference to the archived bootstrap change.
- [x] 2.2 Replace `CLAUDE.md:9` (the "Active work follows the Stage 2 roadmap..." paragraph with its broken `docs/timetrak_post_mvp_openspec_roadmap.md` reference) with a paragraph describing Stage 3 operating posture (per-change evolution, archive often, specs are the baseline) and linking to the new `docs/timetrak_stage_3_roadmap.md` created in task 3.1. The paragraph MUST explicitly state that the roadmap is orientation, not a commitment.
- [x] 2.3 Rewrite `CLAUDE.md:24` (the "Accepted MVP domains are `auth`, ..." sentence) to stop enumerating domains inline and instead point at `openspec/config.yaml` as the source of truth plus `ls openspec/specs/` for a live list. Do not re-enumerate the eleven current domains here — removing the inline list is the point (see design.md Decision 2).
- [x] 2.4 Leave `CLAUDE.md:11` (the one-change-per-unit rule paragraph) unchanged — it is already Stage-3-compatible.
- [x] 2.5 Confirm no other sentence in `CLAUDE.md` says "Stage 2" or "Stabilize" (`grep -nE 'Stage 2|Stabilize' CLAUDE.md`). If any remain, revise to Stage 3 or delete.

## 3. Create the Stage 3 roadmap doc

- [x] 3.1 Create `docs/timetrak_stage_3_roadmap.md`. Structure: (a) a bold preamble stating "This is orientation, not a plan. Every candidate MUST go through `/opsx:propose` before implementation. Candidates may be removed, re-ordered, or abandoned at any time." (b) a "Likely next" section with short one-paragraph sketches for CSV export, invoice generation from rate snapshots, and team workspaces with non-owner roles; (c) a "Later / exploratory" section with one-paragraph sketches for native mobile or PWA timer, email/webhook notifications, audit log, and competitor-tool data import; (d) a closing note on how candidates graduate (via `/opsx:propose`) and how this doc is kept current (shrinks as candidates become real changes or are abandoned).
- [x] 3.2 Do not include deadlines, owner names, sequencing claims ("do X before Y"), or effort estimates. Candidates are orientation only (design.md Decision 3).
- [x] 3.3 Confirm the doc does not duplicate `openspec/specs/` content or create requirements. It references domains by name but MUST NOT describe accepted behavior.

## 4. Update openspec/config.yaml context

- [x] 4.1 In `openspec/config.yaml` lines 6–8, rewrite the `Stage:` paragraph from Stage 2 language ("Post-MVP — Stage 2 (Stabilize). ... hardening authorization, timer integrity, rate resolution, reporting correctness, UI polish, accessibility compliance, and establishing the component library foundation.") to Stage 3 language describing the operating posture (per-change delta against the accepted baseline, new features and UX changes shepherded via `/opsx:propose`, specs kept as the source of truth, archive often). Wording MUST align with the flow doc section referenced in 1.3.
- [x] 4.2 Do not touch `openspec/config.yaml` outside of the specific lines enumerated in this group. Specifically out of scope: the `Accepted domains:` line (line 20, fixed in the previous change), the `context.Documentation` block, and every bullet in the `rules` block **except** line 67 (see 4.4).
- [x] 4.3 Verify `openspec validate --all` still passes after the config edit.
- [x] 4.4 **Amendment (2026-04-22, per design.md Decision 5).** Edit `openspec/config.yaml:43` — rewrite the UI-direction bullet "Component library foundation is being established in Stage 2 — prefer reusable partials over bespoke markup" to reflect the current state: the foundation, partials, and showcase are archived baseline capabilities, so the bullet should read as present-tense guidance ("prefer reusable partials from the component library foundation over bespoke markup" or similar). Do not reference "Stage 2" or "Stabilize" in the rewrite.
- [x] 4.5 **Amendment (2026-04-22, per design.md Decision 5).** Delete `openspec/config.yaml:67` — the "Stage 2 changes should harden, polish, or stabilize existing behavior — not expand the domain model." bullet. The adjacent line 68 ("Stage 3 changes may introduce new capability…") already carries the live guidance; do not add a replacement "Stage 3 …" restriction line, which would just recreate the same drift class.

## 5. Validate and self-check

- [x] 5.1 Run `openspec validate --all`. Expect 11/11 pass and no new warnings compared to 1.1. The `Rules for 'specs' must be an array of strings` warning IS pre-existing; any *other* new warning is a regression to investigate.
- [x] 5.2 Run `openspec status --change transition-to-stage-3` and confirm all four artifacts report `done`.
- [x] 5.3 Re-run `grep -nE 'Stage 2|Stabilize' CLAUDE.md openspec/config.yaml`. Expected matches: zero. The phrase may still appear in `openspec/changes/archive/**` and in **this change's own** `proposal.md` / `design.md` / `tasks.md` (both are historical record describing the transition); those MUST NOT be edited.
- [x] 5.4 Re-run `grep -rn 'timetrak_post_mvp_openspec_roadmap' CLAUDE.md docs/ openspec/specs/ openspec/config.yaml` and confirm zero matches — the dangling reference is gone from every live surface. Matches inside this change's own `proposal.md` / `design.md` / `tasks.md` and inside `openspec/changes/archive/**` are expected (they describe what this change fixes) and MUST NOT be edited.
- [x] 5.5 Sanity-check the Stage 3 roadmap doc against `CLAUDE.md`: every candidate domain named in the roadmap (e.g. `tracking`, `rates`, `reporting`) MUST already exist as a capability under `openspec/specs/`, or be explicitly framed as a candidate for a new capability.

## 6. Commit and archive

- [ ] 6.1 Use the `tt-conventional-commit` skill to commit all edits together as `docs(openspec): transition repo framing to Stage 3` (or `docs(meta): ...` if preferred). Single commit. No Claude Code attribution trailer.
- [ ] 6.2 Once merged, run `/opsx:archive transition-to-stage-3`. The archive move should be a pure `git mv`; if the archive tool re-inserts any TBD placeholder into `openspec/specs/workspace/spec.md`, revert it in the same archive commit.
