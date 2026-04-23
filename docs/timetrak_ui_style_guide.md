# TimeTrak UI Style Guide

## Status

This guide covers UI direction for all active and planned work on TimeTrak.
The MVP has shipped. We are now in **Stage 2 — Stabilize**, working toward a polished, accessible, and component-driven product.

Stage 2 UI priorities in order:
1. `polish-mvp-ui-for-accessibility-and-consistency` — close rough edges, align with this guide
2. `create-reusable-ui-partials-and-patterns` — reduce markup duplication, standardize layout patterns
3. `establish-custom-component-library-foundation` — define design tokens, base components, and product-specific widgets
4. `create-component-library-showcase-and-usage-docs` — internal showcase + usage docs
5. `refine-timetrak-brand-and-product-visual-language` — stronger identity, visual taste

For brand marks, browser-tab identity, and voice / microcopy, see
`docs/timetrak_brand_guidelines.md` (companion doc, narrative reference).

---

## Product Feel

TimeTrak should feel like a calm, trustworthy work tool for freelancers and contractors.

The UI should be:
- practical, not flashy
- polished, not overdesigned
- efficient, not cluttered
- warm enough to feel human, but serious enough to handle money and billing

Avoid the generic AI-SaaS look:
- oversized hero sections
- random gradients
- floating decorative blobs
- vague copy like "Boost productivity"
- dashboards full of empty cards
- too many identical rounded panels
- charts where a table would work better

The product should look closer to:
- a focused operations app
- a billing-aware work tracker
- a tool someone would trust for invoices

---

## Design Principles

### 1. Data-first
TimeTrak is about time, money, projects, and clients.
Design around those objects, not abstract "insights."

### 2. Clear hierarchy
Every screen should make the primary action obvious:
- start timer
- stop timer
- add entry
- edit entry
- view billable totals
- switch project/client

### 3. One strong system
Use one spacing scale, one border system, one radius system, and one accent color.
Consistency makes the app feel designed.

### 4. Accessible by default
Prefer semantic HTML, clear focus states, strong contrast, visible labels, and keyboard-friendly flows.
WCAG 2.2 AA is the enforced baseline, not an aspiration.

### 5. Calm visual tone
Use restrained surfaces, borders, and typography instead of heavy shadows and visual noise.

---

## Visual Direction

### Tone
- professional
- modern but understated
- slightly warm
- tool-like rather than marketing-like

### Density
- medium density
- enough breathing room to feel premium
- compact enough to work well for tables and timesheets

### Surfaces
- light app background
- white or near-white cards/panels
- subtle borders
- soft shadows only where needed

### Border Radius
- small to medium radius only
- do not make everything pill-shaped

---

## Layout Rules

### App Shell
Use a stable application shell:
- left sidebar for primary navigation
- top bar for workspace, running timer, quick add, and user menu
- main content area with page title, actions, filters, and content

### Page Structure
Each page should generally follow:
1. page title + short supporting text
2. primary actions
3. filters or view controls if needed
4. primary content
5. secondary content only if it helps the task

### Pages
- Dashboard
- Time Entries / Timesheet
- Clients
- Projects
- Reports
- Settings

---

## Typography

Use a clean sans-serif with strong readability.
Examples:
- Inter
- Geist
- system-ui stack

### Type Hierarchy
- Page title: bold, large, clearly distinct
- Section title: medium-large, semibold
- Card title: medium, semibold
- Body text: regular
- Secondary text: muted but still readable
- Helper text: slightly smaller, readable contrast
- Numeric summaries: large, bold, tabular if possible

### Rules
- Avoid oversized headings
- Avoid tiny muted text
- Use tabular numerals for time and money if possible
- Keep line lengths comfortable in forms and detail views

---

## Color System

Use one accent color and a restrained neutral scale.

### Design Tokens
```css
:root {
  --bg: #f6f7f9;
  --surface: #ffffff;
  --surface-alt: #f1f3f6;

  --text: #171a1f;
  --text-muted: #596273;

  --border: #c7cfdb;
  --border-strong: #9aa6b2;

  --accent: #2563eb;
  --accent-hover: #1d4ed8;
  --accent-soft: #dbeafe;

  --success: #157347;
  --success-soft: #d1fadf;

  --warning: #b45309;
  --warning-soft: #fef0c7;

  --danger: #b42318;
  --danger-soft: #fee4e2;

  --focus: #1d4ed8;
}
```

### Color Usage Rules
- Accent is for primary actions, active navigation, selected states, and links
- Success/warning/danger are for meaning, never decoration
- Muted text must still be readable
- Do not rely on color alone for status or meaning

---

## Spacing Scale

Use a simple 8px-based system.

```css
:root {
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 24px;
  --space-6: 32px;
  --space-7: 40px;
  --space-8: 48px;
}
```

Rules:
- 16px is the default interior spacing
- 24px between larger sections
- 32px+ between major page zones
- avoid random spacing values unless truly necessary

---

## Radius, Borders, and Shadows

### Radius
- Inputs/buttons/small cards: 8px
- Larger cards/modals: 12px
- Avoid overly soft 20px+ radii everywhere

### Borders
Borders are a primary visual tool in TimeTrak.
Use them for:
- cards
- filters
- inputs
- tables
- panels

### Shadows
Use sparingly:
- none or very soft shadow for standard cards
- slightly stronger shadow for modals/dropdowns only

---

## Core Components

## Buttons

### Types
- Primary
- Secondary
- Tertiary / Ghost
- Danger

### Rules
- Primary button used once per area if possible
- Secondary buttons should remain clearly visible
- Ghost buttons only for low-emphasis actions
- Icon-only buttons must include accessible labels
- Minimum comfortable hit area

### Examples
- Primary: Start Timer, Save Entry
- Secondary: Cancel, Export CSV
- Danger: Delete Entry

---

## Inputs and Forms

### Form Rules
- Every input gets a visible label
- Helper text for unusual or money/time formatting
- Validation errors shown inline and in text
- Required fields clearly marked
- Group related controls with fieldsets where useful

### Field Styling
- clear border
- visible hover/focus states
- sufficient padding
- no ultra-thin ghost inputs

### Rate Inputs
Rate fields should visually reinforce precision:
- currency prefix or suffix
- monospaced or tabular numeral feel if possible
- example helper text

---

## Tables

Tables are a core part of the product.
They should feel first-class, not like an afterthought.

### Use tables for
- time entries
- project/client summaries
- reports
- invoice-prep views

### Table Rules
- strong header row
- visible row separators
- hover state
- keyboard-focusable row actions
- right-align time and money columns
- left-align names and descriptions
- use consistent empty, loading, and filtered states

### Suggested Columns for Time Entries
- Date
- Client
- Project
- Task
- Description
- Start
- End
- Duration
- Billable
- Amount
- Actions

---

## Badges and Status

Use badges for:
- Billable / Non-billable
- Running
- Archived
- Draft

Rules:
- status must include text, not color alone
- keep badges simple and readable
- do not overuse badges

---

## Cards

Cards are useful, but not every screen needs card soup.

Use cards for:
- dashboard summary metrics
- project/client summary blocks
- timer widget
- compact panels

Do not use cards where:
- a table is clearer
- a simple section layout is cleaner

---

## Navigation

### Sidebar
Use simple navigation labels:
- Dashboard
- Time
- Clients
- Projects
- Reports
- Settings

### Active State
Use:
- accent text/icon
- accent-soft background
- weight change or left border

Do not rely only on color.

---

## Timer Widget

The timer widget is a signature part of the app.
It should feel intentional.

Show:
- current project/client
- elapsed time
- start/stop control
- optional description
- clear running state

Rules:
- very readable elapsed time
- primary action obvious
- cannot look like a decorative card
- should work beautifully with keyboard and HTMX partial updates

---

## Empty States

Good empty states make the app feel designed.

### Rules
- explain what the screen is for
- explain what to do next
- include one clear action
- avoid jokey filler copy

### Example
No time entries yet.
Track your first block of work to start building your timesheet.
[Add entry] [Start timer]

---

## Microcopy

Write like a real product, not a template.

### Good
- Start timer
- Stop timer
- Add time entry
- Billable amount this week
- No client selected
- Rate applies from this date
- Archived projects are hidden from new entries

### Avoid
- Optimize your workflow
- Unlock insights
- Supercharge productivity
- Seamlessly manage your business

---

## Accessibility

TimeTrak targets WCAG 2.2 AA. This is enforced, not aspirational.

### Required habits
- semantic HTML first
- keyboard-operable interactions
- visible focus states
- visible labels and instructions
- contrast-safe color choices
- no color-only status communication
- generous target sizes
- proper headings and table semantics

### Focus styling
```css
:focus-visible {
  outline: 3px solid var(--focus);
  outline-offset: 2px;
}
```

### Acceptance checks per screen
- Can complete the core flow with keyboard only
- Focus is always visible
- Form fields have labels
- Errors are specific and helpful
- Status is not conveyed by color alone
- Tables use proper headers
- Buttons/links have clear names
- Icon-only controls have accessible labels
- Interactive targets are comfortably sized

---

## HTMX Interaction Guidelines

Use HTMX where it improves speed without creating a fragile UI.

Good HTMX use cases:
- start/stop timer
- inline entry edits
- row updates
- filters
- modal forms
- pagination
- dashboard widgets

Rules:
- partial responses should preserve context
- after swaps, keyboard focus should remain sensible
- loading states should be clear
- success and error states should be communicated in text
- avoid turning the app into a pseudo-SPA

---

## Screen-by-Screen Guidance

## Dashboard
Prioritize:
- Today hours
- This week hours
- Billable this month
- Running timer
- Recent entries

Use a mix of:
- 3–4 summary cards
- one timer panel
- one recent entries table/list

## Time Entries
This should be the strongest screen in the app.

Prioritize:
- date range
- add entry
- start timer
- filters
- readable table

## Clients
List should be simple and businesslike.
Show:
- name
- active projects
- total tracked time
- status

## Projects
Show:
- linked client
- rate summary
- total time
- recent activity
- active status

## Reports
Focus on readability over flash.
Tables first.
Charts only if they clearly add value.

---

## Component Library Direction (Stage 2+)

The `establish-custom-component-library-foundation` change will formalize the following.
Anticipate this when writing new templates — avoid bespoke markup that will be hard to migrate.

### Planned scope
- design tokens: color, spacing, radius, typography, borders, shadows, focus
- base atoms: button, input, label, hint text, error text, badge, icon button
- form primitives: field wrapper, select, textarea, checkbox, switch, segmented controls
- table primitives: table shell, sortable header, empty state row, row actions
- layout primitives: page header, action bar, sidebar section, card shell, filter bar
- feedback primitives: toast/inline alert, empty state, loading shell, confirmation modal
- product-specific components: timer card, time entry row, money summary card, project/client summary blocks

### Visual goal
The component library should feel:
- calm
- practical
- data-first
- slightly warm
- distinct to TimeTrak — not a generic AI-SaaS template

---

## Component Identity (Stage 3)

The `sharpen-component-identity` change (accepted via
`openspec/specs/ui-component-identity/spec.md`) adds a small, load-bearing
set of authoring contracts on top of the tokens and partials system.
Components remain calm and tool-like — but each is *opinionated* and
recognizable within that register.

These rules are enforceable: a CSS audit test, the `/dev/showcase`
gallery, and PR review all check them. Violations block merge.

### Shape-language taxonomy

Three shapes, three semantics. Do not mix them.

| Shape | Token | Semantic |
|---|---|---|
| Pill (fully rounded) | `var(--radius-pill)` | Actions — buttons, timer control |
| Rectangle | `var(--radius-sm)` | Status / metadata — chips, badges, tags |
| Circle | `50%` | Presence dots — running indicator, avatar fallback |

A chip is never a pill. A button is never a rectangle. A new shape
requires a change proposal that amends
`ui-component-identity.Shape language taxonomy`.

### Two-weight border contract

Every surface edge is either:

- **1px solid `var(--color-border)`** — structure, at-rest (cards,
  inputs at rest, table horizontal dividers).
- **2px solid `var(--color-accent)` or `var(--color-danger)`** —
  state (focus, selection, running, error).

Nothing between 1px and 2px. No dashed, double, inset, or outset
borders. No shadow elevation as a substitute for a border.

### Numeric text contract

Every element that renders a duration, amount, rate, or integer count
MUST apply `font-variant-numeric: tabular-nums`. In tables, numeric
columns are marked with `.col-num` (or
`[data-col-kind="numeric"]`) and render right-aligned. In cards and
inline contexts they remain left-aligned but retain tabular numerals.

Examples requiring `tabular-nums`: timer elapsed `HH:MM:SS`, Duration
column, Hourly rate column, Amount column, dashboard `Billable this
week` figure.

### Accent rationing

The accent color family (`var(--color-accent*)`) is permitted only on
surfaces that answer a "which one?" question for the user:

1. The running-timer fill, 2px border, leading dot, and elapsed readout.
2. The focus ring.
3. The selected/focused table-row 2px inside-left edge rule.
4. The primary button fill, border, and hover (`.btn-primary`).
5. Link text and link hover (`a`, `a:hover`).
6. The active/current navigation item
   (`.nav a[aria-current="page"]` — accent-soft fill + accent text +
   accent left-edge rule).
7. Billable and running status chips (`.tt-chip-billable`,
   `.tt-chip-running`).
8. The running-entry card top border (reserved for the follow-on
   `sharpen-dashboard-and-empty-states` change).

Any other accent usage fails review and the CSS audit test. Secondary
buttons, chips that aren't `billable`/`running`, hover states, at-rest
inputs, table headers, and at-rest cards all use neutral tokens. The
underlying principle: accent answers "which one?" — spread it across
generic chrome and it stops answering anything.

### Timer as signature object

The timer is not a styled button — it is its own partial
(`partials/timer_control`) with a documented state machine
(`idle → running → idle`). Idle is a neutral pill with a leading
neutral dot. Running inverts: `var(--color-accent-soft)` fill, 2px
`var(--color-accent)` border, pulsing accent dot (static under
`prefers-reduced-motion: reduce`), tabular-nums elapsed time, and a
distinct `Stop` affordance that is visually *not* the same pill as the
idle start. The timer is the only surface in the app that uses accent
as a fill.

### Review checklist

Every UI-affecting PR is reviewed against these five questions. Cite
the specific item being addressed or consciously waived in the PR
description.

1. **Shape** — does the component use the correct shape from the
   taxonomy? (`ui-component-identity.Shape language taxonomy`)
2. **Border weight** — does every border conform to the two-weight
   contract (1px structure / 2px state)?
   (`ui-component-identity.Two-weight border contract`)
3. **Numerics** — does every duration, amount, rate, or count render
   with `tabular-nums`? (`ui-component-identity.Numeric text contract`)
4. **Accent** — is the accent color consumed only on an allow-listed
   surface? (`ui-component-identity.Accent rationing`)
5. **State coverage** — does every state (default, hover, focused,
   selected, error, empty, running where applicable) render in
   `/dev/showcase`? (`ui-component-identity.Component identity review
   checklist`)

The `/dev/showcase` index renders this checklist above the gallery so
reviewers can cross-reference live components against the contract.

---

## Visual Anti-Patterns to Avoid

- too many gradients
- giant glassmorphism panels
- oversaturated accent colors
- every section as a card
- weak table styling
- tiny icon buttons
- placeholder-gray text used as body copy
- low-contrast borders
- vague marketing copy inside the product
- dashboards with no meaningful data structure
