# Cambios

Formato basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/). El proyecto aún
no usa versiones numeradas.

## [Sin publicar]

### Añadido

- **Base de apps y memoria**: Flatpak con Flathub añadido solo, AppImage ejecutable directamente y `earlyoom` contra congelamientos por falta de RAM. `maxor update` resume los cambios y avisa si hay que reiniciar por un kernel nuevo.

- **Drivers según el equipo**: `maxor hardware` detecta CPU, GPU y tipo de equipo (Intel, AMD, NVIDIA, híbridos, máquinas virtuales) y `modules/hardware.nix` configura microcódigo, drivers de vídeo y ajustes de portátil; `maxor doctor` avisa si el hardware cambió. Añade `fwupd` y la aceleración de vídeo de Intel. Ver [HARDWARE.md](docs/HARDWARE.md).

- **Maxor Shell**: DankMaterialShell con la identidad de Maxor OS, como capa de parches (logo «M» generado desde Krona One, nombre y textos propios) que sigue recibiendo las mejoras de upstream.
- **Login propio**: greetd con el greeter de Maxor Shell, que hereda el tema, los colores y el wallpaper del usuario. Reemplaza a SDDM.

- **Forma de los temas** (`style.json`): cada tema define esquinas, espacios, bordes, desenfoque, opacidad de ventanas sin foco y velocidad de animaciones. Solo números validados; `maxor` genera el Lua de Hyprland y recarga. Los diez temas oficiales tienen su propia forma.

- **Diez temas oficiales**: cinco oscuros (Sakura nocturna, Glaciar, Obsidiana, Brasa, Ultravioleta) y cinco claros (Alba, Escarcha, Papel frío, Brisa, Ámbar), todos con contraste AA. Los temas son carpetas en `themes/`.
- **Modo claro** de extremo a extremo: DMS, lockscreen y CLI se adaptan al `mode` del tema.
- **`maxor theme install` y `export`**: instalar desde carpeta o `.tar.gz` con validación estricta (solo datos permitidos, sin enlaces ni rutas peligrosas) y empaquetar temas.
- **Interfaz de terminal propia** para toda la CLI: cada comando se dibuja como una ventana dentro de la terminal, con los colores del tema activo, spinner y confirmaciones.

- **`maxor update`, `maxor rollback` y `maxor doctor`**: actualizar mostrando los cambios antes de aplicar, volver atrás y diagnosticar el sistema.
- **Configuración de Hyprland en módulos Lua** (`settings`, `rules`, `binds`) y un `user.lua` personal que el sistema nunca sobrescribe.
- **`nixosModules.default`** para reutilizar los módulos de sistema desde otro flake.

- **Motor de temas** (`maxor theme list | current | apply | undo`) con los temas oficiales
  Sakura nocturna y Glaciar. Un tema es una carpeta de datos validada antes de aplicarse.
- **Lockscreen** con hyprlock e inactividad con hypridle, con la paleta del tema activo.
- **Identidad de Maxor OS**: nombre del sistema (`os-release`), tema Plymouth con barra de
  progreso, fuentes Figtree, Red Hat Mono y Krona One, y wallpaper generado desde la paleta.
- **Arranque** con systemd-boot y partición XBOOTLDR para compartir la ESP con Windows.
- **Documentación**: arquitectura, instalación, temas, identidad, solución de problemas, hoja de
  ruta y licencias de terceros.
- **GitHub CLI** disponible para el usuario.

### Cambiado

- Los wallpapers de los temas ya no llevan ruido (pasan de 20 MB a unos 2 MB).
- La configuración del sistema se divide en `modules/core.nix` y `modules/desktop.nix`; `hosts/nitro` conserva solo lo propio del equipo.

- Hyprland es la única sesión: se eliminaron Budgie y LightDM.
- Esquinas de 8 px y desenfoque moderado en Hyprland.

## Historial previo

Base inicial: NixOS 26.05 con Hyprland 0.55 (configuración en Lua), DankMaterialShell, kitty,
fish, starship y NVIDIA PRIME en modo *offload*.
