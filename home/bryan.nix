{ config, pkgs, lib, inputs, ... }:

{
  imports = [
    inputs.dms.homeModules.dank-material-shell
    ./hyprland.nix
    ./lockscreen.nix
    ./maxor.nix
  ];

  home.username = "bryan";
  home.homeDirectory = "/home/bryan";
  home.stateVersion = "26.05";

  # ── DankMaterialShell: barra, launcher, notificaciones, lockscreen,
  #    centro de control y temas generados desde tu wallpaper (matugen).
  #    Todo se configura desde su app de ajustes (SUPER + ,).
  programs.dank-material-shell = {
    # Maxor Shell: DMS con el logo y el nombre de Maxor OS (ver packages/maxor-shell.nix)
    package = pkgs.callPackage ../packages/maxor-shell.nix {
      dmsShell = inputs.dms.packages.${pkgs.stdenv.hostPlatform.system}.dms-shell;
    };
    enable = true;
    systemd.enable = true;
  };

  # ── Terminal ────────────────────────────────────────────────────────
  programs.kitty = {
    enable = true;
    font = {
      name = "Red Hat Mono";
      size = 12;
    };
    settings = {
      window_padding_width = 12;
      background_opacity = "0.92";
      hide_window_decorations = "yes";
      confirm_os_window_close = 0;
      cursor_shape = "beam";
      scrollback_lines = 10000;
      copy_on_select = "yes";
      tab_bar_style = "powerline";
      enable_audio_bell = "no";
    };
    # Colores que DMS regenera al cambiar de wallpaper
    extraConfig = ''
      include dank-theme.conf
      include dank-tabs.conf
    '';
  };

  # ── Shell ───────────────────────────────────────────────────────────
  programs.fish = {
    enable = true;
    shellInit = ''
      fish_add_path -m $HOME/.local/bin
    '';
    interactiveShellInit = ''
      set -g fish_greeting
      fastfetch
    '';
    shellAliases = {
      ls = "eza --icons --group-directories-first";
      ll = "eza -lah --icons --group-directories-first";
      tree = "eza --tree --icons";
      cat = "bat --paging=never";
      rebuild = "sudo nixos-rebuild switch --flake ~/nixos-config#nitro";
      update = "nix flake update --flake ~/nixos-config && sudo nixos-rebuild switch --flake ~/nixos-config#nitro";
    };
  };

  programs.starship = {
    enable = true;
    enableFishIntegration = true;
  };

  programs.zoxide = {
    enable = true;
    enableFishIntegration = true;
    options = [ "--cmd" "cd" ];
  };

  programs.fzf.enable = true;
  programs.eza.enable = true;
  programs.bat.enable = true;
  programs.btop.enable = true;
  programs.fastfetch = {
    enable = true;
    settings = {
      logo = {
        source = "${../branding/logo.txt}";
        type = "file";
        color = { "1" = "blue"; };
        padding = { top = 1; right = 3; };
      };
      modules = [
        "title"
        "separator"
        "os"
        "host"
        "kernel"
        "uptime"
        "packages"
        "shell"
        "wm"
        "terminal"
        "cpu"
        "gpu"
        "memory"
        "disk"
        "break"
        "colors"
      ];
    };
  };

  programs.git = {
    enable = true;
    settings.user = {
      name = "Bryan Grados";
      email = "218035463+bgrados31@users.noreply.github.com";
    };
  };

  # GitHub CLI: `gh auth login` y de paso queda como credential helper de git.
  programs.gh = {
    enable = true;
    settings.git_protocol = "https";
  };

  # ── Temas GTK / Qt (DMS los recolorea con matugen) ──────────────────
  gtk = {
    enable = true;
    theme = {
      name = "adw-gtk3-dark";
      package = pkgs.adw-gtk3;
    };
    iconTheme = {
      name = "Papirus-Dark";
      package = pkgs.papirus-icon-theme;
    };
    font = {
      name = "Figtree";
      size = 11;
    };
  };

  home.pointerCursor = {
    name = "Bibata-Modern-Classic";
    package = pkgs.bibata-cursors;
    size = 24;
    gtk.enable = true;
  };

  dconf.settings."org/gnome/desktop/interface".color-scheme = "prefer-dark";

  # Claude Code (instalador nativo) vive en ~/.local/bin: declararlo aquí
  # evita depender de fish_variables o ~/.bashrc.
  home.sessionPath = [ "$HOME/.local/bin" ];

  home.sessionVariables = {
    QT_QPA_PLATFORMTHEME = "qt6ct";
    TERMINAL = "kitty";
  };


  # ── Apps ────────────────────────────────────────────────────────────
  home.packages = with pkgs; [
    kdePackages.qt6ct
    libsForQt5.qt5ct
    nautilus
    pavucontrol
    wl-clipboard
    playerctl
    brightnessctl
    ripgrep
    fd
    jq
    imv
    mpv
  ];

  # Archivos que DMS sobreescribe en tiempo de ejecución: solo se crean
  # si no existen, para que DMS pueda editarlos libremente.
  home.activation.dmsStubs = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
    for f in colors outputs layout cursor binds-user windowrules; do
      p="${config.xdg.configHome}/hypr/dms/$f.lua"
      [ -e "$p" ] || { run mkdir -p "$(dirname "$p")"; run touch "$p"; }
    done
    for f in dank-theme dank-tabs; do
      p="${config.xdg.configHome}/kitty/$f.conf"
      [ -e "$p" ] || { run mkdir -p "$(dirname "$p")"; run touch "$p"; }
    done
    run mkdir -p "$HOME/Pictures/Wallpapers"
  '';
}
