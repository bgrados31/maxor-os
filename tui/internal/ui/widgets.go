package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Spin devuelve el fotograma del spinner: el mismo en toda la pantalla.
func Spin(frame int) string { return G.Spin[((frame%len(G.Spin))+len(G.Spin))%len(G.Spin)] }

// Bar dibuja la barra de progreso (la misma de la CLI) como trozos de una fila.
func Bar(p Painter, pct, width int) []Seg {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	n := pct * width / 100
	return []Seg{
		{T: strings.Repeat(G.BarOn, n), S: p.Ac},
		{T: strings.Repeat(G.BarOff, width-n), S: p.Mu},
	}
}

// Skeleton son huecos con un brillo que los recorre de izquierda a derecha mientras
// llegan los datos: se nota que la pantalla está viva y no una caja vacía.
func Skeleton(p Painter, frame int, widths ...int) Line {
	block := "█"
	if G.BarOn == "#" { // ASCII: sin degradado
		return skeletonPlain(p, widths...)
	}
	total := 0
	for _, w := range widths {
		total += w + 2
	}
	center := float64((frame*2)%(total+18)) - 9 // el brillo entra por la izquierda y sale por la derecha
	var segs []Seg
	x := 0
	for i, w := range widths {
		if i > 0 {
			segs = append(segs, Seg{T: "  "})
			x += 2
		}
		for k := 0; k < w; k++ {
			d := float64(x) - center
			if d < 0 {
				d = -d
			}
			t := 1 - d/7
			if t < 0 {
				t = 0
			}
			segs = append(segs, Seg{T: block, S: p.Fill.Foreground(lipgloss.Color(Mix(p.Dim, p.Glow, t)))})
			x++
		}
	}
	return Line{L: segs}
}

func skeletonPlain(p Painter, widths ...int) Line {
	var segs []Seg
	for i, w := range widths {
		if i > 0 {
			segs = append(segs, Seg{T: "  "})
		}
		segs = append(segs, Seg{T: strings.Repeat(".", w), S: p.Mu})
	}
	return Line{L: segs}
}

// Duration da el tiempo de una tarea como la CLI: segundos enteros, vacío si es breve.
func Duration(d time.Duration, min time.Duration) string {
	if d < min {
		return ""
	}
	return strings.TrimSpace(formatSeconds(d))
}

func formatSeconds(d time.Duration) string {
	s := int(d.Round(time.Second) / time.Second)
	if s < 60 {
		return itoa(s) + "s"
	}
	return itoa(s/60) + "m" + pad2(s%60) + "s"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

// Hint es una tecla con su acción, para la barra de ayuda.
type Hint struct{ Key, Action string }

// Hints dibuja las ayudas de teclas como trozos de una fila.
func Hints(p Painter, hs []Hint) []Seg {
	var segs []Seg
	for i, h := range hs {
		if i > 0 {
			segs = append(segs, Seg{T: "  ", S: p.Mu})
		}
		segs = append(segs, Seg{T: h.Key, S: p.Ac.Bold(true)}, Seg{T: " " + h.Action, S: p.Mu})
	}
	return segs
}

// Styles mínimos reutilizables.
var _ = lipgloss.NewStyle
