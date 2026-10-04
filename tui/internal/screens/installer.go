package screens

import (
	"context"
	"fmt"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/install"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Installer is the guided installer: thirteen steps, one screen each, with a way back and a gate before
// each way forward. It only collects answers and shows progress; the work is done by the engine
// (`maxor-install`), through the install package. See docs/INSTALLER.md.
type Installer struct {
	core.Base
	idx     int
	steps   []wizStep
	st      install.State
	notice  string // why the user cannot go on, or what failed: shown under the step
	running bool   // the engine is installing: nothing can be left
	failed  bool

	changed int // the animation frame at which the current step started
	bodyH   int // rows the card may use, as of the last frame
	fromReview bool // a step is being edited from the review: the review comes back right after it

	mu     sync.Mutex
	events []install.Event
	logs   []string // the last lines the engine printed, for the log under the stages
}

// wizStep is one screen of the wizard.
type wizStep interface {
	ID() string
	Title() string
	Intro() string
	// Enter runs when the step becomes the current one.
	Enter(w *Installer, env *core.Env) tea.Cmd
	// Key handles a key; advance asks to go on (the gate has already passed).
	Key(w *Installer, env *core.Env, k tea.KeyMsg) (advance bool, cmd tea.Cmd)
	// Done receives the result of a task the step started.
	Done(w *Installer, env *core.Env, d task.DoneMsg) tea.Cmd
	Lines(w *Installer, env *core.Env, width int) []ui.Line
	// Gate says why the user cannot go on yet ("" if they can).
	Gate(w *Installer) string
	// Captures is true while a text field has the focus.
	Captures() bool
}

// NewInstaller builds the wizard with all its steps.
func NewInstaller() *Installer {
	w := &Installer{st: install.Default()}
	w.steps = []wizStep{
		&introStep{}, &welcomeStep{}, &keyboardStep{}, &networkStep{}, &regionStep{}, &diskStep{}, &strategyStep{},
		&storageStep{}, &accountStep{}, &lookStep{}, &hardwareStep{}, &summaryStep{}, &installStep{}, &doneStep{},
	}
	return w
}

func (w *Installer) ID() string    { return "install" }
func (w *Installer) Title() string { return "Install" }

func (w *Installer) cur() wizStep { return w.steps[w.idx] }

// Captures: while a text field is focused every key is text; and during the install the screen keeps `q`
// from leaving (ctrl+c is refused by Blocking).
func (w *Installer) Captures() bool { return w.cur().Captures() || w.idx > 0 && w.idx < len(w.steps)-1 }
func (w *Installer) Blocking() bool { return w.running }

func (w *Installer) Init(env *core.Env) tea.Cmd { return w.cur().Enter(w, env) }

// OwnsHorizontal: ← → belong to the steps (they move inside lists), not to tabs (there are none).
func (w *Installer) OwnsHorizontal() bool { return true }

// go_ moves to step i and starts it.
func (w *Installer) goTo(env *core.Env, i int) tea.Cmd {
	if i < 0 || i >= len(w.steps) {
		return nil
	}
	w.idx, w.notice = i, ""
	w.changed = env.Frame
	return w.cur().Enter(w, env)
}

func (w *Installer) advance(env *core.Env) tea.Cmd {
	if g := w.cur().Gate(w); g != "" {
		w.notice = g
		return nil
	}
	// coming back from the review to change something: once that step is done, the review is next
	if w.fromReview {
		if sum := w.indexOf("summary"); sum > w.idx {
			w.fromReview = false
			return w.goTo(env, sum)
		}
	}
	// a step may be skipped when it does not apply (the strategy step when there is only one way)
	next := w.idx + 1
	for next < len(w.steps)-1 {
		if s, ok := w.steps[next].(interface{ Skip(*Installer) bool }); ok && s.Skip(w) {
			next++
			continue
		}
		break
	}
	return w.goTo(env, next)
}

func (w *Installer) back(env *core.Env) tea.Cmd {
	if w.running || w.idx == 0 {
		return nil
	}
	w.fromReview = false
	prev := w.idx - 1
	for prev > 0 {
		if s, ok := w.steps[prev].(interface{ Skip(*Installer) bool }); ok && s.Skip(w) {
			prev--
			continue
		}
		break
	}
	return w.goTo(env, prev)
}

func (w *Installer) push(e install.Event) {
	w.mu.Lock()
	if e.State == "log" {
		w.logs = append(w.logs, e.Message)
		if len(w.logs) > 200 {
			w.logs = w.logs[len(w.logs)-200:]
		}
		w.mu.Unlock()
		return
	}
	w.events = append(w.events, e)
	w.mu.Unlock()
}

// tailLogs are the last n lines the engine printed.
func (w *Installer) tailLogs(n int) []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.logs) > n {
		return append([]string(nil), w.logs[len(w.logs)-n:]...)
	}
	return append([]string(nil), w.logs...)
}

func (w *Installer) snapshot() []install.Event {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]install.Event(nil), w.events...)
}

func (w *Installer) Update(env *core.Env, msg tea.Msg) (core.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case task.DoneMsg:
		return w, w.cur().Done(w, env, m)
	case tea.KeyMsg:
		if w.running {
			return w, nil
		}
		switch m.String() {
		case "esc":
			if w.idx == len(w.steps)-1 {
				return w, nil
			}
			return w, w.back(env)
		case "ctrl+n":
			return w, w.advance(env)
		}
		next, cmd := w.cur().Key(w, env, m)
		if next {
			return w, tea.Batch(cmd, w.advance(env))
		}
		return w, cmd
	}
	return w, nil
}

// ── Drawing ──────────────────────────────────────────────────────────

func (w *Installer) Main(env *core.Env, width, h int) []ui.Line {
	p := env.P
	s := w.cur()
	lines := []ui.Line{w.stepper(env, width), gap(),
		ui.Of(ui.S(p.Bold, s.Title()), ui.S(p.Mu, fmt.Sprintf("   %d of %d", w.idx+1, len(w.steps)))),
		muted(env, s.Intro()), gap()}
	body := s.Lines(w, env, width)
	lines = append(lines, body...)
	if w.notice != "" {
		lines = append(lines, gap(), ui.Of(ui.S(p.Warn, ui.G.Warn+"  "), ui.S(p.Text, w.notice)))
	}
	return lines
}

// stepper is the row of dots that shows how far along the wizard is.
func (w *Installer) stepper(env *core.Env, width int) ui.Line {
	p := env.P
	var segs []ui.Seg
	for i := range w.steps {
		switch {
		case i < w.idx:
			segs = append(segs, ui.S(p.Ok, ui.G.Swatch))
		case i == w.idx:
			segs = append(segs, ui.S(p.Ac, ui.G.Swatch))
		default:
			segs = append(segs, ui.S(p.Mu, ui.G.Info))
		}
		if i < len(w.steps)-1 {
			segs = append(segs, ui.S(p.Mu, " "))
		}
	}
	return ui.Of(segs...)
}

func (w *Installer) Hints(env *core.Env) []ui.Hint {
	if w.running {
		return []ui.Hint{{Key: "…", Action: "installing: please wait"}}
	}
	if w.idx == len(w.steps)-1 {
		return []ui.Hint{{Key: "↑↓", Action: "choose"}, {Key: "⏎", Action: "confirm"}}
	}
	h := []ui.Hint{{Key: "⏎", Action: "continue"}}
	if w.idx > 0 && w.idx < len(w.steps)-1 {
		h = append(h, ui.Hint{Key: "esc", Action: "back"})
	}
	return h
}

// ── Helpers shared by the steps ──────────────────────────────────────

// runTask starts an asynchronous job owned by the wizard.
func runTask(env *core.Env, name, label string, quiet bool, f func(ctx context.Context) (any, error)) tea.Cmd {
	return env.Tasks.Start(task.Task{ID: "install." + name, Label: label, Quiet: quiet, Run: f})
}

// radio draws one option of a single-choice group.
func radio(env *core.Env, on, focused bool, label, hint string) ui.Line {
	p := env.P
	mark := ui.G.Off
	if on {
		mark = ui.G.On
	}
	st := p.Text
	if on {
		st = p.Ac
	}
	segs := []ui.Seg{ui.S(p.Ac, pad(focused)), ui.S(st, mark+" "), ui.S(p.Text, label)}
	if hint != "" {
		segs = append(segs, ui.S(p.Mu, "  "+hint))
	}
	return ui.Line{L: segs, Sel: focused}
}

func pad(focused bool) string {
	if focused {
		return ui.G.Sel + " "
	}
	return "  "
}

// field draws a labelled text field.
func field(env *core.Env, label string, in *ui.Input, focused bool, width int) ui.Line {
	p := env.P
	segs := []ui.Seg{ui.S(p.Mu, fmt.Sprintf("%-18s", label))}
	segs = append(segs, in.Segs(p, focused, max(width-20, 10))...)
	return ui.Line{L: segs, Sel: focused}
}

// meter draws a strength bar.
func meter(env *core.Env, n int) ui.Seg {
	p := env.P
	bar := strings.Repeat(ui.G.BarOn, n) + strings.Repeat(ui.G.BarOff, 4-n)
	st := p.Bad
	switch {
	case n >= 3:
		st = p.Ok
	case n == 2:
		st = p.Warn
	}
	return ui.S(st, bar)
}

// picker is a list you filter by typing and move with the arrows.
type picker struct {
	in   ui.Input
	list listState
}

func newPicker(placeholder string) picker {
	p := picker{}
	p.in.Placeholder = placeholder
	return p
}

// handle processes a key; moved reports that the selection or the filter changed.
func (p *picker) handle(k tea.KeyMsg, n, rows int) (changed bool) {
	switch k.String() {
	case "up":
		p.list.move(-1, n, rows)
		return true
	case "down":
		p.list.move(1, n, rows)
		return true
	case "home":
		p.list.move(-n, n, rows)
		return true
	case "end":
		p.list.move(n, n, rows)
		return true
	case "pgup":
		p.list.move(-rows, n, rows)
		return true
	case "pgdown":
		p.list.move(rows, n, rows)
		return true
	}
	if ch, handled := p.in.Key(k); handled {
		if ch {
			p.list = listState{}
		}
		return ch
	}
	return false
}

// indexOf is the position of the step with this ID, or -1.
func (w *Installer) indexOf(id string) int {
	for i, s := range w.steps {
		if s.ID() == id {
			return i
		}
	}
	return -1
}

// editStep takes the person from the review back to the n-th step of the list (1 is the first question).
func (w *Installer) editStep(env *core.Env, n int) tea.Cmd {
	vis := w.visible()
	if n < 1 || n > len(vis) || vis[n-1] >= w.idx {
		return nil
	}
	w.fromReview = true
	return w.goTo(env, vis[n-1])
}
