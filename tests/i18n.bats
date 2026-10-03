load helper

# Las claves usadas en el código: «@clave» y «t clave».
used_keys() {
  {
    grep -rohE '@[a-z0-9_]+\.[a-z0-9_]+' "$SRC/cmd" "$SRC/lib" "$SRC/main.sh" | sed 's/^@//'
    grep -rohE '\bt (theme|style|version)\.[a-z_]+' "$SRC/cmd" "$SRC/lib" | awk '{print $2}'
  } | sort -u
}
catalog_keys() { grep -ohE 'MSG\[[a-z0-9_.]+\]' "$SRC/lib/lang/en.sh" | sed 's/MSG\[//; s/\]//' | sort -u; }

@test "toda clave usada en el código existe en el catálogo" {
  run comm -23 <(used_keys) <(catalog_keys)
  [ -z "$output" ] || { echo "faltan en lib/lang/en.sh:"; echo "$output"; false; }
}

@test "toda clave del catálogo se usa (salvo las dinámicas cmd., help. y group.)" {
  unused="$(comm -13 <(used_keys) <(catalog_keys) | grep -vE '^(cmd|help|group)\.' || true)"
  [ -z "$unused" ] || { echo "sin usar:"; echo "$unused"; false; }
}

@test "cada comando registrado tiene resumen y ayuda" {
  load_lib
  local c
  for c in "${CMD_ORDER[@]}"; do
    [ -n "${MSG[cmd.$c]:-}" ] || { echo "falta cmd.$c"; false; }
    [ -n "${MSG[help.$c]:-}" ] || { echo "falta help.$c"; false; }
  done
}

@test "cada grupo de la pantalla principal tiene título" {
  load_lib
  local g
  for g in "${CMD_GROUPS[@]}"; do [ -n "${MSG[group.$g]:-}" ] || { echo "falta group.$g"; false; }; done
}

@test "todo comando registrado pertenece a un grupo que se muestra" {
  load_lib
  local c g ok
  for c in "${CMD_ORDER[@]}"; do
    ok=0
    for g in "${CMD_GROUPS[@]}"; do [ "${CMD_GROUP[$c]}" = "$g" ] && ok=1; done
    [ "$ok" = 1 ] || { echo "$c está en el grupo '${CMD_GROUP[$c]}', que no se muestra"; false; }
  done
}

@test "todos los mensajes se formatean sin error de printf" {
  load_lib
  local k out
  for k in "${!MSG[@]}"; do msg out "@$k" uno dos tres cuatro; done
}

@test "una clave que no existe se muestra tal cual" {
  load_lib
  msg out @no.existe
  [ "$out" = "no.existe" ]
}

@test "un texto sin @ se imprime literal, aunque lleve %" {
  load_lib
  msg out "100% listo"
  [ "$out" = "100% listo" ]
}
