# Troubleshooting

Start with `maxor doctor`: it checks services, session, graphics, boot and configuration, and
tells you what to look at. `maxor logs --last` shows the details of the last failure.

## I cannot unlock the screen

hyprlock authenticates with PAM. The configuration declares the service in
`security.pam.services.hyprlock`; if you removed it, you will not be able to unlock.

1. Switch to a TTY with `Ctrl+Alt+F3` and log in.
2. Run `pkill hyprlock`.
3. Go back to the session with `Ctrl+Alt+F1` or `F2`.

To avoid it in the future, do not remove that line from `hosts/<host>/configuration.nix`.

## `maxor theme apply` changes nothing

The command updates two keys of `~/.config/DankMaterialShell/settings.json`. Check that they were
written:

```sh
jq '{currentThemeName, customThemeFile}' ~/.config/DankMaterialShell/settings.json
```

It should show `"custom"` and a path inside `~/.config/maxor/current/`. If it does and the bar
does not change, restart DMS with `systemctl --user restart dms`. If `jq` errors, the file does not
exist yet: open the DMS settings first (`SUPER + ,`).

## A theme gives an "invalid colors" or "incomplete" error

`colors.json` needs all eight keys (`bg`, `s`, `s2`, `fg`, `mu`, `ac`, `ac2`, `on`) and every
value must be `#rrggbb` (six hexadecimal digits). See [THEMING.md](THEMING.md).

## The system does not boot after a `switch`

Pick a previous generation in the boot menu; NixOS keeps the last ten
(`configurationLimit = 10`). Once inside:

```sh
sudo nixos-rebuild switch --rollback
```

## The login does not appear or stays black

The login is greetd with the DMS greeter ([SHELL.md](SHELL.md#the-login-screen)).

1. **Get in through a TTY:** `Ctrl+Alt+F3` and log in.
2. **Look at the status and the log:**

   ```sh
   systemctl status greetd
   journalctl -u greetd -b --no-pager | tail -n 40
   ```

3. **Save the greeter log** if you need more detail: add
   `services.displayManager.dms-greeter.logs.save = true;` to the configuration, apply it and
   reboot; the log ends up in `/tmp/dms-greeter.log`.
4. **Go back to what worked:** reboot and pick a previous generation in the boot menu, or
   `sudo nixos-rebuild switch --rollback` from the TTY.

From the TTY you can also start Hyprland by hand with `start-hyprland` while you fix it.

## The login does not have my theme's colors

Greetd copies the user's theme, colors and wallpaper **at boot**. After `maxor theme apply` the
login changes on the next boot, not instantly. Restart `greetd` only if you are on a TTY (it would
close your graphical session).

## The boot menu does not show Windows

Windows appears because systemd-boot detects its boot manager on the ESP. Check that the ESP is
still mounted at `/efi`:

```sh
findmnt /efi
sudo bootctl list
```

## NVIDIA: an app does not use the dedicated GPU

PRIME is configured in *offload* mode: the integrated graphics draws the desktop and the NVIDIA is
only used when asked.

```sh
nvidia-offload <program>        # in Steam: nvidia-offload %command%
```

To check which GPU a program uses: `nvidia-offload glxinfo | grep "OpenGL renderer"`.

## A Flathub app is not in the launcher or the terminal

Flatpak puts each app's menu entry, icons and command in `~/.local/share/flatpak/exports`, and a
session only sees them if it started with that folder in its environment. `maxor install` links
every Flatpak app into your user folders so it always shows up. If an app installed some other
way is missing:

```sh
maxor apps repair
```

`maxor doctor` warns when an installed Flatpak app would not be visible.

## An app I removed is still in the launcher

`maxor remove` tells the launcher to re-read its entries. If you removed it some other way, run
`maxor apps repair`, which also clears dangling links.

## The boot splash does not appear

Plymouth only shows if the kernel boots with `quiet splash` (already included) and the theme's
font is in the initrd. After changing the theme or the font you need a `switch` and a reboot;
Plymouth cannot be previewed in a running session.

## The full-screen app looks cramped or misses the details panel

With fewer than 96 columns the details do not fit beside the list and are drawn in a compact
drawer below; with `D` you can hide or show them. The minimum size is 64×20. If the terminal is
smaller, the app asks you to enlarge it.

## Errors in the Hyprland configuration

Hyprland 0.55 uses Lua configuration. Errors show on screen at startup or on reload
(`hyprctl reload`). To see the detail:

```sh
journalctl --user -b | grep -i hyprland
```

## Qt apps do not use the theme's colors

Open `qt6ct` once and pick the DMS color scheme.

## Reporting a bug

Open an [issue](https://github.com/bgrados31/maxor-os/issues/new/choose) with the output of:

```sh
nixos-version
maxor version
hyprctl version | head -n 3
journalctl --user -b -p err --no-pager | tail -n 30
```

`maxor debug` writes a report with the version, hardware, doctor and log that you can attach
(review it before sharing).
