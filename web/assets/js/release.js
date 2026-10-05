// The download card and the hero chip, read live from GitHub's public API (no token: 60 requests an
// hour per visitor, so the answer is kept for 15 minutes in this tab and the last good one survives an
// outage). Everything that comes back is treated as untrusted: links must point where we expect, text
// is set with textContent, and the release notes are rebuilt from a tiny Markdown subset as DOM nodes.

import { lang, setText, t } from "./i18n.js";

const REPO = "bgrados31/maxor-os";
const API = `https://api.github.com/repos/${REPO}/releases?per_page=5`;
const FRESH_MS = 15 * 60 * 1000;
const CACHE = "maxor-release-v1";

const ALLOWED = [
  new RegExp(`^https://github\\.com/${REPO}/`),
  /^https:\/\/sourceforge\.net\/projects\/maxor-os\//,
  /^https:\/\/downloads\.sourceforge\.net\/project\/maxor-os\//,
];
const safeUrl = (u) => (typeof u === "string" && ALLOWED.some((re) => re.test(u)) ? u : null);
const TAG = /^v?\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$/;

function store(kind) {
  try { return kind === "session" ? sessionStorage : localStorage; } catch { return null; }
}
function readCache(kind) {
  try { return JSON.parse(store(kind)?.getItem(CACHE) ?? "null"); } catch { return null; }
}
function writeCache(data) {
  const v = JSON.stringify({ at: Date.now(), data });
  try { store("session")?.setItem(CACHE, v); store("local")?.setItem(CACHE, v); } catch { /* full or blocked */ }
}

// Only what the page uses, validated, so a cached copy can never carry anything else.
function pick(list) {
  if (!Array.isArray(list)) throw new Error("unexpected response");
  const r = list.find((x) => x && !x.draft && typeof x.tag_name === "string" && TAG.test(x.tag_name));
  if (!r) return null;
  const body = typeof r.body === "string" ? r.body.slice(0, 20000) : "";
  const assets = (Array.isArray(r.assets) ? r.assets : [])
    .map((a) => ({ name: String(a?.name ?? ""), size: Number(a?.size) || 0, url: safeUrl(a?.browser_download_url) }))
    .filter((a) => a.url);
  return {
    tag: r.tag_name,
    name: typeof r.name === "string" && r.name.trim() ? r.name.trim().slice(0, 120) : r.tag_name,
    pre: Boolean(r.prerelease),
    date: typeof r.published_at === "string" ? r.published_at : null,
    url: safeUrl(r.html_url) ?? `https://github.com/${REPO}/releases`,
    body,
    assets,
  };
}

async function fetchRelease() {
  const cached = readCache("session");
  if (cached && Date.now() - cached.at < FRESH_MS) return cached.data;
  const ctl = new AbortController();
  const timer = setTimeout(() => ctl.abort(), 8000);
  try {
    const res = await fetch(API, {
      headers: { Accept: "application/vnd.github+json" },
      credentials: "omit", referrerPolicy: "no-referrer", signal: ctl.signal,
    });
    if (!res.ok) throw new Error(`GitHub ${res.status}`);
    const data = pick(await res.json());
    writeCache(data);
    return data;
  } catch (err) {
    const stale = readCache("local");
    if (stale && "data" in stale) return stale.data;
    throw err;
  } finally {
    clearTimeout(timer);
  }
}

// ── Markdown, the little the notes use: ### headings, - lists, paragraphs, **bold**, `code`, [text](url) ──
function inline(text) {
  const out = [];
  const re = /\*\*(.+?)\*\*|`([^`]+)`|\[([^\]]+)\]\([^)]+\)/g;
  let last = 0;
  for (const m of text.matchAll(re)) {
    if (m.index > last) out.push(document.createTextNode(text.slice(last, m.index)));
    if (m[1] !== undefined) { const b = document.createElement("strong"); b.append(...inline(m[1])); out.push(b); }
    else if (m[2] !== undefined) { const c = document.createElement("code"); c.textContent = m[2]; out.push(c); }
    else out.push(document.createTextNode(m[3]));
    last = m.index + m[0].length;
  }
  if (last < text.length) out.push(document.createTextNode(text.slice(last)));
  return out;
}

function notes(body, limit = 7) {
  const nodes = [];
  let list = null;
  let items = 0;
  let hidden = 0;
  for (const raw of body.split(/\r?\n/)) {
    const line = raw.trim();
    if (!line) { list = null; continue; }
    const h = line.match(/^#{1,6}\s+(.*)$/);
    const li = line.match(/^[-*]\s+(.*)$/);
    if (items >= limit) { if (li) hidden++; continue; }
    if (h) {
      const el = document.createElement("h4");
      el.append(...inline(h[1]));
      nodes.push(el);
      list = null;
    } else if (li) {
      if (!list) { list = document.createElement("ul"); nodes.push(list); }
      const el = document.createElement("li");
      el.append(...inline(li[1].length > 260 ? li[1].slice(0, 257).trimEnd() + "…" : li[1]));
      list.append(el);
      items++;
    } else if (!/^[-*_]{3,}$/.test(line) && !/\b[a-f0-9]{64}\b/i.test(line)) {
      const p = document.createElement("p");
      p.append(...inline(line.length > 300 ? line.slice(0, 297) + "…" : line));
      nodes.push(p);
      list = null;
    }
  }
  while (nodes.length && nodes[nodes.length - 1].tagName === "H4") nodes.pop();
  if (hidden) {
    const p = document.createElement("p");
    p.textContent = t("rel.more", { n: hidden });
    nodes.push(p);
  }
  return nodes;
}

function size(bytes) {
  const units = ["B", "KB", "MB", "GB"];
  let i = 0;
  let n = bytes;
  while (n >= 1000 && i < units.length - 1) { n /= 1000; i++; }
  return `${new Intl.NumberFormat(lang(), { maximumFractionDigits: 1 }).format(n)} ${units[i]}`;
}

function when(iso) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const days = Math.round((d.getTime() - Date.now()) / 86_400_000);
  const rel = Math.abs(days) < 45
    ? new Intl.RelativeTimeFormat(lang(), { numeric: "auto" }).format(days, "day")
    : new Intl.DateTimeFormat(lang(), { dateStyle: "long" }).format(d);
  return t("rel.published", { rel });
}

// The ISO: a release asset if it fits on GitHub, else the SourceForge link in the notes (GitHub caps
// assets at 2 GB, the ISO is ~4.6 GB).
function findIso(r) {
  const asset = r.assets.find((a) => /\.iso$/i.test(a.name));
  if (asset) return { url: asset.url, size: asset.size };
  for (const m of r.body.matchAll(/https:\/\/[^\s)<>"'\]]+/g)) {
    const u = safeUrl(m[0]);
    if (u && /sourceforge\.net/.test(u)) return { url: u, size: 0 };
  }
  return null;
}

function render(r, state) {
  const card = document.querySelector("[data-release]");
  const chip = document.querySelector("[data-release-chip]");
  if (!card) return;
  const $ = (s) => card.querySelector(s);
  card.setAttribute("aria-busy", "false");
  card.querySelectorAll(".skeleton").forEach((s) => s.classList.remove("skeleton"));

  const tagEl = $("[data-release-tag]");
  const badge = $("[data-release-badge]");
  const dateEl = $("[data-release-date]");
  const nameEl = $("[data-release-name]");
  const notesEl = $("[data-release-notes]");
  const primary = $("[data-release-primary]");
  const primaryText = $("[data-release-primary-text]");
  const link = $("[data-release-link]");
  const sha = $("[data-release-sha]");

  $(".release__head").hidden = !r;
  if (!r) {
    badge.hidden = true;
    setText(dateEl, "");
    setText(nameEl, state === "error" ? t("rel.error") : t("rel.none"));
    const p = document.createElement("p");
    p.textContent = state === "error" ? t("rel.errorBody") : t("rel.noneBody");
    notesEl.replaceChildren(p);
    primary.href = `https://github.com/${REPO}/releases`;
    setText(primaryText, t("rel.all"));
    link.hidden = true;
    sha.hidden = true;
    return;
  }

  setText(tagEl, r.tag);
  badge.hidden = false;
  badge.className = "badge" + (r.pre ? " badge--pre" : "");
  badge.textContent = r.pre ? t("rel.pre") : t("rel.stable");
  setText(dateEl, r.date ? when(r.date) : "");
  setText(nameEl, r.name);
  notesEl.replaceChildren(...notes(r.body));

  const iso = findIso(r);
  primary.href = iso ? iso.url : r.url;
  setText(primaryText, iso ? (iso.size ? t("rel.isoSize", { size: size(iso.size) }) : t("rel.iso")) : t("rel.view", { tag: r.tag }));
  link.href = r.url;
  link.hidden = !iso; // without an ISO the main button already opens the release

  const hash = r.body.match(/\b[a-f0-9]{64}\b/i);
  sha.hidden = !hash;
  if (hash) sha.querySelector("[data-release-sha-value]").textContent = hash[0].toLowerCase();

  if (chip) {
    chip.toggleAttribute("data-pre", r.pre);
    setText(chip.querySelector("[data-release-chip-text]"), t(r.pre ? "chip.pre" : "chip.latest", { tag: r.tag }));
  }
}

export function initRelease() {
  let last = { r: null, state: "loading" };
  const draw = () => { if (last.state !== "loading") render(last.r, last.state); };
  document.addEventListener("maxor:lang", draw);

  if (!document.querySelector("[data-release]")) return;
  const go = () => {
    fetchRelease()
      .then((r) => { last = { r, state: r ? "ok" : "none" }; })
      .catch(() => { last = { r: null, state: "error" }; })
      .finally(draw);
  };
  // The hero chip shows the version too, so ask soon, but after the first paint.
  if ("requestIdleCallback" in window) requestIdleCallback(go, { timeout: 1500 });
  else setTimeout(go, 300);
}
