# ── Hardware: detectar el equipo ─────────────────────────────────────
# Lee /sys y /proc (no pide root) y produce un JSON pequeño que
# modules/hardware.nix usa para elegir microcódigo, drivers de vídeo y
# ajustes de portátil. Nix evalúa sin ver el equipo, por eso el resultado se
# guarda en hosts/<equipo>/hardware.json y se versiona con el resto.

hw_pci_busid() { # 0000:01:00.0 → PCI:1:0:0 (decimal, como pide NixOS)
  local a="$1"
  printf 'PCI:%d:%d:%d' "0x${a:5:2}" "0x${a:8:2}" "${a:11:1}"
}

hw_any() { local p; for p in "$@"; do [ -e "$p" ] && return 0; done; return 1; } # ¿existe alguna coincidencia del glob?

hw_detect() {
  local vid cpu model laptop=false virt=none bt=false chassis sysv
  vid="$(grep -m1 '^vendor_id' /proc/cpuinfo | awk '{print $3}')"
  case "$vid" in
    GenuineIntel) cpu=intel ;;
    AuthenticAMD) cpu=amd ;;
    *) cpu=other ;;
  esac
  model="$(grep -m1 '^model name' /proc/cpuinfo | cut -d: -f2- | sed 's/^ *//')"

  chassis="$(cat /sys/class/dmi/id/chassis_type 2> /dev/null || echo 0)"
  case "$chassis" in 8 | 9 | 10 | 11 | 14 | 30 | 31 | 32) laptop=true ;; esac
  hw_any '/sys/class/power_supply/BAT'* && laptop=true

  if grep -q -m1 -w hypervisor /proc/cpuinfo; then
    sysv="$(cat /sys/class/dmi/id/sys_vendor 2> /dev/null)"
    case "$sysv" in
      innotek*) virt=virtualbox ;;
      VMware*) virt=vmware ;;
      QEMU* | "Red Hat"* | "Proxmox"*) virt=qemu ;;
      *) virt=other ;;
    esac
  fi

  hw_any '/sys/class/bluetooth/hci'* && bt=true

  local d addr class ven dev gv gpus=""
  for d in /sys/bus/pci/devices/*; do
    class="$(cat "$d/class")"
    case "$class" in 0x0300* | 0x0302* | 0x0380*) ;; *) continue ;; esac
    addr="$(basename "$d")"
    ven="$(cat "$d/vendor")"
    dev="$(cat "$d/device")"
    case "$ven" in
      0x8086) gv=intel ;;
      0x1002) gv=amd ;;
      0x10de) gv=nvidia ;;
      *) gv=other ;;
    esac
    gpus+="$(jq -cn --arg v "$gv" --arg id "${ven#0x}:${dev#0x}" --arg bus "$(hw_pci_busid "$addr")" \
      --argjson primary "$([ "$(cat "$d/boot_vga" 2> /dev/null)" = 1 ] && echo true || echo false)" \
      '{vendor: $v, id: $id, bus: $bus, primary: $primary}')"$'\n'
  done

  jq -n --arg cpu "$cpu" --arg model "$model" --argjson laptop "$laptop" \
    --arg virt "$virt" --argjson bt "$bt" --argjson gpus "$(printf '%s' "$gpus" | jq -s '.')" \
    '{version: 1, cpu: {vendor: $cpu, model: $model}, gpus: $gpus, laptop: $laptop, virt: $virt, bluetooth: $bt}'
}

hw_show() {
  local j="$1" n
  echo
  ui_open "maxor · hardware"
  ui_section "Equipo"
  ui_row info "$(jq -r '.cpu.model' <<< "$j")"
  ui_row info "$(jq -r 'if .laptop then "portátil" else "sobremesa" end' <<< "$j")$(jq -r 'if .virt != "none" then " · máquina virtual (" + .virt + ")" else "" end' <<< "$j")"
  ui_section "Gráficos"
  n="$(jq '.gpus | length' <<< "$j")"
  if [ "$n" = 0 ]; then ui_row warn "no se detectó ninguna GPU"; fi
  jq -r '.gpus[] | "\(.vendor) \(.id) \(.bus)"' <<< "$j" | while read -r v id bus; do
    ui_row info "$v · $id · $bus"
  done
  ui_section "Maxor usará"
  jq -r '
    (if .cpu.vendor == "intel" then "microcódigo Intel y thermald"
     elif .cpu.vendor == "amd" then "microcódigo AMD (amd_pstate lo gestiona el kernel)"
     else "sin microcódigo específico" end),
    (if ([.gpus[].vendor] | index("nvidia")) and ([.gpus[].vendor] | map(select(. == "intel" or . == "amd")) | length > 0) and .laptop
     then "gráficos híbridos: iGPU en el escritorio y NVIDIA bajo demanda (nvidia-offload)"
     elif ([.gpus[].vendor] | index("nvidia")) then "controlador NVIDIA"
     elif ([.gpus[].vendor] | index("amd")) then "controlador AMD (amdgpu, Vulkan y OpenCL)"
     elif ([.gpus[].vendor] | index("intel")) then "controlador Intel (Mesa y aceleración de vídeo)"
     else "controladores genéricos de Mesa" end),
    (if .laptop then "perfiles de energía y control térmico de portátil" else empty end),
    (if .virt != "none" then "herramientas de invitado para " + .virt else empty end)
  ' <<< "$j" | while read -r line; do ui_row ok "$line"; done
  ui_line ""
  ui_close
  echo
}

cmd_hardware() {
  local sub="${1:-show}" j out
  shift || true
  j="$(hw_detect)"
  case "$sub" in
    show) hw_show "$j" ;;
    detect)
      case "${1:-}" in
        "") printf '%s\n' "$j" ;;
        --write)
          need_flake
          out="${2:-$flake_dir/hosts/$host/hardware.json}"
          mkdir -p "$(dirname "$out")"
          printf '%s\n' "$j" > "$out"
          ui_say ok "Hardware guardado en $out"
          ;;
        *) die "uso: maxor hardware detect [--write [ruta]]" ;;
      esac
      ;;
    *) die "uso: maxor hardware [show | detect [--write [ruta]]]" ;;
  esac
}
