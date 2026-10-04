{ lib, writeShellApplication, jq, coreutils, gnused, gnugrep, gawk, gnutar, gzip, findutils, procps, ncurses
, curl, openssh, git, util-linux, libnotify
  # Las claves en las que confía y el canal que consulta. Los valores por defecto son los de
  # producción; solo las pruebas (tests/vm) los cambian, para firmar con una clave de prueba.
, releaseKeys ? ../keys/allowed_signers
, releaseUrl ? null
  # La versión que dice ser. Por defecto, VERSION; la VM «vm-old» se hace pasar por una vieja.
, maxorVersion ? lib.removeSuffix "\n" (builtins.readFile ../VERSION) }:

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
    "cmd/release.sh"
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
  runtimeInputs = [ jq coreutils gnused gnugrep gawk gnutar gzip findutils procps ncurses curl openssh git util-linux libnotify ];
  runtimeEnv = lib.optionalAttrs (releaseUrl != null) { MAXOR_RELEASE_URL = releaseUrl; } // {
    MAXOR_PROFILES = "${../modules/profiles-catalog.json}"; # interpolado: Nix lo copia al store y lo mantiene vivo
    # Las claves de confianza van dentro del paquete: no se pueden cambiar sin cambiar el sistema.
    MAXOR_RELEASE_KEYS = "${releaseKeys}";
    MAXOR_VERSION = maxorVersion;
  };
  excludeShellChecks = [ "SC2034" "SC2001" "SC2155" "SC2086" "SC2012" "SC2015" "SC2016" ];
  text = lib.concatMapStringsSep "\n" (f: builtins.readFile (src + "/${f}")) files;
}
