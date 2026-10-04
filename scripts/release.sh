#!/usr/bin/env bash
# Publica una versión de Maxor OS, firmada.
#
#   scripts/release.sh 0.1.0                 muestra el plan y no cambia nada
#   scripts/release.sh 0.1.0 --yes           lo ejecuta
#   scripts/release.sh 0.1.0 --assets-only   solo (re)sube el manifiesto firmado de una etiqueta ya creada
#
# Parte de `development`, mueve «Sin publicar» del CHANGELOG a la versión, la fusiona en
# `main`, crea la etiqueta vX.Y.Z FIRMADA, firma el manifiesto (manifest.json) y sube todo:
# la etiqueta con git y el manifiesto con la GitHub Release. Los equipos solo confían en
# lo firmado con la clave de keys/allowed_signers (ver docs/UPDATES.md); la clave privada
# es ~/.ssh/maxor-release (o MAXOR_RELEASE_KEY) y se crea con scripts/release-key.sh.
set -euo pipefail

die() { printf '✗ %s\n' "$*" >&2; exit 1; }
say() { printf '· %s\n' "$*"; }

ver="${1:-}"
yes=0
assets_only=0
shift || true
for a in "$@"; do
  case "$a" in
    --yes) yes=1 ;;
    --assets-only) assets_only=1 ;;
    *) die "opción desconocida: $a" ;;
  esac
done
[[ "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "uso: scripts/release.sh X.Y.Z [--yes | --assets-only]"

cd "$(git rev-parse --show-toplevel)"
tag="v$ver"
key="${MAXOR_RELEASE_KEY:-$HOME/.ssh/maxor-release}"
signers="$PWD/keys/allowed_signers"

# ── La clave: sin ella no se publica nada ────────────────────────────
for c in ssh-keygen gh jq; do command -v "$c" > /dev/null || die "falta el programa $c"; done
[ -f "$key" ] || die "no existe la clave de release $key (créala con scripts/release-key.sh)"
[ -f "$key.pub" ] || die "falta $key.pub (la parte pública de la clave)"
pub="$(cut -d' ' -f1,2 "$key.pub")"
grep -qF "$pub" "$signers" || die "la clave $key no está en keys/allowed_signers: los equipos no confiarían en ella"
gh auth status > /dev/null 2>&1 || die "gh no tiene sesión (gh auth login)"

gitssh() { git -c gpg.format=ssh -c "gpg.ssh.allowedSignersFile=$signers" "$@"; }

# Firma el manifiesto de una etiqueta y crea (o completa) su GitHub Release.
publish_release() {
  local commit work
  commit="$(git rev-parse "$tag^{commit}")"
  work="$(mktemp -d)"
  scripts/release-manifest.sh "$ver" "$commit" > "$work/manifest.json"
  ssh-keygen -Y sign -f "$key" -n maxor-release "$work/manifest.json" > /dev/null
  ssh-keygen -Y verify -f "$signers" -I maxor-release -n maxor-release -s "$work/manifest.json.sig" < "$work/manifest.json" > /dev/null \
    || die "el manifiesto recién firmado no verifica con keys/allowed_signers"
  scripts/release-notes.sh "$ver" > "$work/notes.md"
  if gh release view "$tag" > /dev/null 2>&1; then
    gh release upload "$tag" "$work/manifest.json" "$work/manifest.json.sig" --clobber
  else
    gh release create "$tag" "$work/manifest.json" "$work/manifest.json.sig" \
      --title "Maxor OS $ver" --notes-file "$work/notes.md" --verify-tag --latest
  fi
  rm -rf "$work"
}

if [ "$assets_only" = 1 ]; then
  git rev-parse -q --verify "refs/tags/$tag" > /dev/null || die "la etiqueta $tag no existe"
  gitssh verify-tag "$tag" > /dev/null 2>&1 || die "la etiqueta $tag no está firmada con la clave de release"
  publish_release
  say "Manifiesto firmado de $tag subido."
  exit 0
fi

[ "$(git branch --show-current)" = development ] || die "hay que estar en la rama development"
[ -z "$(git status --porcelain)" ] || die "hay cambios sin commit"
git fetch -q origin
[ "$(git rev-parse HEAD)" = "$(git rev-parse origin/development)" ] || die "development no coincide con origin/development (haz push o pull)"
git rev-parse -q --verify "refs/tags/$tag" > /dev/null && die "la etiqueta $tag ya existe"
grep -q '^## \[Sin publicar\]' CHANGELOG.md || die "CHANGELOG.md no tiene la sección [Sin publicar]"

pending="$(awk '/^## \[Sin publicar\]/{on=1;next} on&&/^## /{exit} on{print}' CHANGELOG.md | grep -cE '^- ' || true)"
[ "$pending" -gt 0 ] || die "[Sin publicar] está vacío: no hay nada que publicar"

say "Versión:     $ver (etiqueta $tag)"
say "Cambios:     $pending entradas en el CHANGELOG"
say "Commits:     $(git rev-list --count "$(git describe --tags --abbrev=0 2> /dev/null || git rev-list --max-parents=0 HEAD)"..HEAD) desde la versión anterior"
say "Clave:       $(printf '%s' "$pub" | ssh-keygen -lf - 2> /dev/null | cut -d' ' -f1,2 || echo "$key")"
say "Se hará:     comprobar el flake → mover «Sin publicar» a [$ver] → fusionar en main → etiqueta FIRMADA → manifiesto firmado → subir"

if [ "$yes" = 0 ]; then
  say "Esto fue solo el plan. Añade --yes para ejecutarlo."
  exit 0
fi

say "Comprobando el flake…"
nix flake check --no-build
nix eval --raw .#nixosConfigurations.nitro.config.system.build.toplevel.drvPath > /dev/null

date="$(date +%F)"
sed -i "s/^## \[Sin publicar\]$/## [Sin publicar]\n\n## [$ver] - $date/" CHANGELOG.md
scripts/release-notes.sh "$ver" > /dev/null || die "la sección [$ver] quedó vacía"
printf '%s\n' "$ver" > VERSION # `maxor --version` la lee al compilar
git commit -q -am "chore(release): v$ver"

git checkout -q main
git merge -q --no-ff development -m "release: v$ver"
# Etiqueta firmada con la clave de release (ssh-keygen te pedirá su contraseña).
git -c gpg.format=ssh -c "user.signingkey=$key" tag -s -a "$tag" -m "Maxor OS $ver"
gitssh verify-tag "$tag" > /dev/null 2>&1 || die "la etiqueta recién firmada no verifica"
git push -q origin main "$tag"
git checkout -q development
git merge -q --ff-only main

# development sigue con la próxima versión menor, marcada como en desarrollo.
IFS=. read -r major minor _ <<< "$ver"
next="$major.$((minor + 1)).0-dev"
printf '%s\n' "$next" > VERSION
git commit -q -am "chore: abrir $next"
git push -q origin development

publish_release || {
  say "La etiqueta ya está subida, pero falló la GitHub Release. Repítela con:"
  say "  scripts/release.sh $ver --assets-only"
  exit 1
}

say "Listo: $tag publicada, firmada, con su manifiesto firmado."
say "development sigue en $next."
