# Visual identity

Maxor OS design decisions. They are the reference for themes, shell, installer and website.

## Character

**Minimalist, elegant, clean and with a gamer touch.** Dark, calm surfaces, a single accent per
theme, geometric typography and soft edges. Nothing decorative that does not serve a purpose.

## Name and brand

- Name: **Maxor OS**. In identifiers and commands: `maxor`.
- The logotype is **text only**: `MAXOR OS` in Krona One, uppercase and widely tracked.
  There is no graphic symbol.
- `/etc/os-release`: `NAME="Maxor OS"`. `ID` stays `nixos` because several tools depend on that
  value.

## Default palette: Sakura (night)

| Role | Value |
|---|---|
| Background | `#120b12` |
| Surface | `#1d121d` |
| Raised surface | `#2a1a2a` |
| Text | `#fbe9f2` |
| Secondary text | `#a88a9d` |
| Accent | `#ff86b8` |
| Secondary accent | `#ffc2a6` |
| Text on accent | `#1b0b14` |

Second official theme: **Glaciar** (glacier; `#07111a` background, `#5fd4f4` accent). Themes are
described in [THEMING.md](THEMING.md).

## Typography

| Role | Font | Where |
|---|---|---|
| Interface | **Figtree** | Bar, settings, menus, notifications, GTK |
| Terminal | **Red Hat Mono** | kitty, editor, code |
| Logo and titles | **Krona One** | Wordmark, boot, lock screen |
| Icons in text | Symbols Nerd Font | Fallback for kitty and the bar |

The three main fonts are free (SIL OFL).

## Shape and motion

| Aspect | Decision |
|---|---|
| Corners | Soft, 8 px |
| Transparency | Translucent: panels at ~74 % with moderate blur |
| Bar | Floating island, centered |
| Background | Plain with a soft vignette, generated from the palette |
| Motion | Fluid, `cubic-bezier(.4, 0, .2, 1)` (the `maxor` curve in `home/hyprland/settings.lua`; `maxor-out` decelerates what appears), ~380 ms |
| Icons | Papirus |
| Cursor | Bibata Modern |

## Screens

- **Lock screen:** the spaced mark «M A X O R  O S» (Cinzel) on top, a large light clock with a short accent
  rule and the date below, and a glass card holding the user name and a pill-shaped password field whose outline is
  a gradient of the two accents. Background: a blurred screenshot. The screen dims after 150 s, locks at 300 s.
- **Wallpaper:** every theme shares one signature (a vignette, a glow of the second accent in the top-right corner
  with three thin orbits, and fine grain), softer on light themes. See [THEMING.md](THEMING.md).
- **Boot:** `MAXOR OS` in Krona One over the palette background and a thin progress bar in the
  accent color.
- **Login:** the DMS greeter with the Maxor Shell package (see [SHELL.md](SHELL.md)).

## Principles

1. One accent per screen.
2. The emphasis color is never the background or the text color.
3. The theme decides the colors; the system decides the shape. Changing theme moves nothing.
4. Everything must look good in dark. The light theme is a variant, not the default.

## Dark and light themes

Maxor OS is designed dark first, but it offers five light themes with the same rules: a single
accent, surfaces tinted toward that accent and AA contrast at least. The light ones are not an
inversion of the dark ones: each starts from its own hue (pink, ocean blue, ink blue, smoke green,
sand). Full list in [THEMING.md](THEMING.md).

## The CLI and the `maxor` screen

The CLI is linear: a vertical rail with `┌ │ ◇ └`, the theme's accent on the rail and on the
steps, and green, amber and red adjusted to the theme's mode. It is sober on purpose and does not
imitate a terminal. What needs a keyboard and the full screen (the store, the installer) is
`maxor-tui`, Maxor's own screen, with tinted surfaces (no frames) and the same glyphs and states.
One visual vocabulary for everything. Details in [CLI.md](CLI.md#the-interface) and
[TUI.md](TUI.md).
