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

@test "msg y t funcionan con cualquier nombre de variable de destino, también los internos" {
  load_lib
  maxor_lang=es; MSG=(); i18n_init
  local _o _v _f _k _r _res _out _fmt
  msg _o @err.cancelled; [ "$_o" = "Cancelado." ]
  msg _f @err.unknown_command x; [ "$_f" = "comando desconocido: x" ]
  msg _k @theme.invalid a b; [ "$_k" = "tema 'a': b" ]
  msg _out @apps.purge_confirm 3 "12 MB"; [ "$_out" = "¿Borrar también sus datos (12 MB en 3 carpetas)?" ]
  [ "$(t err.cancelled)" = "Cancelado." ]
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
# Estas pruebas valen para todo po/<código>.po, también para los que se añadan: no hay que tocarlas.
# Los idiomas «revisados» tienen que estar completos; el resto puede tener huecos (ahí sale el inglés).
REVIEWED="es"

po_codes() { local f; for f in "$SRC"/po/*.po; do f="${f##*/}"; echo "${f%.po}"; done; }
# las claves de un .po tal como las lee la CLI (claves, formas «clave#N», @plural y @nplurals), una por línea
po_records() { local rec; while IFS= read -r -d $'\036' rec; do echo "${rec%%$'\037'*}"; done < <(awk "$I18N_AWK" "$1") | sort -u; }
en_keys() { local k; for k in "${!MSG[@]}"; do echo "$k"; done | sort -u; }

# Los marcadores %s de un texto, por posición: %s sueltos numerados en orden, %2$s con su número; %% no cuenta.
ph_set() {
  local s="${1//%%/}" next=1 out=""
  while [[ "$s" =~ %(([0-9]+)\$)?s ]]; do
    if [ -n "${BASH_REMATCH[2]}" ]; then out+="${BASH_REMATCH[2]} "; else out+="$next "; next=$((next + 1)); fi
    s="${s#*"${BASH_REMATCH[0]}"}"
  done
  tr ' ' '\n' <<< "$out" | sort -n | grep . | tr '\n' ' ' || true
}
ph_plain() { local s="${1//%%/}"; [[ "$s" =~ %[0-9]*\$?s ]] && [[ ! "$s" =~ %[0-9]+\$s ]]; }
# un % que no sea %s, %N$s ni %% estropearía el texto
ph_stray() { local s="${1//%%/}"; s="$(sed -E 's/%([0-9]+\$)?s//g' <<< "$s")"; [[ "$s" == *%* ]]; }

@test "cada .po se lee, dice su idioma y trae una regla de plurales válida" {
  load_lib
  local c rec plural="" n=""
  for c in $(po_codes); do
    grep -q "^\"Language: $c" "$SRC/po/$c.po" || { echo "$c: la cabecera no dice Language: $c"; false; }
    while IFS= read -r -d $'\036' rec; do
      [ "${rec%%$'\037'*}" = @plural ] && plural="${rec#*$'\037'}"
      [ "${rec%%$'\037'*}" = @nplurals ] && n="${rec#*$'\037'}"
    done < <(awk "$I18N_AWK" "$SRC/po/$c.po")
    [ -n "$plural" ] && [ -n "$n" ] || { echo "$c: falta Plural-Forms"; false; }
    [[ "$plural" =~ ^[0-9n%!=\<\>\&\|\?:\(\)\ +*/-]+$ ]] || { echo "$c: la regla de plurales «$plural» no es válida"; false; }
    # la regla nunca da una forma que el idioma no tenga
    local i f
    for i in 0 1 2 3 4 5 10 11 12 21 22 100 101 102 111 1000; do
      f=$(( $(n=$i; echo $(( ${plural} ))) ))
      [ "$f" -lt "$n" ] || { echo "$c: $i da la forma $f y el idioma tiene $n"; false; }
    done
  done
}

@test "ningún .po trae claves que el inglés no tiene" {
  load_lib
  local c extra
  for c in $(po_codes); do
    extra="$(comm -13 <(en_keys) <(po_records "$SRC/po/$c.po" | grep -v '^@' | sed -E 's/#[0-9]+$//' | sort -u) | grep -v '^$' || true)"
    [ -z "$extra" ] || { echo "$c: claves que no existen en en.sh:"; echo "$extra"; false; }
  done
}

@test "los idiomas revisados traducen todas las claves, todas sus formas plurales" {
  load_lib
  local c k missing=""
  for c in $REVIEWED; do
    local have n=2
    have="$(po_records "$SRC/po/$c.po")"
    n="$(grep -m1 -o 'nplurals=[0-9]*' "$SRC/po/$c.po" | cut -d= -f2)"
    for k in "${!MSG[@]}"; do
      [[ "$k" == *'#'* ]] && continue
      grep -qxF "$k" <<< "$have" || missing+="$k "
      if [ -n "${MSG[$k#1]+x}" ]; then
        local i
        for ((i = 1; i < n; i++)); do grep -qxF "$k#$i" <<< "$have" || missing+="$k#$i "; done
      fi
    done
    [ -z "$missing" ] || { echo "$c: sin traducir: $missing"; false; }
  done
}

@test "la plantilla maxor-cli.pot está al día con en.sh" {
  local tmp="$BATS_TEST_TMPDIR/cli.pot"
  "$ROOT/scripts/i18n.sh" pot "$tmp" > /dev/null
  diff "$tmp" "$SRC/po/maxor-cli.pot" || { echo "ejecuta scripts/i18n.sh pot"; false; }
}

@test "cada traducción lleva los mismos marcadores que el inglés (los posicionales, en cualquier orden)" {
  load_lib
  declare -A en=()
  local k c base
  for k in "${!MSG[@]}"; do en[$k]="${MSG[$k]}"; done
  for c in $(po_codes); do
    MSG=()
    maxor_lang="$c"; i18n_init
    for k in "${!MSG[@]}"; do
      [ "${MSG[$k]}" = "${en[$k]:-}" ] && continue
      base="${k%%#*}"
      [ -n "${en[$k]+x}" ] || [ -n "${en[$base]+x}" ] || continue
      ! ph_stray "${MSG[$k]}" || { echo "$c: $k: un % que no es %s ni %%: «${MSG[$k]}»"; false; }
      local want="${en[$k]-${en[$base]}}" got
      got="$(ph_set "${MSG[$k]}")"; want="$(ph_set "$want")"
      if [ -n "${en[$base#1]+x}" ]; then
        # un texto con número: una forma puede dejar fuera el número («un servicio»), pero no añadir marcadores
        for i in $got; do [[ " $want " == *" $i "* ]] || { echo "$c: $k: marcador $i que el inglés no tiene"; false; }; done
      else
        [ "$got" = "$want" ] || { echo "$c: $k: marcadores «$got» ≠ «$want»: «${MSG[$k]}»"; false; }
      fi
      # sin posiciones, el orden es el del inglés (printf no puede cambiarlo)
    done
  done
}

@test "la ayuda traducida conserva cada comando de ejemplo tal cual" {
  load_lib
  declare -A en=()
  local k c a b
  for k in "${!MSG[@]}"; do [[ "$k" == help.* ]] && en[$k]="${MSG[$k]}"; done
  for c in $(po_codes); do
    MSG=()
    maxor_lang="$c"; i18n_init
    for k in "${!MSG[@]}"; do
      [[ "$k" == help.* ]] || continue
      a="$(grep -E '^    maxor ' <<< "${en[$k]}" | sort)"
      b="$(grep -E '^    maxor ' <<< "${MSG[$k]}" | sort)"
      [ "$a" = "$b" ] || { echo "$c: $k: los comandos de ejemplo cambiaron"; diff <(echo "$a") <(echo "$b") || true; false; }
    done
  done
}

@test "todos los mensajes de todos los idiomas se formatean sin error" {
  load_lib
  local c k out
  for c in $(po_codes); do
    MSG=()
    maxor_lang="$c"; i18n_init
    for k in "${!MSG[@]}"; do msg out "@$k" 3 dos tres cuatro; done
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
  # un regional se carga encima de su idioma
  mkdir -p "$BATS_TEST_TMPDIR/po"
  cp "$SRC/po/pt.po" "$BATS_TEST_TMPDIR/po/pt.po"
  printf 'msgid ""\nmsgstr ""\n"Language: pt_PT\\n"\n"Plural-Forms: nplurals=2; plural=(n != 1);\\n"\n\nmsgctxt "err.cancelled"\nmsgid "Cancelled."\nmsgstr "Cancelado (PT)."\n' > "$BATS_TEST_TMPDIR/po/pt_PT.po"
  MAXOR_PO="$BATS_TEST_TMPDIR/po"; maxor_lang="pt-PT"; MSG=(); i18n_init
  [ "$maxor_lang" = pt_PT ]
  msg out @err.cancelled; [ "$out" = "Cancelado (PT)." ]
  msg out @err.unknown_command_title; [ "$out" = "Comando desconhecido" ]
}

@test "lo que un idioma no traduce cae al inglés, y una entrada fuzzy no se usa" {
  load_lib
  mkdir -p "$BATS_TEST_TMPDIR/po"
  printf 'msgid ""\nmsgstr ""\n"Language: zz\\n"\n"Plural-Forms: nplurals=2; plural=(n != 1);\\n"\n\nmsgctxt "err.cancelled"\nmsgid "Cancelled."\nmsgstr "ZZ"\n\n#, fuzzy\nmsgctxt "err.unknown_command_title"\nmsgid "Unknown command"\nmsgstr "no se usa"\n' > "$BATS_TEST_TMPDIR/po/zz.po"
  MAXOR_PO="$BATS_TEST_TMPDIR/po"; maxor_lang="zz"; MSG=(); i18n_init
  msg out @err.cancelled; [ "$out" = "ZZ" ]
  msg out @err.unknown_command_title; [ "$out" = "Unknown command" ]
  msg out @err.unknown_option x; [ "$out" = "unknown option: x" ]
}

@test "los textos con número usan la regla del idioma: es cuenta 1 aparte, el ruso tiene tres formas" {
  load_lib
  maxor_lang=es; MSG=(); i18n_init
  msg out @doctor.sys_bad 1; [[ "$out" == "1 servicio del sistema"* ]]
  msg out @doctor.sys_bad 2; [[ "$out" == "2 servicios del sistema"* ]]
  maxor_lang=en; MSG=(); i18n_init
  msg out @doctor.sys_bad 1; [[ "$out" == "1 failed system service:"* ]]
  msg out @doctor.sys_bad 0; [[ "$out" == "0 failed system services:"* ]]
  # un idioma de tres formas, escrito aquí: lo que traiga un archivo de la comunidad funciona igual
  mkdir -p "$BATS_TEST_TMPDIR/po"
  printf 'msgid ""\nmsgstr ""\n"Language: ru\\n"\n"Plural-Forms: nplurals=3; plural=(n%%10==1 && n%%100!=11 ? 0 : n%%10>=2 && n%%10<=4 && (n%%100<10 || n%%100>=20) ? 1 : 2);\\n"\n\nmsgctxt "diff.config_files"\nmsgid "x"\nmsgid_plural "y"\nmsgstr[0] "%%s файл"\nmsgstr[1] "%%s файла"\nmsgstr[2] "%%s файлов"\n' > "$BATS_TEST_TMPDIR/po/ru.po"
  MAXOR_PO="$BATS_TEST_TMPDIR/po"; maxor_lang=ru; MSG=(); i18n_init
  local n want
  for n in 1:"1 файл" 2:"2 файла" 5:"5 файлов" 11:"11 файлов" 21:"21 файл" 22:"22 файла" 25:"25 файлов" 0:"0 файлов"; do
    msg out @diff.config_files "${n%%:*}"; [ "$out" = "${n#*:}" ] || { echo "${n%%:*} → «$out», esperaba «${n#*:}»"; false; }
  done
  # una forma sin traducir cae al inglés, no a la de otro número
  printf 'msgid ""\nmsgstr ""\n"Language: ru\\n"\n"Plural-Forms: nplurals=3; plural=(n%%10==1 && n%%100!=11 ? 0 : n%%10>=2 && n%%10<=4 && (n%%100<10 || n%%100>=20) ? 1 : 2);\\n"\n\nmsgctxt "diff.config_files"\nmsgid "x"\nmsgid_plural "y"\nmsgstr[0] "%%s файл"\nmsgstr[1] "%%s файла"\nmsgstr[2] ""\n' > "$BATS_TEST_TMPDIR/po/ru.po"
  MSG=(); i18n_init
  msg out @diff.config_files 5; [ "$out" = "5 configuration files changed (hidden)" ]
}

@test "un marcador con posición, %2\$s, pone cada argumento donde se pide" {
  load_lib
  mkdir -p "$BATS_TEST_TMPDIR/po"
  printf 'msgid ""\nmsgstr ""\n"Language: zz\\n"\n"Plural-Forms: nplurals=1; plural=0;\\n"\n\nmsgctxt "theme.invalid"\nmsgid "x"\nmsgstr "%%2$s ← %%1$s (100%%%% %%2$s)"\n' > "$BATS_TEST_TMPDIR/po/zz.po"
  MAXOR_PO="$BATS_TEST_TMPDIR/po"; maxor_lang=zz; MSG=(); i18n_init
  msg out @theme.invalid primero segundo
  [ "$out" = "segundo ← primero (100% segundo)" ]
  # el inglés también puede usarlos: el orden de los argumentos del código no cambia
  maxor_lang=en; MSG=(); i18n_init
  msg out @apps.purge_confirm 3 "12 MB"
  [ "$out" = "Delete its data too (12 MB in 3 folders)?" ]
}

@test "una regla de plurales con código dentro se ignora, no se ejecuta" {
  load_lib
  mkdir -p "$BATS_TEST_TMPDIR/po"
  printf 'msgid ""\nmsgstr ""\n"Language: zz\\n"\n"Plural-Forms: nplurals=2; plural=a[$(touch %s/hacked)];\\n"\n\nmsgctxt "diff.config_files"\nmsgid "x"\nmsgid_plural "y"\nmsgstr[0] "uno"\nmsgstr[1] "varios"\n' "$BATS_TEST_TMPDIR" > "$BATS_TEST_TMPDIR/po/zz.po"
  MAXOR_PO="$BATS_TEST_TMPDIR/po"; maxor_lang=zz; MSG=(); i18n_init
  msg out @diff.config_files 1; [ "$out" = "uno" ]
  msg out @diff.config_files 3; [ "$out" = "varios" ]
  [ ! -e "$BATS_TEST_TMPDIR/hacked" ]
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
