// TimeTrak minimal JS: theme toggle + HTMX focus helpers. No framework.
(function () {
  const THEME_KEY = "timetrak.theme";
  const root = document.documentElement;

  function applyTheme(theme) {
    root.setAttribute("data-theme", theme);
  }

  const stored = localStorage.getItem(THEME_KEY) || "system";
  applyTheme(stored);

  document.addEventListener("click", function (e) {
    const btn = e.target.closest("[data-theme-set]");
    if (!btn) return;
    const next = btn.getAttribute("data-theme-set");
    localStorage.setItem(THEME_KEY, next);
    applyTheme(next);
  });

  // HTMX: after a swap, move focus to [data-focus-after-swap] within the swapped node.
  document.body.addEventListener("htmx:afterSwap", function (evt) {
    const target = evt.detail.target;
    if (!target) return;
    const focusEl = target.querySelector("[data-focus-after-swap]") || target.querySelector("[autofocus]");
    if (focusEl && typeof focusEl.focus === "function") {
      focusEl.focus();
    }
  });

  // Scope-toggle for the rate form: a [data-scope-select] shows/hides its
  // sibling [data-scope-target] field groups by matching value. The no-JS
  // fallback leaves all groups visible so the form still submits correctly.
  function applyScopeToggle(select) {
    const form = select.closest("form") || document;
    const targets = form.querySelectorAll("[data-scope-target]");
    targets.forEach(function (el) {
      el.hidden = el.getAttribute("data-scope-target") !== select.value;
    });
  }
  document.addEventListener("change", function (e) {
    const select = e.target.closest("[data-scope-select]");
    if (!select) return;
    applyScopeToggle(select);
  });
  // Initialize on load and after any HTMX swap (the form may be re-rendered OOB).
  function initScopeToggles(root) {
    (root || document).querySelectorAll("[data-scope-select]").forEach(applyScopeToggle);
  }
  document.addEventListener("DOMContentLoaded", function () { initScopeToggles(); });
  document.body.addEventListener("htmx:afterSwap", function (evt) {
    initScopeToggles(evt.detail.target);
  });

  // Timer elapsed updater: elements with [data-timer-started-at] tick once per second.
  function fmtElapsed(seconds) {
    const s = Math.max(0, Math.floor(seconds));
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    const sec = s % 60;
    return String(h).padStart(2, "0") + ":" + String(m).padStart(2, "0") + ":" + String(sec).padStart(2, "0");
  }
  setInterval(function () {
    document.querySelectorAll("[data-timer-started-at]").forEach(function (el) {
      const startedIso = el.getAttribute("data-timer-started-at");
      const started = Date.parse(startedIso);
      if (isNaN(started)) return;
      const elapsed = (Date.now() - started) / 1000;
      el.textContent = fmtElapsed(elapsed);
    });
  }, 1000);
})();
