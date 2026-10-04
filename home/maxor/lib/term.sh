# ── Terminal: capacidades, glifos, paleta del tema y utilidades de texto ─
# La salida es lineal, con un riel vertical y la paleta del tema activo (color
# de 24 bits). Si la salida no es una terminal, o NO_COLOR está definido, es el
# mismo texto sin colores ni animación.
shopt -s extglob # patrones como *([0-9;]) en las expansiones, sin procesos externos
ui_on=0
if [ -n "${MAXOR_FORCE_UI:-}" ] || { [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-dumb}" != "dumb" ]; }; then
  ui_on=1
fi

ui_w="${COLUMNS:-$(tput cols 2> /dev/null || echo 80)}"
[ "$ui_w" -gt 78 ] && ui_w=78
[ "$ui_w" -lt 50 ] && ui_w=50

# ── Glifos ───────────────────────────────────────────────────────────
# Unicode, con respaldo ASCII en terminales limitadas (TERM=linux o un locale que
# no es UTF-8) o con MAXOR_ASCII=1. Un solo vocabulario para toda la CLI.
ui_ascii=0
if [ "${MAXOR_ASCII:-0}" = 1 ] || [ "${TERM:-}" = linux ]; then
  ui_ascii=1
elif [ -n "$MAXOR_ORIG_LOCALE" ]; then
  case "${MAXOR_ORIG_LOCALE,,}" in *utf-8* | *utf8*) ;; *) ui_ascii=1 ;; esac
fi
if [ "$ui_ascii" = 1 ]; then
  G_FIND='?' G_TEE='+' G_TOP='+' G_BAR='|' G_END='+' G_OK='o' G_ASK='*' G_TICK='v' G_WARN='!' G_BAD='x' G_INFO='-'
  G_SEL='>' G_ON='[x]' G_OFF='[ ]' G_DOT='*' G_UP='^' G_ADD='+' G_DEL='-' G_CHG='~' G_BARON='#' G_BAROFF='-'
  UI_SPIN=('|' '/' '-' "\\")
else
  G_FIND='⌕' G_TEE='├' G_TOP='┌' G_BAR='│' G_END='└' G_OK='◇' G_ASK='◆' G_TICK='✓' G_WARN='!' G_BAD='✗' G_INFO='·'
  G_SEL='❯' G_ON='◼' G_OFF='◻' G_DOT='•' G_UP='↑' G_ADD='+' G_DEL='−' G_CHG='~' G_BARON='▰' G_BAROFF='▱'
  UI_SPIN=(⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)
fi

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
ui_c() { printf '%s%s%s' "$1" "$2" "$E_RST"; }
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

# ── Repintado en el sitio ────────────────────────────────────────────
# Para animar un bloque de varias líneas (selector, pasos, esqueleto): se sube
# tantas líneas como tenía el marco anterior y se sobrescribe cada una,
# borrando solo el resto de la línea. En salida sincronizada (kitty y otros)
# no hay parpadeo.
ui_count_lines() { local s="${1//[^$'\n']/}"; UI_LINES=$((${#s} + 1)); } # sin procesos: deja el valor en UI_LINES
ui_paint() { # ui_paint "marco" líneas_previas
  local out=""
  if [ "$2" -gt 0 ]; then out=$'\e['"$2"'A'; fi
  out+="${1//$'\n'/$'\e[K\n'}"$'\e[K\n'
  printf '\e[?2026h%s\e[J\e[?2026l' "$out"
}

# Apaga colores y animaciones (--no-color).
ui_disable() {
  ui_on=0
  E_RST="" E_BOLD="" E_NB="" E_FG="" E_MU="" E_AC="" E_AC2="" E_BG="" E_BG2=""
  E_OK="" E_WARN="" E_BAD="" E_BGOK="" E_BGWARN="" E_BGBAD="" E_INK=""
}
