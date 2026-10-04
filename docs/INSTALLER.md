# Maxor installer

The plan for installing Maxor OS on a computer: a live ISO, a guided installer, and an engine that
does the real work. This page is the contract. Each piece is built and tested on its own, so a new
screen (the TUI today, a graphical one later), an unattended install and the automated tests all
drive the same engine.

## Goals

- **Safe by default.** Nothing is written to a disk before a typed confirmation, every destructive
  step can be rehearsed with `--dry-run`, and a machine with Windows is never touched without a plan
  the user has seen.
- **Resumable and honest.** Each stage is idempotent and leaves a sentinel; a failed install says which
  stage failed, keeps its log and can be resumed or undone. No "something went wrong".
- **One engine, many fronts.** The TUI, a future graphical installer, `maxor-install --answers file`
  and the VM tests all go through the same JSON contract.
- **Tested for real.** The unit layer (answers, plan, dry-run) runs in seconds; the real layer
  partitions, formats and installs onto virtual disks in a NixOS test and boots the result.
- **Reproducible and verifiable.** The ISO comes from the flake, and ships with a checksum and a
  signature made by the release key, so the update system's trust root also covers it.

Not goals for the first release: BIOS/legacy boot, ZFS, RAID, Secure Boot signing, and installing from
a language other than English (the catalog is ready for more).

## Pieces

```
installer/
  engine/                 maxor-install: the engine (bash), one file per stage
    maxor-install         orchestrator: validate answers, build the plan, run stages, emit events
    lib/                  common, answers, plan, preflight, disk, luks, filesystem, host, install,
                          bootloader, finish
  iso/                    the live image as a NixOS configuration (nixosConfigurations.maxor-iso)
  schema/answers.v1.json  the answers contract
tui/internal/screens/install*.go   the guided installer (a screen of maxor-tui)
tests/installer/                    bats (unit) + NixOS tests (real disks)
```

```
 ┌────────────┐  answers.json   ┌───────────────┐  events.jsonl  ┌────────────┐
 │ TUI / GUI  │ ───────────────▶│ maxor-install │ ──────────────▶│ TUI / GUI  │
 │ unattended │                 │  (stages)     │  (progress)    │ log, tests │
 └────────────┘                 └───────────────┘                └────────────┘
```

### The answers (`answers.v1.json`)

A single JSON document, versioned (`"schema": 1`), written by whoever collects the answers and read
only by the engine. It never travels in environment variables or arguments. It lives in
`/run/maxor-install/answers.json` (mode 0600, on a tmpfs), and the password is a crypt(3) hash, never
plain text. The engine validates it against the schema and refuses to start on anything unknown.

```json
{
  "schema": 1,
  "locale": "en_US.UTF-8", "keymap": "la-latin1", "timezone": "America/Lima",
  "disk": {
    "device": "/dev/nvme0n1",
    "strategy": "whole",            // whole | alongside
    "region": {"start": 0, "end": 0},   // alongside only: sectors of the free region chosen
    "filesystem": "btrfs",          // ext4 | btrfs
    "encrypt": {"enabled": false},  // passphrase is delivered separately, see below
    "swap": {"kind": "zram", "gib": 0}, // zram | file | none
    "confirmed": "ERASE"            // the typed confirmation; the engine requires it
  },
  "machine": {"hostname": "maxor"},
  "user": {"name": "bryan", "fullname": "Bryan", "password_hash": "$6$…", "autologin": false},
  "look": {"theme": "sakura", "profiles": ["dev"], "extra_apps": []},
  "hardware": {"report": "auto"},   // auto = detect at install time; or an inline hardware.json
  "release": {"channel": "stable"}
}
```

Secrets that must not sit in a file (the LUKS passphrase) arrive on a file descriptor
(`--secret-fd`) and are held only in memory.

### The wizard

Ordered, one screen per step, each with a validation gate and a way back. Each step is a registered
unit (id, title, which answers it produces, a validator, a short explanation), so adding a step is
adding a file, not editing a flow.

| # | Step | Decides | Gate |
|---|---|---|---|
| 1 | Welcome | language of the installer | none |
| 2 | Keyboard | console and desktop layout, with a test field | a valid layout |
| 3 | Network | Ethernet or Wi-Fi (scan, password) | a working connection, or an explicit "offline" |
| 4 | Region | timezone, locale | valid names |
| 5 | Disk | which disk (with size, model, what is on it) | not the live medium |
| 6 | Strategy | erase the disk, or install alongside an existing system | alongside needs a free region ≥ 40 GiB |
| 7 | Storage | filesystem, encryption (+ passphrase), swap | passphrase strength and confirmation |
| 8 | Account | name, hostname, password, autologin | valid names, password confirmed |
| 9 | Look | initial theme (live preview), usage profiles, extra apps | none |
| 10 | Hardware | what was detected and what will be installed (GPU mode on hybrid NVIDIA) | review |
| 11 | Summary | everything, with the partition layout drawn | typed `ERASE` when a disk is erased |
| 12 | Install | live progress by stage, log one key away | cannot leave while it runs |
| 13 | Done | reboot, or stay in the live session; where the install log is | none |

### The engine

`maxor-install --answers FILE [--dry-run] [--resume] [--events FD|FILE]`.

Stages, in order. Each one is a file in `installer/engine/lib/`, takes only validated answers, is
idempotent (a sentinel in `/run/maxor-install/stages/`) and can be skipped by the plan when it does not
apply:

1. **preflight**: root, UEFI, target big enough, not the live medium, clock sane, network reachable
   (or offline allowed), Secure Boot state, nothing on the disk mounted, enough RAM for the plan.
2. **disk**: GPT partitioning. `whole` wipes the disk (typed confirmation); `alongside` creates the
   boot and root partitions in the free region the user saw, re-checks it is still free, and never
   resizes or deletes what exists (shrinking Windows is left to Windows).
3. **luks**: optional LUKS2 on root.
4. **filesystem**: `mkfs`, btrfs subvolumes (`@`, `@home`, `@snapshots`), mounts under `/mnt`, swap.
5. **host**: writes the installed machine's flake (below) and its `hardware.json`, from the answers.
6. **install**: `nixos-install --flake` from the closure on the ISO, falling back to the binary cache
   for what is not there.
7. **bootloader**: systemd-boot on the shared or a dedicated EFI partition, with the Windows entry kept.
8. **finish**: carry the network configuration over, set the password, preseed `maxor firstrun`,
   unmount, report.

Events are JSON lines (`{"stage":"disk","state":"start|ok|fail|skip","message":…,"progress":0.42}`)
written to the events channel and to `/var/log/maxor-install.log`; the screens draw progress from
them. A failure names the stage, keeps the log and leaves the target mounted for inspection.

**Dry run.** `--dry-run` prints every destructive command (`DRYRUN: …`) and every file it would write,
runs no command that changes the system and never probes real disks beyond reading. It is the
contract tests' oracle: the plan for a given answers file is deterministic text.

**Safety rules the engine enforces, not the screen**
- `disk.confirmed` must be exactly `ERASE` for `whole`; `alongside` requires the plan hash the user saw.
- A disk with a mounted filesystem, the live medium, or a partition labeled like a working Maxor
  install is refused unless the answers carry the explicit reclaim flag.
- Secrets never reach the log, the events or the process list.
- On a machine with Windows, the existing EFI partition is mounted and only the Maxor directory is
  added to it; the engine aborts if it cannot back up the existing boot entries first.

### The installed system

The new machine gets a small flake of its own, owned by the user (`~/nixos-config`, a git repository),
that only says what is particular to it and takes everything else from Maxor OS as a flake input:

```nix
{
  inputs.maxor-os.url = "github:bgrados31/maxor-os/v0.1.0";
  outputs = { maxor-os, ... }: {
    nixosConfigurations.maxor = maxor-os.lib.mkSystem {
      hostname = "maxor"; user = "bryan"; timezone = "America/Lima";
      locale = "en_US.UTF-8"; keymap = "la-latin1";
      hardware = ./hardware.json; profiles = [ "dev" ];
    };
  };
}
```

`maxor-os.lib.mkSystem` is the distribution's single entry point (it replaces the personal host
configurations). Updating is then `maxor release apply`: it verifies the signed tag, bumps the input to
it and rebuilds. This also gives machines that are not a clone of this repository a supported way to
follow releases.

### The live ISO

`nixosConfigurations.maxor-iso`, built with `nix build .#iso`. It boots, in this order of importance:

- **Boot menu** with Maxor's name and branding: normal, safe graphics (`nomodeset`), copy to RAM.
- **Live session** straight into Hyprland with Maxor Shell, so the system can be tried before installing.
- **The installer**, offered on first boot as a full-screen window, and reachable with `maxor install`.
- **The closure of the default install** baked in, so a default install works offline; everything else
  comes from the binary cache.
- A checksum and a signature (`SHA256SUMS` + `.sig`, made with the release key) published as release
  assets; the update system's key covers it.

NVIDIA's proprietary driver is not redistributable inside the ISO: it is fetched during the install,
and hybrid graphics fall back to the integrated GPU if that fetch is not possible.

## Testing

| Layer | What | Where |
|---|---|---|
| Unit | answers validation, plan, dry-run matrix (every strategy × filesystem × encryption × swap) | bats, seconds, in CI |
| Real | partition, format, mount and `nixos-install` onto virtual disks, then boot the result | `nix build .#checks.x86_64-linux.installer-vm` (needs KVM) |
| Visual | boot the ISO in QEMU with UEFI, drive the wizard, take screenshots | the headless VM runner |
| Release | ISO checksum and signature verify with the release key | CI, on the release tag |

The installer is never first tried on real hardware: VM, then a spare machine, then the laptop that
dual-boots Windows.

## Milestones

| | What | Done when |
|---|---|---|
| M0 | This plan | reviewed |
| M1 | `lib.mkSystem`: the distribution stops hardcoding a user; hosts become data | **done**: `nitro` and the VM build from it; 654 home files and all of `/etc` compared, only the intended file changed |
| M2 | Engine: answers schema, plan, preflight, dry-run, stages, events | **done**: 52 tests; the generated machine flake evaluates against this repository |
| M3 | Real install onto a virtual disk in a NixOS test | **done**: the engine installs offline and the installed system boots (`installer-full`) |
| M4 | ISO: branded boot, live session, closure baked in, boots in QEMU with UEFI | verified by screenshots |
| M5 | The wizard (TUI) driving the engine | a full install through the screens in the VM |
| M6 | Alongside (dual boot), LUKS, hybrid NVIDIA, binary cache | each tested in the real layer |
| M7 | Signed ISO as a release asset | checksum and signature verify in CI |
