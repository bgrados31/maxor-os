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
  # ...and where each of those copies lives on the internet, so the machine's flake.lock ends up pointing there.
  locks = pkgs.writeText "maxor-install-locks.json" (import ../offline-locks.nix { inherit lib self; });

  # The official themes, where the CLI looks for them ($HOME/.local/share/maxor/themes). An installed system gets them
  # from home-manager; the installation medium has none, so they are linked here. Maxor Dark is the default.
  themes = pkgs.runCommand "maxor-live-themes" { } ''
    mkdir -p $out
    for d in ${../../themes}/*/; do
      id=$(basename $d)
      mkdir -p $out/$id
      cp $d/colors.json $d/theme.toml $out/$id/
      [ -f $d/style.json ] && cp $d/style.json $out/$id/ || true
    done
  '';

  # sway is the whole session: one program, full screen, on a dark background, with no bar. It is used (instead of
  # a plainer kiosk compositor) because it can change the keyboard layout while it runs, which is what lets the
  # installer test the layout that is being chosen. When the installer ends, sway ends and greetd starts it again.
  swayConfig = pkgs.writeText "maxor-installer-sway.conf" ''
    output * bg #050c38 solid_color
    default_border none
    default_floating_border none
    focus_follows_mouse no
    seat * hide_cursor 2000
    # in a shell: sway splits its own commands at «;», so `exec a; b` would run only a and try b as a sway command
    exec ${pkgs.bash}/bin/sh -c '${pkgs.kitty}/bin/kitty --start-as=fullscreen --title="Maxor OS installer" ${maxorTui}/bin/maxor-tui --screen install; ${pkgs.sway}/bin/swaymsg exit'
  '';

  # A system service starts with a minimal PATH, so the programs the installer calls (maxor-install, nmcli, sudo,
  # timedatectl, swaymsg) are put on it here.
  session = pkgs.writeShellScript "maxor-installer-session" ''
    export PATH=/run/wrappers/bin:/run/current-system/sw/bin:$PATH
    # what sway and the programs in it print goes to the journal (journalctl -t maxor-installer), not to the console,
    # where it would show as text when the medium shuts down
    exec ${pkgs.systemd}/bin/systemd-cat -t maxor-installer ${pkgs.sway}/bin/sway --config ${swayConfig}
  '';
in
{
  imports = [
    (modulesPath + "/installer/cd-dvd/installation-cd-base.nix")
    ../../modules/fonts.nix
    ../../modules/branding.nix # the name, the quiet boot and the splash with the progress bar
    ../offline.nix # the tools to build a system without a network
  ];

  # ── Installing without a network ────────────────────────────────────
  # The finished packages of a generic Maxor OS travel on the medium. nixos-install takes them from this store; what
  # describes the machine being installed (disks, account, hardware) is built during the installation, with the tools
  # of offline.nix. Proven by checks.installer-full, which installs a different machine offline.
  isoImage.storeContents = self.lib.offlineStore pkgs;

  # ── The installer session ───────────────────────────────────────────
  # The base image already has the passwordless `nixos` user, who is also the owner of the installer.
  # greetd starts the session at boot, signed in, and brings it back if it ends. Ctrl+Alt+F2 reaches a text console:
  # a way out if something goes wrong.
  services.greetd = {
    enable = true;
    settings.initial_session = { command = "${session}"; user = "nixos"; };
    settings.default_session = { command = "${session}"; user = "nixos"; };
  };
  security.polkit.enable = true; # the session gets its seat (screen, keyboard) from logind
  environment.sessionVariables.WLR_NO_HARDWARE_CURSORS = "1"; # virtual machines draw no cursor plane
  # Opening the installer must not need a login screen or a graphical target of its own.
  services.getty.autologinUser = lib.mkForce null;

  # Terminal look: the installer draws its own colors; this is the canvas under them.
  environment.etc."xdg/kitty/kitty.conf".text = ''
    font_family Red Hat Mono
    font_size 15
    background #050c38
    foreground #eef0ff
    cursor #c084ff
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

  environment.systemPackages = [ maxorInstall maxorTui maxor pkgs.sway ]; # sway: swaymsg changes the layout under test
  environment.etc."maxor-install/overrides".source = overrides;
  environment.etc."maxor-install/locks.json".source = locks;
  # Marks the installation medium: whatever only makes sense here asks for this file.
  environment.etc."maxor-live".text = "${version}\n";
  systemd.tmpfiles.rules = [
    "d /home/nixos/.local 0755 nixos users -"
    "d /home/nixos/.local/share 0755 nixos users -"
    "d /home/nixos/.local/share/maxor 0755 nixos users -"
    "L+ /home/nixos/.local/share/maxor/themes - - - - ${themes}"
  ];

  # ── Image and boot ──────────────────────────────────────────────────
  # The menu waits longer than the installed system's: there must be time to choose.
  boot.loader.timeout = lib.mkForce 10;
  # File name: maxor-os-<version>-<architecture>.iso
  image.baseName = lib.mkForce "maxor-os-${version}-${pkgs.stdenv.hostPlatform.system}";
  system.nixos.label = version; # what the boot menu entry shows after the name
  isoImage = {
    volumeID = "MAXOR_OS";
    # the boot menu entry reads «Install Maxor OS 0.2.0» instead of the NixOS label (version, date and commit)
    prependToMenuLabel = "Install ";
    appendToMenuLabel = "";
    edition = "maxor";
    squashfsCompression = "zstd -Xcompression-level 15";
    makeEfiBootable = true;
    makeUsbBootable = true;
    # the boot menu: the brand gradient, with the mark in Cinzel and the entries in Red Hat Mono
    grubTheme = pkgs.callPackage ./grub-theme.nix { };
  };
  # ZFS comes with the base image; it does not need to force-import the root (and it avoids a warning).
  boot.zfs.forceImportRoot = false;

  system.stateVersion = "26.05"; # DO NOT change
}
