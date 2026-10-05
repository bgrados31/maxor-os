// The official themes (assets/data/themes.json, synced from themes/) and the whole-site theming: picking a
// theme in the tour or in the terminal repaints the page itself, the way `maxor theme apply` repaints the
// system. The choice is remembered; boot.js paints it again before the first frame of the next visit.

const root = document.documentElement;
const STORE = "maxor-site-theme";
let loading;

export function loadThemes() {
  loading ??= fetch(new URL("../data/themes.json", import.meta.url), { credentials: "omit" })
    .then((r) => { if (!r.ok) throw new Error(`themes.json: ${r.status}`); return r.json(); });
  return loading;
}

export async function themeById(id) {
  return (await loadThemes()).find((t) => t.id === id) ?? null;
}

function hex(c) {
  const n = parseInt(c.slice(1), 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}
export function mix(a, b, w) {
  const [x, y] = [hex(a), hex(b)];
  return "#" + x.map((v, i) => Math.round(v * w + y[i] * (1 - w)).toString(16).padStart(2, "0")).join("");
}

// The site's tokens (site.css :root) from a theme. Every pair the page uses is a pair the theme already
// guarantees (text on background, accent on its own text colour).
function siteVars(th) {
  const dark = th.mode === "dark";
  const vars = {
    "--bg": th.bg, "--bg2": mix(th.bg, dark ? "#000000" : "#ffffff", 0.75),
    "--s": th.s, "--s2": th.s2, "--fg": th.fg, "--mu": th.mu, "--ac": th.ac, "--ac2": th.ac2, "--on": th.on,
    "--line": `color-mix(in oklab, ${th.fg} 10%, transparent)`,
    "--line2": `color-mix(in oklab, ${th.fg} 18%, transparent)`,
    "--glass": `color-mix(in oklab, ${th.s} ${dark ? 62 : 72}%, transparent)`,
    "--ok": dark ? "#6ee7b7" : "#0f7a4f", "--bad": dark ? "#ff7b7b" : "#c4262e",
  };
  // The hero stays night; on a dark theme it borrows the accent too.
  if (dark) vars["--hero-ac"] = th.ac;
  return vars;
}

const KEYS = ["--bg", "--bg2", "--s", "--s2", "--fg", "--mu", "--ac", "--ac2", "--on", "--line", "--line2", "--glass", "--ok", "--bad", "--hero-ac"];

function paintSite(vars, mode, id) {
  for (const k of KEYS) root.style.removeProperty(k);
  if (id) {
    for (const [k, v] of Object.entries(vars)) root.style.setProperty(k, v);
    root.setAttribute("data-theme", mode);
    root.setAttribute("data-site-theme", id);
  } else {
    root.removeAttribute("data-site-theme");
  }
  document.querySelectorAll('meta[name="theme-color"]').forEach((m) => {
    m.dataset.orig ??= m.content;
    m.content = id ? vars["--bg"] : m.dataset.orig;
  });
}

export const siteTheme = () => root.getAttribute("data-site-theme");

// A circle of the new colours grows from where the visitor clicked (View Transitions, where supported).
function transition(change, at) {
  const motion = root.classList.contains("motion");
  if (!motion || !document.startViewTransition) { change(); return; }
  const x = at?.x ?? innerWidth / 2;
  const y = at?.y ?? innerHeight / 2;
  const r = Math.hypot(Math.max(x, innerWidth - x), Math.max(y, innerHeight - y));
  const vt = document.startViewTransition(change);
  vt.ready.then(() => {
    root.animate(
      { clipPath: [`circle(0px at ${x}px ${y}px)`, `circle(${r}px at ${x}px ${y}px)`] },
      { duration: 750, easing: "cubic-bezier(.16, 1, .3, 1)", pseudoElement: "::view-transition-new(root)" },
    );
  }).catch(() => {});
}

let history = [];

export async function applySiteTheme(id, at, { remember = true } = {}) {
  const th = await themeById(id);
  if (!th) return null;
  if (remember && siteTheme() !== id) history.push(siteTheme());
  const vars = siteVars(th);
  transition(() => paintSite(vars, th.mode, th.id), at);
  try { localStorage.setItem(STORE, JSON.stringify({ id: th.id, mode: th.mode, vars })); } catch { /* lasts this visit */ }
  document.dispatchEvent(new CustomEvent("maxor:theme", { detail: { id: th.id, name: th.name, site: true } }));
  return th;
}

function storedAppearance() {
  try { const v = localStorage.getItem("maxor-theme"); return v === "light" || v === "dark" ? v : null; } catch { return null; }
}

export function resetSiteTheme(at) {
  if (!siteTheme()) return;
  history.push(siteTheme());
  const mode = storedAppearance();
  transition(() => {
    paintSite({}, null, null);
    if (mode) root.setAttribute("data-theme", mode); else root.removeAttribute("data-theme");
  }, at);
  try { localStorage.removeItem(STORE); } catch { /* none */ }
  const light = mode ? mode === "light" : matchMedia("(prefers-color-scheme: light)").matches;
  document.dispatchEvent(new CustomEvent("maxor:theme", { detail: { id: null, desk: light ? "maxor-light" : "maxor-dark" } }));
}

// The sun/moon button: Maxor Light or Maxor Dark, dropping any theme picked before.
export function setAppearance(mode, at) {
  if (siteTheme()) history.push(siteTheme());
  transition(() => {
    paintSite({}, null, null);
    root.setAttribute("data-theme", mode);
  }, at);
  try { localStorage.setItem("maxor-theme", mode); localStorage.removeItem(STORE); } catch { /* lasts this visit */ }
  document.dispatchEvent(new CustomEvent("maxor:theme", { detail: { id: null, desk: mode === "light" ? "maxor-light" : "maxor-dark" } }));
}

// `maxor theme undo`: back to whatever was there before the last change.
export async function undoSiteTheme(at) {
  if (!history.length) return false;
  const prev = history.pop();
  if (prev) await applySiteTheme(prev, at, { remember: false });
  else { resetSiteTheme(at); history.pop(); }
  return true;
}
