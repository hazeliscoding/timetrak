# TimeTrak brand guidelines

Status: Stage 3. Companion to `docs/timetrak_ui_style_guide.md`; see that
doc for visual-token rules. This doc covers brand marks, browser-tab
identity, and voice / microcopy. It is narrative reference, not spec.

Accepted behavior lives in `openspec/specs/` — in particular
`ui-partials` ("Brand mark partial") and `ui-showcase` ("Brand sub-surface
in the component catalogue").

---

## Wordmark usage

The TimeTrak wordmark is the partial at `web/templates/partials/brandmark.html`:
an accent bar followed by the word `TimeTrak`. It is rendered inline as SVG,
consumes only `currentColor` (for the text) and `var(--color-accent)` (for
the bar), and inherits the current light/dark theme via CSS custom properties.
There is no separate dark-mode asset.

### Clear-space

Maintain at least `--space-3` (12px) of empty space on all sides of the
wordmark. Nothing — not a heading, a border, a divider, or another control —
should intrude into that zone.

### Minimum size

The smallest acceptable rendered size is the `sm` variant, which renders the
wordmark at `var(--space-4)` (16px) tall. Anything smaller drops below the
SVG's legibility floor. If a surface is too cramped for the `sm` variant, it
is too cramped for the wordmark — use page-title text instead.

### Permitted contexts

- The application header (`web/templates/layouts/app.html`)
- The dev component showcase at `/dev/showcase/brand`
- This document

Any other context (marketing page, email footer, OG card, README header,
generated PDF, etc.) is out of scope for the current wordmark and is tracked
as a deferred follow-up in the proposal.

### Prohibited treatments

- No gradients. The wordmark uses flat fill only.
- No glow, drop shadow, or shadow lockup.
- No recolour outside `currentColor` and `var(--color-accent)`. Do not set
  the fill to a hardcoded hex value, a one-off token, or a hover state that
  diverges from the surrounding text colour.
- No tagline lockups. The wordmark never ships with a tagline fragment such
  as "TimeTrak — time tracking for freelancers" or "TimeTrak · billable
  time made simple". Descriptive copy belongs next to the product, not
  fused to the mark.
- No animated, morphing, or transitional variants. The wordmark does not
  fade, slide, pulse, or rebuild on route change.
- No 3D treatment, bevel, or emboss.
- No compositing over photography or illustration.

Accent-bar-and-text is the only supported wordmark treatment. A full logo
system — icon-only mark, app-icon tile, social-share OG card, email-signature
mark — is an explicit follow-up tracked in the proposal's Impact section and
is not part of this change.

---

## Favicon and browser-tab identity

The favicon is a single SVG file at `web/static/favicon.svg`, wired from
`web/templates/layouts/base.html` via
`<link rel="icon" type="image/svg+xml" href="/static/favicon.svg">`.
The glyph is a monochrome capital `T`. Fill references `currentColor`, with
a `prefers-color-scheme` override declared inside the SVG's own `<style>`
element so the mark inverts for dark OS themes.

### Theming is OS-driven, not app-driven

The favicon tracks the **operating-system** colour-scheme preference via
`prefers-color-scheme`. It does **not** follow TimeTrak's in-tab `data-theme`
toggle. This is a browser-platform constraint: a favicon is a separate
resource, loaded outside the document's CSS cascade, and the document's
`data-theme` attribute is not visible to it. Users who switch theme inside
the app will see the favicon continue to match their OS setting until they
change that setting. This is intentional and correct — spending engineering
effort to proxy the in-app theme to the tab icon is not justified.

### No PNG / ICO fallback in this change

Modern evergreen browsers support SVG favicons. Shipping a PNG or `.ico`
fallback is deferred. If a future contract — an enterprise legacy-browser
requirement, a specific crawler, an embedded webview — demands broader
support, that is a separate change proposal; see the deferred follow-ups
listed in the proposal's Impact section.

---

## Title convention

Every page in the app uses the format:

```
<Page name> · TimeTrak
```

The separator is U+00B7 MIDDLE DOT with a single space on each side. The
page name comes first; `TimeTrak` comes second. Pages define the title by
overriding the `{{define "title"}}` block in `base.html`. The base block's
fallback is just `TimeTrak` for pages that do not override it — currently
zero pages rely on the fallback.

### Live reference

Every shipped page already conforms. The list below is the source of truth;
consult it when adding a new page:

| Template                          | Rendered title                  |
| --------------------------------- | ------------------------------- |
| `web/templates/dashboard.html`    | `Dashboard · TimeTrak`          |
| `web/templates/time/index.html`   | `Time · TimeTrak`               |
| `web/templates/clients/index.html`| `Clients · TimeTrak`            |
| `web/templates/projects/index.html`| `Projects · TimeTrak`          |
| `web/templates/rates/index.html`  | `Rates · TimeTrak`              |
| `web/templates/reports/index.html`| `Reports · TimeTrak`            |
| `web/templates/workspace/settings.html` | `Workspace settings · TimeTrak` |
| `web/templates/auth/login.html`   | `Sign in · TimeTrak`            |
| `web/templates/auth/signup.html`  | `Create account · TimeTrak`     |
| `web/templates/errors/not_found.html` | `Not found · TimeTrak`      |

### No follow-up copy pass

All ten existing pages already use middle-dot. There is nothing to sweep.
Any new page should use the same pattern; the HTML comment above the
`{{block "title" ...}}` in `base.html` documents the convention for future
authors.

---

## Voice and microcopy

Three principles govern product copy. They compose; a good line will satisfy
all three.

### 1. Calm

Short sentences. Active verbs. No exclamation marks. No urgency signalling
for non-urgent events. When a timer has run for a long time, the app does
not say "Whoa! Still tracking?" — it says "Running since 09:12." When an
entry saves, the app does not say "Great, saved!" — the row updates.
Confidence, not cheer.

### 2. Specific

Use domain nouns over generic productivity verbs. Prefer "Billable this
week" to "Your productivity"; "Client rate" to "Your earnings"; "Running
entry" to "Your session". Numbers and dates carry the weight of the
interface — copy introduces them and gets out of the way. Prefer a noun
phrase over a sentence wherever a label will do.

### 3. Billing-aware

TimeTrak's work is money. Copy that touches billable time is precise:
currencies appear with their codes, amounts with their minor units, rates
with their scope. Copy that touches non-billable time is not apologetic —
non-billable entries are valid first-class entries, not failure states. A
non-billable row says `Non-billable`, not `Not billable yet` or `Skipped`.

### Before / after examples

The following examples are illustrative, drawn from shipped templates. They
are not a mandate to change every template. Several are `Keep as-is`
positive examples — existing TimeTrak microcopy is already strong in most
places, and showing what works is as useful as flagging what to improve.

#### Empty-state copy

**`web/templates/clients/index.html`:35** — The clients list before any
client exists.
- Before: `No clients yet. Add your first client above to start tracking projects against it.`
- After:  Keep as-is.
- Why:    Specific (uses `client`, `projects`), calm, and gives the user a
  concrete next action tied to the form immediately above.

**`web/templates/projects/index.html`:60** — Projects list when none exist.
- Before: `No projects yet. Create your first project above.`
- After:  `No projects yet. Add one above and assign it to a client.`
- Why:    Specificity — names the relationship (`client`) the reader needs
  to hold in their head, because projects are meaningless without one.

**`web/templates/partials/rates_table.html`:4** — Rates list when no rules
exist.
- Before: `No rate rules yet. Create at least a workspace-default rule so billable entries can be valued.`
- After:  Keep as-is.
- Why:    Billing-aware — explains *why* a default rule matters (billable
  entries cannot be valued without one) rather than telling the user to
  "get started".

**`web/templates/time/index.html`:85** — Entries list with filters applied
and no matches.
- Before: `No entries match your filters. Start a timer from the dashboard or add a manual entry above.`
- After:  Keep as-is.
- Why:    Specific about *why* the list is empty (filters, not absence of
  data) and offers two concrete next actions.

#### Confirmation copy

**`web/templates/partials/entry_row.html`:56** — Delete-entry `hx-confirm`.
- Before: `Delete this entry? Reports will recompute without it.`
- After:  Keep as-is.
- Why:    Billing-aware — warns that reporting totals will change, which is
  the real cost of the action. Calm, no capitalised urgency.

**`web/templates/partials/client_row.html`:43** — Archive-client confirm.
- Before: `Archive this client? They can be unarchived later; existing entries are preserved.`
- After:  Keep as-is.
- Why:    Specific about reversibility and data preservation — both are
  what a freelancer actually wants to know before archiving a client.

**`web/templates/partials/rate_row.html`:63** — Delete-rate-rule confirm
(only shown when the rule has zero referenced entries).
- Before: `Delete this rate rule? This cannot be undone.`
- After:  `Delete this rate rule? No entries reference it, so no totals will change.`
- Why:    Billing-aware — "cannot be undone" is generic SaaS copy; the real
  information is that totals will not shift, because the UI already blocked
  deletion of referenced rules.

#### Validation error copy

**`web/templates/auth/signup.html`:9** — Generic signup fallback error.
- Before: `Please fix the errors below.`
- After:  Keep as-is.
- Why:    Calm, specific, no exclamation; it points at the inline field
  errors rather than repeating them.

**`web/templates/partials/rate_form.html`** (hint on the rate amount
field) — `Enter as a decimal; stored as integer minor units.`
- Before: `Enter as a decimal; stored as integer minor units.`
- After:  Keep as-is.
- Why:    Billing-aware and honest — it tells a technically literate user
  exactly what happens to their input, which matters for a money field.

**`web/templates/workspace/settings.html`:13** — Helper text on the
reporting-timezone picker.
- Before: `Reports bucket time entries into calendar days using this timezone. Changing it takes effect on the next report request — existing entries are not modified.`
- After:  Keep as-is.
- Why:    Specific about scope of effect (next request, not retroactive)
  and billing-aware (existing entries are preserved). Exactly what a
  cautious operator needs before changing a reporting setting.

#### Loading / running-state copy

**`web/templates/partials/timer_widget.html`:7** — Running-timer badge.
- Before: `Running`
- After:  Keep as-is.
- Why:    Calm. One word. The elapsed counter below it carries the data;
  the label just names the state.

**`web/templates/partials/timer_widget.html`:11** — Running-timer start
timestamp.
- Before: `Started {{formatTime .Running.StartedAt}}`
- After:  Keep as-is.
- Why:    Specific (an actual timestamp), calm (no "still running!"), and
  billing-aware implicitly — the user can read the clock and decide.

**`web/templates/partials/spinner.html`:13** — Loading indicator.
- Before: `Loading…`
- After:  Keep as-is.
- Why:    Calm. The sr-only label is the right place for the generic word;
  visible surfaces should name the specific thing that is loading (e.g.
  the report table swaps in with its own copy).

**`web/templates/partials/dashboard_summary.html`:29** — Dashboard card
when no billable entries exist this week.
- Before: `No billable entries yet`
- After:  Keep as-is.
- Why:    Billing-aware and non-apologetic — a week without billable time
  is a state, not an error.

#### Button / action labels

**`web/templates/partials/timer_widget.html`:15 & :31** — Timer controls.
- Before: `Stop timer` / `Start timer`
- After:  Keep as-is.
- Why:    Specific verbs bound to the product's core noun. Not `Go` or
  `Begin session`.

**`web/templates/clients/index.html`:30** — Primary action on the new-client
form.
- Before: `Add client`
- After:  Keep as-is.
- Why:    Specific, imperative, matches the form heading `New client`.

**`web/templates/partials/rate_form.html`:58** — Primary action on the new
rate-rule form.
- Before: `Save rule`
- After:  Keep as-is.
- Why:    Specific — uses the domain noun (`rule`) rather than a generic
  `Save` or `Submit`.

---

## Cross-references

- `docs/timetrak_ui_style_guide.md#microcopy` — the style guide's short
  microcopy section; this document is the expanded companion.
- `openspec/specs/ui-partials/spec.md` — accepted behavior for the
  `brandmark` partial and the rest of the partial catalogue.
- `openspec/specs/ui-showcase/spec.md` — accepted behavior for the
  component showcase, including the `brand` sub-surface.

Deferred follow-ups tracked in the proposal's Impact section and *not
committed* by this document:

- PNG / ICO favicon fallback for legacy browsers
- Open Graph / social-share image
- Email-signature mark
- Marketing-surface brand kit (landing page, app-store assets, press kit)
- Full copy audit across every template

None of the above is scheduled. Propose a separate change if and when any
is needed.
