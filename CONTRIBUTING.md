# How to contribute

Thanks for wanting to improve Maxor OS. This document explains how to propose changes.

## Before you start

- Read the [architecture](docs/ARCHITECTURE.md) and the [roadmap](docs/ROADMAP.md) to see where
  the project is going.
- For anything big, open an issue first and describe the idea; it avoids wasted work.
- Follow the [code of conduct](CODE_OF_CONDUCT.md).

## Environment

You need NixOS (or Nix with flakes) and Git.

```sh
git clone https://github.com/bgrados31/maxor-os
cd maxor-os
nix flake check --no-build          # evaluates the configuration
nix build .#nixosConfigurations.nitro.config.system.build.toplevel   # builds the system
nix build .#checks.x86_64-linux.cli-tests .#checks.x86_64-linux.tui  # CLI and full-screen app tests
```

`nixos-rebuild switch` should only be run on a test machine or a VM.

## Workflow

1. Fork and create a branch from `development`: `feat/my-change` or `fix/my-fix`, and open the
   pull request against `development`. `main` only receives releases (see
   [RELEASING.md](docs/RELEASING.md)).
2. Keep changes small and focused. One change, one purpose.
3. Check that `nix flake check --no-build` passes, that the system builds and that the tests pass.
4. Open a pull request and fill in the template.

## Commit messages

We use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/), in English, in the
imperative mood and without a trailing period.

```
feat(theming): add the Obsidiana theme
fix(lockscreen): use the active theme template when locking
docs: explain how to adapt the host to another GPU
```

Common types: `feat`, `fix`, `docs`, `refactor`, `chore`, `ci`.

## Style

- **Nix:** two-space indentation, a short comment about the *why* of every non-obvious block, and
  no machine-specific value outside `hosts/`.
- **Shell (`maxor` CLI):** must pass `shellcheck` (`writeShellApplication` runs it). User-facing
  text goes through the message catalog (`home/maxor/lib/lang/en.sh`) with `@key`, never inline.
- **Go (full-screen app):** `go vet` and `go test ./...` must pass; they run on every build.
- **Lua (Hyprland):** one responsibility per file once it is modularized.
- **Documentation:** in English, direct, with examples that can be copied and pasted.

## Contributing a theme

A theme is a folder with `colors.json`, `theme.toml` and, optionally, `style.json`; the format is
in [docs/THEMING.md](docs/THEMING.md). The official ones are the folders in `themes/`: add a
folder and the build takes care of the rest.

Requirements:

- All eight palette colors, as `#rrggbb`.
- WCAG AA contrast (4.5:1 or more) between `fg` and `bg`, `mu` and `bg`, `ac` and `s`, and `on`
  and `ac`.
- A license that allows redistribution (CC0-1.0 is recommended).
- A theme never includes scripts or executables.

## Translating

A translation is one JSON file and needs no Go: see [docs/TRANSLATING.md](docs/TRANSLATING.md).

## Security

Do not open a public issue for a vulnerability. Follow [SECURITY.md](SECURITY.md).
