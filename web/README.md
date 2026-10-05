# El sitio web de Maxor OS

Una landing estática: HTML, CSS y módulos JS sin paso de build ni dependencias de terceros. Se publica en
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
| `assets/js/desk.js` | El escritorio del recorrido: se pinta con los temas reales y los aplica tecleando el comando. |
| `assets/js/release.js` | La última versión, leída de la API pública de GitHub (pre-releases incluidas). |
| `assets/js/effects.js` | Apariciones al hacer scroll, el hero, las escenas que avanzan con el scroll, copiar. |
| `assets/data/themes.json` | Copia de `themes/*/colors.json`. Se regenera con `python3 web/tools/sync-themes.py`. |
| `assets/brand/` | La «M» de Cinzel en SVG (`tools/make-brand.py`), la imagen para redes y el ícono táctil (`tools/render-images.mjs`). |
| `assets/fonts/` | Cinzel, Figtree y Red Hat Mono en WOFF2 (latin y latin-ext), con sus licencias OFL. |

## Seguridad

- **CSP estricta** en la página (Pages no permite cabeceras): solo scripts, estilos, fuentes e imágenes
  propios; la única conexión externa es `api.github.com`. Sin código inline, y con Trusted Types, así que
  cualquier `innerHTML` lanzaría un error: todo texto se pone con `textContent`.
- Lo que llega de GitHub se trata como no confiable: solo se aceptan enlaces al repo o al proyecto de
  SourceForge, las notas se reconstruyen como nodos desde un Markdown mínimo y se guarda en caché solo lo
  que se usa (15 min por pestaña; si GitHub falla, la última respuesta buena).
- Sin cookies, sin analítica, sin fuentes ni CDNs externos. `localStorage` solo recuerda idioma y apariencia.

## La descarga

La tarjeta busca la ISO como asset de la release; si no está (GitHub limita los assets a 2 GB y la ISO pesa
~4.6 GB), usa el primer enlace a `sourceforge.net/projects/maxor-os/…` de las notas, y muestra el SHA-256
si las notas lo traen. Para la 0.2.0-beta basta con poner en las notas el enlace de SourceForge y el hash.

## Accesibilidad y movimiento

Navegación por teclado con foco visible, enlace para saltar al contenido, contraste de la paleta oficial
(≥ 5:1 en los pares principales) y `prefers-reduced-motion`: sin animaciones, la página muestra su estado
final. Sin JavaScript se lee completa.
