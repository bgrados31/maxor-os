# ── Actualización: update y rollback ─────────────────────────────────
maxor_cmd update system ""
maxor_cmd rollback system ""

# Compara el resultado compilado (UPDATE_OUT) con el sistema en marcha.
update_compare() {
  nix store diff-closures /run/current-system "$UPDATE_OUT" 2> /dev/null || true
}

cmd_update() {
  local yes=0 lock=1 a
  for a in "$@"; do
    case "$a" in
      -y | --yes) yes=1 ;;
      --no-lock) lock=0 ;;
      *) die_code "$EX_USAGE" @err.unknown_option "$a" ;;
    esac
  done
  need_flake

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
  if [ "$yes" = 0 ] && ! ui_confirm @update.confirm; then
    if [ "$lock" = 1 ]; then ui_step info @update.lock_changed; fi
    ui_outro @update.cancelled
    return 0
  fi
  if ! sudo nixos-rebuild switch --flake "$flake_dir#$host"; then
    ui_outro
    ui_error @update.switch_failed @update.switch_cause @update.switch_hint
    return "$EX_FAIL"
  fi
  if [ "$(readlink -f /run/current-system/kernel)" != "$(readlink -f /run/booted-system/kernel)" ]; then
    ui_step warn @update.reboot
  fi
  ui_outro @update.done
}

cmd_rollback() {
  local yes=0 line
  if [ "${1:-}" = "-y" ]; then yes=1; fi
  echo
  ui_intro @rollback.title
  ui_section @rollback.sec_generations
  while IFS= read -r line; do ui_row info "$line"; done < <(nixos-rebuild list-generations 2> /dev/null | head -n 6 || true)
  if [ "$yes" = 0 ] && ! ui_confirm @rollback.confirm; then
    ui_outro @err.cancelled
    return 0
  fi
  if ! sudo nixos-rebuild switch --rollback; then
    ui_outro
    ui_error @rollback.failed @update.switch_cause @update.switch_hint
    return "$EX_FAIL"
  fi
  ui_outro @rollback.done
}
