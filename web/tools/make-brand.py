"""Draws the website's brand assets from Cinzel itself, so they never depend on a loaded font.

usage: python3 make-brand.py <Cinzel.ttf> <output dir>
writes mark.svg (the «M», white, for masks and CSS) and favicon.svg (the «M» on the night gradient).
Same idea as packages/make-mark.py, which draws the Maxor Shell logo.
"""
import sys

from fontTools.pens.boundsPen import BoundsPen
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.ttLib import TTFont

font = TTFont(sys.argv[1])
glyphs = font.getGlyphSet(location={"wght": 600})
glyph = glyphs[font.getBestCmap()[ord("M")]]
pen = SVGPathPen(glyphs, ntos=lambda v: f"{v:.0f}")
glyph.draw(pen)
bounds = BoundsPen(glyphs)
glyph.draw(bounds)
x0, y0, x1, y1 = bounds.bounds
w, h = x1 - x0, y1 - y0
d = pen.getCommands()


def place(vw, vh, fit):
    s = min(vw * fit / w, vh * fit / h)
    tx = (vw - w * s) / 2 - x0 * s
    ty = (vh - h * s) / 2 + y1 * s  # font Y axis points up
    return f'transform="translate({tx:.2f} {ty:.2f}) scale({s:.5f} {-s:.5f})"'


out = sys.argv[2]
with open(f"{out}/mark.svg", "w") as f:
    f.write(
        f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">'
        f'<path {place(64, 64, 0.92)} fill="#fff" d="{d}"/></svg>\n'
    )
with open(f"{out}/favicon.svg", "w") as f:
    f.write(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">'
        '<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">'
        '<stop offset="0" stop-color="#e02f75"/><stop offset=".45" stop-color="#6700a3"/>'
        '<stop offset="1" stop-color="#050c38"/></linearGradient></defs>'
        '<rect width="64" height="64" rx="14" fill="url(#g)"/>'
        f'<path {place(64, 64, 0.6)} fill="#eef0ff" d="{d}"/></svg>\n'
    )
