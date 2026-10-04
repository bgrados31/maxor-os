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
let
  defaultBg = (builtins.fromJSON (builtins.readFile ../themes/sakura/colors.json)).bg;
in
{
  services.displayManager.dms-greeter = {
    enable = true;
    package = pkgs.callPackage ../packages/maxor-shell.nix {
      dmsShell = inputs.dms.packages.${pkgs.stdenv.hostPlatform.system}.dms-shell;
    };
    compositor.name = "hyprland";
    # El Hyprland del propio login. Sin esto es el de fábrica: un gris #111 con una frase, que se ve un momento
    # mientras el login se desvanece. Con el fondo del tema por defecto, el fundido va del login al mismo color
    # con el que arranca la sesión. (Una configuración Lua sustituye a la del greeter, por eso también fija
    # DMS_RUN_GREETER; dms-greeter le añade la orden que arranca el greeter.)
    compositor.customConfig = ''
      hl.env("DMS_RUN_GREETER", "1")
      hl.config({
        misc = {
          disable_hyprland_logo = true,
          disable_splash_rendering = true,
          background_color = "rgb(${lib.removePrefix "#" defaultBg})",
        },
      })
    '';
  };
}
