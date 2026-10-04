# Security policy

## Supported versions

Maxor OS is under active development and has no stable versions. Only the `main` branch receives
fixes.

## How to report a vulnerability

Use [GitHub's private report](https://github.com/bgrados31/maxor-os/security/advisories/new).
Do not open a public issue or publish details before there is a fix.

Include, if you can:

- which component it affects (the `maxor` CLI, the theme engine, the system configuration);
- how to reproduce it;
- the impact you think it has.

Receipt will be acknowledged within a reasonable time and you will be kept informed about the fix.

## What is in scope

- The `maxor` CLI and the theme engine: for example, a theme that can run code, write outside the
  allowed paths or inject content into configuration files.
- `maxor backup` and `maxor restore`: for example, a backup that can write outside its folders.
- System configurations that weaken security in a non-obvious way.

Out of scope: vulnerabilities in third-party components (Hyprland, DMS, nixpkgs), which should be
reported to their own projects.

## Security model of themes

A theme only provides data. The engine validates that every color is `#rrggbb` before using it,
runs no file from the theme, and only writes to `~/.config/maxor/current/`,
`~/.local/state/maxor/` and two keys of the DMS configuration. Details in
[docs/THEMING.md](docs/THEMING.md#security).

## Security model of the password prompt

The full-screen app asks for your password in its own field only when `sudo` needs it to change
the system. The password goes to `sudo` through standard input and is never saved, logged or shown
(it is drawn as dots). See [docs/TUI.md](docs/TUI.md).

## Security model of updates and releases

Every release is signed. `maxor release` only trusts a manifest, and `maxor release apply` only
trusts a git tag, signed by the release key whose public half is committed in
[`keys/allowed_signers`](keys/allowed_signers) and shipped inside the `maxor` package.

- Key fingerprint: `SHA256:9W5S6iKHa26a9wzHK1MWLzfsMXCbe+CG/GflwisYi7Y` (ed25519, `maxor-release`).
  Check it yourself with `ssh-keygen -lf keys/maxor-release.pub`, and compare it with a copy you got
  from somewhere other than this repository before you rely on it for the first time.
- Anything that fails verification is ignored and reported as such; it is never installed, and "could
  not check" is never shown as "up to date". An older manifest than one already seen is rejected.
- The private key lives only on the maintainer's machine, protected by a passphrase. CI never holds it:
  CI only re-verifies what was published.
- `main` and the `v*` tags cannot be force-pushed or deleted, and a published tag cannot be moved.
- Releases are not reproducible byte for byte yet, and there is no key rotation or manifest expiry yet.
  See [docs/UPDATES.md](docs/UPDATES.md#limits-said-plainly).

If you think the release key has leaked, report it privately (above). The response is to publish a new
`keys/allowed_signers` in a release signed by the old key, and to announce the new fingerprint here.
