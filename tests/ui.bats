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

@test "el riel: intro, paso, bloque, fila y cierre" {
  load_lib
  out="$({ ui_intro "title"; ui_step ok "a done thing"; ui_section "Block"; ui_row ok "a row"; ui_outro "bye"; } | strip_ansi)"
  expected=$'┌  title\n│\n◇  a done thing\n│\n◇  Block\n│  ✓ a row\n│\n└  bye'
  [ "$out" = "$expected" ] || { echo "$out"; false; }
}

@test "la línea vacía del riel nunca se duplica" {
  load_lib
  out="$({ ui_intro "t"; ui_text ""; ui_section "S"; ui_text ""; ui_text ""; ui_outro; } | strip_ansi)"
  dup="$(awk 'prev=="│" && $0=="│"{print "dup"} {prev=$0}' <<< "$out")"
  [ -z "$dup" ] || { echo "$out"; false; }
}

@test "ui_outro sin mensaje no deja espacios al final" {
  load_lib
  out="$({ ui_intro "t"; ui_outro; } | strip_ansi)"
  [[ "$out" == *$'\n└' ]]
}

@test "con color y sin color es el mismo texto" {
  load_lib color
  con="$({ ui_intro "t"; ui_step ok "x"; ui_row warn "y"; ui_outro "z"; } | strip_ansi)"
  load_lib
  sin="$({ ui_intro "t"; ui_step ok "x"; ui_row warn "y"; ui_outro "z"; } | strip_ansi)"
  [ "$con" = "$sin" ]
}

@test "sin color no hay secuencias de escape" {
  load_lib
  out="$({ ui_intro "t"; ui_step ok "x"; ui_row bad "y"; ui_outro "z"; })"
  [[ "$out" != *$'\e'* ]]
}

@test "respaldo ASCII: solo caracteres ASCII, con la misma estructura" {
  export MAXOR_ASCII=1
  load_lib
  out="$({ ui_intro "title"; ui_step ok "done"; ui_section "Block"; ui_row ok "row"; ui_outro "bye"; })"
  ! LC_ALL=C grep -qP '[^\x00-\x7F]' <<< "$out" || { echo "$out"; false; }
  [[ "$out" == "+  title"* ]]
  [[ "$out" == *"o  done"* ]]
  [[ "$out" == *"|  v row"* ]]
}

@test "un locale que no es UTF-8 activa el respaldo ASCII" {
  export LANG=C
  load_lib
  [ "$ui_ascii" = 1 ]
  export LANG=es_PE.UTF-8
  load_lib
  [ "$ui_ascii" = 0 ]
}

@test "una fila muy larga se recorta al ancho de la terminal" {
  load_lib color
  long="$(printf 'x%.0s' $(seq 1 200))"
  out="$(ui_row ok "$long" | strip_ansi)"
  [ "${#out}" -le "$ui_w" ]
  [[ "$out" == *"…" ]]
}

@test "ui_split deja el texto de la derecha pegado al borde" {
  load_lib color
  out="$(ui_split " left" "right " | strip_ansi)"
  [[ "$out" == *"right " ]]
  [ "${#out}" -le "$ui_w" ]
}

@test "un error dentro del riel lo cierra" {
  load_lib
  out="$( (ui_intro "t"; die "boom") 2>&1 | strip_ansi || true)"
  [[ "$out" == *"└  ✗ boom"* ]]
}

@test "ui_confirm acepta y y rechaza el resto" {
  load_lib
  echo y | ui_confirm "Sure?" > /dev/null
  ! (echo n | ui_confirm "Sure?" > /dev/null)
  ! (echo "" | ui_confirm "Sure?" > /dev/null)
}

@test "--no-color apaga los colores" {
  load_lib color
  ui_disable
  run ui_step ok "x"
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
