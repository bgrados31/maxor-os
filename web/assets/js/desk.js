// The desktop picture in "El escritorio", painted with the real themes. Scrolling the tour switches
// workspaces the way Hyprland slides them and applies each step's theme to the picture; the picker
// (and the terminal) theme the whole site, and the picture follows at once.

import { applySiteTheme, loadThemes, mix, siteTheme } from "./themes.js";

const motion = () => document.documentElement.classList.contains("motion");

// Themes without gradient.json get the system's default wallpaper: a halo of the accent over the background.
function wallpaper(th) {
  if (Array.isArray(th.gradient) && th.gradient.length >= 2) {
    const g = th.gradient;
    return [g[0], g.length > 2 ? g[1] : mix(g[0], g[1], 0.5), g[g.length - 1]];
  }
  return [mix(th.ac, th.bg, th.mode === "dark" ? 0.42 : 0.3), mix(th.ac, th.bg, 0.12), th.bg];
}

export async function initDesk() {
  const desk = document.querySelector("[data-desk]");
  if (!desk) return;

  let themes;
  try { themes = await loadThemes(); } catch { return; } // the picture stays in Maxor Dark (CSS defaults)
  const byId = new Map(themes.map((th) => [th.id, th]));

  const cmdEl = desk.querySelector("[data-desk-cmd]");
  const toast = desk.querySelector("[data-desk-toast]");
  const toastName = desk.querySelector("[data-desk-toast-name]");
  const names = desk.querySelectorAll("[data-desk-theme-name], [data-desk-rail-name]");
  const app = desk.querySelector("[data-desk-app]");
  const tiles = desk.querySelector(".desk__tiles");
  const dots = [...desk.querySelectorAll(".desk__ws i")];
  const picker = document.querySelector("[data-theme-picker]");

  let current = null;
  let typing = 0;
  let toastTimer;

  const shelf = ["maxor-dark", "maxor-light", "sakura", "glacier", "ember", "dawn"];
  function drawApp() {
    const ids = shelf.includes(current) ? shelf : [...shelf.slice(0, 5), current];
    app.replaceChildren(...ids.map((id) => {
      const th = byId.get(id);
      const el = document.createElement("span");
      el.className = "mini" + (id === current ? " is-on" : "");
      el.style.setProperty("background", th.bg);
      el.style.setProperty("--mini-s", th.s2);
      el.style.setProperty("--mini-ac", th.ac);
      return el;
    }));
  }

  function paint(id) {
    const th = byId.get(id);
    if (!th || id === current) return false;
    current = id;
    const [g1, g2, g3] = wallpaper(th);
    const vars = { bg: th.bg, s: th.s, s2: th.s2, fg: th.fg, mu: th.mu, ac: th.ac, ac2: th.ac2, on: th.on, g1, g2, g3 };
    for (const [k, v] of Object.entries(vars)) desk.style.setProperty(`--d-${k}`, v);
    names.forEach((n) => { n.textContent = th.name; });
    drawApp();
    picker?.querySelectorAll("button").forEach((b) => b.setAttribute("aria-pressed", String(b.dataset.id === id)));
    return true;
  }

  // The command appears in the terminal as decoration: the colours never wait for it.
  async function type(cmd) {
    const mine = ++typing;
    if (!motion()) { cmdEl.textContent = cmd; return; }
    cmdEl.textContent = "";
    for (let i = 1; i <= cmd.length; i++) {
      await new Promise((r) => setTimeout(r, 14));
      if (mine !== typing) return;
      cmdEl.textContent = cmd.slice(0, i);
    }
  }

  function notify(name) {
    toastName.textContent = name;
    toast.classList.add("is-shown");
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => toast.classList.remove("is-shown"), 2200);
  }

  function apply(id, cmd = `maxor theme apply ${id}`) {
    if (!paint(id)) return;
    type(cmd);
    notify(byId.get(id).name);
  }

  // Hyprland's workspace slide: out to one side, in from the other, the new theme painted in between.
  let ws = 0;
  function workspace(n, then) {
    if (n === ws) { then(); return; }
    const dir = n > ws ? 1 : -1;
    ws = n;
    dots.forEach((d, i) => d.classList.toggle("on", i === n));
    if (!motion() || !tiles.animate) { then(); return; }
    tiles.animate([
      { transform: "translateX(0)", opacity: 1 },
      { transform: `translateX(${-dir * 34}%)`, opacity: 0, offset: 0.42 },
      { transform: `translateX(${dir * 34}%)`, opacity: 0, offset: 0.43 },
      { transform: "translateX(0)", opacity: 1 },
    ], { duration: 620, easing: "cubic-bezier(.4, 0, .2, 1)" });
    setTimeout(then, 260);
  }

  if (picker) {
    picker.replaceChildren(...themes.map((th) => {
      const b = document.createElement("button");
      b.type = "button";
      b.className = "swatch";
      b.dataset.id = th.id;
      b.setAttribute("aria-pressed", "false");
      const dot = document.createElement("span");
      dot.className = "swatch__dot";
      dot.setAttribute("aria-hidden", "true");
      dot.style.setProperty("background", th.bg);
      const ac = document.createElement("i");
      ac.style.setProperty("background", th.ac);
      dot.append(ac);
      const label = document.createElement("span");
      label.textContent = th.name;
      b.append(dot, label);
      b.addEventListener("click", (e) => {
        const r = b.getBoundingClientRect();
        const at = e.detail ? { x: e.clientX, y: e.clientY } : { x: r.left + r.width / 2, y: r.top + r.height / 2 };
        apply(th.id); // the picture first, instantly
        applySiteTheme(th.id, at);
      });
      return b;
    }));
  }

  // Whatever themes the site (the picker, the terminal, a remembered choice) also themes the picture.
  document.addEventListener("maxor:theme", (e) => apply(e.detail.id ?? e.detail.desk ?? "maxor-dark"));
  paint(siteTheme() ?? "maxor-dark");

  const clock = desk.querySelector("[data-desk-clock]");
  const tick = () => { clock.textContent = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit", hour12: false }).format(new Date()); };
  tick();
  setInterval(tick, 20_000);

  new IntersectionObserver((entries, io) => {
    if (entries.some((e) => e.isIntersecting)) { desk.classList.add("is-in"); io.disconnect(); }
  }, { threshold: 0.2 }).observe(desk);

  // The tour: the step that crosses the middle of the screen leads, each one its own workspace.
  const steps = [...document.querySelectorAll(".step[data-step-theme]")];
  const io = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (!e.isIntersecting) continue;
      const n = steps.indexOf(e.target);
      steps.forEach((s) => s.classList.toggle("is-active", s === e.target));
      // A theme the visitor chose wins over the tour's.
      const id = siteTheme() && n === steps.length - 1 ? siteTheme() : e.target.dataset.stepTheme;
      workspace(n, () => apply(id, e.target.dataset.stepCmd || `maxor theme apply ${id}`));
    }
  }, { rootMargin: "-48% 0px -48% 0px" });
  steps.forEach((s) => io.observe(s));
  steps[0]?.classList.add("is-active");
}
