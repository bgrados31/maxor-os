# Herramienta `maxor`

`maxor` es la línea de comandos de Maxor OS. Reúne en un solo sitio lo que el usuario hace con
más frecuencia: cambiar de tema, actualizar, volver atrás y diagnosticar el sistema.

```
maxor theme list              temas instalados
maxor theme current           tema activo
maxor theme apply <nombre>    aplicar un tema
maxor theme undo              volver al tema anterior

maxor update [-y] [--no-lock] actualizar el sistema
maxor rollback [-y]           volver a la generación anterior
maxor doctor                  diagnóstico del sistema
```

## Variables de entorno

| Variable | Valor por defecto | Para qué |
|---|---|---|
| `MAXOR_FLAKE` | `~/nixos-config` | Ruta del flake del sistema |
| `MAXOR_HOST` | nombre del equipo (`hostname`) | Host del flake que se compila |

## `maxor update`

Actualiza el sistema en tres pasos y **no aplica nada sin preguntar**:

1. Actualiza `flake.lock` (`nix flake update`). Con `--no-lock` se salta este paso y solo se
   recompila la configuración actual.
2. Compila la nueva generación sin activarla.
3. Muestra qué paquetes cambian (`nix store diff-closures`) y pregunta si aplicarla.

Con `-y` aplica sin preguntar. Si el sistema ya coincide con lo compilado, lo dice y termina.
Aplicar usa `sudo nixos-rebuild switch`, así que pedirá tu contraseña.

Si cancelas después de actualizar `flake.lock`, el archivo queda modificado: revísalo con
`git diff flake.lock` o restáuralo con `git checkout flake.lock`.

## `maxor rollback`

Enseña las últimas generaciones y, tras confirmar, ejecuta
`sudo nixos-rebuild switch --rollback`. Con `-y` no pregunta. Si el sistema ni siquiera
arranca, elige la generación anterior en el menú de arranque.

## `maxor doctor`

Revisa el sistema por áreas y marca cada comprobación como correcta (✓), aviso (!) o problema
(✗). Devuelve código de salida `1` solo si hay problemas, así que sirve en scripts.

| Área | Qué comprueba |
|---|---|
| Sistema | Versión, servicios fallidos (sistema y usuario), kernel instalado frente al que corre |
| Sesión | Wayland y Hyprland, servicios `dms`, `hypridle` y portales, servicio PAM de hyprlock |
| Gráficos | GPU detectadas, `nvidia-offload` y respuesta del controlador NVIDIA |
| Arranque y disco | `/boot` y `/efi` montados, y espacio libre en `/boot`, `/efi` y `/` |
| Identidad | Fuentes de Maxor y tema activo |
| Configuración | Flake presente y repositorio sin cambios pendientes |

Úsalo antes de abrir un issue: la salida es lo primero que se pedirá.

## Temas

Los comandos `maxor theme …` se describen en [THEMING.md](THEMING.md).

## Dónde está el código

- `home/maxor.nix`: empaquetado, temas oficiales y comandos de tema.
- `home/maxor/system.sh`: `update`, `rollback` y `doctor`.

El script pasa `shellcheck` en cada compilación (lo ejecuta `writeShellApplication`).
