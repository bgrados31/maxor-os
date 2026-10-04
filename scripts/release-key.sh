#!/usr/bin/env bash
# Crea la clave de release (una sola vez) y registra su parte pública en keys/allowed_signers.
#
#   scripts/release-key.sh            crea ~/.ssh/maxor-release si no existe y la registra
#   MAXOR_RELEASE_KEY=/ruta scripts/release-key.sh   usa otra ruta
#
# La clave privada firma las releases (etiqueta de git y manifiesto) y nunca entra en el
# repositorio: solo la pública va en keys/, y de ahí pasa al paquete `maxor`, que es lo que
# decide en qué firmas confiar. ssh-keygen te pedirá una contraseña para la clave: ponla.
# Haz una copia de la clave privada en un sitio seguro; si se pierde no se pueden firmar
# más releases con ella (habría que publicar una versión nueva de keys/allowed_signers).
set -euo pipefail

die() { printf '✗ %s\n' "$*" >&2; exit 1; }
say() { printf '· %s\n' "$*"; }

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
key="${MAXOR_RELEASE_KEY:-$HOME/.ssh/maxor-release}"
signers="$root/keys/allowed_signers"
[ -f "$signers" ] || die "no encuentro $signers"

if [ -f "$key" ]; then
  say "Ya existe la clave $key: se reutiliza."
else
  mkdir -p "$(dirname "$key")"
  chmod 700 "$(dirname "$key")"
  ssh-keygen -t ed25519 -C maxor-release -f "$key"
fi
[ -f "$key.pub" ] || ssh-keygen -y -f "$key" > "$key.pub"

pub="$(cut -d' ' -f1,2 "$key.pub")"
if grep -qF "$pub" "$signers"; then
  say "La clave ya estaba registrada en keys/allowed_signers."
else
  printf 'maxor-release namespaces="git,maxor-release" %s\n' "$pub" >> "$signers"
  say "Registrada en keys/allowed_signers."
fi
cp "$key.pub" "$root/keys/maxor-release.pub"

say "Huella: $(ssh-keygen -lf "$key.pub" | cut -d' ' -f1,2)"
say "Siguiente: git add keys && git commit -m 'Clave de release', y reconstruye el sistema para que el paquete la lleve."
