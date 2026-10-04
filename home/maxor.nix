{ config, pkgs, lib, ... }:

# Motor de temas y CLI de Maxor OS.
#
# Los temas oficiales son las carpetas de ../themes (colors.json + theme.toml);
# aquí se les genera el wallpaper y se instalan junto a los del usuario en
# ~/.local/share/maxor/themes/. La CLI es el paquete packages/maxor.nix.
let
  themesDir = ../themes;
  themeIds = builtins.attrNames (lib.filterAttrs (_: kind: kind == "directory") (builtins.readDir themesDir));

  mkTheme = id:
    let
      colors = builtins.fromJSON (builtins.readFile (themesDir + "/${id}/colors.json"));
      styleFile = themesDir + "/${id}/style.json";
    in
    pkgs.runCommand "maxor-theme-${id}" { nativeBuildInputs = [ pkgs.imagemagick ]; } ''
      mkdir -p $out
      cp ${themesDir + "/${id}/colors.json"} $out/colors.json
      cp ${themesDir + "/${id}/theme.toml"} $out/theme.toml
      ${lib.optionalString (builtins.pathExists styleFile) "cp ${styleFile} $out/style.json"}
      magick -size 2560x1600 radial-gradient:'${colors.s2}'-'${colors.bg}' -colorspace sRGB $out/wallpaper.png
    '';

  officialThemes = lib.genAttrs themeIds mkTheme;

  maxor = pkgs.callPackage ../packages/maxor.nix { };
  maxorTui = pkgs.callPackage ../packages/maxor-tui.nix { };
in
{
  home.packages = [ maxor maxorTui ];

  # Una vez al día mira si alguna app instalada con maxor tiene versión nueva y, si la hay,
  # lo avisa con una notificación. Mirar es barato (no compila ni descarga nada del sistema);
  # actualizar sigue siendo cosa tuya, desde la Tienda.
  systemd.user.services.maxor-app-updates = {
    Unit.Description = "Look for new versions of the apps installed with maxor";
    Service = {
      Type = "oneshot";
      ExecStart = "${maxor}/bin/maxor apps updates --refresh --notify";
      Environment = [ "PATH=${lib.makeBinPath [ pkgs.libnotify pkgs.coreutils pkgs.nix pkgs.flatpak ]}" ];
      Nice = 15;
    };
  };
  systemd.user.timers.maxor-app-updates = {
    Unit.Description = "Daily look for new app versions";
    Timer = {
      OnCalendar = "daily";
      RandomizedDelaySec = "30min";
      Persistent = true;
    };
    Install.WantedBy = [ "timers.target" ];
  };

  # Cada hora pregunta, con una petición condicional (ETag: si no hay nada nuevo, el servidor
  # responde 304 sin cuerpo), si hay una release de Maxor OS. La respuesta solo vale si su
  # firma es de la clave de release; avisa una vez por versión. El mismo chequeo corre
  # al iniciar sesión y cuando se abre la pantalla completa. Sin red no es un fallo (código 4).
  systemd.user.services.maxor-release-check = {
    Unit = {
      Description = "Look for a new signed Maxor OS release";
      After = [ "network-online.target" ];
    };
    Service = {
      Type = "oneshot";
      ExecStart = "${maxor}/bin/maxor release check --notify --quiet";
      SuccessExitStatus = [ 4 ];
      Nice = 15;
    };
  };
  systemd.user.timers.maxor-release-check = {
    Unit.Description = "Hourly look for a new Maxor OS release";
    Timer = {
      OnStartupSec = "2min";
      OnUnitActiveSec = "1h";
      RandomizedDelaySec = "5min";
    };
    Install.WantedBy = [ "timers.target" ];
  };

  # Temas oficiales: carpetas de solo lectura junto a los tuyos, y sus
  # wallpapers en ~/Pictures/Wallpapers para el selector de DMS.
  home.file = (lib.mapAttrs'
    (id: t: lib.nameValuePair ".local/share/maxor/themes/${id}" { source = t; })
    officialThemes) // (lib.mapAttrs'
    (id: t: lib.nameValuePair "Pictures/Wallpapers/maxor-${id}.png" { source = "${t}/wallpaper.png"; })
    officialThemes);
}
