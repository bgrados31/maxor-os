# Herramienta `maxor`

`maxor` es la línea de comandos de Maxor OS. Reúne en un solo sitio lo que el usuario hace con
más frecuencia: cambiar de tema, actualizar, instalar apps, elegir perfiles y diagnosticar el
sistema. Es también la **frontera** sobre la que se construirán las apps gráficas: cada comando
que consulta acepta `--json` (ver [APPS.md](APPS.md)).

> **Idioma.** Hoy la salida de la CLI está en inglés. Los mensajes viven en un catálogo
> (`home/maxor/lib/lang/en.sh`) y el código solo usa claves, de modo que añadir español es crear
> `lang/es.sh`, sin tocar ningún comando. La documentación del repositorio sigue en español.

```
maxor                             comandos agrupados, con su resumen
maxor help <comando>              ayuda de un comando
maxor --version | version [--json]

APARIENCIA
maxor theme list | current | apply <n> | undo | install <ruta> | export <n>

APPS
maxor search [texto]              buscar en nixpkgs y Flathub (selector interactivo)
maxor install <app…>              instalar sin sudo ni rebuild
maxor remove <app…>               quitar
maxor apps [list | update]        lo instalado con maxor

SISTEMA
maxor update [-y] [--no-lock]     actualizar mostrando los cambios
maxor rollback [-y]               volver a la generación anterior
maxor doctor                      diagnóstico del sistema
maxor hardware [show | detect]    equipo detectado y drivers que usará
maxor profile [list | enable | disable] <perfil>

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

Cada comando se dibuja como **una ventana dentro de la terminal**: marco redondeado, barra de
título con tres puntos y superficie propia, todo con la paleta del tema activo en color de 24
bits. Cambias de tema y la propia terminal de `maxor` cambia de color en el mismo instante.

```
╭ ● ● ●  maxor · themes                                                      ╮
│                                                                            │
│ DARK                                                                       │
│  ● sakura        ████████  Sakura nocturna                                 │
│    glaciar       ████████  Glaciar                                         │
│ …                                                                          │
╰────────────────────────────────────────────────────────────────────────────╯
```

Las comprobaciones llevan ✓ (correcta), `!` (aviso) o ✗ (problema). En temas claros se usan
variantes más oscuras de verde, ámbar y rojo para mantener el contraste.

| Situación | Comportamiento |
|---|---|
| La salida no es una terminal (tubería, archivo) | Texto plano, sin marcos, colores ni animación |
| `NO_COLOR` definida, `TERM=dumb` o `--no-color` | Texto plano |
| `MAXOR_FORCE_UI=1` | Fuerza la interfaz incluso sin terminal |
| `COLUMNS` | Ancho de la ventana (entre 50 y 78 columnas) |

### Cargadores

Cada espera tiene el suyo. Todos animan solo en una terminal con color; en cualquier otro caso
ejecutan igual y escriben líneas simples.

| Función | Para qué | Dónde se usa |
|---|---|---|
| `ui_run` | Una tarea corta, con spinner | pasos sueltos de varios comandos |
| `ui_run_tail` | Una tarea larga: spinner y las últimas 4 líneas de su salida | compilaciones |
| `ui_pipeline` | Varias tareas en secuencia: lista de pasos, «paso N de M» y tiempo total; muestra las últimas líneas del paso en curso | `maxor update` |
| `ui_progress_*` | Avance conocido: barra con porcentaje | listo para copias y descargas |
| `ui_skeleton_frame` | Huecos que parpadean mientras llegan los datos | `maxor apps` |
| búsqueda en paralelo | Una barra por origen | `maxor search` |

Un fallo en cualquiera de ellos se anota en el registro (con el comando y su salida completa) y
muestra la causa más `maxor logs --last`.

### Componentes

Viven en `lib/widgets.sh` y se usan dentro de una ventana (entre `ui_open` y `ui_close`).

| Función | Qué dibuja |
|---|---|
| `ui_tabs ACTIVA etiqueta…` | Pestañas con subrayado bajo la activa |
| `ui_status dato…` | Barra de estado: `dato • dato • dato` |
| `ui_crumbs nivel…` | Migas de pan: `maxor › help › theme` |
| `ui_hints tecla:acción…` | Ayuda de teclas para pantallas interactivas |
| `ui_diff "texto" [filas]` | Cambios de paquetes con ↑ + − ~, totales y sin el ruido de archivos de configuración |
| `ui_error título causa qué-probar [log]` | Error con su árbol: causa, qué probar y, si se pide, dónde está el registro |

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
  termina con una ventana que resume cada paso.
- `theme install <ruta>` valida el tema y copia solo los archivos permitidos. Acepta una carpeta o
  un `.tar.gz`; rechaza enlaces, rutas peligrosas, colores inválidos y valores de forma fuera de rango.
- `theme export <nombre>` crea `<nombre>.maxortheme` en el directorio actual.

## `maxor update`

Actualiza el sistema en pasos y **no aplica nada sin preguntar**:

1. Actualiza `flake.lock` (`nix flake update`). Con `--no-lock` se salta este paso y solo se
   recompila la configuración actual.
2. Compila la nueva generación sin activarla, mostrando las últimas líneas de la compilación.
3. Compara con el sistema en marcha y muestra en una ventana qué cambia: paquetes nuevos,
   actualizados, eliminados y con otro tamaño, y un aviso si incluye un kernel nuevo.

Después pregunta si aplicarla. Con `-y` aplica sin preguntar. Si el sistema ya coincide con lo
compilado, lo dice y termina. Aplicar usa `sudo nixos-rebuild switch`, así que pedirá tu contraseña.

Si cancelas después de actualizar `flake.lock`, el archivo queda modificado: revísalo con
`git diff flake.lock` o restáuralo con `git checkout flake.lock`.

## `maxor rollback`

Enseña las últimas generaciones en una ventana y, tras confirmar, ejecuta
`sudo nixos-rebuild switch --rollback`. Con `-y` no pregunta. Si el sistema ni siquiera
arranca, elige la generación anterior en el menú de arranque.

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
│   ├── i18n.sh        msg / t: mensajes por clave
│   ├── lang/en.sh     catálogo de mensajes (inglés)
│   ├── term.sh        capacidades, paleta del tema, texto, repintado en el sitio
│   ├── frame.sh       ventana, líneas, filas y mensajes
│   ├── loaders.sh     spinner, ventana de salida, pasos, barra, esqueleto
│   ├── widgets.sh     pestañas, estado, migas, teclas, diff y errores
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

La CLI pinta ventanas de terminal y, en el buscador, se repinta en cada tecla. Por eso
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
| `ui.bats` | Ancho de las ventanas con y sin colores, texto plano, recorte, repintado |
| `widgets.bats` | `ui_diff` (conteos, ruido de configuración, colores de nix), `ui_error`, pestañas |
| `loaders.bats` | Pasos en orden y su salida, fallo con código, registro del error, `--quiet` |
| `core.bats` | Códigos de salida, registro y rotación, validación de temas y de forma |
| `cli.bats` | El binario completo: versión, ayuda, códigos, JSON, autocompletado, sin `LANG` |
| `release.bats` | Extracción de notas del CHANGELOG |

```
nix build .#checks.x86_64-linux.cli-tests --print-build-logs    # en el sandbox, como en la CI
MAXOR_BIN=$(which maxor) nix run nixpkgs#bats -- tests/        # rápido, con tu binario
```

La CI ejecuta la primera en cada push a `main` y `development` y en cada Pull Request.
