{ lib, writeShellApplication, jq, coreutils, gnused, gnugrep, gawk, gnutar, gzip, findutils, procps, ncurses }:

# La CLI `maxor`. Los scripts reales viven en home/maxor/ (lib/ y cmd/) y se
# ensamblan en este orden; shellcheck revisa el resultado al compilar.
#   lib/core      rutas, códigos de salida, registro, errores
#   lib/i18n      catálogo de mensajes (lib/lang/<idioma>.sh)
#   lib/term      capacidades, paleta del tema, utilidades de texto
#   lib/frame     ventana, líneas, filas, mensajes
#   lib/loaders   spinner, ventana de salida, pasos, barra, esqueleto
#   lib/widgets   segmentos, diff y errores
#   lib/style     forma de los temas (style.json → Lua)
#   lib/registry  registro de comandos, ayuda y errores de uso
#   cmd/*         un archivo por grupo de comandos
#   main          banderas globales y despacho
let
  src = ../home/maxor;
  files = [
    "lib/core.sh"
    "lib/i18n.sh"
    "lib/lang/en.sh"
    "lib/term.sh"
    "lib/frame.sh"
    "lib/loaders.sh"
    "lib/widgets.sh"
    "lib/style.sh"
    "lib/registry.sh"
    "cmd/theme.sh"
    "cmd/update.sh"
    "cmd/doctor.sh"
    "cmd/hardware.sh"
    "cmd/apps.sh"
    "cmd/profile.sh"
    "cmd/backup.sh"
    "cmd/tools.sh"
    "main.sh"
  ];
in
writeShellApplication {
  name = "maxor";
  runtimeInputs = [ jq coreutils gnused gnugrep gawk gnutar gzip findutils procps ncurses ];
  runtimeEnv = {
    MAXOR_PROFILES = ../modules/profiles-catalog.json;
    MAXOR_VERSION = lib.removeSuffix "\n" (builtins.readFile ../VERSION);
  };
  excludeShellChecks = [ "SC2034" "SC2001" "SC2155" "SC2086" "SC2012" "SC2015" "SC2016" ];
  text = lib.concatMapStringsSep "\n" (f: builtins.readFile (src + "/${f}")) files;
}
