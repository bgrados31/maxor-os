load helper

@test "die sale con 1 y los errores tipados con su código" {
  load_lib
  run die "plain"
  [ "$status" = 1 ]
  [[ "$output" == *"plain"* ]]
  run die_code "$EX_USAGE" "usage"
  [ "$status" = 2 ]
  run die_code "$EX_NEEDS" "missing"
  [ "$status" = 3 ]
}

@test "need_flake falla con EX_NEEDS si no hay flake" {
  load_lib
  run need_flake
  [ "$status" = 3 ]
  touch "$MAXOR_FLAKE/flake.nix"
  run need_flake
  [ "$status" = 0 ]
}

@test "los errores se anotan en el registro" {
  load_lib
  run die "something failed"
  grep -q "something failed" "$logfile"
}

@test "log_init rota el registro cuando pasa de 512 KB" {
  load_lib
  head -c 600000 /dev/zero | tr '\0' 'x' > "$logfile"
  log_init
  [ -f "$logfile.1" ]
  [ ! -s "$logfile" ] || [ "$(stat -c %s "$logfile")" -lt 1000 ]
}

@test "--verbose escribe el registro también en stderr" {
  load_lib
  MAXOR_VERBOSE=1
  run log INFO "hello"
  [[ "$output" == *"INFO hello"* ]]
}

@test "theme_check acepta un tema válido y rechaza los malos" {
  load_lib
  good='{"bg":"#101010","s":"#202020","s2":"#303030","fg":"#ffffff","mu":"#aaaaaa","ac":"#ff0000","ac2":"#00ff00","on":"#000000"}'
  echo "$good" > "$BATS_TEST_TMPDIR/c.json"
  run theme_check "$BATS_TEST_TMPDIR/c.json"
  [ "$status" = 0 ]
  echo '{"bg":"red"}' > "$BATS_TEST_TMPDIR/c.json"
  run theme_check "$BATS_TEST_TMPDIR/c.json"
  [ "$status" = 1 ]
  [[ "$output" == *"incomplete"* ]]
  echo "${good//#101010/#zzzzzz}" > "$BATS_TEST_TMPDIR/c.json"
  run theme_check "$BATS_TEST_TMPDIR/c.json"
  [ "$status" = 1 ]
  [[ "$output" == *"#rrggbb"* ]]
  echo 'not json' > "$BATS_TEST_TMPDIR/c.json"
  run theme_check "$BATS_TEST_TMPDIR/c.json"
  [ "$status" = 1 ]
  [[ "$output" == *"not valid JSON"* ]]
}

@test "theme_check_style valida los rangos y solo acepta números" {
  load_lib
  echo '{"rounding":8,"gaps_in":4,"anim":100}' > "$BATS_TEST_TMPDIR/s.json"
  run theme_check_style "$BATS_TEST_TMPDIR/s.json"
  [ "$status" = 0 ]
  echo '{"rounding":99}' > "$BATS_TEST_TMPDIR/s.json"
  run theme_check_style "$BATS_TEST_TMPDIR/s.json"
  [ "$status" = 1 ]
  echo '{"rounding":"8; rm -rf /"}' > "$BATS_TEST_TMPDIR/s.json"
  run theme_check_style "$BATS_TEST_TMPDIR/s.json"
  [ "$status" = 1 ]
}

@test "el Lua generado de un tema solo contiene lo permitido" {
  load_lib
  echo '{"rounding":8,"gaps_in":4,"anim":100,"inactive":90}' > "$BATS_TEST_TMPDIR/s.json"
  out="$(theme_style_lua "$BATS_TEST_TMPDIR/s.json")"
  [[ "$out" == *"rounding = 8"* ]]
  [[ "$out" == *"hl.animation"* ]]
  ! grep -qE 'os\.|io\.|require|exec' <<< "$out"
}

@test "mix mezcla dos colores por porcentaje" {
  load_lib
  [ "$(mix '#000000' '#ffffff' 50)" = '#7f7f7f' ]
  [ "$(mix '#ff0000' '#0000ff' 0)" = '#ff0000' ]
}

# ── Flatpak visible en la sesión ─────────────────────────────────────

fake_flatpak_app() { # fake_flatpak_app id Nombre
  local ex="$XDG_DATA_HOME/flatpak/exports/share"
  mkdir -p "$ex/applications" "$ex/icons/hicolor/scalable/apps"
  printf '[Desktop Entry]\nName=%s\nExec=flatpak run %s\n' "$2" "$1" > "$ex/applications/$1.desktop"
  echo '<svg/>' > "$ex/icons/hicolor/scalable/apps/$1.svg"
}

@test "app_flatpak_expose enlaza la entrada de menú, el icono y un comando corto" {
  load_lib
  fake_flatpak_app com.example.FooApp FooApp
  mkdir -p "$HOME/.local/bin"
  PATH="$HOME/.local/bin:$PATH" app_flatpak_expose com.example.FooApp
  [ -L "$XDG_DATA_HOME/applications/com.example.FooApp.desktop" ]
  [ -L "$XDG_DATA_HOME/icons/hicolor/scalable/apps/com.example.FooApp.svg" ]
  [ -x "$HOME/.local/bin/fooapp" ]
  grep -q 'flatpak run com.example.FooApp' "$HOME/.local/bin/fooapp"
}

@test "app_flatpak_expose no pisa un comando que ya existe" {
  load_lib
  fake_flatpak_app com.example.FooApp FooApp
  mkdir -p "$HOME/.local/bin"
  printf '#!/bin/sh\necho mio\n' > "$HOME/.local/bin/fooapp"
  chmod +x "$HOME/.local/bin/fooapp"
  PATH="$HOME/.local/bin:$PATH" app_flatpak_expose com.example.FooApp
  grep -q 'echo mio' "$HOME/.local/bin/fooapp"
}

@test "app_flatpak_unexpose deja todo como estaba" {
  load_lib
  fake_flatpak_app com.example.FooApp FooApp
  mkdir -p "$HOME/.local/bin"
  PATH="$HOME/.local/bin:$PATH" app_flatpak_expose com.example.FooApp
  app_flatpak_unexpose com.example.FooApp
  [ ! -e "$XDG_DATA_HOME/applications/com.example.FooApp.desktop" ]
  [ ! -e "$XDG_DATA_HOME/icons/hicolor/scalable/apps/com.example.FooApp.svg" ]
  [ ! -e "$HOME/.local/bin/fooapp" ]
}

@test "app_refresh_launcher borra los enlaces rotos de apps que ya no están" {
  load_lib
  mkdir -p "$XDG_DATA_HOME/applications"
  ln -s /no/existe.desktop "$XDG_DATA_HOME/applications/viejo.desktop"
  app_refresh_launcher
  [ ! -e "$XDG_DATA_HOME/applications/viejo.desktop" ] && [ ! -L "$XDG_DATA_HOME/applications/viejo.desktop" ]
}

@test "maxor_sudo usa el sudo de siempre, o lee la contraseña de la entrada estándar" {
  load_lib
  sudo() { echo "sudo $*"; }
  run maxor_sudo true
  [ "$output" = "sudo true" ]
  MAXOR_SUDO_STDIN=1 run maxor_sudo nixos-rebuild switch
  [ "$output" = "sudo -S -p  nixos-rebuild switch" ]
}

@test "los temas renombrados (antes en español) se siguen reconociendo por su nombre viejo" {
  load_lib
  [ "$(theme_alias brasa)" = ember ]
  [ "$(theme_alias alba)" = dawn ]
  [ "$(theme_alias maxor-dark)" = maxor-dark ]
}
