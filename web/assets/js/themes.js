// The official themes (assets/data/themes.json, synced from themes/) and the site's appearance.
// The appearance is one of: the system's (Maxor Dark or Maxor Light by prefers-color-scheme), Maxor Dark,
// Maxor Light, or any official theme, which repaints the whole page the way `maxor theme apply` repaints
// the system. The state lives here, synchronously; the page is painted after (inside a View Transition),
// so asking right after a change always gets the new answer. boot.js repaints it before the first frame.

const root = document.documentElement;
const THEME_KEY = "maxor-site-theme"; // {id, mode, vars}
const MODE_KEY = "maxor-theme"; // "light" | "dark"; absent = the system's
let loading;

export function loadThemes() {
  loading ??= fetch(new URL("../data/themes.json", import.meta.url), { credentials: "omit" })
    .then((r) => { if (!r.ok) throw new Error(`themes.json: ${r.status}`); return r.json(); });
  return loading;
}
export async function themeById(id) {
  const key = String(id ?? "").toLowerCase();
  return (await loadThemes()).find((t) => t.id === key || t.name.toLowerCase() === key) ?? null;
}

function hex(c) {
  const n = parseInt(c.slice(1), 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}
export function mix(a, b, w) {
  const [x, y] = [hex(a), hex(b)];
  return "#" + x.map((v, i) => Math.round(v * w + y[i] * (1 - w)).toString(16).padStart(2, "0")).join("");
}

// The site's tokens (site.css :root) from a theme: every pair the page uses is one the theme guarantees.
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
  if (dark) vars["--hero-ac"] = th.ac; // the hero stays night; a dark theme lends it its accent
  return vars;
}
const KEYS = ["--bg", "--bg2", "--s", "--s2", "--fg", "--mu", "--ac", "--ac2", "--on", "--line", "--line2", "--glass", "--ok", "--bad", "--hero-ac"];

function read(key) { try { return localStorage.getItem(key); } catch { return null; } }
function write(key, v) { try { v == null ? localStorage.removeItem(key) : localStorage.setItem(key, v); } catch { /* this visit only */ } }

// ── state ──
const sys = matchMedia("(prefers-color-scheme: light)");
const state = {
  theme: root.getAttribute("data-site-theme"), // set by boot.js from a saved choice
  mode: (() => { const m = read(MODE_KEY); return m === "light" || m === "dark" ? m : "system"; })(),
};
const past = []; // for `maxor theme undo`: earlier {theme, mode}

export const siteTheme = () => state.theme;
export const appearanceMode = () => state.mode;
export const effectiveMode = () => (state.theme ? root.getAttribute("data-theme") : state.mode === "system" ? (sys.matches ? "light" : "dark") : state.mode);
// The theme id the page shows right now (Maxor Dark/Light when no theme was picked).
export const shownTheme = () => state.theme ?? (effectiveMode() === "light" ? "maxor-light" : "maxor-dark");

function announce() {
  document.dispatchEvent(new CustomEvent("maxor:theme", { detail: { id: state.theme, mode: state.mode, shown: shownTheme() } }));
}

// A circle of the new colours grows from where the visitor clicked (View Transitions, where supported).
function transition(change, at) {
  if (!root.classList.contains("motion") || !document.startViewTransition) { change(); return; }
  const x = at?.x ?? innerWidth / 2;
  const y = at?.y ?? innerHeight / 2;
  const r = Math.hypot(Math.max(x, innerWidth - x), Math.max(y, innerHeight - y));
  const vt = document.startViewTransition(change);
  vt.ready.then(() => root.animate(
    { clipPath: [`circle(0px at ${x}px ${y}px)`, `circle(${r}px at ${x}px ${y}px)`] },
    { duration: 650, easing: "cubic-bezier(.16, 1, .3, 1)", pseudoElement: "::view-transition-new(root)" },
  )).catch(() => {});
}

function paint(th) {
  for (const k of KEYS) root.style.removeProperty(k);
  if (th) {
    for (const [k, v] of Object.entries(siteVars(th))) root.style.setProperty(k, v);
    root.setAttribute("data-theme", th.mode);
    root.setAttribute("data-site-theme", th.id);
  } else {
    root.removeAttribute("data-site-theme");
    if (state.mode === "system") root.removeAttribute("data-theme"); else root.setAttribute("data-theme", state.mode);
  }
  const bg = getComputedStyle(root).getPropertyValue("--bg").trim();
  document.querySelectorAll('meta[name="theme-color"]').forEach((m) => {
    m.dataset.orig ??= m.content;
    m.content = th || state.mode !== "system" ? bg : m.dataset.orig;
  });
}

async function set(next, at, { record = true } = {}) {
  const th = next.theme ? await themeById(next.theme) : null;
  if (next.theme && !th) return null;
  const same = (th?.id ?? null) === state.theme && next.mode === state.mode;
  if (same) return th ?? true;
  if (record) past.push({ theme: state.theme, mode: state.mode });
  state.theme = th?.id ?? null;
  state.mode = next.mode;
  write(MODE_KEY, state.mode === "system" ? null : state.mode);
  write(THEME_KEY, th ? JSON.stringify({ id: th.id, mode: th.mode, vars: siteVars(th) }) : null);
  transition(() => paint(th), at);
  announce();
  return th ?? true;
}

// Picks an official theme for the whole site (null if there is no such theme).
export const applySiteTheme = (id, at) => set({ theme: id, mode: state.mode }, at);
// Back to Maxor (Dark or Light, as the mode says): `maxor rollback`.
export const resetSiteTheme = (at) => set({ theme: null, mode: state.mode }, at);
// "system" | "dark" | "light": Maxor's own look, dropping any theme picked before.
export const setAppearance = (mode, at) => set({ theme: null, mode }, at);
// `maxor theme undo`: false when there is nothing to undo.
export async function undoSiteTheme(at) {
  const prev = past.pop();
  if (!prev) return false;
  await set(prev, at, { record: false });
  return true;
}

sys.addEventListener?.("change", () => { if (!state.theme && state.mode === "system") announce(); });
