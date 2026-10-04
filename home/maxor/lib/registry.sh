# ── Registro de comandos: ayuda, errores de uso y autocompletado ─────
# Cada archivo de cmd/ declara sus comandos con
#
#   maxor_cmd nombre grupo "subcomando subcomando"
#
# y el resto sale de ahí: la lista de `maxor`, `maxor help <comando>` (texto
# help.<nombre> del catálogo), el autocompletado y la despacho de main.sh.
# El resumen de cada comando es la clave cmd.<nombre> del catálogo.
declare -ga CMD_ORDER=()
declare -gA CMD_GROUP=() CMD_SUBS=()
CMD_GROUPS=(appearance apps system tools)

maxor_cmd() { # maxor_cmd nombre grupo "subcomandos"
  CMD_ORDER+=("$1")
  CMD_GROUP["$1"]="$2"
  CMD_SUBS["$1"]="${3:-}"
}

cmd_known() { local c; for c in "${CMD_ORDER[@]}"; do [ "$c" = "$1" ] && return 0; done; return 1; }

# Texto de ayuda de un comando, dentro de una ventana.
help_window() { # help_window comando
  local body line
  msg body "@help.$1"
  echo
  ui_intro "maxor help $1"
  ui_text ""
  while IFS= read -r line; do
    if [ -n "$line" ]; then ui_text " $line"; else ui_text ""; fi
  done <<< "$body"
  ui_outro
}

# Uso incorrecto: muestra la ayuda del comando y sale con EX_USAGE.
usage_error() { # usage_error comando
  help_window "$1" >&2
  exit "$EX_USAGE"
}

cmd_help() {
  local c="${1:-}"
  if [ -n "$c" ]; then
    if cmd_known "$c"; then help_window "$c"; else die_code "$EX_USAGE" @err.unknown_command "$c"; fi
    return 0
  fi
  usage
}

# Pantalla principal: los comandos agrupados, con su resumen.
usage() {
  local g c title sum cur="-" pretty name
  [ -f "$state/current" ] && cur="$(cat "$state/current")"
  pretty="$(os_pretty)"
  echo
  ui_intro "${E_AC}M A X O R${E_RST}  ${E_MU}$pretty · maxor $MAXOR_VERSION${E_RST}"
  for g in "${CMD_GROUPS[@]}"; do
    msg title "@group.$g"
    ui_section "$title"
    for c in "${CMD_ORDER[@]}"; do
      [ "${CMD_GROUP[$c]}" = "$g" ] || continue
      msg sum "@cmd.$c"
      printf -v name '%-12s' "$c"
      ui_truncv sum "$sum" $((ui_w - 20))
      ui_text " ${E_AC}${name}${E_RST}${sum}"
    done
  done
  msg sum @help.footer
  ui_outro "${E_MU}${sum}${E_RST}"
  echo
}
