// Scroll and pointer effects. One passive scroll listener feeds a requestAnimationFrame loop that writes
// CSS variables (--hp on the hero, --p on [data-scrub]); the CSS does the drawing. With reduced motion
// none of this runs and the page shows its final state.

import { t } from "./i18n.js";
import { resetSiteTheme, setAppearance, siteTheme, themeById } from "./themes.js";

const root = document.documentElement;
const motion = root.classList.contains("motion");
const clamp = (v, a = 0, b = 1) => Math.min(b, Math.max(a, v));

function reveals() {
  const els = document.querySelectorAll(".reveal");
  if (!motion || !("IntersectionObserver" in window)) return;
  const io = new IntersectionObserver((entries) => {
    for (const e of entries) if (e.isIntersecting) { e.target.classList.add("is-in"); io.unobserve(e.target); }
  }, { rootMargin: "0px 0px -8% 0px", threshold: 0.12 });
  els.forEach((el) => io.observe(el));
}

// The installer picture: the disk plan fills in, then the stages run.
function drawTui(tui, p) {
  const steps = tui.querySelectorAll(".tui__steps li");
  const stages = tui.querySelectorAll(".stages li");
  const installing = p >= 0.5;
  const q = clamp((p - 0.5) * 2);
  tui.dataset.phase = installing ? "install" : "disk";
  tui.style.setProperty("--q", q.toFixed(3));
  const now = installing ? steps.length - 1 : 4; // Disk, then Installing
  steps.forEach((li, i) => { li.classList.toggle("is-done", i < now || q >= 1); li.classList.toggle("is-now", i === now && q < 1); });
  const k = installing ? Math.min(stages.length, Math.floor(q * stages.length)) : -1;
  stages.forEach((li, i) => { li.classList.toggle("is-done", i < k); li.classList.toggle("is-now", i === k); });
  // the log keeps the last four lines, newest at the bottom
  const log = tui.querySelector("[data-tui-log]");
  const lines = k < 0 ? [LOG.plan] : [LOG.plan, ...LOG.stages.slice(0, k + 1)].slice(-4);
  if (log && log.dataset.k !== String(k)) {
    log.dataset.k = String(k);
    log.replaceChildren(...lines.map((l) => Object.assign(document.createElement("span"), { textContent: l })));
  }
}

// What the engine says at each stage (installer/engine), shortened.
const LOG = {
  plan: "▸ plan     alongside Windows · 230 GB free · LUKS",
  stages: [
    "▸ preflight  UEFI, memory and disk checked",
    "▸ disk       new partitions in the free space",
    "▸ luks       encrypting the system partition",
    "▸ filesystem formatting and mounting",
    "▸ host       writing this machine's configuration",
    "▸ install    installing from the ISO, offline",
    "▸ bootloader the boot menu keeps Windows",
    "▸ finish     done · remove the USB and restart",
    "✓ all done   Maxor OS is installed",
  ],
};

function scroll() {
  const hero = document.querySelector("[data-hero]");
  const scrubs = [...document.querySelectorAll("[data-scrub]")];
  const tui = document.querySelector(".tui");
  const bar = document.querySelector("[data-bar]");
  let lastY = scrollY;
  let queued = false;

  const frame = () => {
    queued = false;
    const vh = innerHeight;
    if (hero && motion) {
      const h = hero.offsetHeight || vh;
      hero.style.setProperty("--hp", clamp(scrollY / h).toFixed(4));
    }
    for (const el of scrubs) {
      // "section": follows the whole section; "self": runs while the element rises to the upper third.
      let p = 1;
      if (motion) {
        const own = el.dataset.scrub === "self";
        const box = (own ? el : el.closest("section") ?? el).getBoundingClientRect();
        p = own ? clamp((vh * 0.9 - box.top) / (vh * 0.78)) : clamp((vh * 0.72 - box.top) / (box.height * 0.62));
      }
      el.style.setProperty("--p", p.toFixed(4));
      if (el === tui) drawTui(tui, p);
    }
    // The bar steps aside while reading down and comes back on the way up.
    if (bar) {
      const y = scrollY;
      if (y > vh * 0.6 && y > lastY + 4) bar.setAttribute("data-hidden", "");
      else if (y < lastY - 4 || y < vh * 0.6) bar.removeAttribute("data-hidden");
      lastY = y;
    }
  };
  const ask = () => { if (!queued) { queued = true; requestAnimationFrame(frame); } };
  addEventListener("scroll", ask, { passive: true });
  addEventListener("resize", ask, { passive: true });
  frame();
  // Keyboard users never lose the bar.
  bar?.addEventListener("focusin", () => bar.removeAttribute("data-hidden"));
}

function navCurrent() {
  const links = new Map([...document.querySelectorAll(".bar__links a")].map((a) => [a.getAttribute("href").slice(1), a]));
  const io = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (!e.isIntersecting) continue;
      links.forEach((a, id) => (id === e.target.id ? a.setAttribute("aria-current", "true") : a.removeAttribute("aria-current")));
    }
  }, { rootMargin: "-45% 0px -50% 0px" });
  links.forEach((_, id) => { const s = document.getElementById(id); if (s) io.observe(s); });
}

function glow() {
  if (!matchMedia("(hover: hover)").matches) return;
  document.querySelectorAll("[data-glow] .tile").forEach((tile) => {
    tile.addEventListener("pointermove", (e) => {
      const r = tile.getBoundingClientRect();
      tile.style.setProperty("--mx", `${e.clientX - r.left}px`);
      tile.style.setProperty("--my", `${e.clientY - r.top}px`);
    }, { passive: true });
  });
}

function copy() {
  document.querySelectorAll("[data-copy-from]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const src = btn.parentElement.querySelector(btn.dataset.copyFrom);
      if (!src) return;
      const label = btn.textContent;
      let ok = false;
      try { await navigator.clipboard.writeText(src.textContent.trim()); ok = true; } catch {
        const range = document.createRange();
        range.selectNodeContents(src);
        getSelection().removeAllRanges();
        getSelection().addRange(range);
      }
      btn.textContent = ok ? t("copy.done") : t("copy.fail");
      btn.toggleAttribute("data-done", ok);
      setTimeout(() => { btn.textContent = label; btn.removeAttribute("data-done"); }, 1800);
    });
  });
}

// Appearance: follows the system until the visitor picks one (or a theme); the choice is remembered.
function appearance() {
  const btn = document.querySelector("[data-theme-toggle]");
  if (!btn) return;
  const sys = matchMedia("(prefers-color-scheme: light)");
  const effective = () => root.getAttribute("data-theme") || (sys.matches ? "light" : "dark");
  const label = () => btn.setAttribute("aria-label", effective() === "dark" ? t("nav.toLight") : t("nav.toDark"));
  btn.addEventListener("click", () => {
    const r = btn.getBoundingClientRect();
    setAppearance(effective() === "dark" ? "light" : "dark", { x: r.left + r.width / 2, y: r.top + r.height / 2 });
    label();
  });
  sys.addEventListener?.("change", label);
  document.addEventListener("maxor:lang", label);
  document.addEventListener("maxor:theme", () => requestAnimationFrame(label));
  label();
}

// The bar shows the theme picked for the site, and a way back to Maxor.
function siteChip() {
  const chip = document.querySelector("[data-site-reset]");
  if (!chip) return;
  const name = chip.querySelector("[data-site-name]");
  const dot = chip.querySelector(".site-dot");
  const show = async () => {
    const id = siteTheme();
    chip.hidden = !id;
    if (!id) return;
    const th = await themeById(id);
    name.textContent = th?.name ?? id;
    dot.style.setProperty("background", th ? `linear-gradient(135deg, ${th.ac}, ${th.ac2})` : "");
    chip.setAttribute("aria-label", t("site.reset", { name: th?.name ?? id }));
  };
  chip.addEventListener("click", () => {
    const r = chip.getBoundingClientRect();
    resetSiteTheme({ x: r.left + r.width / 2, y: r.top + r.height / 2 });
  });
  document.addEventListener("maxor:theme", () => requestAnimationFrame(show));
  document.addEventListener("maxor:lang", show);
  show();
}

export function initEffects() {
  reveals();
  scroll();
  navCurrent();
  glow();
  copy();
  appearance();
  siteChip();
}
