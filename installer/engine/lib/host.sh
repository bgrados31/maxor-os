# ── Etapa host: escribir la configuración de la máquina nueva ────────
# La máquina recibe un flake propio, en ~/nixos-config y como repositorio git, que solo dice lo
# que es particular de ella (maxor.machine, su hardware, su arranque) y toma de Maxor OS todo lo
# demás como entrada de flake. Actualizar es después subir esa entrada (`maxor release apply`).
IN_OS_URL="${MAXOR_INSTALL_OS_URL:-github:bgrados31/maxor-os}"
IN_MAXOR="${MAXOR_BIN:-maxor}"

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
        ({ ... }: {
          maxor.hardware.report = ./host/hardware.json;
          maxor.settings = ./host/maxor.json;
        })$autologin
      ];
    };
  };
}
__END_FLAKE__
}

host_boot_nix() {
  cat << '__END_BOOT__'
{ ... }:

# El arranque de esta máquina. Los discos y, si hay cifrado, el desbloqueo de la raíz están en
# hardware-configuration.nix, que generó nixos-generate-config al instalar.
{
  boot.loader.systemd-boot.enable = true;
  boot.loader.systemd-boot.configurationLimit = 10;
  boot.loader.efi.canTouchEfiVariables = true;

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

  prof="$(jq '{profiles: .look.profiles}' "$IN_ANSWERS")"
  printf '%s\n' "$prof" | write_file "$dir/host/maxor.json" 644

  # El hardware se detecta aquí, en la máquina que se instala.
  if [ "$IN_DRY" = 1 ]; then
    printf 'DRYRUN: %s hardware detect > %s\n' "$IN_MAXOR" "$dir/host/hardware.json"
    printf 'DRYRUN: nixos-generate-config --root %s --show-hardware-config > %s\n' "$IN_ROOT" "$dir/host/hardware-configuration.nix"
  else
    "$IN_MAXOR" hardware detect > "$dir/host/hardware.json" || in_die "$IN_EX_FAIL" "could not detect the hardware"
    nixos-generate-config --root "$IN_ROOT" --show-hardware-config > "$dir/host/hardware-configuration.nix" \
      || in_die "$IN_EX_FAIL" "nixos-generate-config failed"
  fi

  # Repositorio git: Nix solo ve los archivos que git conoce, y el usuario tendrá historial desde el día uno.
  in_run git -C "$dir" init --quiet --initial-branch main
  in_run git -C "$dir" add .
  in_run git -C "$dir" -c user.name=Maxor -c user.email=maxor@localhost commit --quiet -m "Initial Maxor OS configuration"

  # Fija la versión de Maxor OS (flake.lock): desde la copia de la ISO sin red, o desde la red.
  if [ -n "${MAXOR_INSTALL_OS_PATH:-}" ]; then
    in_run nix flake lock --override-input maxor-os "path:$MAXOR_INSTALL_OS_PATH" "$dir"
  else
    in_run nix flake lock "$dir"
  fi
  in_run git -C "$dir" add flake.lock
  in_run git -C "$dir" -c user.name=Maxor -c user.email=maxor@localhost commit --quiet -m "Pin Maxor OS"
}
