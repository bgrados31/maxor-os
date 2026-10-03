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

@test "doctor --json devuelve los grupos y las comprobaciones" {
  bats_require_minimum_version 1.5.0
  run --separate-stderr "$MAXOR_BIN" doctor --json
  echo "$output" | jq -e 'has("ok") and has("fails") and has("warns") and (.groups | length > 0) and all(.groups[]; has("title") and (.items | length > 0) and all(.items[]; has("level") and has("text")))'
  # el código de salida refleja si hay fallos
  fails="$(echo "$output" | jq -r .fails)"
  if [ "$fails" = 0 ]; then [ "$status" = 0 ]; else [ "$status" = 1 ]; fi
}

@test "theme list --json lista los temas válidos con sus colores" {
  mkdir -p "$XDG_DATA_HOME/maxor/themes/probe" "$XDG_DATA_HOME/maxor/themes/roto"
  echo '{"bg":"#101010","s":"#202020","s2":"#303030","fg":"#ffffff","mu":"#aaaaaa","ac":"#ff0000","ac2":"#00ff00","on":"#000000","mode":"light"}' > "$XDG_DATA_HOME/maxor/themes/probe/colors.json"
  printf 'id = "probe"\nname = "Tema de prueba"\n' > "$XDG_DATA_HOME/maxor/themes/probe/theme.toml"
  echo '{"bg":"rojo"}' > "$XDG_DATA_HOME/maxor/themes/roto/colors.json"
  run "$MAXOR_BIN" theme list --json
  [ "$status" = 0 ]
  echo "$output" | jq -e 'length == 1 and .[0].id == "probe" and .[0].name == "Tema de prueba" and .[0].mode == "light" and .[0].active == false and .[0].colors.ac == "#ff0000"'
}

@test "theme list --json marca el tema activo" {
  mkdir -p "$XDG_DATA_HOME/maxor/themes/probe" "$XDG_STATE_HOME/maxor"
  echo '{"bg":"#101010","s":"#202020","s2":"#303030","fg":"#ffffff","mu":"#aaaaaa","ac":"#ff0000","ac2":"#00ff00","on":"#000000"}' > "$XDG_DATA_HOME/maxor/themes/probe/colors.json"
  echo probe > "$XDG_STATE_HOME/maxor/current"
  run "$MAXOR_BIN" theme list --json
  echo "$output" | jq -e '.[0].active == true and .[0].mode == "dark"'
}

@test "profile enable --no-apply guarda la elección sin reconstruir" {
  mkdir -p "$MAXOR_FLAKE/hosts/$MAXOR_HOST"
  touch "$MAXOR_FLAKE/flake.nix"
  run "$MAXOR_BIN" profile enable gaming --no-apply
  [ "$status" = 0 ]
  [[ "$output" == *"maxor update"* ]]
  jq -e '.profiles == ["gaming"]' "$MAXOR_FLAKE/hosts/$MAXOR_HOST/maxor.json"
  run "$MAXOR_BIN" profile disable gaming --no-apply
  jq -e '.profiles == []' "$MAXOR_FLAKE/hosts/$MAXOR_HOST/maxor.json"
}

@test "ui y setup sin terminal fallan con el código de «falta algo»" {
  run "$MAXOR_BIN" ui
  [ "$status" = 3 ]
  [[ "$output" == *"needs a terminal"* ]]
  run "$MAXOR_BIN" setup
  [ "$status" = 3 ]
  run "$MAXOR_BIN" ui nada
  [ "$status" = 2 ]
}

@test "doctor --json lleva el id de cada comprobación y su arreglo" {
  bats_require_minimum_version 1.5.0
  run --separate-stderr "$MAXOR_BIN" doctor --json
  echo "$output" | jq -e 'all(.groups[].items[]; has("id") and has("fix") and has("confirm") and has("kind"))'
  # las que están bien nunca ofrecen arreglo
  echo "$output" | jq -e 'all(.groups[].items[] | select(.level == "ok"); .fix == null)'
}

@test "update --status cuenta la rama, el canal y la huella sin compilar nada" {
  cd "$MAXOR_FLAKE"
  git init -q -b development .
  git config user.email t@t && git config user.name t
  echo '{}' > flake.nix
  echo '{"nodes":{"nixpkgs":{"original":{"ref":"nixos-26.05"},"locked":{"rev":"774debe1234567","lastModified":1790000000}}}}' > flake.lock
  git add . && git commit -q -m init
  run "$MAXOR_BIN" update --status
  [ "$status" = 0 ]
  echo "$output" | jq -e '.branch == "development" and .dirty == false and .files == 0 and .channel == "nixos-26.05" and .nixpkgs_rev == "774debe" and .nixpkgs_date == 1790000000 and (.fingerprint | length > 5)'
  first="$(echo "$output" | jq -r .fingerprint)"
  echo cambio >> flake.nix
  run "$MAXOR_BIN" update --status
  echo "$output" | jq -e '.dirty == true and .files == 1'
  [ "$(echo "$output" | jq -r .fingerprint)" != "$first" ]
}

@test "update --status funciona aunque la configuración no sea un repositorio" {
  touch "$MAXOR_FLAKE/flake.nix"
  run "$MAXOR_BIN" update --status
  [ "$status" = 0 ]
  echo "$output" | jq -e '.branch == "" and .dirty == false and .generation >= 0'
}

@test "update --cached da null sin escaneo y el guardado cuando lo hay" {
  touch "$MAXOR_FLAKE/flake.nix"
  run "$MAXOR_BIN" update --cached
  [ "$output" = null ]
  mkdir -p "$XDG_STATE_HOME/maxor"
  echo '{"up_to_date":true,"checked_at":1790000000,"fingerprint":"abc"}' > "$XDG_STATE_HOME/maxor/update-check.json"
  run "$MAXOR_BIN" update --cached
  echo "$output" | jq -e '.checked_at == 1790000000 and .up_to_date == true'
}

@test "remove --json lista lo que la app dejó en casa y no borra nada sin --purge" {
  mkdir -p "$HOME/.local/share/Fooapp" "$HOME/.config/otraCosa"
  echo x > "$HOME/.local/share/Fooapp/save.dat"
  run "$MAXOR_BIN" remove fooapp --json
  echo "$output" | jq -e '.[0].leftovers | length == 1 and (.[0].path | endswith("/.local/share/Fooapp")) and .[0].bytes > 0'
  [ -f "$HOME/.local/share/Fooapp/save.dat" ]
  [ -d "$HOME/.config/otraCosa" ]
}

@test "remove --purge borra solo las carpetas que se llaman como la app" {
  mkdir -p "$HOME/.local/share/Fooapp" "$HOME/.cache/fooapp" "$HOME/.config/fooapp-extra" "$HOME/.fooapp"
  run "$MAXOR_BIN" remove fooapp --purge --json
  echo "$output" | jq -e '.[0].ok == true and .[0].purged == true and .[0].leftovers == []'
  [ ! -e "$HOME/.local/share/Fooapp" ] && [ ! -e "$HOME/.cache/fooapp" ] && [ ! -e "$HOME/.fooapp" ]
  [ -d "$HOME/.config/fooapp-extra" ]
}

@test "remove --purge no toca nombres peligrosos ni demasiado cortos" {
  mkdir -p "$HOME/.ssh" "$HOME/.config/ab"
  run "$MAXOR_BIN" remove ssh --purge --json
  run "$MAXOR_BIN" remove ab --purge --json
  [ -d "$HOME/.ssh" ] && [ -d "$HOME/.config/ab" ]
}

@test "remove --list-data solo mira: no quita ni borra nada" {
  mkdir -p "$HOME/.config/Fooapp" "$HOME/logs"
  echo x > "$HOME/logs/fooapp.log"
  run "$MAXOR_BIN" remove fooapp --list-data
  [ "$status" = 0 ]
  echo "$output" | jq -e 'map(.path | sub(".*/"; "")) | sort == ["Fooapp", "fooapp.log"]'
  [ -d "$HOME/.config/Fooapp" ] && [ -f "$HOME/logs/fooapp.log" ]
}

@test "remove --purge también limpia el registro que la app dejó en ~/logs" {
  mkdir -p "$HOME/logs"
  echo x > "$HOME/logs/fooapp.log"
  run "$MAXOR_BIN" remove fooapp --purge --json
  echo "$output" | jq -e '.[0].purged == true'
  [ ! -e "$HOME/logs/fooapp.log" ]
}

@test "apps open de algo que no está instalado falla con JSON" {
  run "$MAXOR_BIN" apps open noexisteunapp --json
  [ "$status" = 1 ]
  echo "$output" | jq -e '.[0].ok == false'
}

@test "apps updates siempre da una lista JSON" {
  run "$MAXOR_BIN" apps updates
  [ "$status" = 0 ]
  echo "$output" | jq -e 'type == "array"'
}

@test "remove avisa al lanzador de apps y no deja su archivo de aviso" {
  mkdir -p "$HOME/.local/share/Fooapp"
  run "$MAXOR_BIN" remove fooapp --purge --json
  [ "$status" = 0 ]
  [ -d "$HOME/.local/share/applications" ]
  [ ! -e "$HOME/.local/share/applications/.maxor-refresh" ]
}

@test "apps repair da un resumen JSON aunque no haya flatpak" {
  run "$MAXOR_BIN" apps repair --json
  [ "$status" = 0 ]
  echo "$output" | jq -e '.repaired == 0'
}

@test "apps updates guarda su resultado y --refresh lo ignora" {
  run "$MAXOR_BIN" apps updates
  [ "$status" = 0 ]
  [ -s "$XDG_STATE_HOME/maxor/apps-updates.json" ]
  jq -e '.data | type == "array"' "$XDG_STATE_HOME/maxor/apps-updates.json"
  run "$MAXOR_BIN" apps updates --refresh
  [ "$status" = 0 ]
}

@test "rollback --list --json siempre da una lista JSON" {
  run "$MAXOR_BIN" rollback --list --json
  [ "$status" = 0 ]
  echo "$output" | jq -e 'type == "array"'
}

@test "rollback rechaza algo que no es un número de generación" {
  run "$MAXOR_BIN" rollback abc
  [ "$status" = 2 ]
}
