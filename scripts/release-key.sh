#!/usr/bin/env bash
# Create the release signing key (once) and register its public half in keys/allowed_signers.
#
#   scripts/release-key.sh                        creates ~/.ssh/maxor-release if missing and registers it
#   MAXOR_RELEASE_KEY=/path scripts/release-key.sh   uses another path
#
# The private key signs releases (the git tag and the manifest) and never enters the repository: only
# the public key goes in keys/, and from there into the `maxor` package, which decides whose signatures
# to trust. ssh-keygen will ask for a passphrase: set one. Back the private key up somewhere safe; if it
# is lost, no more releases can be signed with it (a new keys/allowed_signers would have to be published).
set -euo pipefail

die() { printf '✗ %s\n' "$*" >&2; exit 1; }
say() { printf '· %s\n' "$*"; }

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
key="${MAXOR_RELEASE_KEY:-$HOME/.ssh/maxor-release}"
signers="$root/keys/allowed_signers"
[ -f "$signers" ] || die "cannot find $signers"

if [ -f "$key" ]; then
  say "The key $key already exists: reusing it."
else
  mkdir -p "$(dirname "$key")"
  chmod 700 "$(dirname "$key")"
  ssh-keygen -t ed25519 -C maxor-release -f "$key"
fi
[ -f "$key.pub" ] || ssh-keygen -y -f "$key" > "$key.pub"

pub="$(cut -d' ' -f1,2 "$key.pub")"
if grep -qF "$pub" "$signers"; then
  say "The key was already registered in keys/allowed_signers."
else
  printf 'maxor-release namespaces="git,maxor-release" %s\n' "$pub" >> "$signers"
  say "Registered in keys/allowed_signers."
fi
cp "$key.pub" "$root/keys/maxor-release.pub"

say "Fingerprint: $(ssh-keygen -lf "$key.pub" | cut -d' ' -f1,2)"
say "Next: git add keys && git commit -m 'Release key', then rebuild the system so the package carries it."
