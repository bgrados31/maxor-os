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
| `flake.nix` | Entradas (nixpkgs 26.05, home-manager 26.05, DMS), `nixosModules.default` y `nixosConfigurations.nitro` |
| `hosts/nitro/configuration.nix` | Solo lo propio del equipo: arranque (systemd-boot + XBOOTLDR), región y teclado, NVIDIA PRIME, usuario |
| `hosts/nitro/hardware-configuration.nix` | Generado por `nixos-generate-config`; propio de este equipo |
| `modules/core.nix` | Ajustes de Nix, red, zram, audio (PipeWire), Bluetooth, impresión, paquetes base |
| `modules/desktop.nix` | Hyprland, PAM de hyprlock, variables de sesión, servicios del escritorio |
| `modules/greeter.nix` | Login: greetd con el greeter de DMS y el paquete Maxor Shell |
| `packages/` | `maxor-shell.nix` (DMS con la identidad de Maxor), `make-mark.py` (genera el logo) y `fonts.nix` |
| `modules/branding.nix` | `system.nixos.distroName`, tema Plymouth, parámetros de arranque silencioso, `/etc/issue` |
| `modules/fonts.nix` | Fuentes del sistema y la que usa Plymouth |
| `home/bryan.nix` | kitty, fish, starship, GTK/Qt, cursor, paquetes de usuario, `gh` |
| `home/hyprland.nix` | Carga los módulos Lua de Hyprland y crea `user.lua` la primera vez |
| `home/hyprland/*.lua` | `settings.lua` (apariencia), `rules.lua` (reglas), `binds.lua` (atajos), `theme.lua` (forma del tema activo), `user.lua.example` |
| `home/lockscreen.nix` | hyprlock (diseño) e hypridle (inactividad) |
| `home/maxor.nix` | Empaquetado del CLI y de los temas oficiales (genera sus wallpapers) |
| `home/maxor/*.sh` | El CLI: `lib`, `ui` (ventanas), `theme`, `system` (update, rollback, doctor) y `main` |
| `themes/<id>/` | Temas oficiales: `colors.json` y `theme.toml` |

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

- **Login:** greetd con el greeter de DMS (paquete Maxor Shell) dentro de Hyprland; ver
  [SHELL.md](SHELL.md). Hereda del usuario el tema, los colores y el wallpaper.
- **Hyprland:** instalado por `programs.hyprland` (NixOS). home-manager solo escribe la
  configuración (`configType = "lua"`, `package = null`).
- **DMS:** arranca como servicio de usuario de systemd (`programs.dank-material-shell.systemd`).
- **Bloqueo:** `SUPER + ALT + L` ejecuta `loginctl lock-session`; hypridle recibe la señal y
  lanza hyprlock con la configuración del tema activo. Si esa configuración no existe, usa la
  predeterminada (Sakura). PAM de hyprlock se declara en `hosts/nitro/configuration.nix`.

## Arranque

```
UEFI → systemd-boot (ESP, /efi) → kernel + initrd (XBOOTLDR, /boot) → Plymouth → greetd (greeter) → Hyprland
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

## Reutilizar Maxor desde otro flake

Los módulos de sistema se exponen como `nixosModules.default` (núcleo, escritorio, identidad y
fuentes). Otro flake puede importarlos y añadir solo su hardware:

```nix
{
  inputs.maxor.url = "github:bgrados31/maxor-os";
  # …
  modules = [ maxor.nixosModules.default ./mi-equipo.nix ];
}
```

La configuración de usuario (`home/`) todavía no se exporta como módulo; es un paso pendiente de
la [fase 2](ROADMAP.md).

## Configuración de Hyprland en módulos

`hyprland.lua` solo carga módulos, en este orden:

1. `maxor.settings`, `maxor.rules`, `maxor.binds`: de solo lectura, vienen de este repositorio.
2. `dms.*`: colores del tema, monitores, cursor, reglas y **layout** (esquinas y espacios) que gestiona DankMaterialShell.
3. `maxor.theme`: la forma del tema activo. Va después de DMS a propósito: el layout de DMS fija sus propias esquinas y espacios, y si este módulo cargara antes, DMS lo pisaría.
4. `maxor.user`: **tuyo**. Se crea una sola vez desde `user.lua.example`, se carga el último y el
   sistema nunca lo modifica, así que lo que pongas ahí sobrescribe todo lo anterior.

Los monitores los gestiona DMS (`dms.outputs`); por eso no hay un `monitors.lua` propio.
