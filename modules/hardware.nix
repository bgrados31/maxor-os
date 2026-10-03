{ config, lib, pkgs, ... }:

# Drivers y ajustes según el equipo detectado.
#
# Nix evalúa sin mirar la máquina, así que la detección (`maxor hardware detect
# --write`) deja un hardware.json junto al host y este módulo lo traduce:
# microcódigo, drivers de vídeo (Intel, AMD, NVIDIA e híbridos), ajustes de
# portátil y herramientas de invitado en máquinas virtuales.
# Cualquier valor se puede pisar desde el host con lib.mkForce.
let
  cfg = config.maxor.hardware;
  hw = if cfg.report == null then null else builtins.fromJSON (builtins.readFile cfg.report);

  gpus = if hw == null then [ ] else hw.gpus;
  byVendor = v: lib.findFirst (g: g.vendor == v) null gpus;
  nvidia = byVendor "nvidia";
  amd = byVendor "amd";
  intel = byVendor "intel";
  igpu = if intel != null then intel else amd;
  laptop = hw != null && hw.laptop;
  hybrid = laptop && nvidia != null && igpu != null;

  # ID de dispositivo PCI de NVIDIA en número ("10de:28e1" → 0x28e1).
  nvidiaDev = lib.fromHexString (lib.last (lib.splitString ":" nvidia.id));
  # Los módulos de kernel abiertos de NVIDIA exigen Turing (RTX 20 / GTX 16) o
  # más nuevo: sus IDs empiezan en 0x1e00. Pascal y Maxwell usan el cerrado.
  nvidiaOpen = if cfg.nvidia.open != null then cfg.nvidia.open else nvidiaDev >= 7680;
in
{
  options.maxor.hardware = {
    report = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = "hardware.json generado por `maxor hardware detect --write`.";
    };
    nvidia.open = lib.mkOption {
      type = lib.types.nullOr lib.types.bool;
      default = null;
      description = "Forzar módulos de kernel abiertos (true) o cerrados (false) de NVIDIA. null = según la generación de la GPU.";
    };
  };

  config = lib.mkMerge [
    {
      hardware.enableRedistributableFirmware = true; # Wi-Fi, GPU y microcódigo
      services.fwupd.enable = true; # actualizaciones de firmware (BIOS, SSD, periféricos)
      warnings = lib.optional (cfg.report == null)
        "Maxor: sin hardware.json no se eligen drivers por equipo. Ejecuta `maxor hardware detect --write` y apunta maxor.hardware.report a ese archivo.";
    }

    (lib.mkIf (hw != null) {
      # Portátiles: perfiles de energía (los muestra el shell); sobremesa los deja pasar.
      services.power-profiles-daemon.enable = lib.mkDefault hw.laptop;

      # ── Procesador ──
      hardware.cpu.intel.updateMicrocode = lib.mkIf (hw.cpu.vendor == "intel") true;
      hardware.cpu.amd.updateMicrocode = lib.mkIf (hw.cpu.vendor == "amd") true;
      services.thermald.enable = lib.mkDefault (hw.cpu.vendor == "intel" && hw.virt == "none");

      # ── Gráficos ──
      services.xserver.videoDrivers =
        if nvidia != null then [ "nvidia" ]
        else if amd != null then [ "amdgpu" ]
        else if intel != null then [ "modesetting" ]
        else [ ];

      # ── Máquinas virtuales ──
      services.qemuGuest.enable = hw.virt == "qemu";
      services.spice-vdagentd.enable = hw.virt == "qemu";
      virtualisation.virtualbox.guest.enable = hw.virt == "virtualbox";
      virtualisation.vmware.guest.enable = hw.virt == "vmware";
    })

    (lib.mkIf (hw != null && amd != null) {
      hardware.amdgpu.initrd.enable = lib.mkDefault true; # KMS temprano: el splash sale a resolución nativa
      hardware.amdgpu.opencl.enable = lib.mkDefault true;
    })

    (lib.mkIf (hw != null && intel != null) {
      hardware.graphics.extraPackages = [ pkgs.intel-media-driver ]; # decodificación de vídeo por hardware
    })

    (lib.mkIf (hw != null && nvidia != null) {
      warnings = lib.optional (nvidiaDev < 4928)
        "Maxor: la GPU NVIDIA ${nvidia.id} es anterior a Maxwell; el controlador actual puede no soportarla (hace falta uno legacy).";

      hardware.nvidia = {
        modesetting.enable = true;
        open = lib.mkDefault nvidiaOpen;
        nvidiaSettings = true;
        powerManagement.enable = lib.mkDefault laptop;
        package = lib.mkDefault config.boot.kernelPackages.nvidiaPackages.stable;
      };
    })

    # Portátil con iGPU + NVIDIA: el escritorio va por la integrada y la NVIDIA
    # entra bajo demanda con `nvidia-offload <app>` (ahorra batería y calor).
    (lib.mkIf hybrid {
      hardware.nvidia.prime = {
        offload.enable = true;
        offload.enableOffloadCmd = true;
        nvidiaBusId = nvidia.bus;
      } // (if intel != null then { intelBusId = intel.bus; } else { amdgpuBusId = amd.bus; });
    })
  ];
}
