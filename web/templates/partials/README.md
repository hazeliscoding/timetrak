# Partials Catalogue

**Browser-visible reference:** `/dev/showcase/components` (dev-only) renders
every partial live against documented `dict` payloads, with copy-ready
snippets. See `internal/showcase/` for the catalogue definition.

This directory holds TimeTrak's reusable `html/template` partials. Every page
template is parsed together with every layout and every partial in this
directory (see `internal/shared/templates`), so a partial defined here is
callable from any page.

**Sibling doc.** CSS authoring conventions (token taxonomy, `@layer`
order, `tt-<component>` naming, focus / target-size / status rules)
live in [`web/static/css/README.md`](../../static/css/README.md). When
adding a partial that ships new CSS, follow that contract.

## Conventions

### File and block naming

- One file per partial: `web/templates/partials/<name>.html`.
- Each file defines exactly one block: `{{define "<name>"}}` (bare block name,
  matching the file stem).
- Callers invoke it as `{{template "<name>" .Context}}`.

**Deviation from the change proposal:** The original design proposed
`{{define "partials/<name>"}}` to namespace block names. That namespacing
was NOT adopted because Go handlers already call the existing bare names
(`client_row`, `entry_row`, `flash`, `rate_form`, etc.) via
`tpls.RenderPartial(..., "clients.index", "client_row", ...)`. Renaming the
blocks would force handler changes that are out of scope for this refactor.
The bare-name convention is the project-wide rule going forward.

### Context shape

Each partial below documents the keys it expects on `.`. Optional keys use the
documented default when omitted (callers pass them via the `dict` template
func).

### Authoritative HTMX event names

The server emits these on the `HX-Trigger` response header after successful
mutations. Peer partials listen via `hx-trigger="<name> from:body"`.

| Event              | Emitted by                                  | Listened for by                                   |
| ------------------ | ------------------------------------------- | ------------------------------------------------- |
| `timer-changed`    | `POST /timer/start`, `POST /timer/stop`     | dashboard summary                                 |
| `entries-changed`  | timer start/stop, entry create/update/delete | dashboard summary, entries-list (implicit)       |
| `clients-changed`  | client create / update / archive / unarchive | (reserved — no live listeners yet)              |
| `projects-changed` | project create / update / archive / unarchive | (reserved — no live listeners yet)             |
| `rates-changed`    | rate rule create / update / delete          | (reserved — reporting will subscribe in a later change) |
| `workspace-changed`| workspace settings save                      | reports results panel                             |

**Rule:** denied (403/404) requests MUST NOT emit any `*-changed` event; the
shared not-found renderer strips `HX-Trigger` as a safety net.

### `data-focus-after-swap`

`web/static/js/app.js` moves focus to the first `[data-focus-after-swap]`
element inside an HTMX swap target after each swap. Apply this attribute ONLY
on intentional swaps (user submits a form, opens an inline editor, hits Save).
Passive peer-refresh swaps (dashboard summary reacting to `timer-changed`,
rates table refresh) MUST NOT carry the attribute.

**Form-error rule:** when a form partial re-renders with validation errors,
the error summary (or the first invalid control) MUST carry
`data-focus-after-swap` so keyboard and screen-reader users land on the error.

**Contract test.** The `data-focus-after-swap` contract is enforced by
`internal/e2e/browser/focus_after_swap_test.go` (gated by
`//go:build browser`; run via `make test-browser`). When you add a new
HTMX-focus target to a partial, add a matching scenario to that test —
the convention is: every documented intent swap MUST leave
`document.activeElement` carrying `[data-focus-after-swap]` after
`htmx:afterSettle`. If you change a documented target, update both this
README and the scenario.

### Row OOB swap convention

Row partials render `<tr id="<domain>-row-<uuid>">` so handler responses can
replace them either as the direct swap target or via `hx-swap-oob="true"`.
Current row ids: `client-row-<uuid>`, `project-row-<uuid>`, `entry-row-<uuid>`,
`rate-row-<uuid>`.

### Extraction bar

A markup pattern is promoted to a canonical partial ONLY when it is used by
two or more domain templates. One-offs stay in the domain template. When
splitting an existing partial would require more than four optional slot keys,
split the partial instead of adding a fifth.

### Template funcs

No new template funcs were added for this refactor. The existing set is:
`dict`, `seq`, `formatDate`, `formatTime`, `formatDuration`, `formatMinor`,
`iso`, `add`, `sub`.

---

## Canonical partials

### `form_field`

Visible-label text-style input with an optional hint and an optional inline
error. Covers `type=text|email|password|date|time|number|url|tel`.

**Context keys:**

| Key              | Required | Default | Notes                                                     |
| ---------------- | -------- | ------- | --------------------------------------------------------- |
| `ID`             | yes      | —       | Matches the `<label for>` and input `id`.                |
| `Name`           | yes      | —       | Form field name.                                          |
| `Label`          | yes      | —       | Visible label text.                                       |
| `Type`           | no       | `text`  | Any text-like HTML input type.                            |
| `Value`          | no       | `""`    | Current value.                                            |
| `Required`       | no       | false   | Emits `required` + `aria-required="true"`.                |
| `Autofocus`      | no       | false   | Emits `autofocus`.                                        |
| `Autocomplete`   | no       | ``      | If set, emits `autocomplete="..."`.                       |
| `Placeholder`    | no       | ``      | If set, emits `placeholder="..."`.                        |
| `Hint`           | no       | ``      | Renders a `<span class="hint">`.                          |
| `ErrorID`        | no       | ``      | If set and `Invalid` true, links via `aria-describedby`. |
| `Invalid`        | no       | false   | Emits `aria-invalid="true"`.                              |
| `Inputmode`      | no       | ``      | Passes through to `inputmode`.                            |
| `Maxlength`      | no       | ``      | String; when non-empty emits `maxlength="..."`.           |
| `Width`          | no       | ``      | Inline `style="width:..."` (e.g. `7rem`).                 |

**Scope:** covers simple single-input fields. Selects, checkboxes, radio
groups, composite date/time pairs, and rows of side-by-side fields stay
inline in the domain template — extracting them forces slot explosion or
requires template funcs we do not have.

**Accessibility:** `<label for>` is the label source; the control is the focus
target. Error wiring is the caller's responsibility via `ErrorID` + a
sibling `form_errors` summary (or an inline error node with that id).

**Event contract:** neither emits nor listens.

---

### `form_errors`

Top-of-form error summary. Renders a single message or nothing. Focusable so
callers can set `data-focus-after-swap` on the summary to send keyboard and
screen-reader focus to the error.

**Context keys:**

| Key         | Required | Default | Notes                                                                    |
| ----------- | -------- | ------- | ------------------------------------------------------------------------ |
| `ID`        | yes      | —       | Stable id the first invalid control can reference via `aria-describedby`. |
| `Message`   | yes      | —       | Error copy. If empty, the partial renders nothing.                      |
| `NoFocus`   | no       | false   | When truthy, suppresses `data-focus-after-swap`. Default emits it.      |

Renders `role="alert"` + `tabindex="-1"` when a message is present, so the
summary is announced immediately and keyboard-focusable after the swap.

**Accessibility:** the summary is the focus target after a validation swap.
Associated controls should set `aria-invalid="true"` and
`aria-describedby="<ID>"`. Status is conveyed by text + semantic role; never
color alone.

**Event contract:** neither emits nor listens.

---

### `empty_state`

Copy-first empty block. Used when a list, table, or filtered result has no
rows. No icon, no color-only meaning.

**Context keys:**

| Key          | Required | Default | Notes                                                   |
| ------------ | -------- | ------- | ------------------------------------------------------- |
| `Title`      | yes      | —       | `<h2>` copy.                                           |
| `Body`       | yes      | —       | One-sentence explanation (plain text or simple markup — passed as string). |
| `ActionHref` | no       | ``      | If set, renders a primary-action link.                  |
| `ActionText` | no       | ``      | Label for the action link. Required when `ActionHref` set. |
| `Live`       | no       | false   | When true, sets `aria-live="polite"` on the wrapper (use for filtered lists whose empty state appears after an HTMX swap). |

**Accessibility:** the title conveys meaning; body and optional action are
supplementary. Wrapper is a landmark-less `div.empty.card`.

**Event contract:** neither emits nor listens. `Live` is a hint for
peer-refresh consumers that the empty view may arrive via an HTMX swap.

---

### `flash`

Page-level toast list (drawn once near the top of the app shell). Severity
drives the ARIA role.

**Context:** slice of flash entries, each with `.Kind` and `.Message`.

| `.Kind`   | Role         | Notes                                           |
| --------- | ------------ | ----------------------------------------------- |
| `success` | `status`     | Non-urgent confirmation. `aria-live="polite"`. |
| `info`    | `status`     | Non-urgent info. `aria-live="polite"`.         |
| `warn`    | `status`     | Non-urgent warning. `aria-live="polite"`.      |
| `error`   | `alert`      | Urgent. Announced immediately by assistive tech. |

**Accessibility:** status is conveyed via role + text + the `flash-<kind>`
class — color alone is never the signal.

**Event contract:** neither emits nor listens. The flash area is the
canonical OOB-swap target when a handler needs to surface a page-level
message.

---

### `spinner`

Inline loading indicator. Paired with HTMX `hx-indicator`.

**Context:** ignored (no fields read).

**Accessibility:** `role="status"` + `aria-live="polite"` + an sr-only
"Loading…" label. It is a supplementary cue: the eventual swap (table,
results, row) is the real completion signal.

**Event contract:** neither emits nor listens. Does NOT pair with
`data-focus-after-swap` — the indicator itself should not steal focus.

---

### `pagination`

Prev/next navigation for offset-paginated lists.

**Context keys:**

| Key          | Required | Default | Notes                                  |
| ------------ | -------- | ------- | -------------------------------------- |
| `Page`       | yes      | —       | 1-indexed current page.                |
| `TotalPages` | yes      | —       | Total page count (0 or 1 → hidden).    |
| `PrevQuery`  | yes      | —       | Query string for the previous page.    |
| `NextQuery`  | yes      | —       | Query string for the next page.        |

**Accessibility:** rendered in a `<nav aria-label="Pagination">`. Prev/next
anchors carry `rel` attributes and visible text.

**Event contract:** neither emits nor listens.

---

### `confirm_dialog`

Focus-trapped `<dialog>` for destructive actions that need side-effect copy
(e.g. "archive a client with active projects"). Currently UNUSED in the
shipped app — every destructive delete uses native `hx-confirm` for MVP.

**When to prefer `confirm_dialog` over `hx-confirm`:**

- The confirmation must surface data the user did not already see (active
  project count, running-timer warning, cascade impact).
- The flow needs keyboard focus trapped while the user reads consequences.

**When to use `hx-confirm` (the default):**

- Simple row deletes with obvious scope ("Delete this entry?", "Archive this
  client?").
- No additional context needed beyond the button label.

If/when this partial is first adopted, a small focus-trap helper must be
added to `web/static/js/app.js` alongside it.

**Event contract:** neither emits nor listens by itself; the confirm button
triggers whatever `hx-<method>` the caller wires.

---

## Domain partials

These are NOT canonical building blocks; they are domain-specific but live
here so handlers can render them via `RenderPartial`. They DO participate in
the HTMX event contract.

### `client_row`, `project_row`, `entry_row`, `rate_row`

Row renderers. Root element is `<tr id="<domain>-row-<uuid>">` for OOB swap.

| Row            | Emitting event (handler MUST set) |
| -------------- | --------------------------------- |
| `client_row`   | `clients-changed`                 |
| `project_row`  | `projects-changed`                |
| `entry_row`    | `entries-changed`                 |
| `rate_row`     | `rates-changed`                   |

See individual files for the context shapes (`.Client`/`.Project`/`.Entry`/
`.Rule` plus `CSRFToken`, `Edit`, optional `Error`, etc.).

### `rate_form`, `rates_table`

Rates domain composites. `rate_form` supports `hx-swap-oob="true"` via `.OOB`.
`rates_table` renders the full `#rates-table` region (list + empty state).

### `timer_widget`

Dashboard timer control. Emits `timer-changed, entries-changed` on start/stop
via its form posts. Listens indirectly: peer partials listen for its events.

### `tracking_error`

Shared inline error region for tracking integrity failures (active-timer
conflict, cross-workspace project, invalid interval). Consumed by
`timer_widget` and `entry_row`. `role="alert"`, `tabindex="-1"`,
`data-focus-after-swap`.

### `dashboard_summary`

Dashboard totals card row. Swapped in response to `timer-changed`/
`entries-changed` from `body`. Listens; does not emit.

### `report_summary`, `reports.partial.results`, `reports.partial.empty`

Reporting composites. Rendered by `GET /reports/partial` in response to
filter changes and `workspace-changed`.

---

## Deferred / not extracted

Two shared-looking patterns were intentionally NOT extracted this pass:

- **`table_shell`** — each domain table has a unique `<thead>` and row
  template. Go `html/template` cannot accept HTML blocks as slots without
  adding a `safeHTML`-style helper, and the wrapper boilerplate is only ~5
  lines per domain. Extracting it would either require a new template func
  or force ugly string-as-HTML passing. Revisit if a token change needs a
  single place to tweak table chrome.
- **`filter_bar`** — filter control sets vary per domain and share only the
  outer `<form class="card">` wrapper plus optional `hx-trigger` debounce.
  The savings (one line per domain) did not meet the extraction bar. The
  reports page already implements the documented `hx-trigger="change from:find
  select, change from:find input[type=date], submit"` convention inline;
  future filterable tables SHOULD adopt the same contract directly.

## Tracking deviations

- Block names are bare (`client_row`) not namespaced (`partials/client_row`).
  See "File and block naming" above.
- Row ids normalised from `<domain>-<uuid>` to `<domain>-row-<uuid>` as part
  of this change. No Go handler changes were required (only templates
  referenced the old ids).
