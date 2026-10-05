package screens

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/maxor"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Store busca, instala, actualiza y quita apps (nixpkgs y Flathub). No toca nada por
// su cuenta: llama a `maxor search|install|remove|apps … --json`.
//
// Las apps instaladas se ven con casillas: se marcan varias y se actualizan o se
// quitan juntas. Todo lo que cambia paquetes pasa por una cola de trabajos que se
// ejecuta de uno en uno (nix y flatpak no admiten dos operaciones a la vez).
type Store struct {
	core.Base
	in       ui.Input
	zone     int // dónde está el foco: caja de búsqueda, pestañas o lista
	chip     int // pestaña con el foco (posición en chips())
	chipX    [][2]int
	hdr      int // filas de cabecera de la lista (7, o 4 en modo compacto)
	itemH    int // filas por app (3, o 2 en modo compacto)
	actX     [][2]int // columnas de cada botón de la barra de acciones
	actKey   []string // la tecla que hace lo mismo que cada botón
	query    string
	results  []maxor.Result
	searched bool
	showInst bool // ver «Installed» aunque haya resultados
	list     listState
	marks    map[string]bool
	queue    []job
	rows     int               // apps que caben en pantalla
	busy     map[string]string // app → «queued», «installing», «updating» o «removing»
	ask      *purgeAsk         // pregunta pendiente: ¿borrar también sus datos?
	askAcc   []purgeEntry      // datos que dejaron las apps quitadas en esta tanda
	menu     *actionMenu       // menú de acciones sobre las apps elegidas
}

// OpenAppMsg pide abrir una app instalada (la paleta de comandos).
type OpenAppMsg struct{ ID, Name string }

// job es una operación de paquetes en la cola.
type job struct {
	it    item
	op    string // install | update | remove
	purge bool   // al quitar, borrar también sus carpetas
}

// actionMenu es el menú sobre una o varias apps instaladas: se maneja con ↑ ↓ e Intro.
type actionMenu struct {
	items   []item
	sel     int
	confirm bool // pidió borrar los datos: falta el sí final
	data    []maxor.Leftover
	loaded  bool
	dataKey string
}

type menuEntry struct{ label, hint, kind, key string }

// purgeEntry son las carpetas que dejó una app quitada.
type purgeEntry struct {
	ID, Name string
	Left     []maxor.Leftover
}

// purgeAsk es la pregunta tras quitar apps que dejaron carpetas en tu casa.
type purgeAsk struct{ Entries []purgeEntry }

// removed es el resultado de quitar una app.
type removed struct {
	it   item
	left []maxor.Leftover
}

// Zonas del foco: se recorren con ↑ y ↓.
const (
	zSearch = iota
	zChips
	zList
)

type chipDef struct {
	label string
	view  int // 1 resultados · 2 instalado
}

type item struct {
	Source, ID, Name, Version, Desc string
	Installed                       bool
}

func (i item) key() string { return i.Source + ":" + i.ID }

const (
	storeHeader = 7 // caja de búsqueda (3) + hueco + pestañas + barra de acciones + hueco
	storeItemH  = 3 // nombre, descripción y un respiro
	// En una ventana baja (una de cuatro en pantalla) la cabecera se encoge a 4 filas y cada
	// app ocupa 2 en vez de 3, para que quepan más en la lista.
	compactBelow = 22
)

func NewStore() *Store {
	s := &Store{marks: map[string]bool{}, zone: zList, busy: map[string]string{}, hdr: storeHeader, itemH: storeItemH}
	s.in.Placeholder = tr("Search apps in nixpkgs and Flathub…")
	return s
}

func (s *Store) ID() string    { return "store" }
func (s *Store) Title() string { return tr("Store") }

func (s *Store) Init(env *core.Env) tea.Cmd {
	var cmds []tea.Cmd
	if !env.Data.AppsLoaded {
		cmds = append(cmds, LoadApps(env, false))
	}
	if !env.Data.UpdatesKnown {
		cmds = append(cmds, LoadAppUpdates(env, true))
	}
	return tea.Batch(cmds...)
}

// updateFor dice si la app tiene una versión nueva y cuál.
func (s *Store) updateFor(env *core.Env, it item) (maxor.AppUpdate, bool) {
	for _, u := range env.Data.AppUpdates {
		if u.Source == it.Source && u.ID == it.ID {
			return u, true
		}
	}
	return maxor.AppUpdate{}, false
}

// pkgBusy: nix y flatpak no admiten dos operaciones a la vez.
func pkgBusy(env *core.Env) bool {
	for _, id := range []string{"store.install", "store.remove", "store.purge", "store.update"} {
		if env.Tasks.Running(id) {
			return true
		}
	}
	return false
}

// installedView dice si la lista enseña lo instalado (y no los resultados de una búsqueda).
func (s *Store) installedView() bool { return !s.searched || s.showInst }

// targets son las apps sobre las que actúa una orden: las marcadas, o la de la fila.
func (s *Store) targets(its []item) []item {
	var out []item
	for _, it := range its {
		if s.marks[it.key()] && (it.Installed == s.installedView()) {
			out = append(out, it)
		}
	}
	if len(out) == 0 && s.list.sel >= 0 && s.list.sel < len(its) {
		out = []item{its[s.list.sel]}
	}
	return out
}

func (s *Store) markedCount(its []item) (n int) {
	for _, it := range its {
		if s.marks[it.key()] && (it.Installed == s.installedView()) {
			n++
		}
	}
	return
}

// enqueue añade trabajos a la cola, sin repetir lo que ya está en marcha, y empieza.
func (s *Store) enqueue(env *core.Env, jobs []job) tea.Cmd {
	var skipped string
	added := 0
	for _, j := range jobs {
		if st := s.busy[j.it.key()]; st != "" {
			skipped = tr("%s is already %s", j.it.Name, map[string]string{"queued": tr("queued"), "installing": tr("being installed"), "removing": tr("being removed"), "updating": tr("being updated")}[st])
			continue
		}
		s.busy[j.it.key()] = "queued"
		s.queue = append(s.queue, j)
		added++
	}
	cmd := s.startNext(env)
	if skipped != "" && added == 0 {
		return core.Toast("info", skipped)
	}
	return cmd
}

func (s *Store) startNext(env *core.Env) tea.Cmd {
	if len(s.queue) == 0 || pkgBusy(env) {
		return nil
	}
	j := s.queue[0]
	s.queue = s.queue[1:]
	it := j.it
	switch j.op {
	case "update":
		s.busy[it.key()] = "updating"
		return env.Tasks.Start(task.Task{ID: "store.update", Label: tr("Updating %s", it.Name), Run: func(ctx context.Context) (any, error) {
			return it, env.Client.UpdateApp(ctx, it.ID)
		}})
	case "remove":
		s.busy[it.key()] = "removing"
		purge := j.purge
		return env.Tasks.Start(task.Task{ID: "store.remove", Label: tr("Removing %s", it.Name), Run: func(ctx context.Context) (any, error) {
			left, err := env.Client.Remove(ctx, it.ID)
			if err == nil && purge && len(left) > 0 {
				err = env.Client.Purge(ctx, it.ID)
				left = nil
			}
			return removed{it: it, left: left}, err
		}})
	}
	s.busy[it.key()] = "installing"
	return env.Tasks.Start(task.Task{ID: "store.install", Label: tr("Installing %s", it.Name), Run: func(ctx context.Context) (any, error) {
		return it, env.Client.Install(ctx, it.Source, it.ID)
	}})
}

// finishRemovals deja la pregunta de los datos cuando ya no queda nada por quitar.
func (s *Store) finishRemovals(env *core.Env) tea.Cmd {
	// aún hay una quitándose o en cola: la pregunta espera a la última
	if env.Tasks.Running("store.remove") {
		return nil
	}
	for _, j := range s.queue {
		if j.op == "remove" {
			return nil
		}
	}
	if len(s.askAcc) == 0 {
		return nil
	}
	s.ask = &purgeAsk{Entries: s.askAcc}
	var total int64
	for _, e := range s.askAcc {
		total += sumBytes(e.Left)
	}
	s.askAcc = nil
	return core.Toast("warn", fmt.Sprintf(tr("Their data stays on disk (%s): y deletes it, n keeps it"), humanBytes(total)))
}

func (s *Store) menuEntries(env *core.Env) []menuEntry {
	items := s.menu.items
	single := len(items) == 1
	var out []menuEntry
	if single {
		out = append(out, menuEntry{tr("Open"), tr("start it now"), "open", "o"})
	}
	var upd []maxor.AppUpdate
	for _, it := range items {
		if u, ok := s.updateFor(env, it); ok {
			upd = append(upd, u)
		}
	}
	if len(upd) > 0 {
		hint := fmt.Sprintf(tr("%d of %d have a new version"), len(upd), len(items))
		if single {
			hint = "to " + upd[0].Latest
			if upd[0].Current != "" {
				hint = upd[0].Current + " → " + upd[0].Latest
			}
		}
		out = append(out, menuEntry{tr("Update"), hint, "update", "u"})
	}
	out = append(out, menuEntry{tr("Remove"), tr("keeps your saves and settings"), "remove", "r"})
	hint := tr("removes the app and its folders")
	if s.menu.loaded && len(s.menu.data) > 0 {
		hint = tr("frees %s", humanBytes(sumBytes(s.menu.data)))
	} else if s.menu.loaded {
		hint = tr("no extra folders to delete")
	}
	label := tr("Remove and delete its data")
	if !single {
		label = tr("Remove and delete their data")
	}
	return append(out, menuEntry{label, hint, "purge", "d"})
}

func (s *Store) menuJobs(env *core.Env, op string, purge bool) []job {
	var jobs []job
	for _, it := range s.menu.items {
		if op == "update" {
			if _, ok := s.updateFor(env, it); !ok {
				continue
			}
		}
		jobs = append(jobs, job{it: it, op: op, purge: purge})
	}
	return jobs
}

func (s *Store) menuKey(env *core.Env, m tea.KeyMsg) (core.Screen, tea.Cmd) {
	mn := s.menu
	if mn.confirm {
		switch {
		case isKey(m, "enter", "y"):
			jobs := s.menuJobs(env, "remove", true)
			s.menu = nil
			return s, s.enqueue(env, jobs)
		case isKey(m, "esc", "n"):
			mn.confirm = false
		}
		return s, nil
	}
	ents := s.menuEntries(env)
	run := func(kind string) (core.Screen, tea.Cmd) {
		switch kind {
		case "open":
			it := mn.items[0]
			s.menu = nil
			return s, env.Tasks.Start(task.Task{ID: "store.open", Label: tr("Opening %s", it.Name), Run: func(ctx context.Context) (any, error) {
				return it, env.Client.OpenApp(ctx, it.ID)
			}})
		case "update":
			jobs := s.menuJobs(env, "update", false)
			s.menu = nil
			return s, s.enqueue(env, jobs)
		case "remove":
			jobs := s.menuJobs(env, "remove", false)
			s.menu = nil
			return s, s.enqueue(env, jobs)
		case "purge":
			if mn.loaded && len(mn.data) == 0 {
				jobs := s.menuJobs(env, "remove", true)
				s.menu = nil
				return s, s.enqueue(env, jobs)
			}
			mn.confirm = true
		}
		return s, nil
	}
	switch {
	case isKey(m, "esc", "q"):
		s.menu = nil
	case isKey(m, "up", "k"):
		mn.sel = (mn.sel + len(ents) - 1) % len(ents)
	case isKey(m, "down", "j"):
		mn.sel = (mn.sel + 1) % len(ents)
	case isKey(m, "enter"):
		return run(ents[mn.sel].kind)
	default:
		// atajos para quien ya los conoce: o, u, r, d
		for _, e := range ents {
			if isKey(m, e.key) {
				return run(e.kind)
			}
		}
	}
	return s, nil
}

// openMenu abre el menú sobre estas apps y mira qué carpetas tienen en tu casa.
func (s *Store) openMenu(env *core.Env, items []item, sel string) tea.Cmd {
	s.menu = &actionMenu{items: items}
	for i, e := range s.menuEntries(env) {
		if e.kind == sel {
			s.menu.sel = i
		}
	}
	var keys []string
	for _, it := range items {
		keys = append(keys, it.key())
	}
	key := strings.Join(keys, ",")
	s.menu.dataKey = key
	return env.Tasks.Start(task.Task{ID: "store.data", Label: tr("Looking at their folders"), Quiet: true, Run: func(ctx context.Context) (any, error) {
		var all []maxor.Leftover
		for _, it := range items {
			l, err := env.Client.AppData(ctx, it.ID)
			if err != nil {
				return appData{key: key}, err
			}
			all = append(all, l...)
		}
		return appData{key: key, left: all}, nil
	}})
}

type appData struct {
	key  string
	left []maxor.Leftover
}

// Captures: con el foco en la caja de búsqueda todo lo que se teclea es texto.
func (s *Store) Captures() bool { return s.zone == zSearch }

// OwnsHorizontal: en las pestañas, ← y → cambian de pestaña de la Tienda, no de pantalla.
func (s *Store) OwnsHorizontal() bool { return s.zone == zChips }

// chips son las pestañas: la vista (resultados o instalado) y, tras buscar, el origen.
func (s *Store) chips(env *core.Env) []chipDef {
	var out []chipDef
	if s.searched {
		out = append(out, chipDef{label: fmt.Sprintf(tr("Results %d"), len(s.results)), view: 1})
	}
	inst := fmt.Sprintf(tr("Installed %d"), len(env.Data.Apps))
	if n := len(env.Data.AppUpdates); n > 0 {
		inst += fmt.Sprintf(" %s%d", ui.G.Up, n)
	}
	return append(out, chipDef{label: inst, view: 2})
}

// chipActive dice si una pestaña es la elegida.
func (s *Store) chipActive(c chipDef) bool {
	if c.view == 2 {
		return s.installedView()
	}
	return s.searched && !s.showInst
}

// pickChip elige la pestaña i: cambia la vista al instante.
func (s *Store) pickChip(env *core.Env, i int) {
	cs := s.chips(env)
	if i < 0 || i >= len(cs) {
		return
	}
	s.chip = i
	c := cs[i]
	s.showInst = c.view == 2
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
	if s.installedView() {
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
		s.list, s.chip = listState{}, 0
		return nil
	}
	s.query = q
	return env.Tasks.Start(task.Task{ID: "store.search", Label: tr("Searching %s", q), Run: func(ctx context.Context) (any, error) {
		return env.Client.Search(ctx, q)
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
	case OpenAppMsg:
		it := item{ID: m.ID, Name: m.Name}
		return s, env.Tasks.Start(task.Task{ID: "store.open", Label: tr("Opening %s", m.Name), Run: func(ctx context.Context) (any, error) {
			return it, env.Client.OpenApp(ctx, it.ID)
		}})
	case task.DoneMsg:
		if m.Owner() != "store" {
			return s, nil
		}
		switch m.ID {
		case "store.search":
			if m.Err != nil {
				return s, core.Toast("bad", tr("Search failed: %s", oneLine(m.Err.Error())))
			}
			s.results, _ = m.Value.([]maxor.Result)
			s.searched, s.showInst = true, false
			s.list, s.chip = listState{}, 0
			return s, nil
		case "store.install":
			it, _ := m.Value.(item)
			delete(s.busy, it.key())
			if m.Err != nil {
				return s, tea.Batch(core.Toast("bad", tr("Could not install %s: %s", it.Name, oneLine(m.Err.Error()))), s.startNext(env))
			}
			delete(s.marks, it.key())
			return s, tea.Batch(core.Toast("ok", tr("Installed %s. Find it in Super+Space, or press ⏎ here and choose Open", it.Name)), core.Note("ok", tr("Installed %s", it.Name)), LoadApps(env, true), LoadAppUpdates(env, true), s.startNext(env))
		case "store.remove":
			r, _ := m.Value.(removed)
			delete(s.busy, r.it.key())
			delete(s.marks, r.it.key())
			if m.Err != nil {
				return s, tea.Batch(core.Toast("bad", tr("Could not remove %s: %s", r.it.Name, oneLine(m.Err.Error()))), s.startNext(env), s.finishRemovals(env))
			}
			cmds := []tea.Cmd{core.Note("ok", tr("Removed %s", r.it.Name)), LoadApps(env, true), LoadAppUpdates(env, true), s.startNext(env)}
			if len(r.left) > 0 {
				s.askAcc = append(s.askAcc, purgeEntry{ID: r.it.ID, Name: r.it.Name, Left: r.left})
			} else {
				cmds = append(cmds, core.Toast("ok", tr("Removed %s", r.it.Name)))
			}
			cmds = append(cmds, s.finishRemovals(env))
			return s, tea.Batch(cmds...)
		case "store.data":
			if d, ok := m.Value.(appData); ok && s.menu != nil && s.menu.dataKey == d.key {
				s.menu.data, s.menu.loaded = d.left, true
			}
			return s, nil
		case "store.open":
			it, _ := m.Value.(item)
			if m.Err != nil {
				return s, core.Toast("bad", tr("Could not open %s: %s", it.Name, oneLine(m.Err.Error())))
			}
			return s, core.Toast("ok", tr("Opening %s… give it a few seconds", it.Name))
		case "store.update":
			it, _ := m.Value.(item)
			delete(s.busy, it.key())
			delete(s.marks, it.key())
			if m.Err != nil {
				return s, tea.Batch(core.Toast("bad", tr("Could not update %s: %s", it.Name, oneLine(m.Err.Error()))), s.startNext(env))
			}
			return s, tea.Batch(core.Toast("ok", tr("Updated %s", it.Name)), core.Note("ok", tr("Updated %s", it.Name)), LoadApps(env, true), LoadAppUpdates(env, true), s.startNext(env))
		case "store.purge":
			name, _ := m.Value.(string)
			if m.Err != nil {
				return s, core.Toast("bad", tr("Could not delete the data of %s: %s", name, oneLine(m.Err.Error())))
			}
			return s, tea.Batch(core.Toast("ok", tr("Deleted the data of %s", name)), core.Note("ok", tr("Deleted the data of %s", name)), s.startNext(env))
		}
	case tea.KeyMsg:
		return s.key(env, m)
	}
	return s, nil
}

func (s *Store) key(env *core.Env, m tea.KeyMsg) (core.Screen, tea.Cmd) {
	if s.menu != nil && s.zone != zList {
		s.menu = nil
	}
	if s.menu != nil {
		return s.menuKey(env, m)
	}
	if s.ask != nil && s.zone != zSearch {
		ask := s.ask
		switch {
		case isKey(m, "y"):
			s.ask = nil
			var names []string
			for _, e := range ask.Entries {
				names = append(names, e.Name)
			}
			label := strings.Join(names, ", ")
			return s, env.Tasks.Start(task.Task{ID: "store.purge", Label: tr("Deleting the data of %s", label), Run: func(ctx context.Context) (any, error) {
				for _, e := range ask.Entries {
					if err := env.Client.Purge(ctx, e.ID); err != nil {
						return label, err
					}
				}
				return label, nil
			}})
		case isKey(m, "n", "esc"):
			s.ask = nil
			return s, core.Toast("info", tr("Kept their data. Delete it later: maxor remove <app> --purge"))
		}
	}
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
	inst := s.installedView()
	switch {
	case isKey(m, "/"):
		s.zone = zSearch
	case isKey(m, "i"):
		if s.searched {
			for i, c := range s.chips(env) {
				if c.view != 0 && !s.chipActive(c) {
					s.pickChip(env, i)
					break
				}
			}
		}
	case isKey(m, "c"):
		s.in.Set("")
		s.search(env)
	case isKey(m, "esc"):
		s.marks = map[string]bool{}
	case isKey(m, " "):
		// en «Installed» se marcan las instaladas; en los resultados, las que faltan
		if it, ok := cur(); ok && it.Installed == inst {
			if s.marks[it.key()] {
				delete(s.marks, it.key())
			} else {
				s.marks[it.key()] = true
			}
		}
	case isKey(m, "a"):
		all := len(its) > 0
		for _, it := range its {
			if it.Installed == inst && !s.marks[it.key()] {
				all = false
			}
		}
		for _, it := range its {
			if it.Installed != inst {
				continue
			}
			if all {
				delete(s.marks, it.key())
			} else {
				s.marks[it.key()] = true
			}
		}
	case isKey(m, "enter"):
		if inst {
			if t := s.targets(its); len(t) > 0 {
				return s, s.openMenu(env, t, "open")
			}
			return s, nil
		}
		var todo []job
		for _, it := range s.targets(its) {
			if it.Installed {
				return s, s.openMenu(env, []item{it}, "open")
			}
			todo = append(todo, job{it: it, op: "install"})
		}
		return s, s.enqueue(env, todo)
	case isKey(m, "u"):
		var todo []job
		for _, it := range s.targets(its) {
			if _, ok := s.updateFor(env, it); ok && it.Installed {
				todo = append(todo, job{it: it, op: "update"})
			}
		}
		if len(todo) == 0 {
			return s, core.Toast("ok", tr("Nothing to update there: it is already up to date"))
		}
		return s, s.enqueue(env, todo)
	case isKey(m, "R"):
		return s, tea.Batch(core.Toast("info", tr("Looking for new versions…")), LoadAppUpdatesFresh(env))
	case isKey(m, "U"):
		var todo []job
		for _, it := range s.installedItems(env) {
			if _, ok := s.updateFor(env, it); ok {
				todo = append(todo, job{it: it, op: "update"})
			}
		}
		if len(todo) == 0 {
			return s, core.Toast("ok", tr("Everything is up to date"))
		}
		return s, s.enqueue(env, todo)
	case isKey(m, "r"):
		var t []item
		for _, it := range s.targets(its) {
			if it.Installed {
				t = append(t, it)
			}
		}
		if len(t) == 0 {
			return s, nil
		}
		return s, s.openMenu(env, t, "remove")
	}
	return s, nil
}

// menuLines dibuja el menú de acciones en el panel lateral.
func (s *Store) menuLines(env *core.Env, w int) []ui.Line {
	p := env.P
	mn := s.menu
	title := mn.items[0].Name
	if len(mn.items) > 1 {
		title = fmt.Sprintf(tr("%d apps"), len(mn.items))
	}
	lines := []ui.Line{heading(env, tr("What do you want to do?")), gap(), ui.T(p.Bold, title)}
	if len(mn.items) > 1 {
		for i, it := range mn.items {
			if i == 5 {
				lines = append(lines, muted(env, fmt.Sprintf(tr("and %d more"), len(mn.items)-5)))
				break
			}
			lines = append(lines, muted(env, it.Name))
		}
	}
	lines = append(lines, gap())
	if mn.confirm {
		size := tr("their folders")
		if len(mn.data) > 0 {
			size = humanBytes(sumBytes(mn.data))
		}
		lines = append(lines, ui.T(p.Warn.Bold(true), ui.G.Warn+" "+tr("This cannot be undone")))
		for _, l := range ui.Wrap(tr("It deletes the app and %s of saves and settings:", size), w) {
			lines = append(lines, plain(env, l))
		}
		for _, lo := range mn.data {
			for _, l := range ui.Wrap(strings.Replace(lo.Path, os.Getenv("HOME"), "~", 1), w) {
				lines = append(lines, ui.T(p.Warn, l))
			}
		}
		return append(lines, gap(), ui.Of(button(env, true, tr("Yes, delete  ⏎")), space(1), button(env, false, tr("No  esc"))))
	}
	for i, e := range s.menuEntries(env) {
		lines = append(lines, ui.Line{L: []ui.Seg{ui.S(p.Text, " "+e.label)}, R: []ui.Seg{ui.S(p.Mu, e.key+" ")}, Sel: i == mn.sel}, ui.Line{L: []ui.Seg{ui.S(p.Mu, " "+e.hint)}, Sel: i == mn.sel}, gap())
	}
	return lines
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
	hint := tr("↑ or / to search ")
	if s.zone == zSearch {
		bar = ui.S(p2.Ac.Bold(true), "▌")
		hint = tr("⏎ search · ↓ results · esc ")
	}
	pad := ui.Line{L: ui.Spread([]ui.Seg{bar}, nil, w, p2.Fill)}
	left := append([]ui.Seg{bar, ui.S(p2.Ac, "  "+ui.G.Find+"  ")}, s.in.Segs(p2, s.zone == zSearch, w-34)...)
	mid := ui.Line{L: ui.Spread(left, []ui.Seg{ui.S(p2.Mu, hint)}, w, p2.Fill)}
	return []ui.Line{pad, mid, pad}
}

// tabsRow son las vistas y los filtros y, a la derecha, lo que hay marcado o el cargador.
func (s *Store) tabsRow(env *core.Env, w int) ui.Line {
	p := env.P
	var left []ui.Seg
	s.chipX = s.chipX[:0]
	x := 0
	chips := s.chips(env)
	for i, c := range chips {
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

		seg := ui.S(st, l+c.label+r)
		cw := len([]rune(l + c.label + r))
		s.chipX = append(s.chipX, [2]int{x, x + cw})
		x += cw + 1
		left = append(left, seg, ui.Seg{T: " "})
	}
	var right []ui.Seg
	for _, st := range []struct{ id, label string }{
		{"store.search", tr("Searching %s", s.query)},
		{"store.install", tr("Installing")},
		{"store.update", tr("Updating")},
		{"store.remove", tr("Removing")},
		{"data.apps", tr("Loading your apps")},
	} {
		if l, ok := working(env, st.id, st.label); ok {
			right = l.L
			break
		}
	}
	if n := len(s.queue); n > 0 {
		right = append(right, ui.S(p.Mu, fmt.Sprintf(tr("  +%d queued"), n)))
	}
	if right == nil {
		if n := s.markedCount(s.items(env)); n > 0 {
			right = []ui.Seg{ui.S(p.Ac, fmt.Sprintf(tr("%s %d selected "), ui.G.On, n))}
		}
	}
	return ui.Line{L: ui.Spread(left, right, w, p.Fill)}
}

// actionBar son los botones de acciones rápidas, justo debajo de las pestañas. Cambian con
// lo que hay marcado: con apps instaladas marcadas, actualizar (o buscar versiones nuevas)
// y quitar; sin marcar, buscar versiones nuevas y actualizar todas. Cada botón hace lo
// mismo que su tecla y se puede pulsar con el ratón.
func (s *Store) actionBar(env *core.Env, w int) ui.Line {
	p := env.P
	s.actX, s.actKey = s.actX[:0], s.actKey[:0]
	its := s.items(env)
	n := s.markedCount(its)
	type btn struct {
		label, key string
		primary    bool
	}
	var bs []btn
	var note string
	switch {
	case s.installedView() && n > 0:
		upd := 0
		for _, it := range its {
			if _, ok := s.updateFor(env, it); ok && s.marks[it.key()] && it.Installed {
				upd++
			}
		}
		if upd > 0 {
			bs = append(bs, btn{fmt.Sprintf(tr("Update %d"), upd), "u", true})
		} else {
			bs = append(bs, btn{tr("Check for updates"), "R", false})
		}
		bs = append(bs, btn{fmt.Sprintf(tr("Remove %d"), n), "r", upd == 0})
		note = tr("esc clears the selection")
	case s.installedView():
		bs = append(bs, btn{tr("Check for updates"), "R", len(env.Data.AppUpdates) == 0})
		if k := len(env.Data.AppUpdates); k > 0 {
			bs = append([]btn{{fmt.Sprintf(tr("Update all %d"), k), "U", true}}, bs...)
		}
		note = tr("space selects apps to remove or update")
	case n > 0:
		bs = append(bs, btn{fmt.Sprintf(tr("Install %d"), n), "enter", true})
		note = tr("esc clears the selection")
	default:
		note = tr("space selects several apps to install together")
	}
	var segs []ui.Seg
	x := 0
	for _, b := range bs {
		seg := button(env, b.primary, b.label)
		wd := len([]rune(seg.T))
		s.actX, s.actKey = append(s.actX, [2]int{x, x + wd}), append(s.actKey, b.key)
		segs = append(segs, seg, ui.Seg{T: " "})
		x += wd + 1
	}
	segs = append(segs, ui.S(p.Mu, " "+note))
	return ui.Line{L: segs}
}

func (s *Store) Main(env *core.Env, w, h int) []ui.Line {
	p := env.P
	s.hdr, s.itemH = storeHeader, storeItemH
	if h < compactBelow {
		s.hdr, s.itemH = 4, 2
	}
	s.rows = max((h-s.hdr)/s.itemH, 1)
	its := s.items(env)
	var lines []ui.Line
	if s.itemH == 2 {
		box := s.searchBox(env, w)
		lines = append(lines, box[1], s.tabsRow(env, w), s.actionBar(env, w), gap())
	} else {
		lines = s.searchBox(env, w)
		lines = append(lines, gap(), s.tabsRow(env, w), s.actionBar(env, w), gap())
	}

	// Cargando: la lista aparece ya, con la forma que tendrá
	loading := env.Tasks.Loading("store.search") && (s.showInst || !s.searched)
	if (!env.Data.AppsLoaded && !s.searched) || loading {
		if err := env.Data.Err["apps"]; err != nil && !env.Data.AppsLoaded && !loading {
			return append(lines, failed(env, tr("Could not load your apps"), err)...)
		}
		for i := 0; i < s.rows && i < 5; i++ {
			lines = append(lines, ui.Of(ui.S(p.Mu, "   "), ui.Skeleton(p, env.Frame+i*2, 14+i*3%9).L[0]), ui.Of(ui.S(p.Mu, "   "), ui.Skeleton(p, env.Frame+i*2+1, 30+i*5%12).L[0]), gap())
		}
		return lines
	}
	if len(its) == 0 {
		switch {
		case s.searched && !s.showInst:
			return append(lines, plain(env, tr("Nothing matches “%s”.", s.query)), muted(env, tr("Try another word, or fewer letters.")))
		default:
			return append(lines, plain(env, tr("Nothing installed with maxor yet.")), gap(), muted(env, tr("Press / and search: brave, btop, org.mozilla.firefox…")))
		}
	}
	inst := s.installedView()
	from, to := s.list.window(len(its), s.rows)
	for i := from; i < to; i++ {
		it := its[i]
		sel := i == s.list.sel && s.zone == zList
		marked := s.marks[it.key()] && it.Installed == inst
		// la casilla: la marca de la app (◼ marcada, ◻ sin marcar); lo ya instalado en los resultados lleva ✓
		var box ui.Seg
		switch {
		case marked:
			box = ui.S(p.Ac, ui.G.On+"  ")
		case it.Installed && !inst:
			box = ui.S(p.Ok, ui.G.Tick+"  ")
		default:
			box = ui.S(p.Mu, ui.G.Off+"  ")
		}
		src := p.Ac
		if it.Source == "nix" {
			src = p.Ac2
		}
		ver := shortVersion(it.Version)
		// a la derecha, en una sola columna, el estado: lo que está pasando o si está al día
		var status ui.Seg
		u, hasUpd := s.updateFor(env, it)
		switch st := s.busy[it.key()]; {
		case st != "":
			box = ui.S(p.Warn, ui.G.Off+"  ")
			status = ui.S(p.Warn, map[string]string{"queued": "queued", "installing": tr("installing…"), "removing": tr("removing…"), "updating": tr("updating…")}[st])
		case inst && hasUpd:
			status = ui.S(p.Warn.Bold(true), ui.G.Up+" "+u.Latest+" available")
		case inst && env.Data.UpdatesKnown:
			status = ui.S(p.Ok, ui.G.Tick+" "+tr("up to date"))
		case inst && env.Data.Err["appupdates"] != nil:
			status = ui.S(p.Mu, tr("could not check"))
		case inst:
			status = ui.S(p.Mu, tr("checking…"))
		case it.Installed:
			status = ui.S(p.Ok, "installed")
		default:
			status = ui.S(p.Mu, ver)
		}
		var second []ui.Seg
		if inst {
			// lo instalado se explica solo: de dónde viene y qué versión tiene
			txt := srcName(it.Source)
			if ver != "" {
				txt += " · " + ver
			}
			if hasUpd && u.Current != "" {
				txt = srcName(it.Source) + " · " + u.Current + " " + ui.G.Arrow + " " + u.Latest
			}
			second = []ui.Seg{ui.S(p.Mu, "     "), ui.S(src, txt)}
		} else {
			desc := it.Desc
			if desc == "" {
				desc = it.ID
			}
			second = []ui.Seg{ui.S(p.Mu, "     "+desc)}
		}
		right := []ui.Seg{status, ui.Seg{T: " "}}
		if !inst {
			right = []ui.Seg{ui.S(src, srcName(it.Source)), ui.S(p.Mu, "  "), status, ui.Seg{T: " "}}
		}
		nameSt := p.Bold
		if marked {
			nameSt = p.Ac.Bold(true)
		}
		lines = append(lines,
			ui.Line{L: []ui.Seg{box, ui.S(nameSt, it.Name)}, R: right, Sel: sel},
			ui.Line{L: second, Sel: sel})
		if s.itemH > 2 {
			lines = append(lines, gap())
		}
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
	if s.ask != nil {
		var total int64
		for _, e := range s.ask.Entries {
			total += sumBytes(e.Left)
		}
		lines := []ui.Line{heading(env, tr("Their data is still here")), gap()}
		for _, l := range ui.Wrap(tr("These folders were left in your home. They may hold saves and settings (%s).", humanBytes(total)), w) {
			lines = append(lines, plain(env, l))
		}
		for _, e := range s.ask.Entries {
			lines = append(lines, gap(), ui.T(p.Bold, e.Name))
			for _, lo := range e.Left {
				for _, l := range ui.Wrap(strings.Replace(lo.Path, os.Getenv("HOME"), "~", 1)+"  "+humanBytes(lo.Bytes), w) {
					lines = append(lines, ui.T(p.Warn, l))
				}
			}
		}
		return append(lines, gap(), ui.Of(button(env, true, tr("Delete data  y")), space(1), button(env, false, tr("Keep  n"))))
	}
	if s.menu != nil {
		return s.menuLines(env, w)
	}
	its := s.items(env)
	inst := s.installedView()
	// con apps marcadas, el panel es el de las acciones en bloque
	if n := s.markedCount(its); n > 0 {
		lines := []ui.Line{heading(env, fmt.Sprintf(tr("%d selected"), n)), gap()}
		shown := 0
		upd := 0
		for _, it := range its {
			if !(s.marks[it.key()] && it.Installed == inst) {
				continue
			}
			if _, ok := s.updateFor(env, it); ok {
				upd++
			}
			if shown < 6 {
				lines = append(lines, plain(env, it.Name))
				shown++
			}
		}
		if n > shown {
			lines = append(lines, muted(env, fmt.Sprintf(tr("and %d more"), n-shown)))
		}
		lines = append(lines, gap())
		if inst && upd > 0 {
			lines = append(lines, ui.T(p.Warn, fmt.Sprintf(tr("%s %d of them can be updated"), ui.G.Up, upd)), gap())
		}
		return append(lines, muted(env, tr("Use the buttons above,")), muted(env, tr("or press ⏎ for more options.")))
	}
	if len(its) == 0 || s.list.sel >= len(its) {
		return []ui.Line{heading(env, tr("Details")), gap(), muted(env, tr("Search with /")), gap(), muted(env, tr("Mark several with space")), muted(env, tr("and act on them together."))}
	}
	it := its[s.list.sel]
	lines := []ui.Line{heading(env, tr("Details")), gap(), ui.T(p.Bold, it.Name), muted(env, it.ID), gap()}
	if it.Desc != "" {
		for _, l := range ui.Wrap(it.Desc, w) {
			lines = append(lines, plain(env, l))
		}
		lines = append(lines, gap())
	}
	lines = append(lines, ui.Of(ui.S(p.Mu, tr("source   ")), ui.S(p.Text, srcName(it.Source))))
	if v := shortVersion(it.Version); v != "" {
		lines = append(lines, ui.Of(ui.S(p.Mu, tr("version  ")), ui.S(p.Text, v)))
	}
	lines = append(lines, gap())
	switch {
	case s.busy[it.key()] != "":
		lines = append(lines, ui.Of(ui.S(p.Warn, ui.G.Warn+" "+s.busy[it.key()]+"… please wait")))
	case it.Installed:
		if u, ok := s.updateFor(env, it); ok {
			lines = append(lines, ui.T(p.Warn.Bold(true), ui.G.Up+" "+tr("new version %s", u.Latest)), gap(), ui.Of(button(env, true, tr("Update  u")), space(1), button(env, false, tr("More…  ⏎"))))
		} else if env.Data.UpdatesKnown {
			lines = append(lines, ui.T(p.Ok, ui.G.Tick+" "+tr("up to date")), gap(), ui.Of(button(env, true, tr("Actions  ⏎")), space(1), button(env, false, tr("Check again  R"))))
		} else {
			lines = append(lines, ui.Of(button(env, true, tr("Actions  ⏎"))))
		}
		lines = append(lines, gap(), muted(env, tr("Open it, update it or remove it.")))
	default:
		lines = append(lines, ui.Of(button(env, true, tr("Install  ⏎")), space(1), button(env, false, tr("Select  space"))))
	}
	if n := len(env.Data.AppUpdates); inst && n > 0 {
		lines = append(lines, gap(), ui.T(p.Warn, fmt.Sprintf(tr("%s %s can be updated"), ui.G.Up, plural(n, "app", "apps"))), ui.Of(button(env, false, tr("Update all  U"))))
	}
	return lines
}

func (s *Store) Hints(env *core.Env) []ui.Hint {
	if s.ask != nil && s.zone != zSearch {
		return []ui.Hint{{Key: "y", Action: tr("delete their data")}, {Key: "n", Action: tr("keep it")}}
	}
	switch s.zone {
	case zSearch:
		return []ui.Hint{{Key: "⏎", Action: tr("search")}, {Key: "↓", Action: tr("results")}, {Key: "esc", Action: tr("leave")}}
	case zChips:
		return []ui.Hint{{Key: "←→", Action: tr("switch")}, {Key: "↑", Action: tr("search")}, {Key: "↓", Action: tr("list")}}
	}
	if s.menu != nil {
		if s.menu.confirm {
			return []ui.Hint{{Key: "⏎", Action: tr("yes, delete")}, {Key: "esc", Action: tr("no")}}
		}
		return []ui.Hint{{Key: "↑↓", Action: tr("choose")}, {Key: "⏎", Action: tr("do it")}, {Key: "esc", Action: tr("close")}}
	}
	if s.installedView() {
		return []ui.Hint{{Key: "space", Action: tr("select")}, {Key: "a", Action: tr("all")}, {Key: "u", Action: tr("update")}, {Key: "r", Action: tr("remove")}, {Key: "⏎", Action: tr("more")}}
	}
	return []ui.Hint{{Key: "↑", Action: tr("search")}, {Key: "space", Action: tr("select")}, {Key: "a", Action: tr("all")}, {Key: "⏎", Action: tr("install")}}
}

func (s *Store) Click(env *core.Env, x, y int) tea.Cmd {
	switch {
	case y < s.hdr-4+1 && s.hdr == 4 || y < 3 && s.hdr != 4:
		s.zone = zSearch
	case y == s.hdr-2: // la barra de acciones rápidas
		for i, r := range s.actX {
			if x >= r[0] && x < r[1] {
				k := s.actKey[i]
				var msg tea.KeyMsg
				switch k {
				case "enter":
					msg = tea.KeyMsg{Type: tea.KeyEnter}
				default:
					msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
				}
				s.zone = zList
				_, cmd := s.key(env, msg)
				return cmd
			}
		}
	case y == s.hdr-3: // las pestañas
		s.zone = zChips
		for i, r := range s.chipX {
			if x >= r[0] && x < r[1]+1 {
				s.pickChip(env, i)
			}
		}
	case y >= s.hdr:
		s.zone = zList
		s.menu = nil
		its := s.items(env)
		i := s.list.top + (y-s.hdr)/max(s.itemH, 1)
		if i >= 0 && i < len(its) {
			s.list.sel = i
			// un clic en la casilla marca la app
			if x < 5 && its[i].Installed == s.installedView() {
				k := its[i].key()
				if s.marks[k] {
					delete(s.marks, k)
				} else {
					s.marks[k] = true
				}
			}
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

func sumBytes(l []maxor.Leftover) (n int64) {
	for _, x := range l {
		n += x.Bytes
	}
	return
}

// humanBytes escribe un tamaño con unidades legibles.
func humanBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf(tr("%.1f GiB"), float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf(tr("%.1f MiB"), float64(b)/(1<<20))
	}
	return fmt.Sprintf(tr("%d KiB"), (b+1023)/1024)
}

// Brief es el cajón de detalles de una ventana estrecha: lo esencial en pocas filas.
func (s *Store) Brief(env *core.Env, w int) []ui.Line {
	p := env.P
	if s.ask != nil {
		var total int64
		for _, e := range s.ask.Entries {
			total += sumBytes(e.Left)
		}
		return []ui.Line{
			ui.Of(ui.S(p.Warn, ui.G.Warn+" "), ui.S(p.Text, tr("Their data is still on disk (%s)", humanBytes(total)))),
			ui.Of(button(env, true, tr("Delete data  y")), space(1), button(env, false, tr("Keep  n"))),
		}
	}
	if s.menu != nil {
		mn := s.menu
		title := mn.items[0].Name
		if len(mn.items) > 1 {
			title = fmt.Sprintf(tr("%d apps"), len(mn.items))
		}
		if mn.confirm {
			size := ""
			if len(mn.data) > 0 {
				size = " (" + humanBytes(sumBytes(mn.data)) + ")"
			}
			return []ui.Line{
				ui.Of(ui.S(p.Warn.Bold(true), ui.G.Warn+" "+tr("This cannot be undone")+": "), ui.S(p.Text, tr("deletes %s and its data%s", title, size))),
				ui.Of(button(env, true, tr("Yes, delete  ⏎")), space(1), button(env, false, tr("No  esc"))),
			}
		}
		lines := []ui.Line{ui.T(p.Bold, title)}
		for i, e := range s.menuEntries(env) {
			lines = append(lines, ui.Line{L: []ui.Seg{ui.S(p.Text, " "+e.label), ui.S(p.Mu, "  "+e.hint)}, R: []ui.Seg{ui.S(p.Mu, e.key+" ")}, Sel: i == mn.sel})
		}
		return lines
	}
	its := s.items(env)
	inst := s.installedView()
	if n := s.markedCount(its); n > 0 {
		return []ui.Line{ui.Of(ui.S(p.Ac, ui.G.On+" "), ui.S(p.Text, fmt.Sprintf(tr("%d selected"), n)), ui.S(p.Mu, tr("  ·  use the buttons above, or ⏎ for more")))}
	}
	if len(its) == 0 || s.list.sel >= len(its) {
		return []ui.Line{muted(env, tr("Search with /  ·  space selects several apps"))}
	}
	it := its[s.list.sel]
	head := []ui.Seg{ui.S(p.Bold, it.Name), ui.S(p.Mu, "  "+srcName(it.Source))}
	if v := shortVersion(it.Version); v != "" {
		head = append(head, ui.S(p.Mu, " · "+v))
	}
	lines := []ui.Line{ui.Of(head...)}
	if it.Desc != "" && !inst {
		lines = append(lines, muted(env, it.Desc))
	}
	switch {
	case s.busy[it.key()] != "":
		lines = append(lines, ui.T(p.Warn, ui.G.Warn+" "+s.busy[it.key()]+"…"))
	case it.Installed:
		if u, ok := s.updateFor(env, it); ok {
			lines = append(lines, ui.Of(ui.S(p.Warn.Bold(true), ui.G.Up+" "+tr("%s available", u.Latest)+"  "), button(env, true, tr("Update  u")), space(1), button(env, false, tr("More  ⏎"))))
		} else {
			lines = append(lines, ui.Of(ui.S(p.Ok, ui.G.Tick+" "+tr("up to date")+"  "), button(env, true, tr("Actions  ⏎"))))
		}
	default:
		lines = append(lines, ui.Of(button(env, true, tr("Install  ⏎")), space(1), button(env, false, tr("Select  space"))))
	}
	return lines
}
