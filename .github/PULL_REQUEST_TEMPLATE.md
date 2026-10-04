## What changes

<!-- Describe the change and why it is needed. -->

## Type

- [ ] Fix
- [ ] New feature
- [ ] Theme
- [ ] Documentation
- [ ] Refactor or maintenance

## How it was tested

- [ ] `nix flake check --no-build` passes
- [ ] The system builds (`nix build .#nixosConfigurations.nitro.config.system.build.toplevel`)
- [ ] CLI and full-screen app tests pass (`nix build .#checks.x86_64-linux.cli-tests .#checks.x86_64-linux.tui`)
- [ ] Tried on a machine or VM (say which)

## Notes

<!-- Screenshots, limitations, changes that affect people who already use it. -->
