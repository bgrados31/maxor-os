load helper

# maxor firstrun: el tema por defecto escrito antes del primer login, una sola vez, sin pisar lo ya elegido.

setup() {
  load_lib
  mkdir -p "$state" "$themes/maxor-dark" "$themes/maxor-light"
  : > "$themes/maxor-dark/wallpaper.png"
  : > "$themes/maxor-light/wallpaper.png"
  # el tema se simula: aquí solo importa cuándo se pide
  cmd_theme() { echo "THEME $*" >> "$BATS_TEST_TMPDIR/theme.log"; }
}

@test "firstrun deja los ajustes de DMS, el tema por defecto y su wallpaper antes de que DMS arranque" {
  rm -f "$dms_settings" "$dms_session"
  run cmd_firstrun
  [ "$status" = 0 ]
  [ "$(cat "$dms_settings")" = "{}" ]
  [ "$(cat "$BATS_TEST_TMPDIR/theme.log")" = "THEME apply maxor-dark" ]
  [ "$(jq -r .wallpaperPath "$dms_session")" = "$themes/maxor-dark/wallpaper.png" ]
  [ -f "$state/firstrun" ]
}

@test "firstrun no hace nada la segunda vez" {
  cmd_firstrun > /dev/null
  rm "$BATS_TEST_TMPDIR/theme.log"
  run cmd_firstrun
  [ "$status" = 0 ]
  [[ "$output" == *"already applied"* ]]
  [ ! -e "$BATS_TEST_TMPDIR/theme.log" ]
}

@test "unos ajustes de DMS y un wallpaper ya elegidos no se pisan" {
  mkdir -p "$(dirname "$dms_settings")" "$(dirname "$dms_session")"
  echo '{"barConfigs":[]}' > "$dms_settings"
  echo '{"wallpaperPath":"/mio.png"}' > "$dms_session"
  run cmd_firstrun
  [ "$status" = 0 ]
  [ "$(jq -c . "$dms_settings")" = '{"barConfigs":[]}' ]
  [ "$(jq -r .wallpaperPath "$dms_session")" = /mio.png ]
}

@test "un tema ya aplicado no se cambia" {
  echo brasa > "$state/current"
  mkdir -p "$(dirname "$dms_session")"
  echo '{"wallpaperPath":"/mio.png"}' > "$dms_session"
  run cmd_firstrun
  [ "$status" = 0 ]
  [ ! -e "$BATS_TEST_TMPDIR/theme.log" ]
  [[ "$output" == *"nothing to change"* ]]
  [ -f "$state/firstrun" ]
}

@test "firstrun con argumentos es un error de uso" {
  run cmd_firstrun --ahora
  [ "$status" = "$EX_USAGE" ]
}

@test "firstrun aplica el tema elegido al instalar, con su wallpaper" {
  rm -f "$dms_settings" "$dms_session"
  MAXOR_FIRSTRUN_THEME=maxor-light run cmd_firstrun
  [ "$status" = 0 ]
  [ "$(cat "$BATS_TEST_TMPDIR/theme.log")" = "THEME apply maxor-light" ]
  [ "$(jq -r .wallpaperPath "$dms_session")" = "$themes/maxor-light/wallpaper.png" ]
}
