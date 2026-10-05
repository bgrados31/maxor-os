# Motor de temas

Un tema de Maxor es una carpeta de datos. Se instala copiándola, se aplica con un comando y
no requiere `rebuild` ni `sudo`.

## Uso

```sh
maxor theme list                  # temas instalados, agrupados en oscuros y claros
maxor theme current               # nombre del tema activo
maxor theme apply <nombre>        # aplicar
maxor theme undo                  # volver al tema anterior
maxor theme install <ruta>        # instalar desde una carpeta o un .tar.gz
maxor theme export <nombre>       # empaquetar un tema como <nombre>.maxortheme
```

## Temas oficiales

Seis oscuros y seis claros. Todos superan el contraste AA de WCAG (4.5:1) entre el texto y el
fondo, y entre el acento y su texto.

| Tema | Modo | Carácter | Acento |
|---|---|---|---|
| `maxor-dark` · Maxor Dark (por defecto) | oscuro | Azul marino e índigo con acento violeta | `#c084ff` |
| `maxor-light` · Maxor Light | claro | Blanco suave con rosa y violeta | `#bc1f5f` |
| `sakura` · Sakura | oscuro | Ciruela oscura con rosa y durazno | `#ff86b8` |
| `glacier` · Glacier | oscuro | Azul marino profundo con cian hielo | `#5fd4f4` |
| `obsidian` · Obsidian | oscuro | Casi negro con índigo y lila | `#8c99ff` |
| `ember` · Ember | oscuro | Carbón cálido con naranja fuego | `#ff6a3d` |
| `ultraviolet` · Ultraviolet | oscuro | Negro violáceo con magenta y cian eléctricos | `#b84dff` |
| `dawn` · Dawn | claro | Rosa pálido con frambuesa; el día de Sakura | `#c2255c` |
| `frost` · Frost | claro | Blanco azulado con azul océano | `#0a6f94` |
| `paper` · Paper | claro | Gris azulado con azul tinta | `#3b4cff` |
| `breeze` · Breeze | claro | Verde humo claro con esmeralda | `#1a7f50` |
| `amber` · Amber | claro | Arena clara con naranja quemado | `#c2410c` |

El instalador solo ofrece Maxor Dark y Maxor Light; los demás se eligen después con `maxor theme` o en la app
Maxor. Los temas que antes tenían nombre en español (`brasa`, `alba`…) se siguen aceptando por ese nombre.
`gradient.json` (opcional, una lista de colores) hace que el wallpaper del tema sea un degradado diagonal por
esos colores en vez del halo por defecto.

## Qué cambia al aplicar un tema

1. **DMS**: se genera `~/.config/maxor/current/dms-theme.json` y se activa como tema propio. DMS
   recolorea la barra, el launcher y las notificaciones, y regenera los colores de kitty,
   Hyprland (bordes) y GTK.
2. **Modo claro u oscuro**: se avisa a DMS (`dms ipc call theme light|dark`) según el `mode` del
   tema.
3. **Lockscreen**: se rellena la plantilla de hyprlock (`home/lockscreen.nix`; marcadores `@FG_RGB@`, `@AC_RGB@`,
   `@AC2_RGB@`, `@BG_RGB@`…) y se escribe `~/.config/maxor/current/hyprlock.conf`. El diseño es el mismo en todos los
   temas: la marca en Cinzel arriba, un reloj grande y ligero con una línea del segundo acento, y una tarjeta de
   cristal con el usuario y un campo de contraseña en forma de píldora con borde en degradado de los dos acentos.
   En temas claros el fondo desenfocado se aclara para que el texto oscuro se lea. `hypridle` baja el brillo a
   los 150 s, bloquea a los 300 s y apaga la pantalla a los 420 s.
4. **Forma**: si el tema trae `style.json`, se genera `~/.config/maxor/current/hyprland.lua` (la velocidad de las
   animaciones usa las curvas de Maxor que define `home/hyprland/settings.lua`) y se
   recarga Hyprland (esquinas, espacios, desenfoque y velocidad de las animaciones). Esta forma se
   carga después de la de DMS, que genera su propio layout y de otro modo la pisaría.
5. **Wallpaper**: si el tema trae `wallpaper.png`, se aplica con DMS.
6. **kitty**: recibe `SIGUSR1` para releer su configuración.

Además, la propia terminal de `maxor` se dibuja con los colores del tema activo
([CLI.md](CLI.md#la-interfaz)). El tema anterior se guarda en `~/.local/state/maxor/history`;
`undo` lo recupera.

## Anatomía de un tema

```
mi-tema/
├── theme.toml       metadatos
├── colors.json      paleta (obligatorio)
├── style.json       forma: esquinas, espacios, desenfoque, animaciones (opcional)
└── wallpaper.png    fondo (opcional, PNG de hasta 20 MB)
```

### `colors.json`

Ocho colores en formato `#rrggbb` y, opcionalmente, el modo. Cualquier otro formato se rechaza.

| Clave | Uso |
|---|---|
| `bg` | Fondo del escritorio y del bloqueo |
| `s` | Superficies (paneles, barra, ventanas de `maxor`) |
| `s2` | Superficies elevadas (tarjetas, hover, barra de título) |
| `fg` | Texto principal |
| `mu` | Texto secundario |
| `ac` | Color de acento principal |
| `ac2` | Acento secundario |
| `on` | Texto sobre el acento (botones) |
| `mode` | `"dark"` (por defecto) o `"light"` |

```json
{
  "bg": "#120b12", "s": "#1d121d", "s2": "#2a1a2a",
  "fg": "#fbe9f2", "mu": "#a88a9d",
  "ac": "#ff86b8", "ac2": "#ffc2a6", "on": "#1b0b14",
  "mode": "dark"
}
```

Para un tema claro, `bg` es el fondo más tintado, `s` una superficie más clara y `s2` el blanco
de las tarjetas; el texto (`fg`) es oscuro.

### `style.json`

La forma del tema: solo números enteros, todos opcionales. `maxor` los valida y genera el Lua de
Hyprland; el archivo nunca se ejecuta.

| Clave | Rango | Efecto |
|---|---|---|
| `rounding` | 0 a 30 | Radio de las esquinas de las ventanas (px) |
| `gaps_in` | 0 a 20 | Espacio entre ventanas (px) |
| `gaps_out` | 0 a 40 | Espacio entre ventanas y bordes de pantalla (px) |
| `border_size` | 0 a 6 | Grosor del borde (px) |
| `blur_size` | 1 a 20 | Radio del desenfoque |
| `blur_passes` | 1 a 5 | Pasadas del desenfoque |
| `inactive` | 50 a 100 | Opacidad de las ventanas sin foco (%) |
| `anim` | 0 a 300 | Duración de las animaciones (%): 100 es la normal, 60 más rápida, 0 las apaga |

```json
{
  "rounding": 4, "gaps_in": 4, "gaps_out": 8, "border_size": 2,
  "blur_size": 6, "blur_passes": 2, "inactive": 94, "anim": 60
}
```

Si el tema no trae `style.json`, valen los valores por defecto de Maxor.

### `theme.toml`

```toml
name = "Sakura"
id = "sakura"
author = "Maxor OS"
version = "1.0.0"
license = "CC0-1.0"
description = "Ciruela oscura con rosa y durazno."
mode = "dark"
```

`id` identifica el tema: minúsculas, números y guiones, de 2 a 32 caracteres.

## Crear, instalar y compartir un tema

```sh
mkdir mi-tema && cd mi-tema
# crea colors.json y theme.toml con el formato de arriba
cd ..
maxor theme install mi-tema        # lo copia a ~/.local/share/maxor/themes/
maxor theme apply mi-tema
maxor theme export mi-tema         # genera mi-tema.maxortheme para compartirlo
```

`install` acepta una carpeta o un `.tar.gz` (el formato de `export`). Si el tema ya existe pide
`--force` para reemplazarlo; los temas oficiales no se pueden reemplazar.

Los temas oficiales viven en el store de Nix (solo lectura); los tuyos, en
`~/.local/share/maxor/themes/`, junto a ellos.

## Seguridad

Un tema solo aporta datos. `maxor theme install` y `apply`:

- validan que `colors.json` sea JSON, tenga las ocho claves y que cada valor sea `#rrggbb`, y que cada número de `style.json` sea un entero dentro de su rango;
- copian **solo** `colors.json` y `style.json` (reescritos con las claves conocidas), `theme.toml` (hasta 4 KB) y
  `wallpaper.png` (solo si empieza con la firma PNG y pesa menos de 20 MB). Cualquier otro archivo
  del tema se ignora;
- rechazan archivos `.tar.gz` de más de 30 MB, con enlaces simbólicos o archivos especiales, con
  rutas absolutas o con `..`;
- no ejecutan ningún archivo del tema;
- solo escriben en `~/.local/share/maxor/themes/`, `~/.config/maxor/current/`,
  `~/.local/state/maxor/` y dos claves de `settings.json` de DMS (tema y archivo del tema).

## Añadir un tema oficial

Crea `themes/<id>/colors.json` y `themes/<id>/theme.toml` en el repositorio. El build genera el
wallpaper y lo instala junto al resto; no hay que tocar código. Todos los wallpapers llevan la misma firma, que
genera `home/maxor.nix` con ImageMagick: un degradado (el de `gradient.json`, o un halo de `s2` sobre `bg`), una
viñeta suave, un resplandor del segundo acento (`ac2`) en la esquina superior derecha con tres órbitas finas, y
grano fino que además evita las bandas. En los temas claros el resplandor es más tenue y las órbitas son oscuras. Comprueba el contraste antes de proponerlo (ver [CONTRIBUTING.md](../CONTRIBUTING.md)).

## Limitaciones conocidas

- El tema se aplica a lo que DMS controla y al lockscreen. Las apps Qt necesitan haber elegido
  una vez el esquema de colores de DMS en `qt6ct`.
- La forma de la barra (isla flotante, márgenes, transparencia de DMS) se configura en los ajustes
  de DMS (`SUPER + ,`) y todavía no forma parte del tema.
- `style.json` cubre esquinas, espacios, bordes, desenfoque, opacidad y velocidad de animaciones;
  la curva de las animaciones y la disposición del shell siguen pendientes ([hoja de ruta](ROADMAP.md)).
- Instalar desde una URL todavía no existe.

## Forma de los temas oficiales

| Tema | Esquinas | Espacios (dentro / fuera) | Animaciones |
|---|---|---|---|
| Sakura | 8 px | 5 / 10 | 100 % |
| Glacier | 6 px | 4 / 8 | 90 % |
| Obsidian | 12 px | 5 / 12 | 100 % |
| Ember | 4 px | 4 / 8 | 60 % (rápidas) |
| Ultraviolet | 14 px | 6 / 14 | 120 % (más suaves) |
| Dawn | 10 px | 5 / 12 | 100 % |
| Frost | 8 px | 4 / 10 | 90 % |
| Paper | 6 px | 4 / 8 | 80 % |
| Breeze | 12 px | 6 / 12 | 110 % |
| Amber | 8 px | 5 / 10 | 100 % |
