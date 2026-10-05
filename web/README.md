# El sitio web de Maxor OS

Una landing estática: HTML, CSS y módulos JS sin paso de build. Las únicas librerías (GSAP + ScrollTrigger y
Lenis) están copiadas en `assets/vendor/`, servidas por el propio sitio. Se publica en
GitHub Pages con `.github/workflows/pages.yml` (desde `main`, o a mano), que antes pasa `tools/check.py`.

## Verlo en local

```sh
cd web && python3 -m http.server 8000     # http://localhost:8000  (?lang=en para inglés)
python3 web/tools/check.py                # i18n, temas, CSP, enlaces locales (también corre en CI)
```

## Qué hay

| Parte | Qué hace |
|---|---|
| `index.html` | La página, en español. Cada texto lleva `data-i18n`; `assets/i18n/en.json` tiene el inglés. |
| `assets/css/site.css` | Tokens (Maxor Dark por defecto, Maxor Light por sistema o por botón), hero, escritorio, secciones. |
| `assets/js/boot.js` | Corre antes de pintar: apariencia, idioma y si se permite movimiento. Nada parpadea. |
| `assets/js/themes.js` | Los temas y el tema de toda la web: elegir uno repinta la página con un círculo que crece desde el clic (View Transitions) y se recuerda (boot.js lo pinta antes del primer frame). |
| `assets/js/desk.js` | El escritorio del recorrido: cambia de workspace como Hyprland y aplica el tema de cada paso al instante. |
| `assets/js/term.js` | La terminal (Ctrl/⌘ K, la tecla ` o el botón `>_`): `help`, `maxor theme list/apply/undo`, `maxor rollback`, `fastfetch`, `maxor version`, `download`… |
| `assets/js/motion.js` | La capa cinematográfica: scroll suave, parallax del hero (scroll y puntero), la pantalla que se enciende, títulos palabra por palabra, contadores, la marquesina de temas que reacciona a la velocidad del scroll, botones magnéticos. |
| `assets/js/release.js` | La última versión, leída de la API pública de GitHub (pre-releases incluidas). |
| `assets/js/bar.js` | La barra: la sección actual (una tinta que se desliza), el progreso de lectura, el menú en móvil y el panel Apariencia (Sistema / Oscuro / Claro y los 12 temas). |
| `assets/js/effects.js` | Apariciones sin librerías, las escenas que avanzan con el scroll (generaciones, instalador), copiar, apariencia. |
| `assets/data/themes.json` | Copia de `themes/*/colors.json`. Se regenera con `python3 web/tools/sync-themes.py`. |
| `assets/brand/` | La «M» de Cinzel en SVG (`tools/make-brand.py`), la imagen para redes y el ícono táctil (`tools/render-images.mjs`). |
| `assets/fonts/` | Cinzel, Figtree y Red Hat Mono en WOFF2 (latin y latin-ext), con sus licencias OFL. |

## Publicar

`pages.yml` copia `web/` sin `tools/`, sella una versión en cada referencia local (`tools/stamp.py`: así un
despliegue nunca mezcla archivos nuevos con viejos de la caché de 10 minutos de Pages) y publica.

## Seguridad

- **CSP estricta** en la página (Pages no permite cabeceras): solo scripts, estilos, fuentes e imágenes
  propios; la única conexión externa es `api.github.com`. Sin código inline, y con Trusted Types, así que
  cualquier `innerHTML` lanzaría un error: todo texto se pone con `textContent`.
- Lo que llega de GitHub se trata como no confiable: solo se aceptan enlaces al repo o al proyecto de
  SourceForge, las notas se reconstruyen como nodos desde un Markdown mínimo y se guarda en caché solo lo
  que se usa (15 min por pestaña; si GitHub falla, la última respuesta buena).
- Sin cookies, sin analítica, sin fuentes ni CDNs externos. `localStorage` solo recuerda idioma, apariencia y
  el tema elegido (boot.js lo valida: solo variables conocidas y colores).
- La terminal solo compara texto con comandos conocidos y escribe con `textContent`: nada se interpreta.

## La descarga

La tarjeta busca la ISO como asset de la release; si no está (GitHub limita los assets a 2 GB y la ISO pesa
~4.6 GB), usa el primer enlace a `sourceforge.net/projects/maxor-os/…` de las notas, y muestra el SHA-256
si las notas lo traen. Para la 0.2.0-beta basta con poner en las notas el enlace de SourceForge y el hash.

## Accesibilidad y movimiento

Navegación por teclado con foco visible, enlace para saltar al contenido, contraste de la paleta oficial
(≥ 5:1 en los pares principales) y `prefers-reduced-motion`: sin animaciones, la página muestra su estado
final. Sin JavaScript se lee completa, y si las librerías no cargan todo funciona con las apariciones en CSS.
El arranque (la marca y una barra, ~1 s) sale una vez por pestaña y nunca con movimiento reducido.
