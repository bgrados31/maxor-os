{ config, pkgs, lib, ... }:

# Lockscreen de Maxor OS: hyprlock (widgets libres: reloj, fecha, saludo,
# campo de contraseña, reproductor, batería) + hypridle (bloqueo/apagado de
# pantalla por inactividad). Todo se edita en este archivo; los colores
# viven arriba para cambiarlos de un solo lugar.
let
  # Paleta por defecto (Sakura nocturna). El motor de temas usa la misma función
  # con marcadores @..@ para generar la plantilla que `maxor theme apply` rellena.
  sakura = {
    text = "rgba(251, 233, 242, 1.0)";
    muted = "rgba(251, 233, 242, 0.60)";
    accent = "rgba(255, 134, 184, 1.0)";
    inner = "rgba(18, 11, 18, 0.55)";
    hint = "##fbe9f2a0";
    fail = "rgba(255, 107, 129, 1.0)";
    check = "rgba(140, 255, 190, 1.0)";
  };
  placeholders = sakura // {
    text = "rgba(@FG_RGB@, 1.0)";
    muted = "rgba(@FG_RGB@, 0.60)";
    accent = "rgba(@AC_RGB@, 1.0)";
    inner = "rgba(@BG_RGB@, 0.55)";
    hint = "##@FG_HEX@a0";
  };
  font = "Figtree";
  mono = "Red Hat Mono";
  word = "Krona One";

  mkSettings = c: {
      general = {
        hide_cursor = true;
        ignore_empty_input = true;
        grace = 0;
      };

      animations = {
        enabled = true;
        bezier = [ "ease, 0.25, 0.1, 0.25, 1" ];
        animation = [
          "fadeIn, 1, 4, ease"
          "fadeOut, 1, 4, ease"
          "inputFieldDots, 1, 2, ease"
        ];
      };

      # Fondo: captura de pantalla desenfocada (no depende de ningún wallpaper).
      # Para usar una imagen: path = /ruta/imagen.png
      background = [{
        monitor = "";
        path = "screenshot";
        blur_passes = 3;
        blur_size = 8;
        noise = 0.015;
        brightness = 0.6;
        vibrancy = 0.2;
      }];

      label = [
        # Reloj
        {
          monitor = "";
          text = ''cmd[update:1000] date +"%H:%M"'';
          color = c.text;
          font_size = 120;
          font_family = "${font} Bold";
          position = "0, 220";
          halign = "center";
          valign = "center";
        }
        # Fecha
        {
          monitor = "";
          text = ''cmd[update:60000] date +"%A, %d de %B"'';
          color = c.muted;
          font_size = 20;
          font_family = font;
          position = "0, 130";
          halign = "center";
          valign = "center";
        }
        # Saludo
        {
          monitor = "";
          text = "Hola, $USER";
          color = c.text;
          font_size = 18;
          font_family = font;
          position = "0, -20";
          halign = "center";
          valign = "center";
        }
        # Marca
        {
          monitor = "";
          text = "MAXOR OS";
          color = c.accent;
          font_size = 12;
          font_family = word;
          position = "0, 30";
          halign = "center";
          valign = "bottom";
        }
        # Batería
        {
          monitor = "";
          text = ''cmd[update:30000] echo "$(cat /sys/class/power_supply/BAT*/capacity 2>/dev/null | head -1)%"'';
          color = c.muted;
          font_size = 14;
          font_family = mono;
          position = "-30, -25";
          halign = "right";
          valign = "top";
        }
        # Distribución de teclado actual
        {
          monitor = "";
          text = "$LAYOUT";
          color = c.muted;
          font_size = 12;
          font_family = mono;
          position = "30, -25";
          halign = "left";
          valign = "top";
        }
      ];

      input-field = [{
        monitor = "";
        size = "300, 54";
        outline_thickness = 2;
        dots_size = 0.25;
        dots_spacing = 0.35;
        dots_center = true;
        outer_color = c.accent;
        inner_color = c.inner;
        font_color = c.text;
        check_color = c.check;
        fail_color = c.fail;
        fade_on_empty = false;
        rounding = 10;
        placeholder_text = "<span foreground=\"${c.hint}\">Contraseña</span>";
        fail_text = "<i>Incorrecta ($ATTEMPTS)</i>";
        capslock_color = c.fail;
        position = "0, -110";
        halign = "center";
        valign = "center";
      }];
  };
in
{
  programs.hyprlock = {
    enable = true;
    settings = mkSettings sakura;
  };

  # Plantilla con marcadores: `maxor theme apply` la rellena con la paleta del tema
  # y escribe ~/.config/maxor/current/hyprlock.conf.
  xdg.dataFile."maxor/templates/hyprlock.conf.tpl".text =
    lib.hm.generators.toHyprconf { attrs = mkSettings placeholders; };

  services.hypridle = {
    enable = true;
    settings = {
      general = {
        lock_cmd = "pidof hyprlock || { [ -f $HOME/.config/maxor/current/hyprlock.conf ] && hyprlock -c $HOME/.config/maxor/current/hyprlock.conf; } || hyprlock";
        before_sleep_cmd = "loginctl lock-session";
        after_sleep_cmd = "hyprctl dispatch dpms on";
      };
      listener = [
        { timeout = 300; on-timeout = "loginctl lock-session"; }
        { timeout = 420; on-timeout = "hyprctl dispatch dpms off"; on-resume = "hyprctl dispatch dpms on"; }
      ];
    };
  };
}
