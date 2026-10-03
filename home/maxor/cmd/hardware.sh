# ── Hardware: detectar el equipo ─────────────────────────────────────
# Lee /sys y /proc (no pide root) y produce un JSON pequeño que
# modules/hardware.nix usa para elegir microcódigo, drivers de vídeo y
# ajustes de portátil. Nix evalúa sin ver el equipo, por eso el resultado se
# guarda en hosts/<equipo>/hardware.json y se versiona con el resto.
maxor_cmd hardware system "show detect"

hw_any() { local p; for p in "$@"; do [ -e "$p" ] && return 0; done; return 1; } # ¿existe alguna coincidencia del glob?

# hw_read variable archivo → contenido del archivo, o vacío si no se puede leer.
# Sin lanzar `cat`: el detector lee unos veinte archivos de /sys.
hw_read() {
  if [ -r "$2" ]; then printf -v "$1" '%s' "$(< "$2")"; else printf -v "$1" '%s' ""; fi
}

hw_pci_busid() { # 0000:01:00.0 → PCI:1:0:0 (decimal, como pide NixOS)
  local a="$1"
  printf 'PCI:%d:%d:%d' "0x${a:5:2}" "0x${a:8:2}" "${a:11:1}"
}

hw_detect() {
  local vid cpu model laptop=false virt=none bt=false chassis sysv
  vid="$(grep -m1 '^vendor_id' /proc/cpuinfo | awk '{print $3}')"
  case "$vid" in
    GenuineIntel) cpu=intel ;;
    AuthenticAMD) cpu=amd ;;
    *) cpu=other ;;
  esac
  model="$(grep -m1 '^model name' /proc/cpuinfo | cut -d: -f2- | sed 's/^ *//')"

  hw_read chassis /sys/class/dmi/id/chassis_type
  case "$chassis" in 8 | 9 | 10 | 11 | 14 | 30 | 31 | 32) laptop=true ;; esac
  hw_any '/sys/class/power_supply/BAT'* && laptop=true

  if grep -q -m1 -w hypervisor /proc/cpuinfo; then
    hw_read sysv /sys/class/dmi/id/sys_vendor
    case "$sysv" in
      innotek*) virt=virtualbox ;;
      VMware*) virt=vmware ;;
      QEMU* | "Red Hat"* | "Proxmox"*) virt=qemu ;;
      *) virt=other ;;
    esac
  fi

  hw_any '/sys/class/bluetooth/hci'* && bt=true

  local d addr class ven dev gv bv bvf gpus=""
  for d in /sys/bus/pci/devices/*; do
    hw_read class "$d/class"
    case "$class" in 0x0300* | 0x0302* | 0x0380*) ;; *) continue ;; esac
    addr="${d##*/}"
    hw_read ven "$d/vendor"
    hw_read dev "$d/device"
    hw_read bvf "$d/boot_vga"
    if [ "$bvf" = 1 ]; then bv=true; else bv=false; fi
    case "$ven" in
      0x8086) gv=intel ;;
      0x1002) gv=amd ;;
      0x10de) gv=nvidia ;;
      *) gv=other ;;
    esac
    gpus+="$(jq -cn --arg v "$gv" --arg id "${ven#0x}:${dev#0x}" --arg bus "$(hw_pci_busid "$addr")" \
      --argjson primary "$bv" \
      '{vendor: $v, id: $id, bus: $bus, primary: $primary}')"$'\n'
  done

  jq -n --arg cpu "$cpu" --arg model "$model" --argjson laptop "$laptop" \
    --arg virt "$virt" --argjson bt "$bt" --argjson gpus "$(printf '%s' "$gpus" | jq -s '.')" \
    '{version: 1, cpu: {vendor: $cpu, model: $model}, gpus: $gpus, laptop: $laptop, virt: $virt, bluetooth: $bt}'
}

hw_show() {
  local j="$1" kind virtinfo line
  echo
  ui_open @hardware.title
  ui_section @hardware.sec_machine
  ui_row info "$(jq -r '.cpu.model' <<< "$j")"
  if [ "$(jq -r '.laptop' <<< "$j")" = true ]; then msg kind @hardware.laptop; else msg kind @hardware.desktop; fi
  virtinfo="$(jq -r 'if .virt != "none" then .virt else "" end' <<< "$j")"
  if [ -n "$virtinfo" ]; then
    msg line @hardware.vm "$virtinfo"
    ui_row info "$kind · $line"
  else
    ui_row info "$kind"
  fi
  ui_section @hardware.sec_graphics
  if [ "$(jq '.gpus | length' <<< "$j")" = 0 ]; then ui_row warn @hardware.no_gpu; fi
  while read -r v id bus; do
    ui_row info "$v · $id · $bus"
  done < <(jq -r '.gpus[] | "\(.vendor) \(.id) \(.bus)"' <<< "$j")
  ui_section @hardware.sec_uses
  # La lógica de qué se activa vive en modules/hardware.nix; aquí solo se resume.
  local cpuv gv hybrid
  cpuv="$(jq -r '.cpu.vendor' <<< "$j")"
  case "$cpuv" in
    intel) ui_row ok @hardware.use_intel ;;
    amd) ui_row ok @hardware.use_amd ;;
    *) ui_row ok @hardware.use_cpu_other ;;
  esac
  gv="$(jq -r '[.gpus[].vendor] | join(" ")' <<< "$j")"
  hybrid="$(jq -r 'if .laptop and ([.gpus[].vendor] | index("nvidia")) and ([.gpus[].vendor] | map(select(. == "intel" or . == "amd")) | length > 0) then "yes" else "no" end' <<< "$j")"
  if [ "$hybrid" = yes ]; then ui_row ok @hardware.use_hybrid
  elif [[ "$gv" == *nvidia* ]]; then ui_row ok @hardware.use_nvidia
  elif [[ "$gv" == *amd* ]]; then ui_row ok @hardware.use_amd_gpu
  elif [[ "$gv" == *intel* ]]; then ui_row ok @hardware.use_intel_gpu
  else ui_row ok @hardware.use_mesa; fi
  if [ "$(jq -r '.laptop' <<< "$j")" = true ]; then ui_row ok @hardware.use_laptop; fi
  if [ -n "$virtinfo" ]; then ui_row ok @hardware.use_guest "$virtinfo"; fi
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
          track_file "$out"
          ui_say ok @hardware.saved "$out"
          ;;
        *) usage_error hardware ;;
      esac
      ;;
    *) usage_error hardware ;;
  esac
}
