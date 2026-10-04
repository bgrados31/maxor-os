load helper

# Motor del instalador (installer/engine): respuestas, plan, compuertas de seguridad, etapas y
# reanudación. Las órdenes destructivas se sustituyen por shims que anotan cómo las llamaron.

setup() { load_engine; }

calls() { cat "$W/calls" 2> /dev/null || true; }

# ── respuestas ───────────────────────────────────────────────────────

@test "unas respuestas válidas pasan y se completan con los valores por defecto" {
  mk
  run main validate --answers "$W/a.json"
  [ "$status" = 0 ]
  ans_load "$W/a.json"
  [ "$(ans .disk.filesystem)" = btrfs ]
  [ "$(ans .disk.swap.kind)" = zram ]
  [ "$(ans .look.theme)" = sakura ]
  [ "$(ans .locale)" = en_US.UTF-8 ]
  [ "$(ans .hardware.report)" = auto ]
}

@test "un campo desconocido se rechaza, en cualquier nivel" {
  mk '.stray = 1'
  run main validate --answers "$W/a.json"
  [ "$status" = "$IN_EX_ANSWERS" ]
  [[ "$output" == *"answers.stray: unknown field"* ]]
  mk '.disk.extra = 1'
  run main validate --answers "$W/a.json"
  [[ "$output" == *"answers.disk.extra: unknown field"* ]]
}

@test "cada campo con formato malo se señala, todos a la vez" {
  mk '.disk.device = "/dev/zzz" | .machine.hostname = "Bad_Host" | .user.name = "Ana" | .keymap = "a b" | .timezone = "lima"'
  run main validate --answers "$W/a.json"
  [ "$status" = "$IN_EX_ANSWERS" ]
  [[ "$output" == *"disk.device: has an invalid format"* ]]
  [[ "$output" == *"machine.hostname: has an invalid format"* ]]
  [[ "$output" == *"user.name: has an invalid format"* ]]
  [[ "$output" == *"keymap: has an invalid format"* ]]
  [[ "$output" == *"timezone: has an invalid format"* ]]
}

@test "las opciones fuera de la lista se rechazan" {
  mk '.disk.strategy = "nuke"'
  run main validate --answers "$W/a.json"
  [[ "$output" == *"disk.strategy: must be one of whole, alongside"* ]]
  mk '.disk.filesystem = "ntfs"'
  run main validate --answers "$W/a.json"
  [[ "$output" == *"disk.filesystem: must be one of ext4, btrfs"* ]]
  mk '.schema = 2'
  run main validate --answers "$W/a.json"
  [[ "$output" == *"schema: must be one of 1"* ]]
}

@test "una contraseña en claro no pasa y el hash nunca aparece en los errores" {
  mk '.user.password_hash = "plaintext"'
  run main validate --answers "$W/a.json"
  [ "$status" = "$IN_EX_ANSWERS" ]
  [[ "$output" == *"user.password_hash: has an invalid format"* ]]
  mk '.user.name = "root"'
  run main validate --answers "$W/a.json"
  [[ "$output" == *"user.name: is a reserved name"* ]]
  [[ "$output" != *'$6$salt1234'* ]]
}

@test "un nombre completo con dos puntos o saltos de línea se rechaza (rompería /etc/passwd)" {
  mk '.user.fullname = "Ana: root"'
  run main validate --answers "$W/a.json"
  [[ "$output" == *"user.fullname: has an invalid format"* ]]
}

@test "los perfiles tienen que existir en el catálogo y no repetirse" {
  mk '.look.profiles = ["dev", "zzz"]'
  run main validate --answers "$W/a.json"
  [[ "$output" == *"look.profiles: unknown profile zzz"* ]]
  mk '.look.profiles = ["dev", "dev"]'
  run main validate --answers "$W/a.json"
  [[ "$output" == *"look.profiles: has repeated profiles"* ]]
}

@test "un archivo que no es JSON es un error claro" {
  echo 'esto no es json' > "$W/a.json"
  run main validate --answers "$W/a.json"
  [ "$status" = "$IN_EX_ANSWERS" ]
  [[ "$output" == *"not valid JSON"* ]]
}

@test "un swap en archivo necesita tamaño" {
  mk '.disk.swap = {kind: "file", gib: 0}'
  run main validate --answers "$W/a.json"
  [[ "$output" == *"a swap file needs at least 1 GiB"* ]]
}

# ── compuertas de seguridad ──────────────────────────────────────────

@test "borrar el disco entero exige escribir ERASE" {
  mk 'del(.disk.confirmed)'
  run main plan --answers "$W/a.json"
  [ "$status" = "$IN_EX_ANSWERS" ]
  [[ "$output" == *"type ERASE"* ]]
  [[ "$output" != *DRYRUN* ]]
  mk '.disk.confirmed = "erase"'
  run main plan --answers "$W/a.json"
  [ "$status" = "$IN_EX_ANSWERS" ]
}

@test "instalar junto a otro sistema exige región, la huella del plan y escribir INSTALL" {
  mk '.disk.strategy = "alongside" | del(.disk.confirmed)'
  run main plan --answers "$W/a.json"
  [ "$status" = "$IN_EX_ANSWERS" ]
  [[ "$output" == *"disk.region: is required"* ]]
  [[ "$output" == *"disk.plan_hash: is required"* ]]
  [[ "$output" == *"type INSTALL"* ]]
}

@test "una región al revés se rechaza" {
  mk '.disk.strategy = "alongside" | .disk.region = {start: 5000000, end: 4000000} | .disk.plan_hash = ("a" * 64) | .disk.confirmed = "INSTALL"'
  run main plan --answers "$W/a.json"
  [[ "$output" == *"end must be after start"* ]]
}

@test "con una huella que no es la del plan, no se hace nada" {
  mk '.disk.strategy = "alongside" | .disk.region = {start: 4000000, end: 200000000} | .disk.plan_hash = ("a" * 64) | .disk.confirmed = "INSTALL"'
  run main plan --answers "$W/a.json"
  [ "$status" = "$IN_EX_ANSWERS" ]
  [[ "$output" == *"plan_hash does not match"* ]]
  [[ "$output" != *DRYRUN:\ sfdisk* ]]
}

@test "con la huella correcta el plan de instalar junto a otro sistema sale" {
  mk '.disk.strategy = "alongside" | .disk.region = {start: 4000000, end: 200000000} | .disk.confirmed = "INSTALL" | .disk.plan_hash = ("a" * 64)'
  h="$(main hash --answers "$W/a.json")"
  mk ".disk.strategy = \"alongside\" | .disk.region = {start: 4000000, end: 200000000} | .disk.confirmed = \"INSTALL\" | .disk.plan_hash = \"$h\""
  run main plan --answers "$W/a.json"
  [ "$status" = 0 ]
  [[ "$output" == *"sfdisk --append /dev/vda < "* ]]
  [[ "$output" == *"start=4000000, size=2097152"* ]]
  # sin wipefs ni --wipe: lo que ya existe no se toca
  [[ "$output" != *wipefs* ]]
  [[ "$output" != *"--wipe"* ]]
}

@test "la huella no cambia con la contraseña ni con la confirmación, y sí con cualquier otra cosa" {
  mk
  a="$(main hash --answers "$W/a.json")"
  mk '.user.password_hash = "$6$othersalt$zzzzzzzzzzzzzzzzzzzzzzzzzzzz"'
  b="$(main hash --answers "$W/a.json")"
  [ "$a" = "$b" ]
  mk '.disk.device = "/dev/vdb"'
  c="$(main hash --answers "$W/a.json")"
  [ "$a" != "$c" ]
}

# ── el plan (simulacro) ──────────────────────────────────────────────

@test "el plan es texto determinista" {
  mk
  main plan --answers "$W/a.json" > "$W/p1" 2> /dev/null
  main plan --answers "$W/a.json" > "$W/p2" 2> /dev/null
  cmp "$W/p1" "$W/p2"
  grep -q '^DRYRUN: plan [0-9a-f]\{64\}$' "$W/p1"
}

@test "un simulacro no escribe nada: ni estado, ni registro, ni eventos" {
  mk
  run main plan --answers "$W/a.json"
  [ "$status" = 0 ]
  [ ! -e "$W/state" ]
  [ ! -e "$W/install.log" ]
}

@test "plan, disco entero con btrfs: borra, particiona, formatea y crea subvolúmenes" {
  mk
  run main plan --answers "$W/a.json"
  [[ "$output" == *"DRYRUN: wipefs --all --force /dev/vda"* ]]
  [[ "$output" == *'name="MAXOR-ESP"'* ]]
  [[ "$output" == *"mkfs.fat -F 32 -n MAXOR-ESP /dev/vda1"* ]]
  [[ "$output" == *"mkfs.btrfs -f -L maxor-root /dev/vda2"* ]]
  [[ "$output" == *"btrfs subvolume create $W/mnt/@home"* ]]
  [[ "$output" == *"nixos-install --root $W/mnt --flake $W/mnt/home/ana/nixos-config#maxor --no-root-passwd"* ]]
}

@test "plan con ext4 no usa btrfs" {
  mk '.disk.filesystem = "ext4"'
  run main plan --answers "$W/a.json"
  [[ "$output" == *"mkfs.ext4 -F -L maxor-root /dev/vda2"* ]]
  [[ "$output" != *btrfs* ]]
}

@test "plan con cifrado: LUKS2 con la contraseña por la entrada estándar y la raíz en el mapper" {
  mk '.disk.encrypt.enabled = true'
  run main plan --answers "$W/a.json"
  [[ "$output" == *"cryptsetup luksFormat --type luks2 --batch-mode --key-file=- /dev/vda2 (secret on stdin)"* ]]
  [[ "$output" == *"cryptsetup open --key-file=- /dev/vda2 maxor-root (secret on stdin)"* ]]
  [[ "$output" == *"mkfs.btrfs -f -L maxor-root /dev/mapper/maxor-root"* ]]
  [[ "$output" == *"cryptsetup close maxor-root"* ]]
}

@test "plan sin cifrado se salta LUKS" {
  mk
  run main plan --answers "$W/a.json"
  [[ "$output" != *cryptsetup* ]]
  [[ "$output" == *"[luks] skip"* ]]
}

@test "plan con swap en archivo: btrfs usa su subvolumen y mkswapfile, ext4 un archivo normal" {
  mk '.disk.swap = {kind: "file", gib: 8}'
  run main plan --answers "$W/a.json"
  [[ "$output" == *"btrfs subvolume create $W/mnt/@swap"* ]]
  [[ "$output" == *"btrfs filesystem mkswapfile --size 8g $W/mnt/swap/swapfile"* ]]
  mk '.disk.swap = {kind: "file", gib: 4} | .disk.filesystem = "ext4"'
  run main plan --answers "$W/a.json"
  [[ "$output" == *"mkswap $W/mnt/swapfile"* ]]
  [[ "$output" == *"count=4096"* ]]
}

@test "plan con swap zram o ninguno no crea archivos de swap" {
  for k in zram none; do
    mk ".disk.swap = {kind: \"$k\", gib: 0}"
    run main plan --answers "$W/a.json"
    [[ "$output" != *mkswap* ]]
    [[ "$output" != *swapfile* ]]
  done
}

@test "plan en un disco nvme usa las particiones con p" {
  mk '.disk.device = "/dev/nvme0n1"'
  run main plan --answers "$W/a.json"
  [[ "$output" == *"mkfs.fat -F 32 -n MAXOR-ESP /dev/nvme0n1p1"* ]]
  [[ "$output" == *"mkfs.btrfs -f -L maxor-root /dev/nvme0n1p2"* ]]
}

@test "plan: la contraseña del usuario no aparece, ni por argumentos ni en el texto" {
  mk
  run main plan --answers "$W/a.json"
  [[ "$output" != *'$6$salt1234'* ]]
  [[ "$output" == *"-c 'chpasswd --encrypted' (secret on stdin)"* ]]
}

@test "plan: los perfiles y el usuario llegan a la configuración que se escribe" {
  mk '.look.profiles = ["dev", "gaming"] | .user.autologin = true'
  run main plan --answers "$W/a.json"
  [[ "$output" == *'"profiles": ['* ]]
  [[ "$output" == *'"gaming"'* ]]
  [[ "$output" == *"initial_session"* ]]
}

# ── discos: huecos libres y sondeo ───────────────────────────────────

@test "los huecos libres se calculan alineados y sin contar lo pequeño" {
  # un disco de 100 GiB con Windows en las primeras particiones y 50 GiB libres al final
  cat > "$W/sfdisk.json" << 'EOF'
{"partitiontable":{"label":"gpt","firstlba":34,"lastlba":209715166,"sectorsize":512,"partitions":[
 {"node":"/dev/vda1","start":2048,"size":204800},
 {"node":"/dev/vda2","start":206848,"size":104857600}]}}
EOF
  shim sfdisk "cat '$W/sfdisk.json'"
  run disk_free_json /dev/vda
  [ "$status" = 0 ]
  [ "$(jq -r 'length' <<< "$output")" = 1 ]
  [ "$(jq -r '.[0].start' <<< "$output")" = 105064448 ]
  [ "$(jq -r '.[0].end' <<< "$output")" = 209715166 ]
}

@test "un disco sin tabla de particiones es un solo hueco" {
  shim sfdisk 'exit 1'
  shim blockdev 'echo 209715200'
  run disk_free_json /dev/vda
  [ "$(jq -r '.[0].start' <<< "$output")" = 2048 ]
}

@test "el hueco elegido sigue libre solo si cabe entero en un hueco actual" {
  cat > "$W/sfdisk.json" << 'EOF'
{"partitiontable":{"label":"gpt","firstlba":34,"lastlba":209715166,"sectorsize":512,"partitions":[
 {"node":"/dev/vda1","start":2048,"size":104857600}]}}
EOF
  shim sfdisk "cat '$W/sfdisk.json'"
  run disk_region_still_free /dev/vda 104861696 209715166
  [ "$status" = 0 ]
  run disk_region_still_free /dev/vda 100000000 209715166   # pisa la partición existente
  [ "$status" != 0 ]
}

@test "el sondeo de discos enseña qué hay, si está montado y si parece Windows" {
  cat > "$W/lsblk.json" << 'EOF'
{"blockdevices":[
 {"name":"vda","path":"/dev/vda","size":107374182400,"type":"disk","model":"QEMU HARDDISK ","tran":null,"rm":false,"ro":false,"fstype":null,"label":null,"partlabel":null,"mountpoints":[null],
  "children":[
   {"name":"vda1","path":"/dev/vda1","size":104857600,"type":"part","fstype":"vfat","label":"SYSTEM","partlabel":"EFI","mountpoints":[null]},
   {"name":"vda2","path":"/dev/vda2","size":53687091200,"type":"part","fstype":"ntfs","label":"Windows","partlabel":"Basic data","mountpoints":[null]}]},
 {"name":"sda","path":"/dev/sda","size":16000000000,"type":"disk","model":"USB","tran":"usb","rm":true,"ro":false,"mountpoints":["/iso"],"children":[]}
]}
EOF
  cat > "$W/sfdisk.json" << 'EOF'
{"partitiontable":{"label":"gpt","firstlba":34,"lastlba":209715166,"sectorsize":512,"partitions":[
 {"node":"/dev/vda1","start":2048,"size":204800},{"node":"/dev/vda2","start":206848,"size":104857600}]}}
EOF
  shim lsblk "cat '$W/lsblk.json'"
  shim sfdisk "cat '$W/sfdisk.json'"
  shim blockdev 'echo 31250000'
  run disk_probe_json
  [ "$status" = 0 ]
  [ "$(jq -r 'length' <<< "$output")" = 2 ]
  [ "$(jq -r '.[0].windows' <<< "$output")" = true ]
  [ "$(jq -r '.[0].mounted' <<< "$output")" = false ]
  [ "$(jq -r '.[0].model' <<< "$output")" = "QEMU HARDDISK" ]
  [ "$(jq -r '.[0].free[0].start' <<< "$output")" = 105064448 ]
  [ "$(jq -r '.[1].removable' <<< "$output")" = true ]
  [ "$(jq -r '.[1].mounted' <<< "$output")" = true ]
}

# ── preflight ────────────────────────────────────────────────────────

# Un equipo en el que todo está bien.
preflight_ok() {
  local t
  for t in "${PF_NEED_TOOLS[@]}" mkfs.btrfs mkfs.ext4 cryptsetup; do
    case "$t" in
      jq | git) ;; # los de verdad: el propio motor los usa
      *) shim "$t" ;;
    esac
  done
  shim id 'echo 0'
  shim blockdev 'echo 107374182400'
  shim findmnt 'exit 1'
  shim lsblk 'exit 0'
  shim curl 'exit 0'
  in_is_block() { return 0; }
  mk
  ans_load "$W/a.json"
}

@test "preflight: un equipo en regla pasa" {
  preflight_ok
  run stage_preflight
  [ "$status" = 0 ]
}

@test "preflight reúne todos los motivos, no solo el primero" {
  preflight_ok
  shim id 'echo 1000'
  shim blockdev 'echo 10737418240'
  shim curl 'exit 6'
  rmdir "$W/efi"
  printf 'MemTotal:       900000 kB\n' > "$W/meminfo"
  run stage_preflight
  [ "$status" = "$IN_EX_PREFLIGHT" ]
  [[ "$output" == *"must run as root"* ]]
  [[ "$output" == *"did not boot in UEFI mode"* ]]
  [[ "$output" == *"needs at least 32 GiB"* ]]
  [[ "$output" == *"no network"* ]]
  [[ "$output" == *"less than 1.5 GiB of RAM"* ]]
}

@test "preflight se niega a instalar sobre el medio del que arrancó la ISO" {
  preflight_ok
  shim findmnt 'echo /dev/vda1'
  shim lsblk 'case "$*" in *PKNAME*) echo vda ;; esac'
  run stage_preflight
  [ "$status" = "$IN_EX_PREFLIGHT" ]
  [[ "$output" == *"medium this live system booted from"* ]]
}

@test "preflight se niega si algo del disco está montado" {
  preflight_ok
  shim lsblk 'case "$*" in *MOUNTPOINTS*) echo /run/media/usb ;; esac'
  run stage_preflight
  [ "$status" = "$IN_EX_PREFLIGHT" ]
  [[ "$output" == *"is mounted"* ]]
}

@test "preflight se niega con Secure Boot activo, salvo que se permita" {
  preflight_ok
  mkdir -p "$W/efi/efivars"
  printf '\x07\x00\x00\x00\x01' > "$W/efi/efivars/SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c"
  run stage_preflight
  [ "$status" = "$IN_EX_PREFLIGHT" ]
  [[ "$output" == *"Secure Boot is on"* ]]
  mk '.advanced.allow_secureboot = true'
  ans_load "$W/a.json"
  run stage_preflight
  [ "$status" = 0 ]
}

@test "preflight con cifrado exige una contraseña de al menos 8 caracteres" {
  preflight_ok
  mk '.disk.encrypt.enabled = true'
  ans_load "$W/a.json"
  IN_SECRET="corta"
  run stage_preflight
  [ "$status" = "$IN_EX_PREFLIGHT" ]
  [[ "$output" == *"passphrase is missing or shorter than 8"* ]]
  IN_SECRET="una contraseña larga"
  run stage_preflight
  [ "$status" = 0 ]
}

@test "preflight junto a otro sistema se niega si hay restos de una instalación anterior de Maxor" {
  preflight_ok
  mk '.disk.strategy = "alongside" | .disk.region = {start: 4000000, end: 200000000} | .disk.confirmed = "INSTALL" | .disk.plan_hash = ("a" * 64)'
  h="$(main hash --answers "$W/a.json")"
  mk ".disk.strategy = \"alongside\" | .disk.region = {start: 4000000, end: 200000000} | .disk.confirmed = \"INSTALL\" | .disk.plan_hash = \"$h\""
  ans_load "$W/a.json"
  shim lsblk 'case "$*" in *PARTLABEL*) echo maxor-root ;; esac'
  run stage_preflight
  [ "$status" = "$IN_EX_PREFLIGHT" ]
  [[ "$output" == *"previous Maxor install"* ]]
}

# ── etapas reales (con shims) ────────────────────────────────────────

@test "disco entero: borra firmas, escribe la tabla GPT y anota los nodos de las particiones" {
  mk
  ans_load "$W/a.json"
  in_is_block() { return 0; }
  shim wipefs
  shim udevadm
  shim partprobe
  shim sfdisk 'case "$*" in *-J*) echo "{\"partitiontable\":{\"partitions\":[{\"node\":\"/dev/vda1\",\"name\":\"MAXOR-ESP\"},{\"node\":\"/dev/vda2\",\"name\":\"maxor-root\"}]}}" ;; esac' stdin
  mkdir -p "$W/state"
  run stage_disk
  [ "$status" = 0 ]
  [[ "$(calls)" == *"wipefs --all --force /dev/vda"* ]]
  grep -q 'label: gpt' "$W/stdin.sfdisk"
  grep -q 'name="maxor-root"' "$W/stdin.sfdisk"
  [ "$(jq -r .esp "$W/state/devices.json")" = /dev/vda1 ]
  [ "$(jq -r .root "$W/state/devices.json")" = /dev/vda2 ]
}

@test "junto a otro sistema: si el hueco ya no está libre, no se toca nada" {
  mk '.disk.strategy = "alongside" | .disk.region = {start: 4000000, end: 200000000} | .disk.confirmed = "INSTALL" | .disk.plan_hash = ("a" * 64)'
  h="$(main hash --answers "$W/a.json")"
  mk ".disk.strategy = \"alongside\" | .disk.region = {start: 4000000, end: 200000000} | .disk.confirmed = \"INSTALL\" | .disk.plan_hash = \"$h\""
  ans_load "$W/a.json"
  in_is_block() { return 0; }
  shim sfdisk 'echo "{\"partitiontable\":{\"firstlba\":34,\"lastlba\":209715166,\"partitions\":[{\"node\":\"/dev/vda1\",\"start\":2048,\"size\":204000000}]}}"' stdin
  shim wipefs
  mkdir -p "$W/state"
  run stage_disk
  [ "$status" != 0 ]
  [[ "$output" == *"no longer free"* ]]
  [[ "$(calls)" != *"--append"* ]]
  [[ "$(calls)" != *wipefs* ]]
}

@test "cifrado real: la contraseña llega a cryptsetup por la entrada estándar y a ningún otro sitio" {
  mk '.disk.encrypt.enabled = true'
  ans_load "$W/a.json"
  IN_SECRET="correct horse battery staple"
  shim cryptsetup '' stdin
  mkdir -p "$W/state"
  stage_luks
  grep -q 'correct horse battery staple' "$W/stdin.cryptsetup"
  [[ "$(calls)" != *"correct horse"* ]]
  ! grep -q 'correct horse' "$W/install.log" 2> /dev/null
  [[ "$(calls)" == *"cryptsetup luksFormat --type luks2 --batch-mode --key-file=- "* ]]
}

@test "finish: la contraseña del usuario entra por la entrada estándar de chpasswd, no por argumentos" {
  mk
  ans_load "$W/a.json"
  shim nixos-enter '' stdin
  shim umount
  mkdir -p "$W/mnt"
  stage_finish
  grep -q '^ana:\$6\$salt1234' "$W/stdin.nixos-enter"
  [[ "$(calls)" != *'salt1234'* ]]
  [[ "$(calls)" == *"nixos-enter --root $W/mnt -c chpasswd --encrypted"* ]]
  [[ "$(calls)" == *"chown -R ana:users /home/ana/nixos-config"* ]]
  [[ "$(calls)" == *"umount -R $W/mnt"* ]]
}

@test "host: escribe un flake válido con lo especial escapado, y lo deja en git" {
  mk '.user.fullname = "Ana \"la\" ${rara}" | .user.autologin = true | .look.profiles = ["dev"]'
  ans_load "$W/a.json"
  export MAXOR_BIN="$W/bin/maxor"
  shim maxor 'echo "{\"version\":1,\"cpu\":{\"vendor\":\"intel\"},\"gpus\":[],\"laptop\":false,\"virt\":\"none\",\"bluetooth\":false}"'
  IN_MAXOR="$W/bin/maxor"
  shim nixos-generate-config 'echo "{ ... }: { }"'
  shim nix 'touch "${@: -1}/flake.lock"'
  stage_host
  d="$W/mnt/home/ana/nixos-config"
  [ -f "$d/flake.nix" ] && [ -f "$d/host/boot.nix" ] && [ -f "$d/host/hardware.json" ] && [ -f "$d/host/hardware-configuration.nix" ]
  # el nombre con comillas y ${ no rompe la cadena de Nix
  grep -qF 'fullname = "Ana \"la\" \${rara}";' "$d/flake.nix"
  grep -q 'initial_session' "$d/flake.nix"
  grep -q 'nixosConfigurations.maxor = maxor-os.lib.mkSystem' "$d/flake.nix"
  [ "$(jq -r '.profiles[0]' "$d/host/maxor.json")" = dev ]
  # git: los archivos están commiteados (Nix solo ve lo que git conoce)
  [ -z "$(git -C "$d" status --porcelain)" ]
  [ "$(git -C "$d" log --oneline | wc -l)" -ge 1 ]
}

@test "nixstr escapa comillas, barras y \${" {
  [ "$(nixstr 'a"b')" = '"a\"b"' ]
  [ "$(nixstr 'a\b')" = '"a\\b"' ]
  [ "$(nixstr 'x${y}')" = '"x\${y}"' ]
}

# ── etapas, estado y reanudación ─────────────────────────────────────

stub_stages() {
  IN_STAGES=(alfa beta gamma)
  stage_alfa() { echo "alfa" >> "$W/ran"; }
  stage_beta() { echo "beta" >> "$W/ran"; }
  stage_gamma() { echo "gamma" >> "$W/ran"; }
}

@test "las etapas corren en orden, dejan su marca y emiten eventos JSON" {
  mk
  stub_stages
  run main run --answers "$W/a.json" --events "$W/ev.jsonl"
  [ "$status" = 0 ]
  [ "$(tr '\n' ' ' < "$W/ran")" = "alfa beta gamma " ]
  [ -e "$W/state/stages/alfa" ] && [ -e "$W/state/stages/gamma" ]
  jq -e -s 'all(.[]; has("stage") and has("state"))' "$W/ev.jsonl" > /dev/null
  [ "$(jq -r -s 'map(select(.stage == "done")) | .[0].state' "$W/ev.jsonl")" = ok ]
  # el progreso crece
  [ "$(jq -r -s '[.[] | select(has("progress")) | .progress] | (. == sort)' "$W/ev.jsonl")" = true ]
}

@test "una segunda ejecución sin --resume se niega a pisar el progreso anterior" {
  mk
  stub_stages
  main run --answers "$W/a.json" > /dev/null 2>&1
  stub_stages
  run main run --answers "$W/a.json"
  [ "$status" = "$IN_EX_USAGE" ]
  [[ "$output" == *"--resume"* ]]
}

@test "--resume salta lo ya hecho, y --reset lo olvida" {
  mk
  stub_stages
  stage_gamma() { echo "gamma-falla" >> "$W/ran"; return 1; }
  run main run --answers "$W/a.json"
  [ "$status" = "$IN_EX_FAIL" ]
  [ -e "$W/state/stages/alfa" ] && [ -e "$W/state/stages/beta" ] && [ ! -e "$W/state/stages/gamma" ]
  : > "$W/ran"
  stage_gamma() { echo "gamma" >> "$W/ran"; }
  run main run --answers "$W/a.json" --resume
  [ "$status" = 0 ]
  [ "$(tr '\n' ' ' < "$W/ran")" = "gamma " ]
  : > "$W/ran"
  run main run --answers "$W/a.json" --reset
  [ "$status" = 0 ]
  [ "$(tr '\n' ' ' < "$W/ran")" = "alfa beta gamma " ]
}

@test "un fallo para en esa etapa, dice cuál y no corre las siguientes" {
  mk
  stub_stages
  stage_beta() { echo "beta" >> "$W/ran"; false; }
  run main run --answers "$W/a.json" --events "$W/ev.jsonl"
  [ "$status" = "$IN_EX_FAIL" ]
  [ "$(tr '\n' ' ' < "$W/ran")" = "alfa beta " ]
  [ "$(jq -r -s 'map(select(.state == "fail")) | .[0].stage' "$W/ev.jsonl")" = beta ]
}

@test "dentro de una etapa, el primer comando que falla la para (errexit)" {
  # En un proceso de verdad: `bats run` ejecuta con `||`, y bash no aplica errexit dentro de eso.
  mk
  run bash -c "set -euo pipefail; export MAXOR_INSTALL_NO_MAIN=1
    for f in common answers preflight disk luks filesystem host nixinstall bootloader finish; do source $ROOT/installer/engine/lib/\$f.sh; done
    source $ROOT/installer/engine/main.sh
    IN_STAGES=(alfa)
    stage_alfa() { echo uno >> $W/ran; false; echo dos >> $W/ran; }
    main run --answers $W/a.json"
  [ "$status" = "$IN_EX_FAIL" ]
  [ "$(cat "$W/ran")" = uno ]
}

@test "una etapa que sale con 'respuestas inválidas' o 'equipo no apto' conserva ese código" {
  mk
  stub_stages
  stage_alfa() { exit "$IN_EX_PREFLIGHT"; }
  run main run --answers "$W/a.json"
  [ "$status" = "$IN_EX_PREFLIGHT" ]
}

@test "cifrado sin contraseña (--secret-fd) no se instala" {
  mk '.disk.encrypt.enabled = true'
  stub_stages
  run main run --answers "$W/a.json"
  [ "$status" = "$IN_EX_ANSWERS" ]
  [[ "$output" == *"no passphrase was given"* ]]
  [ ! -s "$W/ran" ]
}

@test "la contraseña de cifrado se lee del descriptor y no llega al registro ni a los eventos" {
  mk '.disk.encrypt.enabled = true'
  stub_stages
  stage_alfa() { printf '%s' "$IN_SECRET" > "$W/secret.seen"; }
  exec 7<<< "una contraseña larga"
  run main run --answers "$W/a.json" --secret-fd 7 --events "$W/ev.jsonl"
  exec 7<&-
  [ "$status" = 0 ]
  [ "$(cat "$W/secret.seen")" = "una contraseña larga" ]
  ! grep -rq 'contraseña larga' "$W/install.log" "$W/ev.jsonl"
}

@test "un comando desconocido o sin --answers es un error de uso" {
  run main nada
  [ "$status" = "$IN_EX_USAGE" ]
  run main run
  [ "$status" = "$IN_EX_USAGE" ]
  run main validate
  [ "$status" = "$IN_EX_USAGE" ]
}
