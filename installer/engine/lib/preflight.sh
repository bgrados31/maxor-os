# ── Etapa preflight: ¿se puede instalar aquí? ────────────────────────
# Reúne TODOS los motivos por los que no, no solo el primero, y para antes de tocar nada.
# Cada comprobación usa herramientas del sistema (lsblk, findmnt, blockdev…), así que las pruebas
# pueden sustituirlas por simulaciones.

# Rutas del sistema que las pruebas pueden sustituir.
PF_EFI="${MAXOR_INSTALL_EFI:-/sys/firmware/efi}"
PF_MEMINFO="${MAXOR_INSTALL_MEMINFO:-/proc/meminfo}"

PF_NEED_TOOLS=(sfdisk wipefs lsblk findmnt blockdev mkfs.fat mount umount nixos-install nixos-enter nixos-generate-config jq git curl)

# pf_fail MENSAJE → anota un motivo.
PF_FAILS=()
pf_fail() { PF_FAILS+=("$1"); }

preflight_checks() {
  local disk strategy size need live parent mounted fs min
  disk="$(ans .disk.device)"
  strategy="$(ans .disk.strategy)"
  fs="$(ans .disk.filesystem)"

  [ "$(id -u)" = 0 ] || pf_fail "the installer must run as root"
  [ -d "$PF_EFI" ] || pf_fail "this machine did not boot in UEFI mode; legacy BIOS is not supported yet"

  local t
  for t in "${PF_NEED_TOOLS[@]}"; do
    command -v "$t" > /dev/null 2>&1 || pf_fail "the tool $t is missing from the live system"
  done
  [ "$fs" != btrfs ] || command -v mkfs.btrfs > /dev/null 2>&1 || pf_fail "the tool mkfs.btrfs is missing from the live system"
  [ "$fs" != ext4 ] || command -v mkfs.ext4 > /dev/null 2>&1 || pf_fail "the tool mkfs.ext4 is missing from the live system"
  ! ans_true '.disk.encrypt.enabled' || command -v cryptsetup > /dev/null 2>&1 || pf_fail "the tool cryptsetup is missing from the live system"

  if ! in_is_block "$disk"; then
    pf_fail "$disk is not a block device"
  else
    # tamaño: un disco entero necesita sitio para un sistema cómodo; junto a otro, el hueco elegido
    size="$(blockdev --getsize64 "$disk" 2> /dev/null || echo 0)"
    if [ "$strategy" = whole ]; then
      # El mínimo se puede bajar con MAXOR_INSTALL_MIN_GIB solo para pruebas con discos virtuales pequeños.
      min="${MAXOR_INSTALL_MIN_GIB:-32}"
      need=$((min * 1073741824))
      [ "$size" -ge "$need" ] || pf_fail "$disk has $((size / 1073741824)) GiB; a full install needs at least $min GiB"
    else
      local rs re
      rs="$(ans .disk.region.start)"
      re="$(ans .disk.region.end)"
      min="${MAXOR_INSTALL_MIN_REGION_GIB:-40}"
      need=$((min * 1073741824))
      [ $(((re - rs + 1) * 512)) -ge "$need" ] || pf_fail "the free region is smaller than $min GiB"
    fi

    # el medio del que arrancó la ISO nunca es un destino
    live="$(findmnt -n -o SOURCE /iso 2> /dev/null || true)"
    if [ -n "$live" ]; then
      parent="$(lsblk -no PKNAME "$live" 2> /dev/null | head -n1)"
      if [ -n "$parent" ] && [ "/dev/$parent" = "$disk" ]; then pf_fail "$disk is the medium this live system booted from"; fi
    fi

    # nada del disco montado ni usado como swap
    mounted="$(lsblk -nr -o MOUNTPOINTS "$disk" 2> /dev/null | tr -d '[:space:]')"
    [ -z "$mounted" ] || pf_fail "something on $disk is mounted; unmount it first"

    # restos de una instalación anterior de Maxor: pueden ser una instalación que funciona
    if [ "$strategy" = alongside ] && ! ans_true '.disk.reclaim_leftovers'; then
      if lsblk -nr -o PARTLABEL "$disk" 2> /dev/null | grep -qE '^(maxor-root|MAXOR-ESP)$'; then
        pf_fail "$disk has partitions of a previous Maxor install; they may be a working system"
      fi
    fi
  fi

  [ "$(date +%Y)" -ge 2025 ] || pf_fail "the clock says year $(date +%Y); fix the date so certificates and signatures work"

  if ! ans_true '.network.offline'; then
    curl -fsS -m 8 -o /dev/null https://cache.nixos.org/nix-cache-info 2> /dev/null \
      || pf_fail "no network: cannot reach cache.nixos.org (connect, or choose an offline install)"
  fi

  # systemd-boot no está firmado: con Secure Boot activo no arrancaría
  local sb
  sb="$(ls "$PF_EFI"/efivars/SecureBoot-* 2> /dev/null | head -n1 || true)"
  if [ -n "$sb" ] && [ "$(tail -c1 "$sb" | od -An -tu1 | tr -d ' ')" = 1 ] && ! ans_true '.advanced.allow_secureboot'; then
    pf_fail "Secure Boot is on and the boot loader is not signed yet; turn it off in the firmware"
  fi

  local kb
  kb="$(awk '/^MemTotal:/ {print $2}' "$PF_MEMINFO" 2> /dev/null || echo 0)"
  [ "${kb:-0}" -ge 1500000 ] || pf_fail "less than 1.5 GiB of RAM: the installer needs more to unpack the system"

  if ans_true '.disk.encrypt.enabled'; then
    [ "${#IN_SECRET}" -ge 8 ] || pf_fail "the encryption passphrase is missing or shorter than 8 characters"
  fi
}

stage_preflight() {
  if [ "$IN_DRY" = 1 ]; then
    printf 'DRYRUN: preflight checks (root, UEFI, tools, disk size and state, live medium, mounts, clock, network, Secure Boot, RAM)\n'
    return 0
  fi
  preflight_checks
  if [ "${#PF_FAILS[@]}" -gt 0 ]; then
    local m
    for m in "${PF_FAILS[@]}"; do printf '✗ %s\n' "$m" >&2; done
    in_die "$IN_EX_PREFLIGHT" "this machine cannot be installed to yet (${#PF_FAILS[@]} problem(s) above)"
  fi
}
