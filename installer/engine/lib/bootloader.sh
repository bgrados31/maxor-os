# ── Etapa bootloader: comprobar que la máquina arrancará ─────────────
# systemd-boot lo instala nixos-install (boot.nix); aquí solo se verifica que haya entradas y que,
# junto a Windows, la entrada de Windows siga ahí (systemd-boot la descubre en la ESP).
stage_bootloader() {
  if [ "$IN_DRY" = 1 ]; then
    printf 'DRYRUN: check that %s/boot/loader/entries has at least one entry\n' "$IN_ROOT"
    [ "$(ans .disk.strategy)" != alongside ] || printf 'DRYRUN: check that the existing boot entries (Windows) are still there\n'
    return 0
  fi
  local n
  n="$(find "$IN_ROOT/boot/loader/entries" -name '*.conf' 2> /dev/null | wc -l)"
  [ "$n" -ge 1 ] || in_die "$IN_EX_FAIL" "the boot loader has no entries: the system would not boot"
  [ -e "$IN_ROOT/boot/EFI/systemd/systemd-bootx64.efi" ] || in_die "$IN_EX_FAIL" "systemd-boot is not on the EFI partition"
}
