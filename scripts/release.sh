#!/usr/bin/env bash
# Publica una versión de Maxor OS.
#
#   scripts/release.sh 0.1.0          muestra el plan y no cambia nada
#   scripts/release.sh 0.1.0 --yes    lo ejecuta
#
# Parte de `development`, mueve «Sin publicar» del CHANGELOG a la versión, la
# fusiona en `main`, crea la etiqueta vX.Y.Z y sube todo. La GitHub Release la
# crea la CI al ver la etiqueta (.github/workflows/release.yml).
set -euo pipefail

die() { printf '✗ %s\n' "$*" >&2; exit 1; }
say() { printf '· %s\n' "$*"; }

ver="${1:-}"
yes=0
[ "${2:-}" = "--yes" ] && yes=1
[[ "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "uso: scripts/release.sh X.Y.Z [--yes]"

cd "$(git rev-parse --show-toplevel)"
tag="v$ver"

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
say "Se hará:     comprobar el flake → mover «Sin publicar» a [$ver] → fusionar en main → etiquetar → subir"

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
git tag -a "$tag" -m "Maxor OS $ver"
git push -q origin main "$tag"
git checkout -q development
git merge -q --ff-only main

# development sigue con la próxima versión menor, marcada como en desarrollo.
IFS=. read -r major minor _ <<< "$ver"
next="$major.$((minor + 1)).0-dev"
printf '%s\n' "$next" > VERSION
git commit -q -am "chore: abrir $next"
git push -q origin development

say "Listo: $tag publicada. La CI crea la Release con las notas del CHANGELOG."
say "development sigue en $next."
