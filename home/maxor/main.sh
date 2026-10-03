# ── Despacho: banderas globales y comandos ───────────────────────────
# Banderas globales (valen en cualquier comando y en cualquier posición):
#   --no-color      salida sin colores ni animaciones
#   -q, --quiet     solo errores
#   -v, --verbose   escribe el registro también en stderr
#   --lang CÓDIGO   idioma de los mensajes (hoy solo en)
main() {
  log_init
  local args=()
  while [ $# -gt 0 ]; do
    case "$1" in
      --no-color) ui_disable ;;
      -q | --quiet) MAXOR_QUIET=1 ;;
      -v | --verbose) MAXOR_VERBOSE=1 ;;
      --lang) maxor_lang="${2:-en}"; shift ;;
      --lang=*) maxor_lang="${1#--lang=}" ;;
      *) args+=("$1") ;;
    esac
    shift
  done
  i18n_init
  # El cursor vuelve siempre, aunque el comando se interrumpa a mitad de una animación.
  if [ -t 1 ]; then trap 'printf "\e[?25h"' EXIT; fi
  trap 'exit 130' INT

  set -- "${args[@]+"${args[@]}"}"
  local c="${1:-}"
  shift || true
  case "$c" in
    "") usage ;;
    -h | --help) usage ;;
    -V | --version) cmd_version "$@" ;;
    help) cmd_help "${1:-}" ;;
    *)
      if cmd_known "$c"; then
        log INFO "maxor $c $*"
        "cmd_${c//-/_}" "$@"
      else
        ui_error @err.unknown_command_title @err.unknown_command_cause @err.unknown_command_hint
        exit "$EX_USAGE"
      fi
      ;;
  esac
}

main "$@"
