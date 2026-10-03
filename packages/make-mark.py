"""Genera el símbolo de Maxor (la «M» de Krona One) como SVG, a partir de la
propia fuente, para no depender de que la fuente esté instalada al dibujarlo.

uso: make-mark.py <KronaOne.ttf> <directorio de salida>
"""
import sys

from fontTools.pens.boundsPen import BoundsPen
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.ttLib import TTFont

font = TTFont(sys.argv[1])
glyphs = font.getGlyphSet()
glyph = glyphs[font.getBestCmap()[ord("M")]]

path_pen = SVGPathPen(glyphs)
glyph.draw(path_pen)
bounds_pen = BoundsPen(glyphs)
glyph.draw(bounds_pen)
x0, y0, x1, y1 = bounds_pen.bounds
w, h = x1 - x0, y1 - y0
d = path_pen.getCommands()


def svg(vw, vh, fill, fit, background=""):
    """La M centrada en un lienzo vw×vh, ocupando `fit` del lado limitante."""
    s = min(vw * fit / w, vh * fit / h)
    tx = (vw - w * s) / 2 - x0 * s
    ty = (vh - h * s) / 2 + y1 * s  # el eje Y de las fuentes va hacia arriba
    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{vw}" height="{vh}" '
        f'viewBox="0 0 {vw} {vh}">{background}'
        f'<g transform="translate({tx:.3f} {ty:.3f}) scale({s:.5f} {-s:.5f})">'
        f'<path d="{d}" fill="{fill}"/></g></svg>\n'
    )


out = sys.argv[2]
# Mismo lienzo que el logo original de DMS: la interfaz fija esa proporción.
# En blanco: DMS lo tiñe con el color primario del tema activo.
with open(f"{out}/mark.svg", "w") as f:
    f.write(svg(506.50931, 569.94629, "#ffffff", 0.80))
# Icono de aplicación: cuadrado redondeado con los colores por defecto (Sakura).
with open(f"{out}/icon.svg", "w") as f:
    bg = '<rect width="256" height="256" rx="56" fill="#1d121d"/>'
    f.write(svg(256, 256, "#ff86b8", 0.46, bg))
