## MODIFIED Requirements

### Requirement: Harness MUST reuse the existing server bootstrap and testdb fixtures

The browser harness SHALL launch the TimeTrak HTTP server via the same bootstrap used by `internal/e2e/happy_path_test.go` and SHALL reuse `internal/shared/testdb` for schema setup and per-test truncation. It MUST NOT introduce a parallel server, a parallel migration runner, or a parallel database lifecycle.

"Same bootstrap" means route parity with `cmd/web/main.go`: every route registered by the production server (including static assets at `/static/*` and the settings surface at `/settings`) SHALL be registered by the test bootstrap. The only permitted divergences are the fixed test-only session secret, the `APP_ENV=dev` flag that enables the developer showcase surface, and the use of `httptest.Server` as the network transport.

#### Scenario: Shared server bootstrap
- **WHEN** a browser test starts
- **THEN** the server is constructed through the shared `internal/e2e` helper used by the non-browser e2e test
- **AND** the database is prepared through `internal/shared/testdb`

#### Scenario: Cross-workspace isolation between tests
- **WHEN** two browser tests execute sequentially against the same database
- **THEN** each test sees a clean set of domain tables via the existing truncate fixtures
- **AND** neither test can observe state written by the other

#### Scenario: Static assets are served by the test server

- **WHEN** a browser test requests `/static/css/tokens.css`, `/static/css/app.css`, `/static/js/app.js`, or `/static/vendor/htmx.min.js`
- **THEN** the test server returns the file from `web/static/` with HTTP 200
- **AND** the rendered page has the real compiled CSS applied, so contract assertions that depend on computed styles (target-size, focus-ring contrast, tabular-nums, accent borders) have a valid DOM to inspect.

#### Scenario: Settings route is registered on the test server

- **WHEN** a browser test navigates an authenticated workspace-scoped session to `/settings`
- **THEN** the response status is 200
- **AND** the page renders under the standard app layout with a valid `<title>` and `lang` attribute, so `TestAxeSmokePerPage/settings` can assert WCAG 2.2 AA compliance against a real page.
