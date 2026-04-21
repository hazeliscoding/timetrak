## Context

TimeTrak finished MVP bootstrap (archived 2026-04-17) and has since landed a cluster of Stage 2 changes — authorization hardening, timer integrity, rate resolution snapshotting, reporting correctness, UI polish, and the component library foundation (token taxonomy, reusable partials, showcase surface, browser contract tests). Each archived change used `openspec archive` to materialize or update a spec under `openspec/specs/<domain>/spec.md`, which leaves a placeholder `TBD — update Purpose after archive` line above the real Requirements block.

The side-effects today:

- Ten of the eleven accepted specs still start with a TBD placeholder. Only `openspec/specs/workspace/spec.md` has real Purpose prose.
- `openspec/config.yaml` enumerates the accepted domains as `auth, workspace, clients, projects, tracking, rates, reporting` in two places (the `context` block and the `rules.specs` entry). The four UI capabilities that were promoted to `openspec/specs/` in Stage 2 — `ui-foundation`, `ui-partials`, `ui-showcase`, `ui-browser-tests` — are not listed, so new proposals relying on that enumeration would be told those domains don't exist.
- `openspec/config.yaml` also has a latent `Rules for 'specs' must be an array of strings, ignoring this artifact's rules` warning that the CLI prints on every command. That is a separate config-shape issue and is explicitly out of scope here — it would be its own proposal.

The Requirements bodies themselves are in good shape — they come straight from the archived proposals and already use SHALL/MUST language and GIVEN/WHEN/THEN scenarios. This change is therefore purely a baseline hygiene edit: prose + config.

## Goals / Non-Goals

**Goals:**

- Every file under `openspec/specs/*/spec.md` has an accurate, implementation-agnostic Purpose paragraph that summarizes what the capability governs, without restating the requirements.
- `openspec/config.yaml`'s domain enumerations match the set of directories actually present under `openspec/specs/`.
- `openspec validate` continues to pass before and after the edits, with no new warnings.
- Future proposals can point at the baseline as the source of truth without needing to cross-reference archived changes to understand what each domain covers.

**Non-Goals:**

- Editing, consolidating, splitting, reordering, or renaming any existing Requirement or Scenario.
- Touching any archived change under `openspec/changes/archive/`.
- Rewriting `docs/timetrak_post_mvp_openspec_roadmap.md` or any other narrative doc.
- Fixing the unrelated `Rules for 'specs' must be an array of strings` CLI warning — defer to a separate change.
- Changing any Go source, migrations, templates, static assets, or tests.

## Decisions

**Decision 1: Rewrite Purpose prose in place rather than routing through spec deltas.**

The OpenSpec delta grammar (`## ADDED`, `## MODIFIED`, `## REMOVED`, `## RENAMED`) operates on Requirements, not on the `## Purpose` section, which is non-normative metadata. A full MODIFIED block would require copying the entire requirement body only to change surrounding prose, which would churn every scenario for no semantic reason and make the eventual archive diff noisy.

_Alternative considered:_ Author each Purpose rewrite as a MODIFIED requirement delta. Rejected because it overloads MODIFIED (which the schema reserves for behavior changes) and doubles the surface area of the change with no added safety.

_Implication:_ the change's own `specs/**/*.md` delta files will be intentionally minimal — each will state "no requirement changes" and point at the Purpose edit. This is a deliberate baseline-hygiene pattern and should be reused for any future prose-only spec cleanup.

**Decision 2: Derive each Purpose from the spec's own Requirements, not from external docs.**

Purposes will be composed by reading the Requirements already in the target spec and summarizing their shared intent in 2–4 sentences. This guarantees the Purpose cannot contradict the normative body and keeps the spec file self-contained.

_Alternative considered:_ Write Purposes from `docs/time_tracking_design_doc.md` and `docs/timetrak_ui_style_guide.md`. Rejected because those docs are narrative and evolve independently; anchoring Purpose to them risks drift. Docs remain the reference for readers, but the spec stays the source of truth.

**Decision 3: Update both domain-enumeration sites in `openspec/config.yaml` in one edit.**

The enumeration appears in two places — the `context` block (one line: `Accepted domains: …`) and the `rules.specs` array (one line: `Organize specs by domain: …`). Both will be updated to the same expanded list in the same commit so they cannot drift again. The order will follow the directory listing under `openspec/specs/` for easy mechanical verification: `auth, clients, projects, rates, reporting, tracking, ui-browser-tests, ui-foundation, ui-partials, ui-showcase, workspace`.

_Alternative considered:_ Introduce a structured list (YAML array) instead of a comma-joined sentence. Rejected as out of scope — it is a schema change to the config, not a hygiene fix, and would need its own proposal.

**Decision 4: Leave the `openspec-archive` tooling's TBD-insertion behavior alone.**

The TBD placeholder is inserted by the `openspec archive` command itself when a capability is first materialized. Fixing the tool so it never writes TBD would prevent the problem recurring, but that is a change to external tooling, not to TimeTrak. The authoring convention this change establishes is: whoever runs `/opsx:archive` is responsible for immediately rewriting the Purpose as part of that archive commit. That convention will be surfaced in the `/opsx:archive` skill's follow-up, not enforced here.

## Risks / Trade-offs

- **[Risk]** A written Purpose drifts subtly from a Requirement (e.g. says "tracks billable minutes" when the spec also covers non-billable entries). **Mitigation:** derive every Purpose from the spec's own requirement text and keep each Purpose to 2–4 sentences; reviewer sanity-checks against the requirement headers, not external docs.
- **[Risk]** `openspec validate` rejects the minimal per-capability delta files because they contain no ADDED/MODIFIED/REMOVED sections. **Mitigation:** if validate fails, fall back to single-scenario MODIFIED stubs that copy the first requirement verbatim (no semantic change) and update the Purpose in the same commit; rerun validate before merging. This fallback is documented in `tasks.md`.
- **[Risk]** The config enumeration expansion is interpreted as a claim that `ui-*` domains are first-class product domains on equal footing with `auth` / `tracking` / etc., implying they deserve the same ongoing stewardship. **Mitigation:** keep the `context` line flat but order product domains first and `ui-*` last; no narrative claim beyond "these spec folders exist".
- **[Trade-off]** Purpose prose is editorial by nature; two reviewers could reasonably word it differently. We accept that — the bar is "accurate and not contradictory", not "perfectly phrased".
- **[Trade-off]** Doing this as its own change (rather than bundling with the next feature proposal) costs one extra archive cycle. Worth it: baseline hygiene has no implementation risk and should not be entangled with behavior changes.
