#!/usr/bin/env bash
# Imprime las notas de una versión: la sección «## [X.Y.Z]» de CHANGELOG.md.
# Uso: scripts/release-notes.sh 0.1.0
set -euo pipefail

ver="${1:?uso: scripts/release-notes.sh X.Y.Z}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

notes="$(awk -v v="$ver" '
  $0 ~ "^## \\[" v "\\]" { on = 1; next }
  on && /^## / { exit }
  on { print }
' "$root/CHANGELOG.md")"

# Quita líneas en blanco al principio y al final.
notes="$(printf '%s\n' "$notes" | sed -e '/./,$!d' | sed -e ':a' -e '/^\n*$/{$d;N;ba' -e '}')"
[ -n "$notes" ] || { echo "CHANGELOG.md no tiene una sección [$ver] con contenido" >&2; exit 1; }
printf '%s\n' "$notes"
