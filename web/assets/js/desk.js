// The desktop picture in "El escritorio": painted with the real themes (assets/data/themes.json, synced
// from themes/ by web/tools/sync-themes.py). Each step of the tour applies a theme the way the system
// does: the command is typed, the colours glide, a notification says so.

const motion = () => document.documentElement.classList.contains("motion");

function hex(c) {
  const n = parseInt(c.slice(1), 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}
function mix(a, b, w) {
  const [x, y] = [hex(a), hex(b)];
  return "#" + x.map((v, i) => Math.round(v * w + y[i] * (1 - w)).toString(16).padStart(2, "0")).join("");
}

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
  try {
    const res = await fetch(new URL("../data/themes.json", import.meta.url), { credentials: "omit" });
    themes = await res.json();
  } catch {
    return; // the picture stays in Maxor Dark, painted by the CSS defaults
  }
  const byId = new Map(themes.map((th) => [th.id, th]));

  const cmdEl = desk.querySelector("[data-desk-cmd]");
  const toast = desk.querySelector("[data-desk-toast]");
  const toastName = desk.querySelector("[data-desk-toast-name]");
  const names = desk.querySelectorAll("[data-desk-theme-name], [data-desk-rail-name]");
  const app = desk.querySelector("[data-desk-app]");
  const picker = document.querySelector("[data-theme-picker]");

  let current = "maxor-dark";
  let run = 0; // a newer request cancels the typing of an older one
  let toastTimer;

  // The Maxor app in the picture: six themes, always including the one applied.
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
    if (!th) return;
    current = id;
    const [g1, g2, g3] = wallpaper(th);
    const vars = { bg: th.bg, s: th.s, s2: th.s2, fg: th.fg, mu: th.mu, ac: th.ac, ac2: th.ac2, on: th.on, g1, g2, g3 };
    for (const [k, v] of Object.entries(vars)) desk.style.setProperty(`--d-${k}`, v);
    names.forEach((n) => { n.textContent = th.name; });
    drawApp();
    picker?.querySelectorAll("button").forEach((b) => b.setAttribute("aria-pressed", String(b.dataset.id === id)));
  }

  function notify(name) {
    if (!toast) return;
    toastName.textContent = name;
    toast.classList.add("is-shown");
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => toast.classList.remove("is-shown"), 2400);
  }

  async function apply(id, cmd = `maxor theme apply ${id}`) {
    if (!byId.has(id) || id === current) return;
    const mine = ++run;
    if (motion()) {
      cmdEl.textContent = "";
      for (const ch of cmd) {
        await new Promise((r) => setTimeout(r, 26 + Math.random() * 34));
        if (mine !== run) return;
        cmdEl.textContent += ch;
      }
      await new Promise((r) => setTimeout(r, 220));
      if (mine !== run) return;
    } else {
      cmdEl.textContent = cmd;
    }
    paint(id);
    notify(byId.get(id).name);
  }

  // The picker: every official theme, as a real button.
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
      b.addEventListener("click", () => apply(th.id));
      return b;
    }));
  }

  paint(current);

  // The clock in the bar tells the visitor's own time.
  const clock = desk.querySelector("[data-desk-clock]");
  const tick = () => { clock.textContent = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit", hour12: false }).format(new Date()); };
  tick();
  setInterval(tick, 20_000);

  // First sight: the windows pop in.
  new IntersectionObserver((entries, io) => {
    if (entries.some((e) => e.isIntersecting)) { desk.classList.add("is-in"); io.disconnect(); }
  }, { threshold: 0.2 }).observe(desk);

  // The tour: whichever step crosses the middle of the screen leads.
  const steps = [...document.querySelectorAll(".step[data-step-theme]")];
  const io = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (!e.isIntersecting) continue;
      steps.forEach((s) => s.classList.toggle("is-active", s === e.target));
      const id = e.target.dataset.stepTheme;
      apply(id, e.target.dataset.stepCmd || `maxor theme apply ${id}`);
    }
  }, { rootMargin: "-48% 0px -48% 0px" });
  steps.forEach((s) => io.observe(s));
  steps[0]?.classList.add("is-active");
}
