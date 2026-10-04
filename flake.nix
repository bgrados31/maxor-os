{
  description = "Maxor OS — escritorio Hyprland declarativo con motor de temas";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-26.05";

    home-manager = {
      url = "github:nix-community/home-manager/release-26.05";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    # Shell completa (barra, launcher, notificaciones, lockscreen, temas por wallpaper)
    dms = {
      url = "github:AvengeMedia/DankMaterialShell";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, home-manager, ... }@inputs:
  let
    pkgs = nixpkgs.legacyPackages.x86_64-linux;

    # El punto de entrada de la distribución: un sistema Maxor OS completo a partir de lo propio de
    # la máquina (maxor.machine: nombre, usuario, región, teclado) y los módulos extra que quiera
    # (arranque, discos). Lo usan los hosts de este repositorio y el flake que escribe el instalador:
    #   mkSystem { machine = { hostname = "maxor"; user = "ana"; … }; modules = [ ./boot.nix ]; }
    mkSystem = { machine ? { }, modules ? [ ] }: nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      specialArgs = { inherit inputs; };
      modules = [ self.nixosModules.default { maxor.machine = machine; } ] ++ modules;
    };

    # La máquina virtual de pruebas (hosts/vm): el mismo Maxor OS sobre QEMU.
    # cliVersion, si se da, es la versión que dice ser la CLI (ver vm-old).
    mkVm = { cliVersion ? null }: mkSystem {
      modules = [ ./hosts/vm/configuration.nix { maxor.machine.cliVersion = cliVersion; } ];
    };

    # El lanzador: una ventana de QEMU con aceleración 3D y su disco en ~/.local/state/maxor-vm/.
    # MAXOR_VM_HEADLESS=1 arranca sin ventana (para pruebas automáticas, con monitor en un socket).
    mkVmRunner = name: cfg: pkgs.writeShellScriptBin name ''
      dir="''${XDG_STATE_HOME:-$HOME/.local/state}/maxor-vm"
      mkdir -p "$dir"
      export NIX_DISK_IMAGE="$dir/${name}.qcow2"
      if [ -z "''${QEMU_OPTS:-}" ]; then
        if [ -n "''${MAXOR_VM_HEADLESS:-}" ]; then
          export QEMU_OPTS="-display none -vga none -device virtio-vga -monitor unix:''${XDG_RUNTIME_DIR:-/tmp}/${name}.monitor,server,nowait"
        else
          export QEMU_OPTS="-vga none -device virtio-vga-gl -display gtk,gl=on"
        fi
      fi
      echo "Maxor OS VM (${name}): usuario bryan, contraseña maxor. Disco: $NIX_DISK_IMAGE" >&2
      exec ${cfg.config.system.build.vm}/bin/run-${cfg.config.networking.hostName}-vm "$@"
    '';
  in {
    # La VM de pruebas: `nix run .#vm` (esta versión) y `nix run .#vm-old` (se hace pasar por
    # la 0.0.1: la release publicada aparece como actualización disponible).
    nixosConfigurations.maxor-vm = mkVm { };
    nixosConfigurations.maxor-vm-old = mkVm { cliVersion = "0.0.1"; };
    packages.x86_64-linux.vm = mkVmRunner "maxor-vm" self.nixosConfigurations.maxor-vm;
    packages.x86_64-linux.vm-old = mkVmRunner "maxor-vm-old" self.nixosConfigurations.maxor-vm-old;
    apps.x86_64-linux.vm = { type = "app"; program = "${self.packages.x86_64-linux.vm}/bin/maxor-vm"; meta.description = "Maxor OS en una máquina virtual de pruebas"; };
    apps.x86_64-linux.vm-old = { type = "app"; program = "${self.packages.x86_64-linux.vm-old}/bin/maxor-vm-old"; meta.description = "La VM de pruebas, haciéndose pasar por la 0.0.1 para ver una actualización"; };

    # La CLI como paquete propio, para construirla y probarla sin el sistema entero.
    packages.x86_64-linux.maxor = nixpkgs.legacyPackages.x86_64-linux.callPackage ./packages/maxor.nix { };
    packages.x86_64-linux.maxor-install = nixpkgs.legacyPackages.x86_64-linux.callPackage ./packages/maxor-install.nix { inherit (self.packages.x86_64-linux) maxor; };
    packages.x86_64-linux.maxor-tui = nixpkgs.legacyPackages.x86_64-linux.callPackage ./packages/maxor-tui.nix { };

    # La pantalla completa compila y corre sus pruebas de Go (go test) al construirse.
    checks.x86_64-linux.tui = self.packages.x86_64-linux.maxor-tui;

    # El motor del instalador compila (y shellcheck lo revisa) como parte de las comprobaciones.
    checks.x86_64-linux.maxor-install = self.packages.x86_64-linux.maxor-install;

    # What each GPU mode turns on, evaluated with the hardware of a hybrid laptop (Intel + NVIDIA). Evaluation only: the
    # derivation is built (and the assertions run) when the check is evaluated.
    checks.x86_64-linux.gpu-modes =
      let
        pkgs = nixpkgs.legacyPackages.x86_64-linux;
        cfgOf = mode: (mkSystem {
          machine = { hostname = "t"; user = "u"; };
          modules = [ ({ ... }: {
            maxor.hardware.report = ./hosts/nitro/hardware.json;
            maxor.hardware.gpu.mode = mode;
            boot.loader.grub.enable = false;
            fileSystems."/" = { device = "x"; fsType = "ext4"; };
          }) ];
        }).config;
        offload = c: c.hardware.nvidia.prime.offload.enable;
        sync = c: c.hardware.nvidia.prime.sync.enable;
        nvidia = c: builtins.elem "nvidia" c.services.xserver.videoDrivers;
        auto = cfgOf "auto"; hybrid = cfgOf "hybrid"; integrated = cfgOf "integrated"; only = cfgOf "nvidia";
        results = {
          "auto on a hybrid laptop is hybrid" = offload auto && nvidia auto && !(sync auto);
          "hybrid is offload" = offload hybrid && !(sync hybrid);
          "integrated has no NVIDIA driver at all" = !(nvidia integrated) && !(offload integrated) && !(sync integrated);
          "nvidia only is PRIME sync on a laptop" = sync only && !(offload only) && nvidia only;
        };
        failed = builtins.attrNames (nixpkgs.lib.filterAttrs (_: ok: !ok) results);
      in
      pkgs.runCommand "check-gpu-modes" { } (
        if failed == [ ] then "touch $out"
        else "echo 'failed: ${builtins.concatStringsSep ", " failed}'; exit 1");

    # The language and keyboard lists of the installer, generated from the system's data: they must be complete and every
    # locale must fit what the answers schema accepts.
    checks.x86_64-linux.catalog =
      let pkgs = nixpkgs.legacyPackages.x86_64-linux; catalog = pkgs.callPackage ./packages/catalog.nix { }; in
      pkgs.runCommand "check-catalog" { nativeBuildInputs = [ pkgs.jq ]; } ''
        jq -e '(.locales | length) > 250 and (.layouts | length) > 400' ${catalog}
        jq -e '.locales[] | select(.code == "es_PE.UTF-8" and .name == "Spanish (Peru)")' ${catalog} > /dev/null
        jq -e '.layouts[] | select(.xkb == "latam" and .variant == "")' ${catalog} > /dev/null
        if jq -e '.locales[] | select(.code | test("^[a-z]{2,3}_[A-Z]{2}\\.UTF-8(@[a-z]+)?$") | not)' ${catalog}; then
          echo "a locale does not fit the answers schema"; exit 1
        fi
        touch $out
      '';

    # Pruebas de la CLI (tests/): `nix build .#checks.x86_64-linux.cli-tests` o
    # `nix flake check`. Corren en el sandbox, sin tocar nada del usuario.
    checks.x86_64-linux.cli-tests =
      let pkgs = nixpkgs.legacyPackages.x86_64-linux; in
      pkgs.runCommand "maxor-cli-tests"
        {
          nativeBuildInputs = with pkgs; [ bats shellcheck jq gnugrep gnused gawk coreutils findutils gnutar ncurses git openssh curl util-linux openssl ];
          MAXOR_BIN = "${self.packages.x86_64-linux.maxor}/bin/maxor";
        } ''
        cp -r ${self} src
        chmod -R u+w src
        cd src
        patchShebangs scripts
        shellcheck -x scripts/*.sh
        bats tests/
        touch $out
      '';

    # El sistema de releases de punta a punta, en máquinas virtuales (necesita KVM):
    # `nix build .#checks.x86_64-linux.release-vm -L`. No va en la CI: tarda y pide KVM.
    checks.x86_64-linux.release-vm = import ./tests/vm/release.nix { pkgs = nixpkgs.legacyPackages.x86_64-linux; };

    # Las etapas reales del instalador (disco, LUKS, sistema de archivos) sobre discos virtuales (necesita KVM).
    checks.x86_64-linux.installer-disks = import ./tests/vm/installer-disks.nix { pkgs = nixpkgs.legacyPackages.x86_64-linux; };

    # La instalación completa y el arranque del sistema instalado (necesita KVM y tarda).
    checks.x86_64-linux.installer-full = import ./tests/vm/installer-full.nix { pkgs = nixpkgs.legacyPackages.x86_64-linux; inherit self inputs; };

    # Módulos de sistema de Maxor OS, reutilizables desde otro flake.
    nixosModules.default = {
      imports = [
        ./modules/core.nix
        ./modules/machine.nix
        ./modules/hardware.nix
        ./modules/profiles.nix
        ./modules/desktop.nix
        ./modules/greeter.nix
        ./modules/branding.nix
        ./modules/fonts.nix
      ];
    };

    nixosConfigurations.nitro = mkSystem { modules = [ ./hosts/nitro/configuration.nix ]; };

    # La ISO de Maxor OS: sistema vivo con el instalador. `nix build .#iso`.
    nixosConfigurations.maxor-iso = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      specialArgs = { inherit inputs self; };
      modules = [ ./installer/iso/configuration.nix ];
    };
    packages.x86_64-linux.iso = self.nixosConfigurations.maxor-iso.config.system.build.isoImage;

    # Una máquina de ejemplo con otro usuario, otro teclado y otra región, y sin identidad de git:
    # la prueba de que la distribución no tiene ningún nombre escrito a mano. Se evalúa en la CI
    # (`nix flake check`) pero no se construye. Es también la plantilla de lo que
    # escribe el instalador.
    nixosConfigurations.example = mkSystem {
      machine = {
        hostname = "example";
        user = "ana";
        fullname = "Ana Pérez";
        timezone = "Europe/Madrid";
        locale = "es_ES.UTF-8";
        keymap = "es";
        xkb.layout = "es";
      };
      modules = [
        ({ ... }: {
          boot.loader.systemd-boot.enable = true;
          fileSystems."/" = { device = "/dev/disk/by-label/root"; fsType = "ext4"; };
          system.stateVersion = "26.05";
        })
      ];
    };

    # Para quien arma su propia máquina: `maxor-os.lib.mkSystem { machine = …; modules = […]; }`.
    lib.mkSystem = mkSystem;
  };
}
