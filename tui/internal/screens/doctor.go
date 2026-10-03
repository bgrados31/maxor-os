package screens

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Doctor muestra las comprobaciones del sistema, agrupadas.
type Doctor struct {
	core.Base
	list listState
	rows int
}

func NewDoctor() *Doctor { return &Doctor{} }

func (d *Doctor) ID() string    { return "doctor" }
func (d *Doctor) Title() string { return "Doctor" }

func (d *Doctor) Init(env *core.Env) tea.Cmd { return LoadDoctor(env, false) }

type doctorRow struct {
	group string
	item  int // -1 si es una cabecera de grupo
	level string
	text  string
}

// flat aplana los grupos en filas; solo las comprobaciones se pueden seleccionar.
func (d *Doctor) flat(env *core.Env) (rows []doctorRow, items []int) {
	if env.Data.Doctor == nil {
		return nil, nil
	}
	for _, g := range env.Data.Doctor.Groups {
		rows = append(rows, doctorRow{group: g.Title, item: -1})
		for _, it := range g.Items {
			items = append(items, len(rows))
			rows = append(rows, doctorRow{group: g.Title, item: len(items) - 1, level: it.Level, text: it.Text})
		}
	}
	return
}

// RunDoctorMsg pide volver a ejecutar las comprobaciones (la paleta de comandos).
type RunDoctorMsg struct{}

func (d *Doctor) Update(env *core.Env, msg tea.Msg) (core.Screen, tea.Cmd) {
	if _, ok := msg.(RunDoctorMsg); ok {
		return d, LoadDoctor(env, false)
	}
	if k, ok := msg.(tea.KeyMsg); ok {
		_, items := d.flat(env)
		if dy, mv := listKey(k); mv {
			d.list.move(dy, len(items), 1<<20) // la ventana se ajusta al dibujar
			return d, nil
		}
		if isKey(k, "r") {
			return d, LoadDoctor(env, false)
		}
	}
	return d, nil
}

// advice son los consejos para los avisos conocidos.
var advice = []struct{ match, text string }{
	{"uncommitted changes", "Commit or discard the changes: git -C ~/nixos-config status. maxor update keeps working meanwhile."},
	{"kernel is not the running one", "Reboot to start the kernel that is installed."},
	{"hardware changed", "Run maxor hardware detect --write, review it and commit it."},
	{"no hardware.json", "Run maxor hardware detect --write so drivers can be chosen for this machine."},
	{"theme missing", "Apply a theme again from the Themes tab."},
	{"no theme applied", "Pick one in the Themes tab."},
	{"failed system service", "See which one: systemctl --failed"},
	{"failed user service", "See which one: systemctl --user --failed"},
	{"is not active", "Check it: systemctl --user status <service>"},
}

func (d *Doctor) Main(env *core.Env, w, h int) []ui.Line {
	p := env.P
	if line, ok := working(env, "data.doctor", "Running checks"); ok && env.Data.Doctor == nil {
		return []ui.Line{line, gap(), muted(env, "This takes a moment.")}
	}
	if env.Data.Doctor == nil {
		if err := env.Data.Err["doctor"]; err != nil {
			return failed(env, "Could not run the checks", err)
		}
		return append([]ui.Line{heading(env, "Checks")}, skeletonRows(env, 8, 3, 28)...)
	}
	doc := env.Data.Doctor
	rows, items := d.flat(env)
	d.rows = h
	var summary ui.Line
	switch {
	case doc.Fails > 0:
		summary = ui.T(p.Bad.Bold(true), ui.G.Bad+"  problems found")
	case doc.Warns > 0:
		summary = ui.T(p.Warn.Bold(true), ui.G.Warn+"  works, with warnings")
	default:
		summary = ui.T(p.Ok.Bold(true), ui.G.Tick+"  all good")
	}
	// Se desplaza para que la fila seleccionada se vea.
	selRow := 0
	if len(items) > 0 {
		selRow = items[min(d.list.sel, len(items)-1)]
	}
	avail := h - 2
	top := 0
	if selRow >= avail {
		top = selRow - avail + 1
	}
	lines := []ui.Line{summary, gap()}
	for i := top; i < len(rows) && len(lines) < h; i++ {
		r := rows[i]
		if r.item < 0 {
			lines = append(lines, ui.T(p.Ac2.Bold(true), strings.ToUpper(r.group)))
			continue
		}
		st, g := p.Level(r.level)
		lines = append(lines, ui.Line{L: []ui.Seg{ui.S(st, g+" "), ui.S(p.Text, r.text)}, Sel: i == selRow})
	}
	return lines
}

func (d *Doctor) Side(env *core.Env, w, h int) []ui.Line {
	p := env.P
	rows, items := d.flat(env)
	if len(items) == 0 {
		return []ui.Line{heading(env, "Details"), gap(), muted(env, "Waiting for the checks.")}
	}
	r := rows[items[min(d.list.sel, len(items)-1)]]
	title := map[string]string{"ok": "Passing", "warn": "Needs attention", "bad": "Problem"}[r.level]
	st, _ := p.Level(r.level)
	lines := []ui.Line{heading(env, "Details"), gap(), ui.T(st.Bold(true), title), muted(env, r.group), gap()}
	for _, l := range ui.Wrap(r.text, w) {
		lines = append(lines, plain(env, l))
	}
	for _, a := range advice {
		if r.level != "ok" && strings.Contains(r.text, a.match) {
			lines = append(lines, gap())
			for _, l := range ui.Wrap(a.text, w) {
				lines = append(lines, muted(env, l))
			}
			break
		}
	}
	return lines
}

func (d *Doctor) Hints(env *core.Env) []ui.Hint {
	return []ui.Hint{{Key: "↑↓", Action: "move"}, {Key: "r", Action: "run again"}}
}

func (d *Doctor) Wheel(env *core.Env, dy int) tea.Cmd {
	_, items := d.flat(env)
	d.list.move(dy*3, len(items), 1<<20)
	return nil
}
