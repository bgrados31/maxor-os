# A generic machine, the one whose finished packages the installation medium carries. The machine being installed
# replaces this with what nixos-generate-config finds on it; the packages are the same, only the parts that
# describe the machine differ (and are built during the installation).
{ lib, ... }: {
  boot.initrd.availableKernelModules = [ "ahci" "xhci_pci" "nvme" "usbhid" "usb_storage" "sd_mod" "sr_mod" "virtio_pci" "virtio_blk" ];
  fileSystems."/" = { device = "/dev/disk/by-label/maxor-root"; fsType = "btrfs"; options = [ "subvol=@" "compress=zstd" "noatime" ]; };
  fileSystems."/boot" = { device = "/dev/disk/by-label/MAXOR-ESP"; fsType = "vfat"; options = [ "umask=0077" ]; };
  nixpkgs.hostPlatform = lib.mkDefault "x86_64-linux";
}
