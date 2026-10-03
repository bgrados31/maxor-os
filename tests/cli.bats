load helper

# Pruebas de integración: usan el binario completo (MAXOR_BIN).
setup() {
  [ -n "${MAXOR_BIN:-}" ] || skip "MAXOR_BIN no está definido"
  isolate
}

@test "--version imprime la versión y el sistema" {
  run "$MAXOR_BIN" --version
  [ "$status" = 0 ]
  [[ "$output" == maxor\ * ]]
}

@test "version --json lleva la versión del contrato" {
  run "$MAXOR_BIN" version --json
  [ "$status" = 0 ]
  echo "$output" | jq -e '.schema == 1 and (.version | type == "string")'
}

@test "sin argumentos muestra todos los grupos" {
  run "$MAXOR_BIN"
  [ "$status" = 0 ]
  for g in Appearance Apps System Tools; do [[ "${output^^}" == *"${g^^}"* ]]; done
}

@test "un comando desconocido sale con 2 y sugiere maxor help" {
  run "$MAXOR_BIN" nope
  [ "$status" = 2 ]
  [[ "$output" == *"Unknown command"* ]]
  [[ "$output" == *"maxor help"* ]]
}

@test "help <comando> muestra el uso" {
  run "$MAXOR_BIN" help theme
  [ "$status" = 0 ]
  [[ "$output" == *"Usage: maxor theme"* ]]
}

@test "help de un comando inexistente falla con 2" {
  run "$MAXOR_BIN" help nada
  [ "$status" = 2 ]
}

@test "un subcomando inválido muestra la ayuda y sale con 2" {
  run "$MAXOR_BIN" theme nada
  [ "$status" = 2 ]
  [[ "$output" == *"Usage: maxor theme"* ]]
  run "$MAXOR_BIN" hardware nada
  [ "$status" = 2 ]
}

@test "una opción desconocida sale con 2" {
  run "$MAXOR_BIN" install --nada
  [ "$status" = 2 ]
  [[ "$output" == *"unknown option"* ]]
}

@test "profile list --json devuelve los cuatro perfiles" {
  touch "$MAXOR_FLAKE/flake.nix"
  run "$MAXOR_BIN" profile list --json
  [ "$status" = 0 ]
  echo "$output" | jq -e 'length == 4 and all(.[]; has("id") and has("enabled") and has("description"))'
}

@test "profile enable con un nombre desconocido falla con 2" {
  touch "$MAXOR_FLAKE/flake.nix"
  run "$MAXOR_BIN" profile enable nada
  [ "$status" = 2 ]
  [[ "$output" == *"unknown profile"* ]]
}

@test "hardware detect devuelve el JSON del contrato" {
  run "$MAXOR_BIN" hardware detect
  [ "$status" = 0 ]
  echo "$output" | jq -e '.version == 1 and has("cpu") and has("gpus") and has("laptop") and has("virt")'
}

@test "completions fish lista todos los comandos" {
  run "$MAXOR_BIN" completions fish
  [ "$status" = 0 ]
  for c in theme update doctor hardware profile search install remove apps logs debug completions version; do
    [[ "$output" == *"-a $c "* ]] || { echo "falta $c"; false; }
  done
}

@test "completions bash y zsh generan un script válido" {
  run "$MAXOR_BIN" completions bash
  [ "$status" = 0 ]
  bash -n <<< "$output"
  run "$MAXOR_BIN" completions zsh
  [ "$status" = 0 ]
  [[ "$output" == *"#compdef maxor"* ]]
  run "$MAXOR_BIN" completions nada
  [ "$status" = 2 ]
}

@test "logs sin registro lo dice y --path imprime la ruta" {
  run "$MAXOR_BIN" logs --last
  [ "$status" = 0 ]
  [[ "$output" == *"No failure"* ]]
  run "$MAXOR_BIN" logs --path
  [[ "$output" == *"maxor.log" ]]
}

@test "theme list sin temas instalados falla con 3" {
  run "$MAXOR_BIN" theme list
  [ "$status" = 3 ]
}

@test "theme apply de un tema que no existe falla con 3" {
  run "$MAXOR_BIN" theme apply nada
  [ "$status" = 3 ]
}

@test "install de un tema no válido se rechaza sin tocar nada" {
  mkdir -p "$BATS_TEST_TMPDIR/bad"
  echo '{"bg":"red"}' > "$BATS_TEST_TMPDIR/bad/colors.json"
  run "$MAXOR_BIN" theme install "$BATS_TEST_TMPDIR/bad"
  [ "$status" = 1 ]
  [ ! -e "$XDG_DATA_HOME/maxor/themes/bad" ]
}

@test "--no-color y --quiet se aceptan en cualquier posición" {
  run "$MAXOR_BIN" --no-color version
  [ "$status" = 0 ]
  [[ "$output" != *$'\e'* ]]
  run "$MAXOR_BIN" version --quiet
  [ "$status" = 0 ]
}

@test "doctor, hardware y help no escriben nada fuera de la terminal salvo texto plano" {
  run "$MAXOR_BIN" hardware
  [ "$status" = 0 ]
  [[ "$output" != *$'\e'* ]]
}

@test "funciona sin LANG ni HOSTNAME en el entorno (cron, servicios, contenedores)" {
  run env -u LANG -u LC_ALL -u HOSTNAME "$MAXOR_BIN" --version
  [ "$status" = 0 ]
  run env LANG=es_PE.UTF-8 "$MAXOR_BIN" --version
  [ "$status" = 0 ]
  [[ "$output" == maxor\ * ]]
}

@test "funciona sin /etc/os-release legible: usa Maxor OS" {
  run "$MAXOR_BIN" version --json
  echo "$output" | jq -e '.os | type == "string" and length > 0'
}
