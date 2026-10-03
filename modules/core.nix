{ config, pkgs, lib, ... }:

# Núcleo de Maxor OS: lo que todo equipo necesita, sin escritorio ni hardware
# específico. Región, teclado, usuario y GPU se definen en hosts/<equipo>/.
{
  # ── Nix ─────────────────────────────────────────────────────────────
  nix.settings = {
    experimental-features = [ "nix-command" "flakes" ];
    warn-dirty = false;
    connect-timeout = 5;
    download-buffer-size = 268435456; # 256 MB: descargas de cache más rápidas
  };
  nix.optimise.automatic = true; # deduplica en segundo plano (auto-optimise-store frena cada build)
  nix.gc = {
    automatic = true;
    dates = "weekly";
    options = "--delete-older-than 14d";
  };
  nixpkgs.config.allowUnfree = true;
  programs.nix-ld.enable = true; # ejecutar binarios genéricos de Linux

  # ── Sistema ─────────────────────────────────────────────────────────
  zramSwap.enable = true; # swap comprimido en RAM
  boot.kernel.sysctl."vm.swappiness" = 150; # con zram conviene usarlo antes que el disco
  services.fstrim.enable = true; # TRIM semanal del NVMe
  services.journald.extraConfig = "SystemMaxUse=200M";
  documentation.nixos.enable = false; # no generar el manual de opciones en cada rebuild
  networking.networkmanager.enable = true;
  programs.fish.enable = true;

  hardware.graphics = {
    enable = true;
    enable32Bit = true; # Steam y Wine
  };

  # ── Audio y Bluetooth ───────────────────────────────────────────────
  services.pulseaudio.enable = false;
  security.rtkit.enable = true;
  services.pipewire = {
    enable = true;
    alsa.enable = true;
    alsa.support32Bit = true;
    pulse.enable = true;
  };
  hardware.bluetooth = {
    enable = true;
    powerOnBoot = true;
  };

  services.printing.enable = true;

  environment.systemPackages = with pkgs; [
    git
    wget
    curl
    neovim
    unzip
    pciutils
    usbutils
  ];
}
