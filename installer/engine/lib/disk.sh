# ── Etapa disk: particionado GPT ─────────────────────────────────────
# `whole` borra el disco y crea la ESP (1 GiB) y la raíz. `alongside` crea las mismas dos
# particiones SOLO en el hueco libre que el usuario vio, tras comprobar que sigue libre, y nunca
# toca lo que ya existe. Tipos GPT: ESP y «Linux root x86-64» (así systemd las reconoce).
DISK_ESP_TYPE="C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
DISK_ROOT_TYPE="4F68BCE3-E8CD-4DB1-96E7-FBCAF984B709"
DISK_ESP_SECTORS=2097152 # 1 GiB

# Huecos libres de un disco, en sectores, a partir de `sfdisk -J` (entrada estándar).
# Alinea al MiB y descarta lo menor de 64 MiB.
DISK_JQ_FREE='
def align: ((. + 2047) / 2048 | floor) * 2048;
(.partitiontable // {}) as $t
| ($t.firstlba // 2048) as $first
| ($t.lastlba // 0) as $last
| ([$t.partitions[]? | {s: .start, e: (.start + .size - 1)}] | sort_by(.s)) as $parts
| (reduce ($parts[], {s: ($last + 1), e: ($last + 1)}) as $x ({cur: $first, gaps: []};
    (.cur | align) as $a
    | if (($x.s - 1) - $a + 1) >= 131072
      then .gaps += [{start: $a, end: ($x.s - 1), sectors: (($x.s - 1) - $a + 1)}] else . end
    | .cur = ($x.e + 1)))
| .gaps
'

# disk_free_json /dev/x → los huecos libres de ese disco como JSON.
disk_free_json() {
  local dump
  dump="$(sfdisk -J "$1" 2> /dev/null || true)"
  if [ -z "$dump" ]; then
    # sin tabla de particiones: todo el disco, menos los extremos reservados de GPT
    local sectors
    sectors="$(blockdev --getsz "$1" 2> /dev/null || echo 0)"
    [[ "$sectors" =~ ^[0-9]+$ ]] || sectors=0
    # un dispositivo sin medio (lector de tarjetas vacío, disquetera) no tiene huecos: no es un error
    jq -nc --argjson n "$sectors" 'if $n < 4096 then [] else [{start: 2048, end: ($n - 2049), sectors: ($n - 2048 - 2048)}] end'
  else
    jq -c "$DISK_JQ_FREE" <<< "$dump"
  fi
}

# disk_probe_json → los discos del equipo, con lo que hay en ellos y su espacio libre. Solo lee.
disk_probe_json() {
  local lsb disks d out='[]' free
  lsb="$(lsblk -J -b -o NAME,PATH,SIZE,TYPE,MODEL,TRAN,RM,RO,FSTYPE,LABEL,PARTLABEL,MOUNTPOINTS)"
  # Solo discos con medio: sin disqueteras (fd), memoria comprimida (zram), lectores vacíos (tamaño 0)
  # ni dispositivos de red (nbd).
  disks="$(jq -r '.blockdevices[] | select(.type == "disk" and ((.size | tonumber) > 0) and (.name | test("^(fd|zram|ram|nbd)[0-9]*$") | not)) | .path' <<< "$lsb")"
  for d in $disks; do
    free="$(disk_free_json "$d")"
    out="$(jq -c --arg d "$d" --argjson free "$free" --argjson all "$lsb" '
      . + [ $all.blockdevices[] | select(.path == $d)
        | { path: .path, model: ((.model // "") | gsub("^\\s+|\\s+$"; "")), size: (.size | tonumber), transport: (.tran // ""),
            removable: ((.rm | tostring) == "true" or (.rm | tostring) == "1"),
            readonly: ((.ro | tostring) == "true" or (.ro | tostring) == "1"),
            mountpoints: [ (.mountpoints // [])[] | select(. != null) ],
            partitions: [ (.children // [])[] | { path: .path, size: (.size | tonumber), fstype: (.fstype // ""), label: (.label // ""),
                          partlabel: (.partlabel // ""), mountpoints: [ (.mountpoints // [])[] | select(. != null) ] } ],
            free: $free } ]
      | map(. + { mounted: (([.partitions[].mountpoints[]] + .mountpoints) | length > 0),
                  windows: ([.partitions[] | select(.fstype == "ntfs" or .fstype == "BitLocker")] | length > 0),
                  maxor: ([.partitions[] | select(.partlabel == "maxor-root" or .partlabel == "MAXOR-ESP")] | length > 0) })' <<< "$out")"
  done
  printf '%s\n' "$out"
}

# El hueco libre elegido sigue libre: que quepa entero dentro de alguno de los huecos actuales.
disk_region_still_free() { # disk start end
  disk_free_json "$1" | jq -e --argjson s "$2" --argjson e "$3" 'any(.[]; .start <= $s and .end >= $e)' > /dev/null
}

disk_sfdisk_script() { # los dos sectores de inicio y fin → el guion de sfdisk
  case "$1" in
    whole)
      cat << EOS
label: gpt
unit: sectors
first-lba: 2048

start=2048, size=$DISK_ESP_SECTORS, type=$DISK_ESP_TYPE, name="MAXOR-ESP"
type=$DISK_ROOT_TYPE, name="maxor-root"
EOS
      ;;
    alongside)
      local s="$2" e="$3"
      cat << EOS
start=$s, size=$DISK_ESP_SECTORS, type=$DISK_ESP_TYPE, name="MAXOR-ESP"
start=$((s + DISK_ESP_SECTORS)), size=$((e - s - DISK_ESP_SECTORS + 1)), type=$DISK_ROOT_TYPE, name="maxor-root"
EOS
      ;;
  esac
}

stage_disk() {
  local disk strategy script rs re
  disk="$(ans .disk.device)"
  strategy="$(ans .disk.strategy)"
  script="$IN_STATE/disk.sfdisk"
  [ "$IN_DRY" = 1 ] || mkdir -p "$IN_STATE"

  if [ "$strategy" = whole ]; then
    in_run wipefs --all --force "$disk"
    disk_sfdisk_script whole | write_file "$script" 600
    in_run_stdin "$script" sfdisk --wipe always "$disk"
  else
    rs="$(ans .disk.region.start)"
    re="$(ans .disk.region.end)"
    if [ "$IN_DRY" = 1 ]; then
      printf 'DRYRUN: check that sectors %s..%s of %s are still free\n' "$rs" "$re" "$disk"
    else
      disk_region_still_free "$disk" "$rs" "$re" || in_die "$IN_EX_FAIL" "the free region $rs..$re of $disk is no longer free; nothing was changed"
    fi
    disk_sfdisk_script alongside "$rs" "$re" | write_file "$script" 600
    in_run_stdin "$script" sfdisk --append "$disk"
  fi
  in_run udevadm settle
  in_run partprobe "$disk"

  # los nodos de las particiones que acabamos de crear (con `alongside` no se sabe el número hasta ahora)
  if [ "$IN_DRY" != 1 ]; then
    local dump esp root
    dump="$(sfdisk -J "$disk")"
    esp="$(jq -r '.partitiontable.partitions[] | select(.name == "MAXOR-ESP") | .node' <<< "$dump" | tail -n1)"
    root="$(jq -r '.partitiontable.partitions[] | select(.name == "maxor-root") | .node' <<< "$dump" | tail -n1)"
    in_is_block "$esp" && in_is_block "$root" || in_die "$IN_EX_FAIL" "the new partitions did not appear on $disk"
    jq -nc --arg esp "$esp" --arg root "$root" '{esp: $esp, root: $root}' > "$IN_STATE/devices.json"
  fi
}
