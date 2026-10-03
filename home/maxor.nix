{ config, pkgs, lib, ... }:

# Motor de temas y CLI de Maxor OS.
#
# Los temas oficiales son las carpetas de ../themes (colors.json + theme.toml);
# aquí se les genera el wallpaper y se instalan junto a los del usuario en
# ~/.local/share/maxor/themes/. La CLI vive en ./maxor/ (scripts reales, con
# shellcheck al compilar) y se ensambla en este orden:
#
#   lib/   núcleo, terminal, marco de ventana, cargadores y motor de formas
#   cmd/   un archivo por grupo de comandos (theme, update, doctor, hardware…)
#   main.sh  ayuda y despacho de comandos
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

  maxor = pkgs.writeShellApplication {
    name = "maxor";
    runtimeInputs = with pkgs; [ jq coreutils gnused gnugrep gawk gnutar findutils procps ncurses ];
    runtimeEnv.MAXOR_PROFILES = ../modules/profiles-catalog.json;
    excludeShellChecks = [ "SC2001" "SC2155" "SC2086" "SC2012" "SC2015" "SC2016" ];
    text = lib.concatMapStringsSep "\n" (f: builtins.readFile (./maxor + "/${f}")) [
      "lib/core.sh"
      "lib/term.sh"
      "lib/frame.sh"
      "lib/loaders.sh"
      "lib/style.sh"
      "cmd/theme.sh"
      "cmd/update.sh"
      "cmd/doctor.sh"
      "cmd/hardware.sh"
      "cmd/apps.sh"
      "cmd/profile.sh"
      "main.sh"
    ];
  };
in
{
  home.packages = [ maxor ];

  # Temas oficiales: carpetas de solo lectura junto a los tuyos, y sus
  # wallpapers en ~/Pictures/Wallpapers para el selector de DMS.
  home.file = (lib.mapAttrs'
    (id: t: lib.nameValuePair ".local/share/maxor/themes/${id}" { source = t; })
    officialThemes) // (lib.mapAttrs'
    (id: t: lib.nameValuePair "Pictures/Wallpapers/maxor-${id}.png" { source = "${t}/wallpaper.png"; })
    officialThemes);
}
