# ── Componentes: segmentos, diff y errores ───────────────────────────
# Piezas reutilizables que cualquier comando puede pedir en lugar de dibujar la
# suya. Van dentro de una ventana (entre ui_open y ui_close). Las etiquetas
# aceptan «@clave» (ver i18n.sh).

# ── Segmentos (estilo de subrayado y puntos) ─────────────────────────
# ui_tabs ACTIVA etiqueta…       pestañas; la activa (desde 0) lleva subrayado
ui_tabs() {
  local act="$1" i=0 lbl ul top="" under=""
  shift
  for lbl in "$@"; do
    msg lbl "$lbl"
    ui_repv ul "${#lbl}" ' '
    if [ "$i" = "$act" ]; then
      ui_repv ul "${#lbl}" '▔'
      top+="${E_BOLD}${E_AC}${lbl}${E_FG}${E_NB}   "
      under+="${E_AC}${ul}${E_FG}   "
    else
      top+="${E_MU}${lbl}${E_FG}   "
      under+="${ul}   "
    fi
    i=$((i + 1))
  done
  ui_line " $top"
  ui_line " $under"
}

# ui_status dato…                barra de estado: dato • dato • dato
ui_status() {
  local out="" d first=1
  for d in "$@"; do
    if [ "$first" = 1 ]; then first=0; else out+=" ${E_AC}•${E_FG} "; fi
    out+="${E_MU}${d}${E_FG}"
  done
  ui_line " $out"
}

# ui_crumbs nivel…               migas de pan: maxor › temas › brasa (el último destaca)
ui_crumbs() {
  local out="" d i=0 n=$#
  for d in "$@"; do
    i=$((i + 1))
    msg d "$d"
    if [ "$i" = "$n" ]; then out+="${E_BOLD}${d}${E_NB}"; else out+="${E_MU}${d}${E_FG} ${E_AC}›${E_FG} "; fi
  done
  ui_line " $out"
}

# ui_hints tecla:acción…         ayuda de teclas fija al pie de una pantalla interactiva
ui_hints() {
  local out="" h k a
  for h in "$@"; do
    k="${h%%:*}"
    msg a "${h#*:}"
    out+="${E_AC}${k}${E_FG} ${a}   "
  done
  ui_line " $out"
}

# ── Diff de cambios (salida de `nix store diff-closures`) ────────────
# ui_diff "texto" [máximo de filas]
# Cuenta nuevos, actualizados y eliminados, oculta los archivos de configuración
# (unit-*, etc) y muestra cada paquete con su símbolo y sus versiones.
# Deja los totales en UI_DIFF_ADD, UI_DIFF_UPD, UI_DIFF_DEL y UI_DIFF_CFG.
UI_DIFF_ADD=0 UI_DIFF_UPD=0 UI_DIFF_DEL=0 UI_DIFF_CFG=0 UI_DIFF_CHG=0
ui_diff() {
  local text="${1//$'\e'\[*([0-9;])m/}" max="${2:-14}" line name rest ver size
  local -a upd=() add=() del=() chg=()
  UI_DIFF_ADD=0 UI_DIFF_UPD=0 UI_DIFF_DEL=0 UI_DIFF_CFG=0 UI_DIFF_CHG=0
  while IFS= read -r line; do
    [ -n "$line" ] || continue
    name="${line%%: *}"
    rest="${line#*: }"
    ver="$rest"; size=""
    if [[ "$rest" =~ ^(.*),\ ([+-]?[0-9.]+\ [KMGT]?i?B)$ ]]; then
      ver="${BASH_REMATCH[1]}"; size="${BASH_REMATCH[2]}"
    elif [[ "$rest" =~ ^[+-]?[0-9.]+\ [KMGT]?i?B$ ]]; then
      ver=""; size="$rest"
    fi
    if [[ "$ver" == *ε* ]]; then
      UI_DIFF_CFG=$((UI_DIFF_CFG + 1))
    elif [[ "$ver" == "∅ →"* ]]; then
      UI_DIFF_ADD=$((UI_DIFF_ADD + 1)); add+=("$name|${ver#∅ → }|$size")
    elif [[ "$ver" == *"→ ∅" ]]; then
      UI_DIFF_DEL=$((UI_DIFF_DEL + 1)); del+=("$name|${ver% → ∅}|$size")
    elif [[ "$ver" == *"→"* ]]; then
      UI_DIFF_UPD=$((UI_DIFF_UPD + 1)); upd+=("$name|$ver|$size")
    else
      UI_DIFF_CHG=$((UI_DIFF_CHG + 1)); chg+=("$name||$size")
    fi
  done <<< "$text"

  local sum l_new l_upd l_del l_chg chgtxt=""
  msg l_new @diff.new; msg l_upd @diff.updated; msg l_del @diff.removed; msg l_chg @diff.changed
  if [ "$UI_DIFF_CHG" -gt 0 ]; then chgtxt="   ${E_MU}~${UI_DIFF_CHG} ${l_chg}${E_FG}"; fi
  ui_line " ${E_OK}+${UI_DIFF_ADD}${E_FG} ${E_MU}${l_new}${E_FG}   ${E_WARN}↑${UI_DIFF_UPD}${E_FG} ${E_MU}${l_upd}${E_FG}   ${E_BAD}−${UI_DIFF_DEL}${E_FG} ${E_MU}${l_del}${E_FG}${chgtxt}"
  if [ "$UI_DIFF_CFG" -gt 0 ]; then
    msg sum @diff.config_files "$UI_DIFF_CFG"
    ui_line " ${E_MU}${sum}${E_FG}"
  fi
  ui_line ""

  local shown=0 total=$((UI_DIFF_ADD + UI_DIFF_UPD + UI_DIFF_DEL + UI_DIFF_CHG)) item n v s sym col nm
  local nu=${#upd[@]} na=${#add[@]} nd=${#del[@]}
  for item in "${upd[@]}" "${add[@]}" "${del[@]}" "${chg[@]}"; do
    [ -n "$item" ] || continue
    if [ "$shown" -ge "$max" ]; then break; fi
    IFS='|' read -r n v s <<< "$item"
    if [ "$shown" -lt "$nu" ]; then sym="↑"; col="$E_WARN"
    elif [ "$shown" -lt $((nu + na)) ]; then sym="+"; col="$E_OK"
    elif [ "$shown" -lt $((nu + na + nd)) ]; then sym="−"; col="$E_BAD"
    else sym="~"; col="$E_MU"; fi
    ui_truncv nm "$n" 26
    printf -v nm '%-26s' "$nm"
    ui_truncv v "$v" $((ui_w - 46))
    ui_split " ${col}${sym}${E_FG} ${nm} ${E_MU}${v}${E_FG}" "${E_MU}${s}${E_FG} "
    shown=$((shown + 1))
  done
  if [ "$total" -gt "$shown" ]; then
    msg sum @diff.more "$((total - shown))"
    ui_line " ${E_MU}${sum}${E_FG}"
  fi
  return 0
}

# ── Errores con causa, sugerencia y registro ─────────────────────────
# ui_error título [causa [qué probar [log]]]
# Cada texto admite «@clave». Va a stderr y se anota en el registro; con el cuarto
# argumento «log» añade la línea que apunta a `maxor logs --last`.
ui_error() {
  local title cause="" hint="" lbl txt rows=() i n glyph
  msg title "$1"
  if [ -n "${2:-}" ]; then msg cause "$2"; msg lbl @ui.cause; rows+=("$lbl|$cause|"); fi
  if [ -n "${3:-}" ]; then msg hint "$3"; msg lbl @ui.try; rows+=("$lbl|$hint|$E_AC"); fi
  if [ "${4:-}" = log ]; then msg lbl @ui.details; rows+=("$lbl|maxor logs --last|$E_AC"); fi
  log ERROR "$title${cause:+ · $cause}"
  {
    printf ' %s✗%s  %s%s%s\n' "$E_BAD" "$E_RST" "$E_BOLD" "$title" "$E_RST"
    n=${#rows[@]}
    for ((i = 0; i < n; i++)); do
      IFS='|' read -r lbl txt col <<< "${rows[i]}"
      glyph="├"; [ "$i" = $((n - 1)) ] && glyph="└"
      printf '    %s%s%s %s%-8s%s %s%s%s\n' "$E_MU" "$glyph" "$E_RST" "$E_MU" "$lbl" "$E_RST" "${col:-}" "$txt" "$E_RST"
    done
  } >&2
  return 0
}
