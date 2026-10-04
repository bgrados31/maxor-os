# shellcheck shell=bash
# Ayudantes comunes de las pruebas de la CLI.
#
# Las pruebas «unitarias» cargan las bibliotecas de home/maxor en el propio
# proceso (sin main.sh) para llamar a cada función. Las de integración usan el
# binario completo que indica MAXOR_BIN (lo da `nix flake check`).

ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
SRC="$ROOT/home/maxor"

# Entorno aislado: ningún archivo del usuario real se toca ni se lee.
isolate() {
  export HOME="$BATS_TEST_TMPDIR/home"
  export XDG_CONFIG_HOME="$HOME/.config" XDG_DATA_HOME="$HOME/.local/share" XDG_STATE_HOME="$HOME/.local/state"
  export MAXOR_FLAKE="$BATS_TEST_TMPDIR/flake" MAXOR_HOST="testhost"
  export NO_COLOR='' TERM=xterm
  mkdir -p "$HOME" "$MAXOR_FLAKE"
  unset MAXOR_LANG MAXOR_QUIET MAXOR_VERBOSE
}

# load_lib [color]  → carga las bibliotecas; con «color» la interfaz dibuja ventanas.
load_lib() {
  isolate
  if [ "${1:-}" = color ]; then export MAXOR_FORCE_UI=1 COLUMNS=60; else unset MAXOR_FORCE_UI; fi
  local f
  for f in lib/core.sh lib/i18n.sh lib/lang/en.sh lib/term.sh lib/frame.sh lib/loaders.sh \
    lib/widgets.sh lib/style.sh lib/registry.sh; do
    # shellcheck disable=SC1090
    source "$SRC/$f"
  done
  for f in "$SRC"/cmd/*.sh; do
    # shellcheck disable=SC1090
    source "$f"
  done
  i18n_init
  log_init
}

# strip_ansi texto → sin secuencias de color
strip_ansi() { sed 's/\x1b\[[0-9;?]*[a-zA-Z]//g'; }

# load_engine → carga las bibliotecas del motor del instalador (installer/engine) en el propio proceso,
# con estado, registro y destino dentro del directorio temporal de la prueba. Sin `main`.
load_engine() {
  isolate
  W="$BATS_TEST_TMPDIR"
  export MAXOR_INSTALL_NO_MAIN=1 MAXOR_INSTALL_STATE="$W/state" MAXOR_INSTALL_LOG="$W/install.log" MAXOR_INSTALL_ROOT="$W/mnt"
  export MAXOR_INSTALL_SCHEMA="$ROOT/installer/schema/answers.v1.json" MAXOR_PROFILES="$ROOT/modules/profiles-catalog.json"
  export MAXOR_INSTALL_EFI="$W/efi" MAXOR_INSTALL_MEMINFO="$W/meminfo" MAXOR_INSTALL_NM_DIR="$W/nm" MAXOR_INSTALL_OVERRIDES_FILE="$W/overrides"
  mkdir -p "$W/bin" "$W/efi"
  printf 'MemTotal:       4000000 kB\n' > "$W/meminfo"
  PATH="$W/bin:$PATH"
  local f
  for f in lib/common.sh lib/answers.sh lib/preflight.sh lib/disk.sh lib/luks.sh lib/filesystem.sh lib/host.sh lib/nixinstall.sh lib/bootloader.sh lib/finish.sh main.sh; do
    # shellcheck disable=SC1090
    source "$ROOT/installer/engine/$f"
  done
}

# shim NOMBRE [CUERPO] [stdin] → un programa de mentira en $W/bin que anota cómo lo llamaron
# ($W/calls). Con «stdin» también guarda lo que recibe por la entrada estándar en $W/stdin.NOMBRE
# (solo si se pide: leer una entrada que nadie cierra bloquearía). Sin CUERPO imprime $W/out.NOMBRE.
shim() {
  local name="$1" body="${2:-}" reads="${3:-}"
  {
    printf '#!%s\n' "$(command -v bash)"
    printf 'printf "%%s %%s\\n" "%s" "$*" >> "%s/calls"\n' "$name" "$W"
    if [ "$reads" = stdin ]; then printf 'cat >> "%s/stdin.%s"\n' "$W" "$name"; fi
    if [ -n "$body" ]; then printf '%s\n' "$body"; else printf '[ -f "%s/out.%s" ] && cat "%s/out.%s"\nexit 0\n' "$W" "$name" "$W" "$name"; fi
  } > "$W/bin/$name"
  chmod +x "$W/bin/$name"
}

# base_answers → unas respuestas válidas (disco entero, btrfs); mk 'FILTRO jq' las modifica y las guarda.
base_answers() {
  jq -nc '{schema: 1, timezone: "America/Lima", keymap: "la-latin1", xkb: {layout: "latam"},
    disk: {device: "/dev/vda", strategy: "whole", confirmed: "ERASE"}, machine: {hostname: "maxor"},
    user: {name: "ana", fullname: "Ana", password_hash: "$6$salt1234$abcdefghijklmnopqrstuvwxyz"}, look: {profiles: ["dev"]}}'
}
mk() { base_answers | jq -c "${1:-.}" > "$W/a.json"; }
