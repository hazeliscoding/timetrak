---
name: tt-add-template
description: Add a server-rendered Go template page and/or HTMX partial under web/templates/ that respects TimeTrak's UI direction (calm, data-first), wires the right *-changed HX-Trigger events, preserves focus-after-swap, and meets WCAG 2.2 AA. Use when implementing a UI-affecting task.
license: MIT
compatibility: TimeTrak stdlib html/template + HTMX. Templates auto-loaded by internal/shared/templates.
metadata:
  author: timetrak
  version: "1.0"
---

Create a new page template (under `web/templates/<domain>/`) and/or HTMX partial (under `web/templates/partials/`) following the patterns landed in the bootstrap change. Safe to invoke mid-`openspec-apply-change` for any task that touches the rendered UI.

**Steps**

1. **Decide page vs. partial vs. both**
   - **Page**: full document, declares `title`, `shell`, `content` blocks. Lives at `web/templates/<domain>/<name>.html`.
   - **Partial**: HTMX swap target. Defines a single `{{define "<name>"}}...{{end}}` block. Lives at `web/templates/partials/<name>.html`.
   - Inline edit / row update / list item → partial. New screen → page (often referencing one or more partials).

2. **Read one peer template for tone**
   Read `web/templates/clients/index.html` (page) and `web/templates/partials/client_row.html` (partial). Match: section/heading hierarchy, `class="card"`, `class="row-between"`, `class="stack"`, `class="muted"`, `class="num"`, `class="btn btn-primary"`, `class="btn btn-ghost"`, `class="badge"`, `class="empty card"`, `class="flash flash-error"`.

3. **Page skeleton**

   ```html
   {{define "title"}}<Domain> · TimeTrak{{end}}
   {{define "shell"}}{{template "app-shell" .}}{{end}}
   {{define "content"}}
   <section class="stack">
     <div class="row-between">
       <h1><Heading></h1>
       <!-- filters / view-switchers go here -->
     </div>

     <!-- create form (if applicable) in a card -->

     {{if not .<Items>}}
       <div class="empty card">
         <h2>No <items> yet</h2>
         <p class="muted">Domain-specific copy explaining the next step.</p>
       </div>
     {{else}}
       <table class="table" aria-label="<Items>">
         <caption class="sr-only"><Items> in this workspace</caption>
         <thead>
           <tr>
             <th scope="col">...</th>
             <th scope="col" class="num">...</th>
             <th scope="col"><span class="sr-only">Actions</span></th>
           </tr>
         </thead>
         <tbody id="<items>-tbody" aria-live="polite">
           {{range .<Items>}}{{template "<item>_row" (dict "CSRFToken" $.CSRFToken "<Item>" .)}}{{end}}
         </tbody>
       </table>
     {{end}}
   </section>
   {{end}}
   ```

4. **Partial skeleton**

   ```html
   {{define "<name>"}}
   <tr id="<item>-{{.<Item>.ID}}">
     {{if .Edit}}
       <td colspan="N">
         <form hx-patch="/<resource>/{{.<Item>.ID}}"
               hx-target="#<item>-{{.<Item>.ID}}"
               hx-swap="outerHTML"
               class="row">
           <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
           <div class="field" style="flex:1;margin-bottom:0">
             <label class="sr-only" for="edit-name-{{.<Item>.ID}}">Name</label>
             <input id="edit-name-{{.<Item>.ID}}" name="name" type="text"
                    value="{{.<Item>.Name}}" required autofocus data-focus-after-swap>
           </div>
           <button type="submit" class="btn btn-primary">Save</button>
           <button type="button" class="btn"
                   hx-get="/<resource>/{{.<Item>.ID}}/row"
                   hx-target="#<item>-{{.<Item>.ID}}"
                   hx-swap="outerHTML">Cancel</button>
         </form>
       </td>
     {{else}}
       <td><strong>{{.<Item>.Name}}</strong></td>
       <!-- ... -->
       <td class="row" style="justify-content:flex-end">
         <button type="button" class="btn btn-ghost"
                 hx-get="/<resource>/{{.<Item>.ID}}/edit"
                 hx-target="#<item>-{{.<Item>.ID}}"
                 hx-swap="outerHTML">Edit</button>
       </td>
     {{end}}
   </tr>
   {{end}}
   ```

5. **HTMX wiring (binding)**

   - Server emits `HX-Trigger` on state changes. Allowed events:
     - `timer-changed` (timer start/stop)
     - `entries-changed` (entry CRUD; also fires on timer changes)
     - `clients-changed`, `projects-changed`, `rates-changed`
   - Pages that should refresh on a peer event use:
     ```html
     <div hx-get="/dashboard/summary"
          hx-trigger="timer-changed from:body, entries-changed from:body"
          hx-target="this" hx-swap="outerHTML">…</div>
     ```
   - Destructive actions: `hx-confirm="Archive <X>?"` (native `confirm()`). For flows needing focus trap, use `partials/confirm_dialog.html` (`<dialog>`).
   - Focus restoration after swap: set `data-focus-after-swap` on the element to receive focus. The handler in `web/static/js/app.js` does the restore on `htmx:afterSwap`.

6. **Template funcs available** (registered in `internal/shared/templates`):
   `dict`, `seq`, `formatDate`, `formatTime`, `formatDuration`, `formatMinor`, `iso`, `add`, `sub`. Use `formatMinor` to display money — never divide by 100 in templates.

7. **Accessibility checklist (WCAG 2.2 AA, binding)**

   - Every input has a visible `<label>` (or `class="sr-only"` if visually redundant in a table row).
   - Status badges include `aria-label` (don't rely on color alone).
   - Tables have `<caption>` (use `class="sr-only"` if redundant) and `<th scope="col">`.
   - Live-update regions use `aria-live="polite"` (or `assertive` for errors).
   - Errors use `role="alert"`.
   - Focus is preserved across swaps via `data-focus-after-swap`.
   - All interactive controls have a visible focus state (CSS handled by tokens; don't override).
   - Target sizes ≥ 24×24 CSS px.

8. **UI direction (binding)**
   - Calm, data-first. No blob art, no oversized hero, no random gradients, no vague productivity copy.
   - One restrained accent color; prefer borders + spacing over shadows.
   - Domain-specific copy: `Start timer`, `Billable this week`, `Client rate`, `Running entry` — never "Boost productivity!".

9. **Verify**
   ```bash
   make run     # then load the page; templates parse at startup
   ```
   Template parse errors fail fast on boot. If you added a new partial referenced by an existing page, restart the server.

**Guardrails**
- Never inline `<script>` in a template. Add to `web/static/js/app.js` if absolutely necessary.
- Never introduce a SPA framework or client-state library — the model is server-rendered + HTMX.
- Never use color as the sole status signal.
- Always include CSRF token (`<input type="hidden" name="csrf_token" value="{{.CSRFToken}}">`) on POST/PUT/PATCH/DELETE forms.

**Fluid Workflow Integration**
Invoke when `tasks.md` (in an OpenSpec change) lists "add page for X", "extract row partial", "wire HX-Trigger Y", or any task that says "render". Pair with `tt-scaffold-domain` if the route doesn't yet exist.
