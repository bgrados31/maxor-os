# ── Cargadores: esperas con estilo ───────────────────────────────────
# Cada espera tiene su cargador:
#
#   ui_run           una tarea corta, con spinner
#   ui_run_tail      una tarea larga: spinner y las últimas líneas de su salida
#   ui_pipeline      varias tareas en secuencia, con su lista de pasos
#   ui_progress_*    avance conocido: barra con porcentaje
#   ui_skeleton_*    huecos que parpadean mientras llegan los datos
#
# Todos animan solo en una terminal con color; en cualquier otro caso (tubería,
# NO_COLOR, --quiet) ejecutan igual y escriben líneas simples. Un fallo se
# anota en el registro (`maxor logs --last`) y muestra la causa y qué probar.
UI_OUT=""
UI_SPIN=(⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏)
MAXOR_QUIET="${MAXOR_QUIET:-0}"

ui_anim() { [ "$ui_on" = 1 ] && [ -t 1 ] && [ "$MAXOR_QUIET" != 1 ]; }

# Línea plana (sin borde) con texto a ambos lados.
ui_flat_split() { # ui_flat_split "izquierda" "derecha"
  local l r pad
  ui_len "$1"; l=$UI_LEN
  ui_len "$2"; r=$UI_LEN
  ui_repv pad $((ui_w - 2 - l - r < 1 ? 1 : ui_w - 2 - l - r)) ' '
  printf ' %s%s%s\n' "$1" "$pad" "$2"
}

# Falla de un comando: al registro, y en pantalla causa + qué probar.
#   ui_run_fail "etiqueta" código archivo_con_el_error "comando"
ui_run_fail() {
  local label="$1" rc="$2" errf="$3" cmd="$4" last line shown
  log ERROR "$label (exit $rc): $cmd"
  mkdir -p "$logdir" 2> /dev/null || true
  {
    printf '%(%F %T)T  %s\n$ %s\nexit %s\n\n' -1 "$label" "$cmd" "$rc"
    cat "$errf"
  } > "$logdir/last-error.log" 2> /dev/null || true
  tail -n 40 "$errf" 2> /dev/null | while IFS= read -r line; do log ERROR "  | $line"; done
  ui_say bad "$label"
  last="$(tail -n 6 "$errf" 2> /dev/null | tr -d '\r')"
  if [ -n "$last" ]; then
    while IFS= read -r line; do
      ui_truncv shown "$line" $((ui_w - 8))
      printf '    %s│%s %s\n' "$E_MU" "$E_RST" "$shown"
    done <<< "$last"
  fi
  ui_say info @ui.log_hint
}

# ui_run mensaje|@clave comando args…   → stdout del comando en $UI_OUT
ui_run() {
  local label out errf rc=0 pid i=0
  msg label "$1"
  shift
  out="$(mktemp)"; errf="$(mktemp)"
  if ui_anim; then
    "$@" > "$out" 2> "$errf" &
    pid=$!
    printf '\e[?25l'
    while kill -0 "$pid" 2> /dev/null; do
      printf '\r %s%s%s  %s' "$E_AC" "${UI_SPIN[i % 10]}" "$E_RST" "$label"
      i=$((i + 1))
      sleep 0.1
    done
    printf '\r\e[2K\e[?25h'
    wait "$pid" || rc=$?
  else
    "$@" > "$out" 2> "$errf" || rc=$?
  fi
  UI_OUT="$(cat "$out")"
  if [ "$rc" = 0 ]; then
    if [ "$MAXOR_QUIET" != 1 ]; then ui_say ok "$label"; fi
  else
    ui_run_fail "$label" "$rc" "$errf" "$*"
  fi
  rm -f "$out" "$errf"
  return "$rc"
}

# ui_run_tail mensaje|@clave comando args…
# Como ui_run, pero muestra las últimas 4 líneas de lo que el proceso imprime
# (en gris, bajo el spinner): se ve que avanza sin inundar la terminal.
ui_run_tail() {
  local label out errf rc=0 pid i=0 frame prev=0 line tl
  msg label "$1"
  shift
  out="$(mktemp)"; errf="$(mktemp)"
  if ui_anim; then
    "$@" > "$out" 2> "$errf" &
    pid=$!
    printf '\e[?25l'
    while kill -0 "$pid" 2> /dev/null; do
      frame=" ${E_AC}${UI_SPIN[i % 10]}${E_RST}  $label"
      tl="$(tail -n 4 "$errf" 2> /dev/null | tr -d '\r')"
      while IFS= read -r line; do
        [ -n "$line" ] || continue
        ui_truncv line "$line" $((ui_w - 8))
        frame+=$'\n'"    ${E_MU}│ ${line}${E_RST}"
      done <<< "$tl"
      ui_paint "$frame" "$prev"
      ui_count_lines "$frame"; prev=$UI_LINES
      i=$((i + 1))
      sleep 0.1
    done
    if [ "$prev" -gt 0 ]; then printf '\e[%dA\e[J' "$prev"; fi
    printf '\e[?25h'
    wait "$pid" || rc=$?
  else
    "$@" > "$out" 2> "$errf" || rc=$?
  fi
  UI_OUT="$(cat "$out")"
  if [ "$rc" = 0 ]; then
    if [ "$MAXOR_QUIET" != 1 ]; then ui_say ok "$label"; fi
  else
    ui_run_fail "$label" "$rc" "$errf" "$*"
  fi
  rm -f "$out" "$errf"
  return "$rc"
}

# ── Pasos en secuencia ───────────────────────────────────────────────
# ui_pipeline título|@clave  paso1 función1  [paso2 función2 …]
# Cada paso es una función; se ejecutan una tras otra y se detiene en la
# primera que falla. Lo que cada una imprime queda en UI_OUTS[0], UI_OUTS[1]…
# Las funciones corren en un subproceso: no pueden cambiar variables del padre.
UI_OUTS=()
UI_PL_LABELS=()
UI_PL_ST=()
UI_PL_ERR=""

ui_pipeline_frame() { # título spin actual total segundos estado
  local title="$1" spin="$2" cur="$3" n="$4" secs="$5" state="$6" right i g tl line shown
  case "$state" in
    done) msg right @ui.done_in "$secs" ;;
    failed) msg right @ui.failed ;;
    *) msg right @ui.step_of "$((cur + 1))" "$n" ;;
  esac
  ui_flat_split "${E_BOLD}${title}${E_NB}" "${E_MU}${right}${E_RST}"
  for ((i = 0; i < n; i++)); do
    case "${UI_PL_ST[i]}" in
      ok) g="${E_OK}✓${E_RST}  ${UI_PL_LABELS[i]}" ;;
      run) g="${E_AC}${UI_SPIN[spin % 10]}${E_RST}  ${E_BOLD}${UI_PL_LABELS[i]}${E_NB}" ;;
      bad) g="${E_BAD}✗${E_RST}  ${UI_PL_LABELS[i]}" ;;
      *) g="${E_MU}○  ${UI_PL_LABELS[i]}${E_RST}" ;;
    esac
    printf ' %s\n' "$g"
    # bajo el paso en curso, las últimas líneas de lo que imprime (en gris)
    if [ "${UI_PL_ST[i]}" = run ] && [ -n "${UI_PL_ERR:-}" ]; then
      tl="$(tail -n 3 "$UI_PL_ERR" 2> /dev/null | tr -d '\r')"
      while IFS= read -r line; do
        [ -n "$line" ] || continue
        ui_truncv shown "$line" $((ui_w - 9))
        printf '    %s│ %s%s\n' "$E_MU" "$shown" "$E_RST"
      done <<< "$tl"
    fi
  done
}

ui_pipeline() {
  local title l fn i n rc=0 pid k frame prev=0 tmp start=$SECONDS anim=0
  msg title "$1"
  shift
  UI_PL_LABELS=(); UI_PL_ST=(); UI_OUTS=()
  local fns=()
  while [ $# -ge 2 ]; do
    msg l "$1"
    UI_PL_LABELS+=("$l"); fns+=("$2"); UI_PL_ST+=(wait)
    shift 2
  done
  n=${#fns[@]}
  tmp="$(mktemp -d)"
  if ui_anim; then anim=1; printf '\e[?25l'; fi
  UI_PL_ERR="$tmp/err"
  for ((i = 0; i < n; i++)); do
    UI_PL_ST[i]=run
    fn="${fns[i]}"
    if [ "$anim" = 1 ]; then
      "$fn" > "$tmp/out" 2> "$tmp/err" &
      pid=$!
      k=0
      while kill -0 "$pid" 2> /dev/null; do
        frame="$(ui_pipeline_frame "$title" "$k" "$i" "$n" $((SECONDS - start)) running)"
        ui_paint "$frame" "$prev"
        ui_count_lines "$frame"; prev=$UI_LINES
        k=$((k + 1))
        sleep 0.1
      done
      wait "$pid" || rc=$?
    else
      "$fn" > "$tmp/out" 2> "$tmp/err" || rc=$?
    fi
    UI_OUTS[i]="$(cat "$tmp/out")"
    if [ "$rc" != 0 ]; then
      UI_PL_ST[i]=bad
      if [ "$anim" = 1 ]; then
        frame="$(ui_pipeline_frame "$title" 0 "$i" "$n" $((SECONDS - start)) failed)"
        ui_paint "$frame" "$prev"
        printf '\e[?25h'
      fi
      ui_run_fail "${UI_PL_LABELS[i]}" "$rc" "$tmp/err" "$fn"
      rm -rf "$tmp"
      return "$rc"
    fi
    UI_PL_ST[i]=ok
    if [ "$anim" = 0 ] && [ "$MAXOR_QUIET" != 1 ]; then ui_say ok "${UI_PL_LABELS[i]}"; fi
  done
  if [ "$anim" = 1 ]; then
    frame="$(ui_pipeline_frame "$title" 0 "$n" "$n" $((SECONDS - start)) "done")"
    ui_paint "$frame" "$prev"
    printf '\e[?25h'
  fi
  rm -rf "$tmp"
  return 0
}

# ── Barra de progreso ────────────────────────────────────────────────
# ui_progress_begin TOTAL mensaje|@clave [args…]
# ui_progress_set N [etiqueta]
# ui_progress_end
UI_PG_TOTAL=1
UI_PG_LABEL=""

ui_bar() { # ui_bar variable porcentaje(0-100) [ancho]
  local _v="$1" pct="$2" w="${3:-24}" n out="" k
  n=$((pct * w / 100))
  for ((k = 0; k < w; k++)); do
    if [ "$k" -lt "$n" ]; then out+="${E_AC}▰"; else out+="${E_MU}▱"; fi
  done
  printf -v "$_v" '%s%s' "$out" "$E_RST"
}
ui_progress_begin() {
  UI_PG_TOTAL="$1"
  shift
  msg UI_PG_LABEL "$@"
  [ "$UI_PG_TOTAL" -gt 0 ] || UI_PG_TOTAL=1
  ui_progress_set 0
}
ui_progress_set() { # ui_progress_set n [texto]
  local n="$1" pct bar
  pct=$((n * 100 / UI_PG_TOTAL))
  [ "$pct" -gt 100 ] && pct=100
  if ui_anim; then
    ui_bar bar "$pct"
    printf '\r\e[2K %s  %s%3d%%%s  %s%s%s' "$bar" "$E_BOLD" "$pct" "$E_NB" "$UI_PG_LABEL" "${2:+ · }" "${2:-}"
  fi
  return 0
}
ui_progress_end() {
  if ui_anim; then printf '\r\e[2K'; fi
  if [ "$MAXOR_QUIET" != 1 ]; then ui_say ok "$UI_PG_LABEL"; fi
  return 0
}

# ── Esqueleto ────────────────────────────────────────────────────────
# Huecos que parpadean mientras llegan los datos: la lista aparece enseguida y
# la pantalla no salta cuando llegan. Se usa con ui_paint para sustituirlo.
#   ui_skeleton_frame título|@clave filas paso
ui_skeleton_frame() {
  local rows="$2" step="$3" r ch w1 w2 w3 a b c
  ch="░"; [ $((step % 6)) -ge 3 ] && ch="▒"
  ui_open "$1"
  ui_line ""
  for ((r = 0; r < rows; r++)); do
    w1=$((6 + (r * 3) % 4)); w2=$((16 + (r * 7) % 12)); w3=$((5 + r % 3))
    ui_repv a "$w1" "$ch"; ui_repv b "$w2" "$ch"; ui_repv c "$w3" "$ch"
    ui_line " ${E_MU}${a}  ${b}  ${c}${E_FG}"
  done
  ui_line ""
  ui_close
}
