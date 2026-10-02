# Arquitectura

Maxor OS es un único flake con un host (`nitro`) y una configuración de usuario gestionada por
home-manager como módulo de NixOS. La regla de diseño es separar cuatro cosas:

1. **Lo que cambia por máquina** (discos, GPU, nombre): `hosts/<equipo>/`.
2. **Lo que define a Maxor** (nombre, fuentes, arranque): `modules/`.
3. **Lo que ve y toca el usuario** (terminal, atajos, bloqueo, temas): `home/`.
4. **Lo que el usuario puede cambiar sin `rebuild`** (temas): `~/.local/share/maxor/` y
   `~/.config/maxor/`.

## Mapa del repositorio

| Ruta | Responsabilidad |
|---|---|
| `flake.nix` | Entradas (nixpkgs 26.05, home-manager 26.05, DMS) y la definición de `nixosConfigurations.nitro` |
| `hosts/nitro/configuration.nix` | Arranque (systemd-boot + XBOOTLDR), Nix, red, NVIDIA PRIME, sesión, audio, Bluetooth, usuario |
| `hosts/nitro/hardware-configuration.nix` | Generado por `nixos-generate-config`; propio de este equipo |
| `modules/branding.nix` | `system.nixos.distroName`, tema Plymouth, parámetros de arranque silencioso, `/etc/issue` |
| `modules/fonts.nix` | Fuentes del sistema y la que usa Plymouth |
| `home/bryan.nix` | kitty, fish, starship, GTK/Qt, cursor, paquetes de usuario, `gh` |
| `home/hyprland.nix` | Configuración de Hyprland en Lua y todos los atajos |
| `home/lockscreen.nix` | hyprlock (diseño) e hypridle (inactividad) |
| `home/maxor.nix` | CLI `maxor` y empaquetado de los temas oficiales |

## Cómo se reparten las responsabilidades de color

Hay una sola fuente de verdad para los colores: el tema activo.

```
 ~/.local/share/maxor/themes/<tema>/colors.json
                  │
        maxor theme apply
                  │
   ┌──────────────┼───────────────────────┐
   ▼              ▼                       ▼
 dms-theme.json   hyprlock.conf       wallpaper.png
   │              (plantilla +          (dms ipc wallpaper)
   │               paleta)
   ▼
 DMS (matugen/plantillas)
   │
   ├─ barra, launcher, notificaciones
   ├─ kitty      (~/.config/kitty/dank-theme.conf)
   ├─ Hyprland   (~/.config/hypr/dms/colors.lua)
   └─ GTK / Qt
```

`maxor theme apply` genera un tema propio para DMS y lo activa editando
`~/.config/DankMaterialShell/settings.json` (`currentThemeName = "custom"`,
`customThemeFile`). DMS se encarga de recolorear el resto. El lockscreen no depende de DMS:
`home/lockscreen.nix` publica una plantilla con marcadores (`@FG_RGB@`, `@AC_RGB@`, …) que el
CLI rellena con la paleta.

## Sesión de escritorio

- **Login:** SDDM en Wayland con `defaultSession = "hyprland"`. Es provisional: el plan es
  reemplazarlo por un greeter propio (greetd).
- **Hyprland:** instalado por `programs.hyprland` (NixOS). home-manager solo escribe la
  configuración (`configType = "lua"`, `package = null`).
- **DMS:** arranca como servicio de usuario de systemd (`programs.dank-material-shell.systemd`).
- **Bloqueo:** `SUPER + ALT + L` ejecuta `loginctl lock-session`; hypridle recibe la señal y
  lanza hyprlock con la configuración del tema activo. Si esa configuración no existe, usa la
  predeterminada (Sakura). PAM de hyprlock se declara en `hosts/nitro/configuration.nix`.

## Arranque

```
UEFI → systemd-boot (ESP, /efi) → kernel + initrd (XBOOTLDR, /boot) → Plymouth → SDDM → Hyprland
```

La ESP de 100 MB se comparte con Windows (arranque dual). Los kernels e initrd viven en una
partición XBOOTLDR de 1 GB montada en `/boot`, para que las generaciones de NixOS no llenen la
ESP. `configurationLimit = 10` limita cuántas generaciones se conservan.

## Principios

- **Declarativo primero.** Si algo se puede expresar en Nix, no se edita a mano.
- **Reversible.** Cada cambio de sistema es una generación; cada cambio de tema se deshace con
  `maxor theme undo`.
- **Datos, no código, en los temas.** Un tema no puede ejecutar nada. Ver
  [THEMING.md](THEMING.md#seguridad).
- **Separar sistema y usuario.** El usuario personaliza en `~/.config/maxor/`; el sistema se
  actualiza sin pisarle nada.
