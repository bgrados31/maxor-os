# ── Núcleo del motor de instalación ──────────────────────────────────
# Lo que comparten todas las etapas: registro, eventos de progreso, el envoltorio que hace
# posible el simulacro (`in_run`: o ejecuta o solo imprime) y utilidades de discos.
#
# Regla de oro: TODA orden que cambia el sistema pasa por `in_run`, `in_run_secret`, `in_run_stdin` o `write_file`.
# Con IN_DRY=1 esas cuatro solo imprimen lo que harían, así el plan de una instalación es un texto
# determinista que se puede comprobar sin tocar ningún disco.
IN_STATE="${MAXOR_INSTALL_STATE:-/run/maxor-install}"
IN_LOG="${MAXOR_INSTALL_LOG:-/var/log/maxor-install.log}"
IN_ROOT="${MAXOR_INSTALL_ROOT:-/mnt}"
IN_SCHEMA="${MAXOR_INSTALL_SCHEMA:-}"
IN_DRY=0
IN_EVENTS=""   # archivo donde se escriben los eventos JSON (uno por línea)
IN_ANSWERS=""  # las respuestas ya normalizadas
IN_STAGE=""    # etapa en curso, para los mensajes de error
IN_SECRET=""   # contraseña de LUKS (llega por un descriptor; nunca a disco ni al registro)

# Códigos de salida: lo que hace un script o una pantalla con el motor.
IN_EX_FAIL=1       # una etapa falló
IN_EX_USAGE=2      # uso incorrecto
IN_EX_ANSWERS=3    # las respuestas no son válidas
IN_EX_PREFLIGHT=4  # el equipo no cumple los requisitos
IN_EX_CANCEL=130

# in_log NIVEL texto… → al registro (un simulacro no escribe nada, ni siquiera el registro)
in_log() {
  [ "$IN_DRY" = 1 ] && return 0
  local line
  printf -v line '%(%FT%T)T %-5s %s' -1 "$1" "${*:2}"
  { mkdir -p "$(dirname "$IN_LOG")" && printf '%s\n' "$line" >> "$IN_LOG"; } 2> /dev/null || true
}

# in_emit etapa estado mensaje [progreso 0..1] → un evento JSON y una línea legible.
# estado: start | ok | skip | fail | info
in_emit() {
  local line
  line="$(jq -nc --arg stage "$1" --arg state "$2" --arg message "${3:-}" --arg p "${4:-}" \
    '{stage: $stage, state: $state, message: $message} + (if $p == "" then {} else {progress: ($p | tonumber)} end)')"
  if [ -n "$IN_EVENTS" ] && [ "$IN_DRY" != 1 ]; then printf '%s\n' "$line" >> "$IN_EVENTS"; fi
  in_log INFO "$1 $2 ${3:-}"
  printf '· [%s] %s%s\n' "$1" "$2" "${3:+ — $3}" >&2
}

in_die() { # in_die código mensaje…
  local code="$1"
  shift
  in_emit "${IN_STAGE:-engine}" fail "$*"
  printf '✗ %s\n' "$*" >&2
  exit "$code"
}

# in_q ARG… → los argumentos para mostrarlos: tal cual si son simples, con comillas si no.
in_q() {
  local a
  for a in "$@"; do
    if [[ "$a" =~ ^[A-Za-z0-9_@%+=:,./~-][A-Za-z0-9_@%+=:,./~#-]*$ ]]; then
      printf ' %s' "$a"
    elif [[ "$a" != *"'"* ]]; then
      printf " '%s'" "$a"
    else
      printf ' %q' "$a"
    fi
  done
}

# in_run orden args… → ejecuta; en simulacro imprime `DRYRUN: orden args` y no hace nada.
in_run() {
  if [ "$IN_DRY" = 1 ]; then
    printf 'DRYRUN:'
    in_q "$@"
    printf '\n'
    return 0
  fi
  in_log CMD "$*"
  "$@"
}

# in_run_secret SECRETO orden args… → la orden lee SECRETO por la entrada estándar. El secreto nunca se
# imprime, ni en el simulacro, ni en el registro, ni en la lista de procesos.
in_run_secret() {
  local secret="$1"
  shift
  if [ "$IN_DRY" = 1 ]; then
    printf 'DRYRUN:'
    in_q "$@"
    printf ' (secret on stdin)\n'
    return 0
  fi
  in_log CMD "$* (secret on stdin)"
  printf '%s' "$secret" | "$@"
}

# write_file ruta modo  (el contenido por la entrada estándar)
write_file() {
  local path="$1" mode="$2" content
  content="$(cat)"
  if [ "$IN_DRY" = 1 ]; then
    printf 'DRYRUN: write %s (mode %s)\n' "$path" "$mode"
    printf '%s\n' "$content" | sed 's/^/DRYRUN:   | /'
    return 0
  fi
  in_log CMD "write $path"
  mkdir -p "$(dirname "$path")"
  printf '%s\n' "$content" > "$path"
  chmod "$mode" "$path"
}

# part_dev /dev/nvme0n1 2 → /dev/nvme0n1p2 ; part_dev /dev/sda 2 → /dev/sda2
part_dev() {
  case "$1" in
    *[0-9]) printf '%sp%s' "$1" "$2" ;;
    *) printf '%s%s' "$1" "$2" ;;
  esac
}

# Sectores de 512 bytes en n GiB / MiB.
gib_sectors() { printf '%d' $(($1 * 2097152)); }
mib_sectors() { printf '%d' $(($1 * 2048)); }

# ans '.ruta' → un valor de las respuestas (cadena vacía si no existe)
ans() { jq -r "$1 // empty" "$IN_ANSWERS"; }
ans_true() { [ "$(jq -r "$1 // false" "$IN_ANSWERS")" = true ]; }

# in_run_stdin ARCHIVO orden args… → como run, con el archivo como entrada estándar de la orden.
in_run_stdin() {
  local file="$1"
  shift
  if [ "$IN_DRY" = 1 ]; then
    printf 'DRYRUN:'
    in_q "$@"
    printf ' < %s\n' "$file"
    return 0
  fi
  in_log CMD "$* < $file"
  "$@" < "$file"
}

# Dispositivos del destino. Las etapas corren en subprocesos, así que lo que la etapa del disco
# descubre (los nodos de las particiones nuevas) se guarda en $IN_STATE/devices.json y se lee de ahí.
# En un simulacro se calculan los previsibles.
dev_get() { # dev_get esp|root
  local f="$IN_STATE/devices.json" disk
  if [ -r "$f" ]; then
    jq -r --arg k "$1" '.[$k]' "$f"
    return 0
  fi
  disk="$(ans .disk.device)"
  if [ "$(ans .disk.strategy)" = whole ]; then
    if [ "$1" = esp ]; then part_dev "$disk" 1; else part_dev "$disk" 2; fi
  else
    printf '%s<new-%s-partition>' "$disk" "$1"
  fi
}
# El dispositivo que lleva el sistema de archivos raíz: el cifrado, si lo hay.
dev_rootfs() {
  if ans_true '.disk.encrypt.enabled'; then printf '/dev/mapper/maxor-root'; else dev_get root; fi
}

# ¿Es un dispositivo de bloque? (las pruebas lo sustituyen: no hay discos de verdad en el sandbox)
in_is_block() { [ -b "$1" ]; }
