# ── Sistema: update, rollback, doctor ────────────────────────────────
flake_dir="${MAXOR_FLAKE:-$HOME/nixos-config}"
host="${MAXOR_HOST:-$(hostname)}"

need_flake() {
  [ -f "$flake_dir/flake.nix" ] || die "no encuentro el flake en $flake_dir (usa MAXOR_FLAKE=/ruta)"
}

confirm() { # confirm "pregunta"  →  0 si el usuario acepta
  local r
  read -r -p "$1 [s/N] " r
  case "$r" in s | S | y | Y) return 0 ;; *) return 1 ;; esac
}

cmd_update() {
  local yes=0 lock=1 a
  for a in "$@"; do
    case "$a" in
      -y | --yes) yes=1 ;;
      --no-lock) lock=0 ;;
      *) die "opción desconocida: $a" ;;
    esac
  done
  need_flake
  if [ "$lock" = 1 ]; then
    echo "→ actualizando las entradas del flake (flake.lock)…"
    nix flake update --flake "$flake_dir"
  fi
  echo "→ compilando $host (todavía no se aplica nada)…"
  local out
  out="$(nix build --no-link --print-out-paths "$flake_dir#nixosConfigurations.$host.config.system.build.toplevel")"
  if [ "$out" = "$(readlink -f /run/current-system)" ]; then
    echo "El sistema ya está al día."
    return 0
  fi
  echo
  echo "Cambios respecto al sistema actual:"
  nix store diff-closures /run/current-system "$out" || true
  echo
  if [ "$yes" = 0 ] && ! confirm "¿Aplicar ahora?"; then
    echo "Cancelado. No se aplicó nada (flake.lock sí cambió; revísalo con git)."
    return 0
  fi
  sudo nixos-rebuild switch --flake "$flake_dir#$host"
}

cmd_rollback() {
  local yes=0
  [ "${1:-}" = "-y" ] && yes=1
  echo "Generaciones recientes:"
  nixos-rebuild list-generations 2>/dev/null | head -n 6 || true
  echo
  if [ "$yes" = 0 ] && ! confirm "¿Volver a la generación anterior?"; then
    echo "Cancelado."
    return 0
  fi
  sudo nixos-rebuild switch --rollback
}

cmd_doctor() {
  local fails=0 warns=0 c_ok="" c_warn="" c_bad="" c_off=""
  if [ -t 1 ]; then c_ok=$'\e[32m'; c_warn=$'\e[33m'; c_bad=$'\e[31m'; c_off=$'\e[0m'; fi
  ok() { printf '  %s✓%s %s\n' "$c_ok" "$c_off" "$*"; }
  warn() { printf '  %s!%s %s\n' "$c_warn" "$c_off" "$*"; warns=$((warns + 1)); }
  bad() { printf '  %s✗%s %s\n' "$c_bad" "$c_off" "$*"; fails=$((fails + 1)); }
  section() { printf '\n%s\n' "$1"; }

  section "Sistema"
  ok "$(grep -m1 '^PRETTY_NAME=' /etc/os-release | cut -d= -f2 | tr -d '"') · kernel $(uname -r)"
  local nf nu
  nf="$(systemctl --failed --no-legend 2>/dev/null | wc -l)"
  nu="$(systemctl --user --failed --no-legend 2>/dev/null | wc -l)"
  [ "$nf" = 0 ] && ok "sin servicios del sistema fallidos" || bad "$nf servicio(s) del sistema fallido(s): systemctl --failed"
  [ "$nu" = 0 ] && ok "sin servicios de usuario fallidos" || bad "$nu servicio(s) de usuario fallido(s): systemctl --user --failed"
  if [ -e /run/booted-system/kernel ] && [ "$(readlink -f /run/booted-system/kernel)" != "$(readlink -f /run/current-system/kernel)" ]; then
    warn "el kernel instalado no es el que está corriendo: reinicia para usarlo"
  else
    ok "el kernel en uso es el instalado"
  fi

  section "Sesión"
  [ "${XDG_SESSION_TYPE:-}" = "wayland" ] && ok "sesión Wayland" || warn "la sesión no es Wayland (XDG_SESSION_TYPE=${XDG_SESSION_TYPE:-vacío})"
  [ "${XDG_CURRENT_DESKTOP:-}" = "Hyprland" ] && ok "escritorio Hyprland" || warn "XDG_CURRENT_DESKTOP=${XDG_CURRENT_DESKTOP:-vacío} (¿lo ejecutas desde una TTY?)"
  local u
  for u in dms hypridle xdg-desktop-portal xdg-desktop-portal-hyprland; do
    if systemctl --user is-active --quiet "$u"; then ok "$u activo"; else bad "$u no está activo: systemctl --user status $u"; fi
  done
  if [ -f /etc/pam.d/hyprlock ]; then
    ok "hyprlock tiene su servicio PAM (podrás desbloquear)"
  else
    bad "falta /etc/pam.d/hyprlock: no podrías desbloquear la pantalla"
  fi

  section "Gráficos"
  local gpus
  gpus="$(lspci 2>/dev/null | grep -E 'VGA|3D' | sed 's/^[^ ]* //' || true)"
  if [ -n "$gpus" ]; then
    while IFS= read -r line; do ok "$line"; done <<< "$gpus"
  else
    warn "no se detectó ninguna GPU con lspci"
  fi
  if grep -q -i nvidia <<< "$gpus"; then
    if command -v nvidia-offload > /dev/null; then ok "nvidia-offload disponible (PRIME offload)"; else warn "hay NVIDIA pero falta nvidia-offload"; fi
    if command -v nvidia-smi > /dev/null && nvidia-smi -L > /dev/null 2>&1; then ok "el controlador NVIDIA responde"; else warn "nvidia-smi no responde (¿driver sin cargar?)"; fi
  fi

  section "Arranque y disco"
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

  section "Identidad"
  local f fams
  fams="$(fc-list : family 2> /dev/null || true)"
  for f in "Figtree" "Red Hat Mono" "Krona One"; do
    if grep -q -i -F "$f" <<< "$fams"; then ok "fuente $f instalada"; else warn "falta la fuente $f"; fi
  done
  if [ -f "$state/current" ]; then
    ok "tema activo: $(cat "$state/current")"
    if [ -f "$cfg/current/dms-theme.json" ]; then ok "tema de DMS generado"; else warn "falta $cfg/current/dms-theme.json: aplica un tema otra vez"; fi
  else
    warn "ningún tema aplicado todavía: maxor theme apply sakura"
  fi

  section "Configuración"
  if [ -f "$flake_dir/flake.nix" ]; then
    ok "flake en $flake_dir"
    if git -C "$flake_dir" rev-parse --git-dir > /dev/null 2>&1; then
      if [ -n "$(git -C "$flake_dir" status --porcelain)" ]; then warn "hay cambios sin commit en $flake_dir"; else ok "repositorio sin cambios pendientes"; fi
    fi
  else
    warn "no encuentro el flake en $flake_dir (MAXOR_FLAKE)"
  fi

  echo
  if [ "$fails" -gt 0 ]; then
    printf '%s%d problema(s)%s y %d aviso(s).\n' "$c_bad" "$fails" "$c_off" "$warns"
    return 1
  elif [ "$warns" -gt 0 ]; then
    printf '%sTodo funciona%s, con %d aviso(s).\n' "$c_warn" "$c_off" "$warns"
  else
    printf '%sTodo en orden.%s\n' "$c_ok" "$c_off"
  fi
}
