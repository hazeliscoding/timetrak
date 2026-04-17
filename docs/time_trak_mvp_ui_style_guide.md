# TimeTrak MVP UI Style Guide

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
Design around those objects, not abstract “insights.”

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

### Recommended MVP Pages
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

### Suggested Tokens
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
- invoice-prep views later

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

## Accessibility Baseline

TimeTrak MVP should target WCAG 2.2 AA.

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

### Accessibility acceptance checks per screen
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
