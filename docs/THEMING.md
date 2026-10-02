# Motor de temas

Un tema de Maxor es una carpeta de datos. Se instala copiándola, se aplica con un comando y
no requiere `rebuild` ni `sudo`.

## Uso

```sh
maxor theme list            # temas instalados (el activo lleva *)
maxor theme current         # nombre del tema activo
maxor theme apply <nombre>  # aplicar
maxor theme undo            # volver al tema anterior
```

## Qué cambia al aplicar un tema

1. **DMS**: se genera `~/.config/maxor/current/dms-theme.json` y se activa como tema propio.
   DMS recolorea la barra, el launcher y las notificaciones, y regenera los colores de kitty,
   Hyprland (bordes) y GTK.
2. **Lockscreen**: se rellena la plantilla de hyprlock y se escribe
   `~/.config/maxor/current/hyprlock.conf`.
3. **Wallpaper**: si el tema trae `wallpaper.png`, se aplica con DMS.
4. **kitty**: recibe `SIGUSR1` para releer su configuración.

El tema anterior se guarda en `~/.local/state/maxor/history`; `undo` lo recupera.

## Anatomía de un tema

```
mi-tema/
├── theme.toml       metadatos
├── colors.json      paleta (obligatorio)
└── wallpaper.png    fondo (opcional)
```

### `colors.json`

Ocho colores en formato `#rrggbb`. Cualquier otro formato se rechaza.

| Clave | Uso |
|---|---|
| `bg` | Fondo del escritorio y del bloqueo |
| `s` | Superficies (paneles, barra) |
| `s2` | Superficies elevadas (tarjetas, hover) |
| `fg` | Texto principal |
| `mu` | Texto secundario |
| `ac` | Color de acento principal |
| `ac2` | Acento secundario |
| `on` | Texto sobre el acento (botones) |

```json
{
  "bg": "#120b12", "s": "#1d121d", "s2": "#2a1a2a",
  "fg": "#fbe9f2", "mu": "#a88a9d",
  "ac": "#ff86b8", "ac2": "#ffc2a6", "on": "#1b0b14",
  "mode": "dark"
}
```

`mode` es opcional (`dark` por defecto).

### `theme.toml`

```toml
name = "Sakura nocturna"
id = "sakura"
author = "Maxor OS"
version = "1.0.0"
license = "CC0-1.0"
description = "Ciruela oscura con rosa y durazno."
mode = "dark"
```

## Crear y usar un tema propio

```sh
mkdir -p ~/.local/share/maxor/themes/mi-tema
cp /ruta/a/colors.json ~/.local/share/maxor/themes/mi-tema/
maxor theme apply mi-tema
```

Los temas oficiales viven en el store de Nix (solo lectura); los tuyos, en
`~/.local/share/maxor/themes/`, junto a ellos.

## Temas oficiales

| Tema | Carácter | Acento |
|---|---|---|
| **Sakura nocturna** (por defecto) | Ciruela oscura, suave y con carácter | `#ff86b8` |
| **Glaciar** | Azul marino profundo, limpio y técnico | `#5fd4f4` |

Los temas oficiales se definen en [`home/maxor.nix`](../home/maxor.nix) con la función
`mkTheme`, que además genera el wallpaper (degradado radial con grano) a partir de la paleta.

## Seguridad

Un tema solo aporta datos. El CLI:

- valida que `colors.json` tenga las ocho claves y que cada valor sea `#rrggbb`;
- no ejecuta ningún archivo del tema;
- solo escribe en `~/.config/maxor/current/`, en `~/.local/state/maxor/` y en dos claves de
  `settings.json` de DMS.

## Limitaciones conocidas

- El tema se aplica a lo que DMS controla y al lockscreen. Las apps Qt necesitan haber elegido
  una vez el esquema de colores de DMS en `qt6ct`.
- La forma de la barra (isla flotante, márgenes, transparencia) se configura en los ajustes de
  DMS (`SUPER + ,`) y todavía no forma parte del tema.
- Los temas aún no incluyen configuración de Hyprland ni del shell (esquinas, animaciones,
  disposición). Está en la [hoja de ruta](ROADMAP.md).
