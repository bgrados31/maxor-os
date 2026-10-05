# ── Marco: el riel, las líneas y los mensajes ────────────────────────
# Toda la salida es lineal y cuelga de un riel vertical:
#
#   ┌  maxor install btop          ui_intro
#   │
#   ◇  Installed btop              ui_step ok …   (también ui_run)
#   │
#   ◇  System                      ui_section
#   │  ✓ no failed services        ui_row ok …
#   │
#   └  Done                        ui_outro
#
# Donde un texto es un mensaje, vale «@clave» + argumentos (ver i18n.sh).
# Fuera de una terminal (o con NO_COLOR) es el mismo texto sin colores.
UI_RAIL=0
UI_GAP=0 # 1 si lo último que se escribió fue una línea vacía del riel: nunca se duplica

ui_rail() { # línea vacía del riel (una sola, aunque se pida varias veces seguidas)
  [ "$UI_GAP" = 1 ] && return 0
  UI_GAP=1
  printf '%s%s%s\n' "$E_MU" "$G_BAR" "$E_RST"
}

ui_intro() { # ui_intro título|@clave [args…]
  local t
  msg t "$@"
  UI_RAIL=1
  UI_GAP=0
  printf '%s%s%s  %s%s%s\n' "$E_AC" "$G_TOP" "$E_RST" "$E_BOLD" "$t" "$E_RST"
}
ui_outro() { # ui_outro [mensaje|@clave [args…]]
  local t=""
  [ $# -gt 0 ] && msg t "$@"
  if [ "$UI_RAIL" = 1 ]; then
    ui_rail
    printf '%s%s%s' "$E_AC" "$G_END" "$E_RST"
    if [ -n "$t" ]; then printf '  %s' "$t"; fi
    printf '\n'
  fi
  UI_RAIL=0
  UI_GAP=0
}
ui_close() { ui_outro; } # cierre sin mensaje

# Una línea de texto dentro del riel (el texto suele empezar con un espacio).
ui_text() { # ui_text "texto (puede llevar color)"
  if [ "$UI_RAIL" = 1 ]; then
    if [ -z "$1" ]; then ui_rail; else UI_GAP=0; printf '%s%s%s %s\n' "$E_MU" "$G_BAR" "$E_RST" "$1"; fi
  else
    printf ' %s\n' "$1"
  fi
}
ui_split() { # ui_split "izquierda" "derecha"  → la derecha queda pegada al borde
  local l r pad
  ui_len "$1"; l=$UI_LEN
  ui_len "$2"; r=$UI_LEN
  ui_repv pad $((ui_w - 4 - l - r < 1 ? 1 : ui_w - 4 - l - r)) ' '
  ui_text "$1$pad$2"
}

# Bloque con título: deja una línea de riel y abre «◇  Título».
ui_section() { # ui_section título|@clave [args…]
  local s
  msg s "$@"
  if [ "$UI_RAIL" = 1 ]; then ui_rail; fi
  UI_GAP=0
  printf '%s%s%s  %s%s%s\n' "$E_OK" "$G_OK" "$E_RST" "$E_BOLD" "$s" "$E_RST"
}

# ui_glyph variable_glifo variable_color nivel paso|fila
ui_glyph() {
  local _lvl="$3" _kind="$4" _g _c
  case "$_lvl" in
    ok) _c="$E_OK"; if [ "$_kind" = row ]; then _g="$G_TICK"; else _g="$G_OK"; fi ;;
    warn) _c="$E_WARN"; _g="$G_WARN" ;;
    bad) _c="$E_BAD"; _g="$G_BAD" ;;
    *) _c="$E_MU"; _g="$G_INFO" ;;
  esac
  printf -v "$1" '%s' "$_g"
  printf -v "$2" '%s' "$_c"
}

# Una fila dentro de un bloque: «│  ✓ mensaje».
ui_row() { # ui_row ok|warn|bad|info mensaje|@clave [args…]
  local g c t lvl="$1"
  shift
  ui_glyph g c "$lvl" row
  msg t "$@"
  ui_truncv t "$t" $((ui_w - 8))
  ui_text " ${c}${g}${E_RST} $t"
}
ui_kv() { # ui_kv clave valor
  local k v
  printf -v k '%-12s' "$1"
  ui_truncv v "$2" $((ui_w - 20))
  ui_text " ${E_MU}${k}${E_RST} $v"
}

# La línea de un paso, sin la línea de riel previa (la usan ui_step y los cargadores).
ui_stepline() { # ui_stepline nivel texto [detalle]
  local g c
  ui_glyph g c "$1" step
  UI_GAP=0
  printf '%s%s%s  %s' "$c" "$g" "$E_RST" "$2"
  if [ -n "${3:-}" ]; then printf '  %s%s%s' "$E_MU" "$3" "$E_RST"; fi
  printf '\n'
}
# Un paso suelto: deja una línea de riel y escribe «◇  mensaje».
ui_step() { # ui_step ok|warn|bad|info mensaje|@clave [args…]
  local t lvl="$1"
  shift
  msg t "$@"
  if [ "$UI_RAIL" = 1 ]; then ui_rail; fi
  ui_stepline "$lvl" "$t"
}
ui_say() { ui_step "$@"; } # el mismo mensaje, dentro o fuera del riel

ui_swatch() { # ui_swatch #hex ...  → bloques de color
  local h c
  for h in "$@"; do
    if [ "$ui_on" = 1 ]; then ui_fgv c "$h"; printf '%s██%s' "$c" "$E_RST"; else printf '#'; fi
  done
}

ui_confirm() { # ui_confirm pregunta|@clave [args…]  →  0 si el usuario acepta
  local r q hint yes w
  msg q "$@"
  msg hint @ui.yes_no
  msg yes @ui.yes_words
  if [ "$UI_RAIL" = 1 ]; then ui_rail; fi
  UI_GAP=0
  printf '%s%s%s  %s %s%s%s ' "$E_AC" "$G_ASK" "$E_RST" "$q" "$E_MU" "$hint" "$E_RST"
  read -r r
  r="${r,,}"
  # «y» y «yes» valen en todos los idiomas; cada catálogo añade sus palabras (ui.yes_words)
  for w in y yes $yes; do [ "$r" = "$w" ] && return 0; done
  return 1
}
