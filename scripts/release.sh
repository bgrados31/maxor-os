#!/usr/bin/env bash
# Publish a signed Maxor OS release.
#
#   scripts/release.sh 0.1.0                 print the plan and change nothing
#   scripts/release.sh 0.1.0 --yes           run it
#   scripts/release.sh 0.1.0 --assets-only   only (re)upload the signed manifest of an existing tag
#
# Starts from `development`, moves [Unreleased] in CHANGELOG.md to the version, merges into `main`,
# creates the SIGNED tag vX.Y.Z, signs the manifest (manifest.json) and uploads everything: the tag with
# git and the manifest with the GitHub Release. Machines only trust what is signed by the key in
# keys/allowed_signers (see docs/UPDATES.md); the private key is ~/.ssh/maxor-release (or MAXOR_RELEASE_KEY)
# and is created with scripts/release-key.sh.
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
    *) die "unknown option: $a" ;;
  esac
done
[[ "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "usage: scripts/release.sh X.Y.Z [--yes | --assets-only]"

cd "$(git rev-parse --show-toplevel)"
tag="v$ver"
key="${MAXOR_RELEASE_KEY:-$HOME/.ssh/maxor-release}"
signers="$PWD/keys/allowed_signers"

# ── The key: nothing is published without it ─────────────────────────
for c in ssh-keygen gh jq; do command -v "$c" > /dev/null || die "missing program: $c"; done
[ -f "$key" ] || die "the release key $key does not exist (create it with scripts/release-key.sh)"
[ -f "$key.pub" ] || die "missing $key.pub (the public half of the key)"
pub="$(cut -d' ' -f1,2 "$key.pub")"
grep -qF "$pub" "$signers" || die "the key $key is not in keys/allowed_signers: machines would not trust it"
gh auth status > /dev/null 2>&1 || die "gh is not logged in (gh auth login)"

gitssh() { git -c gpg.format=ssh -c "gpg.ssh.allowedSignersFile=$signers" "$@"; }

# Sign the manifest of a tag and create (or complete) its GitHub Release.
publish_release() {
  local commit work
  commit="$(git rev-parse "$tag^{commit}")"
  work="$(mktemp -d)"
  scripts/release-manifest.sh "$ver" "$commit" > "$work/manifest.json"
  ssh-keygen -Y sign -f "$key" -n maxor-release "$work/manifest.json" > /dev/null
  ssh-keygen -Y verify -f "$signers" -I maxor-release -n maxor-release -s "$work/manifest.json.sig" < "$work/manifest.json" > /dev/null \
    || die "the manifest that was just signed does not verify against keys/allowed_signers"
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
  git rev-parse -q --verify "refs/tags/$tag" > /dev/null || die "the tag $tag does not exist"
  gitssh verify-tag "$tag" > /dev/null 2>&1 || die "the tag $tag is not signed by the release key"
  publish_release
  say "Signed manifest of $tag uploaded."
  exit 0
fi

[ "$(git branch --show-current)" = development ] || die "you must be on the development branch"
[ -z "$(git status --porcelain)" ] || die "there are uncommitted changes"
git fetch -q origin
[ "$(git rev-parse HEAD)" = "$(git rev-parse origin/development)" ] || die "development differs from origin/development (push or pull first)"
git rev-parse -q --verify "refs/tags/$tag" > /dev/null && die "the tag $tag already exists"
grep -q '^## \[Unreleased\]' CHANGELOG.md || die "CHANGELOG.md has no [Unreleased] section"

pending="$(awk '/^## \[Unreleased\]/{on=1;next} on&&/^## /{exit} on{print}' CHANGELOG.md | grep -cE '^- ' || true)"
[ "$pending" -gt 0 ] || die "[Unreleased] is empty: there is nothing to publish"

say "Version:    $ver (tag $tag)"
say "Changes:    $pending entries in the changelog"
say "Commits:    $(git rev-list --count "$(git describe --tags --abbrev=0 2> /dev/null || git rev-list --max-parents=0 HEAD)"..HEAD) since the previous version"
say "Key:        $(printf '%s' "$pub" | ssh-keygen -lf - 2> /dev/null | cut -d' ' -f1,2 || echo "$key")"
say "It will:    check the flake → move [Unreleased] to [$ver] → merge into main → SIGNED tag → signed manifest → upload"

if [ "$yes" = 0 ]; then
  say "That was only the plan. Add --yes to run it."
  exit 0
fi

say "Checking the flake…"
nix flake check --no-build
nix eval --raw .#nixosConfigurations.nitro.config.system.build.toplevel.drvPath > /dev/null

date="$(date +%F)"
sed -i "s/^## \[Unreleased\]$/## [Unreleased]\n\n## [$ver] - $date/" CHANGELOG.md
scripts/release-notes.sh "$ver" > /dev/null || die "the [$ver] section ended up empty"
printf '%s\n' "$ver" > VERSION # `maxor --version` reads it at build time
git commit -q -am "chore(release): v$ver"

git checkout -q main
git merge -q --no-ff development -m "release: v$ver"
# Tag signed with the release key (ssh-keygen will ask for its passphrase).
git -c gpg.format=ssh -c "user.signingkey=$key" tag -s -a "$tag" -m "Maxor OS $ver"
gitssh verify-tag "$tag" > /dev/null 2>&1 || die "the tag that was just signed does not verify"
git push -q origin main "$tag"
git checkout -q development
git merge -q --ff-only main

# development continues with the next minor version, marked as in development.
IFS=. read -r major minor _ <<< "$ver"
next="$major.$((minor + 1)).0-dev"
printf '%s\n' "$next" > VERSION
git commit -q -am "chore: open $next"
git push -q origin development

publish_release || {
  say "The tag is already pushed, but the GitHub Release failed. Repeat it with:"
  say "  scripts/release.sh $ver --assets-only"
  exit 1
}

say "Done: $tag published, signed, with its signed manifest."
say "development continues at $next."
