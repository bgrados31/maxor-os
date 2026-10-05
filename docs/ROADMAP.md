# Roadmap

Maxor OS aims to be a complete NixOS-based distribution: a Hyprland desktop, 100 % customizable,
with an app store, a theme store, its own shell, its own boot menu and its own installer.
Experience references: Ryoku OS, Omarchy and end-4's dotfiles.

## Status

| Phase | Goal | Status |
|---|---|---|
| 0 | NixOS + Hyprland + DMS base working | Done |
| 1 | Hyprland only, minimal branding (`os-release`, Plymouth, fastfetch) | Done |
| 2 | Tidy, modular repo, `maxor` CLI (update, rollback, doctor, apps, profiles, backup) | Almost done |
| 3 | Theme engine and official themes | Almost done |
| 4 | Maxor Shell and its own greeter | In progress |
| 5 | Maxor Store (Flatpak apps and themes) | Started (the Store tab of the full-screen app) |
| 6 | Boot menu (Limine), live ISO and installer | Pending |
| 7 | Theme Store on the web, community, documentation and site | Pending |

## Phase 2: tidy repo and CLI

- [x] Repository with documentation, license and CI.
- [x] Split `configuration.nix` into `modules/core.nix` and `modules/desktop.nix`; `branding` and `fonts` were already modules.
- [ ] Extract `modules/shell.nix` and `modules/theming.nix` once the shell of its own exists (phase 4).
- [x] Leave `hosts/<host>` with only what belongs to the machine (boot, GPU, region, user).
- [ ] Export `homeModules.default` to reuse the user configuration.
- [x] Hyprland configuration in Lua modules (`settings`, `rules`, `binds`) and a `user.lua` that is never overwritten.
      Monitors are managed by DMS.
- [x] `maxor update`, `maxor rollback` and `maxor doctor`, with linear output and a single loader.
- [x] `maxor search|install|remove|apps`, profiles, hardware detection, `maxor backup` and `maxor restore`.
- [x] `maxor-tui`: the full-screen app in Go (home, store, themes, update, doctor, profiles, first-run setup).

## Phase 3: theme engine

- [x] Theme format (`colors.json`, `theme.toml`, `wallpaper.png`).
- [x] `maxor theme list | current | apply | undo`.
- [x] Light mode: DMS, the lock screen and the CLI adapt to the theme's `mode`.
- [x] Ten official themes: five dark and five light, all with AA contrast.
- [x] Put Hyprland's shape in the theme (corners, gaps, borders, blur, opacity, animation speed) with `style.json`.
- [ ] Animation curve per theme.
- [ ] Put the bar layout in the theme.
- [x] `maxor theme install <folder|file>` with strict schema and file validation.
- [ ] `maxor theme install <url>` and optional theme signing.
- [x] `maxor theme export`.

## Phase 4: complete identity

Two paths for the shell. It starts with A and migrates to B when there is capacity.

- **A. Patch layer on DankMaterialShell:** Maxor Shell changes the logo, the name and a few texts
  without copying the code, and keeps receiving upstream improvements
  ([SHELL.md](SHELL.md)). DMS is distributed under MIT.
- **B. Own shell in Quickshell (QML):** full control of bar, dock, launcher, control center,
  notifications, lock screen and OSD.

- [x] `maxor-shell` package: "M" logo generated from Krona One, its own name and texts.
- [x] Own greeter: greetd with the DMS greeter and the Maxor Shell package, with the user's theme
      and wallpaper. It replaces SDDM.
- [ ] Krona One font and Maxor brand inside DMS's interface.
- [ ] A settings screen with Maxor's brand.
- [x] Translations: the installer, the full-screen app and the `maxor` tool in es, pt, fr, de and it, as gettext
      `.po` files with real plurals and positional placeholders (see [TRANSLATING.md](TRANSLATING.md)).
- [ ] Set up Weblate for the community and get the machine translations (pt, fr, de, it) reviewed by native speakers.
- [ ] Translate the installation medium's boot menu.
- [ ] Integrate the theme's corner radius with DMS.
- [ ] UWSM session.
- [ ] Decide whether to move to a full fork or to the own shell (path B).

## Phase 5: store

- Backend: Flatpak (Flathub) for user apps and nixpkgs packages for system tools, installable
  without editing `.nix`.
- Frontend: Maxor Store with three tabs: Apps, Themes and Extras. Today the Store, Themes and
  Profiles tabs of the full-screen app cover apps, themes and usage profiles.

## Phase 6: boot and installer

- Boot menu with Limine (with a themed GRUB as plan B). Tested first in a VM and then on a USB
  stick; the bootloader is never changed on a machine with Windows without a backup.
- ISO generated from the flake, with a live Hyprland session.
- Installer: Calamares with Maxor's brand, or one of its own with a choice of theme and profile
  (Gaming, Dev, Minimal, Creator). Automatic GPU detection, including hybrid NVIDIA.
- Own binary cache so that installing takes minutes and not hours.

## Phase 7: community

- A web Theme Store with previews, votes and search.
- Author accounts, theme signing and moderation.
- Public site and documentation.

## Risks and rules

- **Do not replace the bootloader without a backup.** This laptop shares its ESP with Windows.
- Test the ISO and the installer in a VM before real hardware.
- Review and respect the licenses of DMS, Quickshell, Hyprland, Calamares, fonts and wallpapers
  before redistributing (see [THIRD_PARTY.md](../THIRD_PARTY.md)).
- Hyprland moves fast: the version is pinned in `flake.lock` and updated by channel.
- Hybrid NVIDIA is the most fragile part; the installer must be able to fall back to a profile
  with no dedicated GPU.
- A community theme is an attack surface: data only and no execution by default.
