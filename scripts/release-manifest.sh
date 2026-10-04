#!/usr/bin/env bash
# Print the manifest (JSON) of a release. scripts/release.sh signs it.
#
#   scripts/release-manifest.sh 0.1.0 <commit>
#
# `sequence` grows with every release (seconds since 1970): machines reject a manifest whose sequence is
# lower than the highest one they have seen, so nobody can push them back to an old version by replaying
# an old but validly signed manifest.
set -euo pipefail

ver="${1:?usage: scripts/release-manifest.sh X.Y.Z COMMIT}"
commit="${2:?missing commit}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo="${MAXOR_REPO_URL:-https://github.com/bgrados31/maxor-os}"

[[ "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]] || { echo "invalid version: $ver" >&2; exit 1; }
[[ "$commit" =~ ^[0-9a-f]{40}$ ]] || { echo "the commit must be the full hash (40 hex characters)" >&2; exit 1; }

# Summary: the bold title of the first entry of the notes; without one, the start of the entry cut at a
# whole word. It is what shows up in the update notification.
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
