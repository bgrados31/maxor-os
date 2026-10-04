# ── Diagnóstico: doctor ──────────────────────────────────────────────
maxor_cmd doctor system ""

# Qué comando arregla (o ayuda a ver) cada comprobación. Va en el --json para que
# la pantalla completa lo ofrezca: la lógica está aquí y no se duplica.
#   doctor_fix id [argumentos del mensaje…]  →  imprime «comando», «1» si pide confirmación y
#   el tipo: «fix» (arregla algo) o «inspect» (solo enseña información para que decidas)
doctor_fix() {
  local id="$1"
  shift
  case "$id" in
    hw_changed | hw_missing) printf 'maxor hardware detect --write\t0\tfix' ;;
    git_dirty) printf "git -C '%s' status\t0\tinspect" "$flake_dir" ;;
    theme_none) printf 'maxor theme apply maxor-dark\t0\tfix' ;;
    dms_warn) printf 'maxor theme apply %s\t0\tfix' "$(cat "$state/current" 2> /dev/null || echo maxor-dark)" ;;
    flatpak_env) printf 'maxor apps repair\t0\tfix' ;;
    sys_bad) printf 'systemctl --failed\t0\tinspect' ;;
    usr_bad) printf 'systemctl --user --failed\t0\tinspect' ;;
    unit_bad) printf 'systemctl --user status %s\t0\tinspect' "${1:-dms}" ;;
    disk_full) printf 'nix-collect-garbage -d\t1\tfix' ;;
    *) printf '\t0\t' ;;
  esac
}

cmd_doctor() {
  local json=0 fails=0 warns=0 rows=() cur_title=""
  case "${1:-}" in
    --json) json=1 ;;
    "") ;;
    *) usage_error doctor ;;
  esac
  # Cada comprobación se anota (para --json) y, si no es --json, se dibuja.
  sec() { msg cur_title "$@"; if [ "$json" = 0 ]; then ui_section "$@"; fi; }
  rec() {
    local lvl="$1" key="$2" t id="" fix
    shift
    msg t "$@"
    if [[ "$key" == @doctor.* ]]; then id="${key#@doctor.}"; fi
    # los argumentos del mensaje (sin la clave) sirven también para el arreglo
    fix=$'\t0\t'
    if [ "$json" = 1 ] && [ "$lvl" != ok ]; then fix="$(doctor_fix "$id" "${@:2}")"; fi
    rows+=("$cur_title"$'\t'"$lvl"$'\t'"$t"$'\t'"$id"$'\t'"$fix")
    if [ "$json" = 0 ]; then ui_row "$lvl" "$t"; fi
  }
  ok() { rec ok "$@"; }
  warn() { rec warn "$@"; warns=$((warns + 1)); }
  bad() { rec bad "$@"; fails=$((fails + 1)); }

  if [ "$json" = 0 ]; then
    echo
    ui_intro @doctor.title "$host"
  fi

  sec @doctor.sec_system
  ok "$(os_pretty) · kernel $(uname -r)"
  local nf nu
  nf="$(systemctl --failed --no-legend 2> /dev/null | wc -l || true)"
  nu="$(systemctl --user --failed --no-legend 2> /dev/null | wc -l || true)"
  if [ "$nf" = 0 ]; then ok @doctor.sys_ok; else bad @doctor.sys_bad "$nf"; fi
  if [ "$nu" = 0 ]; then ok @doctor.usr_ok; else bad @doctor.usr_bad "$nu"; fi
  if [ -e /run/booted-system/kernel ] && [ "$(readlink -f /run/booted-system/kernel)" != "$(readlink -f /run/current-system/kernel)" ]; then
    warn @doctor.kernel_mismatch
  else
    ok @doctor.kernel_ok
  fi

  sec @doctor.sec_session
  local empty
  msg empty @doctor.empty
  if [ "${XDG_SESSION_TYPE:-}" = "wayland" ]; then ok @doctor.wayland_ok; else warn @doctor.wayland_warn "${XDG_SESSION_TYPE:-$empty}"; fi
  if [ "${XDG_CURRENT_DESKTOP:-}" = "Hyprland" ]; then ok @doctor.desktop_ok; else warn @doctor.desktop_warn "${XDG_CURRENT_DESKTOP:-$empty}"; fi
  local u
  for u in dms hypridle xdg-desktop-portal xdg-desktop-portal-hyprland; do
    if systemctl --user is-active --quiet "$u"; then ok @doctor.unit_ok "$u"; else bad @doctor.unit_bad "$u" "$u"; fi
  done
  local hidden
  hidden="$(app_flatpak_hidden | wc -l)"
  if [ "$hidden" -gt 0 ]; then warn @doctor.flatpak_env "$hidden"; fi
  if [ -f /etc/pam.d/hyprlock ]; then
    ok @doctor.pam_ok
  else
    bad @doctor.pam_bad
  fi

  sec @doctor.sec_graphics
  local gpus line
  gpus="$(lspci 2> /dev/null | grep -E 'VGA|3D' | sed 's/^[^ ]* //; s/^[A-Za-z0-9 ]*controller: //' || true)"
  if [ -n "$gpus" ]; then
    while IFS= read -r line; do ok "$line"; done <<< "$gpus"
  else
    warn @doctor.no_gpu
  fi
  if grep -q -i nvidia <<< "$gpus"; then
    if command -v nvidia-offload > /dev/null; then ok @doctor.offload_ok; else warn @doctor.offload_warn; fi
    if command -v nvidia-smi > /dev/null && nvidia-smi -L > /dev/null 2>&1; then ok @doctor.nv_ok; else warn @doctor.nv_warn; fi
  fi

  sec @doctor.sec_boot
  local m use
  for m in /boot /efi; do
    if findmnt -n "$m" > /dev/null 2>&1; then
      use="$(df --output=pcent "$m" | tail -n1 | tr -dc '0-9')"
      if [ "${use:-0}" -ge 90 ]; then bad @doctor.disk_full "$m" "$use"
      elif [ "${use:-0}" -ge 80 ]; then warn @doctor.disk_warn "$m" "$use"
      else ok @doctor.disk_ok "$m" "$use"; fi
    elif [ "$m" = /boot ] && [ -d /sys/firmware/efi ]; then
      # solo un equipo UEFI necesita /boot montado; una VM con arranque directo no tiene
      bad @doctor.boot_unmounted
    fi
  done
  use="$(df --output=pcent / | tail -n1 | tr -dc '0-9')"
  if [ "${use:-0}" -ge 90 ]; then warn @doctor.root "$use"; else ok @doctor.root "$use"; fi

  sec @doctor.sec_identity
  local f fams
  fams="$(fc-list : family 2> /dev/null || true)"
  for f in "Figtree" "Red Hat Mono" "Cinzel"; do
    if grep -q -i -F "$f" <<< "$fams"; then ok @doctor.font_ok "$f"; else warn @doctor.font_warn "$f"; fi
  done
  if [ -f "$state/current" ]; then
    ok @doctor.theme_ok "$(cat "$state/current")"
    if [ -f "$cfg/current/dms-theme.json" ]; then ok @doctor.dms_ok; else warn @doctor.dms_warn; fi
  else
    warn @doctor.theme_none
  fi

  sec @doctor.sec_config
  if [ -f "$flake_dir/flake.nix" ]; then
    ok @doctor.flake_ok "${flake_dir/#$HOME/~}"
    if git -C "$flake_dir" rev-parse --git-dir > /dev/null 2>&1; then
      if [ -n "$(git -C "$flake_dir" status --porcelain)" ]; then warn @doctor.git_dirty; else ok @doctor.git_clean; fi
    fi
  else
    warn @doctor.flake_missing "$flake_dir"
  fi
  local hwf="$flake_dir/hosts/$host/hardware.json" now sig
  sig='[.cpu.vendor, .laptop, .virt, ([.gpus[] | .vendor + .id + .bus] | sort)]'
  if [ -f "$hwf" ]; then
    now="$(hw_detect)"
    if [ "$(jq -c "$sig" <<< "$now")" = "$(jq -c "$sig" "$hwf")" ]; then
      ok @doctor.hw_ok
    else
      warn @doctor.hw_changed
    fi
  elif [ -f "$flake_dir/flake.nix" ]; then
    warn @doctor.hw_missing
  fi

  if [ "$json" = 1 ]; then
    printf '%s\n' "${rows[@]}" | jq -R -s -c --argjson f "$fails" --argjson w "$warns" '
      split("\n") | map(select(length > 0) | split("\t")) as $r
      | {ok: ($f == 0), fails: $f, warns: $w,
         groups: (reduce $r[] as $x ([];
           if length > 0 and .[-1].title == $x[0]
           then .[-1].items += [{level: $x[1], text: $x[2], id: $x[3], fix: (if $x[4] == "" then null else $x[4] end), confirm: ($x[5] == "1"), kind: ($x[6] // "")}]
           else . + [{title: $x[0], items: [{level: $x[1], text: $x[2], id: $x[3], fix: (if $x[4] == "" then null else $x[4] end), confirm: ($x[5] == "1"), kind: ($x[6] // "")}]}] end))}'
    [ "$fails" -eq 0 ]
    return
  fi
  local txt
  if [ "$fails" -gt 0 ]; then
    msg txt @doctor.sum_bad "$fails" "$warns"
    ui_outro "${E_BAD}${txt}${E_RST}"
    echo
    return 1
  elif [ "$warns" -gt 0 ]; then
    msg txt @doctor.sum_warn "$warns"
    ui_outro "${E_WARN}${txt}${E_RST}"
  else
    msg txt @doctor.sum_ok
    ui_outro "${E_OK}${txt}${E_RST}"
  fi
  echo
}
