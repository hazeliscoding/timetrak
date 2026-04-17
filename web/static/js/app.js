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
