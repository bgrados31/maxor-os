package app

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/screens"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// paletteItem es una orden de la paleta de comandos (se abre con «:»).
type paletteItem struct {
	Label string
	Hint  string
	Run   func(m *Model) tea.Cmd
}

type palette struct {
	in  ui.Input
	sel int
}

func (m *Model) paletteItems() []paletteItem {
	var items []paletteItem
	for _, s := range m.screens {
		id := s.ID()
		items = append(items, paletteItem{Label: "go " + strings.ToLower(s.Title()), Hint: tr("open %s", s.Title()), Run: func(m *Model) tea.Cmd { return core.Go(id) }})
	}
	items = append(items,
		paletteItem{Label: "update check", Hint: tr("check for updates"), Run: func(m *Model) tea.Cmd {
			return core.GoThen("update", screens.CheckUpdateMsg{})
		}},
		paletteItem{Label: "doctor run", Hint: tr("run the health checks"), Run: func(m *Model) tea.Cmd {
			return core.GoThen("doctor", screens.RunDoctorMsg{})
		}},
	)
	for _, th := range m.env.Data.Themes {
		id, name := th.ID, th.Name
		items = append(items, paletteItem{Label: "theme " + id, Hint: tr("apply %s", name), Run: func(m *Model) tea.Cmd {
			return tea.Batch(core.Go("themes"), m.env.Tasks.Start(task.Task{ID: "themes.apply", Label: tr("Applying %s", id), Run: func(ctx context.Context) (any, error) {
				return id, m.env.Client.ApplyTheme(ctx, id)
			}}))
		}})
	}
	for _, a := range m.env.Data.Apps {
		id, name := a.ID, a.Name
		items = append(items, paletteItem{Label: "open " + strings.ToLower(name), Hint: tr("start %s", name), Run: func(m *Model) tea.Cmd {
			return core.GoThen("store", screens.OpenAppMsg{ID: id, Name: name})
		}})
	}
	items = append(items,
		paletteItem{Label: "backup", Hint: tr("save your setup in one file"), Run: func(m *Model) tea.Cmd { return core.GoThen("home", screens.BackupMsg{}) }},
		paletteItem{Label: "details", Hint: tr("show or hide the details panel"), Run: func(m *Model) tea.Cmd { return m.toggleDetails() }},
		paletteItem{Label: "exit", Hint: tr("back to your terminal"), Run: func(m *Model) tea.Cmd { return core.Quit() }},
		paletteItem{Label: "quit", Hint: tr("back to your terminal"), Run: func(m *Model) tea.Cmd { return tea.Quit }})
	return items
}

// matches filtra las órdenes por las palabras escritas. Con «search …» o
// «install …» ofrece buscar en la Tienda.
func (m *Model) matches() []paletteItem {
	q := strings.TrimSpace(strings.ToLower(m.pal.in.Text()))
	for _, pre := range []string{"search ", "install ", "s "} {
		if strings.HasPrefix(q, pre) && strings.TrimSpace(q[len(pre):]) != "" {
			term := strings.TrimSpace(m.pal.in.Text()[len(pre):])
			return []paletteItem{{Label: "search " + term, Hint: tr("look for it in nixpkgs and Flathub"), Run: func(m *Model) tea.Cmd {
				return core.GoThen("store", core.SearchMsg{Query: term})
			}}}
		}
	}
	words := strings.Fields(q)
	var out []paletteItem
	for _, it := range m.paletteItems() {
		l := strings.ToLower(it.Label + " " + it.Hint)
		ok := true
		for _, w := range words {
			if !strings.Contains(l, w) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, it)
		}
	}
	return out
}

func (m *Model) paletteKey(k tea.KeyMsg) tea.Cmd {
	items := m.matches()
	switch k.String() {
	case "esc":
		m.overlay = ovNone
		return nil
	case "up", "ctrl+p":
		if m.pal.sel > 0 {
			m.pal.sel--
		}
		return nil
	case "down", "ctrl+n":
		if m.pal.sel < len(items)-1 {
			m.pal.sel++
		}
		return nil
	case "enter":
		m.overlay = ovNone
		if m.pal.sel >= 0 && m.pal.sel < len(items) {
			return items[m.pal.sel].Run(m)
		}
		return nil
	}
	if changed, _ := m.pal.in.Key(k); changed {
		m.pal.sel = 0
	}
	return nil
}
