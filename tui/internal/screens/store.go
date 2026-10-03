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
	showInst bool // ver «Installed» aunque haya resultados
	list     listState
	marks    map[string]bool
	queue    []item
	armed    string
	rows     int // apps que caben en pantalla
}

type item struct {
	Source, ID, Name, Version, Desc string
	Installed                       bool
}

func (i item) key() string { return i.Source + ":" + i.ID }

const (
	storeHeader = 6 // caja de búsqueda (3) + hueco + pestañas + hueco
	storeItemH  = 3 // nombre, descripción y un respiro
)

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

func (s *Store) installedItems(env *core.Env) []item {
	out := make([]item, 0, len(env.Data.Apps))
	for _, a := range env.Data.Apps {
		out = append(out, item{Source: a.Source, ID: a.ID, Name: a.Name, Version: a.Version, Installed: true})
	}
	return out
}

// items son las filas de la lista: los resultados de la búsqueda, o lo instalado.
func (s *Store) items(env *core.Env) []item {
	if !s.searched || s.showInst {
		return s.installedItems(env)
	}
	inst := s.installed(env)
	out := make([]item, 0, len(s.results))
	for _, r := range s.results {
		it := item{Source: r.Source, ID: r.ID, Name: r.Name, Version: r.Version, Desc: r.Description}
		it.Installed = inst[it.key()]
		out = append(out, it)
	}
	return out
}

func (s *Store) search(env *core.Env) tea.Cmd {
	q := strings.TrimSpace(s.in.Text())
	if q == "" {
		s.searched, s.results, s.query, s.showInst = false, nil, "", false
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
			s.searched, s.showInst = true, false
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
	case isKey(m, "i"):
		if s.searched {
			s.showInst = !s.showInst
			s.list = listState{}
		}
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

func srcName(src string) string {
	switch src {
	case "nix":
		return "nixpkgs"
	case "flatpak":
		return "flathub"
	}
	return src
}

// searchBox es la caja de búsqueda: un bloque de tono propio, de tres filas, con
// aire alrededor del texto y una barra de acento cuando tiene el foco.
func (s *Store) searchBox(env *core.Env, w int) []ui.Line {
	p2 := ui.NewPainter(env.Theme, env.Theme.P.S2)
	bar := ui.S(p2.Fill, " ")
	hint := "/ to search "
	if s.focus {
		bar = ui.S(p2.Ac.Bold(true), "▌")
		hint = "⏎ search · esc cancel "
	}
	pad := ui.Line{L: ui.Spread([]ui.Seg{bar}, nil, w, p2.Fill)}
	left := append([]ui.Seg{bar, ui.S(p2.Ac, "  "+ui.G.Find+"  ")}, s.in.Segs(p2, s.focus, w-30)...)
	mid := ui.Line{L: ui.Spread(left, []ui.Seg{ui.S(p2.Mu, hint)}, w, p2.Fill)}
	return []ui.Line{pad, mid, pad}
}

// tabsRow son las dos vistas (resultados e instalado) y, a la derecha, el cargador.
func (s *Store) tabsRow(env *core.Env, w int, nRes, nInst int) ui.Line {
	p := env.P
	chip := func(label string, active bool) ui.Seg {
		if active {
			return ui.S(p.Btn, " "+label+" ")
		}
		return ui.S(p.Mu, " "+label+" ")
	}
	var left []ui.Seg
	if s.searched {
		left = append(left, chip(fmt.Sprintf("Results %d", nRes), !s.showInst), ui.Seg{T: " "})
	}
	left = append(left, chip(fmt.Sprintf("Installed %d", nInst), !s.searched || s.showInst))
	var right []ui.Seg
	for _, st := range []struct{ id, label string }{
		{"store.search", "Searching " + s.query},
		{"store.install", "Installing"},
		{"store.remove", "Removing"},
		{"data.apps", "Loading your apps"},
	} {
		if l, ok := working(env, st.id, st.label); ok {
			right = l.L
			break
		}
	}
	if n := len(s.queue); n > 0 {
		right = append(right, ui.S(p.Mu, fmt.Sprintf("  +%d queued", n)))
	}
	return ui.Line{L: ui.Spread(left, right, w, p.Fill)}
}

func (s *Store) Main(env *core.Env, w, h int) []ui.Line {
	p := env.P
	s.rows = max((h-storeHeader)/storeItemH, 1)
	its := s.items(env)
	nRes := len(s.results)
	lines := s.searchBox(env, w)
	lines = append(lines, gap(), s.tabsRow(env, w, nRes, len(env.Data.Apps)), gap())

	// Cargando: la lista aparece ya, con la forma que tendrá
	loading := env.Tasks.Loading("store.search") && (s.showInst || !s.searched)
	if (!env.Data.AppsLoaded && !s.searched) || loading {
		if err := env.Data.Err["apps"]; err != nil && !env.Data.AppsLoaded && !loading {
			return append(lines, failed(env, "Could not load your apps", err)...)
		}
		for i := 0; i < s.rows && i < 5; i++ {
			lines = append(lines, ui.Of(ui.S(p.Mu, "   "), ui.Skeleton(p, env.Frame+i*2, 14+i*3%9).L[0]), ui.Of(ui.S(p.Mu, "   "), ui.Skeleton(p, env.Frame+i*2+1, 30+i*5%12).L[0]), gap())
		}
		return lines
	}
	if len(its) == 0 {
		switch {
		case s.searched && !s.showInst:
			return append(lines, plain(env, "Nothing matches “"+s.query+"”."), muted(env, "Try another word, or fewer letters."))
		default:
			return append(lines, plain(env, "Nothing installed with maxor yet."), gap(), muted(env, "Press / and search: brave, btop, org.mozilla.firefox…"))
		}
	}
	from, to := s.list.window(len(its), s.rows)
	for i := from; i < to; i++ {
		it := its[i]
		sel := i == s.list.sel
		var mark ui.Seg
		switch {
		case it.Installed:
			mark = ui.S(p.Ok, ui.G.Tick+"  ")
		case s.marks[it.key()]:
			mark = ui.S(p.Ac, ui.G.On+"  ")
		default:
			mark = ui.S(p.Mu, ui.G.Off+"  ")
		}
		src := p.Ac
		if it.Source == "nix" {
			src = p.Ac2
		}
		ver := shortVersion(it.Version)
		right := []ui.Seg{ui.S(src, srcName(it.Source))}
		if ver != "" {
			right = append(right, ui.S(p.Mu, "  "+ver))
		}
		right = append(right, ui.Seg{T: " "})
		desc := it.Desc
		if desc == "" {
			desc = it.ID
		}
		lines = append(lines,
			ui.Line{L: []ui.Seg{mark, ui.S(p.Bold, it.Name)}, R: right, Sel: sel},
			ui.Line{L: []ui.Seg{ui.S(p.Mu, "     "+desc)}, Sel: sel},
			gap())
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
		return []ui.Line{heading(env, "Details"), gap(), muted(env, "Search with /"), gap(), muted(env, "Mark several with space"), muted(env, "and install them together.")}
	}
	it := its[s.list.sel]
	lines := []ui.Line{heading(env, "Details"), gap(), ui.T(p.Bold, it.Name), muted(env, it.ID), gap()}
	if it.Desc != "" {
		for _, l := range ui.Wrap(it.Desc, w) {
			lines = append(lines, plain(env, l))
		}
		lines = append(lines, gap())
	}
	lines = append(lines, ui.Of(ui.S(p.Mu, "source   "), ui.S(p.Text, srcName(it.Source))))
	if v := shortVersion(it.Version); v != "" {
		lines = append(lines, ui.Of(ui.S(p.Mu, "version  "), ui.S(p.Text, v)))
	}
	lines = append(lines, gap())
	switch {
	case it.Installed:
		lines = append(lines, ui.Of(ui.S(p.Ok, ui.G.Tick+" installed  "), button(env, false, "Remove  r")))
	case s.marks[it.key()]:
		lines = append(lines, ui.Of(button(env, true, "Install  ⏎"), space(1), ui.S(p.Mu, "marked")))
	default:
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
	h := []ui.Hint{{Key: "/", Action: "search"}, {Key: "↑↓", Action: "move"}, {Key: "space", Action: "mark"}, {Key: "⏎", Action: "install"}, {Key: "r", Action: "remove"}}
	if s.searched {
		h = append(h, ui.Hint{Key: "i", Action: "installed"})
	}
	return h
}

func (s *Store) Click(env *core.Env, x, y int) tea.Cmd {
	switch {
	case y < 3:
		s.focus = true
	case y == 4: // las pestañas Results / Installed
		if s.searched {
			res := len(fmt.Sprintf(" Results %d ", len(s.results)))
			s.showInst = x >= res+1
			s.list = listState{}
		}
	case y >= storeHeader:
		s.focus = false
		its := s.items(env)
		i := s.list.top + (y-storeHeader)/storeItemH
		if i >= 0 && i < len(its) {
			s.list.sel = i
		}
	}
	return nil
}

func (s *Store) Wheel(env *core.Env, dy int) tea.Cmd {
	s.list.move(dy*2, len(s.items(env)), max(s.rows, 1))
	return nil
}
