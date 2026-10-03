# ── Apps: buscar, instalar, quitar y actualizar ──────────────────────
# Dos orígenes, ninguno pide root ni toca archivos .nix:
#   nix      paquetes de nixpkgs en el perfil del usuario (nix profile)
#   flatpak  apps de Flathub, instaladas para el usuario (flatpak --user)
# Los comandos que listan aceptan --json: es el contrato que usará Maxor Store.

flathub_url="https://dl.flathub.org/repo/flathub.flatpakrepo"

# Los paquetes con licencia no libre (Steam, Spotify…) exigen esto en nix.
app_nix() { NIXPKGS_ALLOW_UNFREE=1 nix "$@" --impure; }

# Flatpak imprime avisos (p. ej. XDG_DATA_DIRS) que aquí solo estorban.
app_flatpak_remote() {
  flatpak remote-add --user --if-not-exists flathub "$flathub_url" > /dev/null 2>&1 || true
}

# ID de Flatpak = dominio inverso con al menos tres partes (org.mozilla.firefox).
app_is_flatpak_id() { [[ "$1" =~ ^[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+){2,}$ ]]; }

# app_step json mensaje comando…  → UI_OUT con la salida; en JSON no pinta nada.
app_step() {
  local json="$1" msg="$2"
  shift 2
  if [ "$json" = 1 ]; then
    UI_OUT="$("$@" 2> /dev/null)" || return 1
  else
    ui_run "$msg" "$@"
  fi
}

# ── Búsqueda ─────────────────────────────────────────────────────────

# Relevancia de un resultado: 0 exacto, 1 empieza por, 2 contiene, 3 solo en la descripción.
app_rank='def rank($q): ($q | ascii_downcase) as $x | (.id | split(".") | last | ascii_downcase) as $i | ((.name // "") | ascii_downcase) as $n | if $i == $x or $n == $x then 0 elif ($i | startswith($x)) or ($n | startswith($x)) then 1 elif ($i | contains($x)) or ($n | contains($x)) then 2 else 3 end; '

# Barra de progreso: indeterminada (un pulso que corre) o llena si ya terminó.
app_bar() { # app_bar paso terminado
  local i="$1" done_="$2" w=18 k out=""
  for ((k = 0; k < w; k++)); do
    if [ "$done_" = 1 ]; then
      out+="$E_OK▰"
    elif (((k - i % (w + 4) + w + 4) % (w + 4) < 4)); then
      out+="$E_AC▰"
    else
      out+="$E_MU▱"
    fi
  done
  printf '%s%s' "$out" "$E_FG"
}

app_search_frame() { # consulta paso listo_nix listo_flatpak
  local q="$1" i="$2" d1="$3" d2="$4" st1 st2
  if [ "$d1" = 1 ]; then st1="$(ui_c "$E_OK" "listo")"; else st1="$(ui_c "$E_MU" "buscando…")"; fi
  if [ "$d2" = 1 ]; then st2="$(ui_c "$E_OK" "listo")"; else st2="$(ui_c "$E_MU" "buscando…")"; fi
  ui_open "maxor · buscar"
  ui_line ""
  ui_line " $(ui_c "$E_AC" "⌕")  ${E_BOLD}${q}${E_NB}$(ui_c "$E_AC" "▏")"
  ui_line ""
  ui_line " $(ui_c "$E_MU" "nixpkgs")  $(app_bar "$i" "$d1")  $st1"
  ui_line " $(ui_c "$E_MU" "flathub")  $(app_bar "$i" "$d2")  $st2"
  ui_line ""
  ui_close
}

# Repinta un marco en su sitio: sube tantas líneas como tenía el anterior.
app_paint() { # app_paint "marco" líneas_previas
  if [ "$2" -gt 0 ]; then printf '\e[%dA\e[J' "$2"; fi
  printf '%s\n' "$1"
}

# Busca en los dos orígenes a la vez. Deja en SEARCH_ALL un JSON ordenado por
# relevancia (con "score" para quien lo necesite).
app_search_fetch() { # app_search_fetch animar palabras…
  local anim="$1"
  shift
  local t p1 p2 i=0 d1 d2 frame prev=0
  t="$(mktemp -d)"
  (nix search nixpkgs --json "$@" > "$t/nix" 2> /dev/null) &
  p1=$!
  (
    app_flatpak_remote
    flatpak search --columns=application,name,description "$*" > "$t/fp" 2> /dev/null
  ) &
  p2=$!
  if [ "$anim" = 1 ]; then
    printf '\e[?25l'
    while kill -0 "$p1" 2> /dev/null || kill -0 "$p2" 2> /dev/null; do
      d1=0; d2=0
      kill -0 "$p1" 2> /dev/null || d1=1
      kill -0 "$p2" 2> /dev/null || d2=1
      frame="$(app_search_frame "$*" "$i" "$d1" "$d2")"
      app_paint "$frame" "$prev"
      prev=$(($(wc -l <<< "$frame") + 1))
      i=$((i + 1))
      sleep 0.08
    done
    # borra la barra: la lista de resultados ocupa su lugar
    printf '\e[%dA\e[J\e[?25h' "$prev"
  fi
  wait "$p1" "$p2" 2> /dev/null || true

  local nix_res="[]" fp_res="[]" q="$*"
  if [ -s "$t/nix" ]; then
    # Solo paquetes de primer nivel: tests.*, haskellPackages.*, python3xxPackages.*
    # son bibliotecas y piezas internas, no apps que alguien quiera instalar.
    nix_res="$(jq -c "$app_rank"'[to_entries[] | select((.key | sub("^legacyPackages\\.[^.]+\\."; "")) | contains(".") | not) | {
        source: "nix",
        id: (.key | sub("^legacyPackages\\.[^.]+\\."; "")),
        name: .value.pname,
        version: .value.version,
        description: (.value.description // "")
      }] | map(. + {score: rank($q)})' --arg q "$q" "$t/nix" 2> /dev/null || echo '[]')"
  fi
  if [ -s "$t/fp" ]; then
    fp_res="$(jq -R -s -c "$app_rank"'[split("\n")[] | select(length > 0) | split("\t") | select(length >= 2) | {
        source: "flatpak",
        id: .[0],
        name: .[1],
        version: "",
        description: (.[2] // "")
      }] | unique_by(.id) | map(. + {score: rank($q)})' --arg q "$q" "$t/fp" 2> /dev/null || echo '[]')"
  fi
  rm -rf "$t"
  # Una sola lista por relevancia. Lo que solo coincide en la descripción se
  # queda fuera salvo que haya muy pocos resultados mejores.
  SEARCH_ALL="$(jq -c -n --argjson a "$nix_res" --argjson b "$fp_res" '
    ($a + $b) | sort_by(.score, (if .source == "nix" then 0 else 1 end)) as $all
    | ($all | map(select(.score < 3))) as $g
    | ($all | map(select(.score >= 3))) as $r
    | ($g + $r[:([3 - ($g | length), 0] | max)]) | .[:14]')"
}

# Ids ya instalados con maxor, uno por línea ("nix:btop", "flatpak:com.x.Y").
app_installed_ids() {
  app_list_json | jq -r '.[] | .source + ":" + .id'
}

# Marco de la lista de resultados.
#   app_pick_frame consulta cursor desde por_vista interactivo
app_pick_frame() {
  local q="$1" cur="$2" from="$3" view="$4" inter="$5"
  local n="${#P_ID[@]}" marked=0 k to up down
  for k in "${!P_SEL[@]}"; do [ "${P_SEL[k]}" = 1 ] && marked=$((marked + 1)); done
  to=$((from + view))
  [ "$to" -gt "$n" ] && to="$n"
  up=$from
  down=$((n - to))

  ui_open "maxor · buscar"
  ui_line ""
  local info="$n resultados"
  [ "$inter" = 1 ] && [ "$marked" -gt 0 ] && info="$n resultados · $marked marcada(s)"
  ui_split " $(ui_c "$E_AC" "⌕")  ${E_BOLD}${q}${E_NB}" "$(ui_c "$E_MU" "$info") "
  if [ "$inter" = 1 ] && [ "$up" -gt 0 ]; then ui_line " $(ui_c "$E_MU" "   ↑ $up más arriba")"; else ui_line ""; fi

  local tag mark box name idp desc room
  for ((k = from; k < to; k++)); do
    # cursor y casilla
    if [ "$inter" = 1 ] && [ "$k" = "$cur" ]; then mark="$(ui_c "$E_AC" "❯")"; else mark=" "; fi
    if [ "${P_INST[k]}" = 1 ]; then
      box="$(ui_c "$E_OK" "✓")"
    elif [ "${P_SEL[k]}" = 1 ]; then
      box="$(ui_c "$E_AC" "◼")"
    elif [ "$inter" = 1 ]; then
      box="$(ui_c "$E_MU" "◻")"
    else
      box="$(ui_c "$E_MU" "·")"
    fi
    # origen a la derecha
    if [ "${P_INST[k]}" = 1 ]; then
      tag="$(ui_c "$E_OK" "instalada")"
    elif [ "${P_SRC[k]}" = nix ]; then
      tag="$(ui_c "$E_AC2" "nixpkgs")"
    else
      tag="$(ui_c "$E_AC" "flathub")"
    fi
    name="$(ui_trunc "${P_NAME[k]}" $((ui_w - 24)))"
    if [ "$inter" = 1 ] && [ "$k" = "$cur" ]; then name="${E_BOLD}${name}${E_NB}"; fi
    ui_split " $mark $box  $name" "$tag "
    # segunda línea: el id, bien visible, y la descripción
    idp="$(ui_trunc "${P_ID[k]}" 38)"
    room=$((ui_w - 15 - ${#idp}))
    [ "$room" -lt 0 ] && room=0
    desc="$(ui_trunc "${P_DESC[k]}" "$room")"
    ui_line "      $(ui_c "$E_MU" "id") $(ui_c "$E_AC" "$idp")  $(ui_c "$E_MU" "$desc")"
  done

  if [ "$inter" = 1 ] && [ "$down" -gt 0 ]; then ui_line " $(ui_c "$E_MU" "   ↓ $down más abajo")"; else ui_line ""; fi
  if [ "$inter" = 1 ]; then
    ui_line " $(ui_c "$E_AC" "↑↓") mover   $(ui_c "$E_AC" "espacio") marcar   $(ui_c "$E_AC" "⏎") instalar   $(ui_c "$E_AC" "q") salir"
  else
    ui_line " $(ui_c "$E_MU" "Instalar con el id:") $(ui_c "$E_AC" "maxor install <id>")"
  fi
  ui_close
}

# Lista interactiva: flechas, espacio para marcar, Intro para instalar.
# Deja en PICKED los elementos elegidos como "origen:id".
app_pick() { # app_pick consulta
  local q="$1" n="${#P_ID[@]}" cur=0 from=0 view lines frame prev=0 key k2 i any
  lines="$(tput lines 2> /dev/null || echo 24)"
  view=$(((lines - 10) / 2))
  [ "$view" -lt 3 ] && view=3
  [ "$view" -gt 7 ] && view=7
  [ "$view" -gt "$n" ] && view="$n"
  PICKED=()

  printf '\e[?25l'
  trap 'printf "\e[?25h"; exit 130' INT
  while :; do
    frame="$(app_pick_frame "$q" "$cur" "$from" "$view" 1)"
    app_paint "$frame" "$prev"
    prev=$(($(wc -l <<< "$frame") + 1))
    IFS= read -rsn1 key || break
    case "$key" in
      $'\e')
        k2=""
        read -rsn2 -t 0.05 k2 || true
        case "$k2" in
          '[A') [ "$cur" -gt 0 ] && cur=$((cur - 1)) ;;
          '[B') [ "$cur" -lt $((n - 1)) ] && cur=$((cur + 1)) ;;
          "") break ;; # Esc solo
        esac
        ;;
      k) [ "$cur" -gt 0 ] && cur=$((cur - 1)) ;;
      j) [ "$cur" -lt $((n - 1)) ] && cur=$((cur + 1)) ;;
      ' ')
        if [ "${P_INST[cur]}" != 1 ]; then
          if [ "${P_SEL[cur]}" = 1 ]; then P_SEL[cur]=0; else P_SEL[cur]=1; fi
        fi
        ;;
      "")
        # Intro: instala lo marcado; si no hay nada marcado, lo resaltado.
        any=0
        for i in "${!P_SEL[@]}"; do [ "${P_SEL[i]}" = 1 ] && any=1; done
        if [ "$any" = 0 ] && [ "${P_INST[cur]}" != 1 ]; then P_SEL[cur]=1; fi
        for i in "${!P_SEL[@]}"; do
          [ "${P_SEL[i]}" = 1 ] && PICKED+=("${P_SRC[i]}:${P_ID[i]}")
        done
        break
        ;;
      q | Q) break ;;
    esac
    [ "$cur" -lt "$from" ] && from=$cur
    [ "$cur" -ge $((from + view)) ] && from=$((cur - view + 1))
  done
  printf '\e[?25h'
  trap - INT
  [ "${#PICKED[@]}" = 0 ] && P_SEL=()
  # deja la lista a la vista, ya sin cursor ni atajos
  frame="$(app_pick_frame "$q" -1 "$from" "$view" 0)"
  app_paint "$frame" "$prev"
}

cmd_search() {
  local json=0 plain=0 q=() a
  for a in "$@"; do
    case "$a" in
      --json) json=1 ;;
      --list) plain=1 ;;
      -*) die "opción desconocida: $a" ;;
      *) q+=("$a") ;;
    esac
  done
  if [ "${#q[@]}" = 0 ]; then
    { [ "$json" = 0 ] && [ -t 0 ] && [ -t 1 ]; } || die "uso: maxor search <texto> [--json] [--list]"
    local ans
    printf '\n %s⌕%s  %s¿Qué app buscas?%s ' "$E_AC" "$E_RST" "$E_BOLD" "$E_RST"
    read -r ans
    [ -n "$ans" ] || return 0
    read -r -a q <<< "$ans"
  fi

  local anim=0
  { [ "$json" = 0 ] && [ "$ui_on" = 1 ] && [ -t 1 ]; } && anim=1
  [ "$anim" = 1 ] && echo
  app_search_fetch "$anim" "${q[@]}"

  if [ "$json" = 1 ]; then
    jq -c 'map(del(.score))' <<< "$SEARCH_ALL"
    return 0
  fi

  # Datos de la lista en arreglos de bash (más rápido de repintar).
  P_SRC=() P_ID=() P_NAME=() P_DESC=() P_INST=() P_SEL=()
  local installed src id name desc k=0
  installed="$(app_installed_ids)"
  while IFS=$'\t' read -r src id name desc; do
    P_SRC[k]="$src"; P_ID[k]="$id"; P_NAME[k]="$name"; P_DESC[k]="$desc"; P_SEL[k]=0
    if grep -qxF "$src:$id" <<< "$installed"; then P_INST[k]=1; else P_INST[k]=0; fi
    k=$((k + 1))
  done < <(jq -r '.[] | [.source, .id, .name, (.description | gsub("[\t\n]"; " "))] | @tsv' <<< "$SEARCH_ALL")

  if [ "$k" = 0 ]; then
    echo
    ui_say warn "Nada coincide con «${q[*]}». Prueba con otra palabra, o con menos letras."
    return 0
  fi

  if [ "$plain" = 1 ] || [ "$ui_on" = 0 ] || ! [ -t 0 ] || ! [ -t 1 ]; then
    echo
    app_pick_frame "${q[*]}" -1 0 "$k" 0
    echo
    return 0
  fi

  app_pick "${q[*]}"
  if [ "${#PICKED[@]}" = 0 ]; then
    ui_say info "No se instaló nada."
    return 0
  fi
  echo
  local p rc=0
  for p in "${PICKED[@]}"; do
    cmd_install "--${p%%:*}" "${p#*:}" || rc=1
  done
  echo
  if [ "$rc" = 0 ]; then ui_say ok "Listo: ${#PICKED[@]} app(s) instalada(s)"; else ui_say warn "Algunas apps no se pudieron instalar"; fi
  return "$rc"
}

# ── Instalar, quitar, listar ─────────────────────────────────────────

cmd_install() {
  local json=0 src="" ids=() a
  for a in "$@"; do
    case "$a" in
      --json) json=1 ;;
      --nix) src=nix ;;
      --flatpak) src=flatpak ;;
      -*) die "opción desconocida: $a" ;;
      *) ids+=("$a") ;;
    esac
  done
  [ "${#ids[@]}" -gt 0 ] || die "uso: maxor install <paquete…> [--nix|--flatpak] [--json]"

  local id s rc=0 results="[]"
  for id in "${ids[@]}"; do
    s="$src"
    if [ -z "$s" ]; then
      if app_is_flatpak_id "$id"; then s=flatpak; else s=nix; fi
    fi
    local ok=true
    if [ "$s" = nix ]; then
      app_step "$json" "Comprobando $id en nixpkgs" app_nix eval --raw "nixpkgs#$id.meta.name" || {
        [ "$json" = 1 ] || ui_say info "Prueba: maxor search $id"
        ok=false
      }
      if [ "$ok" = true ]; then
        app_step "$json" "Instalando $id (nixpkgs)" app_nix profile add "nixpkgs#$id" || ok=false
      fi
    else
      app_flatpak_remote
      app_step "$json" "Instalando $id (Flathub)" flatpak install --user -y --noninteractive flathub "$id" || ok=false
    fi
    [ "$ok" = true ] || rc=1
    results="$(jq -c --arg id "$id" --arg s "$s" --argjson ok "$ok" '. + [{id: $id, source: $s, ok: $ok}]' <<< "$results")"
  done
  [ "$json" = 1 ] && printf '%s\n' "$results"
  return "$rc"
}

# Origen de una app ya instalada: nix | flatpak | vacío.
app_installed_source() {
  local id="$1"
  if nix profile list --json 2> /dev/null | jq -e --arg id "$id" '.elements | has($id)' > /dev/null; then
    echo nix
  elif flatpak list --user --app --columns=application 2> /dev/null | grep -qx "$id"; then
    echo flatpak
  fi
}

cmd_remove() {
  local json=0 ids=() a
  for a in "$@"; do
    case "$a" in
      --json) json=1 ;;
      -*) die "opción desconocida: $a" ;;
      *) ids+=("$a") ;;
    esac
  done
  [ "${#ids[@]}" -gt 0 ] || die "uso: maxor remove <paquete…> [--json]"

  local id s rc=0 results="[]"
  for id in "${ids[@]}"; do
    s="$(app_installed_source "$id")"
    local ok=true
    case "$s" in
      nix) app_step "$json" "Quitando $id" nix profile remove "$id" || ok=false ;;
      flatpak) app_step "$json" "Quitando $id" flatpak uninstall --user -y --noninteractive "$id" || ok=false ;;
      *)
        [ "$json" = 1 ] || ui_say warn "$id no está instalado con maxor (mira: maxor apps)"
        ok=false
        ;;
    esac
    [ "$ok" = true ] || rc=1
    results="$(jq -c --arg id "$id" --arg s "$s" --argjson ok "$ok" '. + [{id: $id, source: $s, ok: $ok}]' <<< "$results")"
  done
  [ "$json" = 1 ] && printf '%s\n' "$results"
  return "$rc"
}

app_list_json() {
  local nix_l fp_l
  nix_l="$(nix profile list --json 2> /dev/null | jq -c '[.elements | to_entries[] | {
      source: "nix",
      id: .key,
      name: .key,
      version: ((.value.storePaths[0] // "") | split("/") | last | .[33:])
    }]' 2> /dev/null || echo '[]')"
  fp_l="$(flatpak list --user --app --columns=application,name,version 2> /dev/null \
    | jq -R -s -c '[split("\n")[] | select(length > 0) | split("\t") | {source: "flatpak", id: .[0], name: (.[1] // .[0]), version: (.[2] // "")}]' 2> /dev/null || echo '[]')"
  jq -c -n --argjson a "$nix_l" --argjson b "$fp_l" '$a + $b'
}

cmd_apps() {
  local sub="${1:-list}" json=0 a
  shift || true
  for a in "$@"; do [ "$a" = "--json" ] && json=1; done
  case "$sub" in
    list | --json)
      [ "$sub" = "--json" ] && json=1
      local all src id ver
      all="$(app_list_json)"
      if [ "$json" = 1 ]; then
        printf '%s\n' "$all"
        return 0
      fi
      echo
      ui_open "maxor · apps instaladas"
      ui_line ""
      if [ "$(jq 'length' <<< "$all")" = 0 ]; then
        ui_row info "nada instalado con maxor todavía"
      else
        while IFS=$'\t' read -r src id ver; do
          ui_split " $(ui_c "$E_MU" "·")  $id" "$(ui_c "$E_AC2" "$src") "
          [ -n "$ver" ] && ui_line "      $(ui_c "$E_MU" "versión $ver")"
        done < <(jq -r '.[] | [.source, .id, .version] | @tsv' <<< "$all")
      fi
      ui_line ""
      ui_close
      echo
      ;;
    update)
      [ "$json" = 1 ] && die "update no admite --json"
      echo
      ui_run "Actualizando paquetes de nixpkgs" app_nix profile upgrade --all || true
      ui_run "Actualizando apps de Flatpak" flatpak update --user -y --noninteractive || true
      ui_say ok "Apps al día"
      ;;
    *) die "uso: maxor apps [list [--json] | update]" ;;
  esac
}
