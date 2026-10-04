# Test virtual machine

A throwaway Maxor OS you can boot in a window, to try changes, the full-screen app and the release
system without touching your real machine. It is the real thing: the same modules, desktop, login and
`maxor` CLI, on QEMU (`hosts/vm/`), without the laptop's NVIDIA or partitions.

```
nix run .#vm        this repository's version
nix run .#vm-old    pretends to be 0.0.1, so the published release shows up as an available update
```

- **Login:** the first session of every boot signs in on its own. Log out to see the real greeter. The
  user is `bryan`, the password `maxor` (only for this machine).
- **Disk:** `~/.local/state/maxor-vm/<name>.qcow2`. It is kept between boots; delete it to start from scratch.
- **Window:** QEMU/GTK with 3D acceleration (virtio-gpu). It needs a graphical session on the host and `/dev/kvm`.
- **Network:** the VM reaches the internet through QEMU's user networking, so `maxor release check`
  talks to the real GitHub channel and verifies the real signature.

What to try in `vm-old`:

```
maxor release check --force      Maxor OS 0.1.0 is available (you have 0.0.1)
maxor                            the full-screen app: banner on Home, block on Update
```

## Driving it without a window (for automated checks)

`MAXOR_VM_HEADLESS=1 nix run .#vm-old` starts it with no window and a QEMU monitor socket at
`$XDG_RUNTIME_DIR/maxor-vm-old.monitor`. `screendump file.ppm` and `sendkey` through that socket let a
script look at the screen and type (the keyboard layout is Latin American: `-` is the `slash` key).

## Not the same test as `release-vm`

`nix build .#checks.x86_64-linux.release-vm -L` runs the release system across two small virtual
machines (a server and a client) as an automated check. This page is about the full desktop VM, for
looking and trying by hand.
