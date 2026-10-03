package screens

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/maxor"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Store busca, instala y quita apps (nixpkgs y Flathub). No instala nada por su
// cuenta: llama a `maxor search|install|remove … --json`.
type Store struct {
	core.Base
	in       ui.Input
	focus    bool
	query    string
	results  []maxor.Result
	searched bool
	list     listState
	marks    map[string]bool
	queue    []item
	armed    string
	rows     int
}

type item struct {
	Source, ID, Name, Version, Desc string
	Installed                       bool
}

func (i item) key() string { return i.Source + ":" + i.ID }

const storeItemsTop = 3 // filas de Main antes de la primera app

func NewStore() *Store {
	s := &Store{marks: map[string]bool{}}
	s.in.Placeholder = "Search apps in nixpkgs and Flathub…"
	return s
}

func (s *Store) ID() string    { return "store" }
func (s *Store) Title() string { return "Store" }

func (s *Store) Init(env *core.Env) tea.Cmd {
	if env.Data.AppsLoaded {
		return nil
	}
	return LoadApps(env, false)
}

func (s *Store) Captures() bool { return s.focus }

func (s *Store) installed(env *core.Env) map[string]bool {
	m := map[string]bool{}
	for _, a := range env.Data.Apps {
		m[a.Source+":"+a.ID] = true
	}
	return m
}

// items son las filas de la lista: los resultados de la búsqueda, o lo instalado.
func (s *Store) items(env *core.Env) []item {
	inst := s.installed(env)
	var out []item
	if s.searched {
		for _, r := range s.results {
			it := item{Source: r.Source, ID: r.ID, Name: r.Name, Version: r.Version, Desc: r.Description}
			it.Installed = inst[it.key()]
			out = append(out, it)
		}
		return out
	}
	for _, a := range env.Data.Apps {
		out = append(out, item{Source: a.Source, ID: a.ID, Name: a.Name, Version: a.Version, Installed: true})
	}
	return out
}

func (s *Store) search(env *core.Env) tea.Cmd {
	q := strings.TrimSpace(s.in.Text())
	if q == "" {
		s.searched, s.results, s.query = false, nil, ""
		s.list = listState{}
		return nil
	}
	s.query = q
	return env.Tasks.Start(task.Task{ID: "store.search", Label: "Searching " + q, Run: func(ctx context.Context) (any, error) {
		return env.Client.Search(ctx, q)
	}})
}

func (s *Store) startNext(env *core.Env) tea.Cmd {
	if len(s.queue) == 0 || env.Tasks.Running("store.install") {
		return nil
	}
	it := s.queue[0]
	s.queue = s.queue[1:]
	return env.Tasks.Start(task.Task{ID: "store.install", Label: "Installing " + it.Name, Run: func(ctx context.Context) (any, error) {
		return it, env.Client.Install(ctx, it.Source, it.ID)
	}})
}

func (s *Store) Update(env *core.Env, msg tea.Msg) (core.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case FocusSearchMsg:
		s.focus = true
		return s, nil
	case core.SearchMsg:
		s.in.Set(m.Query)
		s.focus = false
		return s, s.search(env)
	case task.DoneMsg:
		if m.Owner() != "store" {
			return s, nil
		}
		switch m.ID {
		case "store.search":
			if m.Err != nil {
				return s, core.Toast("bad", "Search failed: "+oneLine(m.Err.Error()))
			}
			s.results, _ = m.Value.([]maxor.Result)
			s.searched = true
			s.list = listState{}
			return s, nil
		case "store.install":
			it, _ := m.Value.(item)
			if m.Err != nil {
				return s, tea.Batch(core.Toast("bad", "Could not install "+it.Name+": "+oneLine(m.Err.Error())), s.startNext(env))
			}
			delete(s.marks, it.key())
			return s, tea.Batch(core.Toast("ok", "Installed "+it.Name), core.Note("ok", "Installed "+it.Name), LoadApps(env, true), s.startNext(env))
		case "store.remove":
			it, _ := m.Value.(item)
			if m.Err != nil {
				return s, core.Toast("bad", "Could not remove "+it.Name+": "+oneLine(m.Err.Error()))
			}
			return s, tea.Batch(core.Toast("ok", "Removed "+it.Name), core.Note("ok", "Removed "+it.Name), LoadApps(env, true))
		}
	case tea.KeyMsg:
		return s.key(env, m)
	}
	return s, nil
}

func (s *Store) key(env *core.Env, m tea.KeyMsg) (core.Screen, tea.Cmd) {
	if s.focus {
		switch {
		case isKey(m, "enter"):
			s.focus = false
			return s, s.search(env)
		case isKey(m, "esc"):
			s.focus = false
			return s, nil
		}
		s.in.Key(m)
		return s, nil
	}
	its := s.items(env)
	armed := s.armed
	s.armed = ""
	if d, mv := listKey(m); mv {
		s.list.move(d, len(its), max(s.rows, 1))
		return s, nil
	}
	cur := func() (item, bool) {
		if s.list.sel >= 0 && s.list.sel < len(its) {
			return its[s.list.sel], true
		}
		return item{}, false
	}
	switch {
	case isKey(m, "/"):
		s.focus = true
	case isKey(m, "c"):
		s.in.Set("")
		s.search(env)
	case isKey(m, " "):
		if it, ok := cur(); ok && !it.Installed {
			if s.marks[it.key()] {
				delete(s.marks, it.key())
			} else {
				s.marks[it.key()] = true
			}
		}
	case isKey(m, "enter"):
		var todo []item
		for _, it := range its {
			if s.marks[it.key()] && !it.Installed {
				todo = append(todo, it)
			}
		}
		if len(todo) == 0 {
			if it, ok := cur(); ok && !it.Installed {
				todo = []item{it}
			} else if ok && it.Installed {
				return s, core.Toast("info", it.Name+" is already installed")
			}
		}
		s.queue = append(s.queue, todo...)
		return s, s.startNext(env)
	case isKey(m, "r"):
		it, ok := cur()
		if !ok || !it.Installed {
			return s, nil
		}
		if armed != it.key() {
			s.armed = it.key()
			return s, core.Toast("warn", "Press r again to remove "+it.Name)
		}
		return s, env.Tasks.Start(task.Task{ID: "store.remove", Label: "Removing " + it.Name, Run: func(ctx context.Context) (any, error) {
			return it, env.Client.Remove(ctx, it.ID)
		}})
	}
	return s, nil
}

func (s *Store) Main(env *core.Env, w, h int) []ui.Line {
	p := env.P
	s.rows = h - storeItemsTop
	// Campo de búsqueda
	field := []ui.Seg{ui.S(p.Ac, ui.G.Find+"  ")}
	field = append(field, s.in.Segs(p, s.focus, w-6)...)
	lines := []ui.Line{{L: field}}

	// Estado: el mismo cargador de siempre
	status := gap()
	for _, st := range []struct{ id, label string }{
		{"store.search", "Searching " + s.query},
		{"store.install", "Installing"},
		{"store.remove", "Removing"},
		{"data.apps", "Loading your apps"},
	} {
		if l, ok := working(env, st.id, st.label); ok {
			status = l
			break
		}
	}
	if n := len(s.queue); n > 0 {
		status = ui.Of(append(status.L, ui.S(p.Mu, fmt.Sprintf("  +%d queued", n)))...)
	}
	lines = append(lines, status)

	its := s.items(env)
	switch {
	case !s.searched && !env.Data.AppsLoaded:
		if err := env.Data.Err["apps"]; err != nil {
			return append(lines, failed(env, "Could not load your apps", err)...)
		}
		return append(append(lines, heading(env, "Installed with maxor")), skeletonRows(env, 5, 12, 10)...)
	case s.searched:
		// el texto de la búsqueda se muestra tal como se escribió, sin pasarlo a mayúsculas
		lines = append(lines, muted(env, fmt.Sprintf("%d results for “%s”", len(its), s.query)))
	default:
		lines = append(lines, heading(env, fmt.Sprintf("Installed with maxor · %d", len(its))))
	}
	if len(its) == 0 {
		if s.searched {
			return append(lines, muted(env, "Nothing matches “"+s.query+"”."), muted(env, "Try another word, or fewer letters."))
		}
		return append(lines, muted(env, "Nothing installed with maxor yet."), muted(env, "Press / to search."))
	}
	from, to := s.list.window(len(its), s.rows)
	for i := from; i < to; i++ {
		it := its[i]
		var mark ui.Seg
		switch {
		case it.Installed:
			mark = ui.S(p.Ok, ui.G.Tick+" ")
		case s.marks[it.key()]:
			mark = ui.S(p.Ac, ui.G.On+" ")
		default:
			mark = ui.S(p.Mu, ui.G.Off+" ")
		}
		src := p.Ac
		if it.Source == "nix" {
			src = p.Ac2
		}
		srcName := it.Source
		if srcName == "nix" {
			srcName = "nixpkgs"
		} else if srcName == "flatpak" {
			srcName = "flathub"
		}
		lines = append(lines, ui.Line{
			L:   []ui.Seg{mark, ui.S(p.Text, it.Name)},
			R:   []ui.Seg{ui.S(src, srcName), ui.S(p.Mu, " "+shortVersion(it.Version)+" ")},
			Sel: i == s.list.sel,
		})
	}
	return lines
}

// shortVersion quita el nombre del paquete que `nix profile` antepone («btop-1.4.7»).
func shortVersion(v string) string {
	if i := strings.LastIndex(v, "-"); i >= 0 && i+1 < len(v) && v[i+1] >= '0' && v[i+1] <= '9' {
		return v[i+1:]
	}
	return v
}

func (s *Store) Side(env *core.Env, w, h int) []ui.Line {
	p := env.P
	its := s.items(env)
	if len(its) == 0 || s.list.sel >= len(its) {
		return []ui.Line{heading(env, "Details"), gap(), muted(env, "Search with /")}
	}
	it := its[s.list.sel]
	lines := []ui.Line{heading(env, "Details"), gap(), ui.T(p.Bold, it.Name), muted(env, it.ID), gap()}
	if it.Desc != "" {
		for _, l := range ui.Wrap(it.Desc, w) {
			lines = append(lines, plain(env, l))
		}
		lines = append(lines, gap())
	}
	src := it.Source
	if src == "nix" {
		src = "nixpkgs"
	} else if src == "flatpak" {
		src = "flathub"
	}
	lines = append(lines, ui.Of(ui.S(p.Mu, "source   "), ui.S(p.Text, src)))
	if it.Version != "" {
		lines = append(lines, ui.Of(ui.S(p.Mu, "version  "), ui.S(p.Text, shortVersion(it.Version))))
	}
	lines = append(lines, gap())
	if it.Installed {
		lines = append(lines, ui.Of(ui.S(p.Ok, ui.G.Tick+" installed  "), button(env, false, "Remove  r")))
	} else if s.marks[it.key()] {
		lines = append(lines, ui.Of(button(env, true, "Install  ⏎"), space(1), ui.S(p.Mu, "marked")))
	} else {
		lines = append(lines, ui.Of(button(env, true, "Install  ⏎"), space(1), button(env, false, "Mark  space")))
	}
	if n := len(s.marks); n > 0 {
		lines = append(lines, gap(), muted(env, fmt.Sprintf("%d marked · ⏎ installs them all", n)))
	}
	return lines
}

func (s *Store) Hints(env *core.Env) []ui.Hint {
	if s.focus {
		return []ui.Hint{{Key: "⏎", Action: "search"}, {Key: "esc", Action: "cancel"}}
	}
	return []ui.Hint{{Key: "/", Action: "search"}, {Key: "↑↓", Action: "move"}, {Key: "space", Action: "mark"}, {Key: "⏎", Action: "install"}, {Key: "r", Action: "remove"}}
}

func (s *Store) Click(env *core.Env, x, y int) tea.Cmd {
	if y == 0 {
		s.focus = true
		return nil
	}
	its := s.items(env)
	i := s.list.top + y - storeItemsTop
	if i >= 0 && i < len(its) {
		s.list.sel = i
		s.focus = false
	}
	return nil
}

func (s *Store) Wheel(env *core.Env, dy int) tea.Cmd {
	s.list.move(dy*2, len(s.items(env)), max(s.rows, 1))
	return nil
}
