# ── Temas: list | current | apply | undo | install | export ──────────
maxor_cmd theme appearance "list current apply undo install export"

theme_tmp=""

theme_meta() { # theme_meta carpeta clave → valor de theme.toml (o vacío)
  grep -m1 "^$2 *=" "$1/theme.toml" 2> /dev/null | sed -E 's/^[^=]*= *"?([^"]*)"?.*/\1/' || true
}

# Devuelve 0 si colors.json es válido; si no, imprime el motivo y devuelve 1.
theme_check() {
  local f="$1"
  jq -e . "$f" > /dev/null 2>&1 || { t theme.check_json; echo; return 1; }
  jq -e 'has("bg") and has("s") and has("s2") and has("fg") and has("mu") and has("ac") and has("ac2") and has("on")' "$f" > /dev/null \
    || { t theme.check_missing; echo; return 1; }
  # Solo se aceptan colores #rrggbb: nada más pasa al sistema.
  jq -e '[.bg,.s,.s2,.fg,.mu,.ac,.ac2,.on] | all(type == "string" and test("^#[0-9a-fA-F]{6}$"))' "$f" > /dev/null \
    || { t theme.check_hex; echo; return 1; }
  jq -e '(.mode // "dark") | IN("dark", "light")' "$f" > /dev/null \
    || { t theme.check_mode; echo; return 1; }
}

theme_mode_word() { # theme_mode_word variable modo
  if [ "$2" = light ]; then msg "$1" @theme.mode_light; else msg "$1" @theme.mode_dark; fi
}

theme_list() {
  [ -d "$themes" ] || die_code "$EX_NEEDS" @theme.none "$themes"
  local cur="" rows file mode s2 ac ac2 fg dir n name mark m label l
  [ -f "$state/current" ] && cur="$(cat "$state/current")"
  # Una sola lectura de todos los temas (y solo los válidos), no una por tema.
  rows="$(jq -r 'select([.bg,.s,.s2,.fg,.mu,.ac,.ac2,.on] | all(type == "string" and test("^#[0-9a-fA-F]{6}$")))
    | [input_filename, (.mode // "dark"), .s2, .ac, .ac2, .fg] | @tsv' "$themes"/*/colors.json 2> /dev/null || true)"
  ui_intro @theme.title_list
  for m in dark light; do
    if [ "$m" = light ]; then msg label @theme.light; else msg label @theme.dark; fi
    ui_section "$label"
    while IFS=$'\t' read -r file mode s2 ac ac2 fg; do
      [ "$mode" = "$m" ] || continue
      dir="${file%/colors.json}"
      n="${dir##*/}"
      name=""
      if [ -f "$dir/theme.toml" ]; then
        while IFS= read -r l; do
          if [[ "$l" =~ ^name\ *=\ *\"?([^\"]*) ]]; then name="${BASH_REMATCH[1]}"; break; fi
        done < "$dir/theme.toml"
      fi
      [ -n "$name" ] || name="$n"
      mark=" "; [ "$n" = "$cur" ] && mark="${E_AC}●${E_RST}"
      ui_truncv name "$name" 24
      printf -v n '%-13s' "$n"
      ui_text " $mark $n $(ui_swatch "$s2" "$ac" "$ac2" "$fg")  ${E_MU}${name}${E_RST}"
    done <<< "$rows"
  done
  msg l @theme.legend
  ui_outro "${E_MU}${l}${E_RST}"
}

theme_apply() {
  local name="$1" dir="$themes/$1" err
  [ -f "$dir/colors.json" ] || die_code "$EX_NEEDS" @theme.missing "$name"
  err="$(theme_check "$dir/colors.json")" || die @theme.invalid "$name" "$err"

  local bg s s2 fg mu ac ac2 on mode mw
  bg="$(jq -r .bg "$dir/colors.json")"; s="$(jq -r .s "$dir/colors.json")"; s2="$(jq -r .s2 "$dir/colors.json")"
  fg="$(jq -r .fg "$dir/colors.json")"; mu="$(jq -r .mu "$dir/colors.json")"
  ac="$(jq -r .ac "$dir/colors.json")"; ac2="$(jq -r .ac2 "$dir/colors.json")"; on="$(jq -r .on "$dir/colors.json")"
  mode="$(jq -r '.mode // "dark"' "$dir/colors.json")"
  theme_mode_word mw "$mode"

  mkdir -p "$cfg/current" "$state"
  local steps=() sm

  # historial para `undo`
  local prev=""
  [ -f "$state/current" ] && prev="$(cat "$state/current")"
  if [ -n "$prev" ] && [ "$prev" != "$name" ]; then echo "$prev" >> "$state/history"; fi
  echo "$name" > "$state/current"

  # 1) DMS: tema propio. DMS recolorea barra, kitty, Hyprland y GTK desde aquí.
  jq -n \
    --arg name "$(theme_meta "$dir" name)" \
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
    msg sm @theme.step_dms; steps+=("ok|$sm")
    if command -v dms > /dev/null && dms ipc call theme "$([ "$mode" = light ] && echo light || echo dark)" > /dev/null 2>&1; then
      msg sm @theme.step_mode "$mw"; steps+=("ok|$sm")
    else
      msg sm @theme.step_mode_fail; steps+=("warn|$sm")
    fi
  else
    msg sm @theme.step_dms_missing; steps+=("warn|$sm")
  fi

  # 2) hyprlock: rellena la plantilla con la paleta
  if [ -f "$tpl/hyprlock.conf.tpl" ]; then
    local bright="0.60"; [ "$mode" = light ] && bright="0.95"
    sed -e "s|@FG_RGB@|$(rgb "$fg")|g" -e "s|@AC_RGB@|$(rgb "$ac")|g" -e "s|@BG_RGB@|$(rgb "$bg")|g" \
      -e "s|@FG_HEX@|$(hex "$fg")|g" -e "s|@LOCK_BRIGHTNESS@|$bright|g" \
      "$tpl/hyprlock.conf.tpl" > "$cfg/current/hyprlock.conf"
    msg sm @theme.step_lock; steps+=("ok|$sm")
  fi

  # 3) forma del tema (esquinas, espacios, desenfoque, animaciones) → Lua de Hyprland
  if [ -f "$dir/style.json" ]; then
    if err="$(theme_check_style "$dir/style.json")"; then
      theme_style_lua "$dir/style.json" > "$cfg/current/hyprland.lua"
      local rnd anm
      rnd="$(jq -r '.rounding // "–"' "$dir/style.json")"; anm="$(jq -r '.anim // "–"' "$dir/style.json")"
      msg sm @theme.step_shape "$rnd" "$anm"; steps+=("ok|$sm")
    else
      rm -f "$cfg/current/hyprland.lua"
      msg sm @theme.step_shape_bad "$err"; steps+=("warn|$sm")
    fi
  else
    rm -f "$cfg/current/hyprland.lua"
  fi
  if [ -n "${HYPRLAND_INSTANCE_SIGNATURE:-}" ] && command -v hyprctl > /dev/null; then hyprctl reload > /dev/null 2>&1 || true; fi

  # 4) wallpaper del tema
  if [ -f "$dir/wallpaper.png" ] && command -v dms > /dev/null; then
    if dms ipc call wallpaper set "$dir/wallpaper.png" > /dev/null 2>&1; then
      msg sm @theme.step_wall; steps+=("ok|$sm")
    else
      msg sm @theme.step_wall_fail; steps+=("warn|$sm")
    fi
  fi

  # 5) kitty relee su configuración
  pkill -USR1 -x kitty 2> /dev/null || true
  msg sm @theme.step_kitty; steps+=("ok|$sm")

  # La ventana ya se dibuja con los colores del tema nuevo.
  ui_palette
  local title hint; title="$(theme_meta "$dir" name)"; [ -n "$title" ] || title="$name"
  ui_intro @theme.applied_title
  ui_text ""
  ui_text " ${E_BOLD}${E_AC}${title}${E_RST}${E_NB}  ${E_MU}($name · $mw)${E_RST}"
  ui_text " $(ui_swatch "$bg" "$s2" "$mu" "$fg" "$ac" "$ac2")"
  ui_text ""
  local st
  for st in "${steps[@]}"; do ui_row "${st%%|*}" "${st#*|}"; done
  msg hint @theme.applied_hint
  ui_outro "${E_MU}${hint}${E_RST}"
}

theme_undo() {
  [ -s "$state/history" ] || die_code "$EX_NEEDS" @theme.no_previous
  local last
  last="$(tail -n1 "$state/history")"
  sed -i '$d' "$state/history"
  # no volver a registrar el actual al deshacer
  rm -f "$state/current"
  theme_apply "$last"
}

theme_install() { # theme_install <carpeta|archivo.tar.gz> [--force]
  local src="" force=0 a
  for a in "$@"; do
    case "$a" in
      --force) force=1 ;;
      -*) die_code "$EX_USAGE" @err.unknown_option "$a" ;;
      *) src="$a" ;;
    esac
  done
  [ -n "$src" ] || die_code "$EX_USAGE" @theme.inst_usage
  local dir err id dest
  theme_tmp="$(mktemp -d)"
  trap 'rm -rf "${theme_tmp:-}"' EXIT

  if [ -d "$src" ]; then
    dir="$src"
  elif [ -f "$src" ]; then
    [ "$(stat -c %s "$src")" -le $((30 * 1024 * 1024)) ] || die @theme.too_big
    tar -tzvf "$src" > "$theme_tmp/list" 2> /dev/null || die @theme.unreadable "$src"
    # Solo archivos y carpetas normales; nada de enlaces, rutas absolutas ni '..'.
    if awk '{ t = substr($1, 1, 1); if (t != "-" && t != "d") bad = 1 } END { exit !bad }' "$theme_tmp/list"; then
      die @theme.has_links
    fi
    if tar -tzf "$src" | grep -q -E '(^/|(^|/)\.\.(/|$))'; then
      die @theme.has_unsafe
    fi
    mkdir "$theme_tmp/x"
    tar -xzf "$src" -C "$theme_tmp/x" --no-same-owner --no-same-permissions
    if [ -f "$theme_tmp/x/colors.json" ]; then
      dir="$theme_tmp/x"
    else
      dir="$(find "$theme_tmp/x" -mindepth 1 -maxdepth 1 -type d | head -n1)"
      [ -n "$dir" ] && [ -f "$dir/colors.json" ] || die @theme.no_colors_in
    fi
  else
    die_code "$EX_NEEDS" @theme.not_found "$src"
  fi

  [ -f "$dir/colors.json" ] || die_code "$EX_NEEDS" @theme.no_colors "$src"
  err="$(theme_check "$dir/colors.json")" || die "$err"
  if [ -f "$dir/style.json" ]; then err="$(theme_check_style "$dir/style.json")" || die "$err"; fi

  id="$(theme_meta "$dir" id)"
  [ -n "$id" ] || id="$(basename "$(readlink -f "$dir")")"
  id="$(printf '%s' "$id" | tr '[:upper:]' '[:lower:]')"
  printf '%s' "$id" | grep -q -E '^[a-z0-9][a-z0-9-]{1,31}$' || die @theme.bad_id "$id"

  dest="$themes/$id"
  if [ -L "$dest" ]; then die @theme.official "$id"; fi
  if [ -e "$dest" ]; then
    [ "$force" = 1 ] || die @theme.exists "$id"
    rm -rf "$dest"
  fi
  mkdir -p "$dest"
  # Solo se copian los archivos permitidos; colors.json se reescribe con las claves conocidas.
  jq '{bg, s, s2, fg, mu, ac, ac2, on, mode: (.mode // "dark")}' "$dir/colors.json" > "$dest/colors.json"
  [ -f "$dir/theme.toml" ] && [ "$(stat -c %s "$dir/theme.toml")" -le 4096 ] && cp "$dir/theme.toml" "$dest/theme.toml"
  if [ -f "$dir/wallpaper.png" ]; then
    if [ "$(stat -c %s "$dir/wallpaper.png")" -le $((20 * 1024 * 1024)) ] && [ "$(head -c 8 "$dir/wallpaper.png" | od -An -tx1 | tr -d ' \n')" = "89504e470d0a1a0a" ]; then
      cp "$dir/wallpaper.png" "$dest/wallpaper.png"
    else
      ui_say warn @theme.wall_ignored
    fi
  fi
  if [ -f "$dir/style.json" ]; then
    jq '{rounding, gaps_in, gaps_out, border_size, blur_size, blur_passes, inactive, anim} | with_entries(select(.value != null))' "$dir/style.json" > "$dest/style.json"
  fi
  local shown
  ui_intro @theme.installed_title
  ui_text ""
  ui_text " ${E_BOLD}${E_AC}${id}${E_RST}${E_NB}  $(ui_swatch "$(jq -r .s2 "$dest/colors.json")" "$(jq -r .ac "$dest/colors.json")" "$(jq -r .ac2 "$dest/colors.json")" "$(jq -r .fg "$dest/colors.json")")"
  ui_text ""
  ui_row ok @theme.v_colors
  if [ -f "$dest/style.json" ]; then ui_row ok @theme.v_shape; fi
  ui_truncv shown "${dest/#$HOME/~}" $((ui_w - 24))
  ui_row ok @theme.copied "$shown"
  ui_row info @theme.nothing_run
  ui_outro "${E_MU}maxor theme apply $id${E_RST}"
}

theme_export() { # theme_export <nombre>
  local name="${1:-}" dir out
  [ -n "$name" ] || die_code "$EX_USAGE" @theme.exp_usage
  dir="$themes/$name"
  [ -f "$dir/colors.json" ] || die_code "$EX_NEEDS" @theme.exp_missing "$name"
  out="$PWD/$name.maxortheme"
  local files=(colors.json)
  [ -f "$dir/theme.toml" ] && files+=(theme.toml)
  [ -f "$dir/wallpaper.png" ] && files+=(wallpaper.png)
  [ -f "$dir/style.json" ] && files+=(style.json)
  tar -czhf "$out" -C "$dir" "${files[@]}"
  ui_say ok @theme.exported "${out/#$HOME/~}"
}

cmd_theme() {
  local sub="${1:-}"
  shift || true
  case "$sub" in
    list) theme_list ;;
    current)
      if [ -f "$state/current" ]; then cat "$state/current"; else t theme.none_applied; echo; fi
      ;;
    apply)
      [ -n "${1:-}" ] || die_code "$EX_USAGE" @theme.apply_usage
      theme_apply "$1"
      ;;
    undo) theme_undo ;;
    install) theme_install "$@" ;;
    export) theme_export "${1:-}" ;;
    *) usage_error theme ;;
  esac
}
