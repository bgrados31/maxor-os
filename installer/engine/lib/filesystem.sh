# ── Etapa filesystem: formatear, montar y preparar el swap ───────────
# La ESP se monta en /boot (systemd-boot guarda ahí kernels e initrd). btrfs lleva subvolúmenes
# @ @home @snapshots (y @swap si hay swap en archivo, sin copia en escritura).
stage_filesystem() {
  local fs rootdev esp swapkind swapgib btrfs_opts="compress=zstd,noatime"
  fs="$(ans .disk.filesystem)"
  rootdev="$(dev_rootfs)"
  esp="$(dev_get esp)"
  swapkind="$(ans .disk.swap.kind)"
  swapgib="$(ans .disk.swap.gib)"

  in_run mkfs.fat -F 32 -n MAXOR-ESP "$esp"
  case "$fs" in
    ext4)
      in_run mkfs.ext4 -F -L maxor-root "$rootdev"
      in_run mkdir -p "$IN_ROOT"
      in_run mount "$rootdev" "$IN_ROOT"
      in_run mkdir -p "$IN_ROOT/boot"
      ;;
    btrfs)
      in_run mkfs.btrfs -f -L maxor-root "$rootdev"
      in_run mkdir -p "$IN_ROOT"
      in_run mount "$rootdev" "$IN_ROOT"
      in_run btrfs subvolume create "$IN_ROOT/@"
      in_run btrfs subvolume create "$IN_ROOT/@home"
      in_run btrfs subvolume create "$IN_ROOT/@snapshots"
      [ "$swapkind" != file ] || in_run btrfs subvolume create "$IN_ROOT/@swap"
      in_run umount "$IN_ROOT"
      in_run mount -o "subvol=@,$btrfs_opts" "$rootdev" "$IN_ROOT"
      in_run mkdir -p "$IN_ROOT/home" "$IN_ROOT/.snapshots" "$IN_ROOT/boot"
      in_run mount -o "subvol=@home,$btrfs_opts" "$rootdev" "$IN_ROOT/home"
      in_run mount -o "subvol=@snapshots,$btrfs_opts" "$rootdev" "$IN_ROOT/.snapshots"
      if [ "$swapkind" = file ]; then
        in_run mkdir -p "$IN_ROOT/swap"
        in_run mount -o subvol=@swap,noatime "$rootdev" "$IN_ROOT/swap"
      fi
      ;;
  esac
  in_run mount -o umask=0077 "$esp" "$IN_ROOT/boot"

  if [ "$swapkind" = file ]; then
    if [ "$fs" = btrfs ]; then
      in_run btrfs filesystem mkswapfile --size "${swapgib}g" "$IN_ROOT/swap/swapfile"
    else
      in_run dd if=/dev/zero of="$IN_ROOT/swapfile" bs=1M count=$((swapgib * 1024)) status=none
      in_run chmod 600 "$IN_ROOT/swapfile"
      in_run mkswap "$IN_ROOT/swapfile"
    fi
  fi
}
