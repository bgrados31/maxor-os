#!/usr/bin/env bash
# Print the notes of a version: the "## [X.Y.Z]" section of CHANGELOG.md.
# Usage: scripts/release-notes.sh 0.1.0
set -euo pipefail

ver="${1:?usage: scripts/release-notes.sh X.Y.Z}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

notes="$(awk -v v="$ver" '
  $0 ~ "^## \\[" v "\\]" { on = 1; next }
  on && /^## / { exit }
  on { print }
' "$root/CHANGELOG.md")"

# Trim blank lines at the start and the end.
notes="$(printf '%s\n' "$notes" | sed -e '/./,$!d' | sed -e ':a' -e '/^\n*$/{$d;N;ba' -e '}')"
[ -n "$notes" ] || { echo "CHANGELOG.md has no section [$ver] with content" >&2; exit 1; }
printf '%s\n' "$notes"
