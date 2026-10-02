{ config, pkgs, lib, ... }:

# Motor de temas de Maxor OS (fase 3).
#
#   maxor theme list | apply <nombre> | undo | current
#
# Un tema es una carpeta con colors.json (+ theme.toml y wallpaper.png).
# Los temas oficiales vienen en el sistema; los tuyos van en
# ~/.local/share/maxor/themes/<nombre>/ y se aplican igual, sin rebuild ni sudo.
# Un tema solo contiene datos: el CLI nunca ejecuta nada que venga de él.
let
  mkTheme = { id, name, desc, palette, mode ? "dark" }:
    let
      colors = pkgs.writeText "colors.json" (builtins.toJSON (palette // { inherit mode; }));
      meta = pkgs.writeText "theme.toml" ''
        name = "${name}"
        id = "${id}"
        author = "Maxor OS"
        version = "1.0.0"
        license = "CC0-1.0"
        description = "${desc}"
        mode = "${mode}"
      '';
    in
    pkgs.runCommand "maxor-theme-${id}" { nativeBuildInputs = [ pkgs.imagemagick ]; } ''
      mkdir -p $out
      cp ${colors} $out/colors.json
      cp ${meta} $out/theme.toml
      magick -size 2560x1600 radial-gradient:'${palette.s2}'-'${palette.bg}' \
        -attenuate 0.12 +noise Gaussian -colorspace sRGB $out/wallpaper.png
    '';

  officialThemes = {
    sakura = mkTheme {
      id = "sakura";
      name = "Sakura nocturna";
      desc = "Ciruela oscura con rosa y durazno.";
      palette = { bg = "#120b12"; s = "#1d121d"; s2 = "#2a1a2a"; fg = "#fbe9f2"; mu = "#a88a9d"; ac = "#ff86b8"; ac2 = "#ffc2a6"; on = "#1b0b14"; };
    };
    glaciar = mkTheme {
      id = "glaciar";
      name = "Glaciar";
      desc = "Azul marino profundo con cian hielo.";
      palette = { bg = "#07111a"; s = "#0e1c29"; s2 = "#152a3b"; fg = "#e4f4fb"; mu = "#7f9db0"; ac = "#5fd4f4"; ac2 = "#8af0d0"; on = "#06141d"; };
    };
  };

  maxor = pkgs.writeShellApplication {
    name = "maxor";
    runtimeInputs = with pkgs; [ jq coreutils gnused gnugrep procps ];
    excludeShellChecks = [ "SC2155" "SC2086" "SC2012" ];
    text = ''
      data="''${XDG_DATA_HOME:-$HOME/.local/share}/maxor"
      cfg="''${XDG_CONFIG_HOME:-$HOME/.config}/maxor"
      state="''${XDG_STATE_HOME:-$HOME/.local/state}/maxor"
      themes="$data/themes"
      tpl="$data/templates"
      dms_settings="''${XDG_CONFIG_HOME:-$HOME/.config}/DankMaterialShell/settings.json"

      die() { echo "maxor: $*" >&2; exit 1; }

      usage() {
        cat <<'EOF'
      maxor — herramienta de Maxor OS

        maxor theme list              temas instalados
        maxor theme current           tema activo
        maxor theme apply <nombre>    aplicar un tema
        maxor theme undo              volver al tema anterior
      EOF
      }

      hex() { printf '%s' "''${1#\#}"; }
      rgb() { local h; h="$(hex "$1")"; printf '%d, %d, %d' "0x''${h:0:2}" "0x''${h:2:2}" "0x''${h:4:2}"; }
      # mix A B P  →  A con P % de B
      mix() {
        local a b p out="#" i x y
        a="$(hex "$1")"; b="$(hex "$2")"; p="$3"
        for i in 0 2 4; do
          x=$((0x''${a:i:2})); y=$((0x''${b:i:2}))
          out+="$(printf '%02x' $(( (x * (100 - p) + y * p) / 100 )))"
        done
        printf '%s' "$out"
      }

      theme_list() {
        local cur=""; [ -f "$state/current" ] && cur="$(cat "$state/current")"
        [ -d "$themes" ] || die "no hay temas en $themes"
        for d in "$themes"/*/; do
          [ -f "$d/colors.json" ] || continue
          n="$(basename "$d")"
          if [ "$n" = "$cur" ]; then echo "* $n"; else echo "  $n"; fi
        done
      }

      theme_apply() {
        local name="$1" dir="$themes/$1"
        [ -f "$dir/colors.json" ] || die "el tema '$name' no existe (maxor theme list)"
        jq -e 'has("bg") and has("s") and has("s2") and has("fg") and has("mu") and has("ac") and has("ac2") and has("on")' \
          "$dir/colors.json" >/dev/null || die "colors.json de '$name' está incompleto"
        # Solo se aceptan colores #rrggbb: nada más pasa al sistema.
        jq -e '[.bg,.s,.s2,.fg,.mu,.ac,.ac2,.on] | all(test("^#[0-9a-fA-F]{6}$"))' \
          "$dir/colors.json" >/dev/null || die "colors.json de '$name' tiene colores inválidos"

        local bg s s2 fg mu ac ac2 on
        bg="$(jq -r .bg "$dir/colors.json")"; s="$(jq -r .s "$dir/colors.json")"; s2="$(jq -r .s2 "$dir/colors.json")"
        fg="$(jq -r .fg "$dir/colors.json")"; mu="$(jq -r .mu "$dir/colors.json")"
        ac="$(jq -r .ac "$dir/colors.json")"; ac2="$(jq -r .ac2 "$dir/colors.json")"; on="$(jq -r .on "$dir/colors.json")"

        mkdir -p "$cfg/current" "$state"

        # historial para `undo`
        local prev=""; [ -f "$state/current" ] && prev="$(cat "$state/current")"
        if [ -n "$prev" ] && [ "$prev" != "$name" ]; then echo "$prev" >> "$state/history"; fi
        echo "$name" > "$state/current"

        # 1) DMS: tema propio. DMS recolorea barra, kitty, Hyprland y GTK desde aquí.
        jq -n \
          --arg name "$(grep -m1 '^name' "$dir/theme.toml" 2>/dev/null | sed 's/^name *= *"\(.*\)"/\1/' || echo "$name")" \
          --arg primary "$ac" --arg primaryText "$on" --arg primaryContainer "$(mix "$ac" "$bg" 62)" \
          --arg secondary "$ac2" --arg surface "$s" --arg surfaceText "$fg" --arg surfaceVariant "$s2" \
          --arg surfaceVariantText "$mu" --arg outline "$(mix "$mu" "$bg" 45)" --arg bg "$bg" \
          --arg low "$(mix "$bg" "$s" 50)" --arg cont "$(mix "$s" "$s2" 50)" --arg high "$s2" --arg highest "$(mix "$s2" "$fg" 8)" \
          '{name:$name, primary:$primary, primaryText:$primaryText, primaryContainer:$primaryContainer,
            secondary:$secondary, surface:$surface, surfaceText:$surfaceText, surfaceVariant:$surfaceVariant,
            surfaceVariantText:$surfaceVariantText, surfaceTint:$primary, background:$bg, backgroundText:$surfaceText,
            outline:$outline, surfaceContainerLowest:$bg, surfaceContainerLow:$low, surfaceContainer:$cont,
            surfaceContainerHigh:$high, surfaceContainerHighest:$highest}' \
          > "$cfg/current/dms-theme.json.tmp"
        mv "$cfg/current/dms-theme.json.tmp" "$cfg/current/dms-theme.json"

        if [ -f "$dms_settings" ]; then
          jq --arg f "$cfg/current/dms-theme.json" '.currentThemeName = "custom" | .customThemeFile = $f' \
            "$dms_settings" > "$dms_settings.tmp" && mv "$dms_settings.tmp" "$dms_settings"
        fi

        # 2) hyprlock: rellena la plantilla con la paleta
        if [ -f "$tpl/hyprlock.conf.tpl" ]; then
          sed -e "s|@FG_RGB@|$(rgb "$fg")|g" -e "s|@AC_RGB@|$(rgb "$ac")|g" -e "s|@BG_RGB@|$(rgb "$bg")|g" \
              -e "s|@FG_HEX@|$(hex "$fg")|g" "$tpl/hyprlock.conf.tpl" > "$cfg/current/hyprlock.conf"
        fi

        # 3) wallpaper del tema
        if [ -f "$dir/wallpaper.png" ] && command -v dms >/dev/null; then
          dms ipc call wallpaper set "$dir/wallpaper.png" >/dev/null 2>&1 || true
        fi

        # 4) kitty relee su configuración
        pkill -USR1 -x kitty 2>/dev/null || true

        echo "tema aplicado: $name"
      }

      theme_undo() {
        [ -s "$state/history" ] || die "no hay tema anterior"
        local last; last="$(tail -n1 "$state/history")"
        sed -i '$d' "$state/history"
        # no volver a registrar el actual al deshacer
        rm -f "$state/current"
        theme_apply "$last"
      }

      case "''${1:-}" in
        theme)
          case "''${2:-}" in
            list) theme_list ;;
            current) cat "$state/current" 2>/dev/null || echo "(ninguno)" ;;
            apply) [ -n "''${3:-}" ] || die "uso: maxor theme apply <nombre>"; theme_apply "$3" ;;
            undo) theme_undo ;;
            *) usage; exit 1 ;;
          esac ;;
        ""|-h|--help|help) usage ;;
        *) usage; exit 1 ;;
      esac
    '';
  };
in
{
  home.packages = [ maxor ];

  # Temas oficiales: carpetas de solo lectura junto a los tuyos.
  home.file = (lib.mapAttrs'
    (id: t: lib.nameValuePair ".local/share/maxor/themes/${id}" { source = t; })
    officialThemes) // (lib.mapAttrs'
    (id: t: lib.nameValuePair "Pictures/Wallpapers/maxor-${id}.png" { source = "${t}/wallpaper.png"; })
    officialThemes);
}
