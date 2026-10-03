# ── Marco: ventana, líneas, filas y mensajes ─────────────────────────
# Donde un texto es un mensaje, vale «@clave» + argumentos (ver i18n.sh).
ui_open() { # ui_open título|@clave [args…]
  local title
  msg title "$@"
  if [ "$ui_on" = 0 ]; then printf '\n%s\n' "$title"; return 0; fi
  local inner=$((ui_w - 2)) pad t
  ui_truncv t "$title" $((inner - 10))
  ui_repv pad $((inner - 8 - ${#t})) ' '
  printf '%s╭%s %s●%s ●%s ●%s  %s%s%s%s%s%s╮%s\n' \
    "$E_AC" "$E_BG2" "$E_AC" "$E_AC2" "$E_MU" "$E_FG" "$E_BOLD" "$t" "$E_NB" "$pad" "$E_RST" "$E_AC" "$E_RST"
}
ui_line() { # ui_line "texto (puede llevar color)"
  if [ "$ui_on" = 0 ]; then printf '  %s\n' "$1"; return 0; fi
  local pad
  ui_len "$1"
  ui_repv pad $((ui_w - 4 - UI_LEN)) ' '
  printf '%s│%s%s%s %s%s %s%s│%s\n' "$E_AC" "$E_RST" "$E_BG" "$E_FG" "$1" "$pad" "$E_RST" "$E_AC" "$E_RST"
}
ui_split() { # ui_split "izquierda" "derecha"  → la derecha queda pegada al borde
  if [ "$ui_on" = 0 ]; then printf '  %s   %s\n' "$1" "$2"; return 0; fi
  local l r pad
  ui_len "$1"; l=$UI_LEN
  ui_len "$2"; r=$UI_LEN
  ui_repv pad $((ui_w - 4 - l - r < 1 ? 1 : ui_w - 4 - l - r)) ' '
  ui_line "$1$pad$2"
}
ui_close() {
  if [ "$ui_on" = 0 ]; then return 0; fi
  local bar
  ui_repv bar $((ui_w - 2)) '─'
  printf '%s╰%s╯%s\n' "$E_AC" "$bar" "$E_RST"
}
ui_section() { # título de bloque dentro de la ventana
  local s
  msg s "$@"
  ui_line ""
  ui_line "${E_BOLD}${E_AC2}${s^^}${E_FG}${E_NB}"
}
ui_row() { # ui_row ok|warn|bad|info mensaje|@clave [args…]
  local glyph col t lvl="$1"
  shift
  case "$lvl" in
    ok) glyph="✓"; col="$E_OK" ;;
    warn) glyph="!"; col="$E_WARN" ;;
    bad) glyph="✗"; col="$E_BAD" ;;
    *) glyph="·"; col="$E_MU" ;;
  esac
  msg t "$@"
  ui_truncv t "$t" $((ui_w - 9))
  ui_line " ${col}${glyph}${E_FG}  $t"
}
ui_kv() { # ui_kv clave valor
  local k v
  printf -v k '%-12s' "$1"
  ui_truncv v "$2" $((ui_w - 19))
  ui_line " ${E_MU}${k}${E_FG} $v"
}
ui_swatch() { # ui_swatch #hex ...  → bloques de color
  local h c
  for h in "$@"; do
    if [ "$ui_on" = 1 ]; then ui_fgv c "$h"; printf '%s██%s' "$c" "$E_FG"; else printf '#'; fi
  done
}
# Mensajes fuera de la ventana.
ui_say() { # ui_say ok|warn|bad|info mensaje|@clave [args…]
  local glyph col t lvl="$1"
  shift
  case "$lvl" in
    ok) glyph="✓"; col="$E_OK" ;;
    warn) glyph="!"; col="$E_WARN" ;;
    bad) glyph="✗"; col="$E_BAD" ;;
    *) glyph="·"; col="$E_MU" ;;
  esac
  msg t "$@"
  printf ' %s%s%s  %s\n' "$col" "$glyph" "$E_RST" "$t"
}
ui_pill() { # ui_pill ok|warn|bad "TEXTO"
  local bg
  case "$1" in ok) bg="$E_BGOK" ;; warn) bg="$E_BGWARN" ;; *) bg="$E_BGBAD" ;; esac
  printf '%s%s%s %s %s' "$bg" "$E_INK" "$E_BOLD" "$2" "$E_RST"
}

ui_confirm() { # ui_confirm pregunta|@clave [args…]  →  0 si el usuario acepta
  local r q
  msg q "$@"
  printf ' %s?%s  %s %s[y/N]%s ' "$E_AC" "$E_RST" "$q" "$E_MU" "$E_RST"
  read -r r
  case "$r" in y | Y | yes | YES | s | S) return 0 ;; *) return 1 ;; esac
}
