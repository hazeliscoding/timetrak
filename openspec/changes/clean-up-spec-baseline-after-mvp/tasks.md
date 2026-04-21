## 1. Baseline survey

- [x] 1.1 Run `grep -n '^TBD' openspec/specs/*/spec.md` and confirm exactly 10 files match (`auth`, `clients`, `projects`, `rates`, `reporting`, `tracking`, `ui-browser-tests`, `ui-foundation`, `ui-partials`, `ui-showcase`). Abort and re-scope if `workspace` also matches or if new spec directories exist that aren't listed in the proposal.
- [x] 1.2 Run `openspec validate` on a clean working tree and capture the baseline output (warnings + exit code) so later steps can confirm no new warnings are introduced.
- [x] 1.3 Read `openspec/specs/workspace/spec.md`'s existing Purpose paragraph and use it as the stylistic reference for the rewrites: short (2–4 sentences), implementation-agnostic, grounded in what the Requirements in the same file actually cover.

## 2. Rewrite Purpose prose in accepted specs

Each task below edits only the `## Purpose` section (lines 3–4) of the named file. Do not touch `## Requirements`, any `### Requirement:` block, or any `#### Scenario:` block. Derive the Purpose from the Requirements already present in that same file — do not import wording from `docs/`.

- [x] 2.1 Rewrite Purpose in `openspec/specs/auth/spec.md` (identity, sessions, password hashing, rate limiting, CSRF posture).
- [x] 2.2 Rewrite Purpose in `openspec/specs/clients/spec.md` (workspace-scoped client CRUD and lifecycle).
- [x] 2.3 Rewrite Purpose in `openspec/specs/projects/spec.md` (workspace-scoped project CRUD under a client, archival rules).
- [x] 2.4 Rewrite Purpose in `openspec/specs/rates/spec.md` (rate rule storage and `Resolve` precedence: project → client → workspace default; integer minor units).
- [x] 2.5 Rewrite Purpose in `openspec/specs/reporting/spec.md` (read-model behavior, snapshot-only totals for closed entries, filter semantics).
- [x] 2.6 Rewrite Purpose in `openspec/specs/tracking/spec.md` (timer start/stop, entry CRUD, active-timer invariant, rate snapshot at stop/save).
- [x] 2.7 Rewrite Purpose in `openspec/specs/ui-browser-tests/spec.md` (build-tag-gated browser contract harness, shared server + testdb fixtures).
- [x] 2.8 Rewrite Purpose in `openspec/specs/ui-foundation/spec.md` (two-layer token taxonomy and the public semantic alias contract).
- [x] 2.9 Rewrite Purpose in `openspec/specs/ui-partials/spec.md` (canonical partial location, naming, slot convention, HTMX event contract).
- [x] 2.10 Rewrite Purpose in `openspec/specs/ui-showcase/spec.md` (dev-only `/dev/showcase` surface, auth requirement, non-registration in prod).

## 3. Update config enumerations

- [x] 3.1 In `openspec/config.yaml`, update the `context` line `- Accepted domains: auth, workspace, clients, projects, tracking, rates, reporting` to enumerate the full current set: `auth, clients, projects, rates, reporting, tracking, ui-browser-tests, ui-foundation, ui-partials, ui-showcase, workspace` (alphabetical, matches directory listing under `openspec/specs/`).
- [x] 3.2 In the same file, update the `rules.specs` entry that reads `Organize specs by domain: auth, workspace, clients, projects, tracking, rates, reporting.` to the same expanded list.
- [x] 3.3 Confirm no other file under version control enumerates the domain list (`grep -rn 'Accepted domains' openspec/ docs/`). If other occurrences exist in documentation, note them for a follow-up doc change but do NOT edit them here — this change is scoped to `openspec/`.

## 4. Validate and self-check

- [x] 4.1 Run `openspec validate` and confirm exit code 0 with no new warnings compared to the baseline captured in 1.2.
- [x] 4.2 Re-run `grep -n '^TBD' openspec/specs/*/spec.md` and confirm zero matches remain.
- [x] 4.3 Run `openspec status --change clean-up-spec-baseline-after-mvp` and confirm it still reports all artifacts `done`.
- [x] 4.4 For each rewritten Purpose, sanity-check it against the spec's own `### Requirement:` headers in the same file. If any sentence in Purpose could be read as contradicting a requirement (e.g. Purpose says "billable only" but a requirement covers non-billable entries), revise Purpose — never edit the requirement.
- [x] 4.5 Fallback — only if `openspec validate` rejects any of the no-op delta files under `openspec/changes/clean-up-spec-baseline-after-mvp/specs/<cap>/spec.md`: copy the first `### Requirement:` block (heading + body + all `#### Scenario:` blocks) from the corresponding baseline spec verbatim under this delta's `## MODIFIED Requirements` heading, with zero edits to the copied text. This makes the delta a semantic no-op that satisfies the validator. Repeat per failing capability only.

## 5. Commit and archive

- [ ] 5.1 Use the `tt-conventional-commit` skill to commit the spec-prose edits and config edits together with a single `docs(openspec): …` or `chore(openspec): …` message (do NOT include a Claude Code attribution trailer, per repo convention).
- [ ] 5.2 Once merged, run `/opsx:archive clean-up-spec-baseline-after-mvp` and, in the same commit, hand-edit any Purpose placeholder the archive tool might re-insert back to the rewritten text so the baseline does not regress.
