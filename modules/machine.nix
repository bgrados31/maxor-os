{ config, lib, pkgs, inputs, ... }:

# Quién y dónde es esta máquina: nombre, usuario, región y teclado. Es lo único que una instalación
# necesita decir para ser suya; todo lo demás (escritorio, shell, CLI, temas) lo pone Maxor OS.
# El instalador escribe estas opciones; un host propio (hosts/<equipo>) también las rellena.
#
#   maxor.machine = { hostname = "maxor"; user = "ana"; timezone = "Europe/Madrid"; … };
#
# Aquí también se conecta home-manager con el usuario, para que ninguna otra parte de la
# distribución tenga un nombre de usuario escrito a mano.
let
  cfg = config.maxor.machine;
  inherit (lib) mkOption mkIf types;
in
{
  imports = [ inputs.home-manager.nixosModules.home-manager ];

  options.maxor.machine = {
    hostname = mkOption {
      type = types.str;
      default = "maxor";
      description = "Nombre del equipo en la red.";
    };
    user = mkOption {
      type = types.strMatching "[a-z_][a-z0-9_-]*";
      description = "Usuario principal (administrador, con fish y el escritorio de Maxor).";
    };
    fullname = mkOption {
      type = types.str;
      default = "";
      description = "Nombre completo del usuario; vacío usa el nombre de usuario.";
    };
    passwordHash = mkOption {
      type = types.nullOr types.str;
      default = null;
      description = "Hash crypt(3) de la contraseña. Null la deja sin fijar (se pone con passwd).";
    };
    timezone = mkOption {
      type = types.str;
      default = "UTC";
      description = "Zona horaria, por ejemplo America/Lima.";
    };
    locale = mkOption {
      type = types.str;
      default = "en_US.UTF-8";
      description = "Idioma del sistema.";
    };
    keymap = mkOption {
      type = types.str;
      default = "us";
      description = "Teclado de la consola (por ejemplo la-latin1). Vacío: el mismo que el del escritorio (xkb).";
    };
    xkb = {
      layout = mkOption {
        type = types.str;
        default = "us";
        description = "Distribución del teclado del escritorio (por ejemplo latam).";
      };
      variant = mkOption {
        type = types.str;
        default = "";
        description = "Variante de la distribución del teclado.";
      };
    };
    git = {
      name = mkOption { type = types.nullOr types.str; default = null; description = "Nombre de git; null no lo fija."; };
      email = mkOption { type = types.nullOr types.str; default = null; description = "Correo de git; null no lo fija."; };
    };
    cliVersion = mkOption {
      type = types.nullOr types.str;
      default = null;
      description = "Versión que dice ser la CLI `maxor`. Null usa VERSION; solo la VM «vm-old» lo cambia.";
    };
    homeModules = mkOption {
      type = types.listOf types.deferredModule;
      default = [ ];
      description = "Módulos de home-manager extra para el usuario (lo suyo, encima del escritorio de Maxor).";
    };
  };

  config = {
    networking.hostName = cfg.hostname;
    time.timeZone = cfg.timezone;
    i18n.defaultLocale = cfg.locale;
    # An empty console keymap means "the same as the desktop layout" (every XKB layout works that way).
    console = if cfg.keymap == "" then { useXkbConfig = true; } else { keyMap = cfg.keymap; };
    services.xserver.xkb = {
      layout = cfg.xkb.layout;
      variant = cfg.xkb.variant;
    };

    users.users.${cfg.user} = {
      isNormalUser = true;
      description = if cfg.fullname == "" then cfg.user else cfg.fullname;
      extraGroups = [ "networkmanager" "wheel" "video" "audio" ];
      shell = pkgs.fish;
      hashedPassword = mkIf (cfg.passwordHash != null) cfg.passwordHash;
    };

    # El greeter de login hereda el tema y el wallpaper de este usuario.
    services.displayManager.dms-greeter.configHome = "/home/${cfg.user}";

    home-manager = {
      useGlobalPkgs = true;
      useUserPackages = true;
      backupFileExtension = "hm-backup";
      extraSpecialArgs = {
        inherit inputs;
        maxorVersion = cfg.cliVersion;
      };
      users.${cfg.user} = {
        imports = [ ../home/user.nix ] ++ cfg.homeModules;
      };
    };
  };
}
