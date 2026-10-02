<div align="center">

```
 ███╗   ███╗ █████╗ ██╗  ██╗ ██████╗ ██████╗
 ████╗ ████║██╔══██╗╚██╗██╔╝██╔═══██╗██╔══██╗
 ██╔████╔██║███████║ ╚███╔╝ ██║   ██║██████╔╝
 ██║╚██╔╝██║██╔══██║ ██╔██╗ ██║   ██║██╔══██╗
 ██║ ╚═╝ ██║██║  ██║██╔╝ ██╗╚██████╔╝██║  ██║
 ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝
                  O  S
```

**Un escritorio Hyprland declarativo, temable en un comando y reproducible en cualquier máquina.**

[![CI](https://github.com/bgrados31/maxor-os/actions/workflows/check.yml/badge.svg)](https://github.com/bgrados31/maxor-os/actions/workflows/check.yml)
![NixOS 26.05](https://img.shields.io/badge/NixOS-26.05-5277C3?logo=nixos&logoColor=white)
![Hyprland](https://img.shields.io/badge/Hyprland-0.55-58E1FF)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

[Instalación](docs/INSTALL.md) · [Arquitectura](docs/ARCHITECTURE.md) · [Temas](docs/THEMING.md) · [Identidad](docs/IDENTITY.md) · [Hoja de ruta](docs/ROADMAP.md)

</div>

---

## Qué es

Maxor OS es una configuración de NixOS que convierte una instalación limpia en un escritorio
Hyprland completo y con identidad propia. Todo el sistema (arranque, shell, terminal, bloqueo,
fuentes y temas) vive en un flake: un cambio es un archivo, y volver atrás es elegir la
generación anterior en el menú de arranque.

El objetivo a largo plazo es una distribución completa, con tienda de apps, tienda de temas e
instalador propios. Hoy el proyecto está en la **fase 3 de 7** (ver [hoja de ruta](docs/ROADMAP.md)).

## Qué incluye

| Capa | Componentes |
|---|---|
| **Sesión** | Hyprland 0.55 con configuración en Lua, portales `xdg-desktop-portal-hyprland` y `-gtk`, SDDM en Wayland |
| **Shell** | [DankMaterialShell](https://github.com/AvengeMedia/DankMaterialShell): barra, launcher, notificaciones, centro de control |
| **Bloqueo** | hyprlock con diseño propio e inactividad con hypridle; la paleta sale del tema activo |
| **Motor de temas** | CLI `maxor theme`: aplica una paleta completa sin `rebuild` ni `sudo` |
| **Terminal** | kitty, fish, starship, zoxide, eza, bat, btop, fastfetch |
| **Arranque** | systemd-boot con partición XBOOTLDR, Plymouth propio y arranque silencioso |
| **Gráficos** | Intel + NVIDIA con PRIME offload (`nvidia-offload <app>`) |
| **Identidad** | `os-release` como "Maxor OS", Figtree, Red Hat Mono y Krona One, iconos Papirus y cursor Bibata |

## Capturas

> Pendiente: las capturas se añadirán en `docs/img/` cuando termine la fase 4 (shell propio).

## Inicio rápido

Requiere NixOS 25.11 o posterior con flakes habilitados. El host de referencia es `nitro`
(Acer Nitro AN16, Intel + NVIDIA); para otra máquina lee primero la
[guía de instalación](docs/INSTALL.md), que explica qué adaptar.

```sh
git clone https://github.com/bgrados31/maxor-os ~/nixos-config
cd ~/nixos-config
sudo nixos-rebuild switch --flake .#nitro
```

Después de reiniciar, el login (SDDM) entra directo a Hyprland. Si algo falla, elige una
generación anterior en el menú de arranque o, desde una TTY (`Ctrl+Alt+F3`):

```sh
sudo nixos-rebuild switch --rollback
```

## Temas

```sh
maxor theme list              # temas instalados
maxor theme apply glaciar     # barra, kitty, Hyprland, GTK y bloqueo cambian a la vez
maxor theme undo              # volver al tema anterior
```

Temas oficiales actuales: **Sakura nocturna** (por defecto) y **Glaciar**. Un tema es una
carpeta de datos (`colors.json`, `theme.toml`, `wallpaper.png`) y nunca ejecuta código.
Detalles y cómo crear el tuyo en [docs/THEMING.md](docs/THEMING.md).

## Atajos principales

| Atajo | Acción |
|---|---|
| `SUPER + Enter` / `T` | Terminal (kitty) |
| `SUPER + Space` / `D` | Launcher |
| `SUPER + ,` | Ajustes de DMS (barra, wallpaper, tema) |
| `SUPER + Y` | Selector de wallpaper |
| `SUPER + Q` | Cerrar ventana |
| `SUPER + 1-9` | Cambiar de workspace |
| `SUPER + TAB` | Vista general |
| `SUPER + V` | Historial del portapapeles |
| `SUPER + X` | Menú de energía |
| `SUPER + ALT + L` | Bloquear pantalla |
| `Print` / `SUPER + SHIFT + S` | Captura |
| `SUPER + SHIFT + /` | Lista completa de atajos |

## Estructura

```
flake.nix                      entradas: nixpkgs 26.05, home-manager, DMS
hosts/nitro/                   configuración propia de la máquina (arranque, NVIDIA, usuario)
modules/branding.nix           nombre del sistema, Plymouth, arranque silencioso
modules/fonts.nix              Figtree, Red Hat Mono, Krona One
home/bryan.nix                 usuario: kitty, fish, GTK/Qt, apps
home/hyprland.nix              Hyprland (Lua) y atajos
home/lockscreen.nix            hyprlock y hypridle
home/maxor.nix                 CLI `maxor` y temas oficiales
branding/                      logotipo en texto
docs/                          documentación
```

## Documentación

- [Instalación y adaptación a otro equipo](docs/INSTALL.md)
- [Arquitectura](docs/ARCHITECTURE.md)
- [Motor de temas](docs/THEMING.md)
- [Identidad visual](docs/IDENTITY.md)
- [Solución de problemas](docs/TROUBLESHOOTING.md)
- [Hoja de ruta](docs/ROADMAP.md)
- [Licencias de terceros](THIRD_PARTY.md)
- [Cómo contribuir](CONTRIBUTING.md) · [Seguridad](SECURITY.md) · [Cambios](CHANGELOG.md)

## Licencia

[MIT](LICENSE). Los componentes y fuentes de terceros conservan sus propias licencias; ver
[THIRD_PARTY.md](THIRD_PARTY.md).
