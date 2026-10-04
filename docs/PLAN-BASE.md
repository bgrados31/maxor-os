# Base plan

What has to be built so that Maxor OS is solid before the store, the installer and the
community. It complements the [roadmap](ROADMAP.md): the roadmap says *which phases there are*,
this plan says *in what order and with what decisions*.

## Principles

1. **Everything goes through the CLI.** Every graphical app is a face of `maxor …`; the logic
   lives in a single place and every graphical command has a terminal equivalent. For that the CLI
   gets `--json` on every subcommand.
2. **A single interface technology per layer.** The CLI is bash with `--json` as the boundary; the
   full-screen app is Go (Bubble Tea) and calls the CLI; graphical apps will be Quickshell (QML),
   the same base as DMS. What is written for the apps is reused for the own shell later.
3. **A single source of identity.** Colors, fonts, shapes and logos come from *design tokens* (a
   JSON); the shell, the apps, the greeter, Plymouth and the web consume them. Changing the brand
   is touching one file.
4. **It never breaks.** Every change is built before asking for the `switch`; there are fallback
   generations; what the user touches (`~/.config/maxor/`) is not overwritten by the system.
5. **Replace, do not rewrite.** The own shell is born piece by piece on top of the current one.

## M1. Solid base (optimization and health)

Done: zram, TRIM, thermald, power-profiles, bounded journald, Nix GC and optimise, quiet boot and
shutdown, clean session, Thunar.

- [x] `maxor update` with a diff summary (updated, new, removed) and a reboot notice if the kernel changes.
- [ ] Own binary cache (Cachix) and substituters configured from the flake.
- [x] `nix-ld` + AppImage (`programs.appimage`) so that foreign binaries work without fighting Nix.
- [x] Flatpak and Flathub active from the first boot (the store uses them).
- [x] `earlyoom`. Pending: evaluate a newer or `zen` kernel and an `scx` scheduler.
- [x] Profiles `maxor profile enable|disable` (gaming, dev, creator, office) with their own catalog ([APPS.md](APPS.md)).
- [x] Per-machine drivers: CPU/GPU/laptop/VM detection with `maxor hardware` and `modules/hardware.nix` ([HARDWARE.md](HARDWARE.md)).
- [ ] Graphics: guided PRIME switching with `maxor gpu` and a "no dedicated GPU" mode.
- [x] `maxor doctor --json`, with a suggested command per check. Pending: `maxor doctor --fix` to run them unattended.
- [x] `maxor backup` (user configuration) and `maxor restore`.
- [x] User apps without editing `.nix`: `maxor search|install|remove|apps`, nixpkgs and Flathub, with `--json` ([APPS.md](APPS.md)).
- [x] Full-screen app (`maxor-tui`) with store, themes, update (with rollback), doctor and profiles.
- [ ] UWSM session (ordered user services, clean shutdown).
- [ ] Firmware (`fwupd`), printing (CUPS) and codecs: what "simply must work".
- [ ] Tests: `nix flake check` with a VM boot test of the configuration.

## M2. Interface library and own apps

`maxor-ui` library (QML): window, cards, buttons, lists, switches, dialogs, icons, typography and
animations, all reading the design tokens and the active theme.

Apps, in order:

| App | What for | Talks to |
|---|---|---|
| **Maxor Welcome** | First boot: language, theme, profile, GPU, accounts, shortcut tour | `maxor profile`, `maxor theme` |
| **Maxor Settings** | Settings center: themes, shortcuts, displays, profile, graphics, updates, backups | the whole CLI |
| **Maxor Update / Recovery** | Update, see changes, go back to a generation, safe mode | `maxor update`, `rollback` |
| **Maxor Theme Studio** | Edit a theme live (palette, shape, wallpaper) and export it | `maxor theme` |
| **Maxor Store** (phase 5) | Apps (Flatpak/Nix), themes, extras | `maxor install`, `theme install` |
| **Maxor Installer** (phase 6) | Install the system | `nixos-install` + `maxor` |

## M3. Perfect branding

A single pass that covers every touch point, with the identity already set in
[IDENTITY.md](IDENTITY.md):

- [ ] Logo suite: text mark, small version, monochrome, app icons, favicon.
- [ ] Design tokens (`branding/tokens.json`) and a generator that produces CSS/QML/Nix/Lua from them.
- [ ] Own icon and cursor themes (or a curated selection with the theme's palette).
- [ ] GTK/Qt/libadwaita consistency with the active theme, no "windows from another system".
- [ ] Complete boot with a single palette: boot menu, Plymouth, greeter, desktop.
- [ ] Complete `os-release` (`LOGO`, `ANSI_COLOR`, `HOME_URL`, `SUPPORT_URL`), fastfetch and `issue`.
- [ ] Krona One and the brand inside DMS's interface; settings with the brand; translations.
- [ ] System sounds (startup, notification, error) with the same character.
- [ ] Official wallpapers per theme and screenshots for the README.
- [ ] Site and documentation with the same design system.

## M4. Own shell

Decision: **gradual strangling**, not a total rewrite.

1. Meanwhile, Maxor Shell remains the patch layer on DMS (it already works).
2. The M2 apps build `maxor-ui`, which is the foundation of the own shell.
3. One piece is replaced at a time, starting with the most visible and isolated ones: **lock
   screen**, **OSD** (volume/brightness), **launcher**, **notifications**, **control center**,
   **bar**. Each own piece coexists with DMS through the same IPC until the last component is
   replaced and DMS stops being a dependency.
4. Exit criterion for each piece: as stable as DMS's, with the tokens and the theme.

Risk: maintaining both during the transition. Mitigation: small pieces, one at a time, each behind
an option (`maxor.shell.<piece> = "dms" | "maxor"`).

## Proposed order

1. M1 (base) and README screenshots: a few sessions, the system ends up healthy and presentable.
2. Design tokens (first step of M3), because everything else consumes them.
3. `maxor-ui` + Welcome + Settings (M2).
4. The rest of M3 (branding) with the apps that already exist.
5. First pieces of the own shell (M4) and, in parallel, the store (phase 5).
6. Boot menu, ISO and installer (phase 6) at the end, with the base stable.
