# Herramienta `maxor`

`maxor` es la línea de comandos de Maxor OS. Reúne en un solo sitio lo que el usuario hace con
más frecuencia: cambiar de tema, actualizar, instalar apps, elegir perfiles y diagnosticar el
sistema. Es también la **frontera** sobre la que se construirán las apps gráficas: cada comando
que consulta acepta `--json` (ver [APPS.md](APPS.md)).

> **Idioma.** La salida de la CLI está en español, portugués, francés, alemán, italiano e inglés
> (`--lang`, `MAXOR_LANG` o el idioma del sistema). Los mensajes en inglés viven en un catálogo
> (`home/maxor/lib/lang/en.sh`), el código solo usa claves, y cada idioma es un archivo gettext
> `home/maxor/po/<código>.po`: añadir uno no toca ningún comando ni el paquete. Cómo traducir:
> [TRANSLATING.md](TRANSLATING.md). La documentación del repositorio sigue en español.

```
maxor                             comandos agrupados, con su resumen
maxor help <comando>              ayuda de un comando
maxor --version | version [--json]

APARIENCIA
maxor theme list | current | apply <n> | undo | install <ruta> | export <n>

APPS
maxor search [texto]              buscar en nixpkgs y Flathub (selector interactivo)
maxor install <app…>              instalar sin sudo ni rebuild
maxor remove <app…> [--purge]     quitar (con --purge, también sus carpetas en tu casa)
maxor apps [list | update]        lo instalado con maxor

SISTEMA
maxor update [-y] [--no-lock]     actualizar mostrando los cambios
maxor rollback [N]                volver a la generación anterior (o a la N)
maxor doctor                      diagnóstico del sistema
maxor hardware [show | detect]    equipo detectado y drivers que usará
maxor profile [list | enable | disable] <perfil>
maxor backup [archivo]            guardar tu configuración en un solo archivo
maxor restore <archivo> [--apps]  recuperarla, en este equipo o en otro

HERRAMIENTAS
maxor logs [--last | --path]      registro de la CLI
maxor debug                       informe para adjuntar a un error
maxor completions fish|bash|zsh   autocompletado, generado desde el registro de comandos
```

## Banderas globales

Valen en cualquier comando y en cualquier posición.

| Bandera | Efecto |
|---|---|
| `--no-color` | Salida sin colores ni animaciones |
| `-q`, `--quiet` | Solo errores: se callan los éxitos |
| `-v`, `--verbose` | Escribe el registro también en stderr |
| `--lang <código>` | Idioma de los mensajes (hoy solo `en`) |

## Códigos de salida

Los scripts y las apps reaccionan al código, no al texto del error.

| Código | Significado |
|---|---|
| `0` | Todo bien |
| `1` | Falló la operación |
| `2` | Uso incorrecto: comando, opción o argumento inválido (se muestra la ayuda) |
| `3` | Falta algo: archivo, tema, flake o configuración |
| `4` | Sin red o servicio remoto caído |
| `5` | Permisos insuficientes |
| `130` | El usuario canceló (Ctrl-C) |

## La interfaz

Toda la salida es **lineal y cuelga de un riel vertical**, con la paleta del tema activo en color
de 24 bits. No imita una terminal ni dibuja marcos: no mide el ancho de cada línea, se copia y se
pega limpia y se ve igual en una terminal estrecha o en un log.

```
┌  maxor update nitro
│
◇  Building nitro  14s
│
◇  Changes
│  +1 new   ↑2 updated   −0 removed
│  ↑ firefox                  149.0 → 150.0
│
◆  Apply now? [y/N]
│
└  Cancelled. Nothing was applied.
```

| Glifo | Significa |
|---|---|
| `┌` `└` | Empieza y termina un comando (`ui_intro`, `ui_outro`) |
| `│` | El riel; una sola línea vacía entre bloques, nunca dos |
| `◇` | Un paso hecho (`ui_step`, `ui_run`) o el título de un bloque (`ui_section`) |
| `◆` | Una pregunta (`ui_confirm`) |
| `✓` `!` `✗` | Una comprobación correcta, un aviso o un problema (`ui_row`) |

En temas claros se usan variantes más oscuras de verde, ámbar y rojo para mantener el contraste.
Para lo que necesita teclado, ratón y pantalla completa (la tienda, el instalador) está
[`maxor-tui`](TUI.md); `maxor` en una terminal lo abre.

| Situación | Comportamiento |
|---|---|
| La salida no es una terminal (tubería, archivo) | El mismo texto, sin colores ni animación |
| `NO_COLOR` definida, `TERM=dumb` o `--no-color` | El mismo texto, sin colores |
| `MAXOR_ASCII=1`, `TERM=linux` o un locale que no es UTF-8 | Glifos ASCII (`+ \| o v x`) con la misma estructura |
| `MAXOR_FORCE_UI=1` | Fuerza colores y animación incluso sin terminal |
| `MAXOR_NO_TUI=1` | No abre la pantalla completa: muestra la CLI de siempre |
| `COLUMNS` | Ancho de las líneas (entre 50 y 78 columnas) |

### El cargador

Hay **uno solo**, `ui_run`, y la pantalla completa habla el mismo idioma (mismo spinner, mismos
estados, mismos tiempos). Así no hay cargadores distintos según el comando.

```
⠹  Building nitro  12s           trabajando (y los segundos si pasan de 3)
│  copying path …/firefox-150    lo último que imprime, si tarda más de 2 s
◇  Built nitro  14s              hecho (con el tiempo si pasó de 2 s)
✗  Building nitro                falló: las últimas líneas y `maxor logs --last`
```

- Nada aparece en los primeros 150 ms: las tareas rápidas no parpadean.
- Fuera de una terminal, con `NO_COLOR` o con `--quiet` no hay animación: solo el resultado.
- Un fallo se anota en el registro con el comando, el código de salida y su salida completa.
- `ui_progress_*` es lo mismo cuando se conoce el avance: añade la barra con porcentaje.
- La búsqueda en paralelo de `maxor search` usa las mismas líneas: una por origen.

### Componentes

Viven en `lib/widgets.sh` y se usan dentro del riel.

| Función | Qué dibuja |
|---|---|
| `ui_diff "texto" [filas]` | Cambios de paquetes con ↑ + − ~, totales y sin el ruido de archivos de configuración |
| `diff_json "texto"` | Lo mismo como JSON, para la pantalla completa y los scripts |
| `ui_error título causa qué-probar [log]` | Error con su árbol (`├` causa, qué probar, `└` registro) |

## Registro y diagnóstico

Todo se anota en `~/.local/state/maxor/logs/maxor.log` (rota a 512 KB).

- `maxor logs` muestra las últimas 40 líneas; `--last` el detalle de la última falla (comando,
  código de salida y la salida completa); `--path` la ruta.
- `maxor debug` escribe `maxor-debug-<fecha>.txt` con versión, sistema, hardware, `doctor` y
  registro. **Revísalo antes de compartirlo:** incluye el nombre del equipo y el hardware.

## Variables de entorno

| Variable | Valor por defecto | Para qué |
|---|---|---|
| `MAXOR_FLAKE` | `~/nixos-config` | Ruta del flake del sistema |
| `MAXOR_HOST` | nombre del equipo (`hostname`) | Host del flake que se compila |
| `MAXOR_LANG` | `LANG` (cae a `en`) | Idioma de los mensajes |
| `NO_COLOR` | sin definir | Desactiva marcos y colores |

## Temas

Referencia completa, formato y seguridad en [THEMING.md](THEMING.md). Resumen:

- `theme list` agrupa en oscuros y claros, con una muestra de colores y el activo marcado.
- `theme apply` actualiza DMS, el modo claro u oscuro, el lockscreen, la forma de Hyprland (esquinas,
  espacios, desenfoque y animaciones), el wallpaper y kitty, y
  termina con un resumen de cada paso.
- `theme install <ruta>` valida el tema y copia solo los archivos permitidos. Acepta una carpeta o
  un `.tar.gz`; rechaza enlaces, rutas peligrosas, colores inválidos y valores de forma fuera de rango.
- `theme export <nombre>` crea `<nombre>.maxortheme` en el directorio actual.

## `maxor update`

Actualiza el sistema en pasos y **no aplica nada sin preguntar**:

1. Actualiza `flake.lock` (`nix flake update`). Con `--no-lock` se salta este paso y solo se
   recompila la configuración actual.
2. Compila la nueva generación sin activarla, mostrando las últimas líneas de la compilación.
3. Compara con el sistema en marcha y muestra qué cambia: paquetes nuevos,
   actualizados, eliminados y con otro tamaño, y un aviso si incluye un kernel nuevo.

Después pregunta si aplicarla. Con `-y` aplica sin preguntar. Si el sistema ya coincide con lo
compilado, lo dice y termina. Aplicar usa `sudo nixos-rebuild switch`, así que pedirá tu contraseña.

Si cancelas después de actualizar `flake.lock`, el archivo queda modificado: revísalo con
`git diff flake.lock` o restáuralo con `git checkout flake.lock`.

## Variables de entorno de la pantalla completa

- `MAXOR_SUDO_STDIN=1`: los comandos que necesitan root (`update`, `rollback`) usan `sudo -S -p ''` y leen la contraseña de la entrada estándar. Es lo que usa la pantalla completa; en una terminal no hace falta.

## `maxor backup` y `maxor restore`

Guardan lo que es tuyo y no sale de git ni de Nix en un solo `.tar.gz`:

| Dentro | Qué es |
|---|---|
| `manifest.json` | versión de maxor, equipo, fecha, tema en uso, perfiles y apps instaladas |
| `hosts/maxor.json` | los perfiles activos del equipo |
| `themes/<id>/` | los temas que hiciste tú (los oficiales ya vienen con Maxor) |
| `dms/` | los ajustes de DankMaterialShell (barra, lanzador…) |

`hardware.json` no va: es de cada equipo y se vuelve a detectar solo.

- `maxor backup [archivo|carpeta] [--json]` guarda la copia (por defecto `~/maxor-backup-<equipo>-<fecha>.tar.gz`); `--json` da `{path, bytes, apps, themes, host}`. `maxor backup --list <archivo> [--json]` cuenta lo que lleva sin tocar nada.
- `maxor restore <archivo> [--apps] [-y]` enseña lo que hay, pide confirmación y recupera perfiles, temas propios (nunca pisa uno que ya existe), ajustes de la barra (guarda los de antes como `settings.json.before-restore`) y el tema en uso. Con `--apps` instala también las apps de la copia. Solo extrae rutas conocidas y rechaza archivos con rutas absolutas o `..`.
- Después de recuperar los perfiles, aplícalos con `maxor update --no-lock`.

## `maxor rollback`

Enseña las últimas generaciones y, tras confirmar, ejecuta
`sudo nixos-rebuild switch --rollback`. Con `-y` no pregunta. Si el sistema ni siquiera
arranca, elige la generación anterior en el menú de arranque.

- `maxor rollback N` vuelve a una generación concreta (`nix-env --switch-generation` y `switch-to-configuration`).
- `maxor rollback --list [--json]` solo las enseña; con `--json` da `[{generation, date, nixos, kernel, current}]` (las 12 últimas). La pestaña Update de la pantalla completa lo usa con `g`.

## `maxor doctor`

Revisa el sistema por áreas. Devuelve código de salida `1` solo si hay problemas, así que sirve
en scripts.

| Área | Qué comprueba |
|---|---|
| Sistema | Versión, servicios fallidos (sistema y usuario), kernel instalado frente al que corre |
| Sesión | Wayland y Hyprland, servicios `dms`, `hypridle` y portales, servicio PAM de hyprlock |
| Gráficos | GPU detectadas, `nvidia-offload` y respuesta del controlador NVIDIA |
| Arranque y disco | `/boot` y `/efi` montados, y espacio libre en `/boot`, `/efi` y `/` |
| Identidad | Fuentes de Maxor, tema activo y tema de DMS generado |
| Configuración | Flake presente, repositorio sin cambios pendientes y `hardware.json` al día |

Úsalo antes de abrir un issue, o usa `maxor debug`, que lo incluye.

## Arquitectura

```
home/maxor/
├── lib/
│   ├── core.sh        rutas, códigos de salida, registro, errores, nombre del sistema
│   ├── i18n.sh        msg / t: mensajes por clave, plurales y carga de los .po
│   ├── lang/en.sh     catálogo de mensajes (inglés); los demás idiomas están en po/
│   ├── term.sh        capacidades, paleta del tema, texto, repintado en el sitio
│   ├── frame.sh       el riel, las líneas, las filas y los mensajes
│   ├── loaders.sh     el cargador único (ui_run) y la barra de avance
│   ├── widgets.sh     diff y errores
│   ├── style.sh       style.json → Lua de Hyprland
│   └── registry.sh    registro de comandos, ayuda y errores de uso
├── cmd/               un archivo por grupo de comandos
│   ├── theme.sh  update.sh  doctor.sh  hardware.sh
│   └── apps.sh   profile.sh  tools.sh
└── main.sh            banderas globales y despacho
```

El paquete es `packages/maxor.nix`: concatena los archivos en ese orden y `shellcheck` revisa el
resultado en cada compilación. `home/maxor.nix` lo instala junto con los temas oficiales.

### Cómo añadir un comando

1. Crea `home/maxor/cmd/<nombre>.sh` con `maxor_cmd <nombre> <grupo> "subcomandos"` y una función
   `cmd_<nombre>`.
2. Añade el archivo a la lista de `packages/maxor.nix`.
3. En `lib/lang/en.sh` añade `cmd.<nombre>` (resumen de una línea) y `help.<nombre>` (el texto
   de `maxor help <nombre>`), y las claves de los mensajes que uses.

Con eso aparece solo en la pantalla principal, en `maxor help`, en el autocompletado y en la
comprobación de las pruebas. Usa `usage_error <nombre>` para un uso incorrecto y `die_code` con un
código de `EX_*` para los fallos.

### Mensajes

El código nunca lleva texto de cara al usuario: usa `@clave` donde iría el texto.

```bash
ui_row ok @doctor.kernel_ok                 # sin argumentos
ui_say warn @apps.not_installed "$id"       # con argumentos para el formato
msg etiqueta @update.step_build "$host"     # a una variable, para componer líneas
die_code "$EX_NEEDS" @theme.missing "$name" # error con código de salida
```

Un texto que no empieza por `@` se imprime tal cual. Una clave que falta se muestra como la
propia clave, y las pruebas fallan si el código usa una clave que no está en el catálogo.

## Rendimiento de la interfaz

La CLI pinta muchas líneas por comando y el cargador se repinta varias veces por segundo. Por eso
`lib/term.sh` y `lib/frame.sh` siguen una regla: **en los caminos que se repiten no se lanzan
procesos** (nada de `sed`, `wc`, `cat` ni `$(…)` por línea).

- El largo visible de un texto con colores se calcula con una expansión de bash (`ui_len`), no con `sed | wc`.
- Los ayudantes con variante `…v` (`ui_repv`, `ui_truncv`, `ui_fgv`) dejan el resultado en una variable en vez de imprimirlo, para no crear subprocesos.
- Los archivos pequeños de `/sys` se leen con `$(< archivo)` o `hw_read`, no con `cat`.
- Una consulta de `jq` por lote de datos, no una por elemento (por ejemplo `maxor theme list`).
- Para animar varias líneas se usa `ui_paint`: sube las líneas del marco anterior y sobrescribe
  cada una, en salida sincronizada, sin borrar antes la pantalla.

El selector de `maxor search` además apaga el eco del terminal mientras dura y procesa juntas las
teclas acumuladas (mantener una flecha).

Tiempos en el Nitro, antes y después: ayuda 352 → 108 ms, `theme list` 1562 → 271 ms,
`profile list` 184 → 67 ms, `hardware` 367 → 135 ms, `doctor` unos 3 s → 0,5 s (en caliente).

## Pruebas

`tests/` contiene pruebas con [bats](https://github.com/bats-core/bats-core):

| Archivo | Qué cubre |
|---|---|
| `i18n.bats` | Todas las claves usadas existen, ninguna sobra, cada comando tiene resumen y ayuda |
| `ui.bats` | El riel (estructura exacta, sin líneas vacías dobles), con y sin colores, respaldo ASCII, recorte, repintado |
| `widgets.bats` | `ui_diff` y `diff_json` (conteos, ruido de configuración, colores de nix), `ui_error` |
| `loaders.bats` | El cargador único: salida, fallo con código y registro, riel en secuencia, `--quiet` |
| `core.bats` | Códigos de salida, registro y rotación, validación de temas y de forma |
| `cli.bats` | El binario completo: versión, ayuda, códigos, JSON, autocompletado, sin `LANG` |
| `release.bats` | Extracción de notas del CHANGELOG |

```
nix build .#checks.x86_64-linux.cli-tests --print-build-logs    # en el sandbox, como en la CI
MAXOR_BIN=$(which maxor) nix run nixpkgs#bats -- tests/        # rápido, con tu binario
```

La CI ejecuta la primera en cada push a `main` y `development` y en cada Pull Request.
