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

// Home es el inicio: el estado del equipo de un vistazo y las acciones rápidas.
type Home struct {
	core.Base
	list listState
	half int // ancho de una tarjeta, para el clic
}

func NewHome() *Home { return &Home{} }

func (h *Home) ID() string    { return "home" }
func (h *Home) Title() string { return "Home" }

type action struct {
	key, label, screen string
	then               string // acción extra al llegar: «check» o «search»
}

var homeActions = []action{
	{"u", "Check for updates", "update", "check"},
	{"s", "Search apps", "store", "search"},
	{"t", "Change theme", "themes", ""},
	{"d", "Run doctor", "doctor", ""},
	{"b", "Back up my setup", "", "backup"},
}

// CheckUpdateMsg y FocusSearchMsg los entiende la pantalla de destino.
type CheckUpdateMsg struct{}
type FocusSearchMsg struct{}

// BackupMsg pide guardar una copia de seguridad (la paleta de comandos).
type BackupMsg struct{}

func (h *Home) Init(env *core.Env) tea.Cmd {
	var cmds []tea.Cmd
	if env.Data.Doctor == nil {
		cmds = append(cmds, LoadDoctor(env, true))
	}
	if !env.Data.AppsLoaded {
		cmds = append(cmds, LoadApps(env, true))
	}
	if !env.Data.ThemesLoaded {
		cmds = append(cmds, LoadThemes(env, true))
	}
	if env.Data.Hardware == nil {
		cmds = append(cmds, LoadHardware(env, true))
	}
	if env.Data.UpdateStatus == nil {
		cmds = append(cmds, LoadUpdateStatus(env, true))
	}
	if !env.Data.CacheLoaded {
		cmds = append(cmds, LoadUpdateCache(env, true))
	}
	if !env.Data.UpdatesKnown {
		cmds = append(cmds, LoadAppUpdates(env, true))
	}
	cmds = append(cmds, ReleaseInit(env)...)
	return tea.Batch(cmds...)
}

// backup guarda tu configuración en un archivo con `maxor backup` y lo cuenta al terminar.
func (h *Home) backup(env *core.Env) tea.Cmd {
	return env.Tasks.Start(task.Task{ID: "home.backup", Label: "Saving your setup", Run: func(ctx context.Context) (any, error) {
		return env.Client.Backup(ctx)
	}})
}

func (h *Home) run(env *core.Env, i int) tea.Cmd {
	a := homeActions[i]
	switch a.then {
	case "backup":
		return h.backup(env)
	case "check":
		return core.GoThen(a.screen, CheckUpdateMsg{})
	case "search":
		return core.GoThen(a.screen, FocusSearchMsg{})
	}
	return core.Go(a.screen)
}

func (h *Home) Update(env *core.Env, msg tea.Msg) (core.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case BackupMsg:
		return h, h.backup(env)
	case task.DoneMsg:
		if m.ID == "home.backup" {
			if m.Err != nil {
				return h, core.Toast("bad", "Could not save the backup: "+oneLine(m.Err.Error()))
			}
			b, _ := m.Value.(maxor.BackupInfo)
			path := strings.Replace(b.Path, os.Getenv("HOME"), "~", 1)
			return h, tea.Batch(core.Toast("ok", fmt.Sprintf("Saved %s (%s, %s)", path, humanBytes(b.Bytes), plural(b.Apps, "app", "apps"))), core.Note("ok", "Saved a backup: "+path))
		}
	}
	if k, ok := msg.(tea.KeyMsg); ok {
		if d, mv := listKey(k); mv {
			h.list.move(d, len(homeActions), len(homeActions))
			return h, nil
		}
		if isKey(k, "v") && releaseAvailable(env) {
			return h, core.Go("update")
		}
		if isKey(k, "enter") {
			return h, h.run(env, h.list.sel)
		}
		for i, a := range homeActions {
			if isKey(k, a.key) {
				return h, h.run(env, i)
			}
		}
	}
	return h, nil
}

func greeting(env *core.Env) string {
	switch hr := env.Now().Hour(); {
	case hr < 6:
		return "Good night"
	case hr < 12:
		return "Good morning"
	case hr < 19:
		return "Good afternoon"
	}
	return "Good evening"
}

// tile es una tarjeta de cuatro filas: título, dato, detalle y un hueco.
func (h *Home) tile(env *core.Env, title string, loading bool, big []ui.Seg, sub string) [][]ui.Seg {
	p := env.P
	rows := [][]ui.Seg{{ui.S(p.Mu, title)}}
	if loading {
		// la tarjeta ya tiene su forma: título real y dos barras que brillan, cada
		// tarjeta con su fase para que el brillo no vaya a la vez en todas
		phase := env.Frame + len(title)*4
		return append(rows, ui.Skeleton(p, phase, 13).L, ui.Skeleton(p, phase+3, 22).L)
	}
	return append(rows, big, []ui.Seg{ui.S(p.Mu, sub)})
}

func (h *Home) Main(env *core.Env, w, hh int) []ui.Line {
	p := env.P
	d := env.Data
	lines := []ui.Line{ui.T(p.Bold, greeting(env)+" · "+env.Host), gap()}
	if b := releaseBanner(env); len(b) > 0 {
		lines = append(append(lines, b...), gap())
	}

	// Sistema
	var sys [][]ui.Seg
	switch {
	case d.Doctor != nil:
		doc := d.Doctor
		switch {
		case doc.Fails > 0:
			sys = h.tile(env, "SYSTEM", false, []ui.Seg{ui.S(p.Bad.Bold(true), ui.G.Bad+" "+plural(doc.Fails, "problem", "problems"))}, "maxor doctor")
		case doc.Warns > 0:
			sys = h.tile(env, "SYSTEM", false, []ui.Seg{ui.S(p.Warn.Bold(true), ui.G.Warn+" "+plural(doc.Warns, "warning", "warnings"))}, "press d for details")
		default:
			sys = h.tile(env, "SYSTEM", false, []ui.Seg{ui.S(p.Ok.Bold(true), ui.G.Tick+" Healthy")}, "all checks pass")
		}
	case d.Err["doctor"] != nil:
		sys = h.tile(env, "SYSTEM", false, []ui.Seg{ui.S(p.Bad, "could not check")}, "maxor logs --last")
	default:
		sys = h.tile(env, "SYSTEM", true, nil, "")
	}

	// Actualizaciones: lo que dejó el último escaneo, que la pestaña Update repite sola
	var upd [][]ui.Seg
	switch {
	case d.Update == nil && !d.CacheLoaded:
		upd = h.tile(env, "UPDATES", true, nil, "")
	case d.Update == nil:
		upd = h.tile(env, "UPDATES", false, []ui.Seg{ui.S(p.Mu, "Not scanned yet")}, "open Update to scan")
	case d.Update.UpToDate:
		upd = h.tile(env, "UPDATES", false, []ui.Seg{ui.S(p.Ok.Bold(true), ui.G.Tick+" Up to date")}, "scanned "+ago(env.Now(), d.Update.CheckedAt))
	default:
		n := d.Update.Counts.New + d.Update.Counts.Updated + d.Update.Counts.Removed + d.Update.Counts.Changed
		sub := "scanned " + ago(env.Now(), d.Update.CheckedAt)
		if d.Update.Kernel {
			sub = "new kernel · " + sub
		}
		upd = h.tile(env, "UPDATES", false, []ui.Seg{ui.S(p.Warn.Bold(true), plural(n, "change", "changes"))}, sub)
	}

	// Apps
	var apps [][]ui.Seg
	if d.AppsLoaded {
		nix, fp := 0, 0
		for _, a := range d.Apps {
			if a.Source == "flatpak" {
				fp++
			} else {
				nix++
			}
		}
		main := []ui.Seg{ui.S(p.Ac.Bold(true), fmt.Sprintf("%d installed", len(d.Apps)))}
		sub := fmt.Sprintf("nix %d · flathub %d", nix, fp)
		if n := len(d.AppUpdates); n > 0 {
			main = append(main, ui.S(p.Warn.Bold(true), fmt.Sprintf("  %s%d", ui.G.Up, n)))
			sub = plural(n, "update", "updates") + " available · " + sub
		}
		apps = h.tile(env, "APPS", false, main, sub)
	} else {
		apps = h.tile(env, "APPS", d.Err["apps"] == nil, []ui.Seg{ui.S(p.Bad, "could not load")}, "")
	}

	// Tema
	var th [][]ui.Seg
	if t := ActiveTheme(env); t != nil {
		th = h.tile(env, "THEME", false, []ui.Seg{ui.S(p.Ac2.Bold(true), t.Name)}, t.Mode+" · press t to change")
	} else {
		th = h.tile(env, "THEME", !d.ThemesLoaded, []ui.Seg{ui.S(p.Ac2.Bold(true), env.Theme.Name)}, "press t to change")
	}

	half := (w - 2) / 2
	h.half = half
	row := func(a, b [][]ui.Seg) {
		for i := 0; i < 3; i++ {
			l := ui.Line{L: append(append(ui.Cell(a[i], half, p.Fill), ui.S(p.Fill, "  ")), ui.Cell(b[i], w-half-2, p.Fill)...)}
			lines = append(lines, l)
		}
		lines = append(lines, gap())
	}
	row(sys, upd)
	row(apps, th)

	if d.Hardware != nil {
		hw := d.Hardware
		gpus := make([]string, 0, len(hw.GPUs))
		for _, g := range hw.GPUs {
			gpus = append(gpus, g.Vendor)
		}
		kind := "desktop"
		if hw.Laptop {
			kind = "laptop"
		}
		lines = append(lines, heading(env, "This machine"),
			plain(env, hw.CPU.Model),
			muted(env, kind+" · "+strings.Join(gpus, " + ")))
	}
	return lines
}

func (h *Home) Side(env *core.Env, w, hh int) []ui.Line {
	p := env.P
	lines := []ui.Line{heading(env, "Quick actions"), gap()}
	for i, a := range homeActions {
		lines = append(lines, ui.Line{
			L:   []ui.Seg{ui.S(p.Ac, a.key), ui.S(p.Text, "  "+a.label)},
			Sel: i == h.list.sel,
		})
	}
	return lines
}

func (h *Home) Hints(env *core.Env) []ui.Hint {
	return []ui.Hint{{Key: "↑↓", Action: "move"}, {Key: "⏎", Action: "go"}, {Key: "u s t d", Action: "shortcuts"}}
}


// Click lleva a la pestaña de la tarjeta pulsada.
func (h *Home) Click(env *core.Env, x, y int) tea.Cmd {
	right := x > h.half
	switch {
	case y >= 2 && y <= 4 && !right:
		return core.Go("doctor")
	case y >= 2 && y <= 4 && right:
		return core.Go("update")
	case y >= 6 && y <= 8 && !right:
		return core.Go("store")
	case y >= 6 && y <= 8 && right:
		return core.Go("themes")
	}
	return nil
}
