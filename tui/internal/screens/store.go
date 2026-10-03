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
	zone     int // dónde está el foco: caja de búsqueda, pestañas o lista
	chip     int // pestaña con el foco (posición en chips())
	src      string // filtro por origen: "", "nix" o "flatpak"
	chipX    [][2]int
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

// Zonas del foco: se recorren con ↑ y ↓.
const (
	zSearch = iota
	zChips
	zList
)

type chipDef struct {
	label string
	view  int    // 1 resultados · 2 instalado (0 si es un filtro)
	src   string // valor del filtro cuando view == 0
	group bool   // es un filtro de origen
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
	s := &Store{marks: map[string]bool{}, zone: zList}
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

// Captures: con el foco en la caja de búsqueda todo lo que se teclea es texto.
func (s *Store) Captures() bool { return s.zone == zSearch }

// OwnsHorizontal: en las pestañas, ← y → cambian de pestaña de la Tienda, no de pantalla.
func (s *Store) OwnsHorizontal() bool { return s.zone == zChips }

// chips son las pestañas: la vista (resultados o instalado) y, tras buscar, el origen.
func (s *Store) chips(env *core.Env) []chipDef {
	var out []chipDef
	if s.searched {
		out = append(out, chipDef{label: fmt.Sprintf("Results %d", len(s.results)), view: 1})
	}
	out = append(out, chipDef{label: fmt.Sprintf("Installed %d", len(env.Data.Apps)), view: 2})
	if s.searched {
		out = append(out, chipDef{label: "All", group: true}, chipDef{label: "nixpkgs", src: "nix", group: true}, chipDef{label: "flathub", src: "flatpak", group: true})
	}
	return out
}

// chipActive dice si una pestaña es la elegida (su vista o su filtro).
func (s *Store) chipActive(c chipDef) bool {
	switch {
	case c.group:
		return c.src == s.src
	case c.view == 2:
		return !s.searched || s.showInst
	}
	return s.searched && !s.showInst
}

// pickChip elige la pestaña i: cambia la vista o el filtro al instante.
func (s *Store) pickChip(env *core.Env, i int) {
	cs := s.chips(env)
	if i < 0 || i >= len(cs) {
		return
	}
	s.chip = i
	c := cs[i]
	switch {
	case c.group:
		s.src, s.showInst = c.src, false // un filtro siempre se ve sobre los resultados
	case c.view == 2:
		s.showInst = true
	default:
		s.showInst = false
	}
	s.list = listState{}
}

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
		if s.src != "" && r.Source != s.src {
			continue
		}
		it := item{Source: r.Source, ID: r.ID, Name: r.Name, Version: r.Version, Desc: r.Description}
		it.Installed = inst[it.key()]
		out = append(out, it)
	}
	return out
}

func (s *Store) search(env *core.Env) tea.Cmd {
	q := strings.TrimSpace(s.in.Text())
	if q == "" {
		s.searched, s.results, s.query, s.showInst, s.src = false, nil, "", false, ""
		s.list, s.chip = listState{}, 0
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
		s.zone = zSearch
		return s, nil
	case core.SearchMsg:
		s.in.Set(m.Query)
		s.zone = zList
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
			s.searched, s.showInst, s.src = true, false, ""
			s.list, s.chip = listState{}, 0
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
	switch s.zone {
	case zSearch:
		switch {
		case isKey(m, "enter"):
			s.zone = zList
			return s, s.search(env)
		case isKey(m, "esc"):
			s.zone = zList
			return s, nil
		case isKey(m, "down"):
			s.zone = zList
			if s.searched {
				s.zone = zChips
			}
			return s, nil
		case isKey(m, "up"):
			return s, nil
		}
		s.in.Key(m)
		return s, nil
	case zChips:
		switch {
		case isKey(m, "left", "h"):
			s.pickChip(env, s.chip-1)
			return s, nil
		case isKey(m, "right", "l"):
			s.pickChip(env, s.chip+1)
			return s, nil
		case isKey(m, "up", "k"):
			s.zone = zSearch
			return s, nil
		case isKey(m, "down", "j", "enter"):
			s.zone = zList
			return s, nil
		case isKey(m, "/"):
			s.zone = zSearch
			return s, nil
		case isKey(m, "esc"):
			s.zone = zList
			return s, nil
		}
		return s, nil
	}
	its := s.items(env)
	if isKey(m, "up", "k") && s.list.sel == 0 {
		s.zone = zSearch
		if s.searched {
			s.zone = zChips
		}
		return s, nil
	}
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
		s.zone = zSearch
	case isKey(m, "i"):
		if s.searched {
			cs := s.chips(env)
			for i, c := range cs {
				if c.view != 0 && !s.chipActive(c) {
					s.pickChip(env, i)
					break
				}
			}
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
	hint := "↑ or / to search "
	if s.zone == zSearch {
		bar = ui.S(p2.Ac.Bold(true), "▌")
		hint = "⏎ search · ↓ results · esc "
	}
	pad := ui.Line{L: ui.Spread([]ui.Seg{bar}, nil, w, p2.Fill)}
	left := append([]ui.Seg{bar, ui.S(p2.Ac, "  "+ui.G.Find+"  ")}, s.in.Segs(p2, s.zone == zSearch, w-34)...)
	mid := ui.Line{L: ui.Spread(left, []ui.Seg{ui.S(p2.Mu, hint)}, w, p2.Fill)}
	return []ui.Line{pad, mid, pad}
}

// tabsRow son las dos vistas (resultados e instalado) y, a la derecha, el cargador.
func (s *Store) tabsRow(env *core.Env, w int, nRes, nInst int) ui.Line {
	p := env.P
	var left []ui.Seg
	s.chipX = s.chipX[:0]
	x := 0
	for i, c := range s.chips(env) {
		active, focused := s.chipActive(c), s.zone == zChips && i == s.chip
		l, r := " ", " "
		st := p.Mu
		if active {
			st = p.Btn
		}
		if focused {
			l, r = "‹", "›"
			st = p.Btn.Bold(true).Underline(true)
			if !active {
				st = p.Ac.Bold(true)
			}
		}
		if c.group && (i == 0 || !s.chips(env)[i-1].group) {
			left = append(left, ui.S(p.Mu, " │ "))
			x += 3
		}
		seg := ui.S(st, l+c.label+r)
		w := len([]rune(l + c.label + r))
		s.chipX = append(s.chipX, [2]int{x, x + w})
		x += w + 1
		left = append(left, seg, ui.Seg{T: " "})
	}
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
		sel := i == s.list.sel && s.zone == zList
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
	switch s.zone {
	case zSearch:
		return []ui.Hint{{Key: "⏎", Action: "search"}, {Key: "↓", Action: "results"}, {Key: "esc", Action: "leave"}}
	case zChips:
		return []ui.Hint{{Key: "←→", Action: "switch"}, {Key: "↑", Action: "search"}, {Key: "↓", Action: "list"}}
	}
	return []ui.Hint{{Key: "↑", Action: "search"}, {Key: "space", Action: "mark"}, {Key: "⏎", Action: "install"}, {Key: "r", Action: "remove"}, {Key: "/", Action: "search"}}
}

func (s *Store) Click(env *core.Env, x, y int) tea.Cmd {
	switch {
	case y < 3:
		s.zone = zSearch
	case y == 4: // las pestañas
		s.zone = zChips
		for i, r := range s.chipX {
			if x >= r[0] && x < r[1]+1 {
				s.pickChip(env, i)
			}
		}
	case y >= storeHeader:
		s.zone = zList
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

// Zone dice dónde está el foco: «search», «chips» o «list» (para las pruebas).
func (s *Store) Zone() string {
	return [...]string{"search", "chips", "list"}[s.zone]
}
