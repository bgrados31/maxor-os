# Cambios

Formato basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/). El proyecto aún
no usa versiones numeradas.

## [Sin publicar]

### Añadido

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

- Hyprland es la única sesión: se eliminaron Budgie y LightDM. El login pasa a SDDM en Wayland.
- Esquinas de 8 px y desenfoque moderado en Hyprland.

## Historial previo

Base inicial: NixOS 26.05 con Hyprland 0.55 (configuración en Lua), DankMaterialShell, kitty,
fish, starship y NVIDIA PRIME en modo *offload*.
