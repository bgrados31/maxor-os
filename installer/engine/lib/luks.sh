# ── Etapa luks: cifrado opcional de la raíz (LUKS2) ──────────────────
# La contraseña llega por un descriptor de archivo y solo vive en memoria: in_run_secret se la da a
# cryptsetup por la entrada estándar, así no aparece en el registro, en el simulacro ni en `ps`.
stage_luks() {
  if ! ans_true '.disk.encrypt.enabled'; then
    in_emit luks skip "encryption not requested"
    return 0
  fi
  local root
  root="$(dev_get root)"
  in_run_secret "$IN_SECRET" cryptsetup luksFormat --type luks2 --batch-mode --key-file=- "$root"
  in_run_secret "$IN_SECRET" cryptsetup open --key-file=- "$root" maxor-root
}
