# ── Idiomas: catálogo de mensajes ────────────────────────────────────
# Todo texto que ve el usuario vive en un catálogo. El inglés está en lib/lang/en.sh como
# MSG[clave]='formato printf', y el código usa «@clave» donde iría el texto:
#
#   ui_row ok @doctor.kernel_ok            # sin argumentos
#   ui_say warn @apps.not_installed "$id"  # con argumentos para el formato
#   msg texto @update.step_build "$host"   # a una variable, para componer líneas
#
# Un texto que no empieza por «@» se imprime tal cual. Si falta una clave se muestra la propia clave
# (y tests/ avisa en CI).
#
# Los demás idiomas son archivos gettext estándar, po/<código>.po, que cualquier herramienta de
# traducción (Weblate, Poedit, Lokalize…) edita: el msgctxt es la clave, el msgid el inglés. Se leen
# al arrancar, solo el del idioma en uso (MAXOR_PO es la carpeta). Lo que un idioma no traduce cae al
# inglés, y un código regional (pt_BR) se carga encima del de su idioma (pt).
#
# Textos con número: si la clave tiene una forma plural en inglés (MSG[clave#1]), el PRIMER argumento
# es la cantidad y elige la forma por la regla del idioma (Plural-Forms). Los marcadores pueden
# llevar posición, %2$s, cuando un idioma necesita otro orden. Cómo traducir: docs/TRANSLATING.md.
declare -gA MSG=()
declare -gA EN=() # las formas en inglés de los textos con número, para cuando un idioma no las traduce
maxor_lang="${MAXOR_LANG:-${LANG:-en}}"
# i18n_init lo normaliza: es_PE.UTF-8 → es_PE, pt-BR → pt_BR, C.UTF-8 → C (sin catálogo: inglés)
I18N_PLURAL='n != 1' # la regla del idioma en uso; el inglés cuenta así
I18N_NPLURALS=2

# Lee un .po y deja sus claves en MSG (el msgctxt es la clave; la forma N de un plural, «clave#N») y
# su regla de plurales en I18N_PLURAL. Ignora lo marcado «fuzzy» y lo vacío. awk es de POSIX.
I18N_AWK='
function unq(s,   r, i, c, n) {
  r = ""; n = length(s)
  for (i = 1; i <= n; i++) {
    c = substr(s, i, 1)
    if (c == "\\") {
      i++; c = substr(s, i, 1)
      if (c == "n") r = r "\n"; else if (c == "t") r = r "\t"; else r = r c
    } else r = r c
  }
  return r
}
function quoted(l) { sub(/^[^"]*"/, "", l); sub(/"[ \t\r]*$/, "", l); return unq(l) }
function begin() { if (!have) { have = 1; fuzzy = pf; pf = 0 } }
function flush(   i, n, L, p, e, k) {
  if (!have) return
  if (ctx == "" && id == "") {
    n = split(str[0], L, "\n")
    for (i = 1; i <= n; i++) if (L[i] ~ /^Plural-Forms:/) {
      p = L[i]; sub(/^Plural-Forms:[ \t]*/, "", p)
      if (match(p, /nplurals=[0-9]+/)) printf "@nplurals\037%s\036", substr(p, RSTART + 9, RLENGTH - 9)
      if (match(p, /plural=[^;]*/)) { e = substr(p, RSTART + 7, RLENGTH - 7); printf "@plural\037%s\036", e }
    }
  } else if (ctx != "" && !fuzzy) {
    for (i = 0; i <= maxi; i++) if (str[i] != "") { k = (i == 0) ? ctx : ctx "#" i; printf "%s\037%s\036", k, str[i] }
  }
  have = 0; ctx = ""; id = ""; fuzzy = 0; maxi = 0; state = ""; cur = ""; split("", str)
}
/^#,/ { if ($0 ~ /fuzzy/) pf = 1; next }
/^#/ { next }
/^[ \t\r]*$/ { flush(); next }
/^msgctxt / { if (state == "str") flush(); begin(); ctx = quoted($0); cur = "ctx"; next }
/^msgid_plural / { cur = "plural"; next }
/^msgid / { if (state == "str") flush(); begin(); id = quoted($0); cur = "id"; next }
/^msgstr\[/ { state = "str"; idx = $0; sub(/^msgstr\[/, "", idx); sub(/\].*/, "", idx); idx += 0; if (idx > maxi) maxi = idx; str[idx] = quoted($0); cur = "str"; next }
/^msgstr / { state = "str"; idx = 0; str[0] = quoted($0); cur = "str"; next }
/^"/ { q = quoted($0); if (cur == "ctx") ctx = ctx q; else if (cur == "id") id = id q; else if (cur == "str") str[idx] = str[idx] q; next }
END { flush() }
'
i18n_load_po() { # i18n_load_po archivo
  local rec k v
  [ -r "$1" ] || return 0
  while IFS= read -r -d $'\036' rec; do
    k="${rec%%$'\037'*}"
    v="${rec#*$'\037'}"
    case "$k" in
      @plural)
        # la regla viene de un archivo del repositorio, pero se evalúa como aritmética: solo estos caracteres
        if [[ "$v" =~ ^[0-9n%!=\<\>\&\|\?:\(\)\ +*/-]+$ ]]; then I18N_PLURAL="$v"; fi
        ;;
      @nplurals) I18N_NPLURALS="$v" ;;
      *) MSG[$k]="$v" ;;
    esac
  done < <(awk "$I18N_AWK" "$1")
}

# i18n_form N → pone en _form el número de forma plural de N en el idioma en uso
i18n_form() {
  local n="${1:-0}"
  [[ "$n" =~ ^[0-9]+$ ]] || n=0
  _form=$(( I18N_PLURAL )) # bash evalúa el texto de la variable como expresión (ya validado: solo n, números y operadores)
  [[ "$_form" =~ ^[0-9]+$ ]] || _form=0
}

# i18n_format variable formato [argumentos…]: printf, y además %2$s (cada argumento donde se pida)
i18n_format() {
  # los nombres de aquí no pueden coincidir con la variable de quien llama (printf -v la asigna por nombre)
  local __out="$1" __fmt="$2" __rest __res __re
  shift 2
  if [[ "$__fmt" == *'$s'* && "$__fmt" =~ %[0-9]+\$s ]]; then
    local -a __args=("$@")
    __res=""
    __re='^([^%]*)%(([0-9]+)\$s|%)(.*)$'
    __rest="$__fmt"
    while [[ "$__rest" =~ $__re ]]; do
      __res+="${BASH_REMATCH[1]}"
      if [ -n "${BASH_REMATCH[3]}" ]; then __res+="${__args[$((BASH_REMATCH[3] - 1))]-}"; else __res+="%"; fi
      __rest="${BASH_REMATCH[4]}"
    done
    printf -v "$__out" '%s' "$__res$__rest"
  else
    # shellcheck disable=SC2059
    printf -v "$__out" "$__fmt" "$@"
  fi
}

msg() { # msg variable texto|@clave [argumentos…]
  # (los locales llevan __ para no chocar con la variable de destino de quien llama)
  local __v="$1" __k="$2" __f
  shift 2
  if [ "${__k:0:1}" = "@" ]; then
    __k="${__k:1}"
    __f="${MSG[$__k]-}"
    if [ -n "${MSG[$__k#1]+x}" ]; then
      # un texto con número: el primer argumento elige la forma
      i18n_form "${1:-0}"
      if [ "$_form" = 0 ]; then __f="${MSG[$__k]-}"; else __f="${MSG[$__k#$_form]-}"; fi
      if [ -z "$__f" ]; then
        if [ "${1:-0}" = 1 ]; then __f="${EN[$__k]-${MSG[$__k]-}}"; else __f="${EN[$__k#1]-${MSG[$__k#1]-}}"; fi
      fi
    fi
    [ -n "$__f" ] || __f="$__k"
    i18n_format "$__v" "$__f" "$@"
  else
    printf -v "$__v" '%s' "$__k"
  fi
}
t() { local __o; msg __o "@$1" "${@:2}"; printf '%s' "$__o"; } # t clave [argumentos…] → imprime

i18n_init() {
  local full="${maxor_lang%%.*}" base k dir="${MAXOR_PO:-}"
  full="${full%%@*}"
  full="${full//-/_}"
  base="${full%%_*}"
  maxor_lang="$base"
  I18N_PLURAL='n != 1'
  I18N_NPLURALS=2
  msgs_en
  [ "$base" != en ] || return 0
  [ -n "$dir" ] || return 0
  for k in "${!MSG[@]}"; do
    if [[ "$k" == *'#1' ]]; then EN[$k]="${MSG[$k]}"; EN[${k%'#1'}]="${MSG[${k%'#1'}]}"; fi
  done
  i18n_load_po "$dir/$base.po"
  if [ "$full" != "$base" ] && [ -r "$dir/$full.po" ]; then i18n_load_po "$dir/$full.po"; maxor_lang="$full"; fi
}
