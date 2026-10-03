# ── Diagnóstico: doctor ──────────────────────────────────────────────
cmd_doctor() {
  local fails=0 warns=0
  ok() { ui_row ok "$*"; }
  warn() { ui_row warn "$*"; warns=$((warns + 1)); }
  bad() { ui_row bad "$*"; fails=$((fails + 1)); }

  echo
  ui_open "maxor · doctor · $host"

  ui_section "Sistema"
  ok "$(grep -m1 '^PRETTY_NAME=' /etc/os-release | cut -d= -f2 | tr -d '"') · kernel $(uname -r)"
  local nf nu
  nf="$(systemctl --failed --no-legend 2> /dev/null | wc -l)"
  nu="$(systemctl --user --failed --no-legend 2> /dev/null | wc -l)"
  if [ "$nf" = 0 ]; then ok "sin servicios del sistema fallidos"; else bad "$nf servicio(s) del sistema fallido(s): systemctl --failed"; fi
  if [ "$nu" = 0 ]; then ok "sin servicios de usuario fallidos"; else bad "$nu servicio(s) de usuario fallido(s): systemctl --user --failed"; fi
  if [ -e /run/booted-system/kernel ] && [ "$(readlink -f /run/booted-system/kernel)" != "$(readlink -f /run/current-system/kernel)" ]; then
    warn "el kernel instalado no es el que corre: reinicia para usarlo"
  else
    ok "el kernel en uso es el instalado"
  fi

  ui_section "Sesión"
  if [ "${XDG_SESSION_TYPE:-}" = "wayland" ]; then ok "sesión Wayland"; else warn "la sesión no es Wayland (XDG_SESSION_TYPE=${XDG_SESSION_TYPE:-vacío})"; fi
  if [ "${XDG_CURRENT_DESKTOP:-}" = "Hyprland" ]; then ok "escritorio Hyprland"; else warn "XDG_CURRENT_DESKTOP=${XDG_CURRENT_DESKTOP:-vacío} (¿desde una TTY?)"; fi
  local u
  for u in dms hypridle xdg-desktop-portal xdg-desktop-portal-hyprland; do
    if systemctl --user is-active --quiet "$u"; then ok "$u activo"; else bad "$u no está activo: systemctl --user status $u"; fi
  done
  if [ -f /etc/pam.d/hyprlock ]; then
    ok "hyprlock tiene servicio PAM: podrás desbloquear"
  else
    bad "falta /etc/pam.d/hyprlock: no podrías desbloquear la pantalla"
  fi

  ui_section "Gráficos"
  local gpus line
  gpus="$(lspci 2> /dev/null | grep -E 'VGA|3D' | sed 's/^[^ ]* //; s/^[A-Za-z0-9 ]*controller: //' || true)"
  if [ -n "$gpus" ]; then
    while IFS= read -r line; do ok "$line"; done <<< "$gpus"
  else
    warn "no se detectó ninguna GPU con lspci"
  fi
  if grep -q -i nvidia <<< "$gpus"; then
    if command -v nvidia-offload > /dev/null; then ok "nvidia-offload disponible (PRIME offload)"; else warn "hay NVIDIA pero falta nvidia-offload"; fi
    if command -v nvidia-smi > /dev/null && nvidia-smi -L > /dev/null 2>&1; then ok "el controlador NVIDIA responde"; else warn "nvidia-smi no responde (¿driver sin cargar?)"; fi
  fi

  ui_section "Arranque y disco"
  local m use
  for m in /boot /efi; do
    if findmnt -n "$m" > /dev/null 2>&1; then
      use="$(df --output=pcent "$m" | tail -n1 | tr -dc '0-9')"
      if [ "${use:-0}" -ge 90 ]; then bad "$m al ${use}%: borra generaciones viejas (nix-collect-garbage -d)"
      elif [ "${use:-0}" -ge 80 ]; then warn "$m al ${use}%"
      else ok "$m montado (${use}% usado)"; fi
    elif [ "$m" = /boot ]; then
      bad "/boot no está montado"
    fi
  done
  use="$(df --output=pcent / | tail -n1 | tr -dc '0-9')"
  if [ "${use:-0}" -ge 90 ]; then warn "/ al ${use}%"; else ok "/ al ${use}%"; fi

  ui_section "Identidad"
  local f fams
  fams="$(fc-list : family 2> /dev/null || true)"
  for f in "Figtree" "Red Hat Mono" "Krona One"; do
    if grep -q -i -F "$f" <<< "$fams"; then ok "fuente $f instalada"; else warn "falta la fuente $f"; fi
  done
  if [ -f "$state/current" ]; then
    ok "tema activo: $(cat "$state/current")"
    if [ -f "$cfg/current/dms-theme.json" ]; then ok "tema de DMS generado"; else warn "falta el tema de DMS: aplica un tema otra vez"; fi
  else
    warn "ningún tema aplicado todavía: maxor theme apply sakura"
  fi

  ui_section "Configuración"
  if [ -f "$flake_dir/flake.nix" ]; then
    ok "flake en ${flake_dir/#$HOME/~}"
    if git -C "$flake_dir" rev-parse --git-dir > /dev/null 2>&1; then
      if [ -n "$(git -C "$flake_dir" status --porcelain)" ]; then warn "hay cambios sin commit en el repositorio"; else ok "repositorio sin cambios pendientes"; fi
    fi
  else
    warn "no encuentro el flake en $flake_dir (MAXOR_FLAKE)"
  fi
  local hwf="$flake_dir/hosts/$host/hardware.json" cur sig
  sig='[.cpu.vendor, .laptop, .virt, ([.gpus[] | .vendor + .id + .bus] | sort)]'
  if [ -f "$hwf" ]; then
    cur="$(hw_detect)"
    if [ "$(jq -c "$sig" <<< "$cur")" = "$(jq -c "$sig" "$hwf")" ]; then
      ok "hardware.json coincide con este equipo"
    else
      warn "el hardware cambió respecto a hardware.json: maxor hardware detect --write"
    fi
  elif [ -f "$flake_dir/flake.nix" ]; then
    warn "sin hardware.json: maxor hardware detect --write"
  fi

  ui_line ""
  ui_close
  echo
  if [ "$fails" -gt 0 ]; then
    printf ' %s  %d problema(s) y %d aviso(s)\n' "$(ui_pill bad "REVISAR")" "$fails" "$warns"
    return 1
  elif [ "$warns" -gt 0 ]; then
    printf ' %s  funciona, con %d aviso(s)\n' "$(ui_pill warn "AVISOS")" "$warns"
  else
    printf ' %s  todo en orden\n' "$(ui_pill ok "OK")"
  fi
}
