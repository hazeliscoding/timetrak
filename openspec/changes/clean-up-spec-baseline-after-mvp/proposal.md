## Why

The accepted spec baseline under `openspec/specs/` was materialized from a string of archived MVP and Stage 2 changes. Ten of the eleven spec files still carry placeholder Purpose prose (`TBD - created by archiving change …`), and `openspec/config.yaml` still enumerates only the original MVP domains — the four UI capabilities promoted in Stage 2 (`ui-foundation`, `ui-partials`, `ui-showcase`, `ui-browser-tests`) are absent. The baseline is the contract future changes are measured against, so the TBD residue and the stale domain list create avoidable friction for every subsequent proposal and review. Stage 2 is the right moment to fix it, before more changes layer on top.

## What Changes

- Replace the `TBD …` Purpose placeholder in each of the ten affected spec files (`auth`, `clients`, `projects`, `rates`, `reporting`, `tracking`, `ui-browser-tests`, `ui-foundation`, `ui-partials`, `ui-showcase`) with a real, implementation-agnostic Purpose statement derived from the requirements already present in that spec.
- Extend the two domain enumerations in `openspec/config.yaml` (the `context.Accepted domains` line and the `rules.specs` "Organize specs by domain" rule) to include `ui-foundation`, `ui-partials`, `ui-showcase`, and `ui-browser-tests` so the accepted-domain list matches what is actually under `openspec/specs/`.
- No requirements are added, removed, or modified. No code, migrations, templates, HTMX wiring, or tests change. This is a documentation-and-config hygiene change.
- Out of scope: rewording existing requirements, merging or splitting spec files, adjusting scenarios, revising the Stage 2 roadmap, or touching any archived change under `openspec/changes/archive/`.

## Capabilities

### New Capabilities

- _None._ This change does not introduce a new capability.

### Modified Capabilities

- `auth`: rewrite Purpose prose (no requirement changes).
- `clients`: rewrite Purpose prose (no requirement changes).
- `projects`: rewrite Purpose prose (no requirement changes).
- `rates`: rewrite Purpose prose (no requirement changes).
- `reporting`: rewrite Purpose prose (no requirement changes).
- `tracking`: rewrite Purpose prose (no requirement changes).
- `ui-browser-tests`: rewrite Purpose prose (no requirement changes).
- `ui-foundation`: rewrite Purpose prose (no requirement changes).
- `ui-partials`: rewrite Purpose prose (no requirement changes).
- `ui-showcase`: rewrite Purpose prose (no requirement changes).

## Impact

- **Files:** `openspec/specs/<domain>/spec.md` for each of the ten domains above; `openspec/config.yaml`.
- **Code / runtime:** none. No Go packages, templates, static assets, migrations, or tests are touched.
- **Tooling:** `openspec validate` should continue to pass before and after; CI behavior is unchanged.
- **Follow-ups:** future proposals can rely on the accepted-domain list and real Purpose prose without needing to re-discover the history from archived changes.
- **Risks:** low — edits are confined to prose and a config enumeration. The main risk is writing a Purpose that subtly contradicts existing requirements; mitigated by deriving each Purpose from the spec's own requirement text and keeping it intentionally short.
- **Assumptions:** Purpose prose is not a normative requirement and editing it in-place does not require delta spec files under `changes/<name>/specs/`. If the `openspec` tooling disagrees at validate time, the fallback is to add minimal delta stubs that carry only the Purpose rewrite.
