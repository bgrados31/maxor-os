{ config, lib, pkgs, inputs, self, modulesPath, ... }:

# The Maxor OS installation medium. Like any installer ISO it is NOT a desktop: the boot menu starts a minimal
# system whose only program is the installer, full screen on a dark background (cage, a one-program Wayland
# compositor, and a terminal). Nothing is written to a disk until the user confirms at the end of the wizard.
# Built with `nix build .#iso`. See docs/INSTALLER.md.
let
  version = lib.removeSuffix "\n" (builtins.readFile ../../VERSION);
  maxor = pkgs.callPackage ../../packages/maxor.nix { };
  maxorTui = pkgs.callPackage ../../packages/maxor-tui.nix { };
  maxorInstall = pkgs.callPackage ../../packages/maxor-install.nix { inherit maxor; };

  # To install without a network: local copies of Maxor OS and its inputs, with their metadata (see
  # installer/offline-overrides.nix). The installer reads them from /etc/maxor-install/overrides.
  overrides = pkgs.writeText "maxor-install-overrides" (import ../offline-overrides.nix { inherit lib self inputs; });

  # What cage runs. A system service starts with a minimal PATH, so the programs the installer calls (maxor-install,
  # nmcli, sudo, timedatectl) are put on it here.
  session = pkgs.writeShellScript "maxor-installer-session" ''
    export PATH=/run/wrappers/bin:/run/current-system/sw/bin:$PATH
    exec ${pkgs.kitty}/bin/kitty --start-as=fullscreen --title='Maxor OS installer' \
      ${maxorTui}/bin/maxor-tui --screen install
  '';
in
{
  imports = [
    (modulesPath + "/installer/cd-dvd/installation-cd-base.nix")
    ../../modules/fonts.nix
  ];

  # ── The installer session ───────────────────────────────────────────
  # The base image already has the passwordless `nixos` user, who is also the owner of the installer.
  services.cage = {
    enable = true;
    user = "nixos";
    program = "${session}";
    # -s lets Ctrl+Alt+F2 reach a text console: a way out if something goes wrong.
    extraArguments = [ "-s" ];
  };
  # Opening the installer must not need a login screen or a graphical target of its own.
  services.getty.autologinUser = lib.mkForce null;

  # Terminal look: the installer draws its own colors; this is the canvas under them.
  environment.etc."xdg/kitty/kitty.conf".text = ''
    font_family Red Hat Mono
    font_size 15
    background #120b12
    foreground #fbe9f2
    cursor #ff86b8
    cursor_blink_interval 0
    window_padding_width 12
    hide_window_decorations yes
    confirm_os_window_close 0
    enable_audio_bell no
    remember_window_size no
    shell_integration disabled
  '';

  # ── What the installer needs ────────────────────────────────────────
  networking.hostName = "maxor-live";
  networking.networkmanager.enable = true;
  # The base image uses a bare wpa_supplicant; Maxor uses NetworkManager (Wi-Fi for the installer and the system).
  networking.wireless.enable = lib.mkImageMediaOverride false;
  nix.settings.experimental-features = [ "nix-command" "flakes" ];
  security.sudo.wheelNeedsPassword = false;

  environment.systemPackages = [ maxorInstall maxorTui maxor ];
  environment.etc."maxor-install/overrides".source = overrides;
  # Marks the installation medium: whatever only makes sense here asks for this file.
  environment.etc."maxor-live".text = "${version}\n";

  # ── Image and boot ──────────────────────────────────────────────────
  # A quiet, dark boot: the installer is the first thing on screen.
  boot.kernelParams = [ "quiet" "loglevel=3" "vt.global_cursor_default=0" ];
  boot.consoleLogLevel = 3;
  # The menu waits longer than the installed system's: there must be time to choose.
  boot.loader.timeout = lib.mkForce 10;
  # File name: maxor-os-<version>-<architecture>.iso
  image.baseName = lib.mkForce "maxor-os-${version}-${pkgs.stdenv.hostPlatform.system}";
  isoImage = {
    volumeID = "MAXOR_OS";
    edition = "maxor";
    prependToMenuLabel = "Maxor OS · ";
    squashfsCompression = "zstd -Xcompression-level 15";
    makeEfiBootable = true;
    makeUsbBootable = true;
  };
  # ZFS comes with the base image; it does not need to force-import the root (and it avoids a warning).
  boot.zfs.forceImportRoot = false;

  system.stateVersion = "26.05"; # DO NOT change
}
