theme_tmp=""
# ── Temas ────────────────────────────────────────────────────────────
theme_meta() { # theme_meta carpeta clave → valor de theme.toml (o vacío)
  grep -m1 "^$2 *=" "$1/theme.toml" 2> /dev/null | sed -E 's/^[^=]*= *"?([^"]*)"?.*/\1/' || true
}

# Devuelve 0 si colors.json es válido; si no, imprime el motivo y devuelve 1.
theme_check() {
  local f="$1"
  jq -e . "$f" > /dev/null 2>&1 || { echo "colors.json no es un JSON válido"; return 1; }
  jq -e 'has("bg") and has("s") and has("s2") and has("fg") and has("mu") and has("ac") and has("ac2") and has("on")' "$f" > /dev/null \
    || { echo "colors.json está incompleto (necesita bg, s, s2, fg, mu, ac, ac2, on)"; return 1; }
  # Solo se aceptan colores #rrggbb: nada más pasa al sistema.
  jq -e '[.bg,.s,.s2,.fg,.mu,.ac,.ac2,.on] | all(type == "string" and test("^#[0-9a-fA-F]{6}$"))' "$f" > /dev/null \
    || { echo "colors.json tiene valores que no son #rrggbb"; return 1; }
  jq -e '(.mode // "dark") | IN("dark", "light")' "$f" > /dev/null \
    || { echo 'mode debe ser "dark" o "light"'; return 1; }
}

theme_list() {
  [ -d "$themes" ] || die "no hay temas en $themes"
  local cur="" rows file mode s2 ac ac2 fg dir n name mark m label l
  [ -f "$state/current" ] && cur="$(cat "$state/current")"
  # Una sola lectura de todos los temas (y solo los válidos), no una por tema.
  rows="$(jq -r 'select([.bg,.s,.s2,.fg,.mu,.ac,.ac2,.on] | all(type == "string" and test("^#[0-9a-fA-F]{6}$")))
    | [input_filename, (.mode // "dark"), .s2, .ac, .ac2, .fg] | @tsv' "$themes"/*/colors.json 2> /dev/null || true)"
  ui_open "maxor · temas"
  for m in dark light; do
    label="Oscuros"; [ "$m" = light ] && label="Claros"
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
      mark=" "; [ "$n" = "$cur" ] && mark="${E_AC}●${E_FG}"
      ui_truncv name "$name" 24
      printf -v n '%-13s' "$n"
      ui_line " $mark $n $(ui_swatch "$s2" "$ac" "$ac2" "$fg")  ${E_MU}${name}${E_FG}"
    done <<< "$rows"
  done
  ui_line ""
  ui_line " ${E_MU}● activo · maxor theme apply <nombre>${E_FG}"
  ui_close
}

theme_apply() {
  local name="$1" dir="$themes/$1" err
  [ -f "$dir/colors.json" ] || die "el tema '$name' no existe (maxor theme list)"
  err="$(theme_check "$dir/colors.json")" || die "tema '$name': $err"

  local bg s s2 fg mu ac ac2 on mode
  bg="$(jq -r .bg "$dir/colors.json")"; s="$(jq -r .s "$dir/colors.json")"; s2="$(jq -r .s2 "$dir/colors.json")"
  fg="$(jq -r .fg "$dir/colors.json")"; mu="$(jq -r .mu "$dir/colors.json")"
  ac="$(jq -r .ac "$dir/colors.json")"; ac2="$(jq -r .ac2 "$dir/colors.json")"; on="$(jq -r .on "$dir/colors.json")"
  mode="$(jq -r '.mode // "dark"' "$dir/colors.json")"

  mkdir -p "$cfg/current" "$state"
  local steps=()

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
    steps+=("ok|DMS: barra, launcher, kitty, Hyprland y GTK")
    if command -v dms > /dev/null && dms ipc call theme "$([ "$mode" = light ] && echo light || echo dark)" > /dev/null 2>&1; then
      steps+=("ok|Modo $([ "$mode" = light ] && echo claro || echo oscuro) del escritorio")
    else
      steps+=("warn|No pude cambiar el modo claro/oscuro (¿DMS en marcha?)")
    fi
  else
    steps+=("warn|DMS no está configurado todavía (abre SUPER + , una vez)")
  fi

  # 2) hyprlock: rellena la plantilla con la paleta
  if [ -f "$tpl/hyprlock.conf.tpl" ]; then
    local bright="0.60"; [ "$mode" = light ] && bright="0.95"
    sed -e "s|@FG_RGB@|$(rgb "$fg")|g" -e "s|@AC_RGB@|$(rgb "$ac")|g" -e "s|@BG_RGB@|$(rgb "$bg")|g" \
      -e "s|@FG_HEX@|$(hex "$fg")|g" -e "s|@LOCK_BRIGHTNESS@|$bright|g" \
      "$tpl/hyprlock.conf.tpl" > "$cfg/current/hyprlock.conf"
    steps+=("ok|Pantalla de bloqueo")
  fi

  # 3) forma del tema (esquinas, espacios, desenfoque, animaciones) → Lua de Hyprland
  local shape=""
  if [ -f "$dir/style.json" ]; then
    if err="$(theme_check_style "$dir/style.json")"; then
      theme_style_lua "$dir/style.json" > "$cfg/current/hyprland.lua"
      shape="$(jq -r '"esquinas \(.rounding // "–") px · animaciones \(.anim // "–") %"' "$dir/style.json")"
      steps+=("ok|Forma: $shape")
    else
      rm -f "$cfg/current/hyprland.lua"
      steps+=("warn|style.json ignorado: $err")
    fi
  else
    rm -f "$cfg/current/hyprland.lua"
  fi
  if [ -n "${HYPRLAND_INSTANCE_SIGNATURE:-}" ] && command -v hyprctl > /dev/null; then hyprctl reload > /dev/null 2>&1 || true; fi

  # 4) wallpaper del tema
  if [ -f "$dir/wallpaper.png" ] && command -v dms > /dev/null; then
    if dms ipc call wallpaper set "$dir/wallpaper.png" > /dev/null 2>&1; then
      steps+=("ok|Fondo de pantalla")
    else
      steps+=("warn|No pude poner el fondo de pantalla")
    fi
  fi

  # 5) kitty relee su configuración
  pkill -USR1 -x kitty 2> /dev/null || true
  steps+=("ok|kitty recargado")

  # La ventana ya se dibuja con los colores del tema nuevo.
  ui_palette
  local title; title="$(theme_meta "$dir" name)"; [ -n "$title" ] || title="$name"
  ui_open "maxor · tema aplicado"
  ui_line ""
  ui_line " ${E_BOLD}$(ui_c "$E_AC" "$title")${E_NB}  $(ui_c "$E_MU" "($name · $([ "$mode" = light ] && echo claro || echo oscuro))")"
  ui_line " $(ui_swatch "$bg" "$s2" "$mu" "$fg" "$ac" "$ac2")"
  ui_line ""
  local st
  for st in "${steps[@]}"; do ui_row "${st%%|*}" "${st#*|}"; done
  ui_line ""
  ui_line " $(ui_c "$E_MU" "maxor theme undo  para volver al anterior")"
  ui_close
}

theme_undo() {
  [ -s "$state/history" ] || die "no hay tema anterior"
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
      -*) die "opción desconocida: $a" ;;
      *) src="$a" ;;
    esac
  done
  [ -n "$src" ] || die "uso: maxor theme install <carpeta|archivo.tar.gz> [--force]"
  local dir err id dest
  theme_tmp="$(mktemp -d)"
  trap 'rm -rf "${theme_tmp:-}"' EXIT

  if [ -d "$src" ]; then
    dir="$src"
  elif [ -f "$src" ]; then
    [ "$(stat -c %s "$src")" -le $((30 * 1024 * 1024)) ] || die "el archivo pesa más de 30 MB"
    tar -tzvf "$src" > "$theme_tmp/list" 2> /dev/null || die "no puedo leer '$src' (se espera un .tar.gz)"
    # Solo archivos y carpetas normales; nada de enlaces, rutas absolutas ni '..'.
    if awk '{ t = substr($1, 1, 1); if (t != "-" && t != "d") bad = 1 } END { exit !bad }' "$theme_tmp/list"; then
      die "el archivo contiene enlaces o archivos especiales: se rechaza"
    fi
    if tar -tzf "$src" | grep -q -E '(^/|(^|/)\.\.(/|$))'; then
      die "el archivo contiene rutas peligrosas: se rechaza"
    fi
    mkdir "$theme_tmp/x"
    tar -xzf "$src" -C "$theme_tmp/x" --no-same-owner --no-same-permissions
    if [ -f "$theme_tmp/x/colors.json" ]; then
      dir="$theme_tmp/x"
    else
      dir="$(find "$theme_tmp/x" -mindepth 1 -maxdepth 1 -type d | head -n1)"
      [ -n "$dir" ] && [ -f "$dir/colors.json" ] || die "el archivo no contiene colors.json"
    fi
  else
    die "'$src' no existe"
  fi

  [ -f "$dir/colors.json" ] || die "falta colors.json en '$src'"
  err="$(theme_check "$dir/colors.json")" || die "$err"
  if [ -f "$dir/style.json" ]; then err="$(theme_check_style "$dir/style.json")" || die "$err"; fi

  id="$(theme_meta "$dir" id)"
  [ -n "$id" ] || id="$(basename "$(readlink -f "$dir")")"
  id="$(printf '%s' "$id" | tr '[:upper:]' '[:lower:]')"
  printf '%s' "$id" | grep -q -E '^[a-z0-9][a-z0-9-]{1,31}$' || die "el id '$id' no es válido (minúsculas, números y guiones; 2 a 32 caracteres)"

  dest="$themes/$id"
  if [ -L "$dest" ]; then die "'$id' es un tema oficial y no se puede reemplazar"; fi
  if [ -e "$dest" ]; then
    [ "$force" = 1 ] || die "el tema '$id' ya está instalado (usa --force para reemplazarlo)"
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
      ui_say warn "wallpaper.png ignorado (no es un PNG válido o pesa más de 20 MB)"
    fi
  fi
  if [ -f "$dir/style.json" ]; then
    jq '{rounding, gaps_in, gaps_out, border_size, blur_size, blur_passes, inactive, anim} | with_entries(select(.value != null))' "$dir/style.json" > "$dest/style.json"
  fi
  ui_open "maxor · tema instalado"
  ui_line ""
  ui_line " ${E_BOLD}$(ui_c "$E_AC" "$id")${E_NB}  $(ui_swatch "$(jq -r .s2 "$dest/colors.json")" "$(jq -r .ac "$dest/colors.json")" "$(jq -r .ac2 "$dest/colors.json")" "$(jq -r .fg "$dest/colors.json")")"
  ui_line ""
  ui_row ok "Validado: ocho colores #rrggbb"
  if [ -f "$dest/style.json" ]; then ui_row ok "Validada la forma (style.json: solo números)"; fi
  ui_row ok "Copiado a $(ui_trunc "${dest/#$HOME/~}" $((ui_w - 24)))"
  ui_row info "No se ejecutó nada del tema"
  ui_line ""
  ui_line " $(ui_c "$E_MU" "maxor theme apply $id")"
  ui_close
}

theme_export() { # theme_export <nombre>
  local name="${1:-}" dir out
  [ -n "$name" ] || die "uso: maxor theme export <nombre>"
  dir="$themes/$name"
  [ -f "$dir/colors.json" ] || die "el tema '$name' no existe"
  out="$PWD/$name.maxortheme"
  local files=(colors.json)
  [ -f "$dir/theme.toml" ] && files+=(theme.toml)
  [ -f "$dir/wallpaper.png" ] && files+=(wallpaper.png)
  [ -f "$dir/style.json" ] && files+=(style.json)
  tar -czhf "$out" -C "$dir" "${files[@]}"
  ui_say ok "Exportado: ${out/#$HOME/~}"
}
