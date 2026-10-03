# Identidad visual

Decisiones de diseño de Maxor OS. Sirven de referencia para temas, shell, instalador y sitio web.

## Carácter

**Minimalista, elegante, limpio y con un punto gamer.** Superficies oscuras y calmadas, un solo
acento por tema, tipografía geométrica y bordes suaves. Nada decorativo que no cumpla una función.

## Nombre y marca

- Nombre: **Maxor OS**. En identificadores y comandos: `maxor`.
- El logotipo es **solo texto**: `MAXOR OS` en Krona One, en mayúsculas y con tracking amplio.
  No hay símbolo gráfico.
- `/etc/os-release`: `NAME="Maxor OS"`. `ID` se mantiene en `nixos` porque varias herramientas
  dependen de ese valor.

## Paleta por defecto: Sakura nocturna

| Rol | Valor |
|---|---|
| Fondo | `#120b12` |
| Superficie | `#1d121d` |
| Superficie elevada | `#2a1a2a` |
| Texto | `#fbe9f2` |
| Texto secundario | `#a88a9d` |
| Acento | `#ff86b8` |
| Acento secundario | `#ffc2a6` |
| Texto sobre acento | `#1b0b14` |

Segundo tema oficial: **Glaciar** (`#07111a` de fondo, acento `#5fd4f4`). Los temas se
describen en [THEMING.md](THEMING.md).

## Tipografía

| Rol | Fuente | Dónde |
|---|---|---|
| Interfaz | **Figtree** | Barra, ajustes, menús, notificaciones, GTK |
| Terminal | **Red Hat Mono** | kitty, editor, código |
| Logo y títulos | **Krona One** | Wordmark, arranque, bloqueo |
| Iconos en texto | Symbols Nerd Font | Respaldo para kitty y la barra |

Las tres fuentes principales son libres (SIL OFL).

## Forma y movimiento

| Aspecto | Decisión |
|---|---|
| Esquinas | Suaves, 8 px |
| Transparencia | Translúcida: paneles al ~74 % con desenfoque moderado |
| Barra | Isla flotante, centrada |
| Fondo | Liso con viñeta suave, generado desde la paleta |
| Movimiento | Fluido, `cubic-bezier(.4, 0, .2, 1)`, ~380 ms |
| Iconos | Papirus |
| Cursor | Bibata Modern |

## Pantallas

- **Bloqueo:** reloj grande y centrado, fecha debajo, campo de contraseña redondeado y la marca
  abajo. Fondo: captura de pantalla desenfocada.
- **Arranque:** `MAXOR OS` en Krona One sobre el fondo de la paleta y una barra de progreso fina
  con el color de acento.
- **Login:** pendiente (greeter propio, fase 4).

## Principios

1. Un acento por pantalla.
2. El color de énfasis nunca es el de fondo ni el de texto.
3. El tema decide los colores; el sistema decide la forma. Cambiar de tema no mueve nada de sitio.
4. Todo debe verse bien en oscuro. El tema claro es una variante, no el valor por defecto.

## Temas oscuros y claros

Maxor OS se diseña primero en oscuro, pero ofrece cinco temas claros con las mismas reglas:
un solo acento, superficies tintadas hacia ese acento y contraste AA como mínimo. Los claros no
son una inversión de los oscuros: cada uno parte de su propio matiz (rosa, azul océano, azul
tinta, verde humo, arena). Lista completa en [THEMING.md](THEMING.md).

## La terminal de `maxor`

La CLI se dibuja como una ventana dentro de la terminal: marco redondeado en el color de acento,
barra de título con tres puntos (acento, acento secundario y gris), superficie propia y verde,
ámbar y rojo ajustados al modo del tema. Es el sello visual de las herramientas de Maxor y la
base del futuro instalador. Detalles en [CLI.md](CLI.md#la-interfaz).
