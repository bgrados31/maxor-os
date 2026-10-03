{ config, pkgs, lib, ... }:

# Escritorio de Maxor OS: Hyprland como única sesión, login y servicios que
# el escritorio necesita. La configuración de Hyprland en sí vive en home/.
{
  programs.hyprland.enable = true;

  # SDDM en Wayland hace de login por ahora; se reemplazará por el greeter de
  # Maxor. El respaldo ante un fallo son las generaciones del menú de arranque.
  services.displayManager.sddm = {
    enable = true;
    wayland.enable = true;
  };
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
