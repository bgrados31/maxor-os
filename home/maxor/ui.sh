# ── Interfaz: una terminal dentro de la terminal ─────────────────────
# Cada comando se dibuja como una ventana con barra de título, pintada con
# la paleta del tema activo (color de 24 bits). Si la salida no es una
# terminal, o NO_COLOR está definido, todo se imprime como texto plano.
shopt -s extglob # patrones como *([0-9;]) en las expansiones, sin procesos externos
ui_on=0
if [ -n "${MAXOR_FORCE_UI:-}" ] || { [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-dumb}" != "dumb" ]; }; then
  ui_on=1
fi

ui_w="${COLUMNS:-$(tput cols 2> /dev/null || echo 80)}"
[ "$ui_w" -gt 78 ] && ui_w=78
[ "$ui_w" -lt 50 ] && ui_w=50

E_RST="" E_BOLD="" E_NB="" E_FG="" E_MU="" E_AC="" E_AC2="" E_BG="" E_BG2=""
E_OK="" E_WARN="" E_BAD="" E_BGOK="" E_BGWARN="" E_BGBAD="" E_INK=""

ui_fgv() { local h="${2#\#}"; printf -v "$1" '\e[38;2;%d;%d;%dm' "0x${h:0:2}" "0x${h:2:2}" "0x${h:4:2}"; } # sin procesos: deja el color en la variable
ui_bgv() { local h="${2#\#}"; printf -v "$1" '\e[48;2;%d;%d;%dm' "0x${h:0:2}" "0x${h:2:2}" "0x${h:4:2}"; }
ui_fg() { local h; h="$(hex "$1")"; printf '\e[38;2;%d;%d;%dm' "0x${h:0:2}" "0x${h:2:2}" "0x${h:4:2}"; }
ui_bg() { local h; h="$(hex "$1")"; printf '\e[48;2;%d;%d;%dm' "0x${h:0:2}" "0x${h:2:2}" "0x${h:4:2}"; }

# Lee la paleta del tema activo; si no hay, usa Sakura nocturna.
ui_palette() {
  [ "$ui_on" = 1 ] || return 0
  local pb="#120b12" ps="#1d121d" ps2="#2a1a2a" pfg="#fbe9f2" pmu="#a88a9d" pac="#ff86b8" pac2="#ffc2a6" pmode="dark"
  local cur="" f
  [ -f "$state/current" ] && cur="$(cat "$state/current")"
  f="$themes/$cur/colors.json"
  # Una sola lectura: valida el tema y saca los colores a la vez.
  local row
  if [ -n "$cur" ] && [ -f "$f" ] && row="$(jq -er 'select([.bg,.s,.s2,.fg,.mu,.ac,.ac2,.on] | all(test("^#[0-9a-fA-F]{6}$")))
      | [.bg,.s,.s2,.fg,.mu,.ac,.ac2,.on,(.mode // "dark")] | @tsv' "$f" 2> /dev/null)"; then
    IFS=$'\t' read -r pb ps ps2 pfg pmu pac pac2 _ pmode <<< "$row"
  fi
  local ok warn bad
  if [ "$pmode" = "light" ]; then ok="#1a7f50"; warn="#9a6700"; bad="#c92a3e"; else ok="#7fe3a8"; warn="#ffc66b"; bad="#ff6b81"; fi
  E_RST=$'\e[0m' E_BOLD=$'\e[1m' E_NB=$'\e[22m'
  ui_fgv E_FG "$pfg"; ui_fgv E_MU "$pmu"; ui_fgv E_AC "$pac"; ui_fgv E_AC2 "$pac2";
  ui_bgv E_BG "$ps"; ui_bgv E_BG2 "$ps2";
  ui_fgv E_OK "$ok"; ui_fgv E_WARN "$warn"; ui_fgv E_BAD "$bad";
  ui_bgv E_BGOK "$ok"; ui_bgv E_BGWARN "$warn"; ui_bgv E_BGBAD "$bad";
  ui_fgv E_INK "$pb";
}
ui_palette

# Texto con color; al terminar vuelve al color base de la ventana.
ui_c() { printf '%s%s%s' "$1" "$2" "$E_FG"; }
# Todo lo de abajo se ejecuta sin lanzar procesos (nada de sed, wc ni $(…) en
# los caminos calientes): una ventana de 30 líneas se pinta en milisegundos.
# Largo visible: quita las secuencias de color y cuenta caracteres.
ui_len() { local s="${1//$'\e'\[*([0-9;])m/}"; UI_LEN=${#s}; }
ui_vlen() { ui_len "$1"; printf '%s' "$UI_LEN"; }
ui_repv() { # ui_repv variable n carácter  → repite el carácter n veces
  local _s=""
  if [ "$2" -gt 0 ]; then printf -v _s '%*s' "$2" ''; _s="${_s// /$3}"; fi
  printf -v "$1" '%s' "$_s"
}
ui_rep() { local _r; ui_repv _r "$1" "$2"; printf '%s' "$_r"; }
ui_truncv() { # ui_truncv variable texto máximo
  local _t="$2"
  if [ "${#2}" -gt "$3" ]; then _t="${2:0:$(($3 - 1))}…"; fi
  printf -v "$1" '%s' "$_t"
}
ui_trunc() { local _o; ui_truncv _o "$1" "$2"; printf '%s' "$_o"; } # ui_trunc texto máximo

ui_open() { # ui_open "título"
  if [ "$ui_on" = 0 ]; then printf '\n%s\n' "$1"; return 0; fi
  local inner=$((ui_w - 2)) pad t
  ui_truncv t "$1" $((inner - 10))
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
  ui_line ""
  ui_line "${E_BOLD}${E_AC2}${1^^}${E_FG}${E_NB}"
}
ui_row() { # ui_row ok|warn|bad|info "mensaje"
  local glyph col t
  case "$1" in
    ok) glyph="✓"; col="$E_OK" ;;
    warn) glyph="!"; col="$E_WARN" ;;
    bad) glyph="✗"; col="$E_BAD" ;;
    *) glyph="·"; col="$E_MU" ;;
  esac
  ui_truncv t "$2" $((ui_w - 9))
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
ui_say() { # ui_say ok|warn|bad|info mensaje
  local glyph col
  case "$1" in
    ok) glyph="✓"; col="$E_OK" ;;
    warn) glyph="!"; col="$E_WARN" ;;
    bad) glyph="✗"; col="$E_BAD" ;;
    *) glyph="·"; col="$E_MU" ;;
  esac
  printf ' %s%s%s  %s\n' "$col" "$glyph" "$E_RST" "$2"
}
ui_pill() { # ui_pill ok|warn|bad "TEXTO"
  local bg
  case "$1" in ok) bg="$E_BGOK" ;; warn) bg="$E_BGWARN" ;; *) bg="$E_BGBAD" ;; esac
  printf '%s%s%s %s %s' "$bg" "$E_INK" "$E_BOLD" "$2" "$E_RST"
}
die() { printf '%s✗%s maxor: %s\n' "$E_BAD" "$E_RST" "$*" >&2; exit 1; }

ui_confirm() { # ui_confirm "pregunta"  →  0 si el usuario acepta
  local r
  printf ' %s?%s  %s %s[s/N]%s ' "$E_AC" "$E_RST" "$1" "$E_MU" "$E_RST"
  read -r r
  case "$r" in s | S | y | Y) return 0 ;; *) return 1 ;; esac
}

# Ejecuta un comando largo con un spinner. La salida estándar queda en $UI_OUT;
# si falla, se muestran las últimas líneas del error.
UI_OUT=""
ui_run() { # ui_run "mensaje" comando args…
  local msg="$1" log out rc=0 pid i=0
  shift
  log="$(mktemp)"; out="$(mktemp)"
  if [ "$ui_on" = 1 ] && [ -t 1 ]; then
    local frames=(⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)
    "$@" > "$out" 2> "$log" &
    pid=$!
    printf '\e[?25l'
    while kill -0 "$pid" 2> /dev/null; do
      printf '\r %s%s%s  %s' "$E_AC" "${frames[i % 10]}" "$E_RST" "$msg"
      i=$((i + 1))
      sleep 0.1
    done
    printf '\r\e[2K\e[?25h'
    wait "$pid" || rc=$?
  else
    "$@" > "$out" 2> "$log" || rc=$?
  fi
  UI_OUT="$(cat "$out")"
  if [ "$rc" = 0 ]; then
    ui_say ok "$msg"
  else
    ui_say bad "$msg"
    tail -n 15 "$log" | sed 's/^/     /' >&2
  fi
  rm -f "$log" "$out"
  return "$rc"
}
