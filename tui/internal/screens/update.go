package screens

import (
	"context"
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/maxor"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Update comprueba qué cambiaría el sistema y, si lo confirmas, lo aplica. La
// comprobación es `maxor update --json` (compila y compara, sin aplicar nada);
// aplicar cede la terminal a `maxor update` para que se vea el progreso real y
// funcione la contraseña de sudo.
type Update struct {
	core.Base
	list listState
	rows int
}

func NewUpdate() *Update { return &Update{} }

func (u *Update) ID() string    { return "update" }
func (u *Update) Title() string { return "Update" }

func (u *Update) Init(env *core.Env) tea.Cmd { return nil }

func (u *Update) check(env *core.Env, lock bool) tea.Cmd {
	label := "Checking pending changes"
	if lock {
		label = "Checking for updates"
	}
	env.Data.Update = nil
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
		return u, u.check(env, true)
	case task.DoneMsg:
		if m.ID != "update.check" {
			return u, nil
		}
		if m.Err != nil {
			return u, core.Toast("bad", "Could not check: "+oneLine(m.Err.Error()))
		}
		v, _ := m.Value.(maxor.UpdateCheck)
		env.Data.Update = &v
		return u, nil
	case core.ExecDoneMsg:
		if m.Tag != "update" {
			return u, nil
		}
		env.Data.Update = nil
		if m.Err != nil {
			return u, core.Toast("bad", "The update did not finish: maxor logs --last")
		}
		return u, tea.Batch(core.Toast("ok", "System updated"), core.Note("ok", "Updated the system"))
	case tea.KeyMsg:
		if d, mv := listKey(m); mv && env.Data.Update != nil {
			u.list.move(d, len(env.Data.Update.Changes), max(u.rows, 1))
			return u, nil
		}
		switch {
		case isKey(m, "c"):
			return u, u.check(env, true)
		case isKey(m, "b"):
			return u, u.check(env, false)
		case isKey(m, "a", "enter"):
			up := env.Data.Update
			if up == nil {
				return u, core.Toast("info", "Check first: press c")
			}
			if up.UpToDate {
				return u, core.Toast("ok", "The system is already up to date")
			}
			return u, tea.ExecProcess(applyCmd(), func(err error) tea.Msg { return core.ExecDoneMsg{Tag: "update", Err: err} })
		}
	}
	return u, nil
}

func (u *Update) Main(env *core.Env, w, h int) []ui.Line {
	p := env.P
	u.rows = h - 6
	lines := []ui.Line{}
	if l, ok := working(env, "update.check", "Checking"); ok {
		return []ui.Line{l, gap(), muted(env, "Building the new system and comparing it with the running one."), muted(env, "This can take several minutes the first time."), gap(), muted(env, "Nothing is applied until you confirm.")}
	}
	up := env.Data.Update
	if up == nil {
		return []ui.Line{
			heading(env, "Updates"), gap(),
			plain(env, "Not checked yet."), gap(),
			muted(env, "Checking builds the new system without applying it"),
			muted(env, "and shows what would change."), gap(),
			ui.Of(button(env, true, "Check for updates  c"), space(1), button(env, false, "Pending changes  b")),
		}
	}
	if up.UpToDate {
		return []ui.Line{ui.T(p.Ok.Bold(true), ui.G.Tick+"  The system is already up to date"), gap(), muted(env, "Nothing to apply.")}
	}
	c := up.Counts
	lines = append(lines, ui.Of(
		ui.S(p.Ok, fmt.Sprintf("%s%d new   ", ui.G.Add, c.New)),
		ui.S(p.Warn, fmt.Sprintf("%s%d updated   ", ui.G.Up, c.Updated)),
		ui.S(p.Bad, fmt.Sprintf("%s%d removed   ", ui.G.Del, c.Removed)),
		ui.S(p.Mu, fmt.Sprintf("%s%d changed", ui.G.Chg, c.Changed)),
	))
	if c.Config > 0 {
		lines = append(lines, muted(env, fmt.Sprintf("%d configuration files changed (hidden)", c.Config)))
	} else {
		lines = append(lines, gap())
	}
	if up.Kernel {
		lines = append(lines, ui.T(p.Warn, ui.G.Warn+"  includes a new kernel: reboot after applying"))
	} else {
		lines = append(lines, gap())
	}
	lines = append(lines, gap())
	from, to := u.list.window(len(up.Changes), u.rows)
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
	if up != nil && !up.UpToDate {
		lines = append(lines, ui.Of(button(env, true, "Apply  ⏎")), gap(),
			muted(env, "Applying hands you the terminal"), muted(env, "(sudo asks for your password)"),
			muted(env, "and brings you back after."), gap())
	}
	lines = append(lines, ui.Of(button(env, up == nil, "Check for updates  c")), gap(), ui.Of(button(env, false, "Pending changes  b")), gap())
	for _, l := range ui.Wrap("Checking for updates refreshes the inputs in flake.lock. Undo it with git checkout flake.lock.", w) {
		lines = append(lines, ui.T(p.Mu, l))
	}
	return lines
}

func (u *Update) Hints(env *core.Env) []ui.Hint {
	h := []ui.Hint{{Key: "c", Action: "check"}, {Key: "b", Action: "pending"}}
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
