// Runs before the first paint (a tiny blocking script: the CSP forbids inline code). It settles the
// appearance, the language and whether things may move, so nothing flashes or jumps later.
(function () {
  var d = document.documentElement;
  function get(k) { try { return localStorage.getItem(k); } catch (e) { return null; } }

  var theme = get("maxor-theme");
  if (theme === "light" || theme === "dark") d.setAttribute("data-theme", theme);

  // A theme picked for the site on an earlier visit (themes.js keeps it): only known tokens, only colours.
  try {
    var saved = JSON.parse(localStorage.getItem("maxor-site-theme") || "null");
    var ok = /^(#[0-9a-f]{6}|color-mix\(in oklab, #[0-9a-f]{6} \d{1,3}%, transparent\))$/i;
    if (saved && /^[a-z0-9-]{1,40}$/.test(saved.id) && (saved.mode === "dark" || saved.mode === "light")) {
      for (var k in saved.vars) if (/^--[a-z0-9-]{1,10}$/.test(k) && ok.test(saved.vars[k])) d.style.setProperty(k, saved.vars[k]);
      d.setAttribute("data-theme", saved.mode);
      d.setAttribute("data-site-theme", saved.id);
    }
  } catch (e) { /* nothing saved, or storage blocked */ }

  d.classList.add("js");
  var still = window.matchMedia && matchMedia("(prefers-reduced-motion: reduce)").matches;
  if (!still) d.classList.add("motion");

  // The first page of a visit boots like the system: the mark and a thin bar, under a second. Once per tab.
  var seen = null;
  try { seen = sessionStorage.getItem("maxor-booted"); sessionStorage.setItem("maxor-booted", "1"); } catch (e) { seen = "1"; }
  if (!still && !seen) d.classList.add("booting");

  var q = null;
  try { q = new URLSearchParams(location.search).get("lang"); } catch (e) { /* old browser */ }
  var lang = q === "es" || q === "en" ? q : get("maxor-lang");
  if (lang !== "es" && lang !== "en") {
    var first = (navigator.languages && navigator.languages[0]) || navigator.language || "es";
    lang = /^es\b/i.test(first) ? "es" : "en";
  }
  d.setAttribute("data-lang", lang);
  // The page is written in Spanish; English swaps the text in. Hide it for that instant only, and never
  // for longer than 900 ms even if the swap fails.
  if (lang !== "es") {
    d.classList.add("i18n-wait");
    setTimeout(function () { d.classList.remove("i18n-wait"); }, 900);
  }
})();
