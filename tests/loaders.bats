load helper

step_one() { echo "one"; }
step_two() { echo "two"; }
step_fail() { echo "boom" >&2; return 3; }

@test "ui_pipeline ejecuta los pasos en orden y guarda lo que imprime cada uno" {
  load_lib
  ui_pipeline "Title" "Step A" step_one "Step B" step_two > /dev/null
  [ "${UI_OUTS[0]}" = one ]
  [ "${UI_OUTS[1]}" = two ]
}

@test "ui_pipeline se detiene en el primer fallo y devuelve su código" {
  load_lib
  run ui_pipeline "Title" "Step A" step_one "Step B" step_fail "Step C" step_two
  [ "$status" = 3 ]
  [[ "$output" == *"Step A"* ]]
  [[ "$output" == *"Step B"* ]]
  [[ "$output" == *"boom"* ]]
  [[ "$output" != *"Step C"* ]]
}

@test "un fallo deja el detalle en el registro" {
  load_lib
  ui_pipeline "Title" "Step B" step_fail > /dev/null 2>&1 || true
  [ -s "$logdir/last-error.log" ]
  grep -q boom "$logdir/last-error.log"
  grep -q "exit 3" "$logdir/last-error.log"
}

@test "ui_run devuelve la salida en UI_OUT y marca el éxito" {
  load_lib
  run ui_run "Doing it" step_one
  [ "$status" = 0 ]
  [[ "$output" == *"Doing it"* ]]
  ui_run "Doing it" step_two > /dev/null
  [ "$UI_OUT" = two ]
}

@test "ui_run con fallo conserva el código y avisa dónde mirar" {
  load_lib
  run ui_run "Doing it" step_fail
  [ "$status" = 3 ]
  [[ "$output" == *"boom"* ]]
  [[ "$output" == *"maxor logs --last"* ]]
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

@test "ui_run_tail se comporta como ui_run fuera de una terminal" {
  load_lib
  run ui_run_tail "Building" step_one
  [ "$status" = 0 ]
  [[ "$output" == *"Building"* ]]
}

@test "ui_progress no dibuja fuera de una terminal y termina con una línea" {
  load_lib
  ui_progress_begin 10 "Copying files" > /dev/null
  ui_progress_set 5 "half" > /dev/null
  run ui_progress_end
  [[ "$output" == *"Copying files"* ]]
}

@test "ui_skeleton_frame dibuja el número de filas pedido dentro de una ventana" {
  load_lib color
  out="$(ui_skeleton_frame "maxor · apps" 3 0 | strip_ansi)"
  [ "$(grep -c '░' <<< "$out")" = 3 ]
  [[ "$out" == *"╭"* && "$out" == *"╰"* ]]
}
