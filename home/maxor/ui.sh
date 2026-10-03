# ── Interfaz: una terminal dentro de la terminal ─────────────────────
# Cada comando se dibuja como una ventana con barra de título, pintada con
# la paleta del tema activo (color de 24 bits). Si la salida no es una
# terminal, o NO_COLOR está definido, todo se imprime como texto plano.
ui_on=0
if [ -n "${MAXOR_FORCE_UI:-}" ] || { [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-dumb}" != "dumb" ]; }; then
  ui_on=1
fi

ui_w="${COLUMNS:-$(tput cols 2> /dev/null || echo 80)}"
[ "$ui_w" -gt 78 ] && ui_w=78
[ "$ui_w" -lt 50 ] && ui_w=50

E_RST="" E_BOLD="" E_NB="" E_FG="" E_MU="" E_AC="" E_AC2="" E_BG="" E_BG2=""
E_OK="" E_WARN="" E_BAD="" E_BGOK="" E_BGWARN="" E_BGBAD="" E_INK=""

ui_fg() { local h; h="$(hex "$1")"; printf '\e[38;2;%d;%d;%dm' "0x${h:0:2}" "0x${h:2:2}" "0x${h:4:2}"; }
ui_bg() { local h; h="$(hex "$1")"; printf '\e[48;2;%d;%d;%dm' "0x${h:0:2}" "0x${h:2:2}" "0x${h:4:2}"; }

# Lee la paleta del tema activo; si no hay, usa Sakura nocturna.
ui_palette() {
  [ "$ui_on" = 1 ] || return 0
  local pb="#120b12" ps="#1d121d" ps2="#2a1a2a" pfg="#fbe9f2" pmu="#a88a9d" pac="#ff86b8" pac2="#ffc2a6" pmode="dark"
  local cur="" f
  [ -f "$state/current" ] && cur="$(cat "$state/current")"
  f="$themes/$cur/colors.json"
  if [ -n "$cur" ] && [ -f "$f" ] && jq -e '[.bg,.s,.s2,.fg,.mu,.ac,.ac2,.on] | all(test("^#[0-9a-fA-F]{6}$"))' "$f" > /dev/null 2>&1; then
    IFS=$'\t' read -r pb ps ps2 pfg pmu pac pac2 _ pmode < <(jq -r '[.bg,.s,.s2,.fg,.mu,.ac,.ac2,.on,(.mode // "dark")] | @tsv' "$f")
  fi
  local ok warn bad
  if [ "$pmode" = "light" ]; then ok="#1a7f50"; warn="#9a6700"; bad="#c92a3e"; else ok="#7fe3a8"; warn="#ffc66b"; bad="#ff6b81"; fi
  E_RST=$'\e[0m' E_BOLD=$'\e[1m' E_NB=$'\e[22m'
  E_FG="$(ui_fg "$pfg")" E_MU="$(ui_fg "$pmu")" E_AC="$(ui_fg "$pac")" E_AC2="$(ui_fg "$pac2")"
  E_BG="$(ui_bg "$ps")" E_BG2="$(ui_bg "$ps2")"
  E_OK="$(ui_fg "$ok")" E_WARN="$(ui_fg "$warn")" E_BAD="$(ui_fg "$bad")"
  E_BGOK="$(ui_bg "$ok")" E_BGWARN="$(ui_bg "$warn")" E_BGBAD="$(ui_bg "$bad")"
  E_INK="$(ui_fg "$pb")"
}
ui_palette

# Texto con color; al terminar vuelve al color base de la ventana.
ui_c() { printf '%s%s%s' "$1" "$2" "$E_FG"; }
# Largo visible: cuenta caracteres sin las secuencias de color.
ui_vlen() { printf '%s' "$1" | sed -E 's/\x1b\[[0-9;]*m//g' | wc -m | tr -d ' '; }
ui_rep() { local n="$1" ch="$2" i; for ((i = 0; i < n; i++)); do printf '%s' "$ch"; done; }
ui_trunc() { # ui_trunc texto máximo
  if [ "${#1}" -gt "$2" ]; then printf '%s…' "${1:0:$(($2 - 1))}"; else printf '%s' "$1"; fi
}

ui_open() { # ui_open "título"
  if [ "$ui_on" = 0 ]; then printf '\n%s\n' "$1"; return 0; fi
  local inner=$((ui_w - 2)) pad t
  t="$(ui_trunc "$1" $((inner - 10)))"
  pad=$((inner - 8 - ${#t}))
  printf '%s╭%s %s●%s ●%s ●%s  %s%s%s%s%s%s╮%s\n' \
    "$E_AC" "$E_BG2" "$E_AC" "$E_AC2" "$E_MU" "$E_FG" "$E_BOLD" "$t" "$E_NB" "$(ui_rep "$pad" ' ')" "$E_RST" "$E_AC" "$E_RST"
}
ui_line() { # ui_line "texto (puede llevar color)"
  if [ "$ui_on" = 0 ]; then printf '  %s\n' "$1"; return 0; fi
  local len pad
  len="$(ui_vlen "$1")"
  pad=$((ui_w - 4 - len))
  [ "$pad" -lt 0 ] && pad=0
  printf '%s│%s%s%s %s%s %s%s│%s\n' "$E_AC" "$E_RST" "$E_BG" "$E_FG" "$1" "$(ui_rep "$pad" ' ')" "$E_RST" "$E_AC" "$E_RST"
}
ui_close() {
  if [ "$ui_on" = 0 ]; then return 0; fi
  printf '%s╰%s╯%s\n' "$E_AC" "$(ui_rep $((ui_w - 2)) '─')" "$E_RST"
}
ui_section() { # título de bloque dentro de la ventana
  ui_line ""
  ui_line "${E_BOLD}$(ui_c "$E_AC2" "${1^^}")${E_NB}"
}
ui_row() { # ui_row ok|warn|bad|info "mensaje"
  local glyph col
  case "$1" in
    ok) glyph="✓"; col="$E_OK" ;;
    warn) glyph="!"; col="$E_WARN" ;;
    bad) glyph="✗"; col="$E_BAD" ;;
    *) glyph="·"; col="$E_MU" ;;
  esac
  ui_line " $(ui_c "$col" "$glyph")  $(ui_trunc "$2" $((ui_w - 9)))"
}
ui_kv() { # ui_kv clave valor
  ui_line " $(ui_c "$E_MU" "$(printf '%-12s' "$1")") $(ui_trunc "$2" $((ui_w - 19)))"
}
ui_swatch() { # ui_swatch #hex ...  → bloques de color
  local h
  for h in "$@"; do
    if [ "$ui_on" = 1 ]; then printf '%s██%s' "$(ui_fg "$h")" "$E_FG"; else printf '#'; fi
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
