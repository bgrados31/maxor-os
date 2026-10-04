package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

func blankRow(p ui.Painter, w int) string { return p.Fill.Render(strings.Repeat(" ", w)) }

// View compone la pantalla: cada fila mide exactamente el ancho, y hay
// exactamente h filas. Así nunca queda texto de antes ni se desplaza nada.
func (m *Model) View() string {
	if m.quitting || m.w == 0 {
		return ""
	}
	w, h := m.w, m.h
	t := m.env.Theme
	page := ui.NewPainter(t, t.P.Bg)
	if w < minW || h < minH {
		return m.tooSmall(page, w, h)
	}
	rows := make([]string, 0, h)
	rows = append(rows, m.topBar(page, w), blankRow(page, w))
	rows = append(rows, m.body(page, w, h-4)...)
	rows = append(rows, m.toastRow(page, w), m.footer(page, w))
	return strings.Join(rows, "\n")
}

func (m *Model) tooSmall(p ui.Painter, w, h int) string {
	msg := fmt.Sprintf("Maxor needs at least %d×%d. Now: %d×%d.", minW, minH, w, h)
	rows := make([]string, h)
	for i := range rows {
		rows[i] = blankRow(p, w)
	}
	if h > 0 {
		rows[h/2] = ui.Line{L: []ui.Seg{ui.S(p.Warn, msg)}}.Render(w, p.Ctx())
	}
	return strings.Join(rows, "\n")
}

// topBar: las pestañas a la izquierda y el cargador a la derecha.
func (m *Model) topBar(p ui.Painter, w int) string {
	var segs []ui.Seg
	m.tabs = m.tabs[:0]
	if m.setupFocus {
		segs = []ui.Seg{ui.S(p.Mu, " maxor "), ui.S(p.Ac, ui.G.Arrow+" "), ui.S(p.Bold, "setup")}
	} else {
		x := 1
		segs = append(segs, ui.Seg{T: " "})
		for i, s := range m.screens {
			label := " " + s.Title() + " "
			st := p.Mu
			if i == m.active {
				st = p.Btn
			}
			segs = append(segs, ui.S(st, label), ui.Seg{T: " "})
			wd := ansi.StringWidth(label)
			m.tabs = append(m.tabs, tabRect{x0: x, x1: x + wd})
			x += wd + 1
		}
	}
	return ui.Line{L: segs, R: m.indicator(p)}.Render(w, p.Ctx())
}

// indicator es el cargador único: el mismo spinner, la misma etiqueta y los
// mismos segundos en todas las pestañas.
func (m *Model) indicator(p ui.Painter) []ui.Seg {
	act := m.env.Tasks.Active()
	if len(act) == 0 {
		return nil
	}
	a := act[0]
	segs := []ui.Seg{ui.S(p.Ac, ui.Spin(m.env.Frame)+"  "), ui.S(p.Text, a.Label)}
	if d := ui.Duration(a.Elapsed, 3*1e9); d != "" {
		segs = append(segs, ui.S(p.Mu, "  "+d))
	}
	if len(act) > 1 {
		segs = append(segs, ui.S(p.Mu, fmt.Sprintf("  +%d", len(act)-1)))
	}
	return append(segs, ui.Seg{T: " "})
}

func (m *Model) toastRow(p ui.Painter, w int) string {
	if !m.toastActive() {
		return blankRow(p, w)
	}
	st, g := p.Level(m.toast.kind)
	return ui.Line{L: []ui.Seg{ui.S(st, " "+g+"  "), ui.S(p.Text, m.toast.text)}}.Render(w, p.Ctx())
}

func (m *Model) footer(p ui.Painter, w int) string {
	s := m.screens[m.active]
	hints := s.Hints(m.env)
	if m.overlay == ovPalette {
		hints = []ui.Hint{{Key: "↑↓", Action: "move"}, {Key: "⏎", Action: "run"}, {Key: "esc", Action: "close"}}
	} else if !s.Captures() {
		// las ayudas globales, sin repetir una tecla que la pantalla ya explica
		have := map[string]bool{}
		for _, h := range hints {
			have[h.Key] = true
		}
		hints = append([]ui.Hint{}, hints...)
		globals := []ui.Hint{{Key: ":", Action: "commands"}, {Key: "?", Action: "help"}, {Key: "q", Action: "quit"}}
		if !m.setupFocus && len(m.screens) > 1 {
			globals = append([]ui.Hint{{Key: "←→", Action: "tabs"}}, globals...)
		}
		for _, g := range globals {
			if !have[g.Key] {
				hints = append(hints, g)
			}
		}
	}
	left := append([]ui.Seg{{T: " "}}, ui.Hints(p, hints)...)
	version := m.opts.Version
	if version == "" {
		version = "dev"
	}
	sep := ui.S(p.Ac, " "+ui.G.Dot+" ")
	right := []ui.Seg{ui.S(p.Mu, m.env.Host), sep, ui.S(p.Mu, m.env.Theme.Name), sep, ui.S(p.Mu, version+" ")}
	return ui.Line{L: left, R: right}.Render(w, p.Ctx())
}

// panel dibuja un panel de pw×ph celdas con margen interior y devuelve sus filas.
func (m *Model) panel(lines []ui.Line, pw, ph int) []string {
	return ui.Block(ui.Inset(lines, 2, 1), pw, ph, m.env.P.Ctx())
}

func (m *Model) body(page ui.Painter, w, h int) []string {
	margin := ui.Seg{T: " "}.T
	pad := page.Fill.Render(margin)
	s := m.screens[m.active]

	var cols [][]string
	switch {
	case m.overlay == ovHelp:
		pw := min(64, w-2)
		cols = [][]string{m.center(m.helpLines(), pw, h, w)}
		return cols[0]
	case m.overlay == ovPalette:
		pw := min(70, w-2)
		return m.center(m.paletteLines(pw), pw, h, w)
	}

	avail := w - 2
	if m.setupFocus {
		pw := min(78, avail)
		x0 := (w - pw) / 2
		m.mainX0, m.mainW, m.bodyTop = x0, pw, 2
		mainRows := m.panel(s.Main(m.env, pw-4, h-2), pw, h)
		left := page.Fill.Render(strings.Repeat(" ", x0))
		right := page.Fill.Render(strings.Repeat(" ", w-x0-pw))
		out := make([]string, h)
		for i := range out {
			out[i] = left + mainRows[i] + right
		}
		return out
	}

	// Los detalles van al lado si la ventana es ancha; si es estrecha (una de cuatro en la
	// pantalla), en un cajón compacto debajo; y con «D» se ocultan del todo.
	sideW := 0
	if !m.prefs.DetailsHidden {
		switch {
		case w >= 120:
			sideW = 40
		case w >= 96:
			sideW = 34
		}
	}
	mw := avail
	var side []ui.Line
	if sideW > 0 {
		side = s.Side(m.env, sideW-4, h-2)
		if len(side) > 0 {
			mw = avail - sideW - 1
		} else {
			sideW = 0
		}
	}
	m.mainX0, m.mainW, m.bodyTop = 1, mw, 2

	// cajón de abajo: solo si no hay panel al lado, no se ocultó y queda alto para la lista
	var drawer []ui.Line
	if sideW == 0 && !m.prefs.DetailsHidden && h >= 17 {
		drawer = m.drawerLines(s, avail-4, h)
	}
	mainH := h
	drawerH := 0
	if len(drawer) > 0 {
		drawerH = len(drawer) + 2
		mainH = h - drawerH - 1
	}
	mainRows := m.panel(s.Main(m.env, mw-4, mainH-2), mw, mainH)
	left := make([]string, mainH)
	right := make([]string, mainH)
	for i := range left {
		left[i], right[i] = pad, pad
	}
	cols = [][]string{left, mainRows}
	if sideW > 0 {
		gapCol := make([]string, h)
		for i := range gapCol {
			gapCol[i] = pad
		}
		cols = append(cols, gapCol, m.panel(side, sideW, h))
	}
	cols = append(cols, right)
	rows := ui.JoinH("", cols...)
	if drawerH > 0 {
		rows = append(rows, blankRow(page, w))
		for _, r := range m.panel(drawer, avail, drawerH) {
			rows = append(rows, pad+r+pad)
		}
	}
	return rows
}

// drawerLines es el contenido del cajón de detalles: la versión corta de la pantalla si la
// tiene, o su panel lateral sin las filas en blanco, recortado a lo que cabe.
func (m *Model) drawerLines(s core.Screen, w, h int) []ui.Line {
	limit := 3
	if h >= 24 {
		limit = 5
	}
	var lines []ui.Line
	if b, ok := s.(core.Briefer); ok {
		lines = b.Brief(m.env, w)
	} else {
		for _, l := range s.Side(m.env, w, h) {
			if len(l.L) > 0 {
				lines = append(lines, l)
			}
		}
		// el título del panel («DETAILS») ya lo dice el cajón: ocupa una fila sin aportar
		if len(lines) > 1 {
			lines = lines[1:]
		}
	}
	if _, ok := s.(core.Briefer); ok {
		limit = 5 // la versión corta ya viene medida: solo se acota el máximo
	}
	if len(lines) > limit {
		lines = lines[:limit]
	}
	return lines
}

// center coloca un panel de pw de ancho en el centro del cuerpo.
func (m *Model) center(lines []ui.Line, pw, h, w int) []string {
	page := ui.NewPainter(m.env.Theme, m.env.Theme.P.Bg)
	ph := min(len(lines)+3, h)
	top := (h - ph) / 2
	x0 := (w - pw) / 2
	panel := m.panel(lines, pw, ph)
	left := page.Fill.Render(strings.Repeat(" ", x0))
	right := page.Fill.Render(strings.Repeat(" ", w-x0-pw))
	out := make([]string, 0, h)
	for i := 0; i < top; i++ {
		out = append(out, blankRow(page, w))
	}
	for _, r := range panel {
		out = append(out, left+r+right)
	}
	for len(out) < h {
		out = append(out, blankRow(page, w))
	}
	return out
}

func (m *Model) helpLines() []ui.Line {
	p := m.env.P
	kv := func(k, v string) ui.Line {
		return ui.Of(ui.S(p.Ac.Bold(true), fmt.Sprintf("%-16s", k)), ui.S(p.Text, v))
	}
	lines := []ui.Line{ui.T(p.Mu, "KEYS"), ui.Blank(), kv("↑ ↓   j k", "move"), kv("⏎", "choose")}
	if !m.setupFocus {
		lines = append(lines, kv("← →   h l", "previous · next tab"), kv("tab  shift+tab", "next · previous tab"), kv("1 … 6", "jump to a tab"))
	}
	lines = append(lines, kv(":", "command palette"), kv("D", "show or hide the details"), kv("?", "this help"), kv("q", "back to your terminal"))
	// y lo propio de la pantalla en la que estás
	if hs := m.screens[m.active].Hints(m.env); len(hs) > 0 {
		lines = append(lines, ui.Blank(), ui.T(p.Mu, strings.ToUpper("On "+m.screens[m.active].Title())), ui.Blank())
		for _, h := range hs {
			lines = append(lines, kv(h.Key, h.Action))
		}
	}
	return append(lines, ui.Blank(), ui.T(p.Mu, "The mouse works too: click tabs and rows, use the wheel."), ui.T(p.Mu, "Press any key to close."))
}

func (m *Model) paletteLines(pw int) []ui.Line {
	p := m.env.P
	field := append([]ui.Seg{ui.S(p.Ac, ui.G.Find+"  ")}, m.pal.in.Segs(p, true, pw-8)...)
	lines := []ui.Line{{L: field}, ui.Blank()}
	items := m.matches()
	if len(items) == 0 {
		return append(lines, ui.T(p.Mu, "No matches. Try: search brave · theme alba · go store"))
	}
	for i, it := range items {
		if i >= 9 {
			break
		}
		lines = append(lines, ui.Line{L: []ui.Seg{ui.S(p.Text, it.Label)}, R: []ui.Seg{ui.S(p.Mu, it.Hint+" ")}, Sel: i == m.pal.sel})
	}
	return lines
}
