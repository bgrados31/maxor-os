# Instalación

Maxor OS aún no tiene instalador propio (está en la [fase 6](ROADMAP.md)). Hoy se instala
aplicando el flake sobre una instalación existente de NixOS.

## Requisitos

- NixOS 25.11 o posterior, arranque UEFI.
- Flakes habilitados (`nix.settings.experimental-features = [ "nix-command" "flakes" ]`).
- Conexión a Internet: la primera compilación descarga varios cientos de MB (fuentes, NVIDIA,
  Hyprland, DMS).
- Un usuario con permisos de `sudo`.

## Instalación en el equipo de referencia (`nitro`)

El host `nitro` está pensado para un Acer Nitro AN16 con Intel + NVIDIA RTX 4050, arranque dual
con Windows y la partición XBOOTLDR descrita abajo. Si tu máquina es esa, o muy parecida:

```sh
git clone https://github.com/bgrados31/maxor-os ~/nixos-config
cd ~/nixos-config
sudo nixos-rebuild boot --flake .#nitro     # primera vez: déjalo para el próximo arranque
reboot
```

`boot` en lugar de `switch` aplica la nueva generación solo en el siguiente arranque y deja la
actual intacta como respaldo.

## Adaptarlo a otro equipo

Copia `hosts/nitro` a `hosts/<tu-equipo>` y ajusta lo siguiente.

### 1. Hardware

Regenera `hardware-configuration.nix` en tu máquina; no uses el del repositorio:

```sh
sudo nixos-generate-config --show-hardware-config > hosts/<tu-equipo>/hardware-configuration.nix
```

### 2. Arranque

`hosts/nitro/configuration.nix` usa systemd-boot con la ESP en `/efi` y una partición XBOOTLDR
en `/boot`. Si tu disco tiene la ESP montada en `/boot` (lo habitual), elimina estas líneas y
deja el valor por defecto:

```nix
boot.loader.efi.efiSysMountPoint = "/efi";
boot.loader.systemd-boot.xbootldrMountPoint = "/boot";
```

### 3. GPU

- **Solo Intel o AMD:** quita el bloque `hardware.nvidia` y `services.xserver.videoDrivers`.
- **Intel + NVIDIA:** cambia los identificadores de bus por los tuyos:

  ```sh
  lspci | grep -E 'VGA|3D'
  ```

  Convierte `00:02.0` en `PCI:0:2:0` y `01:00.0` en `PCI:1:0:0`.

### 4. Usuario, zona horaria y teclado

Edita en `hosts/<tu-equipo>/configuration.nix`: `networking.hostName`, `time.timeZone`,
`console.keyMap`, `services.xserver.xkb` y `users.users.<nombre>`. En `home/bryan.nix` cambia
`home.username`, `home.homeDirectory` y los datos de `programs.git`.

### 5. Registrar el host en el flake

En `flake.nix`, duplica el bloque `nixosConfigurations.nitro`, cámbiale el nombre y apunta
`./hosts/<tu-equipo>/configuration.nix`. Luego:

```sh
sudo nixos-rebuild switch --flake .#<tu-equipo>
```

## Después de instalar

1. Reinicia. El login (SDDM) entra directo a Hyprland.
2. Abre los ajustes de DMS con `SUPER + ,` y configura la barra y el wallpaper.
3. Aplica un tema: `maxor theme apply sakura`.
4. Opcional: abre `qt6ct` una vez y elige el esquema de colores de DMS para las apps Qt.

## Actualizar

```sh
nix flake update --flake ~/nixos-config
sudo nixos-rebuild switch --flake ~/nixos-config#nitro
```

Dentro de fish existen los alias `rebuild` y `update` para ambas cosas.

## Desinstalar o volver atrás

- **Una actualización rompió algo:** elige la generación anterior en el menú de arranque.
- **Desde una TTY:** `sudo nixos-rebuild switch --rollback`.
- **Liberar espacio:** el recolector de basura semanal borra generaciones de más de 14 días.
