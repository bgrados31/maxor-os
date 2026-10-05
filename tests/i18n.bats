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

@test "la ayuda de los comandos principales trae ejemplos que empiezan por maxor" {
  load_lib
  local c
  for c in theme search install remove apps update rollback doctor hardware profile ui backup restore logs; do
    [[ "${MSG[help.$c]}" == *"Examples:"* ]] || { echo "help.$c sin ejemplos"; false; }
    # cada ejemplo es una línea con sangría que empieza por «maxor»
    [[ "${MSG[help.$c]}" == *$'\n    maxor'* ]] || { echo "help.$c: ejemplos mal formados"; false; }
  done
}

# ── Idiomas ──────────────────────────────────────────────────────────
# Estas pruebas valen para todo lib/lang/<código>.sh, también para los que se añadan: no hay que tocarlas.
# Los idiomas «revisados» tienen que estar completos; el resto puede tener huecos (ahí sale el inglés).
REVIEWED="es"

lang_codes() { local f; for f in "$SRC"/lib/lang/*.sh; do f="${f##*/}"; f="${f%.sh}"; [ "$f" = en ] || echo "$f"; done; }
file_keys() { grep -ohE '^[[:space:]]*MSG\[[a-z0-9_.]+\]=' "$1" | sed -E 's/^[[:space:]]*MSG\[//; s/\]=$//' | sort -u; }
specs() { grep -oE '%[-+# 0-9.*]*[a-zA-Z%]' <<< "$1" | tr '\n' ' '; }

@test "cada idioma define su función msgs_<código>" {
  load_lib
  local c
  for c in $(lang_codes); do declare -F "msgs_$c" > /dev/null || { echo "falta msgs_$c en lib/lang/$c.sh"; false; }; done
}

@test "ningún idioma trae claves que el inglés no tiene" {
  local c extra
  for c in $(lang_codes); do
    extra="$(comm -13 <(file_keys "$SRC/lib/lang/en.sh") <(file_keys "$SRC/lib/lang/$c.sh"))"
    [ -z "$extra" ] || { echo "$c: claves que no existen en en.sh:"; echo "$extra"; false; }
  done
}

@test "los idiomas revisados traducen todas las claves" {
  local c missing
  for c in $REVIEWED; do
    missing="$(comm -23 <(file_keys "$SRC/lib/lang/en.sh") <(file_keys "$SRC/lib/lang/$c.sh"))"
    [ -z "$missing" ] || { echo "$c: sin traducir:"; echo "$missing"; false; }
  done
}

@test "cada traducción lleva los mismos %s, en el mismo orden, que el inglés" {
  load_lib
  declare -A en=()
  local k c
  for k in "${!MSG[@]}"; do en[$k]="${MSG[$k]}"; done
  for c in $(lang_codes); do
    MSG=()
    "msgs_$c"
    for k in "${!MSG[@]}"; do
      [ "$(specs "${en[$k]}")" = "$(specs "${MSG[$k]}")" ] || { echo "$c: $k: «${en[$k]}» ≠ «${MSG[$k]}»"; false; }
    done
  done
}

@test "la ayuda traducida conserva cada comando de ejemplo tal cual" {
  load_lib
  declare -A en=()
  local k c a b
  for k in "${!MSG[@]}"; do [[ "$k" == help.* ]] && en[$k]="${MSG[$k]}"; done
  for c in $(lang_codes); do
    MSG=()
    "msgs_$c"
    for k in "${!MSG[@]}"; do
      [[ "$k" == help.* ]] || continue
      a="$(grep -E '^    maxor ' <<< "${en[$k]}" | sort)"
      b="$(grep -E '^    maxor ' <<< "${MSG[$k]}" | sort)"
      [ "$a" = "$b" ] || { echo "$c: $k: los comandos de ejemplo cambiaron"; diff <(echo "$a") <(echo "$b") || true; false; }
    done
  done
}

@test "todos los mensajes de todos los idiomas se formatean sin error de printf" {
  load_lib
  local c k out
  for c in $(lang_codes); do
    MSG=()
    "msgs_$c"
    for k in "${!MSG[@]}"; do msg out "@$k" uno dos tres cuatro; done
  done
}

@test "el idioma sale de MAXOR_LANG o LANG, y un regional cae al idioma base" {
  load_lib
  maxor_lang="es_PE.UTF-8"; i18n_init
  [ "$maxor_lang" = es ]
  msg out @err.cancelled
  [ "$out" = "Cancelado." ]
  maxor_lang="C.UTF-8"; MSG=(); i18n_init
  msg out @err.cancelled
  [ "$out" = "Cancelled." ]
  maxor_lang="xx_YY"; MSG=(); i18n_init
  msg out @err.cancelled
  [ "$out" = "Cancelled." ]
}

@test "lo que un idioma no traduce cae al inglés" {
  load_lib
  msgs_zz() { MSG[err.cancelled]='ZZ'; }
  maxor_lang="zz"; MSG=(); i18n_init
  msg out @err.cancelled
  [ "$out" = "ZZ" ]
  msg out @err.unknown_command_title
  [ "$out" = "Unknown command" ]
}

@test "ui_confirm acepta las palabras de sí del idioma, y y yes siempre" {
  load_lib
  maxor_lang="es"; MSG=(); i18n_init
  echo s | ui_confirm "¿Seguro?" > /dev/null
  echo sí | ui_confirm "¿Seguro?" > /dev/null
  echo y | ui_confirm "¿Seguro?" > /dev/null
  ! (echo n | ui_confirm "¿Seguro?" > /dev/null)
  ! (echo "" | ui_confirm "¿Seguro?" > /dev/null)
}

@test "los catálogos no usan apóstrofos tipográficos (shellcheck los rechaza al compilar): usa ' con $'...\\'...'" {
  ! grep -nF -e '’' -e '‘' "$SRC"/lib/lang/*.sh
}
