// The top bar: which section you are in (an ink that slides under its link), how far down the page you
// are, the menu on small screens and the Appearance panel (System / Dark / Light and every theme).

import { t } from "./i18n.js";
import { appearanceMode, loadThemes, setAppearance, applySiteTheme, shownTheme, siteTheme } from "./themes.js";

const centre = (el) => { const r = el.getBoundingClientRect(); return { x: r.left + r.width / 2, y: r.top + r.height / 2 }; };

function sections(bar) {
  const list = bar.querySelector(".bar__links");
  const ink = list.querySelector(".bar__ink");
  const links = [...list.querySelectorAll("a")];
  let current = null;
  const place = () => {
    const a = current;
    list.classList.toggle("has-ink", Boolean(a));
    if (!a) return;
    ink.style.setProperty("--x", `${a.offsetLeft}px`);
    ink.style.setProperty("--w", `${a.offsetWidth}px`);
  };
  const io = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (e.isIntersecting) current = links.find((a) => a.getAttribute("href") === `#${e.target.id}`) ?? null;
      else if (current?.getAttribute("href") === `#${e.target.id}`) current = null;
    }
    links.forEach((a) => (a === current ? a.setAttribute("aria-current", "true") : a.removeAttribute("aria-current")));
    place();
  }, { rootMargin: "-45% 0px -50% 0px" });
  links.forEach((a) => { const s = document.getElementById(a.getAttribute("href").slice(1)); if (s) io.observe(s); });
  addEventListener("resize", place, { passive: true });
  document.addEventListener("maxor:lang", () => requestAnimationFrame(place));
  document.fonts?.ready.then(place);
}

function progress(bar) {
  const fill = bar.querySelector(".bar__progress i");
  let queued = false;
  const draw = () => {
    queued = false;
    const max = document.documentElement.scrollHeight - innerHeight;
    fill.style.setProperty("transform", `scaleX(${max > 0 ? Math.min(1, scrollY / max) : 0})`);
    bar.toggleAttribute("data-scrolled", scrollY > 40);
  };
  addEventListener("scroll", () => { if (!queued) { queued = true; requestAnimationFrame(draw); } }, { passive: true });
  draw();
}

function menu(bar) {
  const btn = bar.querySelector("[data-menu]");
  const list = bar.querySelector(".bar__links");
  const set = (open) => {
    btn.setAttribute("aria-expanded", String(open));
    bar.toggleAttribute("data-menu-open", open);
    bar.toggleAttribute("data-locked", open);
  };
  btn.addEventListener("click", () => set(btn.getAttribute("aria-expanded") !== "true"));
  list.addEventListener("click", (e) => { if (e.target.closest("a")) set(false); });
  document.addEventListener("keydown", (e) => { if (e.key === "Escape" && bar.hasAttribute("data-menu-open")) { set(false); btn.focus(); } });
  document.addEventListener("click", (e) => { if (!bar.contains(e.target)) set(false); });
  matchMedia("(min-width: 861px)").addEventListener?.("change", () => set(false));
}

async function look(bar) {
  const panel = document.querySelector("[data-look]");
  const btn = bar.querySelector("[data-look-button]");
  const dot = btn.querySelector(".look-dot");
  const grid = panel.querySelector("[data-look-themes]");
  const modes = [...panel.querySelectorAll("[data-mode]")];
  if (!panel.showPopover) { btn.hidden = true; return; } // very old browsers: the tour's picker remains

  // The panel hangs under its button.
  panel.addEventListener("beforetoggle", (e) => {
    if (e.newState !== "open") { bar.removeAttribute("data-locked"); btn.setAttribute("aria-expanded", "false"); return; }
    const r = btn.getBoundingClientRect();
    panel.style.setProperty("--top", `${r.bottom + 12}px`);
    const width = Math.min(340, innerWidth - 24);
    const right = Math.min(Math.max(12, innerWidth - r.right - 8), innerWidth - width - 12);
    panel.style.setProperty("--right", `${right}px`);
    bar.setAttribute("data-locked", "");
    btn.setAttribute("aria-expanded", "true");
  });
  addEventListener("resize", () => panel.matches(":popover-open") && panel.hidePopover(), { passive: true });

  let themes = [];
  try { themes = await loadThemes(); } catch { /* the modes still work */ }
  grid.replaceChildren(...themes.map((th) => {
    const b = document.createElement("button");
    b.type = "button";
    b.className = "look__theme";
    b.dataset.id = th.id;
    b.setAttribute("aria-pressed", "false");
    const sw = document.createElement("span");
    sw.className = "look__sw";
    sw.setAttribute("aria-hidden", "true");
    sw.style.setProperty("--sw-bg", th.bg);
    sw.style.setProperty("--sw-s", th.s2);
    sw.style.setProperty("--sw-ac", th.ac);
    sw.style.setProperty("--sw-ac2", th.ac2);
    const name = document.createElement("span");
    name.textContent = th.name;
    b.append(sw, name);
    b.addEventListener("click", () => applySiteTheme(th.id, centre(b)));
    return b;
  }));

  modes.forEach((m) => m.addEventListener("click", () => setAppearance(m.dataset.mode, centre(m))));

  const sync = async () => {
    const picked = siteTheme();
    modes.forEach((m) => m.setAttribute("aria-pressed", String(!picked && m.dataset.mode === appearanceMode())));
    grid.querySelectorAll("button").forEach((b) => b.setAttribute("aria-pressed", String(b.dataset.id === picked)));
    const th = themes.find((x) => x.id === shownTheme());
    if (th) {
      dot.style.setProperty("--sw-bg", th.bg);
      dot.style.setProperty("--sw-ac", th.ac);
      dot.style.setProperty("--sw-ac2", th.ac2);
    }
    btn.setAttribute("title", `${t("look.now")}: ${th?.name ?? "Maxor"}`);
  };
  document.addEventListener("maxor:theme", sync);
  document.addEventListener("maxor:lang", sync);
  sync();
}

export function initBar() {
  const bar = document.querySelector("[data-bar]");
  if (!bar) return;
  sections(bar);
  progress(bar);
  menu(bar);
  look(bar);
}
