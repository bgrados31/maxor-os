# Plan de la base

Qué hay que construir para que Maxor OS sea sólido antes de la tienda, el instalador y la
comunidad. Complementa la [hoja de ruta](ROADMAP.md): ella dice *qué fases hay*, este plan dice
*en qué orden y con qué decisiones*.

## Principios

1. **Todo pasa por la CLI.** Cada app gráfica es una cara de `maxor …`; la lógica vive en un
   solo sitio y toda orden gráfica tiene su equivalente en terminal. Para ello la CLI gana
   `--json` en cada subcomando.
2. **Una sola tecnología de interfaz.** Quickshell (QML), la misma base de DMS. Lo que se
   escriba para las apps sirve después para el shell propio.
3. **Una sola fuente de identidad.** Los colores, fuentes, formas y logos salen de *design
   tokens* (un JSON); el shell, las apps, el greeter, Plymouth y la web los consumen. Cambiar
   la marca es tocar un archivo.
4. **Nunca se rompe.** Todo cambio se compila antes de pedir el `switch`; hay generaciones de
   respaldo; lo que el usuario toca (`~/.config/maxor/`) no lo pisa el sistema.
5. **Reemplazar, no reescribir.** El shell propio nace pieza a pieza sobre el actual.

## M1. Base sólida (optimización y salud)

Hecho: zram, TRIM, thermald, power-profiles, journald acotado, GC y optimise de Nix, arranque
y apagado silenciosos, sesión limpia, Thunar.

- [ ] `maxor update` con diff de paquetes (`nvd`/`nh`) y reporte de qué cambió.
- [ ] Caché binaria propia (Cachix) y sustituyentes configurados desde el flake.
- [ ] `nix-ld` + `appimage-run` para que los binarios ajenos funcionen sin pelear con Nix.
- [ ] Flatpak y Flathub activos desde el primer arranque (los usará la tienda).
- [ ] `earlyoom`, y evaluar un kernel más reciente o `zen` y un planificador `scx`.
- [ ] Perfiles `maxor profile gaming|dev|minimal|creator` (Steam/gamemode, toolchains, etc.).
- [x] Drivers por equipo: detección de CPU/GPU/portátil/VM con `maxor hardware` y `modules/hardware.nix` ([HARDWARE.md](HARDWARE.md)).
- [ ] Gráficos: conmutación PRIME guiada con `maxor gpu` y modo «sin GPU dedicada».
- [ ] `maxor doctor --fix` para lo corregible, y `maxor doctor --json`.
- [ ] `maxor backup` (config de usuario) y `maxor restore`.
- [ ] Paquetes de usuario sin editar `.nix`: `~/.config/maxor/packages.nix` y `maxor install`.
- [ ] Sesión por UWSM (servicios de usuario ordenados, cierre limpio).
- [ ] Firmware (`fwupd`), impresión (CUPS) y códecs: lo que «simplemente debe funcionar».
- [ ] Pruebas: `nix flake check` con una prueba de arranque en VM de la configuración.

## M2. Biblioteca de interfaz y apps propias

Biblioteca `maxor-ui` (QML): ventana, tarjetas, botones, listas, interruptores, diálogos,
iconos, tipografía y animaciones, todo leyendo los design tokens y el tema activo.

Apps, en orden:

| App | Para qué | Habla con |
|---|---|---|
| **Maxor Welcome** | Primer arranque: idioma, tema, perfil, GPU, cuentas, tour de atajos | `maxor profile`, `maxor theme` |
| **Maxor Settings** | Centro de ajustes: temas, atajos, pantallas, perfil, gráficos, actualizaciones, copias | toda la CLI |
| **Maxor Update / Recovery** | Actualizar, ver cambios, volver a una generación, modo seguro | `maxor update`, `rollback` |
| **Maxor Theme Studio** | Editar un tema en vivo (paleta, forma, wallpaper) y exportarlo | `maxor theme` |
| **Maxor Store** (fase 5) | Apps (Flatpak/Nix), temas, extras | `maxor install`, `theme install` |
| **Maxor Installer** (fase 6) | Instalar el sistema | `nixos-install` + `maxor` |

## M3. Branding perfecto

Un solo pase que cubre cada punto de contacto, con la identidad ya fijada en
[IDENTITY.md](IDENTITY.md):

- [ ] Suite de logos: marca de texto, versión pequeña, monocromo, iconos de aplicación, favicon.
- [ ] Design tokens (`branding/tokens.json`) y un generador que produce CSS/QML/Nix/Lua desde ellos.
- [ ] Tema de iconos y de cursor propios (o selección curada con la paleta del tema).
- [ ] Coherencia GTK/Qt/libadwaita con el tema activo, sin ventanas «de otro sistema».
- [ ] Arranque completo con una sola paleta: menú de arranque, Plymouth, greeter, escritorio.
- [ ] `os-release` completo (`LOGO`, `ANSI_COLOR`, `HOME_URL`, `SUPPORT_URL`), fastfetch y `issue`.
- [ ] Krona One y la marca dentro de la interfaz de DMS; ajustes con marca; traducciones.
- [ ] Sonidos del sistema (inicio, notificación, error) con el mismo carácter.
- [ ] Wallpapers oficiales por tema y capturas para el README.
- [ ] Sitio y documentación con el mismo sistema de diseño.

## M4. Shell propio

Decisión: **estrangulación gradual**, no reescritura total.

1. Mientras tanto, Maxor Shell sigue siendo la capa de parches sobre DMS (ya funciona).
2. Las apps de M2 construyen `maxor-ui`, que es el cimiento del shell propio.
3. Se reemplaza una pieza por vez, empezando por las más visibles y aisladas: **lockscreen**,
   **OSD** (volumen/brillo), **launcher**, **notificaciones**, **centro de control**, **barra**.
   Cada pieza propia convive con DMS vía el mismo IPC hasta que el último componente se
   sustituye y DMS deja de ser dependencia.
4. Criterio de salida de cada pieza: igual de estable que la de DMS, con los tokens y el tema.

Riesgo: mantener ambos durante la transición. Mitigación: piezas pequeñas, una a la vez, cada
una detrás de una opción (`maxor.shell.<pieza> = "dms" | "maxor"`).

## Orden propuesto

1. M1 (base) y capturas del README: pocas sesiones, el sistema queda sano y presentable.
2. Design tokens (primer paso de M3), porque todo lo demás los consume.
3. `maxor-ui` + Welcome + Settings (M2).
4. Resto de M3 (branding) con las apps ya existentes.
5. Primeras piezas del shell propio (M4) y, en paralelo, la tienda (fase 5).
6. Boot menu, ISO e instalador (fase 6) al final, con la base estable.
