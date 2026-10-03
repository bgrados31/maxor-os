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
