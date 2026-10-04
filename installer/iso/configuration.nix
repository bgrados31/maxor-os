{ config, lib, pkgs, inputs, self, modulesPath, ... }:

# La ISO de Maxor OS: un sistema vivo (sesión Hyprland con Maxor Shell, sin instalar nada) que lleva el
# instalador. Es el mismo Maxor OS que se instala —mismos módulos, escritorio y CLI—, así que se puede probar
# antes de decidir. Se construye con `nix build .#iso`. Ver docs/INSTALLER.md.
let
  version = lib.removeSuffix "\n" (builtins.readFile ../../VERSION);
  maxor = pkgs.callPackage ../../packages/maxor.nix { };
  maxorTui = pkgs.callPackage ../../packages/maxor-tui.nix { };
  maxorInstall = pkgs.callPackage ../../packages/maxor-install.nix { inherit maxor; };

  # Para instalar sin red: la copia local de Maxor OS y de sus inputs, con sus metadatos (ver
  # installer/offline-overrides.nix). El instalador la lee de /etc/maxor-install/overrides.
  overrides = pkgs.writeText "maxor-install-overrides" (import ../offline-overrides.nix { inherit lib self inputs; });
in
{
  imports = [ (modulesPath + "/installer/cd-dvd/installation-cd-base.nix") ];

  # ── Quién es el sistema vivo ────────────────────────────────────────
  maxor.machine = {
    hostname = "maxor-live";
    user = "live";
    fullname = "Maxor live session";
  };
  # El sistema vivo arranca en cualquier equipo: sin informe de hardware concreto, los drivers genéricos de la ISO.
  maxor.hardware.report = ./hardware.json; # un archivo del repositorio: se lee al evaluar, no puede ser una derivación (IFD)

  # ── Sesión viva: entra sola, sin contraseña ─────────────────────────
  users.users.live = {
    initialHashedPassword = "";
    extraGroups = [ "wheel" ];
  };
  security.sudo.wheelNeedsPassword = false;
  services.greetd.settings.initial_session = {
    command = "${config.programs.hyprland.package}/bin/start-hyprland";
    user = "live";
  };
  # La ISO base usa wpa_supplicant a secas; Maxor usa NetworkManager (Wi-Fi para el instalador y el sistema).
  networking.wireless.enable = lib.mkImageMediaOverride false;

  # ── El instalador ───────────────────────────────────────────────────
  environment.systemPackages = [ maxorInstall maxorTui ];
  environment.etc."maxor-install/overrides".source = overrides;
  # Marca la sesión viva: lo que solo tiene sentido en la ISO (el instalador, los avisos) pregunta por este archivo.
  environment.etc."maxor-live".text = "${version}\n";

  # El instalador se abre solo, a pantalla completa, cuando el escritorio está listo. Colgado de dms.service como
  # maxor-first-run (ver home/maxor.nix): DMS arranca después de graphical-session.target. Si se cierra, se puede
  # volver a abrir con `maxor-tui --screen install`.
  systemd.user.services.maxor-installer = {
    description = "Maxor OS installer";
    after = [ "dms.service" ];
    wantedBy = [ "dms.service" ];
    serviceConfig = {
      ExecStart = "${pkgs.kitty}/bin/kitty --start-as=fullscreen --title='Maxor OS installer' ${maxorTui}/bin/maxor-tui --screen install";
      Restart = "no";
    };
  };

  # ── Imagen y arranque ───────────────────────────────────────────────
  # El menú de la ISO espera más que el del sistema instalado: hay que dar tiempo a elegir.
  boot.loader.timeout = lib.mkForce 10;
  # Nombre del archivo: maxor-os-<versión>-<arquitectura>.iso
  image.baseName = lib.mkForce "maxor-os-${version}-${pkgs.stdenv.hostPlatform.system}";
  isoImage = {
    volumeID = "MAXOR_OS";
    edition = "maxor";
    prependToMenuLabel = "Maxor OS · ";
    squashfsCompression = "zstd -Xcompression-level 15";
    makeEfiBootable = true;
    makeUsbBootable = true;
  };
  # ZFS viene con la ISO base; no hace falta que fuerce importar la raíz (y evita un aviso).
  boot.zfs.forceImportRoot = false;

  system.stateVersion = "26.05"; # NO cambiar
}
