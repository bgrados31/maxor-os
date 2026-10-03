package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/bgrados31/maxor-os/tui/internal/theme"
)

// Painter reúne los estilos sobre un fondo concreto. Cada trozo de texto lleva
// su propio fondo, porque lipgloss reinicia los colores después de cada uno: así
// no quedan huecos con el fondo de la terminal dentro de un panel.
type Painter struct {
	Bg                                     lipgloss.Color
	Fill                                   lipgloss.Style
	Text, Mu, Ac, Ac2, Ok, Warn, Bad, Bold lipgloss.Style
	Sel                                    lipgloss.Style // fila seleccionada: fondo de acento
	Dim, Glow                              string         // extremos del brillo del esqueleto (#rrggbb)
	Btn, Btn2                              lipgloss.Style // botones principal y secundario
}

// NewPainter crea los estilos de un tema sobre el color de fondo bg (#rrggbb).
func NewPainter(t theme.Theme, bg string) Painter {
	b := lipgloss.Color(bg)
	base := lipgloss.NewStyle().Background(b)
	fg := func(c string) lipgloss.Style { return base.Foreground(lipgloss.Color(c)) }
	return Painter{
		Dim:  Mix(bg, t.P.Mu, 0.16),
		Glow: Mix(bg, t.P.Mu, 0.55),
		Bg:   b,
		Fill: base,
		Text: fg(t.P.Fg),
		Mu:   fg(t.P.Mu),
		Ac:   fg(t.P.Ac),
		Ac2:  fg(t.P.Ac2),
		Ok:   fg(t.OK),
		Warn: fg(t.Warn),
		Bad:  fg(t.Bad),
		Bold: fg(t.P.Fg).Bold(true),
		Sel:  lipgloss.NewStyle().Background(lipgloss.Color(t.P.Ac)).Foreground(lipgloss.Color(t.P.On)).Bold(true),
		Btn:  lipgloss.NewStyle().Background(lipgloss.Color(t.P.Ac)).Foreground(lipgloss.Color(t.P.On)).Bold(true),
		Btn2: lipgloss.NewStyle().Background(lipgloss.Color(t.P.S2)).Foreground(lipgloss.Color(t.P.Fg)),
	}
}

// Level devuelve el estilo y el glifo de fila de un nivel: ok, warn, bad o info.
func (p Painter) Level(level string) (lipgloss.Style, string) {
	switch level {
	case "ok":
		return p.Ok, G.Tick
	case "warn":
		return p.Warn, G.Warn
	case "bad":
		return p.Bad, G.Bad
	}
	return p.Mu, G.Info
}

// Mix mezcla dos colores #rrggbb: t=0 da a y t=1 da b.
func Mix(a, b string, t float64) string {
	pa, pb := parseHex(a), parseHex(b)
	var out [3]int
	for i := range out {
		out[i] = int(float64(pa[i])*(1-t) + float64(pb[i])*t + 0.5)
	}
	return fmt.Sprintf("#%02x%02x%02x", out[0], out[1], out[2])
}

func parseHex(h string) [3]int {
	var r, g, b int
	fmt.Sscanf(strings.TrimPrefix(h, "#"), "%02x%02x%02x", &r, &g, &b)
	return [3]int{r, g, b}
}
