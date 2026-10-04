# Maxor Shell

Maxor Shell is [DankMaterialShell](https://github.com/AvengeMedia/DankMaterialShell) (DMS) with
Maxor OS's identity. It provides the bar, the launcher, the notifications, the control center and
the login screen.

## How it is built: a patch layer, not a copy

Maxor Shell **is not a copy of DMS's code**. It is a Nix package
([`packages/maxor-shell.nix`](../packages/maxor-shell.nix)) that takes the upstream package and
changes only what the user sees. This way:

- we keep receiving DMS's improvements and fixes with `nix flake update`;
- there is no fork to maintain and no merge conflicts;
- the change is small and easy to review (one file and a logo generator);
- if a full fork is ever needed, the starting point is already isolated.

Every replaced text uses `substituteInPlace --replace-fail`: if upstream rewrites that line, the
build **fails** instead of silently leaving a half-done brand. That forces the patch to be
reviewed when updating.

## What changes

| Where | Change |
|---|---|
| Launcher button logo, "About", welcome and what's new | The Krona One "M", tinted with the theme's primary color |
| Launcher button in the bar | Switches to brand mode (`launcherLogoMode = "dank"`) the first time, if the widget was not customized; DMS ships it in "apps" mode and without this setting the logo would not show |
| App icon | The "M" in a rounded square with the Sakura colors |
| "About" | `DANK LINUX` becomes `MAXOR OS` |
| Welcome | `Welcome to Maxor OS` |
| MPRIS identity, updates and plugins | `Maxor Shell` |

The logo is **generated from the font itself** (`packages/make-mark.py`, with fontTools): the
outline of Krona One's "M" is extracted and written as an SVG. This way it does not depend on the
font being installed to be drawn, and it follows Maxor's identity rule: text-only logo, no symbol.

## What does not change (on purpose)

- Internal names (`dms`, `DankBar`, `~/.config/DankMaterialShell/` paths, `dms ipc …` commands):
  they are the interface the other components use and changing them would break things.
- DMS's license and copyright notice (MIT): they are kept in the package.
- Translations: the changed strings stop matching their translations and are shown in English
  until they are translated.

## The login screen

The login is the **DMS greeter on greetd** ([`modules/greeter.nix`](../modules/greeter.nix)),
run with the Maxor Shell package inside Hyprland. It replaces SDDM.

On every boot, greetd copies from the configured user
(`services.displayManager.dms-greeter.configHome`) their DMS settings, their own theme, the colors
and the wallpaper. That is why **the login carries the colors of the last theme you applied with
`maxor theme apply`**.

If the login does not appear, see
[TROUBLESHOOTING.md](TROUBLESHOOTING.md#the-login-does-not-appear-or-stays-black).

## What is missing for a complete identity

- Maxor fonts (Krona One for the logo) inside DMS's own interface.
- A settings screen with Maxor's brand and a shortcut to the future Maxor Store.
- Integrating the theme's corner radius with DMS's (today it is set by hand in `SUPER + ,`).
- Translations of the changed strings.
- A shell of its own in Quickshell (path B of the [roadmap](ROADMAP.md)).
