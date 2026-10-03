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
