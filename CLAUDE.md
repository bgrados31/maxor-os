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

## How to check work

```
cd tui && CGO_ENABLED=0 nix shell nixpkgs#go -c go test ./...      # Go (also runs inside `nix build .#maxor-tui`)
nix shell nixpkgs#bats nixpkgs#jq nixpkgs#openssl nixpkgs#util-linux nixpkgs#ncurses -c bats tests/   # bash, ~228 tests
nix build .#checks.x86_64-linux.installer-full       # installs offline (in Spanish) in a VM and boots it, ~7 min
nix build .#iso                                       # the installer ISO, ~25 min, ~4.6 GB
scripts/tui-gallery.sh OUT                            # photographs every installer step at 4 sizes (vhs), ~2 min
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

Also on branch `claude/vigilant-ptolemy-ng8e4i` (2026-10-04, evening; not yet merged into `development`): Hyprland tuned (every setting guarded with `pcall`, Maxor motion curves
`maxor`/`maxor-out`, vfr, blur noise/vibrancy, DMS layer blur, idle inhibit on fullscreen); lock screen redesigned
(spaced Cinzel mark on top, 168 px light clock, accent rule, glass card with a pill input and a two-accent gradient
outline, dims at 150 s); wallpaper signature (vignette, `ac2` glow top-right with three orbits, grain; light themes
softer). None of this was seen on a real screen yet: check it on the laptop before the release.

Translations (same branch), two catalogs with one workflow (docs/TRANSLATING.md): the TUI (`tui/internal/i18n`,
English text as key, `lang/<code>.json`, `go test ./internal/i18n -update` syncs them) and the `maxor` CLI
(`home/maxor/lib/lang/<code>.sh`, id as key, `scripts/i18n.sh new|missing|status`; the package picks up every file of
that folder, no Nix edit). Everything the user reads is translated: installer, every TUI screen, palette, help, and
all 304 CLI messages (help texts included). Languages: `es` reviewed by me, `pt fr de it` machine translations
awaiting native review. Tests (apply to any future language): same `%s` in the same order, no stale keys, help example
commands unchanged, shellcheck-safe quoting (no typographic apostrophes), reviewed languages complete, every installer
step and every tab fits a small window in each language. The TUI runs the CLI with `MAXOR_LANG=<its language>` and
reads Doctor advice by check `id`, not by text; `sudo`/`nixos-rebuild` streams run with `LC_MESSAGES=C` because the
runner recognises their English lines. Palette command words (`go`, `theme`, `search`, `open`) stay English on
purpose. Not translated on purpose: the engine/nixos-rebuild log, keyboard layout and language names (system data).
Still open: the live ISO's boot menu; the maxor man page / completions if they ever carry text.

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
4. Ideas noted, not done: translate the installer UI (today English only), zone list shown as «Lima, Perú · UTC−5»,
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
