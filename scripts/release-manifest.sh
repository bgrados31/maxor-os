#!/usr/bin/env bash
# Imprime el manifiesto (JSON) de una release. Lo firma scripts/release.sh.
#
#   scripts/release-manifest.sh 0.1.0 <commit> [CHANGELOG.md]
#
# `sequence` crece con cada release (segundos desde 1970): los equipos rechazan un
# manifiesto con una secuencia menor que la ya vista, así nadie puede hacerlos volver
# a una versión vieja repitiendo un manifiesto antiguo pero bien firmado.
set -euo pipefail

ver="${1:?uso: scripts/release-manifest.sh X.Y.Z COMMIT [CHANGELOG]}"
commit="${2:?falta el commit}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo="${MAXOR_REPO_URL:-https://github.com/bgrados31/maxor-os}"

[[ "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]] || { echo "versión inválida: $ver" >&2; exit 1; }
[[ "$commit" =~ ^[0-9a-f]{40}$ ]] || { echo "el commit debe ser el hash completo (40 hex)" >&2; exit 1; }

# Resumen: el título en negrita de la primera entrada de las notas; si no lo tiene, su
# principio cortado en una palabra entera. Es lo que se ve en el aviso de versión nueva.
first="$("$root/scripts/release-notes.sh" "$ver" 2> /dev/null | grep -m1 '^- ' || true)"
if [[ "$first" =~ \*\*([^*]+)\*\* ]]; then
  summary="${BASH_REMATCH[1]}"
else
  summary="$(printf '%s' "$first" | sed -E 's/^- +//; s/`//g' | cut -c1-90 | sed -E 's/ [^ ]*$//')"
fi

jq -n --arg v "$ver" --arg c "$commit" --arg s "$summary" --arg u "$repo/releases/tag/v$ver" \
  --argjson seq "$(date +%s)" --arg d "$(date -u +%FT%TZ)" '{
    schema: 1,
    product: "maxor-os",
    version: $v,
    tag: ("v" + $v),
    commit: $c,
    sequence: $seq,
    published: $d,
    summary: $s,
    url: $u
  }'
