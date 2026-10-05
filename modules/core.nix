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
  # Con poca RAM libre mata el proceso más pesado antes de que el sistema se congele.
  services.earlyoom = {
    enable = true;
    freeMemThreshold = 5;
    enableNotifications = true;
  };
  networking.networkmanager.enable = true;
  programs.fish.enable = true;

  # ── Apps ajenas a Nix: Flatpak (las usará Maxor Store) y AppImage ───
  services.flatpak.enable = true;
  systemd.services.flatpak-flathub = {
    description = "Añadir el repositorio Flathub a Flatpak";
    # Cuelga de network-online, no de multi-user: con red lenta retenía graphical.target ~6 s sin que nada lo necesite.
    wantedBy = [ "network-online.target" ];
    wants = [ "network-online.target" ];
    after = [ "network-online.target" ];
    path = [ pkgs.flatpak ];
    script = "flatpak remote-add --system --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo";
    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
      Restart = "on-failure"; # sin red al arrancar: reintenta
      RestartSec = 60;
    };
  };
  programs.appimage = {
    enable = true;
    binfmt = true; # los .AppImage se ejecutan directamente
  };

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
