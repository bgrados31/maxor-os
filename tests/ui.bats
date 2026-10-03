load helper

@test "ui_len cuenta solo lo visible" {
  load_lib
  ui_len $'\e[38;2;255;134;184mabc\e[0m'
  [ "$UI_LEN" = 3 ]
  ui_len $'\e[1mñandú\e[22m'
  [ "$UI_LEN" = 5 ]
}

@test "ui_repv repite un carácter y admite cero" {
  load_lib
  ui_repv r 4 '─'; [ "$r" = '────' ]
  ui_repv r 0 'x'; [ -z "$r" ]
  ui_repv r -3 'x'; [ -z "$r" ]
}

@test "ui_truncv recorta con puntos suspensivos y respeta lo corto" {
  load_lib
  ui_truncv t "abcdefghij" 6; [ "$t" = "abcde…" ]
  ui_truncv t "abc" 6; [ "$t" = "abc" ]
}

@test "todas las líneas de una ventana miden lo mismo" {
  load_lib color
  out="$({ ui_open "title"; ui_line ""; ui_row ok "row ok"; ui_row bad "a much longer row of text that must be cut to fit the width of the window"; ui_split " left" "right "; ui_section "section"; ui_kv key value; ui_close; } | strip_ansi)"
  [ -n "$out" ]
  while IFS= read -r line; do
    [ "${#line}" = "$ui_w" ] || { echo "ancho ${#line} ≠ $ui_w: '$line'"; false; }
  done <<< "$out"
}

@test "las líneas con colores no desalinean el borde" {
  load_lib color
  out="$(ui_line " ${E_AC}●${E_FG} texto ${E_OK}✓${E_FG}" | strip_ansi)"
  [ "${#out}" = "$ui_w" ]
}

@test "sin color la salida es texto plano, sin bordes" {
  load_lib
  run ui_row ok "plain"
  [ "$status" = 0 ]
  [[ "$output" != *$'\e'* ]]
  [[ "$output" == *"plain"* ]]
}

@test "--no-color apaga los colores" {
  load_lib color
  ui_disable
  run ui_say ok "x"
  [[ "$output" != *$'\e'* ]]
}

@test "ui_count_lines cuenta las líneas de un marco" {
  load_lib
  ui_count_lines $'a\nb\nc'; [ "$UI_LINES" = 3 ]
  ui_count_lines "solo"; [ "$UI_LINES" = 1 ]
}

@test "ui_paint sube las líneas previas y borra el resto de cada una" {
  load_lib
  out="$(ui_paint $'uno\ndos' 2)"
  [[ "$out" == *$'\e[2A'* ]]
  [[ "$out" == *$'uno\e[K'* ]]
}

@test "ui_bar llena la proporción pedida" {
  load_lib
  ui_bar b 50 10
  plain="$(printf '%s' "$b" | strip_ansi)"
  [ "$plain" = '▰▰▰▰▰▱▱▱▱▱' ]
}
