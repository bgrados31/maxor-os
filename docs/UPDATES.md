# Updates and releases

How a Maxor OS release gets from the maintainer to a running machine, and why you can trust it.
This is the contract; the code is `home/maxor/cmd/release.sh` (the engine), `scripts/release*.sh`
(the publishing side) and the Update screen of `maxor-tui`.

## In one minute

- Every release is **signed** with the project's release key. The matching public key ships inside
  the `maxor` package (`keys/allowed_signers`), so what is not signed by it is refused, wherever it
  came from (a mirror, a hijacked account, a man in the middle).
- Your machine asks for the **latest signed manifest** every hour with a conditional request
  (`ETag`: if nothing changed the server answers `304` with no body) and again when you open `maxor`.
- If there is a newer verified release you get **one** desktop notification per version, a banner on
  Home and a block on Update. You decide when to install it.
- Installing never changes the system halfway: NixOS keeps the previous generation, so
  `maxor rollback` always works.

## What is checked

| Check | What it stops |
|---|---|
| The manifest has a valid `ssh-keygen -Y` signature from the release key (namespace `maxor-release`) | Forged or tampered releases |
| The manifest has the exact shape (`schema`, `product`, `version`, `tag == v<version>`, 40-hex `commit`, integer `sequence`) | Signed-but-garbage input |
| `sequence` is **not lower** than the highest one ever seen on this machine | Replaying an old, validly signed manifest to push you back to a vulnerable version |
| On `apply`: the git **tag is signed** by the same key, and points at the commit the manifest names | A manifest that lies about which code it is |
| On `apply`: your checkout moves **only in a straight line** (`--ff-only`), and is clean | Overwriting your own commits or uncommitted work |
| On `apply`: it updates with the `flake.lock` of that release (`--no-lock`) | Installing something that was never tested together |
| CI re-verifies the tag, `VERSION`, the changelog, the manifest signature and the commit match (`.github/workflows/release.yml`) | A mistake while publishing |

Anything that fails a check is **ignored, never installed**, and is reported as a different state
from "no update":

| State | Meaning | What you see |
|---|---|---|
| `ok` | Verified. `available` says whether it is newer than what you run | "Maxor OS 0.2.0 is available" or "is the latest release" |
| `unavailable` | Could not look (offline, HTTP error, no key). **Not** "up to date": nothing was verified | A warning with the last time something verified |
| `insecure` | Looked, and it failed verification (`bad_signature`, `unsigned`, `invalid_manifest`, `rollback`) | A red notice, a critical notification (once), exit code 1 |
| `never` | Never checked | "Not checked yet" |

## Commands

```
maxor release check [--force] [--json] [--notify]   ask the channel, verify, remember the answer
maxor release status [--json]                       show the last answer, no network
maxor release apply [--yes]                         verify the tag, fast-forward, update the system
```

`check` does not hit the network again within 10 minutes unless you pass `--force`. Exit codes follow
`docs/CLI.md`: `0` ok, `4` could not reach the channel, `1` something failed verification.

The state lives in `~/.local/state/maxor/release.json` (and `release/` next to it: the verified
manifest, its signature, the `ETag` and the highest `sequence` seen).

## Where it runs

- **Timer** `maxor-release-check` (user): two minutes after login and then every hour, runs
  `maxor release check --notify --quiet`. Being offline is not a failure (`SuccessExitStatus=4`);
  a failed verification is, on purpose, so it shows up in `systemctl --user --failed`.
- **maxor-tui**: Home, Update and Exit ask once per session. Update's `v` installs, `r` and `c` ask
  again with `--force`. The install runs in the panel, with your password in its own field.

## For maintainers: publishing a release

One-time setup (the private key never enters the repository):

```
scripts/release-key.sh          # creates ~/.ssh/maxor-release (set a passphrase!) and registers the public key
git add keys && git commit -m "Release key"
```

Then rebuild the system so the `maxor` package carries the key. Back the private key up somewhere safe.

To publish:

```
scripts/release.sh 0.1.0          # shows the plan, changes nothing
scripts/release.sh 0.1.0 --yes    # runs it
```

It checks the flake, moves the changelog, merges into `main`, creates a **signed** tag, builds and signs
`manifest.json`, pushes, and creates the GitHub Release with `manifest.json` and `manifest.json.sig`
attached. If the release upload fails after the tag is already pushed, repeat only that part with
`scripts/release.sh 0.1.0 --assets-only`.

The manifest (`scripts/release-manifest.sh`) is small on purpose:

```json
{"schema":1,"product":"maxor-os","version":"0.2.0","tag":"v0.2.0",
 "commit":"<40 hex>","sequence":1791000000,"published":"…","summary":"…","url":"…"}
```

`sequence` is the release time in seconds, so it only grows.

## Limits, said plainly

- **The repository must be public** for machines to read the channel: GitHub answers `404` for
  anonymous requests to a private repository's releases, which shows up as `unavailable`. To test
  against anything else, set `MAXOR_RELEASE_URL` (any URL or `file://` directory holding
  `manifest.json` and `manifest.json.sig`).
- **`apply` follows a git checkout.** A machine that uses Maxor OS as a flake *input* (not a clone)
  is not supported by `apply` yet; the notice and verification already work for it.
- **One key, no rotation yet.** Losing or leaking the key means publishing a new
  `keys/allowed_signers` through a normal signed release (the old key signs the handover). A
  threshold or hardware-key scheme is future work.
- **No manifest expiry yet.** An attacker who can block the channel can keep you on an old version
  silently; that is why `unavailable` and a stale last-verified time are shown and never read as "up to date".
- The signature proves who published a release, not that the code is bug-free; it is also not a
  replacement for reading the changelog.
