<div align="center">

```
 ███╗   ███╗ █████╗ ██╗  ██╗ ██████╗ ██████╗
 ████╗ ████║██╔══██╗╚██╗██╔╝██╔═══██╗██╔══██╗
 ██╔████╔██║███████║ ╚███╔╝ ██║   ██║██████╔╝
 ██║╚██╔╝██║██╔══██║ ██╔██╗ ██║   ██║██╔══██╗
 ██║ ╚═╝ ██║██║  ██║██╔╝ ██╗╚██████╔╝██║  ██║
 ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝
                  O  S
```

**A declarative Hyprland desktop, themeable with one command and reproducible on any machine.**

[![CI](https://github.com/bgrados31/maxor-os/actions/workflows/check.yml/badge.svg)](https://github.com/bgrados31/maxor-os/actions/workflows/check.yml)
![NixOS 26.05](https://img.shields.io/badge/NixOS-26.05-5277C3?logo=nixos&logoColor=white)
![Hyprland](https://img.shields.io/badge/Hyprland-0.55-58E1FF)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

[Install](docs/INSTALL.md) · [Architecture](docs/ARCHITECTURE.md) · [Theming](docs/THEMING.md) · [Identity](docs/IDENTITY.md) · [Roadmap](docs/ROADMAP.md)

</div>

---

## What it is

Maxor OS is a NixOS configuration that turns a clean install into a complete Hyprland desktop
with its own identity. The whole system (boot, shell, terminal, lock screen, fonts and themes)
lives in one flake: a change is a file, and going back is picking the previous generation in the
boot menu.

The long-term goal is a full distribution with its own app store, theme store and installer.
Today the project sits between **phases 2 and 3 of 7** (see the [roadmap](docs/ROADMAP.md)).

## What is included

| Layer | Components |
|---|---|
| **Session** | Hyprland 0.55 configured in Lua, `xdg-desktop-portal-hyprland` and `-gtk` portals, login with greetd |
| **Shell** | **Maxor Shell** ([DankMaterialShell](https://github.com/AvengeMedia/DankMaterialShell) with Maxor's identity): bar, launcher, notifications, control center and login screen |
| **Lock screen** | hyprlock with its own design (spaced Cinzel mark, light clock, glass card, gradient password field) and idle handling with hypridle that dims, locks and sleeps the screen; the palette comes from the active theme |
| **Languages** | the installer, the full-screen app and the `maxor` tool speak Spanish, Portuguese, French, German and Italian; plurals and word order follow each language, and a new language is a standard gettext file anyone can send ([how](docs/TRANSLATING.md)) |
| **Theme engine** | `maxor theme` CLI: ten themes (five light and five dark), install and export, no `rebuild` and no `sudo` |
| **Terminal** | kitty, fish, starship, zoxide, eza, bat, btop, fastfetch |
| **Boot** | systemd-boot with an XBOOTLDR partition, its own Plymouth theme and a quiet boot |
| **Graphics** | Intel + NVIDIA with PRIME offload (`nvidia-offload <app>`), drivers picked from the detected hardware |
| **Apps** | `maxor` installs from nixpkgs and Flathub without `sudo`, usage profiles (gaming, dev, creator, office) |
| **Identity** | `os-release` as "Maxor OS", Figtree, Red Hat Mono and Krona One, Papirus icons and the Bibata cursor |

## Screenshots

> Pending: screenshots will be added to `docs/img/` once phase 4 (the shell) is finished.

## Quick start

Requires NixOS 25.11 or later with flakes enabled. The reference host is `nitro`
(Acer Nitro AN16, Intel + NVIDIA); for another machine read the
[install guide](docs/INSTALL.md) first, which explains what to adapt.

```sh
git clone https://github.com/bgrados31/maxor-os ~/nixos-config
cd ~/nixos-config
sudo nixos-rebuild switch --flake .#nitro
```

After rebooting, the Maxor login goes straight into Hyprland. If something fails, pick an
earlier generation in the boot menu or, from a TTY (`Ctrl+Alt+F3`):

```sh
sudo nixos-rebuild switch --rollback
```

## Themes

```sh
maxor theme list              # 5 dark and 5 light
maxor theme apply glacier     # bar, kitty, Hyprland, GTK and lock screen change together
maxor theme undo              # back to the previous theme
```

Twelve official themes: Maxor Dark (the default) and Maxor Light, then Sakura, Glacier, Obsidian, Ember,
Ultraviolet, Dawn, Frost, Paper, Breeze and Amber. A theme is a folder of data
(`colors.json`, `theme.toml`, `wallpaper.png`) and never runs code. Details and how to make your
own in [docs/THEMING.md](docs/THEMING.md).

## The `maxor` tool

```sh
maxor update       # updates, shows the changes and asks before applying
maxor rollback     # goes back to a previous generation
maxor doctor       # diagnostics: services, session, GPU, boot, fonts
maxor install btop # installs an app without sudo or a rebuild
maxor backup       # saves your setup (profiles, themes, apps) in one file
```

The output is linear, hangs from a rail and uses the colors of the active theme. Plain `maxor`
opens the full-screen app (store, themes, updates, doctor, profiles). Reference in
[docs/CLI.md](docs/CLI.md) and [docs/TUI.md](docs/TUI.md).

## Main shortcuts

| Shortcut | Action |
|---|---|
| `SUPER + Enter` / `T` | Terminal (kitty) |
| `SUPER + Space` / `D` | Launcher |
| `SUPER + ,` | DMS settings (bar, wallpaper, theme) |
| `SUPER + Y` | Wallpaper picker |
| `SUPER + Q` | Close window |
| `SUPER + 1-9` | Switch workspace |
| `SUPER + TAB` | Overview |
| `SUPER + V` | Clipboard history |
| `SUPER + X` | Power menu |
| `SUPER + ALT + L` | Lock screen |
| `Print` / `SUPER + SHIFT + S` | Screenshot |
| `SUPER + SHIFT + /` | Full list of shortcuts |

## Layout

```
flake.nix                      inputs: nixpkgs 26.05, home-manager, DMS
hosts/nitro/                   what belongs to the machine: boot, NVIDIA, region, user
modules/core.nix               Nix, network, audio, Bluetooth, base packages
modules/desktop.nix            Hyprland, PAM, desktop services
modules/greeter.nix            login: greetd and the Maxor Shell greeter
modules/hardware.nix           drivers from the detected hardware
modules/profiles.nix           usage profiles (gaming, dev, creator, office)
modules/branding.nix           system name, Plymouth, quiet boot
modules/fonts.nix              Figtree, Red Hat Mono, Krona One
packages/                      Maxor Shell, the CLI, the full-screen app, the logo generator
home/bryan.nix                 user: kitty, fish, GTK/Qt, apps
home/hyprland.nix              loads Hyprland's Lua modules
home/hyprland/                 settings.lua, rules.lua, binds.lua and user.lua.example
home/lockscreen.nix            hyprlock and hypridle
home/maxor.nix                 the `maxor` CLI, official themes, the daily update check
home/maxor/                    the CLI as scripts: lib/ (UI, loaders, messages) and cmd/ (commands)
tui/                           the full-screen app (Go and Bubble Tea)
themes/                        official themes (colors.json + theme.toml)
tests/                         bats tests for the CLI
docs/                          documentation
```

## Documentation

- [Install and adapt to another machine](docs/INSTALL.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Maxor Shell and the login](docs/SHELL.md)
- [The `maxor` tool](docs/CLI.md) and [the full-screen app](docs/TUI.md)
- [Apps and profiles](docs/APPS.md)
- [Hardware detection](docs/HARDWARE.md)
- [Theme engine](docs/THEMING.md)
- [Visual identity](docs/IDENTITY.md)
- [Translating Maxor OS](docs/TRANSLATING.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Roadmap](docs/ROADMAP.md) · [Base plan](docs/PLAN-BASE.md) · [Releasing](docs/RELEASING.md)
- [Third-party licenses](THIRD_PARTY.md)
- [How to contribute](CONTRIBUTING.md) · [Security](SECURITY.md) · [Changelog](CHANGELOG.md)

## License

[MIT](LICENSE). Third-party components and fonts keep their own licenses; see
[THIRD_PARTY.md](THIRD_PARTY.md).
