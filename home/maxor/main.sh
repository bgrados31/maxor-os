# ── Ayuda y despacho ─────────────────────────────────────────────────
usage() {
  local cur="(ninguno)"
  [ -f "$state/current" ] && cur="$(cat "$state/current")"
  echo
  ui_open "maxor"
  ui_line ""
  ui_line " ${E_BOLD}$(ui_c "$E_AC" "M A X O R   O S")${E_NB}"
  ui_line " $(ui_c "$E_MU" "$(grep -m1 '^PRETTY_NAME=' /etc/os-release | cut -d= -f2 | tr -d '"') · tema $cur")"
  ui_section "Temas"
  ui_line " $(ui_c "$E_AC" "theme list")                  temas instalados"
  ui_line " $(ui_c "$E_AC" "theme current")               tema activo"
  ui_line " $(ui_c "$E_AC" "theme apply") <nombre>        aplicar un tema"
  ui_line " $(ui_c "$E_AC" "theme undo")                  volver al anterior"
  ui_line " $(ui_c "$E_AC" "theme install") <ruta>        instalar desde carpeta o .tar.gz"
  ui_line " $(ui_c "$E_AC" "theme export") <nombre>       empaquetar un tema"
  ui_section "Sistema"
  ui_line " $(ui_c "$E_AC" "update") [-y] [--no-lock]     actualizar mostrando los cambios"
  ui_line " $(ui_c "$E_AC" "rollback") [-y]               volver a la generación anterior"
  ui_line " $(ui_c "$E_AC" "doctor")                      diagnóstico del sistema"
  ui_line ""
  ui_close
  echo
}

case "${1:-}" in
  theme)
    case "${2:-}" in
      list) theme_list ;;
      current) cat "$state/current" 2> /dev/null || echo "(ninguno)" ;;
      apply) [ -n "${3:-}" ] || die "uso: maxor theme apply <nombre>"; theme_apply "$3" ;;
      undo) theme_undo ;;
      install) shift 2; theme_install "$@" ;;
      export) theme_export "${3:-}" ;;
      *) usage; exit 1 ;;
    esac ;;
  update) shift; cmd_update "$@" ;;
  rollback) shift; cmd_rollback "$@" ;;
  doctor) cmd_doctor ;;
  "" | -h | --help | help) usage ;;
  *) usage; exit 1 ;;
esac
