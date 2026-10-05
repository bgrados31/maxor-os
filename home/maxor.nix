{ config, pkgs, lib, maxorVersion ? null, osConfig ? { }, ... }:

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
      # El wallpaper: un degradado en diagonal por los colores de gradient.json (la paleta de la marca), o, sin
      # él, un halo de la segunda superficie sobre el fondo.
      gradientFile = themesDir + "/${id}/gradient.json";
      stops = builtins.fromJSON (builtins.readFile gradientFile);
      last = builtins.length stops - 1;
      points = lib.imap0 (i: c: "${toString (i * 2560 / last)},${toString (i * 1600 / last)} ${c}") stops;
      base =
        if builtins.pathExists gradientFile && last > 0 then
          "-size 2560x1600 xc: -sparse-color Shepards '${lib.concatStringsSep " " points}' -colorspace sRGB"
        else
          "-size 2560x1600 radial-gradient:'${colors.s2}'-'${colors.bg}' -colorspace sRGB";
      # Firma visual de Maxor: viñeta suave, un resplandor del segundo acento en la esquina superior
      # derecha con tres órbitas finas a su alrededor, y grano (que además evita las bandas del degradado).
      # En temas claros el resplandor es más tenue y las órbitas oscuras, para no lavar el fondo.
      dark = (colors.mode or "dark") == "dark";
      glow = if dark then "0.70" else "0.35";
      vignette = if dark then "gray45" else "gray80";
      orbit = if dark then "rgba(255,255,255,0.08)" else "rgba(0,0,0,0.10)";
      wallpaper = ''
        magick ${base} \
          \( -size 2560x1600 radial-gradient:white-${vignette} \) -compose multiply -composite \
          \( -size 2560x1600 -define gradient:center=2300,100 -define gradient:radii=1100,1100 radial-gradient:'${colors.ac2}'-black -evaluate multiply ${glow} \) -compose screen -composite \
          \( -size 2560x1600 xc:none -fill none -stroke '${orbit}' -strokewidth 2 -draw 'circle 2300,100 2300,700' -draw 'circle 2300,100 2300,1100' -draw 'circle 2300,100 2300,1500' \) -compose over -composite \
          \( -size 2560x1600 xc:gray50 +noise Gaussian -attenuate 0.25 -colorspace gray \) -compose overlay -composite \
          -depth 8 PNG24:$out/wallpaper.png
      '';
    in
    pkgs.runCommand "maxor-theme-${id}" { nativeBuildInputs = [ pkgs.imagemagick ]; } ''
      mkdir -p $out
      cp ${themesDir + "/${id}/colors.json"} $out/colors.json
      cp ${themesDir + "/${id}/theme.toml"} $out/theme.toml
      ${lib.optionalString (builtins.pathExists styleFile) "cp ${styleFile} $out/style.json"}
      ${wallpaper}
    '';

  officialThemes = lib.genAttrs themeIds mkTheme;

  maxor = pkgs.callPackage ../packages/maxor.nix (lib.optionalAttrs (maxorVersion != null) { inherit maxorVersion; });
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

  # Aspecto de Maxor (el tema elegido al instalar y su wallpaper) escrito una sola vez, al arrancar el equipo y antes
  # del login (machine.nix hace esperar a greetd): el login y la primera sesión ya salen con él, sin
  # recargas. No pisa un tema ya aplicado. Nunca rompe la activación.
  home.activation.maxorFirstRun = lib.hm.dag.entryAfter [ "linkGeneration" ] ''
    MAXOR_FIRSTRUN_THEME=${lib.escapeShellArg (osConfig.maxor.theme or "maxor-dark")} \
      run ${maxor}/bin/maxor firstrun > /dev/null 2>&1 || true
  '';

  # Temas oficiales: carpetas de solo lectura junto a los tuyos, y sus
  # wallpapers en ~/Pictures/Wallpapers para el selector de DMS.
  home.file = (lib.mapAttrs'
    (id: t: lib.nameValuePair ".local/share/maxor/themes/${id}" { source = t; })
    officialThemes) // (lib.mapAttrs'
    (id: t: lib.nameValuePair "Pictures/Wallpapers/maxor-${id}.png" { source = "${t}/wallpaper.png"; })
    officialThemes);
}
