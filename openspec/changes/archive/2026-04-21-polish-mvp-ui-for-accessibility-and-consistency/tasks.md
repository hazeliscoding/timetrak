## 1. Shared layout, tokens, and helpers

- [x] 1.1 Audit `web/templates/layouts/base.html` skip link: confirm `#main` is the first landmark and that the skip link is the first focusable element on every page; fix if the order regressed.
- [x] 1.2 Add `data-focus-after-swap` convention documentation (a short comment block near the `htmx:afterSwap` handler in `web/static/js/app.js`) that cross-references the focus-flow catalogue in `design.md`.
- [x] 1.3 Update the theme toggle buttons in `web/templates/layouts/app.html` to render `aria-pressed="true|false"` based on the stored theme; update `app.js` to keep `aria-pressed` in sync with `applyTheme`.
- [x] 1.4 Add a numeric-alignment utility class to `web/static/css/app.css` (e.g. `.num { text-align: right; font-variant-numeric: tabular-nums; }`); do not use inline `style=` in templates.
- [x] 1.5 Add a reduced-motion guard to `web/static/css/app.css` that wraps any transition introduced by this change in `@media (prefers-reduced-motion: reduce) { *, *::before, *::after { transition: none !important; animation: none !important; } }` scoped tightly; verify no regression on existing transitions.
- [x] 1.6 Ensure every handler's page render sets an explicit `<title>` block (Dashboard, Time, Clients, Projects, Rates, Reports, Settings, Login, Signup). Confirm no page falls back to the layout default.
- [x] 1.7 Verify landmark roles (`banner`, `navigation`, `main`) are present exactly once on every rendered page; add `role="contentinfo"` to a footer if one exists, otherwise leave unset.

## 2. Form error wiring and consistency (per decision D3)

- [x] 2.1 Define a uniform error-map shape on handlers (`map[string]string` keyed by field name) for `auth`, `clients`, `projects`, `rates`, `workspace`, and tracking inline-edit flows; thread it through to templates.
- [x] 2.2 Add `aria-describedby` + `aria-invalid="true"` wiring on every input in `web/templates/auth/login.html` and `web/templates/auth/signup.html`; render a top-of-form error summary with `role="alert"`, `tabindex="-1"`, and `data-focus-after-swap`.
- [x] 2.3 Apply the same wiring to `web/templates/clients/index.html` (create and inline-edit rows) and update `web/templates/partials/client_row.html` where inline edit renders.
- [x] 2.4 Apply the same wiring to `web/templates/projects/index.html` and `web/templates/partials/project_row.html`.
- [x] 2.5 Apply the same wiring to `web/templates/rates/index.html` and `web/templates/partials/rate_form.html`; verify the scope toggle still works with no-JS fallback.
- [x] 2.6 Apply the same wiring to `web/templates/workspace/settings.html`.
- [x] 2.7 Apply the same wiring to the entry inline editor (`web/templates/partials/entry_row.html`); the row-level error summary replaces a top-of-form summary.
- [x] 2.8 Mark every required input with both visible text and `aria-required="true"` across the forms touched above.

## 3. Table semantics (per decision D5)

- [x] 3.1 Add `<caption>` (visually hidden where appropriate via `.sr-only`), `<th scope="col">` on header cells, and `<th scope="row">` on the identifying cell per row to the clients table in `web/templates/clients/index.html`; update `web/templates/partials/client_row.html` accordingly.
- [x] 3.2 Same treatment for the projects table (`web/templates/projects/index.html`, `web/templates/partials/project_row.html`).
- [x] 3.3 Same treatment for the rates table (`web/templates/partials/rates_table.html`, `web/templates/partials/rate_row.html`).
- [x] 3.4 Same treatment for the time entries list (`web/templates/time/index.html`, `web/templates/partials/entry_row.html`).
- [x] 3.5 Same treatment for the reports results table (`web/templates/reports/index.html`, `web/templates/partials/report_results.html`).
- [x] 3.6 Replace any inline `style="text-align:right"` on numeric cells with the `.num` utility class across all five tables.
- [x] 3.7 If the reports results table has any sortable column today, add `aria-sort` to the currently-sorted header and convert the header's click target to a `<button>` inside the `<th>`. If no sortable columns exist, skip this task and note in the verification section.

## 4. Status pills and non-color-only indicators (per decision D4)

- [x] 4.1 Audit `Archived` pills on Clients and Projects: ensure visible text `Archived` and verify 4.5:1 contrast against the pill background; adjust only the offending pill color token if needed.
- [x] 4.2 Audit the `Running` indicator on the timer widget (`web/templates/partials/timer_widget.html`): ensure visible text and contrast.
- [x] 4.3 Audit `No rate` indicator on Projects and anywhere a rate is resolved: ensure visible text and contrast.
- [x] 4.4 Audit `Billable` / `Non-billable` indicators on time entries: ensure visible text and contrast.
- [x] 4.5 Audit rate-scope labels (`project` / `client` / `workspace default`) in `web/templates/partials/rate_row.html`: ensure text is the primary differentiator, color is decorative, and all three variants meet 4.5:1 contrast.
- [x] 4.6 Document any token-level contrast change made during the audit in `design.md` under a new "Implementation notes" section (inline append) so the follow-up token work knows about it.

## 5. Destructive confirmation rationalization (per decision D1)

- [x] 5.1 List every destructive action on the app today (clients delete/archive, projects delete, rates delete, time entry delete, timer stop) and classify each as native-`confirm()` or `<dialog>` per D1. Record the classification in a short table inside `design.md` under "Implementation notes".
- [x] 5.2 For all native-`confirm()` actions, ensure the template uses `hx-confirm` with domain-specific copy (e.g. `hx-confirm="Delete this rate rule? This cannot be undone."`). Remove any bare `hx-delete` without a confirm.
- [x] 5.3 If any action is classified as `<dialog>`-based, put `web/templates/partials/confirm_dialog.html` into use on that flow, wire the partial to show side-effect counts, and add a small focus-trap helper to `web/static/js/app.js` that traps `Tab`/`Shift+Tab` inside the open dialog and restores focus to the invoker on close. If no action warrants `<dialog>` during the audit, leave `confirm_dialog.html` unused and skip the focus-trap helper.
- [x] 5.4 For every destructive action, ensure the swap target receives `data-focus-after-swap` pointing at the documented stable anchor (e.g. `New client` button, `Start timer` button, `Add rule` button).

## 6. Focus-after-swap audit (per decision D2)

- [x] 6.1 Walk every row in the focus-flow catalogue in `design.md` and verify the markup matches. Add `data-focus-after-swap` where the catalogue says `intent`; remove it where the catalogue says `passive`.
- [x] 6.2 Verify the dashboard summary target (`#dashboard-summary`) does NOT carry `data-focus-after-swap` and its peer-refresh triggers (`timer-changed from:body, entries-changed from:body`) do not pull focus.
- [x] 6.3 Verify the rates-table peer-refresh (`rates-changed`) does NOT pull focus.
- [x] 6.4 Verify the timer widget start/stop, entry inline editor open/save/cancel, reports filter submit, and pagination all land focus on their catalogue-documented anchors.

## 7. Live regions and empty/loading/error states (per decision D6)

- [x] 7.1 Verify `#global-status` in `web/templates/layouts/app.html` remains the single page-level `aria-live="polite"` region and that HTMX responses update it via `hx-swap-oob` where appropriate.
- [x] 7.2 Ensure the reports empty-state partial (`web/templates/partials/report_empty.html`) wraps its message in `aria-live="polite"`.
- [x] 7.3 Ensure the rates empty state (inside `rates_table.html` or a new empty partial) wraps its message in `aria-live="polite"`.
- [x] 7.4 Ensure the time entries empty state wraps its message in `aria-live="polite"`.
- [x] 7.5 Ensure `web/templates/partials/spinner.html` renders with `role="status"` and `aria-hidden="true"` on any decorative glyph.
- [x] 7.6 Ensure `web/templates/partials/tracking_error.html` (and any analog) carries `role="alert"`, `tabindex="-1"`, and `data-focus-after-swap`.
- [x] 7.7 Audit the flash partial (`web/templates/partials/flash.html`) for `role="status"` on success and `role="alert"` on error; document in `design.md` under "Implementation notes" which variants the handlers emit.

## 8. Keyboard navigability walkthroughs

- [x] 8.1 Keyboard-only walkthrough: Dashboard — every interactive element reachable via `Tab`, operable via `Enter`/`Space`, and focus ring visible. Log any gap as a numbered sub-task and fix inline.
- [x] 8.2 Keyboard-only walkthrough: Time entries list (including inline editor open/save/cancel, pagination).
- [x] 8.3 Keyboard-only walkthrough: Clients (including new-client, edit, delete, archive).
- [x] 8.4 Keyboard-only walkthrough: Projects (including new-project, edit, delete).
- [x] 8.5 Keyboard-only walkthrough: Rates (including rate form scope toggle, add, delete).
- [x] 8.6 Keyboard-only walkthrough: Reports (including filter bar, submit, pagination, empty state).
- [x] 8.7 Keyboard-only walkthrough: Auth (login + signup, including error flow).
- [x] 8.8 Keyboard-only walkthrough: Workspace settings.
- [x] 8.9 Keyboard-only walkthrough: Timer widget start, stop, error flow (trigger HTTP 409 with a concurrent start).

## 9. Accessibility validation per surface

- [x] 9.1 Screen-reader spot check (VoiceOver or NVDA) on the Dashboard: every region read-out, timer state announced, live-region updates audible.
- [x] 9.2 Screen-reader spot check on Time entries list: table caption, column headers, row headers, billable indicator announced; editor open/save/cancel transitions announced.
- [x] 9.3 Screen-reader spot check on Clients, Projects, Rates: table semantics read correctly; form errors announced via role=alert summary.
- [x] 9.4 Screen-reader spot check on Reports: filter labels announced, results caption reflects filter, empty state announced.
- [x] 9.5 Screen-reader spot check on Auth and Workspace settings: labels, required markers, error wiring.
- [x] 9.6 Automated pass: run axe-core (or Lighthouse accessibility) on each page and record zero critical / serious violations. Warnings triaged; any deferred items recorded as follow-ups.
- [x] 9.7 Contrast verification: measure every pill variant, muted text, focus ring, and body text combination against 4.5:1 (3:1 for focus ring per SC 1.4.11). Record measurements in a short table in `design.md` under "Implementation notes".
- [x] 9.8 Reduced-motion verification: toggle OS-level `prefers-reduced-motion` and confirm transitions collapse to instant across the polish surface.

## 10. Verification — reviewer walkthrough

The following steps must pass before the change is marked complete. Reviewers run them manually.

- [x] 10.1 Load each page (Dashboard, Time, Clients, Projects, Rates, Reports, Settings, Login, Signup); confirm each has a distinct `<title>`.
- [x] 10.2 Tab from the top of any page; confirm the skip-to-content link is the first focusable element and that activating it moves focus into `#main`.
- [x] 10.3 Submit each form with invalid data via HTMX; confirm the error summary appears, has `role="alert"`, receives focus, and each invalid input carries `aria-invalid="true"` + `aria-describedby`.
- [x] 10.4 Walk the focus-flow catalogue end-to-end: start timer, stop timer, create/edit/delete rows, open/save/cancel inline editor, submit reports filter. Focus lands on the documented anchor every time.
- [x] 10.5 Confirm no passive peer-refresh (dashboard summary, rates table on `rates-changed` when user is elsewhere) pulls focus.
- [x] 10.6 Confirm every status pill reads its text label with a screen reader and passes 4.5:1 contrast.
- [x] 10.7 Confirm both destructive confirmation patterns behave as documented; `<dialog>`-based (if any) traps focus and restores on close.
- [x] 10.8 Run axe-core / Lighthouse on every page; zero critical or serious violations.
- [x] 10.9 Confirm `prefers-reduced-motion` collapses all transitions.
- [x] 10.10 `make fmt && make vet && make test` pass.
