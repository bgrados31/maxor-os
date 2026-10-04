# What nixos-generate-config would write for the test VM, preseeded (MAXOR_INSTALL_HWCONFIG) so the system the
# installer produces is exactly the one that is already built.
{ lib, ... }: {
  boot.initrd.availableKernelModules = [ "virtio_pci" "virtio_blk" "virtio_net" "ahci" "xhci_pci" "sd_mod" "sr_mod" ];
  fileSystems."/" = { device = "/dev/disk/by-label/maxor-root"; fsType = "btrfs"; options = [ "subvol=@" "compress=zstd" "noatime" ]; };
  fileSystems."/home" = { device = "/dev/disk/by-label/maxor-root"; fsType = "btrfs"; options = [ "subvol=@home" "compress=zstd" "noatime" ]; };
  fileSystems."/.snapshots" = { device = "/dev/disk/by-label/maxor-root"; fsType = "btrfs"; options = [ "subvol=@snapshots" "compress=zstd" "noatime" ]; };
  fileSystems."/boot" = { device = "/dev/disk/by-label/MAXOR-ESP"; fsType = "vfat"; options = [ "umask=0077" ]; };
  nixpkgs.hostPlatform = lib.mkDefault "x86_64-linux";
}
