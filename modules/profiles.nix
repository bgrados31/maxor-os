{ config, lib, pkgs, ... }:

# Perfiles de uso: paquetes y servicios por tipo de usuario.
#
# Se activan con `maxor profile enable <nombre>`, que escribe
# hosts/<equipo>/maxor.json y reconstruye. Los nombres y descripciones viven en
# profiles-catalog.json (los lee también la CLI); aquí solo va lo que instalan.
let
  cfg = config.maxor;
  catalog = builtins.fromJSON (builtins.readFile ./profiles-catalog.json);
  settings = if cfg.settings == null then { } else builtins.fromJSON (builtins.readFile cfg.settings);
  has = name: builtins.elem name cfg.profiles;
in
{
  options.maxor = {
    settings = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = "maxor.json del equipo (lo escribe la CLI `maxor`).";
    };
    profiles = lib.mkOption {
      type = lib.types.listOf (lib.types.enum (builtins.attrNames catalog));
      default = settings.profiles or [ ];
      description = "Perfiles activos. Por defecto salen de maxor.json.";
    };
    theme = lib.mkOption {
      type = lib.types.strMatching "[a-z0-9-]{1,32}";
      default = if (settings.theme or null) == null then "maxor-dark" else settings.theme;
      description = ''
        Tema del primer arranque (el que se eligió al instalar: maxor-dark o maxor-light). Solo se aplica una
        vez; después el tema lo cambia cada usuario con `maxor theme`. Por defecto sale de maxor.json.
      '';
    };
  };

  config = lib.mkMerge [
    (lib.mkIf (has "gaming") {
      programs.steam = {
        enable = true;
        gamescopeSession.enable = true;
      };
      programs.gamemode.enable = true;
      environment.systemPackages = with pkgs; [ mangohud protonup-qt lutris ];
    })

    (lib.mkIf (has "dev") {
      virtualisation.podman = {
        enable = true;
        dockerCompat = true; # `docker` apunta a Podman, sin demonio ni root
        defaultNetwork.settings.dns_enabled = true;
      };
      programs.direnv = {
        enable = true;
        nix-direnv.enable = true;
      };
      environment.systemPackages = with pkgs; [
        gcc gnumake cmake pkg-config
        nodejs python3 go rustup
        gh lazygit jq ripgrep fd
      ];
    })

    (lib.mkIf (has "creator") {
      environment.systemPackages = with pkgs; [
        obs-studio gimp inkscape krita kdePackages.kdenlive blender audacity handbrake
      ];
    })

    (lib.mkIf (has "office") {
      environment.systemPackages = with pkgs; [
        libreoffice-fresh
        hunspell
        hunspellDicts.es_ES
        hunspellDicts.en_US
        thunderbird
        kdePackages.okular
      ];
    })
  ];
}
