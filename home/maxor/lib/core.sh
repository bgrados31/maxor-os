# ── Núcleo: rutas, errores, registro y flake ─────────────────────────
# El locale original se guarda antes de forzar C.UTF-8: term.sh lo usa para decidir los glifos.
MAXOR_ORIG_LOCALE="${LC_ALL:-${LC_CTYPE:-${LANG:-}}}"
# C.UTF-8 existe siempre en glibc: así ${#var} cuenta caracteres, no bytes.
export LC_ALL=C.UTF-8

MAXOR_VERSION="${MAXOR_VERSION:-dev}"

data="${XDG_DATA_HOME:-$HOME/.local/share}/maxor"
cfg="${XDG_CONFIG_HOME:-$HOME/.config}/maxor"
state="${XDG_STATE_HOME:-$HOME/.local/state}/maxor"
themes="$data/themes"
tpl="$data/templates"
dms_settings="${XDG_CONFIG_HOME:-$HOME/.config}/DankMaterialShell/settings.json"

# ── Colores ──────────────────────────────────────────────────────────
hex() { printf '%s' "${1#\#}"; }
rgb() { local h; h="$(hex "$1")"; printf '%d, %d, %d' "0x${h:0:2}" "0x${h:2:2}" "0x${h:4:2}"; }
# mix A B P  →  A con P % de B
mix() {
  local a b p out="#" i x y
  a="$(hex "$1")"; b="$(hex "$2")"; p="$3"
  for i in 0 2 4; do
    x=$((0x${a:i:2})); y=$((0x${b:i:2}))
    out+="$(printf '%02x' $(((x * (100 - p) + y * p) / 100)))"
  done
  printf '%s' "$out"
}

# ── Códigos de salida ────────────────────────────────────────────────
# Los scripts y las apps reaccionan al código, no al texto del error.
EX_FAIL=1     # falló la operación
EX_USAGE=2    # uso incorrecto: opción o argumento inválido
EX_NEEDS=3    # falta algo: archivo, programa o configuración
EX_NET=4      # sin red o servicio remoto caído
EX_PERM=5     # permisos insuficientes
EX_CANCEL=130 # el usuario canceló

# ── Registro ─────────────────────────────────────────────────────────
# Todo se anota en ~/.local/state/maxor/logs/maxor.log (rota a 512 KB).
# `maxor logs` lo muestra y `maxor debug` lo incluye en el informe.
logdir="$state/logs"
logfile="$logdir/maxor.log"

log_init() {
  mkdir -p "$logdir" 2> /dev/null || return 0
  if [ -f "$logfile" ] && [ "$(stat -c %s "$logfile" 2> /dev/null || echo 0)" -gt 524288 ]; then
    mv -f "$logfile" "$logfile.1"
  fi
  return 0
}
log() { # log NIVEL texto…
  if [ "${MAXOR_VERBOSE:-0}" = 1 ]; then printf '%s %s\n' "$1" "${*:2}" >&2; fi
  [ -d "$logdir" ] || return 0
  printf '%(%FT%T)T %-5s %s\n' -1 "$1" "${*:2}" >> "$logfile" 2> /dev/null || true
}

# ── Errores ──────────────────────────────────────────────────────────
die_code() { # die_code CÓDIGO mensaje|@clave [args…]
  local c="$1" m
  shift
  msg m "$@"
  log ERROR "$m"
  if [ "${UI_RAIL:-0}" = 1 ]; then
    # dentro del riel: se cierra con el error
    printf '%s%s%s\n%s%s%s  %s%s%s %s\n' "$E_MU" "$G_BAR" "$E_RST" "$E_AC" "$G_END" "$E_RST" "$E_BAD" "$G_BAD" "$E_RST" "$m" >&2
  else
    printf '%s%s%s maxor: %s\n' "$E_BAD" "$G_BAD" "$E_RST" "$m" >&2
  fi
  exit "$c"
}
die() { die_code "$EX_FAIL" "$@"; } # die mensaje|@clave [args…]

# ── Flake y equipo ───────────────────────────────────────────────────
flake_dir="${MAXOR_FLAKE:-$HOME/nixos-config}"
host="${MAXOR_HOST:-$(< /proc/sys/kernel/hostname)}"

# sudo para lo que necesita root. La pantalla completa pide la contraseña en su propio
# campo y la entrega por la entrada estándar (MAXOR_SUDO_STDIN=1), así no hay que ceder
# la terminal; en una terminal normal es el sudo de siempre.
maxor_sudo() {
  if [ "${MAXOR_SUDO_STDIN:-0}" = 1 ]; then
    sudo -S -p '' "$@"
  else
    sudo "$@"
  fi
}

need_flake() {
  [ -f "$flake_dir/flake.nix" ] || die_code "$EX_NEEDS" @core.no_flake "$flake_dir"
}

# Nombre del sistema (PRETTY_NAME de /etc/os-release). Sin procesos y sin fallar
# si el archivo no existe (contenedores, pruebas): entonces, «Maxor OS».
os_pretty() {
  local l
  if [ -r /etc/os-release ]; then
    while IFS= read -r l; do
      case "$l" in
        PRETTY_NAME=*)
          l="${l#PRETTY_NAME=}"
          printf '%s' "${l//\"/}"
          return 0
          ;;
      esac
    done < /etc/os-release
  fi
  printf 'Maxor OS'
}
