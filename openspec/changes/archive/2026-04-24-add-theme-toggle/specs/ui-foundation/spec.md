## ADDED Requirements

### Requirement: FOUC-prevention head script is the single sanctioned inline script

The `base.html` layout SHALL carry exactly one inline `<script>` element, placed before any `<link rel="stylesheet">` in `<head>`, whose sole purpose is to read the user's stored theme preference from `localStorage` under the key `timetrak.theme` and apply it to `<html>` as a `data-theme` attribute before first paint. The script MUST be ≤30 lines, MUST be wrapped in a `try { ... } catch (e) { }` so a localStorage-denied environment falls through cleanly, and MUST NOT reference any symbol outside its own IIFE scope.

No other inline `<script>` element is permitted anywhere in the product's template tree. All other client-side behavior lives in `web/static/js/app.js` (or a successor external script) and is subject to the normal caching / CSP story.

#### Scenario: First paint renders the stored theme

- **WHEN** a user who previously selected `dark` returns to any page
- **THEN** the inline head-script reads `localStorage.timetrak.theme` and sets `<html data-theme="dark">` before the browser paints the first frame
- **AND** the user MUST NOT see a flash of the default `system` theme

#### Scenario: localStorage is denied

- **WHEN** the user's browser denies `localStorage.getItem` (strict privacy modes, private-browsing flavors)
- **THEN** the inline head-script's `try { ... } catch` swallows the error
- **AND** the document falls through to its default `data-theme` attribute (`system`) without a render error

#### Scenario: Contributor attempts to add another inline script

- **WHEN** a proposed change adds an inline `<script>` anywhere in `web/templates/`
- **THEN** the review MUST block the change unless the change also amends this requirement
- **AND** the acceptable alternative is to extend `web/static/js/app.js` or ship a new external script
