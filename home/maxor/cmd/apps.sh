# ── Apps: buscar, instalar, quitar y actualizar ──────────────────────
maxor_cmd search apps ""
maxor_cmd install apps ""
maxor_cmd remove apps ""
maxor_cmd apps apps "list update updates open repair"
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

# La búsqueda en paralelo habla el mismo idioma que ui_run: spinner por origen y,
# al terminar, el mismo «◇» de cualquier paso hecho.
app_search_frame() { # consulta paso listo_nix listo_flatpak
  local i="$2" d1="$3" d2="$4" sp w_done l1 l2
  sp="${UI_SPIN[i % ${#UI_SPIN[@]}]}"
  msg w_done @search.done
  msg l1 @search.searching_in nixpkgs
  msg l2 @search.searching_in Flathub
  if [ "$d1" = 1 ]; then printf '%s%s%s  nixpkgs  %s%s%s\n' "$E_OK" "$G_OK" "$E_RST" "$E_MU" "$w_done" "$E_RST"; else printf '%s%s%s  %s\n' "$E_AC" "$sp" "$E_RST" "$l1"; fi
  if [ "$d2" = 1 ]; then printf '%s%s%s  flathub  %s%s%s\n' "$E_OK" "$G_OK" "$E_RST" "$E_MU" "$w_done" "$E_RST"; else printf '%s%s%s  %s\n' "$E_AC" "$sp" "$E_RST" "$l2"; fi
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
      ui_paint "$frame" "$prev"
      ui_count_lines "$frame"; prev=$UI_LINES
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

# Los resultados como lista lineal (la misma que ve quien no tiene terminal o pide
# --list). Cada resultado: nombre y origen, y debajo su id, que es lo que se pasa
# a `maxor install`. Lo interactivo (marcar, instalar) vive en la pantalla completa.
app_results_list() { # app_results_list consulta
  local q="$1" n="${#P_ID[@]}" k tag name idp room desc mark
  msg tag @search.results_for "$n" "$q"
  echo
  ui_intro @search.title
  ui_section "$tag"
  for ((k = 0; k < n; k++)); do
    if [ "${P_INST[k]}" = 1 ]; then
      msg tag @search.installed; tag="${E_OK}${tag}${E_RST}"; mark="${E_OK}${G_TICK}${E_RST}"
    elif [ "${P_SRC[k]}" = nix ]; then
      tag="${E_AC2}nixpkgs${E_RST}"; mark="${E_MU}${G_INFO}${E_RST}"
    else
      tag="${E_AC}flathub${E_RST}"; mark="${E_MU}${G_INFO}${E_RST}"
    fi
    ui_truncv name "${P_NAME[k]}" $((ui_w - 24))
    ui_split " $mark  $name" "$tag "
    ui_truncv idp "${P_ID[k]}" 38
    room=$((ui_w - 18 - ${#idp}))
    [ "$room" -lt 0 ] && room=0
    ui_truncv desc "${P_DESC[k]}" "$room"
    ui_text "    ${E_MU}id${E_RST} ${E_AC}${idp}${E_RST}  ${E_MU}${desc}${E_RST}"
  done
  msg tag @search.install_hint
  ui_outro "${E_MU}${tag}${E_RST} ${E_AC}maxor install <id>${E_RST}"
  echo
}

cmd_search() {
  local json=0 plain=0 q=() a
  for a in "$@"; do
    case "$a" in
      --json) json=1 ;;
      --list) plain=1 ;;
      -*) die_code "$EX_USAGE" @err.unknown_option "$a" ;;
      *) q+=("$a") ;;
    esac
  done
  # En una terminal, con la pantalla completa instalada, se busca ahí (marcar,
  # instalar y ver detalles); con --json, --list o sin ella, la lista de abajo.
  if [ "$json" = 0 ] && [ "$plain" = 0 ] && [ -t 0 ] && [ -t 1 ] && [ -z "${MAXOR_NO_TUI:-}" ] && command -v maxor-tui > /dev/null; then
    if [ "${#q[@]}" -gt 0 ]; then exec maxor-tui --screen store --search "${q[*]}"; fi
    exec maxor-tui --screen store
  fi
  if [ "${#q[@]}" = 0 ]; then usage_error search; fi

  local anim=0
  { [ "$json" = 0 ] && [ "$ui_on" = 1 ] && [ -t 1 ]; } && anim=1
  [ "$anim" = 1 ] && echo
  # Lo ya instalado se consulta mientras se busca, no después.
  local inst_f ip
  inst_f="$(mktemp)"
  app_installed_ids > "$inst_f" 2> /dev/null &
  ip=$!
  app_search_fetch "$anim" "${q[@]}"
  wait "$ip" 2> /dev/null || true

  if [ "$json" = 1 ]; then
    jq -c 'map(del(.score))' <<< "$SEARCH_ALL"
    rm -f "$inst_f"
    return 0
  fi

  P_SRC=() P_ID=() P_NAME=() P_DESC=() P_INST=()
  local installed src id name desc k=0
  installed="$(cat "$inst_f")"
  rm -f "$inst_f"
  while IFS=$'\t' read -r src id name desc; do
    P_SRC[k]="$src"; P_ID[k]="$id"; P_NAME[k]="$name"; P_DESC[k]="$desc"
    if grep -qxF "$src:$id" <<< "$installed"; then P_INST[k]=1; else P_INST[k]=0; fi
    k=$((k + 1))
  done < <(jq -r '.[] | [.source, .id, .name, (.description | gsub("[\t\n]"; " "))] | @tsv' <<< "$SEARCH_ALL")

  if [ "$k" = 0 ]; then
    echo
    ui_say warn @search.none "${q[*]}"
    return 0
  fi
  app_results_list "${q[*]}"
}

# ── Instalar, quitar, listar ─────────────────────────────────────────

cmd_install() {
  local json=0 src="" ids=() a
  for a in "$@"; do
    case "$a" in
      --json) json=1 ;;
      --nix) src=nix ;;
      --flatpak) src=flatpak ;;
      -*) die_code "$EX_USAGE" @err.unknown_option "$a" ;;
      *) ids+=("$a") ;;
    esac
  done
  [ "${#ids[@]}" -gt 0 ] || usage_error install
  app_lock

  local id s m rc=0 results="[]"
  for id in "${ids[@]}"; do
    s="$src"
    if [ -z "$s" ]; then
      if app_is_flatpak_id "$id"; then s=flatpak; else s=nix; fi
    fi
    local ok=true
    if [ "$s" = nix ]; then
      msg m @apps.checking "$id"
      app_step "$json" "$m" app_nix eval --raw "nixpkgs#$id.meta.name" || {
        [ "$json" = 1 ] || ui_say info @apps.try_search "$id"
        ok=false
      }
      if [ "$ok" = true ]; then
        msg m @apps.installing_nix "$id"
        app_step "$json" "$m" app_nix profile add "nixpkgs#$id" || ok=false
      fi
    else
      app_flatpak_remote
      msg m @apps.installing_flatpak "$id"
      app_step "$json" "$m" flatpak install --user -y --noninteractive flathub "$id" || ok=false
      [ "$ok" = true ] && app_flatpak_expose "$id"
    fi
    [ "$ok" = true ] || rc=1
    results="$(jq -c --arg id "$id" --arg s "$s" --argjson ok "$ok" '. + [{id: $id, source: $s, ok: $ok}]' <<< "$results")"
  done
  app_refresh_launcher
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

# Una sola operación de paquetes a la vez (dos `nix profile` o `flatpak` juntos fallan o
# se pisan): si otra maxor está instalando o quitando, esta espera su turno.
app_lock() {
  command -v flock > /dev/null 2>&1 || return 0
  mkdir -p "$state" 2> /dev/null || return 0
  exec 9> "$state/apps.lock"
  flock -w 600 9 || true
}

# ── Flatpak visible en la sesión ─────────────────────────────────────
# Flatpak deja las entradas de menú, los iconos y los comandos de cada app en
# ~/.local/share/flatpak/exports. Una sesión solo los ve si arrancó con esa carpeta en
# XDG_DATA_DIRS y PATH, y una sesión que empezó antes de que el sistema la añadiera
# (o un lanzador que no la lee) no muestra nada: ni Super+Espacio ni la terminal.
# Para que funcione siempre y sin volver a iniciar sesión, cada app se enlaza a las
# carpetas del usuario, que todas las sesiones leen:
#   entrada de menú  →  ~/.local/share/applications/<id>.desktop
#   iconos           →  ~/.local/share/icons/…
#   comando          →  ~/.local/bin/<nombre>  (nombre de la app en minúsculas)
app_flatpak_data() { printf '%s' "${XDG_DATA_HOME:-$HOME/.local/share}"; }

app_flatpak_expose() { # app_flatpak_expose id
  local id="$1" data ex src f rel name slug
  data="$(app_flatpak_data)"
  ex="$data/flatpak/exports"
  src="$ex/share/applications/$id.desktop"
  [ -e "$src" ] || return 0
  mkdir -p "$data/applications"
  ln -sf "$src" "$data/applications/$id.desktop"
  while IFS= read -r f; do
    rel="${f#"$ex/share/icons/"}"
    mkdir -p "$data/icons/$(dirname "$rel")"
    ln -sf "$f" "$data/icons/$rel"
  done < <(find "$ex/share/icons" \( -type f -o -type l \) -name "$id.*" 2> /dev/null)
  name="$(grep -m1 '^Name=' "$src" | cut -d= -f2-)"
  slug="$(tr '[:upper:]' '[:lower:]' <<< "$name" | tr -cd 'a-z0-9-')"
  # un comando corto con el nombre de la app, salvo que ya haya otro con ese nombre
  if [ -n "$slug" ] && { ! command -v "$slug" > /dev/null 2>&1 || grep -q "maxor flatpak shim: $id\$" "$HOME/.local/bin/$slug" 2> /dev/null; }; then
    mkdir -p "$HOME/.local/bin"
    printf '#!/bin/sh\n# maxor flatpak shim: %s\nexec flatpak run %s "$@"\n' "$id" "$id" > "$HOME/.local/bin/$slug"
    chmod +x "$HOME/.local/bin/$slug"
  fi
}

app_flatpak_unexpose() { # app_flatpak_unexpose id
  local id="$1" data f
  data="$(app_flatpak_data)"
  rm -f "$data/applications/$id.desktop"
  find "$data/icons" -type l -name "$id.*" -delete 2> /dev/null || true
  for f in "$HOME"/.local/bin/*; do
    [ -f "$f" ] && grep -q "maxor flatpak shim: $id\$" "$f" 2> /dev/null && rm -f "$f"
  done
  return 0
}

# Ids de las apps de flatpak instaladas que esta sesión no podría mostrar.
app_flatpak_hidden() {
  local data id
  data="$(app_flatpak_data)"
  case ":${XDG_DATA_DIRS:-}:" in *":$data/flatpak/exports/share:"*) return 0 ;; esac
  while IFS= read -r id; do
    [ -n "$id" ] || continue
    [ -e "$data/applications/$id.desktop" ] || echo "$id"
  done < <(flatpak list --user --app --columns=application 2> /dev/null)
}

# El lanzador de apps (Super+Espacio) se entera de las apps nuevas, pero no de las que
# se quitan: el perfil de nix cambia de carpeta y su vigilancia sigue mirando la vieja.
# Crear y borrar un archivo en la carpeta de aplicaciones del usuario le hace releer todo.
app_refresh_launcher() {
  local d="${XDG_DATA_HOME:-$HOME/.local/share}/applications" f
  mkdir -p "$d" 2> /dev/null || return 0
  find "$d" -maxdepth 1 -xtype l -delete 2> /dev/null || true # enlaces de apps ya quitadas
  rm -f "$state/apps-updates.json" # cambió algo: las versiones nuevas hay que volver a mirarlas
  f="$d/.maxor-refresh"
  : > "$f" 2> /dev/null || return 0
  sleep 0.3
  rm -f "$f"
}

# Tamaño legible de un número de bytes.
app_human() {
  local b="$1"
  if [ "$b" -ge 1073741824 ]; then printf '%d.%d GiB' $((b / 1073741824)) $((b % 1073741824 * 10 / 1073741824))
  elif [ "$b" -ge 1048576 ]; then printf '%d.%d MiB' $((b / 1048576)) $((b % 1048576 * 10 / 1048576))
  else printf '%d KiB' $(((b + 1023) / 1024)); fi
}

# Carpetas que una app deja en tu casa: solo las que se llaman exactamente como la
# app (sin distinguir mayúsculas), p. ej. ~/.local/share/ATLauncher para «atlauncher».
# Imprime «ruta» y «bytes» separados por tabulador, una por línea.
app_leftovers() {
  local id="$1" name base root p
  name="${id,,}"
  [ "${#name}" -ge 3 ] || return 0
  case "$name" in ssh | gnupg | gpg | nix | local | config | cache | share | state | var | bin | home) return 0 ;; esac
  for root in "$HOME/.config" "$HOME/.local/share" "$HOME/.cache" "$HOME/.local/state" "$HOME/.var/app" "$HOME"; do
    [ -d "$root" ] || continue
    for p in "$root"/* "$root"/.[!.]*; do
      [ -e "$p" ] || [ -L "$p" ] || continue
      base="${p##*/}"
      base="${base,,}"
      if [ "$root" = "$HOME" ]; then
        [ "$base" = ".$name" ] || continue
      else
        [ "$base" = "$name" ] || continue
      fi
      printf '%s\t%s\n' "$p" "$(du -sb -- "$p" 2> /dev/null | cut -f1 || echo 0)"
    done
  done
  # algunas apps (ATLauncher) escriben su registro en ~/logs/<nombre>.log
  for p in "$HOME"/logs/*; do
    [ -f "$p" ] || continue
    base="${p##*/}"
    [ "${base,,}" = "$name.log" ] || continue
    printf '%s\t%s\n' "$p" "$(du -sb -- "$p" 2> /dev/null | cut -f1 || echo 0)"
  done
}

# Borra las carpetas de app_leftovers; nunca sale de tu casa ni toca las raíces.
app_purge() {
  local id="$1" p b n=0
  while IFS=$'\t' read -r p b; do
    [ -n "$p" ] || continue
    case "$p" in "$HOME"/?*) ;; *) continue ;; esac
    rm -rf -- "$p" && n=$((n + 1))
  done < <(app_leftovers "$id")
  echo "$n"
}

app_leftovers_json() {
  app_leftovers "$1" | jq -R -s -c 'split("\n") | map(select(length > 0) | split("\t") | {path: .[0], bytes: (.[1] | tonumber? // 0)})'
}

cmd_remove() {
  local json=0 purge=0 ids=() a list_data=0
  for a in "$@"; do
    case "$a" in
      --json) json=1 ;;
      --purge) purge=1 ;;
      --list-data) list_data=1 ;;
      -*) die_code "$EX_USAGE" @err.unknown_option "$a" ;;
      *) ids+=("$a") ;;
    esac
  done
  [ "${#ids[@]}" -gt 0 ] || usage_error remove
  # Solo mirar: qué carpetas tiene la app en tu casa, sin quitar ni borrar nada.
  if [ "$list_data" = 1 ]; then
    app_leftovers_json "${ids[0]}"
    return 0
  fi
  app_lock

  local id s m rc=0 results="[]" left total count purged
  for id in "${ids[@]}"; do
    s="$(app_installed_source "$id")"
    local ok=true
    msg m @apps.removing "$id"
    case "$s" in
      nix) app_step "$json" "$m" nix profile remove "$id" || ok=false ;;
      flatpak)
        app_step "$json" "$m" flatpak uninstall --user -y --noninteractive "$id" || ok=false
        [ "$ok" = true ] && app_flatpak_unexpose "$id"
        ;;
      *)
        # con --purge se puede limpiar lo que dejó una app que ya no está
        if [ "$purge" = 0 ] || [ -z "$(app_leftovers "$id")" ]; then
          [ "$json" = 1 ] || ui_say warn @apps.not_installed "$id"
          ok=false
        fi
        ;;
    esac
    left="$(app_leftovers_json "$id")"
    count="$(jq 'length' <<< "$left")"
    total="$(jq '[.[].bytes] | add // 0' <<< "$left")"
    purged=false
    if [ "$ok" = true ] && [ "$count" -gt 0 ]; then
      if [ "$purge" = 1 ]; then
        app_purge "$id" > /dev/null
        purged=true
        [ "$json" = 1 ] || ui_say ok @apps.purged "$(app_human "$total")"
      elif [ "$json" = 0 ]; then
        if [ -t 0 ] && ui_confirm @apps.purge_confirm "$count" "$(app_human "$total")"; then
          app_purge "$id" > /dev/null
          purged=true
          ui_say ok @apps.purged "$(app_human "$total")"
        else
          ui_say info @apps.kept "$(app_human "$total")" "$id"
        fi
      fi
    fi
    [ "$ok" = true ] || rc=1
    results="$(jq -c --arg id "$id" --arg s "$s" --argjson ok "$ok" --argjson left "$left" --argjson purged "$purged" \
      '. + [{id: $id, source: $s, ok: $ok, purged: $purged, leftovers: (if $purged then [] else $left end)}]' <<< "$results")"
  done
  app_refresh_launcher
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

# Carga la lista de apps; en una terminal con color muestra un esqueleto
# parpadeando mientras llega, y la lista lo sustituye sin que la pantalla salte.
app_list_load() { # → APPS_ALL (con el cargador único; la línea de resultado queda en el riel)
  ui_run @apps.loading app_list_json
  APPS_ALL="$UI_OUT"
}

# Apps con versión nueva. nix: lo que ofrece el nixpkgs de este sistema frente a lo instalado
# (las versiones nuevas llegan con `maxor update`); flatpak: lo que anuncia Flathub.
# Con caché: mirar versiones nuevas compara cada app con nixpkgs y pregunta a Flathub, y
# tarda unos segundos. El resultado vale 10 minutos mientras no cambien las apps ni el
# sistema; instalar, quitar o actualizar lo descarta. `apps updates --refresh` lo ignora.
app_updates_cached() { # app_updates_cached [refresh]
  local f="$state/apps-updates.json" key now at
  key="$( (nix profile list --json 2> /dev/null || true; readlink /run/current-system || true; flatpak list --user --app --columns=application,active 2> /dev/null || true) | sha256sum | cut -c1-24)"
  now="$(printf '%(%s)T' -1)"
  if [ "${1:-}" != refresh ] && [ -s "$f" ] \
    && [ "$(jq -r '.key' "$f" 2> /dev/null)" = "$key" ] \
    && at="$(jq -r '.at' "$f" 2> /dev/null)" && [ $((now - at)) -lt 600 ]; then
    jq -c '.data' "$f"
    return 0
  fi
  local data
  data="$(app_updates_json)"
  mkdir -p "$state" 2> /dev/null && jq -cn --arg key "$key" --argjson at "$now" --argjson data "$data" '{key: $key, at: $at, data: $data}' > "$f" 2> /dev/null || true
  printf '%s\n' "$data"
}

app_updates_json() {
  local out="[]" id name cur latest tmp
  tmp="$(mktemp -d)"
  while IFS=$'\t' read -r id name; do
    [ -n "$id" ] || continue
    (app_nix eval --raw "nixpkgs#$id.name" > "$tmp/$id" 2> /dev/null || true) &
  done < <(nix profile list --json 2> /dev/null | jq -r '.elements | to_entries[] | [.key, ((.value.storePaths[0] // "") | split("/") | last | .[33:])] | @tsv')
  wait
  while IFS=$'\t' read -r id name; do
    [ -n "$id" ] || continue
    latest="$(< "$tmp/$id")"
    [ -n "$latest" ] && [ "$latest" != "$name" ] || continue
    cur="$(sed -E 's/^.*-([0-9][^-]*)$/\1/' <<< "$name")"
    latest="$(sed -E 's/^.*-([0-9][^-]*)$/\1/' <<< "$latest")"
    out="$(jq -c --arg id "$id" --arg c "$cur" --arg l "$latest" '. + [{source: "nix", id: $id, current: $c, latest: $l}]' <<< "$out")"
  done < <(nix profile list --json 2> /dev/null | jq -r '.elements | to_entries[] | [.key, ((.value.storePaths[0] // "") | split("/") | last | .[33:])] | @tsv')
  rm -rf "$tmp"
  local fp
  fp="$(flatpak remote-ls --user --updates --app --columns=application,version 2> /dev/null \
    | jq -R -s -c '[split("\n")[] | select(length > 0) | split("\t") | {source: "flatpak", id: .[0], current: "", latest: (.[1] // "")}]' 2> /dev/null || true)"
  jq -e 'type == "array"' <<< "$fp" > /dev/null 2>&1 || fp='[]'
  jq -c -n --argjson a "$out" --argjson b "$fp" '$a + $b'
}

# Abre una app instalada sin que haya que saber su comando: usa su entrada de menú
# (.desktop) y, si no la hay, el programa con el mismo nombre. Se separa de la terminal.
app_open() {
  local id="$1" s d f exec_line dirs
  s="$(app_installed_source "$id")"
  case "$s" in
    flatpak) setsid -f flatpak run "$id" > /dev/null 2>&1 < /dev/null ;;
    nix)
      dirs="$HOME/.nix-profile/share:$HOME/.local/state/nix/profile/share:${XDG_DATA_DIRS:-}"
      IFS=: read -ra dl <<< "$dirs"
      for d in "${dl[@]}"; do
        for f in "$d"/applications/*"$id"*.desktop; do
          [ -f "$f" ] || continue
          exec_line="$(grep -m1 '^Exec=' "$f" | cut -d= -f2- | sed -E 's/ %[a-zA-Z]//g')"
          [ -n "$exec_line" ] || continue
          setsid -f bash -c "$exec_line" > /dev/null 2>&1 < /dev/null
          return 0
        done
      done
      if command -v "$id" > /dev/null 2>&1 || [ -x "$HOME/.nix-profile/bin/$id" ]; then
        setsid -f "$id" > /dev/null 2>&1 < /dev/null || setsid -f "$HOME/.nix-profile/bin/$id" > /dev/null 2>&1 < /dev/null
      else
        return 1
      fi
      ;;
    *) return 1 ;;
  esac
}

cmd_apps() {
  local sub="${1:-list}" json=0 a
  shift || true
  for a in "$@"; do [ "$a" = "--json" ] && json=1; done
  case "$sub" in
    list | --json)
      [ "$sub" = "--json" ] && json=1
      local all src id ver vtxt
      if [ "$json" = 1 ]; then
        app_list_json
        return 0
      fi
      echo
      ui_intro @apps.title
      app_list_load
      all="$APPS_ALL"
      ui_section @apps.sec_installed
      if [ "$(jq 'length' <<< "$all")" = 0 ]; then
        ui_row info @apps.empty
      else
        while IFS=$'\t' read -r src id ver; do
          ui_split " $(ui_c "$E_MU" "·")  $id" "$(ui_c "$E_AC2" "$src") "
          if [ -n "$ver" ]; then msg vtxt @apps.version "$ver"; ui_text "      ${E_MU}${vtxt}${E_RST}"; fi
        done < <(jq -r '.[] | [.source, .id, .version] | @tsv' <<< "$all")
      fi
      ui_outro
      echo
      ;;
    updates)
      local fresh="" notify=0 upd_json n names ttl m
      for a in "$@"; do
        [ "$a" = "--refresh" ] && fresh=refresh
        [ "$a" = "--notify" ] && notify=1
      done
      upd_json="$(app_updates_cached "$fresh")"
      printf '%s\n' "$upd_json"
      # --notify (lo usa el temporizador diario): avisa con una notificación si hay algo nuevo
      if [ "$notify" = 1 ]; then
        n="$(jq 'length' <<< "$upd_json")"
        if [ "$n" -gt 0 ] && command -v notify-send > /dev/null 2>&1; then
          names="$(jq -r '[.[].id] | .[:4] | join(", ")' <<< "$upd_json")"
          [ "$n" -gt 4 ] && names="$names…"
          msg m @apps.notify_body "$n" "$names"
          msg ttl @apps.notify_title
          notify-send -a Maxor -i software-update-available "$ttl" "$m" || true
        fi
      fi
      ;;
    repair)
      # Deja a la vista las apps de flatpak ya instaladas (menú, iconos y comando).
      local rid rn=0
      while IFS= read -r rid; do
        [ -n "$rid" ] || continue
        app_flatpak_expose "$rid"
        rn=$((rn + 1))
      done < <(flatpak list --user --app --columns=application 2> /dev/null)
      app_refresh_launcher
      if [ "$json" = 1 ]; then jq -cn --argjson n "$rn" '{repaired: $n}'; else ui_say ok @apps.repaired "$rn"; fi
      ;;
    open)
      local oid="${1:-}" ok=true
      [ -n "$oid" ] && [ "${oid#-}" = "$oid" ] || usage_error apps
      app_open "$oid" || ok=false
      if [ "$json" = 1 ]; then
        jq -cn --arg id "$oid" --argjson ok "$ok" '[{id: $id, ok: $ok}]'
      elif [ "$ok" = false ]; then
        ui_say warn @apps.not_installed "$oid"
      fi
      [ "$ok" = true ]
      ;;
    update)
      # con nombres: solo esas apps y salida JSON; sin nombres: todas, con la interfaz de siempre
      local uids=() u uok uresults="[]" us
      for u in "$@"; do [ "${u#-}" = "$u" ] && uids+=("$u"); done
      if [ "${#uids[@]}" -gt 0 ]; then
        app_lock
        for u in "${uids[@]}"; do
          uok=true
          us="$(app_installed_source "$u")"
          case "$us" in
            nix) app_nix profile upgrade "$u" > /dev/null 2>&1 || uok=false ;;
            flatpak) flatpak update --user -y --noninteractive "$u" > /dev/null 2>&1 || uok=false ;;
            *) uok=false ;;
          esac
          uresults="$(jq -c --arg id "$u" --arg s "$us" --argjson ok "$uok" '. + [{id: $id, source: $s, ok: $ok}]' <<< "$uresults")"
        done
        app_refresh_launcher
        [ "$json" = 1 ] && printf '%s\n' "$uresults"
        jq -e 'all(.[]; .ok)' <<< "$uresults" > /dev/null
        return $?
      fi
      [ "$json" = 1 ] && die_code "$EX_USAGE" @apps.no_json
      echo
      ui_intro @apps.upd_title
      ui_run @apps.upd_nix app_nix profile upgrade --all || true
      ui_run @apps.upd_flatpak flatpak update --user -y --noninteractive || true
      app_refresh_launcher
      ui_outro @apps.up_to_date
      echo
      ;;
    *) usage_error apps ;;
  esac
}
