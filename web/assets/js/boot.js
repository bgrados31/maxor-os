// Runs before the first paint (a tiny blocking script: the CSP forbids inline code). It settles the
// appearance, the language and whether things may move, so nothing flashes or jumps later.
(function () {
  var d = document.documentElement;
  function get(k) { try { return localStorage.getItem(k); } catch (e) { return null; } }

  var theme = get("maxor-theme");
  if (theme === "light" || theme === "dark") d.setAttribute("data-theme", theme);

  d.classList.add("js");
  if (!window.matchMedia || !matchMedia("(prefers-reduced-motion: reduce)").matches) d.classList.add("motion");

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
