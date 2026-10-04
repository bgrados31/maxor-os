# Hardware: drivers for your machine

Maxor OS picks the microcode, video drivers and laptop settings from the real machine.
It works the same on Intel, AMD, NVIDIA, hybrid graphics and virtual machines.

## How it works

Nix evaluates the configuration without looking at the machine, so detection happens in two steps:

1. **`maxor hardware detect --write`** reads `/sys` and `/proc` (it does not need root) and saves
   `hosts/<host>/hardware.json`: CPU vendor, GPUs with their ID and PCI bus, whether it is a laptop
   and whether it is a virtual machine.
2. **`modules/hardware.nix`** reads that file and enables what fits. The host only points to it:

   ```nix
   maxor.hardware.report = ./hardware.json;
   ```

`maxor hardware` shows what was detected and what Maxor will use. `maxor doctor` warns when the
machine no longer matches `hardware.json` (another GPU, another computer) so you can regenerate it.

## What gets configured

| Detected | Maxor enables |
|---|---|
| Intel CPU | Intel microcode and `thermald` (not in virtual machines) |
| AMD CPU | AMD microcode (the kernel handles `amd_pstate`) |
| AMD GPU | `amdgpu`, early KMS (the splash comes up at native resolution) and OpenCL |
| Intel GPU | the `modesetting` driver and `intel-media-driver` (hardware video) |
| NVIDIA GPU | the proprietary driver with `modesetting`; open modules if the GPU is Turing (RTX 20 / GTX 16) or newer, closed ones on Pascal and Maxwell |
| Laptop with iGPU + NVIDIA | PRIME offload: the desktop runs on the integrated GPU and `nvidia-offload <app>` uses the NVIDIA; it saves battery and heat |
| Laptop | power profiles (power saver, balanced, performance) |
| QEMU/VirtualBox/VMware virtual machine | guest tools (clipboard, resolution) |
| Always | redistributable firmware (Wi-Fi, GPU) and `fwupd` (BIOS, SSD, peripherals) |

An NVIDIA GPU older than Maxwell (Kepler and earlier) is not supported by the current driver:
the build warns that a legacy one is needed.

## Manual overrides

Everything can be overridden from `hosts/<host>/configuration.nix`:

```nix
maxor.hardware.nvidia.open = false;        # force closed NVIDIA modules
services.thermald.enable = lib.mkForce false;
```

## Current limits

- Laptops with a MUX switch and vendor GPU modes are not detected.
- Only the first GPU of each vendor is used.
- A GPU from another vendor (virtio, VMware, etc.) stays on the generic Mesa drivers.
- What was detected must be regenerated when you change machines: it is not detected on every boot.
