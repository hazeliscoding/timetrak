---
name: tt-run-tests
description: Run targeted Go tests, lint, vet, and the full integration suite for TimeTrak. Use when implementing or finishing a task that needs verification, or when triaging a flaky/failing test.
license: MIT
compatibility: TimeTrak. make + go + a running Postgres for integration tests.
metadata:
  author: timetrak
  version: "1.0"
---

Run the right test command for the task at hand. Designed for tight feedback loops while implementing tasks from `tasks.md` and for the final pre-commit verification pass.

**Quick reference**

| Goal | Command |
| --- | --- |
| One test in one package, fastest loop | `go test ./internal/<domain>/... -run TestThing -count=1` |
| One package, all tests | `go test ./internal/<domain>/... -count=1` |
| Full integration suite (serial, binding) | `make test` |
| Vet | `make vet` |
| Format check | `make fmt` then `git diff --quiet` |
| Build everything | `make build` (produces `bin/web` + `bin/migrate`) |
| Re-apply latest migration | `make migrate-redo` |
| Boot Postgres | `make db-up` |

**Steps**

1. **Confirm Postgres is up**
   ```bash
   docker compose ps | grep -q postgres || make db-up
   ```
   If `DATABASE_URL` is unset or Postgres is down, integration tests using `internal/shared/testdb` skip silently — verify it is up before claiming green.

2. **Tight loop while iterating**
   ```bash
   go test ./internal/<domain>/... -run <TestName> -count=1 -v
   ```
   `-count=1` defeats the test cache; `-v` shows subtests. Iterate until green.

3. **Widen to the package, then the suite**
   ```bash
   go test ./internal/<domain>/... -count=1
   make test
   ```
   `make test` runs `go test -p 1 ./...` because all integration packages share one Postgres. Do not work around the `-p 1`.

4. **Pre-commit verification (binding before any commit)**
   ```bash
   make fmt && make vet && make test && go build ./...
   ```
   All four must succeed. If `make fmt` rewrote files, stage them deliberately (do not bundle them with unrelated diffs).

5. **Triaging a failure**
   - Re-run with `-v -run <TestName>` to isolate.
   - Check Postgres state: `psql "$DATABASE_URL" -c '\dt'` and inspect the table referenced by the failing assertion. Remember `testdb.Open` truncates between tests.
   - For HTTP handler tests, log `rr.Body.String()` to see the rendered output.
   - For flaky time-based tests, audit for `time.Now()` — should be `internal/shared/clock`.
   - For `pq: duplicate key value` on `ux_time_entries_one_active_per_user_workspace`: this is the active-timer invariant. Handlers should map `SQLSTATE 23505` to `tracking.ErrActiveTimerExists` → HTTP 409.

6. **Lint**
   ```bash
   make lint   # currently `go vet`-equivalent; add tools as the lint config grows
   ```

**Guardrails**
- Never claim "tests pass" without actually running them in the current branch.
- Never run integration tests in parallel inside one package.
- Never paper over a flake with retries — find the source (clock, ordering, fixture leakage).
- If you added a new domain table, append it to the truncate list in `internal/shared/testdb/testdb.go`; otherwise the next test will see leaked rows.

**Fluid Workflow Integration**
Invoke between every meaningful task in `tasks.md` (run the targeted test) and once at the end of an `openspec-apply-change` session (run the full `make fmt && make vet && make test && go build ./...` gate before any commit or archive.
