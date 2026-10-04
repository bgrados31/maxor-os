load helper

step_one() { echo "one"; }
step_two() { echo "two"; }
step_fail() { echo "boom" >&2; return 3; }

@test "ui_run devuelve la salida en UI_OUT y marca el éxito con ◇" {
  load_lib
  run ui_run "Doing it" step_one
  [ "$status" = 0 ]
  [[ "$output" == *"◇  Doing it"* ]]
  ui_run "Doing it" step_two > /dev/null
  [ "$UI_OUT" = two ]
}

@test "ui_run con fallo conserva el código, muestra la causa y dónde mirar" {
  load_lib
  run ui_run "Doing it" step_fail
  [ "$status" = 3 ]
  [[ "$output" == *"✗  Doing it"* ]]
  [[ "$output" == *"boom"* ]]
  [[ "$output" == *"maxor logs --last"* ]]
}

@test "un fallo deja el detalle en el registro" {
  load_lib
  ui_run "Doing it" step_fail > /dev/null 2>&1 || true
  [ -s "$logdir/last-error.log" ]
  grep -q boom "$logdir/last-error.log"
  grep -q "exit 3" "$logdir/last-error.log"
}

@test "dentro del riel cada paso va precedido de una línea de riel" {
  load_lib
  out="$({ ui_intro "t"; ui_run "One" step_one; ui_run "Two" step_two; ui_outro "end"; } | strip_ansi)"
  expected=$'┌  t\n│\n◇  One\n│\n◇  Two\n│\n└  end'
  [ "$out" = "$expected" ] || { echo "$out"; false; }
}

@test "--quiet calla los éxitos pero no los errores" {
  load_lib
  MAXOR_QUIET=1
  run ui_run "Doing it" step_one
  [ -z "$output" ]
  run ui_run "Doing it" step_fail
  [ "$status" = 3 ]
  [[ "$output" == *"boom"* ]]
}

@test "los pasos en secuencia se hacen con llamadas seguidas y se detienen al fallar" {
  load_lib
  seq() { ui_run "A" step_one && ui_run "B" step_fail && ui_run "C" step_two; }
  run seq
  [ "$status" = 3 ]
  [[ "$output" == *"A"* && "$output" == *"B"* ]]
  [[ "$output" != *"C"* ]]
}

@test "ui_progress no dibuja fuera de una terminal y termina con una línea" {
  load_lib
  ui_progress_begin 10 "Copying files" > /dev/null
  ui_progress_set 5 "half" > /dev/null
  run ui_progress_end
  [[ "$output" == *"Copying files"* ]]
}

@test "el cargador solo existe una vez: no quedan variantes" {
  load_lib
  ! declare -F ui_run_tail ui_pipeline ui_skeleton_frame ui_flat_split > /dev/null
}
