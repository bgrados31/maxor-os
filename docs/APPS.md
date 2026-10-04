# Apps and profiles

Installing software on Maxor OS does not require editing `.nix` or rebuilding the system. There are
two ways, both from `maxor`, and a third (profiles) to set the machine up by kind of use.

## User apps (no sudo, no rebuild)

```
maxor search <text>               search nixpkgs and Flathub
maxor install <app…>              install
maxor remove <app…> [--purge]     remove (with --purge, also the folders it left in your home)
maxor apps                        list what was installed with maxor
maxor apps updates                apps that have a newer version
maxor apps update [app…]          update everything, or just the ones you name
maxor apps open <app>             open an installed app
maxor apps repair                 show Flatpak apps in the app menu and the terminal
```

| Source | How it is installed | Where |
|---|---|---|
| `nix` | `nix profile add nixpkgs#<package>` | the user profile |
| `flatpak` | `flatpak install --user flathub <id>` | `~/.local/share/flatpak` |

Unless told otherwise, an ID with three or more dot-separated parts (`org.mozilla.firefox`) is
Flatpak and any other name (`btop`) is nixpkgs. Force it with `--nix` or `--flatpak`.
Packages with a non-free license (Steam, Spotify…) can be installed.

Only one package operation runs at a time (`apps.lock`): if another `maxor install`, `remove` or
`apps update` is running, the next one waits its turn.

## Search and the Store

In a terminal, `maxor search` opens the **Store** tab of the full-screen app
([TUI.md](TUI.md)) with the search already launched. With `--list`, `--json` or when the output
is not a terminal, it prints the results as a list instead.

Results are sorted by relevance: exact, starts with, contains and, at the end, the ones that only
match the description (shown only when there are very few better ones). From nixpkgs only
top-level packages come out; `tests.*`, `haskellPackages.*` and similar are internal pieces.

What is installed this way is **the user's**: it does not enter the flake or the system
generations, so `maxor rollback` does not undo it. For something you want in the system and in the
repository, use a profile or edit your `configuration.nix`.

### Removing, and what an app leaves behind

`maxor remove` uninstalls the app and then lists the folders it left in your home. A folder
counts only if it is named exactly like the app (case-insensitively) inside `~/.config`,
`~/.local/share`, `~/.cache`, `~/.local/state`, `~/.var/app`, `~/.<name>` or `~/logs/<name>.log`.
Without `--purge` nothing is deleted; with it they are, and it also cleans up after an app that is
already gone. `maxor remove <app> --list-data` only looks.

### Flatpak apps in the launcher and the terminal

Flatpak keeps each app's menu entry, icons and command in `~/.local/share/flatpak/exports`, which
a session only sees if it started with that folder in its environment. To make them show up
always, `maxor install` links each Flatpak app into the user folders: the menu entry into
`~/.local/share/applications`, the icons into `~/.local/share/icons` and a short command named
after the app into `~/.local/bin` (it never overrides a command that already exists).
`remove` undoes it and `maxor apps repair` fixes the ones that were installed before.

After installing or removing, `maxor` tells the launcher (Super+Space) to re-read its entries.

## Profiles

A profile groups system packages and services by kind of use.

```
maxor profile                     list and see which are active
maxor profile enable <name>       enable (shows the changes and asks to confirm)
maxor profile disable <name>      disable
```

| Profile | Includes |
|---|---|
| `gaming` | Steam, Proton, GameMode, MangoHud, Gamescope, Lutris |
| `dev` | C, Node, Python, Go, Rust, Git, GitHub CLI, Podman (with `docker`), direnv |
| `creator` | OBS, GIMP, Inkscape, Krita, Kdenlive, Blender, Audacity, HandBrake |
| `office` | LibreOffice, Spanish and English dictionaries, Thunderbird, Okular |

What is active is saved in `hosts/<host>/maxor.json` and `modules/profiles.nix` turns it into
packages. `enable` and `disable` are equivalent to editing that file and running
`maxor update --no-lock`; if you cancel the confirmation, the file has already changed and it will
be applied on the next update. Names, descriptions and the list of what each one includes live in
`modules/profiles-catalog.json`; adding a profile is an entry there plus its block in
`modules/profiles.nix`.

## JSON contract (for the Store and other apps)

Commands that query accept `--json` and then write **only** JSON to standard output, with no
colors or messages.

| Command | Output |
|---|---|
| `maxor search <text> --json` | `[{source, id, name, version, description}]` in a single list by relevance, nixpkgs and Flathub mixed (max. 14) |
| `maxor apps --json` | `[{source, id, name, version}]` |
| `maxor install <app…> --json` | `[{id, source, ok}]` |
| `maxor apps updates` | `[{source, id, current, latest}]`: only apps with a newer version (nix against the system's nixpkgs; flatpak according to Flathub). Kept for 10 minutes in `apps-updates.json`; `--refresh` ignores it; `--notify` also shows a desktop notification |
| `maxor apps repair --json` | `{repaired: n}`: links the installed Flatpak apps into the user folders (menu, icons and command) |
| `maxor apps update <app…> --json` | `[{id, source, ok}]` |
| `maxor apps open <app> --json` | `[{id, ok}]`: opens it detached from the terminal |
| `maxor remove <app> --list-data` | `[{path, bytes}]`: only looks, removes nothing |
| `maxor remove <app…> [--purge] --json` | `[{id, source, ok, purged, leftovers: [{path, bytes}]}]`. Leftover folders are the ones named exactly like the app (`~/.config`, `~/.local/share`, `~/.cache`, `~/.var/app`, `~/.name`); without `--purge` nothing is deleted |
| `maxor profile list --json` | `[{id, title, description, includes, enabled}]` |
| `maxor hardware detect` | the contents of `hardware.json` ([HARDWARE.md](HARDWARE.md)) |
| `maxor backup --json` | `{path, bytes, apps, themes, host}` ([CLI.md](CLI.md#maxor-backup-and-maxor-restore)) |

`source` is `nix` or `flatpak`. The exit code is 0 if everything went well and 1 if something
failed. An app should not reimplement any of this: it calls `maxor` and reads the JSON.
