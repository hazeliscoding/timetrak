// TimeTrak minimal JS: theme toggle + HTMX focus helpers. No framework.
(function () {
  const THEME_KEY = "timetrak.theme";
  const root = document.documentElement;

  function applyTheme(theme) {
    root.setAttribute("data-theme", theme);
    syncThemePressed(theme);
  }

  function syncThemePressed(theme) {
    document.querySelectorAll("[data-theme-set]").forEach(function (btn) {
      btn.setAttribute("aria-pressed", btn.getAttribute("data-theme-set") === theme ? "true" : "false");
    });
  }

  const stored = localStorage.getItem(THEME_KEY) || "system";
  applyTheme(stored);
  document.addEventListener("DOMContentLoaded", function () { syncThemePressed(stored); });

  document.addEventListener("click", function (e) {
    const btn = e.target.closest("[data-theme-set]");
    if (!btn) return;
    const next = btn.getAttribute("data-theme-set");
    localStorage.setItem(THEME_KEY, next);
    applyTheme(next);
  });

  // HTMX focus-after-swap convention.
  // After HTMX swaps a node in, focus the first [data-focus-after-swap] inside
  // the swap target (falling back to any [autofocus]). Apply this attribute
  // only to intent swaps per the focus-flow catalogue in
  // openspec/changes/polish-mvp-ui-for-accessibility-and-consistency/design.md;
  // passive peer-refresh swaps (dashboard summary, rates-changed, etc.) MUST
  // NOT carry the attribute so focus stays where the user was.
  document.body.addEventListener("htmx:afterSwap", function (evt) {
    const target = evt.detail.target;
    if (!target) return;
    const focusEl = target.querySelector("[data-focus-after-swap]") || target.querySelector("[autofocus]");
    if (focusEl && typeof focusEl.focus === "function") {
      focusEl.focus();
    }
  });

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
  function initScopeToggles(root) {
    (root || document).querySelectorAll("[data-scope-select]").forEach(applyScopeToggle);
  }
  document.addEventListener("DOMContentLoaded", function () { initScopeToggles(); });
  document.body.addEventListener("htmx:afterSwap", function (evt) {
    initScopeToggles(evt.detail.target);
  });

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
