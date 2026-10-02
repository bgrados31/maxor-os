{ config, pkgs, ... }:

{
  imports = [ ./hardware-configuration.nix ../../modules/branding.nix ../../modules/fonts.nix ];

  # ── Boot ────────────────────────────────────────────────────────────
  boot.loader.systemd-boot.enable = true;
  # La ESP (100 MB, compartida con Windows) va en /efi; los kernels e initrd
  # van en la partición XBOOTLDR de 1 GB montada en /boot.
  boot.loader.efi.efiSysMountPoint = "/efi";
  boot.loader.systemd-boot.xbootldrMountPoint = "/boot";
  boot.loader.systemd-boot.configurationLimit = 10;
  boot.initrd.compressorArgs = [ "-19" "-T0" ];
  boot.loader.efi.canTouchEfiVariables = true;
  zramSwap.enable = true; # no hay partición swap

  # ── Nix ─────────────────────────────────────────────────────────────
  nix.settings = {
    experimental-features = [ "nix-command" "flakes" ];
    auto-optimise-store = true;
  };
  nix.gc = {
    automatic = true;
    dates = "weekly";
    options = "--delete-older-than 14d";
  };
  nixpkgs.config.allowUnfree = true;
  programs.nix-ld.enable = true;

  # ── Red / región ────────────────────────────────────────────────────
  networking.hostName = "nitro";
  networking.networkmanager.enable = true;
  time.timeZone = "America/Lima";
  i18n.defaultLocale = "en_US.UTF-8";
  console.keyMap = "la-latin1";

  # ── Gráficos: Intel + NVIDIA RTX 4050 (PRIME offload) ───────────────
  hardware.graphics = {
    enable = true;
    enable32Bit = true;
  };
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

  # ── Escritorio ──────────────────────────────────────────────────────
  programs.hyprland.enable = true;

  # Sin Budgie/GNOME: Hyprland es la única sesión. El respaldo ahora son
  # las generaciones anteriores del menú de arranque. SDDM en Wayland
  # hace de login (luego se reemplaza por el greeter de Maxor).
  services.displayManager.sddm = {
    enable = true;
    wayland.enable = true;
  };
  services.displayManager.defaultSession = "hyprland";

  # hyprlock necesita su servicio PAM; sin esto no podrías desbloquear.
  security.pam.services.hyprlock = { };

  services.xserver.xkb = {
    layout = "latam";
    variant = "";
  };

  environment.sessionVariables = {
    NIXOS_OZONE_WL = "1"; # apps Electron/Chromium nativas en Wayland
  };

  # ── Servicios ───────────────────────────────────────────────────────
  services.printing.enable = true;
  services.pulseaudio.enable = false;
  security.rtkit.enable = true;
  services.pipewire = {
    enable = true;
    alsa.enable = true;
    alsa.support32Bit = true;
    pulse.enable = true;
  };
  hardware.bluetooth.enable = true;
  hardware.bluetooth.powerOnBoot = true;
  services.upower.enable = true;
  services.gnome.gnome-keyring.enable = true;
  services.gvfs.enable = true;

  # ── Usuario ─────────────────────────────────────────────────────────
  programs.fish.enable = true;
  users.users.bryan = {
    isNormalUser = true;
    description = "Bryan";
    extraGroups = [ "networkmanager" "wheel" "video" "audio" ];
    shell = pkgs.fish;
  };

  programs.firefox.enable = true;

  environment.systemPackages = with pkgs; [
    git
    wget
    curl
    neovim
    unzip
    pciutils
    usbutils
  ];


  system.stateVersion = "26.05"; # NO cambiar
}
