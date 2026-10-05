package screens

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Runner es el panel que aplica un cambio del sistema sin salir de la pantalla: pide la
// contraseña en su propio campo (si sudo la necesita) y enseña el progreso real mientras
// `maxor` trabaja. La contraseña va solo a sudo, por la entrada estándar, y no se guarda.
//
//	1 comprobando sudo · 2 pidiendo la contraseña · 3 trabajando · 4 terminado
type Runner struct {
	owner string // la pantalla que lo usa: el texto antes del primer punto de los IDs
	stage int
	title string
	args  []string

	pw    ui.Input
	wrong bool // la contraseña anterior no valió

	mu      sync.Mutex
	lines   []string
	started time.Time
	took    time.Duration

	OK        bool // terminó bien
	Cancelled bool // se cerró sin ejecutar nada
	Code      int
}

func NewRunner(owner string) *Runner {
	r := &Runner{owner: owner}
	r.pw.Mask = true
	return r
}

func (r *Runner) Active() bool   { return r.stage != 0 }
func (r *Runner) Blocking() bool { return r.stage == 3 } // mientras trabaja no se puede salir
func (r *Runner) Captures() bool { return r.stage == 2 } // escribiendo la contraseña: todo es texto

func (r *Runner) idCheck() string { return r.owner + ".priv.check" }
func (r *Runner) idRun() string   { return r.owner + ".priv.run" }

// Begin empieza: mira si sudo ya está listo y, si no, pide la contraseña.
func (r *Runner) Begin(env *core.Env, title string, args ...string) tea.Cmd {
	r.stage, r.title, r.args = 1, title, args
	r.pw.Set("")
	r.wrong, r.OK, r.Cancelled, r.Code = false, false, false, 0
	r.mu.Lock()
	r.lines = nil
	r.mu.Unlock()
	return env.Tasks.Start(task.Task{ID: r.idCheck(), Label: tr("Checking permissions"), Quiet: true, Run: func(ctx context.Context) (any, error) {
		return env.Client.SudoReady(ctx), nil
	}})
}

type runResult struct {
	code int
	took time.Duration
}

func (r *Runner) run(env *core.Env, password string) tea.Cmd {
	r.stage = 3
	r.started = time.Now()
	r.mu.Lock()
	r.lines = nil
	r.mu.Unlock()
	args := r.args
	return env.Tasks.Start(task.Task{ID: r.idRun(), Label: r.title, Quiet: true, Run: func(ctx context.Context) (any, error) {
		t0 := time.Now()
		code, err := env.Client.Stream(ctx, password, r.push, args...)
		return runResult{code: code, took: time.Since(t0)}, err
	}})
}

func (r *Runner) push(l string) {
	r.mu.Lock()
	r.lines = append(r.lines, l)
	if len(r.lines) > 400 {
		r.lines = r.lines[len(r.lines)-400:]
	}
	r.mu.Unlock()
}

func (r *Runner) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.lines...)
}

// wrongPassword reconoce el fallo de sudo por contraseña incorrecta.
func (r *Runner) wrongPassword() bool {
	for _, l := range r.snapshot() {
		low := strings.ToLower(l)
		if strings.Contains(low, "incorrect password") || strings.Contains(low, "sorry, try again") || strings.Contains(low, "no password was provided") {
			return true
		}
	}
	return false
}

// Handle procesa un mensaje si el panel está activo. Devuelve si lo atendió y, cuando algo
// relevante ocurrió, un evento: «finished» (terminó el trabajo) o «closed» (se cerró el panel).
func (r *Runner) Handle(env *core.Env, msg tea.Msg) (cmd tea.Cmd, handled bool, event string) {
	if !r.Active() {
		return nil, false, ""
	}
	switch m := msg.(type) {
	case task.DoneMsg:
		switch m.ID {
		case r.idCheck():
			if ready, _ := m.Value.(bool); ready {
				return r.run(env, ""), true, ""
			}
			r.stage = 2
			return nil, true, ""
		case r.idRun():
			res, _ := m.Value.(runResult)
			r.took = res.took
			r.Code = res.code
			if m.Err == nil && res.code != 0 && r.wrongPassword() {
				r.stage, r.wrong = 2, true
				r.pw.Set("")
				return nil, true, ""
			}
			r.OK = m.Err == nil && res.code == 0
			r.stage = 4
			return nil, true, "finished"
		}
	case tea.KeyMsg:
		switch r.stage {
		case 2:
			switch {
			case isKey(m, "enter"):
				if r.pw.Text() == "" {
					return nil, true, ""
				}
				pw := r.pw.Text()
				r.pw.Set("")
				return r.run(env, pw), true, ""
			case isKey(m, "esc"):
				r.stage, r.Cancelled = 0, true
				return nil, true, "closed"
			}
			r.pw.Key(m)
			return nil, true, ""
		case 3:
			return core.Toast("info", tr("Working: wait until it finishes")), true, ""
		case 4:
			if isKey(m, "enter", "esc", "q") {
				r.stage = 0
				return nil, true, "closed"
			}
			return nil, true, ""
		default:
			return nil, true, ""
		}
	}
	return nil, false, ""
}

// phases son los pasos que se enseñan mientras trabaja, y cuál está en curso según lo que
// ha dicho el comando.
func (r *Runner) phase(lines []string) int {
	cur := 0 // preparar y construir
	for _, l := range lines {
		low := strings.ToLower(l)
		switch {
		case strings.Contains(low, "activating") || strings.Contains(low, "switch-to-configuration") || strings.Contains(low, "setting up"):
			cur = 1
		case strings.Contains(low, "restarting") || strings.Contains(low, "reloading") || strings.Contains(low, "starting the following") || strings.Contains(low, "stopping the following"):
			cur = 2
		}
	}
	return cur
}

// Lines dibuja el panel en el área principal (w × h).
func (r *Runner) Lines(env *core.Env, w, h int) []ui.Line {
	p := env.P
	switch r.stage {
	case 1:
		return []ui.Line{heading(env, r.title), gap(), ui.Of(ui.S(p.Ac, ui.Spin(env.Frame)+"  "), ui.S(p.Text, tr("Checking permissions…")))}
	case 2:
		lines := []ui.Line{heading(env, tr("Administrator password")), gap()}
		for _, l := range ui.Wrap(tr("Changing the system needs your password. It goes only to sudo and is never saved or shown."), w) {
			lines = append(lines, plain(env, l))
		}
		lines = append(lines, gap())
		if r.wrong {
			lines = append(lines, ui.T(p.Bad, ui.G.Bad+" "+tr("That password did not work. Try again.")), gap())
		}
		field := append([]ui.Seg{ui.S(p.Mu, tr("password  "))}, r.pw.Segs(p, true, max(w-14, 8))...)
		lines = append(lines, ui.Of(field...), gap(), ui.Of(button(env, true, tr("Continue  ⏎")), space(1), button(env, false, tr("Cancel  esc"))), gap())
		return append(lines, muted(env, tr("Prefer to type it in the terminal? Cancel, then press t.")))
	}
	snap := r.snapshot()
	steps := []string{tr("Build the new system"), tr("Switch to it"), tr("Restart what changed")}
	cur := r.phase(snap)
	lines := []ui.Line{}
	switch r.stage {
	case 3:
		el := time.Since(r.started).Round(time.Second)
		lines = append(lines, ui.Of(ui.S(p.Ac, ui.Spin(env.Frame)+"  "), ui.S(p.Text.Bold(true), r.title), ui.S(p.Mu, "  "+el.String())), gap())
	default:
		if r.OK {
			lines = append(lines, ui.T(p.Ok.Bold(true), ui.G.Tick+"  "+tr("Done in %s", r.took.Round(time.Second).String())), gap())
		} else {
			lines = append(lines, ui.T(p.Bad.Bold(true), ui.G.Bad+"  "+tr("It did not finish")), gap())
		}
	}
	for i, s := range steps {
		switch {
		case r.stage == 4 && r.OK, i < cur:
			lines = append(lines, ui.Of(ui.S(p.Ok, ui.G.Tick+" "), ui.S(p.Mu, s)))
		case r.stage == 3 && i == cur:
			lines = append(lines, ui.Of(ui.S(p.Ac, ui.Spin(env.Frame)+" "), ui.S(p.Text, s)))
		case r.stage == 4 && i == cur:
			lines = append(lines, ui.Of(ui.S(p.Bad, ui.G.Bad+" "), ui.S(p.Text, s)))
		default:
			lines = append(lines, ui.Of(ui.S(p.Mu, ui.G.Off+" "), ui.S(p.Mu, s)))
		}
	}
	// lo último que dijo el comando, atenuado
	room := h - len(lines) - 5
	if room > 0 && len(snap) > 0 {
		lines = append(lines, gap(), heading(env, tr("Output")))
		if len(snap) > room {
			snap = snap[len(snap)-room:]
		}
		for _, l := range snap {
			lines = append(lines, ui.T(p.Mu, l))
		}
	}
	if r.stage == 4 {
		lines = append(lines, gap())
		if !r.OK {
			lines = append(lines, muted(env, fmt.Sprintf(tr("Exit code %d. More detail: maxor logs --last"), r.Code)))
		}
		lines = append(lines, ui.Of(button(env, true, tr("Close  ⏎"))))
	}
	return lines
}

// Hints son las teclas de cada etapa.
func (r *Runner) Hints() []ui.Hint {
	switch r.stage {
	case 2:
		return []ui.Hint{{Key: "⏎", Action: tr("continue")}, {Key: "esc", Action: tr("cancel")}}
	case 3:
		return []ui.Hint{{Key: "…", Action: tr("working: please wait")}}
	case 4:
		return []ui.Hint{{Key: "⏎", Action: tr("close")}}
	}
	return nil
}

// runErr da el fallo de la última ejecución como error (nil si salió bien).
func runErr(r *Runner) error {
	if r.OK {
		return nil
	}
	return fmt.Errorf("exit code %d", r.Code)
}
