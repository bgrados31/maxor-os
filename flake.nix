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

  outputs = { self, nixpkgs, home-manager, ... }@inputs: {
    # La CLI como paquete propio, para construirla y probarla sin el sistema entero.
    packages.x86_64-linux.maxor = nixpkgs.legacyPackages.x86_64-linux.callPackage ./packages/maxor.nix { };
    packages.x86_64-linux.maxor-tui = nixpkgs.legacyPackages.x86_64-linux.callPackage ./packages/maxor-tui.nix { };

    # La pantalla completa compila y corre sus pruebas de Go (go test) al construirse.
    checks.x86_64-linux.tui = self.packages.x86_64-linux.maxor-tui;

    # Pruebas de la CLI (tests/): `nix build .#checks.x86_64-linux.cli-tests` o
    # `nix flake check`. Corren en el sandbox, sin tocar nada del usuario.
    checks.x86_64-linux.cli-tests =
      let pkgs = nixpkgs.legacyPackages.x86_64-linux; in
      pkgs.runCommand "maxor-cli-tests"
        {
          nativeBuildInputs = with pkgs; [ bats shellcheck jq gnugrep gnused gawk coreutils findutils gnutar ncurses git openssh curl util-linux ];
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

    # Módulos de sistema de Maxor OS, reutilizables desde otro flake.
    nixosModules.default = {
      imports = [
        ./modules/core.nix
        ./modules/hardware.nix
        ./modules/profiles.nix
        ./modules/desktop.nix
        ./modules/greeter.nix
        ./modules/branding.nix
        ./modules/fonts.nix
      ];
    };

    nixosConfigurations.nitro = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      specialArgs = { inherit inputs; };
      modules = [
        self.nixosModules.default
        ./hosts/nitro/configuration.nix
        home-manager.nixosModules.home-manager
        {
          home-manager.useGlobalPkgs = true;
          home-manager.useUserPackages = true;
          home-manager.backupFileExtension = "hm-backup";
          home-manager.extraSpecialArgs = { inherit inputs; };
          home-manager.users.bryan = import ./home/bryan.nix;
        }
      ];
    };
  };
}
