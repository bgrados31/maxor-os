# ── Actualización: update y rollback ─────────────────────────────────
maxor_cmd update system ""
maxor_cmd rollback system "--list"

# Compara el resultado compilado (UPDATE_OUT) con el sistema en marcha.
update_compare() {
  nix store diff-closures /run/current-system "$UPDATE_OUT" 2> /dev/null || true
}

# Huella del repositorio de la configuración: cambia con un commit o con cualquier
# archivo modificado. Sirve para saber si un escaneo guardado sigue valiendo.
update_fingerprint() {
  local head files
  head="$(git -C "$flake_dir" rev-parse HEAD 2> /dev/null || echo none)"
  files="$(git -C "$flake_dir" status --porcelain 2> /dev/null | sha256sum | cut -c1-16)"
  printf '%s-%s' "${head:0:12}" "$files"
}

# Estado barato (milisegundos, no compila nada): en qué rama está la configuración,
# qué canal de nixpkgs usa y cuándo se bloqueó, y qué generación corre.
update_status_json() {
  local branch="" commit="" files=0 channel="" nrev="" ndate=0 gen=0 target
  if git -C "$flake_dir" rev-parse --git-dir > /dev/null 2>&1; then
    branch="$(git -C "$flake_dir" rev-parse --abbrev-ref HEAD 2> /dev/null || true)"
    commit="$(git -C "$flake_dir" rev-parse --short HEAD 2> /dev/null || true)"
    files="$(git -C "$flake_dir" status --porcelain 2> /dev/null | wc -l)"
  fi
  if [ -f "$flake_dir/flake.lock" ]; then
    channel="$(jq -r '.nodes.nixpkgs.original.ref // ""' "$flake_dir/flake.lock" 2> /dev/null || true)"
    nrev="$(jq -r '.nodes.nixpkgs.locked.rev // "" | .[0:7]' "$flake_dir/flake.lock" 2> /dev/null || true)"
    ndate="$(jq -r '.nodes.nixpkgs.locked.lastModified // 0' "$flake_dir/flake.lock" 2> /dev/null || echo 0)"
  fi
  target="$(readlink /nix/var/nix/profiles/system 2> /dev/null || true)" # system-42-link
  target="${target#system-}"
  gen="${target%-link}"
  [[ "$gen" =~ ^[0-9]+$ ]] || gen=0
  jq -cn --arg flake "$flake_dir" --arg branch "$branch" --arg commit "$commit" --argjson files "$files" \
    --arg fp "$(update_fingerprint)" --arg channel "$channel" --arg nrev "$nrev" --argjson ndate "$ndate" --argjson gen "$gen" \
    '{flake: $flake, branch: $branch, commit: $commit, dirty: ($files > 0), files: $files, fingerprint: $fp,
      channel: $channel, nixpkgs_rev: $nrev, nixpkgs_date: $ndate, generation: $gen}'
}

# Último escaneo guardado (null si no hay): la pantalla lo enseña al instante y
# decide si repetirlo comparando la huella y la edad.
update_cached_json() {
  local f="$state/update-check.json"
  if [ -s "$f" ]; then cat "$f"; else echo null; fi
}

# Comprobación sin interfaz, para la pantalla completa y los scripts: compila,
# compara y escribe el resultado como JSON. No aplica nada.
update_check_json() { # update_check_json lock
  local lock="$1" out diff kernel=false changes result
  if [ "$lock" = 1 ]; then nix flake update --flake "$flake_dir" > /dev/null 2>&1 || return "$EX_FAIL"; fi
  out="$(nix build --no-link --print-out-paths "$flake_dir#nixosConfigurations.$host.config.system.build.toplevel" 2> /dev/null)" || return "$EX_FAIL"
  if [ "$out" = "$(readlink -f /run/current-system)" ]; then
    result="$(jq -cn '{up_to_date: true, kernel: false, counts: {new: 0, updated: 0, removed: 0, changed: 0, config: 0}, changes: []}')"
  else
    UPDATE_OUT="$out"
    diff="$(update_compare)"
    if [ "$(readlink -f "$out/kernel")" != "$(readlink -f /run/booted-system/kernel)" ]; then kernel=true; fi
    diff_parse "$diff"
    changes="$(diff_json "$diff")"
    result="$(jq -cn --argjson k "$kernel" --argjson c "$changes" \
      --argjson n "$UI_DIFF_ADD" --argjson u "$UI_DIFF_UPD" --argjson r "$UI_DIFF_DEL" --argjson h "$UI_DIFF_CHG" --argjson f "$UI_DIFF_CFG" \
      '{up_to_date: false, kernel: $k, counts: {new: $n, updated: $u, removed: $r, changed: $h, config: $f}, changes: $c}')"
  fi
  # Se anota cuándo, con qué huella y si refrescó las entradas: es el escaneo guardado.
  result="$(jq -c --argjson at "$(printf '%(%s)T' -1)" --arg fp "$(update_fingerprint)" --argjson lock "$([ "$lock" = 1 ] && echo true || echo false)" \
    '. + {checked_at: $at, fingerprint: $fp, lock: $lock}' <<< "$result")"
  mkdir -p "$state" 2> /dev/null && printf '%s\n' "$result" > "$state/update-check.json" 2> /dev/null || true
  printf '%s\n' "$result"
}

cmd_update() {
  local yes=0 lock=1 check=0 json=0 status=0 cached=0 a
  for a in "$@"; do
    case "$a" in
      -y | --yes) yes=1 ;;
      --no-lock) lock=0 ;;
      --check) check=1 ;;
      --json) json=1 ;;
      --status) status=1 ;;
      --cached) cached=1 ;;
      *) die_code "$EX_USAGE" @err.unknown_option "$a" ;;
    esac
  done
  need_flake
  if [ "$status" = 1 ]; then update_status_json; return 0; fi
  if [ "$cached" = 1 ]; then update_cached_json; return 0; fi
  if [ "$json" = 1 ]; then update_check_json "$lock"; return $?; fi

  local sb out diff
  msg sb @update.step_build "$host"
  echo
  ui_intro @update.title "$host"
  if [ "$lock" = 1 ]; then ui_run @update.step_lock nix flake update --flake "$flake_dir" || return $?; fi
  ui_run "$sb" nix build --no-link --print-out-paths "$flake_dir#nixosConfigurations.$host.config.system.build.toplevel" || return $?
  out="$UI_OUT"
  if [ "$out" = "$(readlink -f /run/current-system)" ]; then
    ui_outro @update.up_to_date
    return 0
  fi
  UPDATE_OUT="$out"
  ui_run @update.step_compare update_compare || return $?
  diff="$UI_OUT"

  ui_section @update.title_changes
  if [ -z "$diff" ]; then
    ui_row info @update.only_config
  else
    ui_diff "$diff"
    if [ "$(readlink -f "$out/kernel")" != "$(readlink -f /run/booted-system/kernel)" ]; then
      ui_row warn @update.kernel_new
    fi
  fi
  if [ "$check" = 1 ]; then
    ui_outro @update.check_done
    return 0
  fi
  if [ "$yes" = 0 ] && ! ui_confirm @update.confirm; then
    if [ "$lock" = 1 ]; then ui_step info @update.lock_changed; fi
    ui_outro @update.cancelled
    return 0
  fi
  if ! maxor_sudo nixos-rebuild switch --flake "$flake_dir#$host"; then
    ui_outro
    ui_error @update.switch_failed @update.switch_cause @update.switch_hint
    return "$EX_FAIL"
  fi
  if [ "$(readlink -f /run/current-system/kernel)" != "$(readlink -f /run/booted-system/kernel)" ]; then
    ui_step warn @update.reboot
  fi
  ui_outro @update.done
}

# Generaciones del sistema (las 12 últimas) como JSON para la pantalla completa.
rollback_list_json() {
  nixos-rebuild list-generations --json 2> /dev/null \
    | jq -c '[.[:12][] | {generation: .generation, date: .date, nixos: .nixosVersion, kernel: .kernelVersion, current: .current}]' \
    || echo '[]'
}

cmd_rollback() {
  local yes=0 list=0 json=0 target="" a line
  for a in "$@"; do
    case "$a" in
      -y | --yes) yes=1 ;;
      --list) list=1 ;;
      --json) json=1 ;;
      -*) die_code "$EX_USAGE" @err.unknown_option "$a" ;;
      *) [[ "$a" =~ ^[0-9]+$ ]] || usage_error rollback; target="$a" ;;
    esac
  done
  if [ "$list" = 1 ]; then
    if [ "$json" = 1 ]; then rollback_list_json; return 0; fi
  fi
  echo
  ui_intro @rollback.title
  ui_section @rollback.sec_generations
  while IFS= read -r line; do ui_row info "$line"; done < <(nixos-rebuild list-generations 2> /dev/null | head -n 6 || true)
  [ "$list" = 1 ] && { ui_outro; return 0; }
  local confirm="@rollback.confirm" cmsg=()
  [ -z "$target" ] || { confirm="@rollback.confirm_to"; cmsg=("$target"); }
  if [ "$yes" = 0 ] && ! ui_confirm "$confirm" "${cmsg[@]}"; then
    ui_outro @err.cancelled
    return 0
  fi
  local ok=true
  if [ -n "$target" ]; then
    # una generación concreta: se elige en el perfil del sistema y se activa
    [ -e "/nix/var/nix/profiles/system-$target-link" ] || die_code "$EX_USAGE" @rollback.unknown "$target"
    # un solo sudo para las dos órdenes: con la contraseña por la entrada estándar solo se lee una vez
    maxor_sudo sh -c 'nix-env -p /nix/var/nix/profiles/system --switch-generation "$1" && /nix/var/nix/profiles/system/bin/switch-to-configuration switch' sh "$target" || ok=false
  else
    maxor_sudo nixos-rebuild switch --rollback || ok=false
  fi
  if [ "$ok" = false ]; then
    ui_outro
    ui_error @rollback.failed @update.switch_cause @update.switch_hint
    return "$EX_FAIL"
  fi
  ui_outro @rollback.done
}
