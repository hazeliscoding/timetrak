## 1. Template partials

- [x] 1.1 Create `web/templates/partials/rate_row.html` defining a `rate_row` template that renders one `<tr id="rate-{{.Rule.ID}}">` in display or edit mode from `{ Rule, Edit, Error, CSRFToken }`
- [x] 1.2 Create `web/templates/partials/rate_form.html` defining a `rate_form` template bound to `id="rate-form"` that renders the "New rate rule" card from `{ Form, Clients, Projects, CSRFToken }`, with inline error and `aria-describedby`
- [x] 1.3 Create `web/templates/partials/rates_table.html` defining a `rates_table` template that renders `<tbody id="rates-tbody">` plus the empty-state fallback
- [x] 1.4 Refactor `web/templates/rates/index.html` to compose the three partials (remove duplicated markup); keep the full-page layout intact so no-JS POST/redirect flow still renders correctly
- [x] 1.5 Replace the inline `onsubmit="return confirm(...)"` on delete with `hx-confirm="Delete this rate rule?"` on the delete button, matching clients/projects

## 2. Backend: routes and handlers

- [x] 2.1 Add `GET /rates/{id}/edit` route rendering `rate_row` with `Edit: true`; return 404 via `sharedhttp.NotFound` when the id is not in the workspace
- [x] 2.2 Add `GET /rates/{id}/row` route rendering `rate_row` with `Edit: false`; workspace-scoped lookup as above
- [x] 2.3 In `POST /rates`, branch on `HX-Request`: on success return `rates_table` swap with `#rates-tbody` plus OOB `rate_form` reset and set `HX-Trigger: rates-changed`; on validation error return 422 with `rate_form` only
- [x] 2.4 In `POST /rates/{id}`, branch on `HX-Request`: success → 200 `rate_row` (display) + `HX-Trigger: rates-changed`; `ErrRuleReferenced` → 409 `rate_row` (edit) with inline error; overlap/validation → 422 `rate_row` (edit) with inline error
- [x] 2.5 In `POST /rates/{id}/delete`, branch on `HX-Request`: success → 200 `rates_table` + `HX-Trigger: rates-changed`; `ErrRuleReferenced` → 409 `rate_row` (display) with inline error; non-HX path keeps 303 redirect
- [x] 2.6 Confirm every `rate_row` render fetches the rule via a workspace-scoped lookup so cross-workspace `GET` returns 404
- [x] 2.7 Confirm all POST handlers still validate CSRF exactly as they do today (no bypass added for HX requests)

## 3. Focus and progressive enhancement

- [x] 3.1 Add `data-focus-after-swap` to the `Effective to` input in edit mode
- [x] 3.2 Add `data-focus-after-swap` to the "Edit end date" disclosure in display mode so keyboard users land there after a successful save
- [x] 3.3 After a successful HTMX create or delete, render the OOB-refreshed `rate_form` with `data-focus-after-swap` on the scope select
- [x] 3.4 Replace inline `onchange="..."` scope toggle with a small delegated listener in `web/static/js/app.js` using `[data-scope-select]` / `[data-scope-target]`, and render all three field groups visible in the no-JS fallback
- [x] 3.5 Verify `app.js` focus helper still fires on `htmx:afterSwap` for each new swap target

## 4. Tests

- [x] 4.1 Add a handler-level test for `GET /rates/{id}/edit` and `GET /rates/{id}/row` covering in-workspace 200 and cross-workspace 404
- [x] 4.2 Add a handler-level test that an HTMX `POST /rates` success responds with `rates_table`, emits `HX-Trigger: rates-changed`, and returns the refreshed `rate_form`
- [x] 4.3 Add a handler-level test that an HTMX `POST /rates/{id}` success responds with a `rate_row` in display mode and emits `rates-changed`
- [x] 4.4 Add a handler-level test that an HTMX update against a referenced rule returns 409, renders `rate_row` in edit mode with inline error, and does NOT emit `rates-changed`
- [x] 4.5 Add a handler-level test that an HTMX delete against an unreferenced rule responds with `rates_table`, emits `rates-changed`, and removes the row
- [x] 4.6 Add a handler-level test that an HTMX delete against a referenced rule returns 409 with `rate_row` (display mode) and does NOT emit `rates-changed`
- [x] 4.7 Add a non-HX regression test confirming `POST /rates`, `POST /rates/{id}`, and `POST /rates/{id}/delete` still respond with HTTP 303 to `/rates` when `HX-Request` is absent
- [x] 4.8 Add a handler-level test asserting tampered hidden fields on a referenced rule cannot mutate immutable columns (e.g., submitting a different `currency_code` still returns 409 and leaves the row unchanged)

## 5. Accessibility validation

- [x] 5.1 Manually verify keyboard-only flow: Tab into the rates table, activate "Edit end date", change the date, submit, confirm focus lands on the refreshed row's disclosure
- [x] 5.2 Manually verify keyboard-only delete flow including the `hx-confirm` prompt
- [x] 5.3 Verify inline error messages have `role="alert"`, `aria-live="assertive"`, and are referenced by `aria-describedby` on the relevant input(s)
- [x] 5.4 Verify visible focus rings on all inputs, buttons, and the disclosure control within both partials
- [x] 5.5 Verify the page is usable end-to-end with JavaScript disabled (create, edit, delete via POST/redirect)
- [x] 5.6 Spot-check color-contrast of error text and the "Referenced by N entries" hint against background tokens

## 6. Documentation

- [x] 6.1 Update `CLAUDE.md` "HTMX peer-refresh events" line to mention `rates-changed` alongside existing events
- [x] 6.2 Update the rates section of `docs/time_tracking_design_doc.md` (if it documents UI flows) to reflect the partial-based interaction model
