# The same boot configuration the engine writes to host/boot.nix.
{ ... }: {
  boot.loader.systemd-boot.enable = true;
  boot.loader.systemd-boot.configurationLimit = 10;
  boot.loader.efi.canTouchEfiVariables = true;
  system.stateVersion = "26.05";
}
