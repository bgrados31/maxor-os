{ config, pkgs, lib, ... }:

# Escritorio de Maxor OS: Hyprland como única sesión, login y servicios que
# el escritorio necesita. La configuración de Hyprland en sí vive en home/.
{
  programs.hyprland.enable = true;

  # El login lo gestiona modules/greeter.nix (greetd + greeter de DMS).
  services.displayManager.defaultSession = "hyprland";

  # hyprlock necesita su servicio PAM; sin esto no podrías desbloquear.
  security.pam.services.hyprlock = { };

  environment.sessionVariables = {
    NIXOS_OZONE_WL = "1"; # apps Electron/Chromium nativas en Wayland
  };

  services.upower.enable = true; # batería
  services.gnome.gnome-keyring.enable = true; # secretos y contraseñas guardadas
  services.gvfs.enable = true; # montar discos y redes desde el gestor de archivos

  programs.firefox.enable = true;
}
