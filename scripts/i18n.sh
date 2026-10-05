#!/usr/bin/env bash
# Mantenimiento de las traducciones de Maxor OS. Hay dos catálogos gettext (.po), uno por programa:
#
#   tui/internal/i18n/lang/   la app de pantalla completa y el instalador (Go)
#   home/maxor/po/            la línea de comandos `maxor` (bash)
#
#   scripts/i18n.sh pot [archivo]   regenera la plantilla de la CLI (maxor-cli.pot) desde lib/lang/en.sh
#   scripts/i18n.sh update          plantillas al día y fusionadas en cada idioma de los dos programas
#   scripts/i18n.sh new <código>    crea el idioma (es, pt, pt_BR, ru…) en los dos programas
#   scripts/i18n.sh status          qué tan traducido está cada idioma
#   scripts/i18n.sh check           lo que mira la CI: plantillas al día y todos los .po bien formados
#
# Necesita gettext (msgmerge, msginit, msgfmt): nix shell nixpkgs#gettext. `update` y `new` también
# usan Go para la parte de la app (nix shell nixpkgs#go). Guía para quien traduce: docs/TRANSLATING.md.
set -euo pipefail
cd "$(dirname "$0")/.."

cli_dir=home/maxor/po
cli_pot=$cli_dir/maxor-cli.pot
tui_dir=tui/internal/i18n/lang
en=home/maxor/lib/lang/en.sh

need() { command -v "$1" > /dev/null || { echo "falta $1: nix shell nixpkgs#gettext" >&2; exit 1; }; }

# po_str palabra texto → «palabra "texto"», partido tras cada salto de línea como hace gettext
po_str() {
  local kw="$1" s="$2" line
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\t'/\\t}"
  if [[ "$s" != *$'\n'* ]]; then
    printf '%s "%s"\n' "$kw" "$s"
    return
  fi
  printf '%s ""\n' "$kw"
  while [ -n "$s" ]; do
    if [[ "$s" == *$'\n'* ]]; then
      line="${s%%$'\n'*}"
      printf '"%s\\n"\n' "$line"
      s="${s#*$'\n'}"
    else
      printf '"%s"\n' "$s"
      s=""
    fi
  done
}

make_pot() {
  declare -gA MSG=()
  # shellcheck source=/dev/null
  source "$en"
  msgs_en
  local k
  cat << 'EOF'
# Maxor OS: messages of the `maxor` command line.
# How to translate: docs/TRANSLATING.md
msgid ""
msgstr ""
"Project-Id-Version: Maxor OS\n"
"Report-Msgid-Bugs-To: https://github.com/bgrados31/maxor-os/issues\n"
"MIME-Version: 1.0\n"
"Content-Type: text/plain; charset=UTF-8\n"
"Content-Transfer-Encoding: 8bit\n"
"Plural-Forms: nplurals=INTEGER; plural=EXPRESSION;\n"
EOF
  # en el orden del catálogo inglés, que sigue el de las secciones del programa
  while IFS= read -r k; do
    echo
    echo "#. $k"
    po_str msgctxt "$k"
    po_str msgid "${MSG[$k]}"
    if [ -n "${MSG[$k#1]+x}" ]; then
      po_str msgid_plural "${MSG[$k#1]}"
      echo 'msgstr[0] ""'
      echo 'msgstr[1] ""'
    else
      echo 'msgstr ""'
    fi
  done < <(grep -ohE '^[[:space:]]*MSG\[[a-z0-9_.]+\]=' "$en" | sed -E 's/^[[:space:]]*MSG\[//; s/\]=$//')
}

cmd_pot() { # cmd_pot [archivo]
  # las formas «clave#1» no encajan en la expresión de claves: van dentro de la entrada de «clave»
  make_pot > "${1:-$cli_pot}"
  echo "escrito ${1:-$cli_pot}"
}

each_cli_po() { local f; for f in "$cli_dir"/*.po; do [ -e "$f" ] && echo "$f"; done; }

cmd_update() {
  need msgmerge
  cmd_pot
  local f
  for f in $(each_cli_po); do
    msgmerge --quiet --update --backup=none --no-wrap "$f" "$cli_pot"
    echo "al día $f"
  done
  if command -v go > /dev/null; then
    (cd tui && CGO_ENABLED=0 go test ./internal/i18n -update > /dev/null) && echo "al día $tui_dir"
  else
    echo "sin go: para la app ejecuta (cd tui && go test ./internal/i18n -update)" >&2
  fi
}

cmd_new() {
  local code="${1:?uso: scripts/i18n.sh new <código>  (es, pt, pt_BR, ru…)}"
  [[ "$code" =~ ^[a-z]{2,3}(_[A-Z]{2})?$ ]] || { echo "el código debe ser como es, pt o pt_BR" >&2; exit 2; }
  need msginit
  [ ! -e "$cli_dir/$code.po" ] || { echo "$cli_dir/$code.po ya existe" >&2; exit 1; }
  [ -e "$cli_pot" ] || cmd_pot
  msginit --no-translator --no-wrap --locale="$code" --input="$cli_pot" --output-file="$cli_dir/$code.po" > /dev/null 2>&1
  # el mismo idioma para la app: el archivo con su cabecera, y go test lo llena
  local plural
  plural="$(sed -n 's/^"Plural-Forms: \(.*\)\\n"$/\1/p' "$cli_dir/$code.po")"
  printf 'msgid ""\nmsgstr ""\n"Language: %s\\n"\n"Content-Type: text/plain; charset=UTF-8\\n"\n"Plural-Forms: %s\\n"\n' "$code" "$plural" > "$tui_dir/$code.po"
  echo "creados $cli_dir/$code.po y $tui_dir/$code.po"
  if command -v go > /dev/null; then
    (cd tui && CGO_ENABLED=0 go test ./internal/i18n -update > /dev/null) && echo "app al día"
  else
    echo "falta llenar la app: (cd tui && go test ./internal/i18n -update)" >&2
  fi
}

cmd_status() {
  need msgfmt
  local f code
  printf '%-8s %-34s %s\n' idioma "línea de comandos" app
  for f in $(each_cli_po); do
    code="$(basename "$f" .po)"
    printf '%-8s %-34s %s\n' "$code" \
      "$(msgfmt --statistics -o /dev/null "$f" 2>&1 | tail -1)" \
      "$(msgfmt --statistics -o /dev/null "$tui_dir/$code.po" 2>&1 | tail -1)"
  done
}

cmd_check() {
  need msgfmt
  local tmp f bad=0
  tmp="$(mktemp)"
  cmd_pot "$tmp" > /dev/null
  if ! diff -q "$tmp" "$cli_pot" > /dev/null; then
    echo "$cli_pot no está al día con $en: ejecuta scripts/i18n.sh pot" >&2
    bad=1
  fi
  rm -f "$tmp"
  for f in $(each_cli_po) "$tui_dir"/*.po; do
    msgfmt --check-format --check-header -o /dev/null "$f" 2> /dev/null || { msgfmt --check-format -o /dev/null "$f" || bad=1; }
  done
  [ "$bad" = 0 ] && echo "bien"
  return "$bad"
}

case "${1:-}" in
  pot) shift; cmd_pot "$@" ;;
  update) shift; cmd_update ;;
  new) shift; cmd_new "$@" ;;
  status) cmd_status ;;
  check) cmd_check ;;
  *) sed -n '2,16p' "$0" | sed 's/^# \{0,1\}//'; exit 2 ;;
esac
