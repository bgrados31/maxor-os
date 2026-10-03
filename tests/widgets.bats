load helper

DIFF=$'firefox: 149.0 → 150.0, +12.3 MiB\nearlyoom: ∅ → 1.9.0, 52.3 KiB\nfont-util: 1.4.2 → ∅, -228.7 KiB\nbash-interactive: 7.2 MiB\nfwupd.conf: ∅ → ε\nunit-fwupd-refresh.timer: ∅ → ε\nmesa: 26.0.1, 26.0.2 → 26.0.3, +1.1 MiB'

@test "ui_diff cuenta nuevos, actualizados y eliminados" {
  load_lib
  ui_diff "$DIFF" > /dev/null
  [ "$UI_DIFF_ADD" = 1 ]
  [ "$UI_DIFF_UPD" = 2 ]
  [ "$UI_DIFF_DEL" = 1 ]
  [ "$UI_DIFF_CFG" = 2 ]
  [ "$UI_DIFF_CHG" = 1 ]
}

@test "ui_diff muestra los cambios solo de tamaño con ~" {
  load_lib
  out="$(ui_diff "$DIFF")"
  [[ "$out" == *"~ bash-interactive"* ]]
  [[ "$out" == *"~1 changed"* ]]
}

@test "ui_diff esconde los archivos de configuración y muestra los paquetes" {
  load_lib
  out="$(ui_diff "$DIFF")"
  [[ "$out" == *firefox* ]]
  [[ "$out" == *earlyoom* ]]
  [[ "$out" == *"149.0 → 150.0"* ]]
  [[ "$out" != *unit-fwupd* ]]
  [[ "$out" == *"2 configuration files"* ]]
}

@test "ui_diff entiende los colores de nix y recorta con «y N más»" {
  load_lib
  many=""
  for i in $(seq 1 30); do many+=$'\e[31;1m'"pkg$i"$'\e[0m: ∅ → 1.0, 1 KiB\n'; done
  out="$(ui_diff "$many" 5)"
  [[ "$out" == *"and 25 more"* ]]
}

@test "ui_diff con texto vacío no falla" {
  load_lib
  run ui_diff ""
  [ "$status" = 0 ]
}

@test "ui_error muestra causa y qué probar con los glifos de árbol correctos" {
  load_lib
  run ui_error "It failed" "the network is down" "maxor doctor"
  [ "$status" = 0 ]
  [[ "$output" == *"It failed"* ]]
  [[ "$output" == *"├ cause"* ]]
  [[ "$output" == *"└ try"* ]]
}

@test "ui_error solo apunta a los registros si se pide" {
  load_lib
  run ui_error "x" "y"
  [[ "$output" != *"maxor logs"* ]]
  run ui_error "x" "y" "z" log
  [[ "$output" == *"maxor logs --last"* ]]
  [[ "$output" == *"└ details"* ]]
}

@test "ui_error anota el fallo en el registro" {
  load_lib
  ui_error "Disk exploded" "no space" 2> /dev/null
  grep -q "Disk exploded" "$logfile"
}

@test "ui_tabs subraya solo la pestaña activa" {
  load_lib
  out="$(ui_tabs 1 Apps Themes Extras)"
  [[ "$out" == *"Apps"*"Themes"*"Extras"* ]]
  under="$(sed -n '2p' <<< "$out")"
  [ "$(grep -o '▔' <<< "$under" | wc -l)" = 6 ]
}

@test "ui_status separa con puntos y ui_crumbs resalta el último" {
  load_lib
  [[ "$(ui_status nitro sakura "gen 42")" == *"nitro"*"•"*"sakura"*"•"*"gen 42"* ]]
  [[ "$(ui_crumbs maxor help theme)" == *"maxor"*"›"*"help"*"›"*"theme"* ]]
}

@test "ui_hints separa tecla y acción" {
  load_lib
  [[ "$(ui_hints "q:quit" "⏎:go")" == *"q quit"*"⏎ go"* ]]
}
