// A small terminal on the website: Ctrl/⌘ K, ` or the >_ button opens it, `help` lists what it understands. It
// speaks the same rail as the real `maxor` CLI and `maxor theme apply` really themes the page. Output is
// built from text nodes only; nothing typed here is ever parsed as HTML or run as code.

import { lang } from "./i18n.js";
import { applySiteTheme, loadThemes, resetSiteTheme, shownTheme, siteTheme, themeById, undoSiteTheme } from "./themes.js";
import { latestRelease } from "./release.js";
import { lenis } from "./motion.js";

const S = {
  es: {
    hello: "Bienvenida a Maxor OS. Escribe help para ver los comandos.",
    help: "Comandos",
    h_list: "los temas oficiales",
    h_apply: "pinta esta página con un tema",
    h_undo: "vuelve al tema anterior",
    h_rollback: "vuelve a Maxor, como una generación anterior",
    h_fetch: "este sistema, en breve",
    h_version: "la última versión publicada",
    h_download: "lleva a la descarga",
    h_clear: "limpia la pantalla",
    h_exit: "cierra la terminal",
    applied: "aplicado: barra · terminal · hyprland · gtk · bloqueo",
    unknownTheme: "No hay un tema «{x}». Mira maxor theme list.",
    usage: "Uso: maxor theme apply <tema>",
    current: "Tema actual",
    dark: "oscuros",
    light: "claros",
    undone: "Vuelto al tema anterior.",
    nothingUndo: "No hay nada que deshacer.",
    rolledBack: "Generación anterior: Maxor, como recién instalado.",
    notFound: "{x}: comando no encontrado. Prueba help.",
    sudo: "Maxor no necesita sudo para esto. Ni para los temas, ni para instalar apps.",
    install: "maxor install {x} funciona en tu Maxor OS. Aquí solo hay una página web.",
    rm: "Buen intento. Aquí no hay nada que borrar, y en Maxor OS volverías atrás con maxor rollback.",
    version: "Última versión",
    noVersion: "Todavía no se pudo leer la versión. Prueba otra vez en un momento.",
    going: "Vamos a la descarga…",
    ls: "escritorio/  temas/  instalador/  descarga/",
    cd: "Aquí no hay carpetas de verdad: usa download, o los enlaces de arriba.",
    man: "No hay manual en la web: escribe help. En Maxor OS, maxor help.",
    arg: "tema",
    whoami: "ana, en una máquina que se configura en un archivo.",
    hyprland: "Hyprland 0.55, configurado en Lua y animado a 60 fps (o lo que dé tu pantalla).",
  },
  en: {
    hello: "Welcome to Maxor OS. Type help to see the commands.",
    help: "Commands",
    h_list: "the official themes",
    h_apply: "paints this page with a theme",
    h_undo: "back to the previous theme",
    h_rollback: "back to Maxor, like an earlier generation",
    h_fetch: "this system, in short",
    h_version: "the latest published release",
    h_download: "takes you to the download",
    h_clear: "clears the screen",
    h_exit: "closes the terminal",
    applied: "applied: bar · terminal · hyprland · gtk · lock",
    unknownTheme: "There is no «{x}» theme. See maxor theme list.",
    usage: "Usage: maxor theme apply <theme>",
    current: "Current theme",
    dark: "dark",
    light: "light",
    undone: "Back to the previous theme.",
    nothingUndo: "Nothing to undo.",
    rolledBack: "Previous generation: Maxor, as freshly installed.",
    notFound: "{x}: command not found. Try help.",
    sudo: "Maxor needs no sudo for this. Not for themes, not for installing apps.",
    install: "maxor install {x} works on your Maxor OS. Here there is only a web page.",
    rm: "Nice try. There is nothing to delete here, and on Maxor OS you'd go back with maxor rollback.",
    version: "Latest release",
    noVersion: "The release could not be read yet. Try again in a moment.",
    going: "Off to the download…",
    ls: "desktop/  themes/  installer/  download/",
    cd: "There are no real folders here: try download, or the links above.",
    man: "No manual on the web: type help. On Maxor OS, maxor help.",
    arg: "theme",
    whoami: "ana, on a machine configured in one file.",
    hyprland: "Hyprland 0.55, configured in Lua and animated at 60 fps (or whatever your screen does).",
  },
};
const s = (k, x = "") => (S[lang()][k] ?? S.es[k]).replace("{x}", x);

const COMMANDS = ["help", "maxor theme list", "maxor theme apply ", "maxor theme current", "maxor theme undo",
  "maxor rollback", "maxor version", "fastfetch", "download", "clear", "exit"];

export function initTerm() {
  const dlg = document.querySelector("[data-term]");
  if (!dlg || typeof dlg.showModal !== "function") return;
  const out = dlg.querySelector("[data-term-out]");
  const form = dlg.querySelector("[data-term-form]");
  const input = dlg.querySelector("[data-term-input]");
  const past = [];
  let cursor = 0;
  let greeted = false;
  let opener = null;

  // ── printing ──
  function line(...parts) {
    const p = document.createElement("p");
    for (const part of parts) {
      const [text, cls] = Array.isArray(part) ? part : [part, null];
      const span = document.createElement("span");
      if (cls) span.className = cls;
      span.textContent = text;
      p.append(span);
    }
    out.append(p);
    out.scrollTop = out.scrollHeight;
  }
  const rail = (title, rows, end) => {
    line(["┌  ", "t-ac"], [title, "t-b"]);
    line(["│", "t-ac"]);
    for (const r of rows) line(["◇  ", "t-ac"], ...r);
    if (end) line(["└  ", "t-ac"], [end, "t-mu"]);
  };

  // ── commands ──
  async function themeList() {
    const themes = await loadThemes();
    const cur = shownTheme();
    for (const mode of ["dark", "light"]) {
      line([`${s(mode)}`, "t-mu"]);
      for (const th of themes.filter((t) => t.mode === mode)) {
        line([th.id === cur ? "  ● " : "  ○ ", "t-ac"], [th.id.padEnd(13), th.id === cur ? "t-b" : ""], [th.name, "t-mu"]);
      }
    }
  }

  async function run(raw) {
    const cmd = raw.trim().replace(/\s+/g, " ");
    line(["~ ❯ ", "t-ac"], [cmd]);
    if (!cmd) return;
    const low = cmd.toLowerCase();
    const [w0, w1, w2, w3] = low.split(" ");

    if (low === "help" || low === "maxor help" || low === "?") {
      const rows = [
        ["maxor theme list", "h_list"], [`maxor theme apply <${s("arg")}>`, "h_apply"], ["maxor theme undo", "h_undo"],
        ["maxor rollback", "h_rollback"], ["fastfetch", "h_fetch"], ["maxor version", "h_version"],
        ["download", "h_download"], ["clear", "h_clear"], ["exit", "h_exit"],
      ];
      rail(s("help"), rows.map(([c, k]) => [[c.padEnd(26), "t-b"], [s(k), "t-mu"]]));
    } else if (low === "clear") {
      out.replaceChildren();
    } else if (low === "exit" || low === "logout") {
      dlg.close();
    } else if (w0 === "maxor" && w1 === "theme" && (w2 === "list" || !w2)) {
      await themeList();
    } else if (w0 === "maxor" && w1 === "theme" && w2 === "current") {
      const th = (await themeById(shownTheme()));
      line([`${s("current")}: `, "t-mu"], [th?.name ?? "Maxor Dark", "t-b"]);
    } else if (w0 === "maxor" && w1 === "theme" && w2 === "apply") {
      if (!w3) { line([s("usage"), "t-bad"]); return; }
      const th = await themeById(w3);
      if (!th) { line([s("unknownTheme", cmd.split(" ")[3]), "t-bad"]); return; }
      await applySiteTheme(th.id);
      rail("maxor theme", [[[th.name, "t-b"]]], s("applied"));
    } else if (w0 === "maxor" && w1 === "theme" && w2 === "undo") {
      line([await undoSiteTheme() ? s("undone") : s("nothingUndo"), "t-mu"]);
    } else if (low === "maxor rollback" || low === "rollback" || low === "nixos-rebuild switch --rollback") {
      resetSiteTheme();
      rail("maxor rollback", [[[s("rolledBack"), "t-b"]]]);
    } else if (low === "fastfetch" || low === "neofetch") {
      const th = (await themeById(shownTheme()));
      const rows = [["OS", "Maxor OS · NixOS 26.05"], ["WM", "Hyprland 0.55"], ["Shell", "fish"], ["Theme", th?.name ?? "Maxor Dark"],
        ["Font", "Figtree · Red Hat Mono"], ["Browser", navigator.userAgent.match(/(Firefox|Edg|Chrome|Safari)\/[\d.]+/)?.[0] ?? "—"]];
      line(["ana", "t-b"], ["@", "t-mu"], ["maxor", "t-b"]);
      for (const [k, v] of rows) line([k.padEnd(9), "t-ac"], [v]);
    } else if (low === "maxor version" || low === "maxor --version") {
      const r = latestRelease();
      if (r) line([`${s("version")}: `, "t-mu"], [r.tag, "t-b"], [r.pre ? "  pre-release" : "", "t-mu"]);
      else line([s("noVersion"), "t-mu"]);
    } else if (low === "download" || low === "maxor download") {
      line([s("going"), "t-mu"]);
      setTimeout(() => {
        dlg.close();
        const target = document.getElementById("descargar");
        if (lenis()) lenis().scrollTo(target, { offset: -90 }); else target?.scrollIntoView();
      }, 350);
    } else if (w0 === "sudo") {
      line([s("sudo"), "t-mu"]);
    } else if (w0 === "maxor" && w1 === "install") {
      line([s("install", w2 ?? "firefox"), "t-mu"]);
    } else if (/^rm\s+-[a-z]*r[a-z]*f?\b|^rm\s+-[a-z]*f[a-z]*r/.test(low)) {
      line([s("rm"), "t-mu"]);
    } else if (w0 === "ls" || w0 === "dir") {
      line([s("ls"), "t-b"]);
    } else if (w0 === "cd") {
      line([s("cd"), "t-mu"]);
    } else if (low === "pwd") {
      line(["/home/ana/maxor-os"]);
    } else if (w0 === "echo") {
      line([cmd.slice(5)]);
    } else if (low === "date") {
      line([new Intl.DateTimeFormat(lang(), { dateStyle: "full", timeStyle: "short" }).format(new Date())]);
    } else if (w0 === "uname") {
      line(["Linux maxor 6.x NixOS 26.05 x86_64 GNU/Linux"]);
    } else if (low === "history") {
      past.forEach((c, i) => line([String(i + 1).padStart(4) + "  ", "t-mu"], [c]));
    } else if (w0 === "man" || low === "--help" || low === "maxor --help") {
      line([s("man"), "t-mu"]);
    } else if (low === "whoami") {
      line([s("whoami")]);
    } else if (low === "hyprland" || low === "hyprctl version") {
      line([s("hyprland")]);
    } else {
      line([s("notFound", w0.slice(0, 40)), "t-bad"]);
    }
  }

  // ── open / close ──
  function open(from) {
    if (dlg.open) return;
    opener = from ?? document.activeElement;
    dlg.showModal();
    lenis()?.stop();
    if (!greeted) { line([s("hello"), "t-mu"]); greeted = true; }
    input.focus();
  }
  dlg.addEventListener("close", () => { lenis()?.start(); opener?.focus?.(); });
  dlg.addEventListener("click", (e) => {
    if (e.target !== dlg) return;
    const r = dlg.getBoundingClientRect();
    const inside = e.clientX >= r.left && e.clientX <= r.right && e.clientY >= r.top && e.clientY <= r.bottom;
    if (!inside) dlg.close(); // a click on the backdrop
  });
  dlg.querySelector("[data-term-close]")?.addEventListener("click", () => dlg.close());
  document.querySelectorAll("[data-term-open]").forEach((b) => b.addEventListener("click", () => open(b)));
  document.addEventListener("keydown", (e) => {
    const typingElsewhere = e.target.closest?.("input, textarea, [contenteditable]");
    // Ctrl/⌘ K anywhere; the backquote too where the keyboard has it as a plain key (not a dead key).
    const combo = e.key.toLowerCase() === "k" && (e.ctrlKey || e.metaKey) && !e.altKey;
    const tick = e.key === "`" && !typingElsewhere && !e.ctrlKey && !e.metaKey && !e.altKey;
    if (combo || tick) { e.preventDefault(); if (dlg.open) dlg.close(); else open(); }
  });

  form.addEventListener("submit", (e) => {
    e.preventDefault();
    const v = input.value.slice(0, 200);
    input.value = "";
    if (v.trim()) past.push(v);
    cursor = past.length;
    run(v);
  });
  input.addEventListener("keydown", (e) => {
    if (e.key === "ArrowUp" && cursor > 0) { e.preventDefault(); input.value = past[--cursor]; }
    else if (e.key === "ArrowDown") { e.preventDefault(); cursor = Math.min(past.length, cursor + 1); input.value = past[cursor] ?? ""; }
    else if (e.key === "Tab") {
      e.preventDefault();
      const v = input.value;
      loadThemes().then((themes) => {
        const all = [...COMMANDS, ...themes.map((t) => `maxor theme apply ${t.id}`)];
        const hits = all.filter((c) => c.startsWith(v) && c !== v);
        if (hits.length === 1) input.value = hits[0];
        else if (hits.length > 1) {
          let pre = hits[0];
          for (const h of hits) while (!h.startsWith(pre)) pre = pre.slice(0, -1);
          if (pre.length > v.length) input.value = pre;
          else line([hits.map((h) => h.split(" ").pop()).join("  "), "t-mu"]);
        }
      });
    } else if (e.key === "l" && e.ctrlKey) { e.preventDefault(); out.replaceChildren(); }
  });
}
