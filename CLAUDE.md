# Maxor OS: notes for Claude

Maxor OS is a NixOS 26.05 + Hyprland distribution (DankMaterialShell as the shell), by Bryan (bgrados31). This file is
the hand-off between sessions: the rules, how to work here, and where the work stands. Keep it current.

## Rules (from Bryan)

- Talk to Bryan in **Spanish**. Code, comments in English or Spanish matching the file around them.
- **No `Co-Authored-By` lines** in commits. Commit messages in English, imperative subject, a body that says why.
- **Never publish without his OK**: no push to `main`, no release, no tag, no upload, unless he asked in this session.
  Work on branch `development`.
- Never `git reset --hard` or anything that drops changes in a dirty tree. Never push with failing tests.
- Do not spend tokens on the unnecessary: measure, then change; no long explorations without a reason.

## Working with Jev and workers (from Bryan, 2026-10-05; adapted to this project)

Decisions go through Jev (TypeSafe System One; skill `typesafe:typesafe-ai`).

**Jev and profiles**
1. Before starting a Claude worker, ask Jev once to pick a host-approved profile of one or two sessions, each with its
   model and effort. No selection, or Jev unreachable (no key, offline): use the host default and say so to Bryan.
2. The host checks the selection and fixes model and effort before each session starts; they do not change mid-session.
   The host controls permissions and approvals. A worker never publishes, pushes, tags or releases (see Rules).
3. Sonnet 5.5 at medium effort for clear coding (Nix, bash, Go edits). Higher effort for complex reasoning: installer
   stages, LUKS/disk, bootloader, Nix module merging, and JSON contracts (`installer/schema/answers.v1.json`). A second
   Claude only for independent research or a review Bryan asked for.
4. Brief a second Claude with: the task, the exact files, limits (branch `development`, no publishing, no
   `git reset --hard`), and how to know it is done. It returns findings with evidence (command + output) to the lead,
   which verifies them before relying on them.

**Scope**
5. Do what was asked, nothing more: no extra features, tests, files, docs, refactors or review rounds. Ask only when
   blocked or before an action that needs approval. When done and checked, stop and report. Exception: a change that
   alters behaviour documented in `docs/` or in this file updates that text in the same commit.

**Checks before saying "done"** (run the one that covers what changed; if none can run, say which and why)
6. Go in `tui/` → `go test ./...` · bash in `home/maxor/` or `tests/` → `bats tests/` · `.nix`, `flake.nix` →
   `nix eval`/`nix build --dry-run` of `nitro`, plus bats · installer engine or stages → `installer-full` check ·
   `.po`/translations → `scripts/i18n.sh check` · lock screen, wallpaper, Hyprland, greeter → cannot be seen without
   Bryan's screen: say "not verified on screen". Report the real result, including failures. New files `git add`ed
   before any `nix build`.
7. Performance work: measure first (`systemd-analyze blame` / `critical-chain`), change, measure again, and report
   both numbers.

## Layout

- `flake.nix`: systems (`nitro` = Bryan's laptop, `maxor-vm`, `maxor-generic*`, `maxor-iso`), packages, checks.
- `modules/`: the system (machine.nix = per machine options, branding.nix = splash/boot, greeter.nix, fonts.nix…).
- `home/`: home-manager (user.nix, maxor.nix = theme engine + CLI wiring, lockscreen.nix, hyprland/).
- `home/maxor/`: the `maxor` CLI (bash, lib/ + cmd/), tests in `tests/*.bats`.
- `tui/`: `maxor-tui` (Go, Bubble Tea): the full-screen app and the **installer** (`internal/screens/installer*.go`).
- `installer/engine/`: `maxor-install`, the bash engine the TUI drives (stages: preflight, disk, luks, filesystem, host,
  install, bootloader, finish). `installer/iso/`: the ISO. `installer/schema/answers.v1.json`: the TUI → engine contract.
- `themes/<id>/`: colors.json, theme.toml, style.json, optional gradient.json (wallpaper gradient).
- `docs/`: INSTALLER.md, THEMING.md, IDENTITY.md, RELEASING.md, UPDATES.md…
- `web/`: the website (static landing, no build; see web/README.md). Published to GitHub Pages by
  `.github/workflows/pages.yml` from `main`; `python3 web/tools/check.py` runs in CI.

## How to check work

```
cd tui && CGO_ENABLED=0 nix shell nixpkgs#go -c go test ./...      # Go (also runs inside `nix build .#maxor-tui`)
nix shell nixpkgs#bats nixpkgs#jq nixpkgs#openssl nixpkgs#util-linux nixpkgs#ncurses -c bats tests/   # bash, ~228 tests
nix build .#checks.x86_64-linux.installer-full       # installs offline (in Spanish) in a VM and boots it, ~7 min
nix build .#iso                                       # the installer ISO, ~25 min, ~4.6 GB
scripts/tui-gallery.sh OUT                            # photographs every installer step at 4 sizes (vhs), ~2 min
python3 web/tools/check.py                            # the website: i18n, themes in sync, CSP hygiene, links
```

New files must be `git add`ed before any `nix build` (flakes only see tracked files).

## Identity (decided by Bryan, 2026-10-04)

- Palette: light #FF5A57 · #E02F75 · #6700A3 · #050C38; dark #6700A3 · #1B2062 · #050C38.
- Default themes `maxor-dark` (accent #c084ff) and `maxor-light` (accent #bc1f5f); every colour pair ≥ 5:1, checked by
  `tui/internal/theme/theme_test.go`. Other official themes have English ids (ember, dawn, glacier…); old Spanish ids
  are aliases (theme_alias in the CLI, theme.Renamed in the TUI).
- The mark «MAXOR OS» in **Cinzel** (splash, ISO boot menu, Maxor Shell logo, lock screen). UI Figtree, terminal Red Hat
  Mono. The installer offers only Appearance: Dark / Light; installer layout «C» (the mark heads the list of steps).

## Where it stands (2026-10-04)

Done and on `development`: offline install from the ISO in any language (all glibc locales shipped), first boot with
no flicker (theme written before the login, greetd waits only on the first boot), boot ~3.5 s faster (no boot menu with
a single OS; dual boot keeps 3 s), the installer redesign (disk maps, storage, motion, keyboard and time zone proposed
from the language, readable review, failures explained), the new identity above.

Also on branch `claude/vigilant-ptolemy-ng8e4i` (2026-10-04, evening): Hyprland tuned (every setting guarded with `pcall`, Maxor motion curves
`maxor`/`maxor-out`, blur noise/vibrancy, DMS layer blur, idle inhibit on fullscreen); lock screen redesigned
(spaced Cinzel mark on top, 168 px light clock, accent rule, glass card with a pill input and a two-accent gradient
outline, dims at 150 s); wallpaper signature (vignette, `ac2` glow top-right with three orbits, grain; light themes
softer). None of this was seen on a real screen yet: check it on the laptop before the release.

Translations (same branch): standard gettext `.po` for both programs, one workflow (docs/TRANSLATING.md,
`scripts/i18n.sh new|update|status|check`). TUI: `tui/internal/i18n/lang/<code>.po` + `maxor-tui.pot` (English text is
the msgid; `tr`/`trn`/`trc`; `go test ./internal/i18n -update` extracts and merges; Go parser + CLDR plural evaluator in
`po.go`/`plural.go`, no new dependency). CLI: `home/maxor/po/<code>.po` + `maxor-cli.pot` (msgctxt = id, msgid =
English; `lib/lang/en.sh` stays the English source; `lib/i18n.sh` reads the active .po with an embedded POSIX awk,
plurals via bash arithmetic on the Plural-Forms rule validated to `n`/digits/operators; counted keys have a `key#1`
form and their FIRST argument is the number; `%2$s` positional placeholders work; the package sets `MAXOR_PO`, no IFD,
so adding a language touches neither Nix nor tests). All 718 TUI texts and 308 CLI messages translated in es (reviewed
by me) and pt fr de it (machine translations awaiting native review); real plurals everywhere a count appears. Tests
for any future language: placeholders, plural forms/rule validity, stale keys, template up to date, help example
commands intact, layout of every installer step and tab at a small size in each language AND in a 40%-longer
pseudo-language (`i18n.SetPseudo`), fuzzy ignored, hostile Plural-Forms rejected. The TUI runs the CLI with
`MAXOR_LANG=<its language>` and reads Doctor advice by check `id`; `sudo`/`nixos-rebuild` streams run with
`LC_MESSAGES=C`. In bash never name a caller's variable like an internal local (`msg`/`i18n_format` use `__`
names; a collision silently swallowed the result once). Not translated on purpose: engine/nixos-rebuild log,
keyboard layout and language names (system data). Still open: the ISO's boot menu; setting up Weblate (Bryan: create
the project at hosted.weblate.org with the two components listed in docs/TRANSLATING.md, "For maintainers").

Website (2026-10-05, branch `claude/blissful-johnson-99ty2v`): `web/` landing in Spanish + English, live
release card from the GitHub API, the tour desktop painted with the real themes. Cinematic layer (GSAP + ScrollTrigger + Lenis,
vendored), a terminal on the page (Ctrl K) and whole-site theming with `maxor theme apply`. Published on GitHub Pages: after
merging to `development`, run the **Website** workflow by hand (pages.yml only runs by itself on `main`). Its og:image/canonical assume
`https://bgrados31.github.io/maxor-os/`; change them if a domain is bought.

Next, in order:
1. **Verify the latest ISO** (`nix build .#iso`) with a full install in a VM (Spanish, offline) to the desktop.
2. **Release 0.2.0-beta** (pre-release, Bryan chose this): set `VERSION` to `0.2.0-beta`, rebuild the ISO, write its
   SHA-256, create the GitHub **pre-release** `v0.2.0-beta` with notes (what is tested: offline installs in VMs; what is
   not: real hardware, NVIDIA, Secure Boot (ISO unsigned: disable it), install alongside Windows on real hardware, LUKS
   end to end). The ISO (4.6 GB) goes to **SourceForge** (GitHub caps assets at 2 GB): Bryan has to create the account,
   the project `maxor-os` and add an SSH key; then upload with
   `rsync -e ssh FILE USER@frs.sourceforge.net:/home/frs/project/maxor-os/0.2.0-beta/`. `scripts/release.sh` (signed
   manifest for `maxor update`) only takes X.Y.Z and needs Bryan's key password: not for the beta.
3. Waiting on Bryan: apply to his laptop (`sudo nixos-rebuild switch --flake ~/nixos-config#nitro`), test on real
   hardware (NVIDIA hybrid, Windows beside it), decide Limine vs systemd-boot (only after testing with Windows).
4. Ideas noted, not done: zone list shown as «Lima, Perú · UTC−5»,
   ~300 ms black between the greeter's Hyprland and the session's (inherent to the compositor hand-off), the initrd
   takes ~2.2 s (switch-root ~1.1 s): look for savings without losing the splash.

## Known gotchas

- GRUB reads only 8-bit RGB PNGs (ImageMagick gradients come out 16-bit: add `-depth 8 PNG24:`).
- sway splits `exec a; b` at `;`: wrap in `sh -c '…'`.
- Nix disables substituters (even the local `auto`) without a non-loopback address: the engine adds 127.0.0.2 to `lo`
  during an offline nixos-install.
- `fs.protected_regular`: root cannot open a file another user created in /tmp; the TUI uses a private temp dir.
- With `-device virtio-vga-gl` QEMU cannot screendump ("no surface"): use plain `virtio-vga` to capture screens.
- In bash tools, never `pkill -f` / `pgrep -f` with a pattern that also matches your own command line.
