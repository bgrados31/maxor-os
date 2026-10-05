#!/usr/bin/env bash
# Ayudante para traducir la CLI `maxor` (la TUI se sincroniza con `go test ./internal/i18n -update`).
#
#   scripts/i18n.sh new <código>        crea home/maxor/lib/lang/<código>.sh con todos los mensajes del inglés
#                                       comentados: descomenta cada uno y tradúcelo
#   scripts/i18n.sh missing <código>    lista las claves que ese idioma aún no traduce
#   scripts/i18n.sh status              cuántas claves traduce cada idioma
#
# Detalles y reglas de estilo: docs/TRANSLATING.md.
set -euo pipefail
cd "$(dirname "$0")/.."
dir=home/maxor/lib/lang

keys() { grep -ohE '^[[:space:]]*MSG\[[a-z0-9_.]+\]=' "$1" | sed -E 's/^[[:space:]]*MSG\[//; s/\]=$//' | sort -u; }

case "${1:-}" in
  new)
    code="${2:?uso: scripts/i18n.sh new <código>  (es, pt, pt_BR, fr…)}"
    [[ "$code" =~ ^[a-z]{2,3}(_[A-Z]{2})?$ ]] || { echo "el código debe ser como es, pt o pt_BR" >&2; exit 2; }
    out="$dir/$code.sh"
    [ ! -e "$out" ] || { echo "$out ya existe" >&2; exit 1; }
    {
      echo "# ── Message catalog: $code ───────────────────────────────────────────"
      echo "# Same keys as lang/en.sh (printf formats: keep every %s, in the same order)."
      echo "# Uncomment and translate what you want: a key left out shows in English. Guide: docs/TRANSLATING.md."
      echo "msgs_${code}() {"
      echo "  :"
      sed -n '/^msgs_en() {/,/^}/p' "$dir/en.sh" | sed '1d;$d' | grep -E '^\s*(MSG\[|# ──)' | sed -E 's/^(\s*)MSG/\1# MSG/; s/^\s*# ──/\n  # ──/'
      echo "}"
    } > "$out"
    echo "creado $out"
    ;;
  missing)
    code="${2:?uso: scripts/i18n.sh missing <código>}"
    comm -23 <(keys "$dir/en.sh") <(keys "$dir/$code.sh")
    ;;
  status)
    total="$(keys "$dir/en.sh" | wc -l)"
    for f in "$dir"/*.sh; do
      c="$(basename "$f" .sh)"
      [ "$c" = en ] && continue
      printf '%-8s %3d / %d\n' "$c" "$(keys "$f" | wc -l)" "$total"
    done
    ;;
  *)
    sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'
    exit 2
    ;;
esac
