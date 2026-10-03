{ config, pkgs, lib, ... }:

# Escritorio de Maxor OS: Hyprland como única sesión, login y servicios que
# el escritorio necesita. La configuración de Hyprland en sí vive en home/.
{
  programs.hyprland.enable = true;

  # El login lo gestiona modules/greeter.nix (greetd + greeter de DMS).
  services.displayManager.defaultSession = "hyprland";

  # Hyprland trae además "hyprland-uwsm.desktop", que necesita uwsm (no está
  # habilitado). El greeter de DMS lo listaba primero y, al elegirlo, la sesión
  # moría al instante (wayland-session-bindpid@ no existe) y volvía al login.
  # Solo exponemos la sesión normal.
  #
  # greetd arranca la sesión en la misma TTY, así que lo que Hyprland imprime
  # al iniciar (logs, mensajes de nix) se veía en pantalla tras el login. Se
  # redirige al journal: `journalctl -t maxor-hyprland -b` para consultarlo.
  services.displayManager.sessionPackages = lib.mkForce [
    (pkgs.runCommand "maxor-hyprland-session"
      { passthru.providedSessions = [ "hyprland" ]; }
      ''
        mkdir -p $out/share/wayland-sessions
        sed 's|^Exec=.*|Exec=${pkgs.writeShellScript "maxor-hyprland-start" ''
          exec ${config.programs.hyprland.package}/bin/start-hyprland > >(${pkgs.systemd}/bin/systemd-cat -t maxor-hyprland) 2>&1
        ''}|' \
          ${config.programs.hyprland.package}/share/wayland-sessions/hyprland.desktop \
          > $out/share/wayland-sessions/hyprland.desktop
      '')
  ];

  # hyprlock necesita su servicio PAM; sin esto no podrías desbloquear.
  security.pam.services.hyprlock = { };

  environment.sessionVariables = {
    NIXOS_OZONE_WL = "1"; # apps Electron/Chromium nativas en Wayland
  };

  services.upower.enable = true; # batería
  services.gnome.gnome-keyring.enable = true; # secretos y contraseñas guardadas
  services.gvfs.enable = true; # montar discos y redes desde el gestor de archivos
  services.tumbler.enable = true; # miniaturas de imágenes y vídeos
  programs.thunar = {
    enable = true; # gestor de archivos rápido (Super+E)
    plugins = with pkgs; [ thunar-volman thunar-archive-plugin ];
  };
  programs.xfconf.enable = true; # Thunar guarda sus ajustes aquí

  programs.firefox.enable = true;
}
