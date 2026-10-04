{ config, pkgs, lib, modulesPath, ... }:

# Máquina virtual de pruebas: el Maxor OS de verdad (mismos módulos, escritorio, login y CLI)
# sobre QEMU, sin NVIDIA ni las particiones del portátil. Sirve para probar cambios, la
# pantalla completa y el sistema de releases sin tocar el equipo real:
#
#   nix run .#vm        la versión de este repositorio
#   nix run .#vm-old    se hace pasar por la 0.0.1, así que la release publicada se ofrece como actualización
#
# Usuario `bryan`, contraseña `maxor`. El disco vive en ~/.local/state/maxor-vm/ (se conserva entre
# arranques; bórralo para empezar de cero).
{
  imports = [ (modulesPath + "/virtualisation/qemu-vm.nix") ];

  maxor.machine = {
    hostname = "maxor-vm";
    user = "bryan";
    fullname = "Bryan";
  };

  maxor.hardware.report = ./hardware.json;
  maxor.settings = ./maxor.json;

  users.users.${config.maxor.machine.user}.initialPassword = "maxor"; # solo para esta máquina de pruebas

  virtualisation = {
    memorySize = 6144;
    cores = 4;
    diskSize = 24576; # MB: sitio para instalar apps y probar actualizaciones
    resolution = { x = 1600; y = 900; };
  };

  # La primera sesión de cada arranque entra sola (para probar sin teclear); al cerrar sesión
  # aparece el login de verdad, el greeter de Maxor Shell.
  services.greetd.settings.initial_session = {
    command = "${config.programs.hyprland.package}/bin/start-hyprland";
    user = config.maxor.machine.user;
  };

  # Sin aceleración 3D (o con ella, según el lanzador) Hyprland necesita estas ayudas en una VM.
  environment.sessionVariables = {
    WLR_NO_HARDWARE_CURSORS = "1";
    AQ_NO_MODIFIERS = "1";
  };

  system.stateVersion = "26.05"; # NO cambiar
}
