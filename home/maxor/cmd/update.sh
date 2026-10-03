# ── Actualización: update y rollback ─────────────────────────────────
maxor_cmd update system ""
maxor_cmd rollback system ""

update_step_lock() { nix flake update --flake "$flake_dir"; }
update_step_build() {
  nix build --no-link --print-out-paths "$flake_dir#nixosConfigurations.$host.config.system.build.toplevel"
}
# Lee el resultado del paso de compilación (UI_OUTS[índice]) y lo compara.
update_step_compare() {
  nix store diff-closures /run/current-system "${UI_OUTS[$UPDATE_BUILD_IDX]}" 2> /dev/null || true
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

  local steps=() sb sc title out diff
  msg sb @update.step_build "$host"
  msg sc @update.step_compare
  msg title @update.title "$host"
  UPDATE_BUILD_IDX=0
  if [ "$lock" = 1 ]; then
    steps+=(@update.step_lock update_step_lock)
    UPDATE_BUILD_IDX=1
  fi
  steps+=("$sb" update_step_build "$sc" update_step_compare)

  echo
  ui_pipeline "$title" "${steps[@]}" || return $?
  out="${UI_OUTS[$UPDATE_BUILD_IDX]}"
  diff="${UI_OUTS[$((UPDATE_BUILD_IDX + 1))]}"

  if [ "$out" = "$(readlink -f /run/current-system)" ]; then
    ui_say ok @update.up_to_date
    return 0
  fi

  echo
  ui_open @update.title_changes
  ui_line ""
  if [ -z "$diff" ]; then
    ui_row info @update.only_config
  else
    ui_diff "$diff"
    if [ "$(readlink -f "$out/kernel")" != "$(readlink -f /run/booted-system/kernel)" ]; then
      ui_line ""
      ui_row warn @update.kernel_new
    fi
  fi
  ui_line ""
  ui_close
  echo
  if [ "$yes" = 0 ] && ! ui_confirm @update.confirm; then
    ui_say info @update.cancelled
    if [ "$lock" = 1 ]; then ui_say info @update.lock_changed; fi
    return 0
  fi
  if ! sudo nixos-rebuild switch --flake "$flake_dir#$host"; then
    ui_error @update.switch_failed @update.switch_cause @update.switch_hint
    return "$EX_FAIL"
  fi
  ui_say ok @update.done
  if [ "$(readlink -f /run/current-system/kernel)" != "$(readlink -f /run/booted-system/kernel)" ]; then
    ui_say warn @update.reboot
  fi
}

cmd_rollback() {
  local yes=0 line
  if [ "${1:-}" = "-y" ]; then yes=1; fi
  echo
  ui_open @rollback.title
  ui_line ""
  while IFS= read -r line; do ui_row info "$line"; done < <(nixos-rebuild list-generations 2> /dev/null | head -n 6 || true)
  ui_line ""
  ui_close
  echo
  if [ "$yes" = 0 ] && ! ui_confirm @rollback.confirm; then
    ui_say info @err.cancelled
    return 0
  fi
  if ! sudo nixos-rebuild switch --rollback; then
    ui_error @rollback.failed @update.switch_cause @update.switch_hint
    return "$EX_FAIL"
  fi
  ui_say ok @rollback.done
}
