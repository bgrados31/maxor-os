{ lib, config, ... }:

# Hyprland en módulos Lua, cada uno con una sola responsabilidad:
#
#   settings.lua   apariencia, entrada y animaciones
#   rules.lua      reglas de ventana y de capa
#   binds.lua      atajos
#   user.lua       tuyo: se carga el último y el sistema nunca lo pisa
#
# Los tres primeros son de solo lectura (vienen de este repo). Los colores,
# monitores y cursor los gestiona DankMaterialShell (módulos `dms.*`).
{
  wayland.windowManager.hyprland = {
    enable = true;
    configType = "lua"; # Hyprland 0.55+ usa Lua
    # Los paquetes vienen de programs.hyprland (NixOS)
    package = null;
    portalPackage = null;
    systemd.variables = [ "--all" ];

    extraConfig = ''
      -- Módulos de Maxor OS
      require("maxor.settings")
      require("maxor.rules")
      require("maxor.binds")

      -- Archivos gestionados por DMS (colores del tema, monitores, etc.)
      require("dms.colors")
      require("dms.outputs")
      require("dms.layout")
      require("dms.cursor")
      require("dms.binds-user")
      require("dms.windowrules")

      -- Tu configuración personal, al final para poder sobrescribir todo
      require("maxor.user")
    '';
  };

  xdg.configFile = {
    "hypr/maxor/settings.lua".source = ./hyprland/settings.lua;
    "hypr/maxor/rules.lua".source = ./hyprland/rules.lua;
    "hypr/maxor/binds.lua".source = ./hyprland/binds.lua;
  };

  # user.lua se crea una sola vez, con ejemplos comentados. Después es tuyo.
  home.activation.maxorUserLua = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
    p="${config.xdg.configHome}/hypr/maxor/user.lua"
    if [ ! -e "$p" ]; then
      run mkdir -p "$(dirname "$p")"
      run cp ${./hyprland/user.lua.example} "$p"
      run chmod 644 "$p"
    fi
  '';
}
