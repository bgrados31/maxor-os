# Apps y perfiles

Instalar software en Maxor OS no exige editar `.nix` ni reconstruir el sistema. Hay dos vías,
las dos desde `maxor`, y una tercera —los perfiles— para configurar el equipo por tipo de uso.

## Apps de usuario (sin sudo, sin rebuild)

```
maxor search <texto>              buscar en nixpkgs y Flathub
maxor install <app…>              instalar
maxor remove <app…>               quitar
maxor apps                        listar lo instalado con maxor
maxor apps update                 actualizar nixpkgs y Flatpak
```

| Origen | Cómo se instala | Dónde |
|---|---|---|
| `nix` | `nix profile add nixpkgs#<paquete>` | perfil del usuario |
| `flatpak` | `flatpak install --user flathub <id>` | `~/.local/share/flatpak` |

Sin indicarlo, un ID con tres partes o más separadas por puntos (`org.mozilla.firefox`) es
Flatpak y cualquier otro nombre (`btop`) es nixpkgs. Se fuerza con `--nix` o `--flatpak`.
Los paquetes con licencia no libre (Steam, Spotify…) se pueden instalar.

## El buscador

`maxor search` busca en nixpkgs y Flathub a la vez (una barra de progreso por origen) y muestra una
sola lista con casillas. Cada resultado trae el nombre y el origen en la primera línea, y su `id`
(lo que se pasa a `maxor install`) en la segunda.

| Tecla | Acción |
|---|---|
| `↑` `↓` (o `k` `j`) | mover el cursor |
| `espacio` | marcar o desmarcar |
| `Intro` | instalar lo marcado (si no hay nada marcado, lo resaltado) |
| `q` o `Esc` | salir sin instalar |

Las apps ya instaladas aparecen con `✓` y no se pueden marcar. Sin texto de búsqueda en una terminal,
pregunta qué app buscas. Con `--list`, o si la salida no es una terminal, imprime la lista sin
selección.

Los resultados van ordenados por relevancia: exacto, empieza por, contiene y, al final, los que solo
coinciden en la descripción (que solo se muestran si hay muy pocos mejores). De nixpkgs solo salen
los paquetes de primer nivel; `tests.*`, `haskellPackages.*` y similares son piezas internas.

Lo instalado así es **del usuario**: no entra en el flake ni en las generaciones del sistema, así
que `maxor rollback` no lo deshace. Para algo que quieras en el sistema y en el repositorio, usa
un perfil o edita tu `configuration.nix`.

## Perfiles

Un perfil agrupa paquetes y servicios de sistema por tipo de uso.

```
maxor profile                     listar y ver cuáles están activos
maxor profile enable <nombre>     activar (muestra los cambios y pide confirmar)
maxor profile disable <nombre>    desactivar
```

| Perfil | Incluye |
|---|---|
| `gaming` | Steam, Proton, GameMode, MangoHud, Gamescope, Lutris |
| `dev` | C, Node, Python, Go, Rust, Git, GitHub CLI, Podman (con `docker`), direnv |
| `creator` | OBS, GIMP, Inkscape, Krita, Kdenlive, Blender, Audacity, HandBrake |
| `office` | LibreOffice, diccionarios es/en, Thunderbird, Okular |

Lo activo se guarda en `hosts/<equipo>/maxor.json` y `modules/profiles.nix` lo traduce a
paquetes. `enable` y `disable` equivalen a editar ese archivo y ejecutar `maxor update --no-lock`;
si cancelas la confirmación, el archivo ya cambió y se aplicará en la próxima actualización.
Los nombres y descripciones viven en `modules/profiles-catalog.json`; añadir un perfil es una
entrada ahí más su bloque en `modules/profiles.nix`.

## Contrato JSON (para Maxor Store y otras apps)

Los comandos que consultan aceptan `--json` y entonces **solo** escriben JSON en la salida
estándar, sin colores ni mensajes.

| Comando | Salida |
|---|---|
| `maxor search <texto> --json` | `[{source, id, name, version, description}]` en una sola lista por relevancia, nixpkgs y Flathub mezclados (máx. 14) |
| `maxor apps --json` | `[{source, id, name, version}]` |
| `maxor install <app…> --json` | `[{id, source, ok}]` |
| `maxor remove <app…> --json` | `[{id, source, ok}]` |
| `maxor profile list --json` | `[{id, title, description, enabled}]` |
| `maxor hardware detect` | el contenido de `hardware.json` ([HARDWARE.md](HARDWARE.md)) |

`source` es `nix` o `flatpak`. El código de salida es 0 si todo salió bien y 1 si algo falló.
Una app no debería reimplementar nada de esto: llama a `maxor` y lee el JSON.
