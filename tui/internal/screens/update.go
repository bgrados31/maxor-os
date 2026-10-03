package screens

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/maxor"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// staleAfter es la edad a partir de la cual un escaneo guardado se repite solo.
const staleAfter = 6 * time.Hour

// Update enseña en qué rama está la configuración, qué canal de nixpkgs usa y qué
// cambiaría el sistema. El escaneo (`maxor update --json`: compila y compara, sin
// aplicar nada) ya debería estar hecho al abrir la pestaña: se guarda en disco y
// se repite solo si cambió el repositorio o tiene más de 6 horas. Se puede repetir
// a mano con «r». Aplicar cede la terminal a `maxor update` (progreso real y sudo).
type Update struct {
	core.Base
	list     listState
	rows     int
	autoDone bool
	lock     bool // si el último escaneo refrescó las entradas del flake
}

func NewUpdate() *Update { return &Update{} }

func (u *Update) ID() string    { return "update" }
func (u *Update) Title() string { return "Update" }

func (u *Update) Init(env *core.Env) tea.Cmd {
	var cmds []tea.Cmd
	if env.Data.UpdateStatus == nil {
		cmds = append(cmds, LoadUpdateStatus(env, true))
	}
	if !env.Data.CacheLoaded {
		cmds = append(cmds, LoadUpdateCache(env, true))
	}
	return tea.Batch(append(cmds, u.decide(env))...)
}

// stale dice si el escaneo guardado ya no vale: otra huella del repositorio o muy viejo.
func (u *Update) stale(env *core.Env) bool {
	up, st := env.Data.Update, env.Data.UpdateStatus
	if up == nil {
		return true
	}
	if st != nil && up.Fingerprint != "" && up.Fingerprint != st.Fingerprint {
		return true
	}
	return env.Now().Sub(time.Unix(up.CheckedAt, 0)) > staleAfter
}

// decide lanza el escaneo automático, una sola vez, cuando ya se sabe si hace falta.
func (u *Update) decide(env *core.Env) tea.Cmd {
	if u.autoDone || env.Data.UpdateStatus == nil || !env.Data.CacheLoaded {
		return nil
	}
	u.autoDone = true
	if !u.stale(env) {
		return nil
	}
	return u.scan(env, false)
}

func (u *Update) scan(env *core.Env, lock bool) tea.Cmd {
	label := "Scanning for pending changes"
	if lock {
		label = "Checking for new versions"
	}
	u.lock = lock
	u.list = listState{}
	return env.Tasks.Start(task.Task{ID: "update.check", Label: label, Run: func(ctx context.Context) (any, error) {
		return env.Client.UpdateCheck(ctx, lock)
	}})
}

// applyCmd cede la terminal a `maxor update --no-lock -y` y espera un Intro antes
// de volver, para que dé tiempo a leer el resultado.
func applyCmd() *exec.Cmd {
	return exec.Command("sh", "-c",
		`"$@"; rc=$?; echo; printf 'Press Enter to return to Maxor… '; read _; exit $rc`,
		"sh", maxor.Bin(), "update", "--no-lock", "-y")
}

func (u *Update) Update(env *core.Env, msg tea.Msg) (core.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case CheckUpdateMsg:
		return u, u.scan(env, true)
	case task.DoneMsg:
		switch m.ID {
		case "data.updatestatus", "data.updatecache":
			return u, u.decide(env)
		case "update.check":
			if m.Err != nil {
				return u, core.Toast("bad", "Could not scan: "+oneLine(m.Err.Error()))
			}
			v, _ := m.Value.(maxor.UpdateCheck)
			env.Data.Update = &v
			env.Data.CacheLoaded = true
			return u, LoadUpdateStatus(env, true)
		}
	case core.ExecDoneMsg:
		if m.Tag != "update" {
			return u, nil
		}
		env.Data.Update = nil
		if m.Err != nil {
			return u, core.Toast("bad", "The update did not finish: maxor logs --last")
		}
		return u, tea.Batch(core.Toast("ok", "System updated"), core.Note("ok", "Updated the system"), LoadUpdateStatus(env, true), u.scan(env, false))
	case tea.KeyMsg:
		if d, mv := listKey(m); mv && env.Data.Update != nil {
			u.list.move(d, len(env.Data.Update.Changes), max(u.rows, 1))
			return u, nil
		}
		switch {
		case isKey(m, "r"):
			return u, u.scan(env, u.lock)
		case isKey(m, "c"):
			return u, u.scan(env, true)
		case isKey(m, "a", "enter"):
			up := env.Data.Update
			if up == nil {
				return u, core.Toast("info", "Nothing scanned yet: press r")
			}
			if up.UpToDate {
				return u, core.Toast("ok", "The system is already up to date")
			}
			return u, tea.ExecProcess(applyCmd(), func(err error) tea.Msg { return core.ExecDoneMsg{Tag: "update", Err: err} })
		}
	}
	return u, nil
}

// configLines es el bloque «Configuration»: rama, canal de nixpkgs y generación.
func (u *Update) configLines(env *core.Env) []ui.Line {
	p := env.P
	st := env.Data.UpdateStatus
	lines := []ui.Line{heading(env, "Configuration")}
	if st == nil {
		return append(lines, ui.Skeleton(p, env.Frame, 22), ui.Skeleton(p, env.Frame+2, 34), ui.Skeleton(p, env.Frame+4, 18))
	}
	switch {
	case st.Branch == "":
		lines = append(lines, muted(env, "not a git repository"))
	case st.Dirty:
		lines = append(lines, ui.Of(ui.S(p.Ac, ui.G.Branch+"  "), ui.S(p.Bold, st.Branch), ui.S(p.Mu, " · "+st.Commit+" · "), ui.S(p.Warn, plural(st.Files, "uncommitted file", "uncommitted files"))))
	default:
		lines = append(lines, ui.Of(ui.S(p.Ac, ui.G.Branch+"  "), ui.S(p.Bold, st.Branch), ui.S(p.Mu, " · "+st.Commit+" · "), ui.S(p.Ok, "clean")))
	}
	if st.Channel != "" || st.NixpkgsRev != "" {
		lines = append(lines, muted(env, fmt.Sprintf("nixpkgs %s · %s · locked %s", st.Channel, st.NixpkgsRev, ago(env.Now(), st.NixpkgsDate))))
	}
	if st.Generation > 0 {
		lines = append(lines, muted(env, fmt.Sprintf("generation %d is running", st.Generation)))
	}
	return lines
}

func (u *Update) Main(env *core.Env, w, h int) []ui.Line {
	p := env.P
	lines := u.configLines(env)
	lines = append(lines, gap(), heading(env, "Scan"))
	u.rows = h - len(lines) - 6

	if l, ok := working(env, "update.check", "Scanning"); ok {
		return append(lines, l, muted(env, "Building the new system and comparing it with the running one."),
			muted(env, "The first time can take several minutes. Nothing is applied."))
	}
	up := env.Data.Update
	if up == nil {
		if env.Data.CacheLoaded && env.Data.UpdateStatus != nil {
			return append(lines, plain(env, "No scan yet."), muted(env, "Press r to scan."))
		}
		return append(lines, ui.Skeleton(p, env.Frame, 26), ui.Skeleton(p, env.Frame+3, 18))
	}
	// Cuándo se hizo y si sigue valiendo
	when := ago(env.Now(), up.CheckedAt)
	mode := "pending changes"
	if up.Lock {
		mode = "with fresh inputs"
	}
	info := []ui.Seg{ui.S(p.Mu, "scanned "+when+" · "+mode)}
	if u.stale(env) {
		info = append(info, ui.S(p.Warn, "  "+ui.G.Warn+" out of date: press r"))
	}
	lines = append(lines, ui.Of(info...))
	if up.UpToDate {
		return append(lines, gap(), ui.T(p.Ok.Bold(true), ui.G.Tick+"  The system is up to date"), muted(env, "Nothing to apply."))
	}
	c := up.Counts
	lines = append(lines, gap(), ui.Of(
		ui.S(p.Ok, fmt.Sprintf("%s%d new   ", ui.G.Add, c.New)),
		ui.S(p.Warn, fmt.Sprintf("%s%d updated   ", ui.G.Up, c.Updated)),
		ui.S(p.Bad, fmt.Sprintf("%s%d removed   ", ui.G.Del, c.Removed)),
		ui.S(p.Mu, fmt.Sprintf("%s%d changed", ui.G.Chg, c.Changed)),
	))
	if up.Kernel {
		lines = append(lines, ui.T(p.Warn, ui.G.Warn+"  includes a new kernel: reboot after applying"))
	}
	if c.Config > 0 {
		lines = append(lines, muted(env, fmt.Sprintf("%d configuration files changed (hidden)", c.Config)))
	}
	lines = append(lines, gap())
	from, to := u.list.window(len(up.Changes), max(u.rows, 1))
	for i := from; i < to; i++ {
		ch := up.Changes[i]
		sym, st := ui.G.Chg, p.Mu
		ver := ch.Size
		switch ch.Kind {
		case "updated":
			sym, st, ver = ui.G.Up, p.Warn, ch.From+" "+ui.G.Arrow+" "+ch.To
		case "new":
			sym, st, ver = ui.G.Add, p.Ok, ch.To
		case "removed":
			sym, st, ver = ui.G.Del, p.Bad, ch.From
		}
		lines = append(lines, ui.Line{
			L:   []ui.Seg{ui.S(st, sym+" "), ui.S(p.Text, ch.Name)},
			R:   []ui.Seg{ui.S(p.Mu, ver+" ")},
			Sel: i == u.list.sel,
		})
	}
	return lines
}

func (u *Update) Side(env *core.Env, w, h int) []ui.Line {
	p := env.P
	lines := []ui.Line{heading(env, "Actions"), gap()}
	up := env.Data.Update
	canApply := up != nil && !up.UpToDate
	if canApply {
		lines = append(lines, ui.Of(button(env, true, "Apply  ⏎")), gap())
	}
	lines = append(lines,
		ui.Of(button(env, !canApply, "Rescan  r")), muted(env, "compile again and compare"), gap(),
		ui.Of(button(env, false, "New versions  c")), muted(env, "refresh nixpkgs, then scan"), gap())
	if st := env.Data.UpdateStatus; st != nil && st.Dirty && canApply {
		for _, l := range ui.Wrap("Uncommitted changes are included in the build.", w) {
			lines = append(lines, ui.T(p.Warn, l))
		}
		lines = append(lines, gap())
	}
	if canApply {
		for _, l := range ui.Wrap("Applying hands you the terminal: sudo asks for your password and you come back after.", w) {
			lines = append(lines, ui.T(p.Mu, l))
		}
		lines = append(lines, gap())
	}
	for _, l := range ui.Wrap("New versions refreshes flake.lock. Undo it with git checkout flake.lock.", w) {
		lines = append(lines, ui.T(p.Mu, l))
	}
	return lines
}

func (u *Update) Hints(env *core.Env) []ui.Hint {
	h := []ui.Hint{{Key: "r", Action: "rescan"}, {Key: "c", Action: "new versions"}}
	if env.Data.Update != nil && !env.Data.Update.UpToDate {
		h = append(h, ui.Hint{Key: "⏎", Action: "apply"}, ui.Hint{Key: "↑↓", Action: "scroll"})
	}
	return h
}

func (u *Update) Wheel(env *core.Env, dy int) tea.Cmd {
	if env.Data.Update != nil {
		u.list.move(dy*2, len(env.Data.Update.Changes), max(u.rows, 1))
	}
	return nil
}
