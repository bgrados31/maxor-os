# ── Etapa install: nixos-install desde el flake de la máquina ────────
stage_install() {
  local dir host args=()
  dir="$(host_dir)"
  host="$(ans .machine.hostname)"
  args=(--root "$IN_ROOT" --flake "$dir#$host" --no-root-passwd)
  # sin red: solo lo que ya está en el disco de la ISO
  if ans_true '.network.offline'; then args+=(--option substituters ""); fi
  in_run nixos-install "${args[@]}"
}
