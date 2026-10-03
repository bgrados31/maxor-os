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

  # ── Identidad del equipo y región ───────────────────────────────────
  networking.hostName = "nitro";
  time.timeZone = "America/Lima";
  i18n.defaultLocale = "en_US.UTF-8";
  console.keyMap = "la-latin1";
  services.xserver.xkb = {
    layout = "latam";
    variant = "";
  };

  # ── Gráficos: Intel + NVIDIA RTX 4050 (PRIME offload) ───────────────
  services.xserver.videoDrivers = [ "nvidia" ];
  hardware.nvidia = {
    modesetting.enable = true;
    open = true;
    nvidiaSettings = true;
    powerManagement.enable = true;
    package = config.boot.kernelPackages.nvidiaPackages.stable;
    prime = {
      offload.enable = true;
      offload.enableOffloadCmd = true; # usa `nvidia-offload <app>` para la GPU dedicada
      intelBusId = "PCI:0:2:0";
      nvidiaBusId = "PCI:1:0:0";
    };
  };

  # ── Login: el greeter hereda el tema y el wallpaper de este usuario ──
  services.displayManager.dms-greeter.configHome = "/home/bryan";

  # ── Usuario ─────────────────────────────────────────────────────────
  users.users.bryan = {
    isNormalUser = true;
    description = "Bryan";
    extraGroups = [ "networkmanager" "wheel" "video" "audio" ];
    shell = pkgs.fish;
  };

  system.stateVersion = "26.05"; # NO cambiar
}
