## MODIFIED Requirements

<!--
  NO-OP DELTA. This change is prose-only: it rewrites the non-normative
  `## Purpose` section of `openspec/specs/ui-browser-tests/spec.md` and does NOT
  alter any requirement. The requirement block below is copied verbatim
  from the baseline so the OpenSpec validator recognises a MODIFIED delta.
  See proposal.md and design.md (Decision 1).
-->

### Requirement: Browser-driven test harness MUST be gated behind a build tag

The repository SHALL provide a browser-driven UI contract test harness under `internal/e2e/browser/`, with every file carrying the `//go:build browser` build constraint. The default `go test ./...` and `make test` commands MUST NOT compile or execute these tests. A dedicated `make test-browser` target SHALL exist to run them, and a `make browser-install` target SHALL install the required browser binaries on demand.

#### Scenario: Default test run skips browser tests
- **WHEN** a developer runs `make test` or `go test ./...`
- **THEN** no files under `internal/e2e/browser/` are compiled or executed
- **AND** the test run remains hermetic with respect to browser binaries

#### Scenario: Opt-in browser test run
- **WHEN** a developer runs `make test-browser` after `make browser-install`
- **THEN** the browser-tagged tests under `internal/e2e/browser/...` are executed against a locally launched server

#### Scenario: Graceful skip when browser binaries are absent
- **WHEN** `make test-browser` is run without Playwright browser binaries installed
- **THEN** each browser test SHALL skip with a message instructing the developer to run `make browser-install`
- **AND** the test suite MUST NOT fail due to the missing binaries
