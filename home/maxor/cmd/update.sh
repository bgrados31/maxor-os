# ── Comandos de actualización: update y rollback ─────────────────────
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
  echo
  if [ "$lock" = 1 ]; then
    ui_run "Actualizando las entradas del flake" nix flake update --flake "$flake_dir" || exit 1
  fi
  ui_run "Compilando $host (todavía no se aplica nada)" \
    nix build --no-link --print-out-paths "$flake_dir#nixosConfigurations.$host.config.system.build.toplevel" || exit 1
  local out="$UI_OUT"
  if [ "$out" = "$(readlink -f /run/current-system)" ]; then
    ui_say ok "El sistema ya está al día"
    return 0
  fi

  local diff n=0 line
  diff="$(nix store diff-closures /run/current-system "$out" 2> /dev/null || true)"
  echo
  ui_open "maxor · cambios"
  ui_line ""
  if [ -z "$diff" ]; then
    ui_row info "Solo cambia la configuración, ningún paquete"
  else
    local plain upd add del
    plain="$(sed 's/\x1b\[[0-9;]*m//g' <<< "$diff")"
    add="$(grep -c ': ∅ → ' <<< "$plain" || true)"
    del="$(grep -c ' → ∅' <<< "$plain" || true)"
    upd="$(grep -c '[0-9] → [0-9]' <<< "$plain" || true)"
    ui_row ok "$upd actualizado(s) · $add nuevo(s) · $del eliminado(s)"
    if [ "$(readlink -f "$out/kernel")" != "$(readlink -f /run/booted-system/kernel)" ]; then
      ui_row warn "incluye un kernel nuevo: tendrás que reiniciar"
    fi
    ui_line ""
    while IFS= read -r line; do
      n=$((n + 1))
      if [ "$n" -le 18 ]; then ui_row info "$line"; fi
    done <<< "$diff"
    if [ "$n" -gt 18 ]; then ui_row info "… y $((n - 18)) más"; fi
  fi
  ui_line ""
  ui_close
  echo
  if [ "$yes" = 0 ] && ! ui_confirm "¿Aplicar ahora?"; then
    ui_say info "Cancelado. No se aplicó nada."
    if [ "$lock" = 1 ]; then ui_say info "flake.lock sí se actualizó: revísalo con git diff flake.lock"; fi
    return 0
  fi
  sudo nixos-rebuild switch --flake "$flake_dir#$host"
  ui_say ok "Sistema actualizado"
  if [ "$(readlink -f /run/current-system/kernel)" != "$(readlink -f /run/booted-system/kernel)" ]; then
    ui_say warn "Reinicia para usar el kernel nuevo"
  fi
}

cmd_rollback() {
  local yes=0 line
  if [ "${1:-}" = "-y" ]; then yes=1; fi
  echo
  ui_open "maxor · generaciones"
  ui_line ""
  while IFS= read -r line; do ui_row info "$line"; done < <(nixos-rebuild list-generations 2> /dev/null | head -n 6 || true)
  ui_line ""
  ui_close
  echo
  if [ "$yes" = 0 ] && ! ui_confirm "¿Volver a la generación anterior?"; then
    ui_say info "Cancelado."
    return 0
  fi
  sudo nixos-rebuild switch --rollback
  ui_say ok "Vuelto a la generación anterior"
}

