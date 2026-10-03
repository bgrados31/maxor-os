# Hardware: drivers según el equipo

Maxor OS elige microcódigo, drivers de vídeo y ajustes de portátil a partir del equipo real.
Funciona igual en Intel, AMD, NVIDIA, gráficos híbridos y máquinas virtuales.

## Cómo funciona

Nix evalúa la configuración sin mirar la máquina, así que la detección va en dos pasos:

1. **`maxor hardware detect --write`** lee `/sys` y `/proc` (no pide root) y guarda
   `hosts/<equipo>/hardware.json`: fabricante de CPU, GPUs con su ID y bus PCI, si es portátil y
   si es una máquina virtual.
2. **`modules/hardware.nix`** lee ese archivo y activa lo que corresponde. El host solo apunta a él:

   ```nix
   maxor.hardware.report = ./hardware.json;
   ```

`maxor hardware` muestra lo detectado y lo que Maxor usará. `maxor doctor` avisa si el equipo
cambió respecto al `hardware.json` (otra GPU, otra máquina) para que lo regeneres.

## Qué se configura

| Detectado | Maxor activa |
|---|---|
| CPU Intel | microcódigo Intel y `thermald` (no en máquinas virtuales) |
| CPU AMD | microcódigo AMD (el kernel gestiona `amd_pstate`) |
| GPU AMD | `amdgpu`, KMS temprano (el splash sale a resolución nativa) y OpenCL |
| GPU Intel | controlador `modesetting` y `intel-media-driver` (vídeo por hardware) |
| GPU NVIDIA | controlador propietario con `modesetting`; módulos abiertos si la GPU es Turing (RTX 20 / GTX 16) o más nueva, cerrados en Pascal y Maxwell |
| Portátil con iGPU + NVIDIA | PRIME offload: el escritorio va por la integrada y `nvidia-offload <app>` usa la NVIDIA; ahorra batería y calor |
| Portátil | perfiles de energía (ahorro, equilibrado, rendimiento) |
| Máquina virtual QEMU/VirtualBox/VMware | herramientas de invitado (portapapeles, resolución) |
| Siempre | firmware redistribuible (Wi-Fi, GPU) y `fwupd` (BIOS, SSD, periféricos) |

Una GPU NVIDIA anterior a Maxwell (Kepler y más antiguas) no la soporta el controlador actual:
la compilación avisa de que hace falta uno legacy.

## Ajustes a mano

Todo se puede pisar desde `hosts/<equipo>/configuration.nix`:

```nix
maxor.hardware.nvidia.open = false;        # forzar módulos cerrados de NVIDIA
services.thermald.enable = lib.mkForce false;
```

## Límites actuales

- No se detectan portátiles con conmutador MUX ni modos de GPU de fabricante.
- Solo se usa la primera GPU de cada fabricante.
- Una GPU de otro fabricante (virtio, VMware, etc.) queda con los controladores genéricos de Mesa.
- Lo detectado debe regenerarse al cambiar de equipo: no se detecta solo en cada arranque.
