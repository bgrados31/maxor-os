# La pantalla completa (`maxor-tui`)

`maxor-tui` es la pantalla propia de Maxor. Se abre **sobre tu terminal** (en la pantalla
alterna, como un instalador), tú eliges y escribes, y al salir vuelves a tu terminal con todo lo
anterior intacto y un resumen corto de lo que hiciste. No abre otra ventana ni otra terminal, y no
imita una terminal: dibuja su propia interfaz con la paleta del tema activo.

```
maxor                 # sin argumentos, en una terminal: abre la pantalla
maxor ui [pantalla]   # home, store, themes, update, doctor o profiles
maxor setup           # el asistente de primer arranque, sin pestañas (solo para estrenar el equipo)
maxor search [texto]  # abre la Tienda (con la búsqueda ya lanzada)
```

Con `MAXOR_NO_TUI=1`, sin terminal o sin el programa instalado, esos comandos caen a la CLI de
siempre (la lista de comandos, o los resultados como lista), así que nada depende de la pantalla.

## Pantallas

| Pestaña | Qué hace | Comandos de `maxor` que usa |
|---|---|---|
| **Home** | El estado del equipo de un vistazo (sistema, actualizaciones, apps, tema, hardware) y las acciones rápidas | `doctor --json`, `apps --json`, `theme list --json`, `hardware detect` |
| **Store** | Buscar en nixpkgs y Flathub, marcar varias apps, instalar y quitar | `search --json`, `install --json`, `remove --json`, `apps --json` |
| **Themes** | Ver los temas; al moverte, **toda la pantalla se pinta con ese tema** sin tocar el sistema; con Intro se aplica | `theme list --json`, `theme apply`, `theme undo` |
| **Update** | Rama y canal de la configuración, escaneo automático de lo que cambiaría (sin aplicar nada), rescan y aplicar | `update --status`, `update --cached`, `update --json [--no-lock]`, `update --no-lock -y` |
| **Doctor** | Las comprobaciones agrupadas. Cada aviso ofrece un comando fijo que decide la CLI: **fix** (arregla algo; `⏎` lo prepara y otro `⏎` lo ejecuta) o **look** (solo enseña información, un `⏎` basta). No hay IA: son comandos que se ven antes de ejecutarlos | `doctor --json` (campos `id`, `fix`, `confirm`, `kind`) |
| **Profiles** | Marcar o desmarcar perfiles (juegos, desarrollo…) y aplicarlos con `a`; es lo que antes hacía el asistente, pero para el día a día | `profile list --json`, `profile enable\|disable --no-apply`, `update --no-lock -y` |

La pantalla **no reimplementa nada**: llama a la CLI y lee su JSON (el contrato está en
[APPS.md](APPS.md) y [CLI.md](CLI.md)). Si algo falla, el código de salida y la última línea de
`stderr` de la CLI son lo que se enseña, con la pista `maxor logs --last`.

## Teclas

| Tecla | Acción |
|---|---|
| `↑` `↓`, `j` `k` | Mover (también `g`/`G` principio y fin, `Ctrl-u`/`Ctrl-d` media página) |
| `←` `→`, `h` `l`, `[` `]`, `Tab`, `Shift-Tab`, `1` … `6` | Cambiar de pestaña (en el asistente y dentro de un campo de texto no cambian) |
| `⏎` | Elegir o ejecutar |
| `espacio` | Marcar (Tienda: casillas en Installed y en los resultados; Setup) |
| `a`, `u`, `U`, `r` | Tienda, con apps instaladas: marcar todas, actualizar las marcadas, actualizar todas, quitar las marcadas |
| `R` | Tienda: volver a mirar las versiones nuevas de las apps |
| `g` | Update: abrir las generaciones del sistema para volver a una anterior |
| `r` / `c` | Update: repetir el escaneo / buscar versiones nuevas. Doctor: comprobar otra vez |
| `/` o `↑` | Buscar (Tienda). Con `↑` y `↓` se recorren la caja, las pestañas Results/Installed/origen y la lista; `←` `→` cambian de pestaña dentro de la Tienda |
| `D` | Mostrar u ocultar los detalles (al lado, o abajo si la ventana es estrecha); se recuerda |
| `:` | **Paleta de comandos**: escribe `go store`, `theme dawn`, `search brave`, `update check`… |
| `?` | Ayuda con todas las teclas |
| `q`, `Ctrl-C` | Volver a tu terminal (la pestaña **Exit** enseña lo pendiente y sale con `⏎`; también `exit` en la paleta) |

**Ventanas estrechas.** Con menos de 96 columnas los detalles no caben al lado y se dibujan en un cajón compacto debajo (hasta 64×20 de mínimo). Cada pantalla decide qué es lo esencial con `Brief` (opcional en `core.Screen`); si no la tiene, se usa su panel lateral sin huecos.

El **ratón** funciona (opcional con `--no-mouse`): clic en una pestaña o una fila y la rueda para
desplazar. Mientras hay un campo de texto con el foco, las teclas globales (`q`, `:`, `?`…) son
texto, no atajos.

## Un solo cargador

Toda espera pasa por el mismo sistema (`internal/task`), con las reglas de la CLI:

- el **mismo spinner** (⠋⠙⠹…), la misma etiqueta y los mismos segundos, en una única posición: la
  esquina de la barra de pestañas, sea cual sea la pestaña;
- **nada aparece antes de 150 ms**: lo rápido no parpadea;
- mientras llegan los datos de una lista se ven los **huecos que parpadean** (esqueleto) y no un
  panel vacío;
- una tarea nueva con el mismo ID **reemplaza** a la anterior y su resultado viejo se ignora
  (buscar «bra» y luego «brave» no muestra lo de «bra»);
- los fallos se enseñan con el mismo glifo `✗`, la causa y `maxor logs --last`.

## Aplicar el sistema sin salir de la pantalla

Reconstruir el sistema necesita `sudo` y puede tardar. **Update** (aplicar y volver a una generación anterior) y **Profiles** lo hacen dentro de la pantalla:

1. si sudo ya está listo no pide nada; si no, un campo propio pide la contraseña (se dibuja con puntos);
2. la contraseña se entrega **solo a sudo, por la entrada estándar** (`MAXOR_SUDO_STDIN=1`): no se guarda, no se escribe en ningún archivo ni aparece en el registro;
3. mientras trabaja se ve el progreso real (construir, activar, reiniciar lo que cambió) y la salida del comando; no se puede salir hasta que termina, para no dejar el sistema a medias;
4. si la contraseña no vale, la vuelve a pedir; si falla, enseña el código y `maxor logs --last`.

Si prefieres teclear la contraseña en la terminal, en Update pulsa `t`: cede la terminal a `maxor update` como antes. El asistente de primer arranque (`maxor setup`) sigue usando la terminal.

## Al salir

Se imprime en tu terminal normal un resumen de 2 a 4 líneas, con los mismos glifos de la CLI:

```
◇  Installed brave
◇  Applied theme dawn
·  Profiles saved: apply them with maxor update --no-lock
```

Si no hiciste nada, no deja nada.

## Idioma

Todo el texto de la pantalla y del instalador está traducido (español, portugués, francés, alemán e italiano; el
inglés es el original). La pantalla toma el idioma de `MAXOR_LANG`, `LC_ALL` o `LANG` al arrancar, y el instalador
cambia al idioma que se elige en su primer paso. Le pide a la CLI que conteste en el mismo idioma, así lo que
muestra Doctor o la Tienda coincide con el resto. Los textos con número usan la regla de plurales de cada idioma,
y los catálogos son archivos gettext (`tui/internal/i18n/lang/*.po`): cómo traducir o añadir un idioma está en
[TRANSLATING.md](TRANSLATING.md). Las pruebas recorren cada pestaña y cada paso del instalador en todos los
idiomas, y en un pseudo-idioma un 40 % más largo, para que ningún texto largo rompa la pantalla.

## Cómo está hecha

Go y [Bubble Tea](https://github.com/charmbracelet/bubbletea) con Lip Gloss, en `tui/`. Es un
programa **aparte** de la CLI de bash: la CLI sigue siendo la fuente de la lógica y la pantalla es
una cara más, igual que lo serán las apps gráficas.

```
tui/
├── main.go                  banderas, comprobación de terminal, resumen al salir
└── internal/
    ├── theme/               lee el tema activo (el mismo que la CLI) o Maxor Dark
    ├── maxor/               cliente de la CLI: cada llamada con --json, con tipos
    ├── task/                el cargador único
    ├── ui/                  glifos (con respaldo ASCII), estilos, filas, paneles, campo de texto
    ├── core/                Env, la interfaz Screen y los mensajes entre pantallas
    ├── screens/             Home, Store, Themes, Update, Doctor, Setup
    └── app/                 el modelo raíz: pestañas, paleta, ayuda, ratón, vista
```

Dos decisiones que conviene conocer:

- **El dibujo es por filas de ancho exacto.** Cada fila se arma con trozos de texto, cada uno con
  su fondo, y se rellena hasta el ancho del panel. Así no quedan huecos con el fondo de la terminal
  y la vista siempre mide exactamente el alto y el ancho de la terminal (hay pruebas para cinco
  tamaños).
- **Bubble Tea consulta a la terminal su color de fondo al arrancar** y espera hasta 5 s si no
  responde (consola de Linux, algunos SSH). Maxor usa solo colores explícitos, así que
  `packages/maxor-tui.nix` quita esa consulta al compilar. Si una versión nueva de Bubble Tea
  cambia esa línea, la compilación falla en vez de volver a tardar.

### Añadir una pantalla

1. Crea `tui/internal/screens/<nombre>.go` con un tipo que cumpla `core.Screen` (`ID`, `Title`,
   `Init`, `Update`, `Main`, `Side`, `Hints`, `Captures`, `Click`, `Wheel`; `core.Base` da los
   que casi nadie necesita).
2. Las esperas, siempre con `env.Tasks.Start(task.Task{ID: "<nombre>.<algo>", …})`: el texto antes
   del primer punto es la pantalla que recibe el resultado.
3. Para llamar a la CLI, añade el método tipado en `internal/maxor/client.go` y su prueba.
4. Regístrala en `app.New` (`internal/app/model.go`).

## Pruebas

`go test ./...` corre en cada compilación del paquete (`nix build .#maxor-tui`) y con
`nix flake check`:

| Paquete | Qué cubre |
|---|---|
| `ui` | Filas de ancho exacto (también con caracteres anchos), recorte, paneles, glifos y su respaldo ASCII, barra, esqueleto, campo de texto |
| `task` | Reemplazo de tareas, resultados viejos ignorados, nada antes de 150 ms, orden del indicador |
| `maxor` | Cada llamada con sus argumentos, códigos de salida, JSON inválido, el ejecutor real y su entorno |
| `app` | Con una CLI de mentira: la vista mide siempre lo mismo (5 tamaños × 6 pantallas), pestañas y ratón, paleta, buscar-marcar-instalar con resumen, vista previa de temas, el asistente de principio a fin, errores sin caerse |

Para ver el diseño sin abrir una terminal: `MAXOR_TUI_SHOW=1 go test ./internal/app -run TestShow -v`
imprime cada pantalla en texto plano.
