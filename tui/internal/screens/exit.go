package screens

import (

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Exit es una pestaña más: llegar a ella no sale, enseña lo que queda pendiente y
// sale con Intro. Así un ← o → de más nunca cierra la pantalla sin querer.
type Exit struct {
	core.Base
	btnY int // fila del botón, para el ratón
}

func NewExit() *Exit { return &Exit{} }

func (e *Exit) ID() string    { return "exit" }
func (e *Exit) Title() string { return tr("Exit") }

func (e *Exit) Init(env *core.Env) tea.Cmd {
	cmds := ReleaseInit(env)
	if !env.Data.UpdatesKnown {
		cmds = append(cmds, LoadAppUpdates(env, true))
	}
	return tea.Batch(cmds...)
}

func (e *Exit) Update(env *core.Env, msg tea.Msg) (core.Screen, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && isKey(k, "enter") {
		return e, core.Quit()
	}
	return e, nil
}

func (e *Exit) Main(env *core.Env, w, h int) []ui.Line {
	p := env.P
	lines := []ui.Line{heading(env, tr("Leave Maxor")), gap(),
		plain(env, tr("Your terminal comes back exactly as you left it.")), gap()}
	// lo que queda pendiente, para no irte sin verlo
	var pending []ui.Line
	if n := len(env.Data.AppUpdates); n > 0 {
		pending = append(pending, ui.Of(ui.S(p.Warn, ui.G.Up+" "), ui.S(p.Text, trn("%d app can be updated", "%d apps can be updated", n)), ui.S(p.Mu, tr("  (Store)"))))
	}
	if d := env.Data.Doctor; d != nil && (d.Warns > 0 || d.Fails > 0) {
		pending = append(pending, ui.Of(ui.S(p.Warn, ui.G.Warn+" "), ui.S(p.Text, trn("%d thing needs attention", "%d things need attention", d.Warns+d.Fails)), ui.S(p.Mu, tr("  (Doctor)"))))
	}
	if releaseAvailable(env) {
		pending = append(pending, ui.Of(ui.S(p.Warn, ui.G.Up+" "), ui.S(p.Text, tr("Maxor OS %s is available", env.Data.Release.Latest)), ui.S(p.Mu, tr("  (Update)"))))
	}
	if u := env.Data.Update; u != nil && !u.UpToDate {
		pending = append(pending, ui.Of(ui.S(p.Warn, ui.G.Up+" "), ui.S(p.Text, tr("a system update is ready to apply")), ui.S(p.Mu, tr("  (Update)"))))
	}
	if len(pending) > 0 {
		lines = append(lines, heading(env, tr("Before you go")))
		lines = append(lines, pending...)
		lines = append(lines, gap())
	} else {
		lines = append(lines, ui.T(p.Ok, ui.G.Tick+" "+tr("Nothing pending.")), gap())
	}
	e.btnY = len(lines)
	return append(lines, ui.Of(button(env, true, tr("Exit  ⏎"))), gap(), muted(env, tr("You can also press q on any tab.")))
}

func (e *Exit) Hints(env *core.Env) []ui.Hint {
	return []ui.Hint{{Key: "⏎", Action: "exit"}, {Key: "←→", Action: tr("stay: pick another tab")}}
}

func (e *Exit) Click(env *core.Env, x, y int) tea.Cmd {
	if y == e.btnY {
		return core.Quit()
	}
	return nil
}
