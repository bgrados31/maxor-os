# Hoja de ruta

Maxor OS aspira a ser una distribución completa basada en NixOS: escritorio Hyprland,
personalizable al 100 %, con tienda de apps, tienda de temas, shell propio, menú de arranque
propio e instalador propio. Referencias de experiencia: Ryoku OS, Omarchy y los dotfiles de end-4.

## Estado

| Fase | Objetivo | Estado |
|---|---|---|
| 0 | Base NixOS + Hyprland + DMS funcionando | Hecha |
| 1 | Solo Hyprland, branding mínimo (`os-release`, Plymouth, fastfetch) | Hecha |
| 2 | Repo ordenado y modular, CLI `maxor` v0 (update, rollback, doctor) | Casi hecha |
| 3 | Motor de temas y temas oficiales | Núcleo hecho, faltan temas |
| 4 | Shell Maxor y greeter propio | Pendiente |
| 5 | Maxor Store (apps Flatpak y temas) | Pendiente |
| 6 | Menú de arranque (Limine), ISO live e instalador | Pendiente |
| 7 | Theme Store web, comunidad, documentación y sitio | Pendiente |

## Fase 2: repo ordenado y CLI

- [x] Repositorio con documentación, licencia y CI.
- [x] Separar `configuration.nix` en `modules/core.nix` y `modules/desktop.nix`; `branding` y `fonts` ya eran módulos.
- [ ] Extraer `modules/shell.nix` y `modules/theming.nix` cuando el shell propio exista (fase 4).
- [x] Dejar `hosts/<equipo>` solo con lo propio del equipo (arranque, GPU, región, usuario).
- [ ] Exportar `homeModules.default` para reutilizar la configuración de usuario.
- [x] Configuración de Hyprland en módulos Lua (`settings`, `rules`, `binds`) y un `user.lua` que nunca se sobrescribe.
      Los monitores los gestiona DMS.
- [x] `maxor update`, `maxor rollback` y `maxor doctor`.

## Fase 3: motor de temas

- [x] Formato de tema (`colors.json`, `theme.toml`, `wallpaper.png`).
- [x] `maxor theme list | current | apply | undo`.
- [x] Dos temas oficiales: Sakura nocturna y Glaciar.
- [ ] Tres temas oficiales más: Obsidiana, Brasa y Ultravioleta, y una variante clara.
- [ ] Incluir en el tema la configuración de Hyprland (esquinas, bordes, animaciones).
- [ ] Incluir en el tema la disposición de la barra.
- [ ] `maxor theme install <url|archivo>` con validación de esquema y firma opcional.
- [ ] `maxor theme export`.

## Fase 4: identidad completa

Dos caminos para el shell. Se empieza por el A y se migra al B cuando haya capacidad.

- **A. Fork de DankMaterialShell:** renombrar, cambiar logo y tipografía, pantalla de ajustes y
  widgets propios, manteniendo upstream como remoto. DMS se distribuye bajo MIT.
- **B. Shell propio en Quickshell (QML):** control total de barra, dock, launcher, centro de
  control, notificaciones, lockscreen y OSD.

Además: greeter propio con greetd en lugar de SDDM y sesión por UWSM.

## Fase 5: tienda

- Backend: Flatpak (Flathub) para apps de usuario y paquetes de nixpkgs para herramientas del
  sistema, instalables sin editar `.nix`.
- Frontend: Maxor Store con tres pestañas: Apps, Temas y Extras.

## Fase 6: arranque e instalador

- Menú de arranque con Limine (con GRUB con tema como plan B). Se prueba primero en una VM y
  luego en un USB; nunca se cambia el cargador en una máquina con Windows sin respaldo.
- ISO generada desde el flake, con sesión live en Hyprland.
- Instalador: Calamares con la marca de Maxor, o uno propio con elección de tema y perfil
  (Gaming, Dev, Minimal, Creator). Detección automática de GPU, incluida NVIDIA híbrida.
- Caché binaria propia para que instalar tarde minutos y no horas.

## Fase 7: comunidad

- Theme Store web con previsualizaciones, votos y búsqueda.
- Cuentas de autor, firma de temas y moderación.
- Sitio y documentación pública.

## Riesgos y reglas

- **No reemplazar el cargador de arranque sin respaldo.** Esta laptop comparte ESP con Windows.
- Probar la ISO y el instalador en una VM antes de hardware real.
- Revisar y respetar las licencias de DMS, Quickshell, Hyprland, Calamares, fuentes y
  wallpapers antes de redistribuir (ver [THIRD_PARTY.md](../THIRD_PARTY.md)).
- Hyprland cambia rápido: la versión queda fijada en `flake.lock` y se actualiza por canal.
- NVIDIA híbrida es la parte más frágil; el instalador debe poder caer a un perfil sin GPU
  dedicada.
- Un tema de la comunidad es superficie de ataque: solo datos y sin ejecución por defecto.
