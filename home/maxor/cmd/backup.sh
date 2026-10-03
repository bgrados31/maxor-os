# ── Copia de seguridad: guardar y recuperar tu Maxor ─────────────────
maxor_cmd backup system "--list"
maxor_cmd restore system ""
# Un solo archivo (.tar.gz) con lo que es tuyo y no sale de git ni de Nix:
#   manifest.json        versión, equipo, fecha, tema, perfiles y apps instaladas
#   hosts/maxor.json     los perfiles activos de este equipo
#   themes/<id>/         los temas que hiciste tú (los oficiales ya vienen con Maxor)
#   dms/                 los ajustes de DankMaterialShell (barra, lanzador…)
# No lleva hardware.json: es de cada equipo y se vuelve a detectar solo.

backup_default_path() { # backup_default_path
  printf '%s/maxor-backup-%s-%(%Y%m%d-%H%M)T.tar.gz' "$HOME" "$host" -1
}

# Los temas hechos por el usuario: carpetas de verdad (los oficiales son enlaces al store).
backup_user_themes() {
  local dir="${XDG_DATA_HOME:-$HOME/.local/share}/maxor/themes" d
  [ -d "$dir" ] || return 0
  for d in "$dir"/*/; do
    [ -d "$d" ] && [ ! -L "${d%/}" ] && basename "$d"
  done
}

backup_manifest() { # backup_manifest → JSON
  local apps themes profiles cur
  apps="$(app_list_json 2> /dev/null | jq -c '[.[] | {source, id}]' 2> /dev/null || echo '[]')"
  themes="$(backup_user_themes | jq -R -s -c 'split("\n") | map(select(length > 0))')"
  profiles="$(profile_enabled 2> /dev/null || echo '[]')"
  cur="$(cat "$state/current" 2> /dev/null || echo sakura)"
  jq -cn --arg v "$MAXOR_VERSION" --arg host "$host" --arg at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" --arg theme "$cur" \
    --argjson apps "$apps" --argjson themes "$themes" --argjson profiles "$profiles" \
    '{schema: 1, maxor: $v, host: $host, created: $at, theme: $theme, profiles: $profiles, themes: $themes, apps: $apps}'
}

backup_pack() { # backup_pack archivo
  local out="$1" tmp
  tmp="$(mktemp -d)"
  mkdir -p "$tmp/hosts" "$tmp/themes" "$tmp/dms"
  backup_manifest > "$tmp/manifest.json"
  local f="$flake_dir/hosts/$host/maxor.json"
  [ -f "$f" ] && cp "$f" "$tmp/hosts/maxor.json"
  local t tdir="${XDG_DATA_HOME:-$HOME/.local/share}/maxor/themes"
  while IFS= read -r t; do
    [ -n "$t" ] && cp -r "$tdir/$t" "$tmp/themes/$t"
  done < <(backup_user_themes)
  local dms="${XDG_CONFIG_HOME:-$HOME/.config}/DankMaterialShell"
  [ -d "$dms" ] && cp -r "$dms/." "$tmp/dms/"
  mkdir -p "$(dirname "$out")"
  tar -czf "$out" -C "$tmp" . || { rm -rf "$tmp"; return 1; }
  rm -rf "$tmp"
}

cmd_backup() {
  local json=0 list=0 dest="" a
  for a in "$@"; do
    case "$a" in
      --json) json=1 ;;
      --list) list=1 ;;
      -*) die_code "$EX_USAGE" @err.unknown_option "$a" ;;
      *) dest="$a" ;;
    esac
  done
  # --list <archivo>: qué lleva una copia, sin tocar nada
  if [ "$list" = 1 ]; then
    [ -f "$dest" ] || die_code "$EX_USAGE" @backup.no_file "${dest:-?}"
    backup_read_manifest "$dest" || die_code "$EX_FAIL" @backup.bad_file "$dest"
    if [ "$json" = 1 ]; then printf '%s\n' "$BACKUP_MANIFEST"; else backup_show "$BACKUP_MANIFEST"; fi
    return 0
  fi
  [ -n "$dest" ] || dest="$(backup_default_path)"
  [ -d "$dest" ] && dest="$dest/$(basename "$(backup_default_path)")"
  local bytes apps themes
  if [ "$json" = 1 ]; then
    backup_pack "$dest" 2> /dev/null || die_code "$EX_FAIL" @backup.failed
  else
    echo
    ui_intro @backup.title "$host"
    ui_run @backup.step_pack backup_pack "$dest" || return $?
  fi
  bytes="$(stat -c %s "$dest")"
  apps="$(tar -xzOf "$dest" ./manifest.json 2> /dev/null | jq '.apps | length' 2> /dev/null || echo 0)"
  themes="$(tar -xzOf "$dest" ./manifest.json 2> /dev/null | jq '.themes | length' 2> /dev/null || echo 0)"
  if [ "$json" = 1 ]; then
    jq -cn --arg p "$dest" --argjson b "$bytes" --argjson a "$apps" --argjson t "$themes" --arg host "$host" '{path: $p, bytes: $b, apps: $a, themes: $t, host: $host}'
  else
    ui_say ok @backup.saved "${dest/#$HOME/~}" "$(app_human "$bytes")"
    ui_say info @backup.contents "$apps" "$themes"
    ui_outro @backup.hint
    echo
  fi
}

# Lee el manifest de una copia y comprueba que es una copia de Maxor y que es segura.
backup_read_manifest() { # backup_read_manifest archivo → BACKUP_MANIFEST
  local f="$1" bad
  tar -tzf "$f" > /dev/null 2>&1 || return 1
  # nada de rutas absolutas ni de «..»: solo se extrae lo que se conoce
  bad="$(tar -tzf "$f" | grep -E '(^/|(^|/)\.\.(/|$))' || true)"
  [ -z "$bad" ] || return 1
  BACKUP_MANIFEST="$(tar -xzOf "$f" ./manifest.json 2> /dev/null)" || return 1
  jq -e '.schema == 1 and (.host | type == "string")' <<< "$BACKUP_MANIFEST" > /dev/null 2>&1
}

backup_show() { # backup_show manifest
  local m="$1" line
  echo
  ui_intro @backup.show_title "$(jq -r '.host' <<< "$m")"
  ui_section @backup.sec_about
  ui_row info "$(jq -r '"maxor \(.maxor) · \(.created)"' <<< "$m")"
  ui_row info "$(jq -r '"theme \(.theme)"' <<< "$m")"
  ui_section @backup.sec_profiles
  if [ "$(jq '.profiles | length' <<< "$m")" = 0 ]; then ui_row info @backup.none; fi
  while IFS= read -r line; do ui_row info "$line"; done < <(jq -r '.profiles[]' <<< "$m")
  ui_section @backup.sec_apps
  if [ "$(jq '.apps | length' <<< "$m")" = 0 ]; then ui_row info @backup.none; fi
  while IFS= read -r line; do ui_row info "$line"; done < <(jq -r '.apps[] | "\(.id)  (\(.source))"' <<< "$m")
  ui_outro
  echo
}

cmd_restore() {
  local yes=0 apps=0 file="" a
  for a in "$@"; do
    case "$a" in
      -y | --yes) yes=1 ;;
      --apps) apps=1 ;;
      -*) die_code "$EX_USAGE" @err.unknown_option "$a" ;;
      *) file="$a" ;;
    esac
  done
  [ -n "$file" ] || usage_error restore
  [ -f "$file" ] || die_code "$EX_USAGE" @backup.no_file "$file"
  backup_read_manifest "$file" || die_code "$EX_FAIL" @backup.bad_file "$file"
  backup_show "$BACKUP_MANIFEST"
  if [ "$yes" = 0 ] && ! ui_confirm @restore.confirm; then
    ui_say info @err.cancelled
    return 0
  fi
  local tmp t theme dms
  tmp="$(mktemp -d)"
  tar -xzf "$file" -C "$tmp" --no-same-owner || { rm -rf "$tmp"; die_code "$EX_FAIL" @backup.bad_file "$file"; }
  echo
  ui_intro @restore.title "$host"
  # perfiles de este equipo
  if [ -f "$tmp/hosts/maxor.json" ] && [ -d "$flake_dir/hosts/$host" ]; then
    cp "$tmp/hosts/maxor.json" "$flake_dir/hosts/$host/maxor.json"
    track_file "$flake_dir/hosts/$host/maxor.json"
    ui_say ok @restore.profiles
  fi
  # temas propios: nunca se pisa uno que ya existe
  local tdir="${XDG_DATA_HOME:-$HOME/.local/share}/maxor/themes"
  mkdir -p "$tdir"
  for t in "$tmp"/themes/*/; do
    [ -d "$t" ] || continue
    if [ -e "$tdir/$(basename "$t")" ]; then continue; fi
    cp -r "$t" "$tdir/$(basename "$t")"
    ui_say ok @restore.theme "$(basename "$t")"
  done
  # ajustes de DankMaterialShell: se guarda una copia de los actuales
  dms="${XDG_CONFIG_HOME:-$HOME/.config}/DankMaterialShell"
  if [ -n "$(ls -A "$tmp/dms" 2> /dev/null)" ]; then
    mkdir -p "$dms"
    [ -f "$dms/settings.json" ] && cp "$dms/settings.json" "$dms/settings.json.before-restore"
    cp -r "$tmp/dms/." "$dms/"
    ui_say ok @restore.dms
  fi
  rm -rf "$tmp"
  theme="$(jq -r '.theme' <<< "$BACKUP_MANIFEST")"
  if [ -e "$tdir/$theme" ] && [ "$(cat "$state/current" 2> /dev/null)" != "$theme" ]; then
    cmd_theme apply "$theme" > /dev/null 2>&1 && ui_say ok @restore.applied_theme "$theme"
  fi
  # apps: solo si se pide, porque descarga cosas
  if [ "$apps" = 1 ]; then
    local src id
    while IFS=$'\t' read -r src id; do
      [ -n "$id" ] || continue
      cmd_install "--$src" "$id" || ui_say warn @restore.app_failed "$id"
    done < <(jq -r '.apps[] | [.source, .id] | @tsv' <<< "$BACKUP_MANIFEST")
  elif [ "$(jq '.apps | length' <<< "$BACKUP_MANIFEST")" -gt 0 ]; then
    ui_say info @restore.apps_hint
  fi
  ui_outro @restore.done
  echo
}
