# ── Apps: buscar, instalar, quitar y actualizar ──────────────────────
# Dos orígenes, ninguno pide root ni toca archivos .nix:
#   nix      paquetes de nixpkgs en el perfil del usuario (nix profile)
#   flatpak  apps de Flathub, instaladas para el usuario (flatpak --user)
# Los comandos que listan aceptan --json: es el contrato que usará Maxor Store.

flathub_url="https://dl.flathub.org/repo/flathub.flatpakrepo"

# Los paquetes con licencia no libre (Steam, Spotify…) exigen esto en nix.
app_nix() { NIXPKGS_ALLOW_UNFREE=1 nix "$@" --impure; }

# Relevancia de un resultado: 0 exacto, 1 empieza por, 2 contiene, 3 solo en la descripción.
app_rank='def rank($q): ($q | ascii_downcase) as $x | (.id | split(".") | last | ascii_downcase) as $i | ((.name // "") | ascii_downcase) as $n | if $i == $x or $n == $x then 0 elif ($i | startswith($x)) or ($n | startswith($x)) then 1 elif ($i | contains($x)) or ($n | contains($x)) then 2 else 3 end; '

app_flatpak_remote() {
  flatpak remote-add --user --if-not-exists flathub "$flathub_url"
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

cmd_search() {
  local json=0 q=() a
  for a in "$@"; do
    case "$a" in
      --json) json=1 ;;
      -*) die "opción desconocida: $a" ;;
      *) q+=("$a") ;;
    esac
  done
  [ "${#q[@]}" -gt 0 ] || die "uso: maxor search <texto> [--json]"

  local nix_res="[]" fp_res="[]"
  if app_step "$json" "Buscando en nixpkgs" nix search nixpkgs --json "${q[@]}"; then
    local nix_in="${UI_OUT:-}"
    [ -n "$nix_in" ] || nix_in="{}"
    nix_res="$(jq -c "$app_rank"'[to_entries[] | {
        source: "nix",
        id: (.key | sub("^legacyPackages\\.[^.]+\\."; "")),
        name: .value.pname,
        version: .value.version,
        description: (.value.description // "")
      }] | map(. + {score: rank($q)}) | sort_by(.score) | map(del(.score)) | .[:20]' --arg q "${q[*]}" <<< "$nix_in" 2> /dev/null || echo '[]')"
  fi
  app_flatpak_remote 2> /dev/null || true
  if app_step "$json" "Buscando en Flathub" flatpak search --columns=application,name,description "${q[*]}"; then
    fp_res="$(jq -R -s -c "$app_rank"'[split("\n")[] | select(length > 0) | split("\t") | select(length >= 2) | {
        source: "flatpak",
        id: .[0],
        name: .[1],
        version: "",
        description: (.[2] // "")
      }] | unique_by(.id) | map(. + {score: rank($q)}) | sort_by(.score) | map(del(.score)) | .[:20]' --arg q "${q[*]}" <<< "$UI_OUT" 2> /dev/null || echo '[]')"
  fi

  local all
  all="$(jq -c -n --argjson a "$nix_res" --argjson b "$fp_res" '$a + $b')"
  if [ "$json" = 1 ]; then
    printf '%s\n' "$all"
    return 0
  fi
  echo
  ui_open "maxor · búsqueda · ${q[*]}"
  ui_line ""
  if [ "$(jq 'length' <<< "$all")" = 0 ]; then
    ui_row warn "sin resultados"
  else
    jq -r '.[] | [.source, .id, .description] | @tsv' <<< "$all" | while IFS=$'\t' read -r src id desc; do
      ui_row info "$(printf '%-8s %-28s' "$src" "$id") $desc"
    done
  fi
  ui_line ""
  ui_close
  echo
  ui_say info "Instalar: maxor install <id>   (ID con puntos = Flatpak, si no nixpkgs; fuerza con --nix o --flatpak)"
}

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
      app_flatpak_remote 2> /dev/null || true
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
      local all
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
        jq -r '.[] | [.source, .id, .version] | @tsv' <<< "$all" | while IFS=$'\t' read -r src id ver; do
          ui_row info "$(printf '%-8s %-30s' "$src" "$id") $ver"
        done
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
