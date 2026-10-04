# ── Etapa host: escribir la configuración de la máquina nueva ────────
# La máquina recibe un flake propio, en ~/nixos-config y como repositorio git, que solo dice lo
# que es particular de ella (maxor.machine, su hardware, su arranque) y toma de Maxor OS todo lo
# demás como entrada de flake. Actualizar es después subir esa entrada (`maxor release apply`).
IN_OS_URL="${MAXOR_INSTALL_OS_URL:-github:bgrados31/maxor-os}"
IN_MAXOR="${MAXOR_BIN:-maxor}"
IN_OVERRIDES_FILE="${MAXOR_INSTALL_OVERRIDES_FILE:-/etc/maxor-install/overrides}"
IN_LOCKS_FILE="${MAXOR_INSTALL_LOCKS_FILE:-/etc/maxor-install/locks.json}"

# nixstr TEXTO → TEXTO como cadena de Nix, entre comillas y con lo especial escapado.
nixstr() { jq -nr --arg v "$1" '$v | @json | gsub("\\$\\{"; "\\${")'; }

host_dir() { printf '%s/home/%s/nixos-config' "$IN_ROOT" "$(ans .user.name)"; }

host_flake_nix() {
  local host user full tz loc km layout variant name autologin=""
  name="$(ans .machine.hostname)"
  host="$(nixstr "$name")"
  user="$(nixstr "$(ans .user.name)")"
  full="$(nixstr "$(ans .user.fullname)")"
  tz="$(nixstr "$(ans .timezone)")"
  loc="$(nixstr "$(ans .locale)")"
  km="$(nixstr "$(ans .keymap)")"
  layout="$(nixstr "$(ans .xkb.layout)")"
  variant="$(nixstr "$(ans .xkb.variant)")"
  gpu="$(ans .hardware.gpu)" # an enum in the schema: safe to write as it is
  if ans_true '.user.autologin'; then
    autologin="
        ({ config, ... }: {
          # Entra sola al arrancar; al cerrar sesión aparece el login.
          services.greetd.settings.initial_session = {
            command = \"\${config.programs.hyprland.package}/bin/start-hyprland\";
            user = $user;
          };
        })"
  fi
  cat << __END_FLAKE__
{
  description = "Maxor OS on $name";

  # De dónde viene Maxor OS. \`maxor release apply\` sube esta versión con una release firmada.
  inputs.maxor-os.url = "$IN_OS_URL";

  outputs = { maxor-os, ... }: {
    nixosConfigurations.$name = maxor-os.lib.mkSystem {
      machine = {
        hostname = $host;
        user = $user;
        fullname = $full;
        timezone = $tz;
        locale = $loc;
        keymap = $km;
        xkb = { layout = $layout; variant = $variant; };
      };
      modules = [
        ./host/hardware-configuration.nix
        ./host/boot.nix
        ./host/local.nix
        ({ ... }: {
          maxor.hardware.report = ./host/hardware.json;
          maxor.hardware.gpu.mode = "$gpu";
          maxor.settings = ./host/maxor.json;
        })$autologin
      ];
    };
  };
}
__END_FLAKE__
}

host_local_nix() {
  cat << '__END_LOCAL__'
{ ... }:

# Your own NixOS options, on top of Maxor OS. This file is yours: the system never overwrites it.
# For example:  services.openssh.enable = true;
{
}
__END_LOCAL__
}

host_boot_nix() {
  local menu=""
  # Junto a otro sistema, el menú se queda a la vista para elegirlo; con Maxor OS solo, arranca directo (el menú
  # sale manteniendo pulsada una tecla al encender, ver modules/branding.nix).
  if [ "$(ans .disk.strategy)" = alongside ]; then
    menu=$'\n  # Hay otro sistema en este equipo: el menú de arranque se muestra para poder elegirlo.\n  boot.loader.timeout = 3;\n'
  fi
  cat << __END_BOOT__
{ ... }:

# El arranque de esta máquina. Los discos y, si hay cifrado, el desbloqueo de la raíz están en
# hardware-configuration.nix, que generó nixos-generate-config al instalar.
{
  boot.loader.systemd-boot.enable = true;
  boot.loader.systemd-boot.configurationLimit = 10;
  boot.loader.efi.canTouchEfiVariables = true;
${menu}
  system.stateVersion = "26.05"; # NO cambiar
}
__END_BOOT__
}

stage_host() {
  local dir prof
  dir="$(host_dir)"
  in_run mkdir -p "$dir/host"
  host_flake_nix | write_file "$dir/flake.nix" 644
  host_boot_nix | write_file "$dir/host/boot.nix" 644
  # host/local.nix es del usuario. Para instalaciones desatendidas, MAXOR_INSTALL_LOCAL_NIX trae su contenido.
  if [ -n "${MAXOR_INSTALL_LOCAL_NIX:-}" ]; then
    write_file "$dir/host/local.nix" 644 < "$MAXOR_INSTALL_LOCAL_NIX"
  else
    host_local_nix | write_file "$dir/host/local.nix" 644
  fi

  prof="$(jq '{profiles: .look.profiles, theme: (.look.theme // "maxor-dark")}' "$IN_ANSWERS")"
  printf '%s\n' "$prof" | write_file "$dir/host/maxor.json" 644

  # El hardware se detecta aquí, en la máquina que se instala. Para imágenes reproducibles o instalaciones sobre
  # hardware conocido se pueden dar ya hechos: MAXOR_INSTALL_HWJSON (lo que produce `maxor hardware detect`) y
  # MAXOR_INSTALL_HWCONFIG (lo que produce nixos-generate-config).
  if [ -n "${MAXOR_INSTALL_HWJSON:-}" ]; then
    in_run cp "$MAXOR_INSTALL_HWJSON" "$dir/host/hardware.json"
  elif [ "$IN_DRY" = 1 ]; then
    printf 'DRYRUN: %s hardware detect > %s\n' "$IN_MAXOR" "$dir/host/hardware.json"
  else
    "$IN_MAXOR" hardware detect > "$dir/host/hardware.json" || in_die "$IN_EX_FAIL" "could not detect the hardware"
  fi
  if [ -n "${MAXOR_INSTALL_HWCONFIG:-}" ]; then
    in_run cp "$MAXOR_INSTALL_HWCONFIG" "$dir/host/hardware-configuration.nix"
  elif [ "$IN_DRY" = 1 ]; then
    printf 'DRYRUN: nixos-generate-config --root %s --show-hardware-config > %s\n' "$IN_ROOT" "$dir/host/hardware-configuration.nix"
  else
    nixos-generate-config --root "$IN_ROOT" --show-hardware-config > "$dir/host/hardware-configuration.nix" \
      || in_die "$IN_EX_FAIL" "nixos-generate-config failed"
  fi

  # Repositorio git: Nix solo ve los archivos que git conoce, y el usuario tendrá historial desde el día uno.
  in_run git -C "$dir" init --quiet --initial-branch main
  in_run git -C "$dir" add .
  in_run git -C "$dir" -c user.name=Maxor -c user.email=maxor@localhost commit --quiet -m "Initial Maxor OS configuration"

  # Fija las versiones (flake.lock). La ISO trae en /etc/maxor-install/overrides (o MAXOR_INSTALL_OVERRIDES)
  # una línea «entrada=referencia» por cada entrada del flake (maxor-os, maxor-os/nixpkgs, …) con su copia local
  # en el disco (ver offline-overrides.nix), que luego se cambia por su referencia de internet; sin ese archivo
  # (instalar desde un sistema ya hecho), se resuelven desde internet.
  local args=() line overrides="${MAXOR_INSTALL_OVERRIDES:-}"
  if [ -z "$overrides" ] && [ -r "$IN_OVERRIDES_FILE" ]; then overrides="$(cat "$IN_OVERRIDES_FILE")"; fi
  if [ -n "$overrides" ]; then
    while IFS= read -r line; do
      [ -n "$line" ] || continue
      args+=(--override-input "${line%%=*}" "${line#*=}")
    done <<< "$overrides"
  fi
  in_run nix flake lock "${args[@]}" "$dir"
  if [ -n "$overrides" ]; then host_lock_canonical "$dir/flake.lock"; fi
  in_run git -C "$dir" add flake.lock
  in_run git -C "$dir" -c user.name=Maxor -c user.email=maxor@localhost commit --quiet -m "Pin Maxor OS"
}

# host_lock_canonical flake.lock → cambia cada copia local (`path:/nix/store/…`) por su referencia de internet,
# la de $IN_LOCKS_FILE con el mismo narHash (ver installer/offline-locks.nix). Nix sigue usando la copia local, que
# encuentra por ese narHash; si un día se borra, la baja otra vez en vez de fallar. Lo que no está en el mapa se
# queda como está.
host_lock_canonical() {
  local lock="$1"
  [ -r "$IN_LOCKS_FILE" ] || return 0
  if [ "$IN_DRY" = 1 ]; then
    printf 'DRYRUN: point %s at the internet references in %s\n' "$lock" "$IN_LOCKS_FILE"
    return 0
  fi
  jq --slurpfile c "$IN_LOCKS_FILE" '.nodes |= map_values(
      if .locked.type? == "path" and ($c[0][.locked.narHash // ""] != null) then .locked = $c[0][.locked.narHash] else . end)' \
    "$lock" | write_file "$lock" 0644 || in_die "$IN_EX_FAIL" "could not rewrite $lock"
}
