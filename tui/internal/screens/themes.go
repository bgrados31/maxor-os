package screens

import (
	"context"
	"fmt"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/maxor"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/theme"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Themes elige el tema. Al moverte, toda la pantalla se pinta con ese tema (sin
// tocar el sistema); con Intro se aplica de verdad con `maxor theme apply`.
type Themes struct {
	core.Base
	list   listState
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
			verb := "Applied theme "
			if m.ID == "themes.undo" {
				verb = "Went back to the previous theme"
			}
			note := verb
			if s, ok := m.Value.(string); ok && m.ID == "themes.apply" {
				note += s
			}
			return t, tea.Batch(func() tea.Msg { return core.ThemeChangedMsg{} }, LoadThemes(env, true), core.Toast("ok", note), core.Note("ok", note))
		}
	case tea.KeyMsg:
		list := ordered(env)
		if d, mv := listKey(m); mv {
			t.list.move(d, len(list), max(t.rows, 1))
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
		return append([]ui.Line{heading(env, "Themes")}, skeletonRows(env, 6, 14, 10)...)
	}
	t.settle(env)
	list := ordered(env)
	lines := []ui.Line{}
	if l, ok := working(env, "themes.apply", "Applying the theme"); ok {
		lines = append(lines, l)
	} else {
		lines = append(lines, heading(env, fmt.Sprintf("%d themes · dark and light", len(list))))
	}
	lines = append(lines, gap())
	from, to := t.list.window(len(list), t.rows)
	for i := from; i < to; i++ {
		th := list[i]
		mark := " "
		if th.Active {
			mark = ui.G.Dot
		}
		sw := func(c string) ui.Seg { return ui.S(p.Fill.Foreground(lipgloss.Color(c)), "██") }
		lines = append(lines, ui.Line{
			L: []ui.Seg{ui.S(p.Ac, mark+" "), ui.S(p.Text, fmt.Sprintf("%-14s", th.ID)), sw(th.Colors.S2), sw(th.Colors.Ac), sw(th.Colors.Ac2), sw(th.Colors.Fg)},
			R: []ui.Seg{ui.S(p.Mu, th.Name+" ")},
			Sel: i == t.list.sel,
		})
	}
	return lines
}

func (t *Themes) Side(env *core.Env, w, h int) []ui.Line {
	p := env.P
	t.settle(env) // el panel lateral se dibuja antes que el principal
	list := ordered(env)
	if len(list) == 0 || t.list.sel >= len(list) {
		return []ui.Line{heading(env, "Preview")}
	}
	th := list[t.list.sel]
	pt := ToTheme(th)
	pp := ui.NewPainter(pt, th.Colors.Bg)
	pb := ui.NewPainter(pt, th.Colors.S2)
	row := func(pa ui.Painter, segs ...ui.Seg) ui.Line { return ui.Line{L: ui.Cell(segs, w, pa.Fill)} }
	sample := []ui.Line{
		row(pb, ui.S(pb.Ac.Bold(true), " maxor "), ui.S(pb.Mu, ui.G.Arrow+" "+th.ID)),
		row(pp, ui.S(pp.Ok, " "+ui.G.Tick+" applied")),
		row(pp, ui.S(pp.Warn, " "+ui.G.Warn+" one warning")),
		row(pp, ui.S(pp.Bad, " "+ui.G.Bad+" a problem")),
		{L: ui.Cell([]ui.Seg{ui.S(pp.Sel, " "+ui.G.Sel+" selected row")}, w, pp.Fill)},
		row(pp, ui.S(pp.Text, " plain text "), ui.S(pp.Mu, "muted")),
		row(pp, ui.S(pp.Ac, " accent "), ui.S(pp.Ac2, "accent 2")),
	}
	lines := []ui.Line{heading(env, "Preview"), gap(), plain(env, th.Name), muted(env, th.Mode+" theme"), gap()}
	lines = append(lines, sample...)
	lines = append(lines, gap(), ui.Of(button(env, true, "Apply  ⏎"), space(1), button(env, false, "Undo  u")))
	_ = p
	return lines
}

func (t *Themes) Hints(env *core.Env) []ui.Hint {
	return []ui.Hint{{Key: "↑↓", Action: "preview"}, {Key: "⏎", Action: "apply"}, {Key: "u", Action: "undo"}, {Key: "esc", Action: "back to current"}}
}

func (t *Themes) Click(env *core.Env, x, y int) tea.Cmd {
	list := ordered(env)
	i := t.list.top + y - 2
	if i >= 0 && i < len(list) {
		t.list.sel = i
		return t.preview(env)
	}
	return nil
}

func (t *Themes) Wheel(env *core.Env, dy int) tea.Cmd {
	t.list.move(dy*2, len(ordered(env)), max(t.rows, 1))
	return t.preview(env)
}
