# Herramienta `maxor`

`maxor` es la línea de comandos de Maxor OS. Reúne en un solo sitio lo que el usuario hace con
más frecuencia: cambiar de tema, actualizar, volver atrás y diagnosticar el sistema.

```
maxor theme list                  temas instalados
maxor theme current               tema activo
maxor theme apply <nombre>        aplicar un tema
maxor theme undo                  volver al anterior
maxor theme install <ruta>        instalar desde carpeta o .tar.gz
maxor theme export <nombre>       empaquetar un tema

maxor update [-y] [--no-lock]     actualizar mostrando los cambios
maxor rollback [-y]               volver a la generación anterior
maxor doctor                      diagnóstico del sistema
maxor hardware [detect [--write]]  equipo detectado y drivers que usará
maxor search/install/remove/apps   apps de nixpkgs y Flathub (ver APPS.md)
maxor profile [enable|disable] n  perfiles: gaming, dev, creator, office
```

## La interfaz

Cada comando se dibuja como **una ventana dentro de la terminal**: marco redondeado, barra de
título con tres puntos y superficie propia, todo con la paleta del tema activo en color de 24
bits. Cambias de tema y la propia terminal de `maxor` cambia de color en el mismo instante.

```
╭ ● ● ●  maxor · temas                                                       ╮
│                                                                            │
│ OSCUROS                                                                    │
│  ● sakura        ████████  Sakura nocturna                                 │
│    glaciar       ████████  Glaciar                                         │
│ …                                                                          │
╰────────────────────────────────────────────────────────────────────────────╯
```

Las comprobaciones llevan ✓ (correcta), `!` (aviso) o ✗ (problema); las operaciones largas
muestran un spinner y los resultados finales, una etiqueta de color (`OK`, `AVISOS`, `REVISAR`).
En temas claros se usan variantes más oscuras de verde, ámbar y rojo para mantener el contraste.

| Situación | Comportamiento |
|---|---|
| La salida no es una terminal (tubería, archivo) | Texto plano, sin marcos ni colores |
| `NO_COLOR` definida, o `TERM=dumb` | Texto plano |
| `MAXOR_FORCE_UI=1` | Fuerza la interfaz incluso sin terminal |
| `COLUMNS` | El ancho de la ventana (entre 50 y 78 columnas) |

La interfaz está en `home/maxor/ui.sh` y es la base que reutilizará el instalador de la fase 6.

## Variables de entorno

| Variable | Valor por defecto | Para qué |
|---|---|---|
| `MAXOR_FLAKE` | `~/nixos-config` | Ruta del flake del sistema |
| `MAXOR_HOST` | nombre del equipo (`hostname`) | Host del flake que se compila |
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

Actualiza el sistema en tres pasos y **no aplica nada sin preguntar**:

1. Actualiza `flake.lock` (`nix flake update`). Con `--no-lock` se salta este paso y solo se
   recompila la configuración actual.
2. Compila la nueva generación sin activarla, con un spinner.
3. Muestra en una ventana qué paquetes cambian (`nix store diff-closures`) y pregunta si aplicarla.

Con `-y` aplica sin preguntar. Si el sistema ya coincide con lo compilado, lo dice y termina.
Aplicar usa `sudo nixos-rebuild switch`, así que pedirá tu contraseña.

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
| Configuración | Flake presente y repositorio sin cambios pendientes |

Úsalo antes de abrir un issue: la salida es lo primero que se pedirá.

## Dónde está el código

| Archivo | Contenido |
|---|---|
| `home/maxor.nix` | Empaquetado, temas oficiales y wallpapers |
| `home/maxor/lib.sh` | Rutas y utilidades de color |
| `home/maxor/ui.sh` | Ventanas, filas, spinner, confirmaciones |
| `home/maxor/style.sh` | `style.json` → Lua de Hyprland (validación y generación) |
| `home/maxor/theme.sh` | `maxor theme …` |
| `home/maxor/system.sh` | `update`, `rollback` y `doctor` |
| `home/maxor/hardware.sh` | `hardware`: detección del equipo |
| `home/maxor/apps.sh` | `search`, `install`, `remove` y `apps` |
| `home/maxor/profile.sh` | `profile` |
| `home/maxor/main.sh` | Ayuda y despacho de comandos |

Los scripts se concatenan en ese orden y pasan `shellcheck` en cada compilación (lo ejecuta
`writeShellApplication`).
