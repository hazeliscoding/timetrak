---
name: tt-write-integration-test
description: Write a workspace-scoped integration test using internal/shared/testdb that opens a real Postgres pool, seeds fixtures, and asserts behavior end-to-end. Use when implementing a task that requires test coverage for a service, handler, or cross-workspace boundary.
license: MIT
compatibility: TimeTrak. Requires DATABASE_URL set; tests skip cleanly when it is not.
metadata:
  author: timetrak
  version: "1.0"
---

Author a Go integration test under the appropriate `internal/<domain>/` package that uses `internal/shared/testdb` to open a real Postgres pool. Safe to invoke mid-`openspec-apply-change` for any task that says "add test for X" or "cover the cross-workspace path".

**Steps**

1. **Pick the test file**
   - Service-level behavior → `internal/<domain>/service_test.go` (`package <domain>`, white-box).
   - Handler-level behavior including authz boundaries → `internal/<domain>/<feature>_test.go` (`package <domain>_test`, black-box).
   - Cross-workspace denial matrix → `internal/<domain>/authz_test.go` (one row per registered route).
   - End-to-end smoke → `internal/e2e/`.

2. **Read peer for tone**
   Read `internal/clients/authz_test.go` (handler-level) and `internal/tracking/service_test.go` (service-level). Match: helper usage, assertion style, table-driven tests where the cases are similar.

3. **Boot the pool and fixtures**

   ```go
   func TestThing(t *testing.T) {
       pool := testdb.Open(t)              // skips if DATABASE_URL unset
       f := testdb.SeedAuthzFixture(t, pool)  // standard 2-workspace, 2-user fixture
       // f.WorkspaceA.ID, f.UserA.ID, f.ClientA.ID, f.ProjectA.ID, etc.

       svc := <domain>.NewService(pool)
       // ... act ...
       // ... assert ...
   }
   ```

   `testdb.Open` truncates every domain table on entry, so each test starts from a clean slate. **Do not** parallelize integration tests inside one package — `make test` runs `go test -p 1 ./...` precisely because all integration packages share one Postgres.

4. **For handler tests, build an httptest request**

   Mirror `internal/clients/authz_test.go`:
   ```go
   req := httptest.NewRequest(http.MethodPost, "/<resource>", strings.NewReader(form.Encode()))
   req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
   ctx := authz.WithWorkspace(req.Context(), f.UserA.ID, f.WorkspaceA.ID)
   req = req.WithContext(ctx)
   rr := httptest.NewRecorder()
   handler.ServeHTTP(rr, req)
   if rr.Code != http.StatusOK { t.Fatalf("status = %d", rr.Code) }
   ```

   For CSRF-protected POST/PUT/PATCH/DELETE: handlers check the form field `csrf_token` (or header `X-CSRF-Token`). Use the helper used by peer tests rather than rebuilding the signing logic.

5. **Cross-workspace denial pattern (binding)**

   For every route that mutates or exposes workspace data, add a row that:
   - Authenticates as UserA in WorkspaceA.
   - Targets a resource ID owned by WorkspaceB.
   - Asserts `rr.Code == http.StatusNotFound` (never 403).
   - Re-queries the resource and asserts it was **not** mutated.

6. **Money assertions**

   Assert in minor units (`int64`). Use `tracking.DurationBillable(seconds, rateMinor)` (or the equivalent helper in your domain) — never compute floats.

7. **Time assertions**

   Use `internal/shared/clock` for deterministic time in tests. Truncate to microseconds before comparing `timestamptz` round-trips from Postgres to avoid flaky equality.

8. **Run**

   ```bash
   # Targeted (fastest feedback loop)
   go test ./internal/<domain>/... -run TestThing -count=1

   # Full integration suite (serial)
   make test
   ```

   If `DATABASE_URL` is unset, `testdb.Open` calls `t.Skip` — verify the test runs locally before relying on it.

**Guardrails**
- Never call `t.Parallel()` inside an integration package — they share one DB.
- Never assert HTTP 403 on cross-workspace access. The contract is 404.
- Never rely on row ordering without an explicit `ORDER BY` in the query under test.
- If you add a new domain table, append it to the `TRUNCATE` list in `internal/shared/testdb/testdb.go` before authoring the test.
- Fixture helpers live in `internal/shared/testdb` — extend them there rather than open-coding seed SQL in each test file.

**Fluid Workflow Integration**
Invoke when a task in `tasks.md` says "add test", "cover the X path", "assert HTTP 404 for cross-workspace Y", or "verify migration Z behaves correctly". Tick the task `- [x]` only after the targeted `go test` and `make test` both pass.
