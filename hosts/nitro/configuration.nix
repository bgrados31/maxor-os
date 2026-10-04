{ config, pkgs, ... }:

# Acer Nitro AN16 (Intel i5-13500H + NVIDIA RTX 4050, arranque dual con Windows).
# Todo lo común a Maxor OS está en modules/; aquí solo queda lo propio del equipo.
{
  imports = [ ./hardware-configuration.nix ];

  # ── Arranque ────────────────────────────────────────────────────────
  boot.loader.systemd-boot.enable = true;
  # La ESP (100 MB, compartida con Windows) va en /efi; los kernels e initrd
  # van en la partición XBOOTLDR de 1 GB montada en /boot.
  boot.loader.efi.efiSysMountPoint = "/efi";
  boot.loader.systemd-boot.xbootldrMountPoint = "/boot";
  boot.loader.systemd-boot.configurationLimit = 10;
  boot.loader.efi.canTouchEfiVariables = true;
  boot.initrd.compressorArgs = [ "-19" "-T0" ];

  # ── Quién y dónde es esta máquina (ver modules/machine.nix) ─────────
  maxor.machine = {
    hostname = "nitro";
    user = "bryan";
    fullname = "Bryan";
    timezone = "America/Lima";
    locale = "en_US.UTF-8";
    keymap = "la-latin1";
    xkb = {
      layout = "latam";
      variant = "";
    };
    git = {
      name = "Bryan Grados";
      email = "218035463+bgrados31@users.noreply.github.com";
    };
  };

  # ── Hardware: drivers según lo detectado (Intel + NVIDIA RTX 4050) ──
  # Se regenera con `maxor hardware detect --write`; ver modules/hardware.nix.
  maxor.hardware.report = ./hardware.json;
  maxor.settings = ./maxor.json; # perfiles activos (maxor profile)

  system.stateVersion = "26.05"; # NO cambiar
}
