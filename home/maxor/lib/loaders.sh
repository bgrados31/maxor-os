# ── Cargador: una sola forma de esperar ──────────────────────────────
# Toda espera de la CLI usa el mismo cargador, y la pantalla completa
# (maxor-tui) habla el mismo idioma: mismo spinner, mismos estados, mismos
# tiempos. Así no hay cargadores distintos según el comando.
#
#   ⠹  Building nitro  12s           trabajando (spinner, y los segundos si pasan de 3)
#   │  copying path …/firefox-150    lo último que imprime, si tarda más de 2 s
#   ◇  Built nitro  14s              hecho (con el tiempo si pasó de 2 s)
#   ✗  Building nitro                falló: causa, y `maxor logs --last`
#
# ui_run     una tarea; su salida queda en UI_OUT
# ui_progress_*   lo mismo cuando se conoce el avance: añade la barra con porcentaje
#
# Reglas: nada aparece en los primeros 150 ms (las tareas rápidas no parpadean);
# fuera de una terminal, con NO_COLOR o con --quiet no hay animación: se escribe
# solo el resultado. Un fallo se anota en el registro (`maxor logs --last`).
# shellcheck disable=SC2034 # lo leen los comandos que llaman a ui_run
UI_OUT=""
MAXOR_QUIET="${MAXOR_QUIET:-0}"

ui_anim() { [ "$ui_on" = 1 ] && [ -t 1 ] && [ "$MAXOR_QUIET" != 1 ]; }

# Falla de una tarea: al registro, y en pantalla las últimas líneas y dónde mirar.
#   ui_run_fail "etiqueta" código archivo_con_el_error "comando" [singap]
# (con «singap» la línea de riel previa ya se imprimió: lo hace el spinner)
ui_run_fail() {
  local label="$1" rc="$2" errf="$3" cmd="$4" last line shown hint
  log ERROR "$label (exit $rc): $cmd"
  mkdir -p "$logdir" 2> /dev/null || true
  {
    printf '%(%F %T)T  %s\n$ %s\nexit %s\n\n' -1 "$label" "$cmd" "$rc"
    cat "$errf"
  } > "$logdir/last-error.log" 2> /dev/null || true
  tail -n 40 "$errf" 2> /dev/null | while IFS= read -r line; do log ERROR "  | $line"; done
  if [ "$UI_RAIL" = 1 ] && [ "${5:-}" != singap ]; then ui_rail; fi
  ui_stepline bad "$label"
  last="$(tail -n 6 "$errf" 2> /dev/null | tr -d '\r')"
  if [ -n "$last" ]; then
    while IFS= read -r line; do
      ui_truncv shown "$line" $((ui_w - 8))
      ui_text " ${E_MU}${shown}${E_RST}"
    done <<< "$last"
  fi
  msg hint @ui.log_hint
  ui_text " ${E_MU}${hint}${E_RST}"
}

# ui_run mensaje|@clave comando args…   → stdout del comando en $UI_OUT
ui_run() {
  local label out errf rc=0 pid i=0 t0=$SECONDS el frame prev=0 line tl shown detail=""
  msg label "$1"
  shift
  out="$(mktemp)"; errf="$(mktemp)"
  if ui_anim; then
    "$@" > "$out" 2> "$errf" &
    pid=$!
    printf '\e[?25l'
    if [ "$UI_RAIL" = 1 ]; then ui_rail; fi
    while kill -0 "$pid" 2> /dev/null; do
      if [ "$i" -ge 2 ]; then # 150 ms sin dibujar nada: las tareas rápidas no parpadean
        el=$((SECONDS - t0))
        frame="${E_AC}${UI_SPIN[i % ${#UI_SPIN[@]}]}${E_RST}  $label"
        if [ "$el" -ge 3 ]; then frame+="  ${E_MU}${el}s${E_RST}"; fi
        if [ "$el" -ge 2 ]; then
          tl="$(tail -n 2 "$errf" 2> /dev/null | tr -d '\r')"
          while IFS= read -r line; do
            [ -n "$line" ] || continue
            ui_truncv shown "$line" $((ui_w - 8))
            frame+=$'\n'"${E_MU}${G_BAR}${E_RST}  ${E_MU}${shown}${E_RST}"
          done <<< "$tl"
        fi
        ui_paint "$frame" "$prev"
        ui_count_lines "$frame"; prev=$UI_LINES
      fi
      i=$((i + 1))
      sleep 0.08
    done
    if [ "$prev" -gt 0 ]; then printf '\e[%dA\e[J' "$prev"; fi
    printf '\e[?25h'
    wait "$pid" || rc=$?
    el=$((SECONDS - t0))
    [ "$el" -ge 2 ] && detail="${el}s"
    if [ "$rc" = 0 ]; then
      if [ "$MAXOR_QUIET" != 1 ]; then ui_stepline ok "$label" "$detail"; fi
    fi
  else
    "$@" > "$out" 2> "$errf" || rc=$?
    if [ "$rc" = 0 ] && [ "$MAXOR_QUIET" != 1 ]; then
      if [ "$UI_RAIL" = 1 ]; then ui_rail; fi
      ui_stepline ok "$label"
    fi
  fi
  UI_OUT="$(cat "$out")"
  if [ "$rc" != 0 ]; then
    if ui_anim; then ui_run_fail "$label" "$rc" "$errf" "$*" singap; else ui_run_fail "$label" "$rc" "$errf" "$*"; fi
  fi
  rm -f "$out" "$errf"
  return "$rc"
}

# ── Avance conocido ──────────────────────────────────────────────────
# ui_progress_begin TOTAL mensaje|@clave [args…]
# ui_progress_set N [texto]
# ui_progress_end
UI_PG_TOTAL=1
UI_PG_LABEL=""

ui_bar() { # ui_bar variable porcentaje(0-100) [ancho]
  local _v="$1" pct="$2" w="${3:-24}" n out="" k
  n=$((pct * w / 100))
  for ((k = 0; k < w; k++)); do
    if [ "$k" -lt "$n" ]; then out+="${E_AC}${G_BARON}"; else out+="${E_MU}${G_BAROFF}"; fi
  done
  printf -v "$_v" '%s%s' "$out" "$E_RST"
}
ui_progress_begin() {
  UI_PG_TOTAL="$1"
  shift
  msg UI_PG_LABEL "$@"
  [ "$UI_PG_TOTAL" -gt 0 ] || UI_PG_TOTAL=1
  if ui_anim && [ "$UI_RAIL" = 1 ]; then ui_rail; fi
  ui_progress_set 0
}
ui_progress_set() { # ui_progress_set n [texto]
  local n="$1" pct bar
  pct=$((n * 100 / UI_PG_TOTAL))
  [ "$pct" -gt 100 ] && pct=100
  if ui_anim; then
    ui_bar bar "$pct"
    printf '\r\e[2K%s%s%s  %s  %s%3d%%%s  %s%s%s' "$E_AC" "${UI_SPIN[n % ${#UI_SPIN[@]}]}" "$E_RST" "$bar" "$E_BOLD" "$pct" "$E_NB" "$UI_PG_LABEL" "${2:+ · }" "${2:-}"
  fi
  return 0
}
ui_progress_end() {
  if ui_anim; then printf '\r\e[2K'; fi
  if [ "$MAXOR_QUIET" != 1 ]; then
    if ! ui_anim && [ "$UI_RAIL" = 1 ]; then ui_rail; fi
    ui_stepline ok "$UI_PG_LABEL"
  fi
  return 0
}
