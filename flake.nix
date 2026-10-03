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
    # Módulos de sistema de Maxor OS, reutilizables desde otro flake.
    nixosModules.default = {
      imports = [
        ./modules/core.nix
        ./modules/desktop.nix
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
