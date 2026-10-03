# Maxor Shell

Maxor Shell es [DankMaterialShell](https://github.com/AvengeMedia/DankMaterialShell) (DMS) con la
identidad de Maxor OS. Da la barra, el launcher, las notificaciones, el centro de control y la
pantalla de login.

## Cómo se construye: una capa de parches, no una copia

Maxor Shell **no es una copia del código de DMS**. Es un paquete de Nix
([`packages/maxor-shell.nix`](../packages/maxor-shell.nix)) que toma el paquete de upstream y le
cambia solo lo que el usuario ve. Así:

- seguimos recibiendo las mejoras y correcciones de DMS con `nix flake update`;
- no hay un fork que mantener ni conflictos de fusión;
- el cambio es pequeño y fácil de revisar (un archivo y un generador de logo);
- si algún día hace falta un fork completo, el punto de partida ya está aislado.

Cada texto que se sustituye usa `substituteInPlace --replace-fail`: si upstream reescribe esa
línea, la compilación **falla** en vez de dejar una marca a medias sin avisar. Eso obliga a
revisar el parche al actualizar.

## Qué cambia

| Dónde | Cambio |
|---|---|
| Logo del botón del launcher, «Acerca de», bienvenida y novedades | La «M» de Krona One, tintada con el color primario del tema |
| Botón del launcher en la barra | Pasa al modo de marca (`launcherLogoMode = "dank"`) la primera vez, si el widget no estaba personalizado; DMS lo trae en modo «apps» y sin ese ajuste el logo no se vería |
| Icono de la aplicación | La «M» en un cuadrado redondeado con los colores de Sakura |
| «Acerca de» | `DANK LINUX` pasa a `MAXOR OS` |
| Bienvenida | `Welcome to Maxor OS` |
| Identidad MPRIS, actualizaciones y plugins | `Maxor Shell` |

El logo se **genera desde la propia fuente** (`packages/make-mark.py`, con fontTools): se extrae el
contorno de la «M» de Krona One y se escribe como SVG. Así no depende de que la fuente esté
instalada para dibujarse, y sigue la regla de identidad de Maxor: logo solo texto, sin símbolo.

## Qué no cambia (a propósito)

- Los nombres internos (`dms`, `DankBar`, rutas `~/.config/DankMaterialShell/`, comandos
  `dms ipc …`): son la interfaz que usan los demás componentes y cambiarlos rompería cosas.
- La licencia y el aviso de copyright de DMS (MIT): se conservan en el paquete.
- Las traducciones: las cadenas cambiadas dejan de coincidir con sus traducciones y se muestran
  en inglés hasta que se traduzcan.

## La pantalla de login

El login es el **greeter de DMS sobre greetd** ([`modules/greeter.nix`](../modules/greeter.nix)),
ejecutado con el paquete Maxor Shell dentro de Hyprland. Reemplaza a SDDM.

En cada arranque, greetd copia del usuario configurado (`services.displayManager.dms-greeter.configHome`)
sus ajustes de DMS, el tema propio, los colores y el wallpaper. Por eso **el login lleva los
colores del último tema que aplicaste con `maxor theme apply`**.

Si el login no aparece, mira [TROUBLESHOOTING.md](TROUBLESHOOTING.md#el-login-no-aparece-o-queda-en-negro).

## Qué falta para una identidad completa

- Fuentes de Maxor (Krona One para el logo) dentro de la propia interfaz de DMS.
- Pantalla de ajustes con la marca de Maxor y acceso directo a la futura Maxor Store.
- Integrar el radio de las esquinas del tema con el de DMS (hoy se ajusta a mano en `SUPER + ,`).
- Traducciones de las cadenas cambiadas.
- Shell propio en Quickshell (camino B de la [hoja de ruta](ROADMAP.md)).
