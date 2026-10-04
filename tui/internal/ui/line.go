package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Seg es un trozo de texto plano con su estilo.
type Seg struct {
	T string
	S lipgloss.Style
}

// Line es una fila: texto a la izquierda y, opcionalmente, a la derecha.
// Con Sel la fila se pinta entera con el estilo de selección.
type Line struct {
	L, R []Seg
	Sel  bool
}

// Ctx lleva los estilos de relleno y selección con los que se dibuja una fila.
type Ctx struct {
	Fill lipgloss.Style
	Sel  lipgloss.Style
	// SelBg, si se da, cambia cómo se ve la fila con el foco: en vez de pintarla entera con Sel, cada trozo
	// conserva su color y solo el fondo se tiñe. Es el foco suave del instalador.
	SelBg lipgloss.Color
}

// Ctx de un Painter.
func (p Painter) Ctx() Ctx { return Ctx{Fill: p.Fill, Sel: p.Sel} }

// T construye una fila de un solo trozo.
func T(s lipgloss.Style, text string) Line { return Line{L: []Seg{{T: text, S: s}}} }

// Blank es una fila vacía.
func Blank() Line { return Line{} }

// Of construye una fila con varios trozos a la izquierda.
func Of(segs ...Seg) Line { return Line{L: segs} }

// S construye un trozo.
func S(s lipgloss.Style, text string) Seg { return Seg{T: text, S: s} }

func segsWidth(segs []Seg) int {
	n := 0
	for _, s := range segs {
		n += ansi.StringWidth(s.T)
	}
	return n
}

// truncate recorta los trozos a max celdas, con puntos suspensivos si hace falta.
func truncate(segs []Seg, max int) []Seg {
	if max <= 0 {
		return nil
	}
	if segsWidth(segs) <= max {
		return segs
	}
	out := make([]Seg, 0, len(segs))
	left := max
	for i, s := range segs {
		w := ansi.StringWidth(s.T)
		last := i == len(segs)-1
		if w <= left && (!last || w <= left) {
			// ¿cabe este y sigue habiendo más? Se reserva una celda para «…» si no es el último.
			out = append(out, s)
			left -= w
			continue
		}
		out = append(out, Seg{T: ansi.Truncate(s.T, left, G.Ellipsis), S: s.S})
		return out
	}
	return out
}

// Render dibuja la fila con exactamente w celdas de ancho.
func (ln Line) Render(w int, c Ctx) string {
	if w <= 0 {
		return ""
	}
	right := ln.R
	rw := segsWidth(right)
	if rw > w {
		right = truncate(right, w)
		rw = segsWidth(right)
	}
	budget := w - rw
	if rw > 0 {
		budget-- // una celda de separación
	}
	left := truncate(ln.L, budget)
	lw := segsWidth(left)

	var b strings.Builder
	soft := ln.Sel && c.SelBg != ""
	paint := func(s Seg) string {
		switch {
		case soft:
			// solo cambia el fondo de la tarjeta: un trozo con fondo propio (un botón, un chip) lo conserva
			if bg := s.S.GetBackground(); bg == c.Fill.GetBackground() || bg == (lipgloss.NoColor{}) {
				return s.S.Background(c.SelBg).Render(s.T)
			}
			return s.S.Render(s.T)
		case ln.Sel:
			return c.Sel.Render(s.T)
		}
		// Un trozo sin fondo propio hereda el del relleno: no quedan huecos.
		return s.S.Inherit(c.Fill).Render(s.T)
	}
	fill := c.Fill
	switch {
	case soft:
		fill = c.Fill.Background(c.SelBg)
	case ln.Sel:
		fill = c.Sel
	}
	for _, s := range left {
		if s.T != "" {
			b.WriteString(paint(s))
		}
	}
	if gap := w - lw - rw; gap > 0 {
		b.WriteString(fill.Render(strings.Repeat(" ", gap)))
	}
	for _, s := range right {
		if s.T != "" {
			b.WriteString(paint(s))
		}
	}
	return b.String()
}

// Block dibuja las filas en un rectángulo de w×h celdas: lo que sobra se corta y
// lo que falta se rellena con el fondo.
func Block(lines []Line, w, h int, c Ctx) []string {
	out := make([]string, 0, h)
	for i := 0; i < h; i++ {
		if i < len(lines) {
			out = append(out, lines[i].Render(w, c))
		} else {
			out = append(out, c.Fill.Render(strings.Repeat(" ", w)))
		}
	}
	return out
}

// Inset añade margen a las filas: izquierda y derecha en celdas, y filas vacías
// arriba. Devuelve las filas ya con el margen, para dibujarlas con Block.
func Inset(lines []Line, left, top int) []Line {
	out := make([]Line, 0, len(lines)+top)
	for i := 0; i < top; i++ {
		out = append(out, Blank())
	}
	pad := Seg{T: strings.Repeat(" ", left)}
	for _, ln := range lines {
		nl := Line{Sel: ln.Sel, R: ln.R}
		nl.L = append([]Seg{pad}, ln.L...)
		out = append(out, nl)
	}
	return out
}

// JoinH une columnas de filas, de la misma altura, separadas por gap.
func JoinH(gap string, cols ...[]string) []string {
	if len(cols) == 0 {
		return nil
	}
	h := len(cols[0])
	out := make([]string, h)
	for i := 0; i < h; i++ {
		var b strings.Builder
		for j, col := range cols {
			if j > 0 {
				b.WriteString(gap)
			}
			if i < len(col) {
				b.WriteString(col[i])
			}
		}
		out[i] = b.String()
	}
	return out
}

// Cell recorta o rellena los trozos de una fila hasta exactamente w celdas: sirve
// para componer varias columnas (p. ej. dos tarjetas lado a lado) en una sola fila.
func Cell(segs []Seg, w int, fill lipgloss.Style) []Seg {
	if w <= 0 {
		return nil
	}
	out := truncate(segs, w)
	if gap := w - segsWidth(out); gap > 0 {
		out = append(out, Seg{T: strings.Repeat(" ", gap), S: fill})
	}
	return out
}

// Wrap parte un texto en filas de hasta w celdas, por palabras. Una palabra más ancha que la fila (una ruta,
// un hash) va en trozos en filas propias: nunca se sale.
func Wrap(text string, w int) []string {
	if w < 4 {
		w = 4
	}
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		cur := ""
		for _, word := range strings.Fields(para) {
			for ansi.StringWidth(word) > w {
				if cur != "" {
					lines, cur = append(lines, cur), ""
				}
				head := ansi.Truncate(word, w, "")
				lines = append(lines, head)
				word = ansi.TruncateLeft(word, ansi.StringWidth(head), "")
			}
			switch {
			case cur == "":
				cur = word
			case ansi.StringWidth(cur)+1+ansi.StringWidth(word) <= w:
				cur += " " + word
			default:
				lines = append(lines, cur)
				cur = word
			}
		}
		lines = append(lines, cur)
	}
	return lines
}

// Spread coloca left a la izquierda y right a la derecha en exactamente w celdas,
// rellenando el hueco con fill: para filas con texto a los dos lados sobre un fondo propio.
func Spread(left, right []Seg, w int, fill lipgloss.Style) []Seg {
	rw := segsWidth(right)
	if rw > w {
		right, rw = truncate(right, w), segsWidth(truncate(right, w))
	}
	budget := w - rw
	if rw > 0 {
		budget--
	}
	l := truncate(left, budget)
	out := append([]Seg{}, l...)
	if gap := w - segsWidth(l) - rw; gap > 0 {
		out = append(out, Seg{T: strings.Repeat(" ", gap), S: fill})
	}
	return append(out, right...)
}
