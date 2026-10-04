# ── Etapa install: nixos-install desde el flake de la máquina ────────
stage_install() {
  local dir host args=()
  dir="$(host_dir)"
  host="$(ans .machine.hostname)"
  args=(--root "$IN_ROOT" --flake "$dir#$host" --no-root-passwd --no-channel-copy)
  # Sin red: solo lo que ya está en el disco del instalador. nixos-install construye en un almacén aparte (/mnt)
  # y toma lo ya compilado del almacén local por el sustituidor `auto`; dejar `substituters` vacío lo borraría
  # también, así que se deja SOLO ese (sin cache.nixos.org).
  if ans_true '.network.offline'; then
    args+=(--option substituters "auto?trusted=1")
    # NixOS marca toplevel, etc, home-manager-path… con allowSubstitutes = false: Nix las reconstruye siempre, y
    # para eso hacen falta herramientas de compilación que no están en el disco. Con esto las toma ya hechas.
    args+=(--option always-allow-substitutes true)
  else
    # Con red, lo que ya trae el medio (los paquetes terminados de Maxor OS) se toma del almacén local en vez de
    # bajarlo otra vez; lo demás viene de la caché de siempre.
    args+=(--option extra-substituters "auto?trusted=1")
  fi
  # Opciones extra para nixos-install (instalaciones avanzadas y pruebas), por ejemplo --impure.
  if [ -n "${MAXOR_INSTALL_EXTRA_ARGS:-}" ]; then
    local extra
    read -ra extra <<< "$MAXOR_INSTALL_EXTRA_ARGS"
    args+=("${extra[@]}")
  fi
  in_run nixos-install "${args[@]}"
}
