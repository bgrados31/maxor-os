# Branches and releases

## Branches

| Branch | Purpose |
|---|---|
| `development` | Day-to-day work. It is the default branch: commits and merges land here. |
| `main` | Published versions only. Nobody commits here; it only moves with a release. |
| `feat/…`, `fix/…` | Changes that take long or are risky. They branch off `development` and come back with a merge or a pull request. |

A small change can go straight to `development`. Anything that touches boot, login or graphics
goes on its own branch and is tested before merging.

## Versions

Numbered `0.N.0` until 1.0, which will arrive with the installer. While it is `0.x`, an
incompatible change bumps the second number; a fix or a small improvement bumps the third.

## Commits and changelog

Commits follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)
(`feat:`, `fix:`, `perf:`, `docs:`…), which makes it possible to draft the changelog.
`CHANGELOG.md` follows the "Keep a Changelog" format: what has not shipped yet goes under
`## [Unreleased]`, grouped into Added, Changed and Fixed, written for people who use Maxor and not
for people who read the code.

## Cutting a version

```
scripts/release.sh 0.1.0          shows the plan and changes nothing
scripts/release.sh 0.1.0 --yes    runs it
```

You must be on `development`, with no pending changes, up to date with `origin` and with
something under "Unreleased". The script then:

1. checks the flake (`nix flake check --no-build` and the evaluation of `nitro`);
2. moves "Unreleased" to the `## [0.1.0] - date` section, writes `VERSION` (which `maxor --version` reads at build time) and commits it;
3. merges `development` into `main` (no fast-forward, so the version stays marked);
4. creates the tag `v0.1.0`, **signed** with the release key (`~/.ssh/maxor-release`, it asks for its passphrase), verifies it, and pushes `main` and the tag;
5. moves `development` forward to `main` and opens the next minor version (`VERSION` becomes `0.2.0-dev`) in its own commit;
6. builds `manifest.json`, signs it, verifies the signature against `keys/allowed_signers`, and creates the **GitHub Release** with the notes of that section and both files attached.

The signed manifest is what machines read to learn that a release exists; see
[UPDATES.md](UPDATES.md) for how it is verified. CI (`.github/workflows/release.yml`) then
re-checks the tag, `VERSION`, the notes and the manifest from the outside and fails loudly if
anything does not match. `scripts/release-notes.sh 0.1.0` prints the notes locally so you can
review them first.

The key is created once with `scripts/release-key.sh`. If the Release upload fails after the tag is
already pushed, repeat only that part with `scripts/release.sh 0.1.0 --assets-only`.

## If something goes wrong

- **The plan refuses to continue:** the message says what is missing (branch, uncommitted changes, empty changelog, release key).
- **A wrong version was pushed:** do not move the tag; publish the next version, versions are not
  reused. Published `v*` tags are protected by a ruleset (no update, no delete): if one really has to
  go, disable the `immutable-release-tags` ruleset in the repository settings, delete the tag and
  the Release, and enable it again.
- **Machines already saw the manifest.** They remember the highest `sequence` they have seen and will
  refuse an older manifest, so a bad release is answered with a newer one, not by going back.
