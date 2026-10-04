# Changelog

All notable changes to Maxor OS. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versions are numbered `0.N.0` until 1.0 (see [docs/RELEASING.md](docs/RELEASING.md)).

## [Unreleased]

### Added

- **Installer wizard** (`maxor-tui --screen install`): thirteen steps (welcome checks, language, keyboard with live test box, network incl. Wi-Fi, time zone, disk, install method, storage and encryption, account, look and profiles, hardware, review, install, done). Erasing needs a typed `ERASE`, installing alongside a typed `INSTALL`; passwords are hashed by the engine, and a failed install can be resumed with `r`.
- **Full-install test** (`installer-full`): two VMs share a disk; the engine installs a complete Maxor OS on the empty virtual disk, offline, and the installed system is then booted and checked (account, password outside the Nix store, the user's configuration repository, the CLI, boot entries).
- **Offline installs and extension points** for the installer: `host/local.nix` is the user's own NixOS configuration, never overwritten; preseeded hardware (`MAXOR_INSTALL_HWJSON`, `MAXOR_INSTALL_HWCONFIG`), local copies of the flake inputs that keep their revision metadata (`installer/offline-overrides.nix`), and `maxor-install hashpw` so the password is hashed by the engine over standard input.
- **Installation ISO** (`nix build .#iso`): boots straight into the installer, full screen on a dark background, with no desktop (a one-program Wayland compositor and a terminal). Quiet boot, NetworkManager, offline-install overrides; Ctrl+Alt+F2 opens a text console as a way out.
- **Installer engine** (`maxor-install`): the program that installs Maxor OS, independent of any screen. It reads a versioned JSON contract of answers and runs eight stages (preflight, GPT partitioning, LUKS2, filesystem with btrfs subvolumes, per-machine flake, `nixos-install`, boot check, finish) with progress events, resume support and a deterministic dry run. Safety lives in the engine: erasing a disk requires typing `ERASE`, installing alongside another system requires the hash of the plan that was shown, and passwords only ever travel on standard input. See [docs/INSTALLER.md](docs/INSTALLER.md).
- **First-login look**: new users now start with the Maxor logo in the bar and the sakura theme (`maxor firstrun`, run once by a user service), without overriding anything already customized.
- **Real-disk installer test** (`installer-disks`): boots a UEFI VM and runs the real partitioning, LUKS2 and filesystem stages on virtual disks, including installing alongside an existing system, which must leave the existing partitions byte-identical.
- **Test virtual machine** (`nix run .#vm`, `nix run .#vm-old`): the real desktop, login and CLI on QEMU, to try changes without touching a real machine. `vm-old` pretends to be 0.0.1 so the published release shows up as an update. A two-VM check (`release-vm`) exercises the release system end to end. See [docs/VM.md](docs/VM.md).
- **Repository hardening**: secret scanning and push protection, Dependabot, branch and tag protection (published `v*` tags are immutable), GitHub Actions pinned to commit hashes, `CODEOWNERS`, and the signing model documented in [SECURITY.md](SECURITY.md).

### Changed

- **No hardcoded user.** Hostname, user, locale, timezone, keyboard and git identity are now options (`maxor.machine`), and `maxor-os.lib.mkSystem` is the single entry point for any machine. The existing `nitro` system is unchanged: of 654 home files and all of `/etc`, only the intended file differs.
- **Doctor** no longer requires `/boot` to be a mount point on machines without UEFI (for example, a virtual machine).
- **Repository documentation and release tooling are in English.**

### Fixed

- **Trust keys and data files could go missing.** The release trust keys, the profile catalog and the installer schema were referenced by source path instead of being copied into the Nix store, so they could disappear after garbage collection or on another machine. They are now real dependencies of the packages.
- The installer's disk probe no longer fails on devices without media (floppy drives, empty card readers).
- A missing signature file on the release channel was reported as an invalid signature instead of an unsigned release.
- The release workflow now fetches the signed tag object before verifying it, and can be re-run by hand for an existing tag.

## [0.1.0] - 2026-10-04

First release.

### Added

- **Theme engine**: `maxor theme list | current | apply | undo | install | export`. Ten official themes (five dark, five light, all AA contrast), each with its own shape: corners, gaps, blur and animation speed. Themes are validated data folders and never run code.
- **Maxor Shell and login**: DankMaterialShell with Maxor's identity as a patch layer (logo, name), and a greetd login that inherits the user's theme, colors and wallpaper.
- **Full-screen app** (`maxor-tui`, Go): Home, Store, Themes, Update, Doctor, Profiles and Exit tabs, with a command palette, vim keys, mouse support and a layout that adapts to the window. System changes are applied inside the screen, with a masked password field handed only to `sudo`.
- **Signed releases and update notices** (`maxor release check | status | apply`): every release carries a signed manifest and a signed git tag. Machines poll hourly with conditional requests, verify the signature and refuse to go back to an older release, and notify once per version. See [docs/UPDATES.md](docs/UPDATES.md).
- **Apps without a rebuild**: search, install, update and remove from nixpkgs and Flathub with no `sudo`, with leftover-data detection, launcher integration and an update check. Flatpak and AppImage work out of the box.
- **Update and recovery**: `maxor update` shows what changes before applying, `maxor rollback` returns to any previous generation, `maxor doctor` offers fixes, and `maxor backup` / `restore` carry profiles, themes and settings between machines.
- **Hardware and profiles**: `maxor hardware` detects CPU, GPU, laptop and virtual machines and configures drivers (including hybrid NVIDIA); profiles (gaming, dev, creator, office) are switched with `maxor profile`.
- **Identity**: `os-release`, a Plymouth boot theme, the Figtree, Red Hat Mono and Krona One fonts, and systemd-boot with an XBOOTLDR partition that shares the ESP with Windows.
- **Reliability and tooling**: `earlyoom`, `fwupd`, `maxor logs | debug | version | completions`, global flags, typed exit codes, a command registry, and an English message catalog.
- **Tests and CI**: 83 shell tests and about 50 Go tests, run by `nix flake check` and by CI on every push.
- **Documentation**: architecture, installation, theming, identity, apps, hardware, updates, troubleshooting, roadmap and third-party licenses.

### Changed

- **CLI**: linear output on a rail (`┌ │ ◇ └`) with a single loader, English-only messages, and much faster startup (help 352 → 108 ms, `theme list` 1562 → 271 ms, `doctor` 3 s → 0.5 s).
- **Hyprland** is the only session. Its configuration is split into Lua modules plus a personal `user.lua` that the system never overwrites.
- **Wallpapers** are about 2 MB instead of 20 MB.

## Earlier history

Initial base: NixOS 26.05 with Hyprland 0.55 (Lua configuration), DankMaterialShell, kitty, fish, starship and NVIDIA PRIME in offload mode.
