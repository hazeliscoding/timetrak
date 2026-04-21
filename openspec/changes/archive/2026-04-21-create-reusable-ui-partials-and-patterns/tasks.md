## 1. Inventory and Convention

- [x] 1.1 Catalogue every file under `web/templates/partials/` with its current `{{define}}` block name, expected `.` shape, and call sites across domain templates.
- [x] 1.2 For each domain (`clients`, `projects`, `rates`, `tracking`, `reporting`, `dashboard`, `auth`, `workspace`), grep their templates and list markup blocks duplicated across two or more domains (forms, tables, empty states, filter bars, flash, spinner, row shapes, confirm).
- [x] 1.3 Decide the canonical partial set against the extraction bar (used in 2+ domains) and record rejections so one-offs do not get extracted.
- [x] 1.4 Draft `web/templates/partials/README.md` documenting: naming rule (`partials/<name>`), slot/`dict` convention with key defaults, authoritative `*-changed` event list, `data-focus-after-swap` rule, row OOB-swap id convention (`<domain>-row-<uuid>`), and the per-partial WCAG 2.2 AA obligations.

## 2. Extract Canonical Building Blocks

- [x] 2.1 Implement `web/templates/partials/form_field.html` (label + native control + hint + inline error; `aria-invalid` and `aria-describedby` wired).
- [x] 2.2 Implement `web/templates/partials/form_errors.html` (top-of-form summary with `role="alert"`, focusable for `data-focus-after-swap`).
- [x] 2.3 Implement `web/templates/partials/table_shell.html` (thead + tbody slot + empty-state slot; preserves accessible table name).
- [x] 2.4 Implement `web/templates/partials/empty_state.html` (title + body + optional action slot; copy-first, no color-only meaning).
- [x] 2.5 Implement `web/templates/partials/filter_bar.html` (native controls; debounced `hx-trigger="change delay:200ms"`).
- [x] 2.6 Normalize `web/templates/partials/flash.html` severity mapping (`info`/`success` → `role="status"`; `warn`/`error` → `role="alert"`).
- [x] 2.7 Normalize `web/templates/partials/spinner.html` to use `aria-live="polite"` and document when it pairs with `data-focus-after-swap`.
- [x] 2.8 Normalize `web/templates/partials/pagination.html` context shape and ensure prev/next controls have accessible names.
- [x] 2.9 Audit `web/templates/partials/confirm_dialog.html` and document in README when to prefer it over `hx-confirm`.
- [x] 2.10 Confirm no new template funcs are needed; if one is, justify it inline in the README and the PR description.

## 3. Row Partial Standardization

- [x] 3.1 Ensure `client_row`, `project_row`, `entry_row`, and `rate_row` each render a root with `id="<domain>-row-<uuid>"`.
- [x] 3.2 Document in the README which `*-changed` event each row's mutating handler MUST emit.
- [x] 3.3 Verify every current handler that returns an OOB row actually emits its documented `*-changed` event (no handler changes — audit only; open a follow-up change if any drift is found).

## 4. Migrate Domain Templates

- [x] 4.1 Migrate `web/templates/clients/` to compose `form_field`, `form_errors`, `table_shell`, `empty_state`, `filter_bar`, and normalized `flash`/`pagination`.
- [x] 4.2 Migrate `web/templates/projects/` to compose the canonical partials; delete now-dead inline markup.
- [x] 4.3 Migrate `web/templates/rates/` (including `rate_form.html`) to compose the canonical partials.
- [x] 4.4 Migrate `web/templates/tracking/` (entry list, edit form, `tracking_error.html`, `timer_widget.html` call sites) to compose the canonical partials.
- [x] 4.5 Migrate `web/templates/reporting/` (`report_empty.html`, `report_results.html`, `report_summary.html`) to compose the canonical partials.
- [x] 4.6 Migrate `web/templates/dashboard/` (including `dashboard_summary.html`) to compose the canonical partials.
- [x] 4.7 After each domain migration, remove any partial file that has become unreferenced.

## 5. Accessibility Validation

- [ ] 5.1 Keyboard-only walkthrough for each migrated domain: Tab/Shift+Tab order, visible focus, Enter/Space on actionable controls, Escape closes dialogs.
- [ ] 5.2 Verify `data-focus-after-swap` lands focus on the correct element after: timer start/stop, entry create/edit/delete, client/project/rate create/edit/delete, report filter change, and form submit with validation errors.
- [ ] 5.3 Screen-reader spot-check (NVDA or VoiceOver): flash announcements (`status` vs `alert`), form error summary, empty states, pagination labels.
- [ ] 5.4 Contrast spot-check on `flash` severities, disabled states, and focus rings against the existing token system.
- [ ] 5.5 Confirm no component conveys status by color alone (icons, text, or ARIA labeling present).

## 6. HTMX Contract Verification

- [x] 6.1 Smoke-test each `*-changed` event end-to-end: `timer-changed`, `entries-changed`, `clients-changed`, `projects-changed`, `rates-changed` trigger their documented peer refreshes.
- [x] 6.2 Verify every OOB row swap still replaces the correct `id="<domain>-row-<uuid>"` after migration.
- [x] 6.3 Verify flash OOB swap target still works for success and error paths across all migrated domains.

## 7. Visual Regression Sanity Check

- [ ] 7.1 Before/after screenshot comparison for one representative page per domain (list + detail/form) to confirm no unintended visual drift.
- [ ] 7.2 Spot-check medium-density table layouts and form spacing against the existing style guide (`docs/timetrak_ui_style_guide.md`); no redesign, only parity.

## 8. Finalize

- [x] 8.1 `make fmt`, `make vet`, `make lint`, `make test` all green.
- [x] 8.2 Update `web/templates/partials/README.md` with any late adjustments discovered during migration.
- [ ] 8.3 Prepare archive entry via `/opsx:archive create-reusable-ui-partials-and-patterns` once merged.
