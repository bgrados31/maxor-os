load helper

# maxor firstrun: logo y tema por defecto, una sola vez, sin pisar lo ya elegido.

setup() {
  load_lib
  export MAXOR_FIRSTRUN_WAIT=0
  mkdir -p "$(dirname "$dms_settings")" "$state"
  # el tema se simula: aquí solo importa cuándo se pide
  cmd_theme() { echo "THEME $*" >> "$BATS_TEST_TMPDIR/theme.log"; }
}

plain_settings() { echo '{"barConfigs":[{"leftWidgets":["launcherButton","workspaceSwitcher"]}]}' > "$dms_settings"; }
logo() { jq -r '.barConfigs[0].leftWidgets[0].launcherLogoMode // "none"' "$dms_settings"; }

@test "firstrun pone el logo de Maxor y aplica sakura una sola vez" {
  plain_settings
  run cmd_firstrun
  [ "$status" = 0 ]
  [ "$(logo)" = dank ]
  [ "$(cat "$BATS_TEST_TMPDIR/theme.log")" = "THEME apply sakura" ]
  [ -f "$state/firstrun" ]
  # el widget siguiente no se toca
  [ "$(jq -r '.barConfigs[0].leftWidgets[1]' "$dms_settings")" = workspaceSwitcher ]
}

@test "firstrun no hace nada la segunda vez" {
  plain_settings
  cmd_firstrun > /dev/null
  rm "$BATS_TEST_TMPDIR/theme.log"
  run cmd_firstrun
  [ "$status" = 0 ]
  [[ "$output" == *"already applied"* ]]
  [ ! -e "$BATS_TEST_TMPDIR/theme.log" ]
}

@test "sin los ajustes de DMS no marca nada y se reintenta después" {
  rm -f "$dms_settings"
  run cmd_firstrun
  [ "$status" = 0 ]
  [[ "$output" == *"not created"* ]]
  [ ! -e "$state/firstrun" ]
  [ ! -e "$BATS_TEST_TMPDIR/theme.log" ]
}

@test "un botón del lanzador ya personalizado no se pisa" {
  echo '{"barConfigs":[{"leftWidgets":[{"id":"launcherButton","launcherLogoMode":"apps"}]}]}' > "$dms_settings"
  run cmd_firstrun
  [ "$status" = 0 ]
  [ "$(logo)" = apps ]
}

@test "un tema ya aplicado no se cambia" {
  plain_settings
  echo brasa > "$state/current"
  run cmd_firstrun
  [ "$status" = 0 ]
  [ ! -e "$BATS_TEST_TMPDIR/theme.log" ]
  [ "$(logo)" = dank ]
  [ -f "$state/firstrun" ]
}

@test "con todo ya hecho dice que no hay nada que cambiar y deja la marca" {
  echo '{"barConfigs":[{"leftWidgets":[{"id":"launcherButton","launcherLogoMode":"dank"}]}]}' > "$dms_settings"
  echo sakura > "$state/current"
  run cmd_firstrun
  [ "$status" = 0 ]
  [[ "$output" == *"nothing to change"* ]]
  [ -f "$state/firstrun" ]
}

@test "firstrun con argumentos es un error de uso" {
  run cmd_firstrun --ahora
  [ "$status" = "$EX_USAGE" ]
}
