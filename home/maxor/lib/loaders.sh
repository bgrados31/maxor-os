# ── Cargadores: spinner y ejecución de tareas largas ─────────────────
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
