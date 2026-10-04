# ── Etapa finish: contraseña, red, registro y desmontar ──────────────
IN_NM_DIR="${MAXOR_INSTALL_NM_DIR:-/etc/NetworkManager/system-connections}"

stage_finish() {
  local user hash dir
  user="$(ans .user.name)"
  hash="$(ans .user.password_hash)"
  dir="/home/$user/nixos-config"

  # La contraseña se fija aquí, no en la configuración: un hash en un archivo de Nix acabaría en
  # /nix/store, que lee cualquiera. Entra por la entrada estándar de chpasswd, nunca por argumentos.
  in_run_secret "$user:$hash" nixos-enter --root "$IN_ROOT" -c 'chpasswd --encrypted'

  # La carpeta de configuración es del usuario (el usuario ya existe: lo creó la activación).
  in_run nixos-enter --root "$IN_ROOT" -c "chown -R $user:users $dir"

  # La red a la que estaba conectada la ISO (Wi-Fi) sigue en la máquina instalada.
  if ! ans_true '.network.offline' && [ -d "$IN_NM_DIR" ]; then
    in_run mkdir -p "$IN_ROOT/etc/NetworkManager/system-connections"
    in_run cp -a "$IN_NM_DIR/." "$IN_ROOT/etc/NetworkManager/system-connections/"
  fi

  in_run mkdir -p "$IN_ROOT/var/log"
  [ "$IN_DRY" = 1 ] || cp "$IN_LOG" "$IN_ROOT/var/log/maxor-install.log" 2> /dev/null || true

  in_run umount -R "$IN_ROOT"
  if ans_true '.disk.encrypt.enabled'; then in_run cryptsetup close maxor-root; fi
}
