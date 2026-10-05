// The cinematic layer: smooth scroll (Lenis), parallax and scroll-scrubbed scenes (GSAP + ScrollTrigger,
// vendored in assets/vendor). It only adds: without the libraries, or with reduced motion, the page keeps
// its CSS reveals and its final states.

import { lang } from "./i18n.js";
import { loadThemes } from "./themes.js";

const root = document.documentElement;
let smooth = null;
export const lenis = () => smooth;

const hover = () => matchMedia("(hover: hover) and (pointer: fine)").matches;

// ── Titles rise word by word. The heading keeps its text for assistive tech (aria-label). ──
function split(el) {
  const text = el.textContent.trim().replace(/\s+/g, " ");
  if (el.dataset.split === text) return el.querySelectorAll(".w > i");
  el.dataset.split = text;
  el.setAttribute("aria-label", text);
  el.replaceChildren(...text.split(" ").flatMap((word, i) => {
    const w = document.createElement("span");
    w.className = "w";
    w.setAttribute("aria-hidden", "true");
    const inner = document.createElement("i");
    inner.textContent = word;
    w.append(inner);
    return i ? [document.createTextNode(" "), w] : [w];
  }));
  return el.querySelectorAll(".w > i");
}

function titles(gsap) {
  document.querySelectorAll(".title, .step__title").forEach((el) => {
    const words = split(el);
    gsap.from(words, {
      yPercent: 115, rotate: 6, opacity: 0, duration: 1, ease: "expo.out", stagger: 0.045,
      scrollTrigger: { trigger: el, start: "top 88%", once: true },
    });
  });
  // A language switch rewrites the text: split again, already in place.
  document.addEventListener("maxor:lang", () => document.querySelectorAll(".title, .step__title").forEach((el) => {
    el.removeAttribute("data-split");
    el.removeAttribute("aria-label");
    split(el);
  }));
}

// ── The hero: layers at different depths, on scroll and under the pointer. ──
function hero(gsap) {
  const h = document.querySelector("[data-hero]");
  if (!h) return;
  const st = { trigger: h, start: "top top", end: "bottom top", scrub: true };
  gsap.to(h.querySelector(".stars"), { yPercent: -30, ease: "none", scrollTrigger: st });
  gsap.to(h.querySelector(".aurora"), { yPercent: 35, scale: 1.25, ease: "none", scrollTrigger: st });
  gsap.to(h.querySelector(".hero__ground"), { "--fade": 1, ease: "none", scrollTrigger: { ...st, end: "60% top" } });
  if (!hover()) return;
  gsap.set(h, { "--mx": 0, "--my": 0 });
  const mx = gsap.quickTo(h, "--mx", { duration: 0.9, ease: "power3.out" });
  const my = gsap.quickTo(h, "--my", { duration: 0.9, ease: "power3.out" });
  h.addEventListener("pointermove", (e) => { mx((e.clientX / innerWidth) * 2 - 1); my((e.clientY / innerHeight) * 2 - 1); });
  h.addEventListener("pointerleave", () => { mx(0); my(0); });
}

// ── The screen powers on: the desktop comes in tilted and dark, and settles flat as the tour starts. ──
function desk(gsap) {
  const d = document.querySelector(".tour__stage .desk");
  if (!d) return;
  gsap.fromTo(d,
    { rotateX: 28, scale: 0.8, yPercent: 10, filter: "brightness(.35) saturate(.6)" },
    { rotateX: 0, scale: 1, yPercent: 0, filter: "brightness(1) saturate(1)", ease: "none",
      scrollTrigger: { trigger: ".tour", start: "top 95%", end: "top 15%", scrub: 0.6 } });
}

// ── Glow orbs drift slower than the page: depth behind each section. ──
function orbs(gsap) {
  document.querySelectorAll(".orb").forEach((o) => {
    gsap.fromTo(o, { yPercent: -30 }, { yPercent: 40, ease: "none",
      scrollTrigger: { trigger: o.parentElement, start: "top bottom", end: "bottom top", scrub: true } });
  });
}

// ── Numbers count when they come into view. ──
function counters(gsap) {
  document.querySelectorAll("[data-count]").forEach((el) => {
    const to = Number(el.dataset.count);
    const from = Number(el.dataset.from ?? 0);
    const dec = Number(el.dataset.decimals ?? 0);
    const fmt = () => new Intl.NumberFormat(lang(), { minimumFractionDigits: dec, maximumFractionDigits: dec });
    const o = { v: from };
    el.textContent = fmt().format(from);
    gsap.to(o, {
      v: to, duration: 1.8, ease: "power3.out",
      onUpdate: () => { el.textContent = fmt().format(o.v); },
      scrollTrigger: { trigger: el, start: "top 90%", once: true },
    });
    document.addEventListener("maxor:lang", () => { el.textContent = fmt().format(o.v); });
  });
}

// ── The marquee of themes: it runs on its own and speeds up (or turns back) with the scroll. ──
// Fills the row (also without the libraries: then it just stands still). Returns the track, or null.
export async function fillMarquee() {
  const track = document.querySelector("[data-marquee]");
  if (!track) return null;
  let themes;
  try { themes = await loadThemes(); } catch { return null; }
  const item = (th) => {
    const li = document.createElement("li");
    const dot = document.createElement("i");
    dot.style.setProperty("background", `linear-gradient(135deg, ${th.ac}, ${th.ac2})`);
    li.append(dot, document.createTextNode(th.name));
    return li;
  };
  const list = () => { const ul = document.createElement("ul"); ul.append(...themes.map(item)); return ul; };
  track.replaceChildren(list(), list());
  return track;
}

async function marquee(gsap, ScrollTrigger) {
  const track = await fillMarquee();
  if (!track) return;
  const loop = gsap.to(track, { xPercent: -50, duration: 38, ease: "none", repeat: -1 });
  let dir = 1;
  ScrollTrigger.create({
    onUpdate(self) {
      const v = self.getVelocity();
      if (v) dir = v < 0 ? -1 : 1;
      loop.timeScale(dir * (1 + Math.min(Math.abs(v) / 260, 7)));
      gsap.to(loop, { timeScale: dir, duration: 1.2, ease: "power2.out", overwrite: true });
      gsap.set(track, { skewX: gsap.utils.clamp(-8, 8, -v / 400) });
      gsap.to(track, { skewX: 0, duration: 0.6, ease: "power3.out", overwrite: "auto" });
    },
  });
}

// ── Buttons lean toward the pointer. ──
function magnets(gsap) {
  if (!hover()) return;
  document.querySelectorAll(".btn--accent, .chip, .hint").forEach((el) => {
    const x = gsap.quickTo(el, "x", { duration: 0.5, ease: "power3.out" });
    const y = gsap.quickTo(el, "y", { duration: 0.5, ease: "power3.out" });
    el.addEventListener("pointermove", (e) => {
      const r = el.getBoundingClientRect();
      x((e.clientX - r.left - r.width / 2) * 0.28);
      y((e.clientY - r.top - r.height / 2) * 0.32);
    });
    el.addEventListener("pointerleave", () => { gsap.to(el, { x: 0, y: 0, duration: 0.9, ease: "elastic.out(1, .4)" }); });
  });
}

// ── Cards come up in sequence, with a little depth. ──
function cards(gsap, ScrollTrigger) {
  const els = gsap.utils.toArray(".tile, .point, .stat");
  els.forEach((el) => el.classList.remove("reveal"));
  gsap.set(els, { opacity: 0, y: 60, rotateX: -14, transformPerspective: 900, transformOrigin: "50% 100%" });
  ScrollTrigger.batch(els, {
    start: "top 90%", once: true,
    onEnter: (batch) => gsap.to(batch, { opacity: 1, y: 0, rotateX: 0, duration: 1, ease: "expo.out", stagger: 0.08, clearProps: "transform" }),
  });
}

// ── The footer mark gathers its letters as the page ends. ──
function footer(gsap) {
  const w = document.querySelector(".foot__word");
  if (!w) return;
  gsap.fromTo(w, { letterSpacing: "0.5em", opacity: 0.15, yPercent: 30 }, {
    letterSpacing: "0.1em", opacity: 1, yPercent: 0, ease: "none",
    scrollTrigger: { trigger: ".foot", start: "top bottom", end: "top 35%", scrub: 0.8 },
  });
}

export function initMotion() {
  const { gsap, ScrollTrigger, Lenis } = window;
  if (!root.classList.contains("motion") || !gsap || !ScrollTrigger) return false;
  gsap.registerPlugin(ScrollTrigger);
  root.classList.add("gsap");

  if (Lenis) {
    smooth = new Lenis({ lerp: 0.11, anchors: { offset: -90 }, autoRaf: false });
    smooth.on("scroll", ScrollTrigger.update);
    gsap.ticker.add((time) => smooth.raf(time * 1000));
    gsap.ticker.lagSmoothing(0);
  }

  titles(gsap);
  hero(gsap);
  desk(gsap);
  orbs(gsap);
  counters(gsap);
  marquee(gsap, ScrollTrigger);
  magnets(gsap);
  cards(gsap, ScrollTrigger);
  footer(gsap);
  // Fonts and the release card change heights: measure again once they settle.
  document.fonts?.ready.then(() => ScrollTrigger.refresh());
  return true;
}
