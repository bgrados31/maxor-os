# Installation

Maxor OS does not have its own installer yet (it is in [phase 6](ROADMAP.md)). Today it is
installed by applying the flake on top of an existing NixOS installation.

## Requirements

- NixOS 25.11 or later, UEFI boot.
- Flakes enabled (`nix.settings.experimental-features = [ "nix-command" "flakes" ]`).
- An Internet connection: the first build downloads several hundred MB (fonts, NVIDIA,
  Hyprland, DMS).
- A user with `sudo` permissions.

## Installing on the reference machine (`nitro`)

The `nitro` host is meant for an Acer Nitro AN16 with Intel + NVIDIA RTX 4050, dual boot with
Windows and the XBOOTLDR partition described below. If your machine is that one, or very similar:

```sh
git clone https://github.com/bgrados31/maxor-os ~/nixos-config
cd ~/nixos-config
sudo nixos-rebuild boot --flake .#nitro     # first time: leave it for the next boot
reboot
```

`boot` instead of `switch` applies the new generation only on the next boot and leaves the current
one untouched as a fallback.

## Adapting it to another machine

Copy `hosts/nitro` to `hosts/<your-host>` and adjust the following.

### 1. Hardware

Regenerate `hardware-configuration.nix` on your machine; do not use the one in the repository:

```sh
sudo nixos-generate-config --show-hardware-config > hosts/<your-host>/hardware-configuration.nix
```

Then let Maxor detect the CPU, GPUs and whether it is a laptop:

```sh
maxor hardware detect --write
```

See [HARDWARE.md](HARDWARE.md).

### 2. Boot

`hosts/nitro/configuration.nix` uses systemd-boot with the ESP at `/efi` and an XBOOTLDR partition
at `/boot`. If your disk has the ESP mounted at `/boot` (the usual case), remove these lines and
leave the default value:

```nix
boot.loader.efi.efiSysMountPoint = "/efi";
boot.loader.systemd-boot.xbootldrMountPoint = "/boot";
```

### 3. GPU

- **Intel or AMD only:** drivers are picked from the detected hardware; remove any leftover
  `hardware.nvidia` block.
- **Intel + NVIDIA:** replace the bus identifiers with yours:

  ```sh
  lspci | grep -E 'VGA|3D'
  ```

  Turn `00:02.0` into `PCI:0:2:0` and `01:00.0` into `PCI:1:0:0`.

### 4. User, time zone and keyboard

Edit in `hosts/<your-host>/configuration.nix`: `networking.hostName`, `time.timeZone`,
`console.keyMap`, `services.xserver.xkb` and `users.users.<name>`. In `home/bryan.nix` change
`home.username`, `home.homeDirectory` and the `programs.git` data.

### 5. Register the host in the flake

In `flake.nix`, duplicate the `nixosConfigurations.nitro` block, rename it and point it at
`./hosts/<your-host>/configuration.nix`. Then:

```sh
sudo nixos-rebuild switch --flake .#<your-host>
```

## After installing

1. Reboot. The Maxor login (the DMS greeter) goes straight into Hyprland.
2. Open the DMS settings with `SUPER + ,` and set up the bar and the wallpaper.
3. Apply a theme: `maxor theme apply sakura`.
4. Run `maxor setup` once to pick a look, usage profiles and see what was detected.
5. Optional: open `qt6ct` once and pick the DMS color scheme for Qt apps.

## Updating

```sh
maxor update
```

It updates the inputs, builds without applying, shows what changes and asks before switching.
The manual way is:

```sh
nix flake update --flake ~/nixos-config
sudo nixos-rebuild switch --flake ~/nixos-config#nitro
```

Inside fish there are the `rebuild` and `update` aliases for both.

## Uninstalling or going back

- **An update broke something:** pick the previous generation in the boot menu.
- **From a TTY:** `sudo nixos-rebuild switch --rollback`, or `maxor rollback`.
- **From the full-screen app:** the Update tab, then `g`.
- **Freeing space:** the weekly garbage collector deletes generations older than 14 days.
