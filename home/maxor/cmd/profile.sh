# ── Perfiles: paquetes y servicios por tipo de usuario ───────────────
# El catálogo (nombres y descripciones) es modules/profiles-catalog.json, que
# el paquete recibe en MAXOR_PROFILES. Lo activo se guarda en
# hosts/<equipo>/maxor.json y modules/profiles.nix lo traduce a paquetes.
maxor_cmd profile system "list enable disable"

profile_file() { printf '%s/hosts/%s/maxor.json' "$flake_dir" "$host"; }

# Un flake de git solo ve archivos que git conoce: se registra el nuevo.
track_file() {
  git -C "$flake_dir" rev-parse --git-dir > /dev/null 2>&1 && git -C "$flake_dir" add -N "$1" 2> /dev/null || true
}

profile_enabled() { # → JSON con la lista de perfiles activos
  local f
  f="$(profile_file)"
  if [ -f "$f" ]; then jq -c '.profiles // []' "$f"; else echo '[]'; fi
}

profile_list() {
  local json=0 a
  for a in "$@"; do [ "$a" = "--json" ] && json=1; done
  need_flake
  local all
  all="$(jq -c --argjson on "$(profile_enabled)" \
    'to_entries | map({id: .key, title: .value.title, description: .value.description, enabled: (.key as $k | $on | index($k) != null)})' \
    "$MAXOR_PROFILES")"
  if [ "$json" = 1 ]; then
    printf '%s\n' "$all"
    return 0
  fi
  echo
  ui_open @profile.title "$host"
  ui_line ""
  local on id desc
  while IFS=$'\t' read -r on id desc; do
    printf -v id '%-9s' "$id"
    if [ "$on" = true ]; then ui_row ok "$id $desc"; else ui_row info "$id $desc"; fi
  done < <(jq -r '.[] | [.enabled, .id, .description] | @tsv' <<< "$all")
  ui_line ""
  ui_close
  echo
  ui_say info @profile.hint
}

profile_set() { # enable|disable nombre [opciones de update]
  local action="$1" name="${2:-}"
  shift 2 || true
  [ -n "$name" ] || usage_error profile
  need_flake
  jq -e --arg n "$name" 'has($n)' "$MAXOR_PROFILES" > /dev/null || die_code "$EX_USAGE" @profile.unknown "$name"
  local f cur new word
  f="$(profile_file)"
  cur="$(profile_enabled)"
  if [ "$action" = enable ]; then
    new="$(jq -c --arg n "$name" '. + [$n] | unique' <<< "$cur")"
    msg word @profile.enabled
  else
    new="$(jq -c --arg n "$name" 'map(select(. != $n))' <<< "$cur")"
    msg word @profile.disabled
  fi
  if [ "$new" = "$cur" ]; then
    ui_say info @profile.already "$name" "$word"
    return 0
  fi
  mkdir -p "$(dirname "$f")"
  jq -n --argjson p "$new" '{profiles: $p}' > "$f"
  track_file "$f"
  ui_say ok @profile.changed "$name" "$word" "${f/#$HOME/~}"
  cmd_update --no-lock "$@"
}

cmd_profile() {
  local sub="${1:-list}"
  shift || true
  case "$sub" in
    list | --json) if [ "$sub" = "--json" ]; then profile_list --json; else profile_list "$@"; fi ;;
    enable | disable) profile_set "$sub" "$@" ;;
    *) usage_error profile ;;
  esac
}
