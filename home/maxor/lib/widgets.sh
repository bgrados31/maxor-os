# ── Componentes: ayuda de teclas, diff y errores ─────────────────────
# Piezas reutilizables que cualquier comando puede pedir en lugar de dibujar la
# suya. Van dentro del riel (entre ui_intro y ui_outro). Las etiquetas aceptan
# «@clave» (ver i18n.sh).

# ── Ayuda de teclas ──────────────────────────────────────────────────
# ui_hints tecla:acción…         ayuda de teclas fija al pie de una pantalla interactiva
ui_hints() {
  local out="" h k a
  for h in "$@"; do
    k="${h%%:*}"
    msg a "${h#*:}"
    out+="${E_AC}${k}${E_RST} ${a}   "
  done
  ui_text " $out"
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
  if [ "$UI_DIFF_CHG" -gt 0 ]; then chgtxt="   ${E_MU}${G_CHG}${UI_DIFF_CHG} ${l_chg}${E_RST}"; fi
  ui_text " ${E_OK}${G_ADD}${UI_DIFF_ADD}${E_RST} ${E_MU}${l_new}${E_RST}   ${E_WARN}${G_UP}${UI_DIFF_UPD}${E_RST} ${E_MU}${l_upd}${E_RST}   ${E_BAD}${G_DEL}${UI_DIFF_DEL}${E_RST} ${E_MU}${l_del}${E_RST}${chgtxt}"
  if [ "$UI_DIFF_CFG" -gt 0 ]; then
    msg sum @diff.config_files "$UI_DIFF_CFG"
    ui_text " ${E_MU}${sum}${E_RST}"
  fi
  ui_text ""

  local shown=0 total=$((UI_DIFF_ADD + UI_DIFF_UPD + UI_DIFF_DEL + UI_DIFF_CHG)) item n v s sym col nm
  local nu=${#upd[@]} na=${#add[@]} nd=${#del[@]}
  for item in "${upd[@]}" "${add[@]}" "${del[@]}" "${chg[@]}"; do
    [ -n "$item" ] || continue
    if [ "$shown" -ge "$max" ]; then break; fi
    IFS='|' read -r n v s <<< "$item"
    if [ "$shown" -lt "$nu" ]; then sym="$G_UP"; col="$E_WARN"
    elif [ "$shown" -lt $((nu + na)) ]; then sym="$G_ADD"; col="$E_OK"
    elif [ "$shown" -lt $((nu + na + nd)) ]; then sym="$G_DEL"; col="$E_BAD"
    else sym="$G_CHG"; col="$E_MU"; fi
    ui_truncv nm "$n" 26
    printf -v nm '%-26s' "$nm"
    ui_truncv v "$v" $((ui_w - 46))
    ui_split " ${col}${sym}${E_RST} ${nm} ${E_MU}${v}${E_RST}" "${E_MU}${s}${E_RST} "
    shown=$((shown + 1))
  done
  if [ "$total" -gt "$shown" ]; then
    msg sum @diff.more "$((total - shown))"
    ui_text " ${E_MU}${sum}${E_RST}"
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
    printf '%s%s%s  %s%s%s\n' "$E_BAD" "$G_BAD" "$E_RST" "$E_BOLD" "$title" "$E_RST"
    n=${#rows[@]}
    for ((i = 0; i < n; i++)); do
      IFS='|' read -r lbl txt col <<< "${rows[i]}"
      glyph="$G_TEE"; [ "$i" = $((n - 1)) ] && glyph="$G_END"
      printf '   %s%s%s %s%-8s%s %s%s%s\n' "$E_MU" "$glyph" "$E_RST" "$E_MU" "$lbl" "$E_RST" "${col:-}" "$txt" "$E_RST"
    done
  } >&2
  return 0
}
