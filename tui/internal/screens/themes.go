package screens

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/maxor"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/theme"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Themes elige el tema. Cada tema se representa con un solo color, el de acento.
// Al moverte, toda la pantalla se pinta con ese tema (sin tocar el sistema); con
// Intro se aplica de verdad con `maxor theme apply`.
type Themes struct {
	core.Base
	list   listState // selección entre los temas (no cuenta las cabeceras)
	top    int       // primera fila visible, contando las cabeceras
	inited bool
	rows   int
}

func NewThemes() *Themes { return &Themes{} }

func (t *Themes) ID() string    { return "themes" }
func (t *Themes) Title() string { return "Themes" }

func (t *Themes) Init(env *core.Env) tea.Cmd {
	if env.Data.ThemesLoaded {
		return nil
	}
	return LoadThemes(env, false)
}

// ordered pone los oscuros antes que los claros, como `maxor theme list`.
func ordered(env *core.Env) []maxor.Theme {
	out := append([]maxor.Theme(nil), env.Data.Themes...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Mode != out[j].Mode {
			return out[i].Mode == "dark"
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// ToTheme convierte un tema de la CLI en uno de la pantalla.
func ToTheme(t maxor.Theme) theme.Theme {
	return theme.From(t.ID, theme.Palette{
		Bg: t.Colors.Bg, S: t.Colors.S, S2: t.Colors.S2, Fg: t.Colors.Fg, Mu: t.Colors.Mu,
		Ac: t.Colors.Ac, Ac2: t.Colors.Ac2, On: t.Colors.On, Mode: t.Mode,
	})
}

// themeRow es una fila de la lista: una cabecera de grupo o un tema.
type themeRow struct {
	header string
	idx    int // índice en ordered(); -1 en una cabecera
}

func (t *Themes) layout(env *core.Env) []themeRow {
	list := ordered(env)
	counts := map[string]int{}
	for _, th := range list {
		counts[th.Mode]++
	}
	var rows []themeRow
	last := ""
	for i, th := range list {
		if th.Mode != last {
			if last != "" {
				rows = append(rows, themeRow{idx: -1}) // aire entre oscuros y claros
			}
			last = th.Mode
			name := "Dark"
			if th.Mode == "light" {
				name = "Light"
			}
			rows = append(rows, themeRow{header: fmt.Sprintf("%s · %d", strings.ToUpper(name), counts[th.Mode]), idx: -1})
		}
		rows = append(rows, themeRow{idx: i})
	}
	return rows
}

func (t *Themes) settle(env *core.Env) {
	if t.inited || !env.Data.ThemesLoaded {
		return
	}
	t.inited = true
	for i, th := range ordered(env) {
		if th.Active {
			t.list.sel = i
		}
	}
}

func (t *Themes) preview(env *core.Env) tea.Cmd {
	th := ordered(env)
	if t.list.sel < 0 || t.list.sel >= len(th) {
		return nil
	}
	id := th[t.list.sel].ID
	return func() tea.Msg { return core.PreviewThemeMsg{ID: id} }
}

func (t *Themes) Update(env *core.Env, msg tea.Msg) (core.Screen, tea.Cmd) {
	t.settle(env)
	switch m := msg.(type) {
	case task.DoneMsg:
		if m.Owner() != "themes" {
			return t, nil
		}
		switch m.ID {
		case "themes.apply", "themes.undo":
			if m.Err != nil {
				return t, core.Toast("bad", "Could not change the theme: "+oneLine(m.Err.Error()))
			}
			note := "Went back to the previous theme"
			if s, ok := m.Value.(string); ok && m.ID == "themes.apply" {
				note = "Applied theme " + s
			}
			return t, tea.Batch(func() tea.Msg { return core.ThemeChangedMsg{} }, LoadThemes(env, true), core.Toast("ok", note), core.Note("ok", note))
		}
	case tea.KeyMsg:
		list := ordered(env)
		if d, mv := listKey(m); mv {
			t.list.move(d, len(list), 1<<20)
			return t, t.preview(env)
		}
		switch {
		case isKey(m, "enter") && len(list) > 0 && t.list.sel < len(list):
			id := list[t.list.sel].ID
			return t, env.Tasks.Start(task.Task{ID: "themes.apply", Label: "Applying " + id, Run: func(ctx context.Context) (any, error) {
				return id, env.Client.ApplyTheme(ctx, id)
			}})
		case isKey(m, "u"):
			return t, env.Tasks.Start(task.Task{ID: "themes.undo", Label: "Going back", Run: func(ctx context.Context) (any, error) {
				return nil, env.Client.UndoTheme(ctx)
			}})
		case isKey(m, "esc"):
			return t, func() tea.Msg { return core.PreviewThemeMsg{} }
		}
	}
	return t, nil
}

func (t *Themes) Main(env *core.Env, w, h int) []ui.Line {
	p := env.P
	t.rows = h - 3
	if !env.Data.ThemesLoaded {
		if err := env.Data.Err["themes"]; err != nil {
			return failed(env, "Could not load the themes", err)
		}
		lines := []ui.Line{heading(env, "Themes"), gap()}
		for i := 0; i < 8; i++ {
			lines = append(lines, ui.Skeleton(p, env.Frame+i*2, 2, 14+i%3*4))
		}
		return lines
	}
	t.settle(env)
	list := ordered(env)
	rows := t.layout(env)

	// Ventana: la fila seleccionada siempre se ve, con su cabecera si es la primera del grupo.
	selRow := 0
	for i, r := range rows {
		if r.idx == t.list.sel {
			selRow = i
		}
	}
	avail := max(t.rows, 1)
	if selRow < t.top {
		t.top = selRow
	}
	if selRow >= t.top+avail {
		t.top = selRow - avail + 1
	}
	if selRow > 0 && rows[selRow-1].idx < 0 && t.top > selRow-1 {
		t.top = selRow - 1
	}
	if t.top > len(rows)-avail {
		t.top = len(rows) - avail
	}
	if t.top < 0 {
		t.top = 0
	}

	var lines []ui.Line
	if l, ok := working(env, "themes.apply", "Applying the theme"); ok {
		lines = append(lines, l)
	} else {
		lines = append(lines, heading(env, fmt.Sprintf("%d themes", len(list))))
	}
	lines = append(lines, gap())
	for i := t.top; i < len(rows) && i < t.top+avail; i++ {
		r := rows[i]
		if r.idx < 0 {
			if r.header == "" {
				lines = append(lines, gap())
			} else {
				lines = append(lines, ui.T(p.Mu.Bold(true), r.header))
			}
			continue
		}
		th := list[r.idx]
		dot := ui.S(p.Fill.Foreground(lipgloss.Color(th.Colors.Ac)).Bold(true), ui.G.Swatch+"  ")
		right := []ui.Seg{ui.S(p.Mu, th.Mode+" ")}
		if th.Active {
			right = []ui.Seg{ui.S(p.Ok, ui.G.Tick+" in use  "), ui.S(p.Mu, th.Mode+" ")}
		}
		lines = append(lines, ui.Line{L: []ui.Seg{dot, ui.S(p.Text, th.Name)}, R: right, Sel: r.idx == t.list.sel})
	}
	return lines
}

func (t *Themes) Side(env *core.Env, w, h int) []ui.Line {
	t.settle(env)
	list := ordered(env)
	if len(list) == 0 || t.list.sel >= len(list) {
		return []ui.Line{heading(env, "Preview")}
	}
	th := list[t.list.sel]
	pt := ToTheme(th)
	pp := ui.NewPainter(pt, th.Colors.Bg)
	pb := ui.NewPainter(pt, th.Colors.S2)
	row := func(pa ui.Painter, segs ...ui.Seg) ui.Line { return ui.Line{L: ui.Cell(segs, w, pa.Fill)} }
	// una ventana en miniatura con la forma real de la pantalla: pestañas, lista con una
	// fila elegida, avisos, un botón y la barra de abajo
	sel := ui.Line{L: ui.Cell([]ui.Seg{ui.S(pp.Sel, " "+ui.G.Sel+" "+ui.G.Swatch+" selected row")}, w, pp.Fill)}
	sample := []ui.Line{
		row(pb, ui.S(pb.Mu, " Home  Store  "), ui.S(pb.Btn, " Themes "), ui.S(pb.Mu, "  Update")),
		row(pp),
		row(pp, ui.S(pp.Mu, " SECTION")),
		row(pp, ui.S(pp.Text, "   "+ui.G.Swatch+" first row")),
		sel,
		row(pp, ui.S(pp.Text, "   "+ui.G.Swatch+" third row")),
		row(pp),
		row(pp, ui.S(pp.Ok, " "+ui.G.Tick+" done  "), ui.S(pp.Warn, ui.G.Warn+" careful  "), ui.S(pp.Bad, ui.G.Bad+" failed")),
		row(pp),
		row(pp, ui.S(pp.Btn, " Apply "), ui.S(pp.Mu, "  muted hint")),
		row(pp),
		row(pb, ui.S(pb.Mu, " ↑↓ move  ⏎ choose"), ui.S(pb.Ac, "  "+th.ID)),
	}
	p := env.P
	lines := []ui.Line{
		heading(env, "Preview"), gap(),
		ui.Of(ui.S(p.Fill.Foreground(lipgloss.Color(th.Colors.Ac)).Bold(true), ui.G.Swatch+"  "), ui.S(p.Bold, th.Name)),
		muted(env, "   "+th.Mode+" theme"), gap(),
	}
	// Con poco alto se enseña lo más propio de Maxor (el fastfetch de muestra); con
	// más, también la ventana en miniatura.
	card := fetchCard(env, th)
	room := h - len(lines) - 4 // los botones y el aviso de «en uso»
	switch {
	case room >= len(sample)+len(card)+1:
		lines = append(lines, sample...)
		lines = append(lines, gap())
		lines = append(lines, card...)
	case room >= len(card):
		lines = append(lines, card...)
	default:
		lines = append(lines, sample...)
	}
	lines = append(lines, gap(), ui.Of(button(env, true, "Apply  ⏎"), space(1), button(env, false, "Undo  u")))
	if th.Active {
		lines = append(lines, gap(), ui.T(p.Ok, ui.G.Tick+" this is the theme in use"))
	}
	return lines
}

// fetchCard imita la salida de fastfetch con los colores del tema que se está mirando
// (toda la pantalla ya está pintada con él): el logo de Maxor, los datos del equipo y
// la tira de colores de siempre.
func fetchCard(env *core.Env, th maxor.Theme) []ui.Line {
	p := env.P
	user := os.Getenv("USER")
	if user == "" {
		user = "you"
	}
	host := env.Host
	if host == "" {
		host = "maxor"
	}
	logo := []string{"██▄  ▄██", "██ ▀▀ ██", "██    ██", "▀▀    ▀▀"}
	info := func(k, v string) []ui.Seg {
		return []ui.Seg{ui.S(p.Ac, fmt.Sprintf("%-6s", k)), ui.S(p.Text, v)}
	}
	pad := func(i int) ui.Seg {
		if i < len(logo) {
			return ui.S(p.Ac.Bold(true), logo[i]+"   ")
		}
		return ui.S(p.Fill, strings.Repeat(" ", 11))
	}
	rows := [][]ui.Seg{
		{ui.S(p.Ac.Bold(true), user), ui.S(p.Mu, "@"), ui.S(p.Ac.Bold(true), host)},
		{ui.S(p.Mu, strings.Repeat("─", len(user)+len(host)+1))},
		info("OS", "Maxor OS"),
		info("WM", "Hyprland"),
		info("Shell", "fish"),
		info("Theme", th.Name),
	}
	var out []ui.Line
	for i, r := range rows {
		out = append(out, ui.Line{L: append([]ui.Seg{pad(i)}, r...)})
	}
	sw := func(st lipgloss.Style) ui.Seg { return ui.S(st, "███") }
	acc := func(c string) lipgloss.Style { return p.Fill.Foreground(lipgloss.Color(c)) }
	out = append(out, ui.Line{L: []ui.Seg{ui.S(p.Fill, strings.Repeat(" ", 11)), sw(acc(th.Colors.Ac)), sw(acc(th.Colors.Ac2)), sw(p.Ok), sw(p.Warn), sw(p.Bad), sw(acc(th.Colors.Mu)), sw(acc(th.Colors.Fg))}})
	return out
}

func (t *Themes) Hints(env *core.Env) []ui.Hint {
	return []ui.Hint{{Key: "↑↓", Action: "preview"}, {Key: "⏎", Action: "apply"}, {Key: "u", Action: "undo"}, {Key: "esc", Action: "back to current"}}
}

func (t *Themes) Click(env *core.Env, x, y int) tea.Cmd {
	rows := t.layout(env)
	i := t.top + y - 2
	if i >= 0 && i < len(rows) && rows[i].idx >= 0 {
		t.list.sel = rows[i].idx
		return t.preview(env)
	}
	return nil
}

func (t *Themes) Wheel(env *core.Env, dy int) tea.Cmd {
	t.list.move(dy*2, len(ordered(env)), 1<<20)
	return t.preview(env)
}

// Brief: el tema elegido con su color, un fastfetch en una fila y los colores de estado.
func (t *Themes) Brief(env *core.Env, w int) []ui.Line {
	p := env.P
	t.settle(env)
	list := ordered(env)
	if len(list) == 0 || t.list.sel >= len(list) {
		return []ui.Line{muted(env, "Waiting for the themes.")}
	}
	th := list[t.list.sel]
	acc := func(c string) lipgloss.Style { return p.Fill.Foreground(lipgloss.Color(c)) }
	head := []ui.Seg{ui.S(acc(th.Colors.Ac).Bold(true), ui.G.Swatch+"  "), ui.S(p.Bold, th.Name), ui.S(p.Mu, "  "+th.Mode)}
	if th.Active {
		head = append(head, ui.S(p.Ok, "  "+ui.G.Tick+" in use"))
	}
	strip := []ui.Seg{ui.S(p.Ac.Bold(true), "▄ maxor  "), ui.S(p.Mu, "Hyprland · fish  ")}
	for _, c := range []string{th.Colors.Ac, th.Colors.Ac2} {
		strip = append(strip, ui.S(acc(c), "███"))
	}
	for _, st := range []lipgloss.Style{p.Ok, p.Warn, p.Bad} {
		strip = append(strip, ui.S(st, "███"))
	}
	return []ui.Line{ui.Of(head...), ui.Of(strip...),
		ui.Of(ui.S(p.Ok, ui.G.Tick+" done  "), ui.S(p.Warn, ui.G.Warn+" careful  "), ui.S(p.Bad, ui.G.Bad+" failed  "), button(env, true, "Apply  ⏎"), space(1), button(env, false, "Undo  u"))}
}
