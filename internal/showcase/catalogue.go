// Package showcase is TimeTrak's developer-only component + token
// reference surface. It is mounted at /dev/showcase and is reachable ONLY
// when APP_ENV=dev. The package is belt-and-suspenders-gated: the route
// is not registered outside dev, AND each handler short-circuits to 404
// at request time if APP_ENV drifts.
//
// The showcase renders the real partials via the application's template
// loader — it never re-implements markup. Copy-ready snippets are loaded
// from colocated fixture files via embed.FS; see snippets.go.
//
// This file owns the catalogue metadata: ComponentEntry / TokenEntry
// slices and the grandfather list used by the coverage test.
package showcase

import (
	"time"

	"github.com/google/uuid"
)

// parseTime panics on failure; callers use well-known literals.
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic("showcase: parseTime: " + err.Error())
	}
	return t
}

// ComponentEntry documents a single reusable partial.
//
// The coverage test asserts every non-grandfathered file under
// web/templates/partials/ has exactly one ComponentEntry whose
// PartialName matches the file stem.
type ComponentEntry struct {
	ID          string             // URL slug + anchor id (stable across renders)
	Name        string             // Display name (e.g. "flash")
	PartialName string             // Template block name invoked by ExecuteTemplate
	SourcePath  string             // web/templates/partials/<name>.html
	SpecRef     string             // openspec/specs/... reference
	Purpose     string             // One-paragraph prose description
	DictKeys    []DictKeyDoc       // Documented keys the partial consumes
	Examples    []ComponentExample // Live permutations
	A11yNotes   []string           // Accessibility obligations verbatim from README
}

// DictKeyDoc documents one key in a partial's dict contract.
type DictKeyDoc struct {
	Name     string
	Required bool
	Default  string // Empty when Required
	Note     string
}

// ComponentExample is one live render + snippet pair for a ComponentEntry.
//
// Dict is the exact payload used both to render the live partial via
// template.ExecuteTemplate AND to drive the copy-ready snippet. One
// struct, two consumers — drift fails at render time.
//
// Most partials consume a map[string]any (built via the dict template
// func by callers). A small minority (currently flash) consume a slice
// directly; those examples set Dict to the slice itself.
type ComponentExample struct {
	ID          string // Per-example anchor suffix
	Label       string // Visible label (variant/state name)
	PartialName string // Block name for this specific example (usually same as entry's PartialName)
	Dict        any    // Payload passed to the partial
	SnippetID   string // Lookup key for LookupSnippet
}

// TokenEntry documents a single CSS custom property.
type TokenEntry struct {
	ID      string // URL slug + anchor id
	Name    string // CSS custom property name, e.g. "--color-accent"
	Family  string // "semantic-color" | "primitive-ramp" | "spacing" | ...
	Role    string // Documented semantic role / usage guidance
	Preview string // "swatch" | "sizing-bar" | "sample-text" | "motion-demo" | "radius" | "shadow" | "z" | "breakpoint"
}

// GrandfatheredPartials enumerates partial file stems that MUST NOT be
// required to appear in ComponentEntries by the coverage test. Initially
// empty — add an entry ONLY when a partial is intentionally excluded
// (e.g. short-lived experimental partial) and document why here.
//
// Note: the empty list is load-bearing. The coverage test fails when a
// new partial lands without a showcase entry; adding it here without
// justification defeats the test.
var GrandfatheredPartials = []string{}

// demoRuleID is a deterministic UUID used by the rate_row / rates_table
// examples so snippet and live render show stable ids.
var demoRuleID = uuid.MustParse("11111111-2222-3333-4444-555555555555")

// demoWorkspaceID / demoClientID / demoProjectID are seeded UUIDs for
// row partials. They are never persisted; they exist only as stable
// string values in the rendered HTML.
var (
	demoClientID  = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	demoProjectID = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000001")
	demoEntryID   = uuid.MustParse("cccccccc-0000-0000-0000-000000000001")
)

// fakeCSRFToken is shown verbatim in example forms. It is NEVER a valid
// CSRF token for any real session — the showcase runs authenticated and
// the middleware validates the real cookie on any actual mutation. The
// displayed value exists only so the partial HTML renders shape-complete.
const fakeCSRFToken = "showcase-demo-csrf-token"

// ComponentEntries drives the component catalogue page.
//
// IMPORTANT: when adding a new partial under web/templates/partials/,
// add a matching entry here OR (rarely) add the file stem to
// GrandfatheredPartials with a justification comment. The
// TestComponentCatalogueCoverage test enforces this.
var ComponentEntries = []ComponentEntry{
	{
		ID:          "brandmark",
		Name:        "brandmark",
		PartialName: "brandmark",
		SourcePath:  "web/templates/partials/brandmark.html",
		SpecRef:     "openspec/specs/ui-partials/spec.md",
		Purpose:     "TimeTrak product wordmark. Inline SVG that consumes only currentColor and var(--color-accent); no raw colour values and no new tokens. The default render (Size=md, Decorative=false) is used in the app header; Decorative=true is reserved for surfaces that already name the product in adjacent text. See docs/timetrak_brand_guidelines.md for usage rules.",
		DictKeys: []DictKeyDoc{
			{Name: "Size", Required: false, Default: "md", Note: "Height token — \"md\" = var(--space-5), \"sm\" = var(--space-4). Width auto-scales from viewBox."},
			{Name: "Decorative", Required: false, Default: "false", Note: "When false, emits role=\"img\" + <title>TimeTrak</title>. When true, emits aria-hidden=\"true\" and omits <title>."},
		},
		Examples: []ComponentExample{
			{ID: "default", Label: "Default (header use)", PartialName: "brandmark", SnippetID: "brandmark.default", Dict: map[string]any{
				"Size": "md", "Decorative": false,
			}},
			{ID: "sm-decorative", Label: "Small, decorative", PartialName: "brandmark", SnippetID: "brandmark.sm_decorative", Dict: map[string]any{
				"Size": "sm", "Decorative": true,
			}},
		},
		A11yNotes: []string{
			"Non-decorative render announces as graphic named \"TimeTrak\" via role=\"img\" + <title>.",
			"Decorative render is aria-hidden; only use adjacent to text that already names the product.",
			"When wrapped in an anchor (app header), the anchor inherits the global :focus-visible outline; no component-scoped override.",
			"Fill/stroke reference only currentColor and var(--color-accent); status is never conveyed by the mark's colour.",
		},
	},
	{
		ID:          "form-field",
		Name:        "form_field",
		PartialName: "form_field",
		SourcePath:  "web/templates/partials/form_field.html",
		SpecRef:     "openspec/specs/ui-partials/spec.md",
		Purpose:     "Visible-label text-style input with optional hint and optional inline error binding. Covers type=text|email|password|date|time|number|url|tel.",
		DictKeys: []DictKeyDoc{
			{Name: "ID", Required: true, Note: "Matches <label for> and input id."},
			{Name: "Name", Required: true, Note: "Form field name."},
			{Name: "Label", Required: true, Note: "Visible label text."},
			{Name: "Type", Required: false, Default: "text", Note: "Any text-like HTML input type."},
			{Name: "Value", Required: false, Default: "", Note: "Current value."},
			{Name: "Required", Required: false, Default: "false", Note: "Emits required + aria-required=\"true\"."},
			{Name: "Autofocus", Required: false, Default: "false", Note: "Emits autofocus."},
			{Name: "Autocomplete", Required: false, Default: "", Note: "Passes through to autocomplete attribute."},
			{Name: "Placeholder", Required: false, Default: "", Note: "Passes through to placeholder attribute."},
			{Name: "Hint", Required: false, Default: "", Note: "Renders a <span class=\"hint\">."},
			{Name: "ErrorID", Required: false, Default: "", Note: "When set with Invalid, links via aria-describedby."},
			{Name: "Invalid", Required: false, Default: "false", Note: "Emits aria-invalid=\"true\"."},
			{Name: "Inputmode", Required: false, Default: "", Note: "Passes through to inputmode."},
			{Name: "Maxlength", Required: false, Default: "", Note: "When non-empty emits maxlength."},
			{Name: "Width", Required: false, Default: "", Note: "Inline style width (e.g. 7rem)."},
		},
		Examples: []ComponentExample{
			{ID: "default", Label: "Default", PartialName: "form_field", SnippetID: "form_field.default", Dict: map[string]any{
				"ID": "demo-email", "Name": "email", "Label": "Email", "Type": "email",
			}},
			{ID: "with-hint", Label: "With hint", PartialName: "form_field", SnippetID: "form_field.with_hint", Dict: map[string]any{
				"ID": "demo-rate", "Name": "hourly", "Label": "Hourly rate",
				"Hint":      "Enter as a decimal; stored as integer minor units.",
				"Inputmode": "decimal", "Width": "8rem",
			}},
			{ID: "invalid", Label: "Invalid + error linkage", PartialName: "form_field", SnippetID: "form_field.invalid", Dict: map[string]any{
				"ID": "demo-name", "Name": "name", "Label": "Name", "Required": true,
				"Invalid": true, "ErrorID": "demo-name-error", "Value": "",
			}},
		},
		A11yNotes: []string{
			"<label for> is the label source; the control is the focus target.",
			"Error wiring is the caller's responsibility via ErrorID + a sibling form_errors summary.",
		},
	},
	{
		ID:          "form-errors",
		Name:        "form_errors",
		PartialName: "form_errors",
		SourcePath:  "web/templates/partials/form_errors.html",
		SpecRef:     "openspec/specs/ui-partials/spec.md",
		Purpose:     "Top-of-form error summary. Focusable so callers can set data-focus-after-swap on the summary to send keyboard and screen-reader focus to the error.",
		DictKeys: []DictKeyDoc{
			{Name: "ID", Required: true, Note: "Stable id associated controls can reference via aria-describedby."},
			{Name: "Message", Required: true, Note: "Error copy. If empty the partial renders nothing."},
			{Name: "NoFocus", Required: false, Default: "false", Note: "When truthy suppresses data-focus-after-swap."},
		},
		Examples: []ComponentExample{
			{ID: "default", Label: "Default (focus after swap)", PartialName: "form_errors", SnippetID: "form_errors.default", Dict: map[string]any{
				"ID": "demo-form-error", "Message": "Name is required.",
			}},
			{ID: "no-focus", Label: "NoFocus variant", PartialName: "form_errors", SnippetID: "form_errors.no_focus", Dict: map[string]any{
				"ID": "demo-form-error-2", "Message": "Rate must be greater than zero.", "NoFocus": true,
			}},
		},
		A11yNotes: []string{
			"The summary is the focus target after a validation swap.",
			"Associated controls should set aria-invalid=\"true\" and aria-describedby=\"<ID>\".",
			"Status is conveyed by text + role=\"alert\"; never color alone.",
		},
	},
	{
		ID:          "status-chip",
		Name:        "status_chip",
		PartialName: "status_chip",
		SourcePath:  "web/templates/partials/status_chip.html",
		SpecRef:     "openspec/specs/ui-component-identity/spec.md",
		Purpose:     "Rectangular status/metadata indicator. Pills are reserved for actions; chips are rectangles. Every kind pairs color with a glyph or explicit label so status is never color-only.",
		DictKeys: []DictKeyDoc{
			{Name: "Kind", Required: true, Note: "One of: billable, non-billable, running, draft, archived, warning."},
			{Name: "Label", Required: true, Note: "Human-readable text."},
			{Name: "Variant", Required: true, Note: "filled (soft accent/warning fill) or outlined (neutral border)."},
			{Name: "Glyph", Required: false, Default: "kind-specific", Note: "Leading glyph. State-conveying kinds (running, archived, draft, warning) render a default glyph when omitted."},
		},
		Examples: []ComponentExample{
			{ID: "billable", Label: "billable · filled", PartialName: "status_chip", SnippetID: "status_chip.billable", Dict: map[string]any{
				"Kind": "billable", "Label": "Billable", "Variant": "filled",
			}},
			{ID: "non-billable", Label: "non-billable · outlined", PartialName: "status_chip", SnippetID: "status_chip.non_billable", Dict: map[string]any{
				"Kind": "non-billable", "Label": "Non-billable", "Variant": "outlined",
			}},
			{ID: "running", Label: "running · filled (glyph = ●)", PartialName: "status_chip", SnippetID: "status_chip.running", Dict: map[string]any{
				"Kind": "running", "Label": "Running", "Variant": "filled",
			}},
			{ID: "archived", Label: "archived · outlined (glyph = ⊘)", PartialName: "status_chip", SnippetID: "status_chip.archived", Dict: map[string]any{
				"Kind": "archived", "Label": "Archived", "Variant": "outlined",
			}},
			{ID: "draft", Label: "draft · outlined (glyph = ○)", PartialName: "status_chip", SnippetID: "status_chip.draft", Dict: map[string]any{
				"Kind": "draft", "Label": "Draft", "Variant": "outlined",
			}},
			{ID: "warning", Label: "warning · filled (glyph = ⚠)", PartialName: "status_chip", SnippetID: "status_chip.warning", Dict: map[string]any{
				"Kind": "warning", "Label": "No rate", "Variant": "filled",
			}},
		},
		A11yNotes: []string{
			"Status is never conveyed by color alone. State-conveying kinds (running, archived, draft, warning) pair color with a glyph; billable/non-billable rely on explicit label text.",
			"aria-label is set for running and archived so assistive tech announces state.",
			"Shape language: chips are rectangular (--radius-sm). Pills are reserved for actions (buttons, timer).",
		},
	},
	{
		ID:          "theme-switch",
		Name:        "theme_switch",
		PartialName: "theme_switch",
		SourcePath:  "web/templates/partials/theme_switch.html",
		SpecRef:     "openspec/specs/ui-partials/spec.md",
		Purpose:     "Segmented three-way control for light/dark/system theme. Single pill-shaped radiogroup; selected segment uses accent-soft fill + 2px accent inset edge to answer \"which theme is active?\". State persists to localStorage via the existing data-theme-set click hook in web/static/js/app.js; the FOUC-prevention head-script in base.html applies it synchronously on first paint.",
		DictKeys: []DictKeyDoc{
			{Name: "InitialSelected", Required: false, Default: "\"\"", Note: "One of light|dark|system. Empty = production case (JS synchronizes active state). Set = showcase case (pre-selects the matching segment)."},
		},
		Examples: []ComponentExample{
			{ID: "light-selected", Label: "Light selected", PartialName: "theme_switch", SnippetID: "theme_switch.light_selected", Dict: map[string]any{
				"InitialSelected": "light",
			}},
			{ID: "dark-selected", Label: "Dark selected", PartialName: "theme_switch", SnippetID: "theme_switch.dark_selected", Dict: map[string]any{
				"InitialSelected": "dark",
			}},
			{ID: "system-selected", Label: "System selected", PartialName: "theme_switch", SnippetID: "theme_switch.system_selected", Dict: map[string]any{
				"InitialSelected": "system",
			}},
		},
		A11yNotes: []string{
			"Wrapper is role=\"radiogroup\" with aria-label=\"Theme\"; segments are <button role=\"radio\"> with both aria-pressed (legacy JS hook) and aria-checked (correct radiogroup semantics).",
			"Glyphs are aria-hidden supplementary; the accessible name comes from the per-segment aria-label plus the visible label (or sr-only label under 720px).",
			"The selected segment consumes var(--color-accent-soft) + var(--color-accent) via an allow-listed entry in openspec/specs/ui-component-identity/spec.md (Accent rationing).",
		},
	},
	{
		ID:          "empty-state",
		Name:        "empty_state",
		PartialName: "empty_state",
		SourcePath:  "web/templates/partials/empty_state.html",
		SpecRef:     "openspec/specs/ui-partials/spec.md",
		Purpose:     "Copy-first empty block for lists, tables, and filtered results with no rows. No icon, no color-only meaning.",
		DictKeys: []DictKeyDoc{
			{Name: "Title", Required: true, Note: "<h2> copy."},
			{Name: "Body", Required: true, Note: "One-sentence explanation."},
			{Name: "ActionHref", Required: false, Default: "", Note: "If set, renders a primary-action link."},
			{Name: "ActionText", Required: false, Default: "", Note: "Label for the action. Required when ActionHref set."},
			{Name: "Live", Required: false, Default: "false", Note: "When true, sets aria-live=\"polite\" on the wrapper."},
		},
		Examples: []ComponentExample{
			{ID: "no-action", Label: "Without action", PartialName: "empty_state", SnippetID: "empty_state.no_action", Dict: map[string]any{
				"Title": "No entries match these filters",
				"Body":  "Try a different preset, widen the dates, or clear a filter.",
				"Live":  true,
			}},
			{ID: "with-action", Label: "With action", PartialName: "empty_state", SnippetID: "empty_state.with_action", Dict: map[string]any{
				"Title":      "No clients yet",
				"Body":       "Add your first client to begin tracking time.",
				"ActionHref": "/clients/new",
				"ActionText": "Add client",
			}},
		},
		A11yNotes: []string{
			"Title conveys meaning; body and optional action are supplementary.",
			"Live is a hint for peer-refresh consumers that the empty view may arrive via an HTMX swap.",
		},
	},
	{
		ID:          "flash",
		Name:        "flash",
		PartialName: "flash",
		SourcePath:  "web/templates/partials/flash.html",
		SpecRef:     "openspec/specs/ui-partials/spec.md",
		Purpose:     "Page-level toast list. Severity drives the ARIA role: success/info/warn use role=\"status\"; error uses role=\"alert\".",
		DictKeys: []DictKeyDoc{
			{Name: "(slice)", Required: true, Note: "Pass a slice of flash entries; each entry has .Kind (success|info|warn|error) and .Message."},
		},
		Examples: []ComponentExample{
			{ID: "success", Label: "success", PartialName: "flash", SnippetID: "flash.success",
				Dict: []flashEntry{{Kind: "success", Message: "Client created."}}},
			{ID: "info", Label: "info", PartialName: "flash", SnippetID: "flash.info",
				Dict: []flashEntry{{Kind: "info", Message: "Workspace switched."}}},
			{ID: "warn", Label: "warn", PartialName: "flash", SnippetID: "flash.warn",
				Dict: []flashEntry{{Kind: "warn", Message: "3 entries have no resolved rate."}}},
			{ID: "error", Label: "error", PartialName: "flash", SnippetID: "flash.error",
				Dict: []flashEntry{{Kind: "error", Message: "Could not save: invalid interval."}}},
		},
		A11yNotes: []string{
			"Status is conveyed via role + text + the flash-<kind> class; color is never the sole signal.",
			"Error severity uses role=\"alert\" for immediate announcement; success/info/warn use role=\"status\" + aria-live=\"polite\".",
		},
	},
	{
		ID:          "spinner",
		Name:        "spinner",
		PartialName: "spinner",
		SourcePath:  "web/templates/partials/spinner.html",
		SpecRef:     "openspec/specs/ui-partials/spec.md",
		Purpose:     "Inline loading indicator paired with HTMX hx-indicator. Does NOT pair with data-focus-after-swap — the eventual swap is the real completion signal.",
		DictKeys:    []DictKeyDoc{},
		Examples: []ComponentExample{
			{ID: "default", Label: "Default", PartialName: "spinner", SnippetID: "spinner.default", Dict: map[string]any{}},
		},
		A11yNotes: []string{
			"role=\"status\" + aria-live=\"polite\" + an sr-only \"Loading…\" label.",
			"Supplementary cue only — the eventual swap is the real completion signal.",
		},
	},
	{
		ID:          "pagination",
		Name:        "pagination",
		PartialName: "pagination",
		SourcePath:  "web/templates/partials/pagination.html",
		SpecRef:     "openspec/specs/ui-partials/spec.md",
		Purpose:     "Prev/next navigation for offset-paginated lists. Hidden when TotalPages ≤ 1.",
		DictKeys: []DictKeyDoc{
			{Name: "Page", Required: true, Note: "1-indexed current page."},
			{Name: "TotalPages", Required: true, Note: "Total page count (0 or 1 hides the nav)."},
			{Name: "PrevQuery", Required: true, Note: "Query string for the previous page."},
			{Name: "NextQuery", Required: true, Note: "Query string for the next page."},
		},
		Examples: []ComponentExample{
			{ID: "middle", Label: "Middle page", PartialName: "pagination", SnippetID: "pagination.middle", Dict: map[string]any{
				"Page": 3, "TotalPages": 7, "PrevQuery": "page=2", "NextQuery": "page=4",
			}},
			{ID: "first", Label: "First page", PartialName: "pagination", SnippetID: "pagination.first", Dict: map[string]any{
				"Page": 1, "TotalPages": 4, "PrevQuery": "", "NextQuery": "page=2",
			}},
			{ID: "last", Label: "Last page", PartialName: "pagination", SnippetID: "pagination.last", Dict: map[string]any{
				"Page": 5, "TotalPages": 5, "PrevQuery": "page=4", "NextQuery": "",
			}},
		},
		A11yNotes: []string{
			"Rendered in a <nav aria-label=\"Pagination\">.",
			"Prev/next anchors carry rel attributes and visible text.",
		},
	},
	{
		ID:          "confirm-dialog",
		Name:        "confirm_dialog",
		PartialName: "confirm_dialog",
		SourcePath:  "web/templates/partials/confirm_dialog.html",
		SpecRef:     "openspec/specs/ui-partials/spec.md",
		Purpose:     "Focus-trapped <dialog> for destructive actions that need side-effect copy. Currently UNUSED in the shipped app — every destructive delete uses native hx-confirm for MVP.",
		DictKeys: []DictKeyDoc{
			{Name: "ID", Required: true, Note: "Stable dialog id; referenced by aria-labelledby as <ID>-title."},
			{Name: "Title", Required: true, Note: "Dialog heading."},
			{Name: "Message", Required: true, Note: "Body copy describing consequences."},
			{Name: "Method", Required: true, Note: "HTTP method for the confirm button (post, delete, patch)."},
			{Name: "Action", Required: true, Note: "URL for hx-<method>."},
			{Name: "Target", Required: true, Note: "hx-target selector."},
			{Name: "Swap", Required: true, Note: "hx-swap value."},
			{Name: "ConfirmLabel", Required: true, Note: "Label on the destructive button."},
		},
		Examples: []ComponentExample{
			{ID: "default", Label: "Archive with consequences", PartialName: "confirm_dialog", SnippetID: "confirm_dialog.default", Dict: map[string]any{
				"ID": "demo-confirm", "Title": "Archive client?",
				"Message": "This client has 3 active projects. Archiving preserves time entries; projects become read-only.",
				"Method":  "post", "Action": "/clients/demo/archive",
				"Target": "#client-row-demo", "Swap": "outerHTML",
				"ConfirmLabel": "Archive client",
			}},
		},
		A11yNotes: []string{
			"aria-labelledby on the dialog wires the title as accessible name.",
			"The destructive button carries data-focus-after-swap so focus lands predictably after submit.",
		},
	},
	{
		ID:          "client-row",
		Name:        "client_row",
		PartialName: "client_row",
		SourcePath:  "web/templates/partials/client_row.html",
		SpecRef:     "openspec/specs/ui-partials/spec.md",
		Purpose:     "Row renderer for the clients table. Root is <tr id=\"client-row-<uuid>\"> for OOB swap. Emits clients-changed on handler mutations.",
		DictKeys: []DictKeyDoc{
			{Name: "Client", Required: true, Note: "Client view with ID, Name, ContactEmail, ProjectCount, IsArchived."},
			{Name: "Edit", Required: true, Note: "When true, renders the inline edit form."},
			{Name: "CSRFToken", Required: true, Note: "CSRF token for any mutation form inside the row."},
			{Name: "Error", Required: false, Default: "", Note: "Error message shown in the edit form."},
		},
		Examples: []ComponentExample{
			{ID: "read", Label: "Read mode", PartialName: "client_row", SnippetID: "client_row.read", Dict: map[string]any{
				"Client": demoClient(false), "Edit": false, "CSRFToken": fakeCSRFToken,
			}},
			{ID: "archived", Label: "Archived", PartialName: "client_row", SnippetID: "client_row.archived", Dict: map[string]any{
				"Client": demoClient(true), "Edit": false, "CSRFToken": fakeCSRFToken,
			}},
			{ID: "edit", Label: "Edit mode", PartialName: "client_row", SnippetID: "client_row.edit", Dict: map[string]any{
				"Client": demoClient(false), "Edit": true, "CSRFToken": fakeCSRFToken,
			}},
		},
		A11yNotes: []string{
			"Archived state uses a badge with visible text, not color alone.",
			"Edit-mode inputs carry data-focus-after-swap on the first control.",
		},
	},
	{
		ID:          "project-row",
		Name:        "project_row",
		PartialName: "project_row",
		SourcePath:  "web/templates/partials/project_row.html",
		SpecRef:     "openspec/specs/ui-partials/spec.md",
		Purpose:     "Row renderer for the projects table. Root is <tr id=\"project-row-<uuid>\">. Emits projects-changed on mutations.",
		DictKeys: []DictKeyDoc{
			{Name: "Project", Required: true, Note: "Project view with ID, Name, ClientName, Code, DefaultBillable, IsArchived."},
			{Name: "Edit", Required: true, Note: "When true, renders the inline edit form."},
			{Name: "CSRFToken", Required: true, Note: "CSRF token."},
			{Name: "Error", Required: false, Default: "", Note: "Error message shown in the edit form."},
		},
		Examples: []ComponentExample{
			{ID: "read", Label: "Read mode, billable", PartialName: "project_row", SnippetID: "project_row.read", Dict: map[string]any{
				"Project": demoProject(true, false), "Edit": false, "CSRFToken": fakeCSRFToken,
			}},
			{ID: "archived", Label: "Archived", PartialName: "project_row", SnippetID: "project_row.archived", Dict: map[string]any{
				"Project": demoProject(false, true), "Edit": false, "CSRFToken": fakeCSRFToken,
			}},
		},
		A11yNotes: []string{
			"Billable / archived states carry badge text; color alone never conveys status.",
		},
	},
	{
		ID:          "entry-row",
		Name:        "entry_row",
		PartialName: "entry_row",
		SourcePath:  "web/templates/partials/entry_row.html",
		SpecRef:     "openspec/specs/ui-partials/spec.md",
		Purpose:     "Row renderer for the time-entries table. Root is <tr id=\"entry-row-<uuid>\">. Emits entries-changed on mutations; consumes tracking_error on integrity failures.",
		DictKeys: []DictKeyDoc{
			{Name: "Entry", Required: true, Note: "Entry view with ID, ClientName, ProjectID, ProjectName, Description, StartedAt, EndedAt, DurationSeconds, IsBillable."},
			{Name: "Edit", Required: true, Note: "When true, renders the inline edit form with split start_date / start_time / end_date / end_time inputs."},
			{Name: "Projects", Required: false, Default: "nil", Note: "Projects list shown in the edit select."},
			{Name: "CSRFToken", Required: true, Note: "CSRF token."},
			{Name: "Error", Required: false, Default: "", Note: "Fallback error message (used when ErrorCode is empty)."},
			{Name: "ErrorCode", Required: false, Default: "", Note: "Tracking error taxonomy code for tracking_error partial."},
			{Name: "Timezone", Required: false, Default: "UTC", Note: "Active workspace ReportingTimezone. Edit mode prefills the split date+time inputs in this zone; empty falls back to UTC."},
		},
		Examples: []ComponentExample{
			{ID: "read", Label: "Read mode, billable closed", PartialName: "entry_row", SnippetID: "entry_row.read", Dict: map[string]any{
				"Entry": demoEntry(true, true), "Edit": false, "CSRFToken": fakeCSRFToken,
			}},
			{ID: "running", Label: "Running (no EndedAt)", PartialName: "entry_row", SnippetID: "entry_row.running", Dict: map[string]any{
				"Entry": demoEntry(true, false), "Edit": false, "CSRFToken": fakeCSRFToken,
			}},
			{ID: "edit-utc", Label: "Edit mode (UTC workspace)", PartialName: "entry_row", SnippetID: "entry_row.edit_utc", Dict: map[string]any{
				"Entry": demoEntry(true, true), "Edit": true, "CSRFToken": fakeCSRFToken, "Timezone": "UTC",
				"Projects": []map[string]any{{"ID": demoProjectID.String(), "ClientName": "Acme Co", "Name": "Website redesign"}},
			}},
			{ID: "edit-ny", Label: "Edit mode (America/New_York — local clock)", PartialName: "entry_row", SnippetID: "entry_row.edit_ny", Dict: map[string]any{
				"Entry": demoEntry(true, true), "Edit": true, "CSRFToken": fakeCSRFToken, "Timezone": "America/New_York",
				"Projects": []map[string]any{{"ID": demoProjectID.String(), "ClientName": "Acme Co", "Name": "Website redesign"}},
			}},
		},
		A11yNotes: []string{
			"Running / billable states use badges with visible text.",
			"Edit form uses data-focus-after-swap on the first invalid or the first control otherwise.",
			"Edit mode renders four native date/time inputs (start_date, start_time, end_date, end_time) prefilled in the workspace's ReportingTimezone — no raw ISO strings.",
		},
	},
	{
		ID:          "rate-row",
		Name:        "rate_row",
		PartialName: "rate_row",
		SourcePath:  "web/templates/partials/rate_row.html",
		SpecRef:     "openspec/specs/ui-partials/spec.md",
		Purpose:     "Row renderer for the rate rules table. Root is <tr id=\"rate-row-<uuid>\">. Emits rates-changed on mutations. Referenced rules disable Delete with a visible hint.",
		DictKeys: []DictKeyDoc{
			{Name: "Rule", Required: true, Note: "Rate rule view with ID, Level, ClientID/Name, ProjectID/Name, CurrencyCode, HourlyRateMinor, EffectiveFrom, EffectiveTo, ReferencedByCount."},
			{Name: "Edit", Required: true, Note: "When true, renders the inline edit form (end-date only)."},
			{Name: "CSRFToken", Required: true, Note: "CSRF token."},
			{Name: "Error", Required: false, Default: "", Note: "Error message shown in the edit form or as inline badge."},
			{Name: "AttemptedTo", Required: false, Default: "", Note: "Echoed value of effective_to on edit error."},
		},
		Examples: []ComponentExample{
			{ID: "workspace", Label: "Workspace default, open-ended", PartialName: "rate_row", SnippetID: "rate_row.workspace", Dict: map[string]any{
				"Rule": demoRule("workspace", 0), "Edit": false, "CSRFToken": fakeCSRFToken,
			}},
			{ID: "client-referenced", Label: "Client rule, referenced", PartialName: "rate_row", SnippetID: "rate_row.client_referenced", Dict: map[string]any{
				"Rule": demoRule("client", 7), "Edit": false, "CSRFToken": fakeCSRFToken,
			}},
		},
		A11yNotes: []string{
			"Disabled Delete button carries aria-describedby pointing at the \"Referenced by N entries\" hint.",
			"Edit button carries data-focus-after-swap to move focus to the inline form's date input.",
		},
	},
	{
		ID:          "rate-form",
		Name:        "rate_form",
		PartialName: "rate_form",
		SourcePath:  "web/templates/partials/rate_form.html",
		SpecRef:     "openspec/specs/rates/spec.md",
		Purpose:     "New-rate-rule form. Supports hx-swap-oob=\"true\" via .OOB so a successful create can swap both the form and the table in one response.",
		DictKeys: []DictKeyDoc{
			{Name: "Form", Required: true, Note: "Form view with Scope, ClientID, ProjectID, CurrencyCode, HourlyDecimal, EffectiveFrom, EffectiveTo, Error."},
			{Name: "Clients", Required: true, Note: "Client list shown in the per-client select."},
			{Name: "Projects", Required: true, Note: "Project list shown in the per-project select."},
			{Name: "CSRFToken", Required: true, Note: "CSRF token."},
			{Name: "OOB", Required: false, Default: "false", Note: "When true, renders hx-swap-oob=\"true\" on the form."},
		},
		Examples: []ComponentExample{
			{ID: "empty", Label: "Empty", PartialName: "rate_form", SnippetID: "rate_form.empty", Dict: map[string]any{
				"Form": map[string]any{
					"Scope": "workspace", "CurrencyCode": "USD",
					"HourlyDecimal": "", "EffectiveFrom": "2026-01-01", "EffectiveTo": "",
				},
				"Clients": []map[string]any{}, "Projects": []map[string]any{},
				"CSRFToken": fakeCSRFToken, "OOB": false,
			}},
			{ID: "error", Label: "With validation error", PartialName: "rate_form", SnippetID: "rate_form.error", Dict: map[string]any{
				"Form": map[string]any{
					"Scope": "workspace", "CurrencyCode": "USD",
					"HourlyDecimal": "0", "EffectiveFrom": "2026-01-01", "EffectiveTo": "",
					"Error": "Hourly rate must be greater than zero.",
				},
				"Clients": []map[string]any{}, "Projects": []map[string]any{},
				"CSRFToken": fakeCSRFToken, "OOB": false,
			}},
		},
		A11yNotes: []string{
			"form_errors summary at the top carries data-focus-after-swap on validation replay.",
			"Scope-dependent fields (client / project selects) toggle with hidden attribute, not display:none.",
		},
	},
	{
		ID:          "rates-table",
		Name:        "rates_table",
		PartialName: "rates_table",
		SourcePath:  "web/templates/partials/rates_table.html",
		SpecRef:     "openspec/specs/rates/spec.md",
		Purpose:     "Full #rates-table region (list + empty state). Rendered in response to any rate mutation.",
		DictKeys: []DictKeyDoc{
			{Name: "Rules", Required: true, Note: "Slice of rate rule views; empty slice renders the empty state."},
			{Name: "CSRFToken", Required: true, Note: "CSRF token passed to each rate_row."},
		},
		Examples: []ComponentExample{
			{ID: "empty", Label: "Empty", PartialName: "rates_table", SnippetID: "rates_table.empty", Dict: map[string]any{
				"Rules": []any{}, "CSRFToken": fakeCSRFToken,
			}},
			{ID: "populated", Label: "Populated", PartialName: "rates_table", SnippetID: "rates_table.populated", Dict: map[string]any{
				"Rules":     []any{demoRule("workspace", 0), demoRule("client", 2)},
				"CSRFToken": fakeCSRFToken,
			}},
		},
		A11yNotes: []string{
			"Table has a caption (sr-only) and scope=\"col\" on every header.",
			"Empty state delegates to empty_state (Live=true) for HTMX swap announcement.",
		},
	},
	{
		ID:          "timer-control",
		Name:        "timer_control",
		PartialName: "timer_control",
		SourcePath:  "web/templates/partials/timer_control.html",
		SpecRef:     "openspec/specs/ui-component-identity/spec.md",
		Purpose:     "The app's signature pill. Idle renders a start-entry form; running renders a single accent-filled pill with 2px accent border, pulsing dot, and tabular-nums elapsed time. Emits timer-changed and entries-changed on start/stop.",
		DictKeys: []DictKeyDoc{
			{Name: "Running", Required: false, Default: "nil", Note: "When non-nil, renders the running pill (ClientName, ProjectName, Description, StartedAt)."},
			{Name: "Projects", Required: false, Default: "nil", Note: "Project options when the control is idle."},
			{Name: "CSRFToken", Required: true, Note: "CSRF token."},
			{Name: "Error", Required: false, Default: "", Note: "Fallback error copy (used when ErrorCode is empty)."},
			{Name: "ErrorCode", Required: false, Default: "", Note: "Tracking error taxonomy code; delegates to tracking_error."},
		},
		Examples: []ComponentExample{
			{ID: "idle", Label: "Idle (start form)", PartialName: "timer_control", SnippetID: "timer_control.idle", Dict: map[string]any{
				"Running": nil,
				"Projects": []map[string]any{
					{"ID": demoProjectID.String(), "ClientName": "Acme Co", "Name": "Website redesign"},
				},
				"CSRFToken": fakeCSRFToken,
			}},
			{ID: "running", Label: "Running (accent pill + pulsing dot)", PartialName: "timer_control", SnippetID: "timer_control.running", Dict: map[string]any{
				"Running":   demoRunning(),
				"CSRFToken": fakeCSRFToken,
			}},
		},
		A11yNotes: []string{
			"aria-live=\"polite\" on the section wrapper so state changes are announced.",
			"Running dot is aria-hidden; state is conveyed by the accent fill, elapsed readout, and the distinct Stop control.",
			"Pulsing dot halts under prefers-reduced-motion via the global animation-none rule, leaving a static accent dot.",
			"Stop button uses .btn-ghost so it is visually distinct from the idle .btn-primary Start pill.",
		},
	},
	{
		ID:          "tracking-error",
		Name:        "tracking_error",
		PartialName: "tracking_error",
		SourcePath:  "web/templates/partials/tracking_error.html",
		SpecRef:     "openspec/specs/tracking/spec.md",
		Purpose:     "Shared inline error region for tracking integrity failures (active-timer conflict, cross-workspace project, invalid interval). Consumed by timer_control and entry_row.",
		DictKeys: []DictKeyDoc{
			{Name: "ErrorCode", Required: true, Note: "Stable taxonomy code (e.g. tracking.active_timer)."},
			{Name: "Message", Required: true, Note: "Domain-specific copy for humans."},
		},
		Examples: []ComponentExample{
			{ID: "active-timer", Label: "active_timer", PartialName: "tracking_error", SnippetID: "tracking_error.active_timer", Dict: map[string]any{
				"ErrorCode": "tracking.active_timer",
				"Message":   "You already have a timer running. Stop it before starting a new one.",
			}},
			{ID: "invalid-interval", Label: "invalid_interval", PartialName: "tracking_error", SnippetID: "tracking_error.invalid_interval", Dict: map[string]any{
				"ErrorCode": "tracking.invalid_interval",
				"Message":   "Ended at must be after started at.",
			}},
		},
		A11yNotes: []string{
			"role=\"alert\" + tabindex=\"-1\" + data-focus-after-swap so screen readers announce the failure and focus lands on the error.",
			"Icon + text; never color alone (WCAG 2.2 AA, SC 1.4.1).",
		},
	},
	{
		ID:          "dashboard-summary",
		Name:        "dashboard_summary",
		PartialName: "dashboard_summary",
		SourcePath:  "web/templates/partials/dashboard_summary.html",
		SpecRef:     "openspec/specs/reporting/spec.md",
		Purpose:     "Dashboard totals card row. Swapped in response to timer-changed / entries-changed from body. Listens; does not emit.",
		DictKeys: []DictKeyDoc{
			{Name: "TodayTotalSeconds", Required: true, Note: "Integer seconds."},
			{Name: "TodayBillableSeconds", Required: true, Note: "Integer seconds."},
			{Name: "TodayNonBillableSeconds", Required: true, Note: "Integer seconds."},
			{Name: "WeekTotalSeconds", Required: true, Note: "Integer seconds."},
			{Name: "WeekBillableSeconds", Required: true, Note: "Integer seconds."},
			{Name: "WeekNonBillableSeconds", Required: true, Note: "Integer seconds."},
			{Name: "WeekEstimatedBillable", Required: false, Default: "nil", Note: "map[currency]int64 of estimated minor units."},
			{Name: "EntriesWithoutRate", Required: false, Default: "0", Note: "Count of billable entries with no resolved rate."},
		},
		Examples: []ComponentExample{
			{ID: "default", Label: "With billable + no-rate warning", PartialName: "dashboard_summary", SnippetID: "dashboard_summary.default", Dict: map[string]any{
				"TodayTotalSeconds": int64(12600), "TodayBillableSeconds": int64(10800), "TodayNonBillableSeconds": int64(1800),
				"WeekTotalSeconds": int64(72000), "WeekBillableSeconds": int64(54000), "WeekNonBillableSeconds": int64(18000),
				"WeekEstimatedBillable": map[string]int64{"USD": 675000},
				"EntriesWithoutRate":    2,
			}},
		},
		A11yNotes: []string{
			"Section uses aria-label=\"Summary\" as a landmark label.",
			"No-rate warning uses a badge with text; color is never the only signal.",
		},
	},
	{
		ID:          "report-summary",
		Name:        "report_summary",
		PartialName: "report_summary",
		SourcePath:  "web/templates/partials/report_summary.html",
		SpecRef:     "openspec/specs/reporting/spec.md",
		Purpose:     "Report totals card row (total / billable / non-billable / estimated by currency).",
		DictKeys: []DictKeyDoc{
			{Name: "TotalSeconds", Required: true, Note: "Integer seconds."},
			{Name: "BillableSeconds", Required: true, Note: "Integer seconds."},
			{Name: "NonBillableSeconds", Required: true, Note: "Integer seconds."},
			{Name: "EstimatedByCurrency", Required: false, Default: "nil", Note: "map[currency]int64 of minor units."},
		},
		Examples: []ComponentExample{
			{ID: "default", Label: "Default", PartialName: "report_summary", SnippetID: "report_summary.default", Dict: map[string]any{
				"TotalSeconds": int64(144000), "BillableSeconds": int64(108000), "NonBillableSeconds": int64(36000),
				"EstimatedByCurrency": map[string]int64{"USD": 1350000, "EUR": 450000},
			}},
		},
		A11yNotes: []string{
			"Section uses aria-label=\"Totals\" as a landmark label.",
		},
	},
	{
		ID:          "reports-partial-results",
		Name:        "reports.partial.results",
		PartialName: "reports.partial.results",
		SourcePath:  "web/templates/partials/report_results.html",
		SpecRef:     "openspec/specs/reporting/spec.md",
		Purpose:     "Reports results region. Composes report_summary with grouped totals (day / client / project) and cascades to reports.partial.empty when a grouping has no rows.",
		DictKeys: []DictKeyDoc{
			{Name: "Report", Required: true, Note: "Report view with Totals, ByDay/ByClient/ByProject slices, NoRateCount."},
			{Name: "Grouping", Required: true, Note: "day | client | project."},
			{Name: "SortedCurrencies", Required: false, Default: "nil", Note: "Ordered currency codes for the grand total block."},
		},
		Examples: []ComponentExample{
			{ID: "by-day", Label: "Grouping=day", PartialName: "reports.partial.results", SnippetID: "reports_partial_results.by_day", Dict: demoReportResults("day")},
			{ID: "by-client-empty", Label: "Grouping=client, empty", PartialName: "reports.partial.results", SnippetID: "reports_partial_results.by_client_empty", Dict: demoReportResults("client-empty")},
		},
		A11yNotes: []string{
			"Tables have a <caption> and scope=\"col\" on every header.",
			"No-rate warning uses a badge with text; color is never the only signal.",
		},
	},
	{
		ID:          "reports-partial-empty",
		Name:        "reports.partial.empty",
		PartialName: "reports.partial.empty",
		SourcePath:  "web/templates/partials/report_empty.html",
		SpecRef:     "openspec/specs/reporting/spec.md",
		Purpose:     "Empty-state shim for the reports results region. Delegates to empty_state with Live=true so the swap is announced.",
		DictKeys:    []DictKeyDoc{},
		Examples: []ComponentExample{
			{ID: "default", Label: "Default", PartialName: "reports.partial.empty", SnippetID: "reports_partial_empty.default", Dict: map[string]any{}},
		},
		A11yNotes: []string{
			"Live=true on the underlying empty_state so the empty view is announced after an HTMX swap.",
		},
	},
}

// TokenEntries drives the token catalogue page.
//
// Order in the template: semantic aliases first, scale tokens second,
// primitive ramps last with a visible note that components MUST NOT
// consume them directly.
var TokenEntries = []TokenEntry{
	// ---- Semantic colors ----
	{ID: "color-bg", Name: "--color-bg", Family: "semantic-color", Role: "Page background.", Preview: "swatch"},
	{ID: "color-surface", Name: "--color-surface", Family: "semantic-color", Role: "Cards, tables, inputs, buttons at rest.", Preview: "swatch"},
	{ID: "color-surface-alt", Name: "--color-surface-alt", Family: "semantic-color", Role: "Table headers, hover rows, muted surfaces.", Preview: "swatch"},
	{ID: "color-text", Name: "--color-text", Family: "semantic-color", Role: "Body text. Target ≥4.5:1 on surfaces.", Preview: "sample-text"},
	{ID: "color-text-muted", Name: "--color-text-muted", Family: "semantic-color", Role: "Secondary / helper text. Target ≥4.5:1 on --color-surface.", Preview: "sample-text"},
	{ID: "color-border", Name: "--color-border", Family: "semantic-color", Role: "Default 1px separators. Non-text target ≥3:1.", Preview: "swatch"},
	{ID: "color-border-strong", Name: "--color-border-strong", Family: "semantic-color", Role: "Input, button, and emphatic borders.", Preview: "swatch"},
	{ID: "color-accent", Name: "--color-accent", Family: "semantic-color", Role: "Primary brand signal. Non-text ≥3:1; #fff text on it ≥4.5:1.", Preview: "swatch"},
	{ID: "color-accent-hover", Name: "--color-accent-hover", Family: "semantic-color", Role: "Hover state for accent fills.", Preview: "swatch"},
	{ID: "color-accent-soft", Name: "--color-accent-soft", Family: "semantic-color", Role: "Low-emphasis accent fill (billable badge, nav-current).", Preview: "swatch"},
	{ID: "color-focus", Name: "--color-focus", Family: "semantic-color", Role: "Single focus-ring color. ≥3:1 on every surface.", Preview: "swatch"},
	{ID: "color-success", Name: "--color-success", Family: "semantic-color", Role: "Confirmed success (pair with text or icon).", Preview: "swatch"},
	{ID: "color-success-soft", Name: "--color-success-soft", Family: "semantic-color", Role: "Success fill behind --color-success.", Preview: "swatch"},
	{ID: "color-warning", Name: "--color-warning", Family: "semantic-color", Role: "Warning (pair with text or icon).", Preview: "swatch"},
	{ID: "color-warning-soft", Name: "--color-warning-soft", Family: "semantic-color", Role: "Warning fill behind --color-warning.", Preview: "swatch"},
	{ID: "color-danger", Name: "--color-danger", Family: "semantic-color", Role: "Destructive / error (pair with text or icon).", Preview: "swatch"},
	{ID: "color-danger-soft", Name: "--color-danger-soft", Family: "semantic-color", Role: "Danger fill behind --color-danger.", Preview: "swatch"},
	{ID: "color-info", Name: "--color-info", Family: "semantic-color", Role: "Neutral informational status.", Preview: "swatch"},
	{ID: "color-info-soft", Name: "--color-info-soft", Family: "semantic-color", Role: "Info fill behind --color-info.", Preview: "swatch"},

	// ---- Spacing ----
	{ID: "space-1", Name: "--space-1", Family: "spacing", Role: "4px — fine separator.", Preview: "sizing-bar"},
	{ID: "space-2", Name: "--space-2", Family: "spacing", Role: "8px — tight inline rhythm.", Preview: "sizing-bar"},
	{ID: "space-3", Name: "--space-3", Family: "spacing", Role: "12px — compact stack.", Preview: "sizing-bar"},
	{ID: "space-4", Name: "--space-4", Family: "spacing", Role: "16px — default stack rhythm.", Preview: "sizing-bar"},
	{ID: "space-5", Name: "--space-5", Family: "spacing", Role: "24px — card padding.", Preview: "sizing-bar"},
	{ID: "space-6", Name: "--space-6", Family: "spacing", Role: "32px — section rhythm.", Preview: "sizing-bar"},
	{ID: "space-7", Name: "--space-7", Family: "spacing", Role: "40px — page-level rhythm.", Preview: "sizing-bar"},
	{ID: "space-8", Name: "--space-8", Family: "spacing", Role: "48px — large section gap.", Preview: "sizing-bar"},

	// ---- Radius ----
	{ID: "radius-sm", Name: "--radius-sm", Family: "radius", Role: "8px — controls.", Preview: "radius"},
	{ID: "radius-md", Name: "--radius-md", Family: "radius", Role: "12px — cards.", Preview: "radius"},

	// ---- Typography ----
	{ID: "font-sans", Name: "--font-sans", Family: "typography", Role: "Default UI font stack.", Preview: "sample-text"},
	{ID: "font-mono", Name: "--font-mono", Family: "typography", Role: "Tabular / code font stack.", Preview: "sample-text"},

	// ---- Motion ----
	{ID: "motion-duration-fast", Name: "--motion-duration-fast", Family: "motion", Role: "120ms. Hover / focus affordances.", Preview: "motion-demo"},
	{ID: "motion-duration-normal", Name: "--motion-duration-normal", Family: "motion", Role: "200ms. Component state transitions.", Preview: "motion-demo"},
	{ID: "motion-easing-standard", Name: "--motion-easing-standard", Family: "motion", Role: "cubic-bezier(0.2, 0, 0, 1). Standard easing.", Preview: "motion-demo"},

	// ---- Elevation ----
	{ID: "shadow-none", Name: "--shadow-none", Family: "elevation", Role: "Borders-first default.", Preview: "shadow"},
	{ID: "shadow-sm", Name: "--shadow-sm", Family: "elevation", Role: "Subtle lift.", Preview: "shadow"},
	{ID: "shadow-md", Name: "--shadow-md", Family: "elevation", Role: "Emphatic lift (rare).", Preview: "shadow"},

	// ---- Z-index ----
	{ID: "z-base", Name: "--z-base", Family: "z-index", Role: "0 — default stacking.", Preview: "z"},
	{ID: "z-sticky", Name: "--z-sticky", Family: "z-index", Role: "10 — sticky headers.", Preview: "z"},
	{ID: "z-dropdown", Name: "--z-dropdown", Family: "z-index", Role: "100 — dropdowns / popovers.", Preview: "z"},
	{ID: "z-modal", Name: "--z-modal", Family: "z-index", Role: "1000 — modal dialogs.", Preview: "z"},
	{ID: "z-toast", Name: "--z-toast", Family: "z-index", Role: "1100 — flash toasts (top of stack).", Preview: "z"},

	// ---- Breakpoints ----
	{ID: "bp-sm", Name: "--bp-sm", Family: "breakpoint", Role: "640px.", Preview: "breakpoint"},
	{ID: "bp-md", Name: "--bp-md", Family: "breakpoint", Role: "960px.", Preview: "breakpoint"},
	{ID: "bp-lg", Name: "--bp-lg", Family: "breakpoint", Role: "1280px.", Preview: "breakpoint"},

	// ---- Primitive ramps (rendered in the CLEARLY MARKED section) ----
	{ID: "neutral-0", Name: "--neutral-0", Family: "primitive-ramp", Role: "Neutral ramp anchor (lightest).", Preview: "swatch"},
	{ID: "neutral-50", Name: "--neutral-50", Family: "primitive-ramp", Role: "Neutral ramp.", Preview: "swatch"},
	{ID: "neutral-100", Name: "--neutral-100", Family: "primitive-ramp", Role: "Neutral ramp.", Preview: "swatch"},
	{ID: "neutral-200", Name: "--neutral-200", Family: "primitive-ramp", Role: "Neutral ramp.", Preview: "swatch"},
	{ID: "neutral-300", Name: "--neutral-300", Family: "primitive-ramp", Role: "Neutral ramp.", Preview: "swatch"},
	{ID: "neutral-400", Name: "--neutral-400", Family: "primitive-ramp", Role: "Neutral ramp.", Preview: "swatch"},
	{ID: "neutral-500", Name: "--neutral-500", Family: "primitive-ramp", Role: "Neutral ramp.", Preview: "swatch"},
	{ID: "neutral-600", Name: "--neutral-600", Family: "primitive-ramp", Role: "Neutral ramp.", Preview: "swatch"},
	{ID: "neutral-700", Name: "--neutral-700", Family: "primitive-ramp", Role: "Neutral ramp.", Preview: "swatch"},
	{ID: "neutral-800", Name: "--neutral-800", Family: "primitive-ramp", Role: "Neutral ramp.", Preview: "swatch"},
	{ID: "neutral-900", Name: "--neutral-900", Family: "primitive-ramp", Role: "Neutral ramp anchor (darkest).", Preview: "swatch"},

	{ID: "accent-50", Name: "--accent-50", Family: "primitive-ramp", Role: "Accent ramp.", Preview: "swatch"},
	{ID: "accent-100", Name: "--accent-100", Family: "primitive-ramp", Role: "Accent ramp.", Preview: "swatch"},
	{ID: "accent-200", Name: "--accent-200", Family: "primitive-ramp", Role: "Accent ramp.", Preview: "swatch"},
	{ID: "accent-300", Name: "--accent-300", Family: "primitive-ramp", Role: "Accent ramp.", Preview: "swatch"},
	{ID: "accent-400", Name: "--accent-400", Family: "primitive-ramp", Role: "Accent ramp.", Preview: "swatch"},
	{ID: "accent-500", Name: "--accent-500", Family: "primitive-ramp", Role: "Accent ramp.", Preview: "swatch"},
	{ID: "accent-600", Name: "--accent-600", Family: "primitive-ramp", Role: "Accent ramp.", Preview: "swatch"},
	{ID: "accent-700", Name: "--accent-700", Family: "primitive-ramp", Role: "Accent ramp.", Preview: "swatch"},
	{ID: "accent-800", Name: "--accent-800", Family: "primitive-ramp", Role: "Accent ramp.", Preview: "swatch"},
	{ID: "accent-900", Name: "--accent-900", Family: "primitive-ramp", Role: "Accent ramp.", Preview: "swatch"},

	{ID: "red-500", Name: "--red-500", Family: "primitive-ramp", Role: "Severity anchor.", Preview: "swatch"},
	{ID: "red-600", Name: "--red-600", Family: "primitive-ramp", Role: "Severity anchor.", Preview: "swatch"},
	{ID: "red-soft", Name: "--red-soft", Family: "primitive-ramp", Role: "Severity fill.", Preview: "swatch"},
	{ID: "amber-500", Name: "--amber-500", Family: "primitive-ramp", Role: "Severity anchor.", Preview: "swatch"},
	{ID: "amber-600", Name: "--amber-600", Family: "primitive-ramp", Role: "Severity anchor.", Preview: "swatch"},
	{ID: "amber-soft", Name: "--amber-soft", Family: "primitive-ramp", Role: "Severity fill.", Preview: "swatch"},
	{ID: "green-500", Name: "--green-500", Family: "primitive-ramp", Role: "Severity anchor.", Preview: "swatch"},
	{ID: "green-600", Name: "--green-600", Family: "primitive-ramp", Role: "Severity anchor.", Preview: "swatch"},
	{ID: "green-soft", Name: "--green-soft", Family: "primitive-ramp", Role: "Severity fill.", Preview: "swatch"},
}

// --- helpers to build demo view structs used by row partials ---

type flashEntry struct {
	Kind    string
	Message string
}

func demoClient(archived bool) map[string]any {
	return map[string]any{
		"ID":           demoClientID.String(),
		"Name":         "Acme Co",
		"ContactEmail": "billing@acme.test",
		"ProjectCount": 3,
		"IsArchived":   archived,
	}
}

func demoProject(billable, archived bool) map[string]any {
	return map[string]any{
		"ID":              demoProjectID.String(),
		"Name":            "Website redesign",
		"ClientName":      "Acme Co",
		"Code":            "ACME-WEB",
		"DefaultBillable": billable,
		"IsArchived":      archived,
	}
}

func demoEntry(billable, closed bool) map[string]any {
	started := parseTime("2026-04-20T14:00:00Z")
	entry := map[string]any{
		"ID":              demoEntryID.String(),
		"ProjectID":       demoProjectID.String(),
		"ClientName":      "Acme Co",
		"ProjectName":     "Website redesign",
		"Description":     "Homepage hero polish",
		"StartedAt":       started,
		"DurationSeconds": int64(5400),
		"IsBillable":      billable,
	}
	if closed {
		entry["EndedAt"] = parseTime("2026-04-20T15:30:00Z")
	}
	return entry
}

func demoRule(level string, referencedBy int) map[string]any {
	rule := map[string]any{
		"ID":                demoRuleID.String(),
		"Level":             level,
		"CurrencyCode":      "USD",
		"HourlyRateMinor":   int64(12500),
		"EffectiveFrom":     parseTime("2026-01-01T00:00:00Z"),
		"ReferencedByCount": referencedBy,
	}
	switch level {
	case "client":
		rule["ClientID"] = demoClientID.String()
		rule["ClientName"] = "Acme Co"
	case "project":
		rule["ProjectID"] = demoProjectID.String()
		rule["ProjectName"] = "Website redesign"
	}
	return rule
}

func demoRunning() map[string]any {
	return map[string]any{
		"ClientName":  "Acme Co",
		"ProjectName": "Website redesign",
		"Description": "Homepage hero polish",
		"StartedAt":   parseTime("2026-04-20T14:00:00Z"),
	}
}

func demoReportResults(shape string) map[string]any {
	totals := map[string]any{
		"TotalSeconds": int64(144000), "BillableSeconds": int64(108000), "NonBillableSeconds": int64(36000),
		"EstimatedByCurrency": map[string]int64{"USD": 1350000},
	}
	base := map[string]any{
		"Report": map[string]any{
			"Totals":      totals,
			"NoRateCount": 0,
		},
		"Grouping":         "day",
		"SortedCurrencies": []string{"USD"},
	}
	report := base["Report"].(map[string]any)
	switch shape {
	case "day":
		report["ByDay"] = []map[string]any{
			{"Label": "Mon 2026-04-20", "TotalSeconds": int64(28800), "BillableSeconds": int64(21600), "NonBillableSeconds": int64(7200)},
			{"Label": "Tue 2026-04-21", "TotalSeconds": int64(25200), "BillableSeconds": int64(18000), "NonBillableSeconds": int64(7200)},
		}
	case "client-empty":
		base["Grouping"] = "client"
		report["ByClient"] = []map[string]any{}
	}
	return base
}
