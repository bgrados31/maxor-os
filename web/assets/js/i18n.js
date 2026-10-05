// Two languages. The HTML is the Spanish text; English comes from assets/i18n/en.json, keyed by
// data-i18n (text) and data-i18n-attr ("attr:key; attr:key"). Text is only ever set with textContent.
// Strings built at run time (dates, the release card) live here in both languages.

const STORE = "maxor-lang";
const root = document.documentElement;
const original = new Map(); // element → its Spanish text and attributes, to come back without a reload
let en = null;

const dynamic = {
  es: {
    "meta.title": "Maxor OS · El escritorio Hyprland declarativo",
    "chip.latest": "Ya salió {tag} · ver novedades",
    "chip.pre": "{tag} en prueba · ver novedades",
    "rel.stable": "Estable",
    "rel.pre": "Pre-release",
    "rel.iso": "Descargar la ISO",
    "rel.isoSize": "Descargar la ISO · {size}",
    "rel.view": "Ver la versión {tag}",
    "rel.all": "Ver todas las versiones",
    "rel.published": "Publicada {rel}",
    "rel.error": "No se pudo leer GitHub ahora mismo.",
    "rel.errorBody": "Las versiones siguen en la página del proyecto.",
    "rel.none": "Primera versión en camino",
    "rel.noneBody": "Mientras tanto, instala desde NixOS con el flake.",
    "rel.more": "y {n} cambios más en las notas completas.",
    "copy.done": "Copiado",
    "copy.fail": "Selecciona y copia",
    "look.now": "Ahora",
  },
  en: {
    "meta.title": "Maxor OS · The declarative Hyprland desktop",
    "chip.latest": "{tag} is out · see what's new",
    "chip.pre": "{tag} in testing · see what's new",
    "rel.stable": "Stable",
    "rel.pre": "Pre-release",
    "rel.iso": "Download the ISO",
    "rel.isoSize": "Download the ISO · {size}",
    "rel.view": "See release {tag}",
    "rel.all": "See every release",
    "rel.published": "Published {rel}",
    "rel.error": "GitHub could not be reached right now.",
    "rel.errorBody": "Releases are still on the project page.",
    "rel.none": "The first release is on its way",
    "rel.noneBody": "Meanwhile, install from NixOS with the flake.",
    "rel.more": "and {n} more changes in the full notes.",
    "copy.done": "Copied",
    "copy.fail": "Select and copy",
    "look.now": "Now",
  },
};

export const lang = () => root.getAttribute("data-lang") || "es";

export function t(key, vars = {}) {
  const s = dynamic[lang()][key] ?? dynamic.es[key] ?? key;
  return s.replace(/\{(\w+)\}/g, (_, k) => String(vars[k] ?? ""));
}

function remember(el) {
  if (original.has(el)) return original.get(el);
  const attrs = {};
  for (const [attr] of pairs(el)) attrs[attr] = el.getAttribute(attr);
  const entry = { text: el.hasAttribute("data-i18n") ? el.textContent : null, attrs };
  original.set(el, entry);
  return entry;
}

function* pairs(el) {
  const spec = el.getAttribute("data-i18n-attr");
  if (!spec) return;
  for (const part of spec.split(";")) {
    const [attr, key] = part.split(":").map((x) => x.trim());
    if (attr && key) yield [attr, key];
  }
}

async function loadEnglish() {
  if (en) return en;
  const res = await fetch(new URL("../i18n/en.json", import.meta.url), { credentials: "omit" });
  if (!res.ok) throw new Error(`en.json: ${res.status}`);
  en = await res.json();
  return en;
}

async function apply(next) {
  const els = document.querySelectorAll("[data-i18n], [data-i18n-attr]");
  els.forEach(remember);
  let dict = null;
  if (next === "en") {
    try { dict = await loadEnglish(); } catch { next = "es"; }
  }
  root.setAttribute("data-lang", next);
  root.lang = next;
  for (const el of els) {
    const o = original.get(el);
    if (el.hasAttribute("data-i18n")) {
      const v = dict ? dict[el.getAttribute("data-i18n")] : o.text;
      if (typeof v === "string") el.textContent = v;
    }
    for (const [attr, key] of pairs(el)) {
      const v = dict ? dict[key] : o.attrs[attr];
      if (typeof v === "string") el.setAttribute(attr, v);
    }
  }
  document.title = t("meta.title");
  const desc = document.querySelector('meta[name="description"]');
  if (desc) {
    desc.dataset.es ??= desc.content;
    desc.content = dict?.["meta.description"] ?? desc.dataset.es;
  }
  document.querySelectorAll("[data-lang-set]").forEach((b) => b.setAttribute("aria-pressed", String(b.dataset.langSet === next)));
  root.classList.remove("i18n-wait");
  document.dispatchEvent(new CustomEvent("maxor:lang", { detail: next }));
}

// An element whose text becomes dynamic leaves the dictionary's hands.
export function setText(el, text) {
  if (!el) return;
  el.removeAttribute("data-i18n");
  original.delete(el);
  el.textContent = text;
}

export async function initI18n() {
  await apply(lang());
  document.querySelectorAll("[data-lang-set]").forEach((b) => b.addEventListener("click", () => {
    const next = b.dataset.langSet;
    if (next === lang()) return;
    try { localStorage.setItem(STORE, next); } catch { /* private mode: it lasts this visit */ }
    // ?lang= in the address would win on the next load: keep it in step with the choice
    const url = new URL(location.href);
    if (url.searchParams.has("lang")) { url.searchParams.set("lang", next); history.replaceState(history.state, "", url); }
    apply(next);
  }));
}
