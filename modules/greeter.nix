{ config, pkgs, lib, inputs, ... }:

# Pantalla de login de Maxor OS: el greeter de DankMaterialShell sobre greetd,
# con el paquete Maxor Shell (logo y nombre propios). Corre dentro de Hyprland y
# hereda del usuario el tema activo, los colores y el wallpaper: al cambiar de
# tema con `maxor theme apply`, el login del próximo arranque lo refleja.
#
# Qué usuario aporta esa configuración se define por equipo:
#   services.displayManager.dms-greeter.configHome = "/home/<usuario>";
#
# Si el login no arrancara, elige una generación anterior en el menú de
# arranque (con SDDM) o entra por una TTY con Ctrl+Alt+F3.
{
  services.displayManager.dms-greeter = {
    enable = true;
    package = pkgs.callPackage ../packages/maxor-shell.nix {
      dmsShell = inputs.dms.packages.${pkgs.stdenv.hostPlatform.system}.dms-shell;
    };
    compositor.name = "hyprland";
  };
}
