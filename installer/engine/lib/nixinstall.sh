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
  # Nix cree que no hay internet si no ve otra dirección IPv4 que 127.0.0.1, y entonces apaga TODOS los
  # sustituidores, también `auto`: sin red, nixos-install querría compilarlo todo. Una segunda dirección en lo
  # basta para que siga tomando lo del almacén local; se quita al terminar, salga bien o mal.
  local lo_alias=0 rc=0
  if ans_true '.network.offline' && ! ip -4 -o addr show dev lo 2> /dev/null | grep -q ' 127\.0\.0\.2/'; then
    in_run ip addr add 127.0.0.2/32 dev lo || in_die "$IN_EX_FAIL" "could not add a loopback address for the offline install"
    lo_alias=1
  fi
  in_run nixos-install "${args[@]}" || rc=$?
  if [ "$lo_alias" = 1 ]; then in_run ip addr del 127.0.0.2/32 dev lo || true; fi
  return "$rc"
}
