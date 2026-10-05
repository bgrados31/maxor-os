{ config, pkgs, lib, ... }:

# Lockscreen de Maxor OS: hyprlock (widgets libres: reloj, fecha, saludo,
# campo de contraseña, reproductor, batería) + hypridle (bloqueo/apagado de
# pantalla por inactividad). Todo se edita en este archivo; los colores
# viven arriba para cambiarlos de un solo lugar.
let
  # Paleta por defecto (Maxor Dark). El motor de temas usa la misma función
  # con marcadores @..@ para generar la plantilla que `maxor theme apply` rellena.
  maxorDark = {
    text = "rgba(238, 240, 255, 1.0)";
    muted = "rgba(238, 240, 255, 0.60)";
    accent = "rgba(192, 132, 255, 1.0)";
    accent2 = "rgba(255, 90, 87, 1.0)";
    glass = "rgba(5, 12, 56, 0.35)";
    edge = "rgba(238, 240, 255, 0.14)";
    inner = "rgba(5, 12, 56, 0.55)";
    hint = "##eef0ffa0";
    fail = "rgba(255, 107, 129, 1.0)";
    check = "rgba(140, 255, 190, 1.0)";
    brightness = "0.60";
  };
  placeholders = maxorDark // {
    text = "rgba(@FG_RGB@, 1.0)";
    muted = "rgba(@FG_RGB@, 0.60)";
    accent = "rgba(@AC_RGB@, 1.0)";
    accent2 = "rgba(@AC2_RGB@, 1.0)";
    glass = "rgba(@BG_RGB@, 0.35)";
    edge = "rgba(@FG_RGB@, 0.14)";
    inner = "rgba(@BG_RGB@, 0.55)";
    hint = "##@FG_HEX@a0";
    brightness = "@LOCK_BRIGHTNESS@";
  };
  font = "Figtree";
  mono = "Red Hat Mono";
  word = "Cinzel";

  mkSettings = c: {
      general = {
        hide_cursor = true;
        ignore_empty_input = true;
        immediate_render = true; # pinta ya con la captura, sin esperar a que carguen las fuentes
        grace = 0;
      };

      animations = {
        enabled = true;
        bezier = [ "out, 0.16, 1, 0.3, 1" "ease, 0.25, 0.1, 0.25, 1" ];
        animation = [
          "fadeIn, 1, 5, out"
          "fadeOut, 1, 3, ease"
          "inputFieldDots, 1, 2, ease"
          "inputFieldColors, 1, 3, ease"
        ];
      };

      # Fondo: captura de pantalla desenfocada (no depende de ningún wallpaper).
      # Para usar una imagen: path = /ruta/imagen.png
      background = [{
        monitor = "";
        path = "screenshot";
        blur_passes = 4;
        blur_size = 10;
        noise = 0.02;
        contrast = 0.95;
        brightness = c.brightness;
        vibrancy = 0.25;
        vibrancy_darkness = 0.1;
      }];

      # Tarjeta de cristal que reúne usuario y campo, y una línea de acento bajo el reloj.
      shape = [
        {
          monitor = "";
          size = "420, 190";
          color = c.glass;
          rounding = 28;
          border_size = 1;
          border_color = c.edge;
          position = "0, -135";
          halign = "center";
          valign = "center";
        }
        {
          monitor = "";
          size = "56, 3";
          color = c.accent2;
          rounding = 2;
          position = "0, 95";
          halign = "center";
          valign = "center";
        }
      ];

      label = [
        # Marca arriba, en Cinzel y espaciada
        {
          monitor = "";
          text = "M A X O R   O S";
          color = c.accent;
          font_size = 14;
          font_family = word;
          position = "0, -40";
          halign = "center";
          valign = "top";
        }
        # Reloj
        {
          monitor = "";
          text = ''cmd[update:1000] date +"%H:%M"'';
          color = c.text;
          font_size = 168;
          font_family = "${font} Light";
          position = "0, 215";
          halign = "center";
          valign = "center";
        }
        # Fecha
        {
          monitor = "";
          text = ''cmd[update:60000] date +"%A  ·  %-d %B"'';
          color = c.muted;
          font_size = 20;
          font_family = font;
          position = "0, 130";
          halign = "center";
          valign = "center";
        }
        # Usuario
        {
          monitor = "";
          text = "$USER";
          color = c.text;
          font_size = 17;
          font_family = "${font} Medium";
          position = "0, -78";
          halign = "center";
          valign = "center";
        }
        # Batería (solo si hay)
        {
          monitor = "";
          text = ''cmd[update:30000] c=$(cat /sys/class/power_supply/BAT*/capacity 2>/dev/null | head -1); [ -n "$c" ] && echo "$c%"'';
          color = c.muted;
          font_size = 14;
          font_family = mono;
          position = "-36, 28";
          halign = "right";
          valign = "bottom";
        }
        # Distribución de teclado actual
        {
          monitor = "";
          text = "$LAYOUT";
          color = c.muted;
          font_size = 14;
          font_family = mono;
          position = "36, 28";
          halign = "left";
          valign = "bottom";
        }
      ];

      input-field = [{
        monitor = "";
        size = "340, 56";
        outline_thickness = 2;
        dots_size = 0.22;
        dots_spacing = 0.4;
        dots_center = true;
        dots_rounding = -1;
        outer_color = "${c.accent} ${c.accent2} 45deg";
        inner_color = c.inner;
        font_color = c.text;
        check_color = "${c.check} ${c.accent} 120deg";
        fail_color = c.fail;
        fade_on_empty = false;
        rounding = 28;
        placeholder_text = "<span foreground=\"${c.hint}\">Contraseña</span>";
        fail_text = "<i>Incorrecta ($ATTEMPTS)</i>";
        capslock_color = c.fail;
        position = "0, -165";
        halign = "center";
        valign = "center";
      }];
  };
in
{
  programs.hyprlock = {
    enable = true;
    settings = mkSettings maxorDark;
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
        # Antes de bloquear, baja el brillo (y lo devuelve al mover el ratón)
        { timeout = 150; on-timeout = "brightnessctl -s set 20%"; on-resume = "brightnessctl -r"; }
        { timeout = 300; on-timeout = "loginctl lock-session"; }
        { timeout = 330; on-timeout = "hyprctl dispatch dpms off"; on-resume = "hyprctl dispatch dpms on"; }
      ];
    };
  };
}
