# Cambios

Formato basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/). El proyecto aún
no usa versiones numeradas.

## [Sin publicar]

### Cambiado

- **La CLI es lineal y cuelga de un riel** (`┌ │ ◇ └`) en vez de dibujar una ventana que imita una terminal: no mide el ancho de cada línea, se copia limpia y se ve igual en un log. Glifos Unicode con respaldo ASCII automático (`TERM=linux`, un locale que no es UTF-8 o `MAXOR_ASCII=1`).
- **Un solo cargador para todo** (`ui_run`): spinner, las últimas líneas si tarda, el tiempo y el fallo con `maxor logs --last`. Desaparecen `ui_run_tail`, `ui_pipeline`, el esqueleto y las pestañas de la CLI; la pantalla completa habla el mismo idioma.
- **`maxor search` en una terminal abre la Tienda** de la pantalla completa; sin ella, o con `--list` y `--json`, imprime la lista. Se quitó el selector interactivo de bash.

- **La CLI habla inglés** (por ahora). Todos los mensajes salen de un catálogo (`home/maxor/lib/lang/en.sh`) y el código solo usa claves, así que añadir español es crear un archivo. Los perfiles también describen sus paquetes en inglés.
- **CLI reorganizada** en `lib/` (núcleo, mensajes, terminal, marco, cargadores, componentes) y `cmd/` (un archivo por grupo de comandos), con el paquete propio `packages/maxor.nix`. Las salidas existentes se conservan.
- **`maxor update` usa la lista de pasos**: «paso 1 de 3», tiempo total y las últimas líneas de la compilación bajo el paso en curso. El diff muestra ↑ + − ~, oculta los archivos de configuración y avisa si hay un kernel nuevo.
- **CLI mucho más rápida**: la interfaz de ventanas ya no lanza procesos por línea; el selector de `maxor search` no parpadea, no deja caracteres en pantalla y aguanta mantener una flecha. Ayuda 352 → 108 ms, `theme list` 1562 → 271 ms, `doctor` 3 s → 0,5 s. Ver [CLI.md](docs/CLI.md).

### Añadido

- **Releases firmadas y aviso de versión nueva** (`maxor release check|status|apply`, ver [UPDATES.md](docs/UPDATES.md)): cada release lleva un manifiesto firmado con la clave del proyecto (`keys/allowed_signers`, dentro del paquete) y una etiqueta de git firmada. La CLI consulta cada hora con una petición condicional (ETag), verifica la firma, la forma del manifiesto y que la secuencia nunca retroceda, y avisa **una vez por versión** con una notificación; lo que no verifica se ignora y se distingue de «sin actualización» (sin red también: nunca se dice «al día» sin haberlo comprobado). `maxor release apply` comprueba la etiqueta firmada y que apunte al commit del manifiesto, avanza el repositorio solo en línea recta y actualiza con el `flake.lock` de la release. En la pantalla completa, Inicio, Update y Exit muestran el aviso y `v` en Update la instala sin salir. Temporizador `maxor-release-check` (cada hora y al iniciar sesión). `scripts/release.sh` ahora firma la etiqueta y el manifiesto y crea la Release; `scripts/release-key.sh` crea la clave; la CI verifica de forma independiente la etiqueta, el manifiesto y que coincidan. 38 pruebas nuevas del motor y 12 de la pantalla.
- **Ejemplos en la ayuda**: `maxor help <comando>` termina con un bloque «Examples» de dos a cuatro órdenes que se pueden copiar, para los comandos principales.
- **`maxor backup` y `maxor restore`**: guardan en un solo archivo tus perfiles, temas propios, ajustes de la barra, el tema en uso y la lista de apps, y lo recuperan en este equipo o en otro (con `--apps` también instala las apps). En la pantalla completa, `b` en Inicio (o `backup` en la paleta) guarda una copia.
- **Aplicar el sistema sin salir de la pantalla**: Update (aplicar y volver a una generación anterior) y Profiles ya no ceden la terminal. Si sudo necesita contraseña, un campo propio la pide (con puntos) y la entrega solo a sudo por la entrada estándar; se ve el progreso real y la salida mientras trabaja, no se puede salir hasta que termina, una contraseña mala se vuelve a pedir y un fallo enseña su código. `t` en Update conserva el modo de siempre, por la terminal. En la CLI, `MAXOR_SUDO_STDIN=1` hace que sudo lea la contraseña de la entrada estándar.
- **Los detalles se adaptan a tu ventana**: con ≥96 columnas van en un panel al lado; en una ventana estrecha (una de cuatro en la pantalla, ~90×25) pasan a un cajón compacto debajo, con lo esencial de cada pestaña (la app elegida y sus botones, el comando del arreglo del Doctor, los botones de Update, el tema elegido con un fastfetch en una fila); y `D` los oculta o los muestra, y lo recuerda (`tui-prefs.json`). El tamaño mínimo baja a 64×20 y la Tienda usa una cabecera compacta en ventanas bajas para que quepan más apps.
- **Los perfiles dicen lo que instalan**: cada uno lista sus programas (`includes` en el catálogo y en `maxor profile list --json`), en una línea en la lista y uno a uno en el panel.
- **Aviso diario de apps con versión nueva**: un temporizador de usuario (`maxor-app-updates`) ejecuta `maxor apps updates --refresh --notify` una vez al día y avisa con una notificación; mirar no compila ni descarga nada.
- **Volver a una versión anterior del sistema desde Update**: `g` abre las últimas generaciones (con fecha, versión y kernel), se elige una y `⏎` vuelve a ella; tras volver se escanea otra vez. La CLI suma `maxor rollback <N>` y `maxor rollback --list [--json]`. Los datos y las apps instaladas con maxor no se tocan.
- **Exit es una pestaña más**: al llegar a ella no sale; enseña lo que queda pendiente (apps por actualizar, avisos del doctor, una actualización lista) y sale con `⏎` o con un clic en su botón.
- **La Tienda tiene una barra de acciones rápidas** bajo las pestañas, con botones que también se pulsan con el ratón: con apps marcadas, `Update n` (o `Check for updates`) y `Remove n`; sin marcar, `Update all n` y `Check for updates`; en los resultados, `Install n`. Se quitan los filtros nixpkgs/All/flathub: cada fila ya dice de dónde viene.
- **Las apps de Flathub salen en Super+Espacio y en la terminal**: Flatpak deja sus entradas de menú, iconos y comandos en una carpeta que solo ven las sesiones que arrancaron con ella en el entorno (la tuya empezó antes de que el sistema la añadiera). Ahora `maxor install` enlaza cada app a las carpetas del usuario (entrada de menú, icono y un comando corto con su nombre, como `atlauncher`), `remove` lo deshace y `maxor apps repair` arregla las ya instaladas. El doctor avisa si alguna no se vería.
- **Botón Exit** a la derecha de las pestañas (`⏻ Exit`), y `exit` en la paleta de comandos. La paleta también abre apps: `open atlauncher`.
- **Más rápido**: `maxor apps updates` guarda su resultado 10 minutos (de ~1,1 s a ~0,08 s) y se descarta al instalar, quitar o actualizar; `R` en la Tienda lo vuelve a mirar. La tarjeta Apps de Inicio avisa de las versiones nuevas y, al instalar, el aviso dice dónde encontrar la app.
- **Installed con casillas**: `space` marca (o un clic en la casilla), `a` marca o desmarca todas, `u` actualiza las marcadas que tengan versión nueva, `U` actualiza todas las pendientes, `r` quita las marcadas y `⏎` abre el menú sobre ellas. Cada fila dice su estado a la derecha (`✓ up to date`, `↑ 1.2.3 available`, `checking…`, `queued`, `updating…`) y debajo de dónde viene y de qué versión a cuál se actualiza. Todo pasa por una cola que hace una operación a la vez, y al quitar varias apps hay una sola pregunta por sus datos.
- **Quitar una app ya no la deja en Super+Espacio**: el lanzador no se enteraba de las apps quitadas porque el perfil de nix cambia de carpeta. `maxor install`, `remove` y `apps update` ahora le avisan creando y borrando un archivo en `~/.local/share/applications`.
- **Las apps instaladas se manejan con un menú, sin letras que aprender**: en Installed, `⏎` abre un menú con Abrir, Actualizar (solo si hay versión nueva), Quitar y «Quitar y borrar sus datos» (con el tamaño que libera y un sí final). La lista avisa de las versiones nuevas con `↑ 1.2.3` y la pestaña con `Installed 3 ↑1`. La CLI suma `maxor apps updates`, `apps update <app>`, `apps open <app>` y `remove --list-data`.
- **Themes**: aire entre los temas oscuros y los claros, y debajo de la vista previa un fastfetch de muestra con los colores del tema (logo de Maxor, datos del equipo y la tira de colores).
- **Quitar una app ya no deja su carpeta olvidada**: `maxor remove` avisa de lo que dejó en tu casa (por ejemplo `~/.local/share/ATLauncher`) y `--purge` lo borra, también cuando la app ya no está instalada. En la Tienda, tras quitar una app con datos aparece la pregunta «Delete data  y / Keep  n» con las rutas y el tamaño.
- **Sin instalaciones repetidas**: una app que se está instalando, está en cola o se está quitando no se puede volver a lanzar (la fila dice `installing…`), y `maxor install` y `remove` esperan su turno si hay otra en marcha.
- **Profiles sustituye a la pestaña Setup**: marcas o desmarcas perfiles y los aplicas con `a`. El asistente sigue existiendo, pero solo con `maxor setup` (para estrenar el equipo).
- **La Tienda se maneja entera con flechas**: `↑` `↓` pasan de la caja de búsqueda a las pestañas y a la lista; con el foco en la caja todo lo que escribes es texto; `←` `→` cambian entre Results, Installed y un filtro por origen (All, nixpkgs, flathub).
- **Doctor distingue «fix» de «look»**: lo que arregla algo pide confirmación; lo que solo enseña información (`git status`, `systemctl --failed`) se abre con un `⏎`. `doctor --json` suma `kind`.
- **Themes**: la vista previa es una ventana en miniatura con la forma real de la pantalla.
- **Mejoras de la pantalla completa**:
  - **← →** (y `h` `l`, `[` `]`) cambian de pestaña además de `Tab`; la ayuda `?` suma las teclas de la pantalla en la que estás.
  - **Themes**: cada tema es un punto de su color de acento, agrupados en oscuros y claros, con una vista previa más clara.
  - **Update**: enseña la rama, los archivos sin guardar, el canal de nixpkgs y la generación; el escaneo se hace solo al abrir (se guarda en disco y se repite si cambió el repositorio o tiene más de 6 h); `r` repite el escaneo y `c` busca versiones nuevas.
  - **Doctor**: cada aviso con arreglo lo ofrece (`⏎` lo prepara, otro `⏎` lo ejecuta cediendo la terminal) y vuelve a comprobar.
  - **Store**: caja de búsqueda propia, pestañas Results/Installed, filas de dos líneas con aire.
  - **Home**: esqueleto con un brillo que se mueve y tarjetas en las que se puede pulsar.
  - La CLI suma `maxor update --status` y `--cached`, y `doctor --json` trae `id`, `fix` y `confirm` por comprobación.
- **`maxor-tui`, la pantalla completa** (Go y Bubble Tea, en `tui/`): se abre sobre tu terminal, con pestañas Home, Store, Themes, Update, Doctor y Setup; paleta de comandos con `:`, teclas vim, ratón, y al salir deja un resumen corto. Ver [TUI.md](docs/TUI.md).
  - Un solo cargador (spinner único en la barra de pestañas, nada antes de 150 ms, esqueletos mientras llegan los datos).
  - Los temas se previsualizan en toda la pantalla al moverte, sin tocar el sistema.
  - Reconstruir el sistema cede un momento la terminal a `maxor update` (se ve el progreso y funciona sudo).
  - `maxor` sin argumentos, `maxor ui [pantalla]` y `maxor setup` la abren; sin terminal, la CLI de siempre.
- **Salidas JSON para la pantalla**: `maxor doctor --json`, `theme list --json`, `update --check`/`--json` (cambios clasificados) y `profile enable|disable --no-apply`.

- **Cargadores**: `ui_run_tail` (spinner con las últimas líneas de la salida), `ui_pipeline` (pasos en secuencia), `ui_progress_*` (barra con porcentaje) y `ui_skeleton_frame` (huecos que parpadean, usado en `maxor apps`).
- **Componentes**: pestañas con subrayado, barra de estado, migas de pan, ayuda de teclas, `ui_diff` y `ui_error` (causa, qué probar y registro).
- **Registro de comandos**: la pantalla principal, `maxor help <comando>` y el autocompletado salen del mismo registro. `maxor completions fish|bash|zsh`.
- **Banderas globales** `--no-color`, `-q/--quiet`, `-v/--verbose` y `--lang`; **códigos de salida** distintos para uso, falta de algo, red, permisos y cancelación.
- **`maxor logs`, `maxor debug` y `maxor version [--json]`**: registro con rotación, informe para adjuntar a un error y versión con el schema del contrato JSON.
- **Pruebas de la CLI** (bats, 83) y de la pantalla completa (Go, ~50) y `nix flake check` que las ejecuta en el sandbox; la CI las corre en cada push. Detectaron y corrigieron dos fallos: la CLI se caía sin `LANG` y sin `/etc/os-release`.
- **`VERSION`** en la raíz: `maxor --version` la lee al compilar y `scripts/release.sh` la actualiza.
- **Apps sin rebuild**: `maxor search` es un selector interactivo (barras de progreso por origen, lista con casillas, id visible); `install`, `remove` y `apps` instalan desde nixpkgs (perfil del usuario) y Flathub (Flatpak `--user`) sin sudo, con resultados por relevancia y salida `--json` para la futura Maxor Store. Ver [APPS.md](docs/APPS.md).
- **Perfiles**: `maxor profile enable|disable` activa gaming, dev, creator u office; el equipo guarda lo activo en `hosts/<equipo>/maxor.json`.

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
