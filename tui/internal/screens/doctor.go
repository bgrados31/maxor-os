package screens

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/maxor"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Doctor muestra las comprobaciones del sistema, agrupadas. Cada aviso que tiene
// arreglo lo ofrece: Intro lo prepara y un segundo Intro lo ejecuta (cediendo la
// terminal, como Update). El comando lo decide la CLI (`doctor --json`, campo fix).
type Doctor struct {
	core.Base
	list  listState
	rows  int
	inited bool
	armed string // id de la comprobación cuyo arreglo está preparado
}

func NewDoctor() *Doctor { return &Doctor{} }

func (d *Doctor) ID() string    { return "doctor" }
func (d *Doctor) Title() string { return "Doctor" }

func (d *Doctor) Init(env *core.Env) tea.Cmd { return LoadDoctor(env, false) }

// RunDoctorMsg pide volver a ejecutar las comprobaciones (la paleta de comandos).
type RunDoctorMsg struct{}

type doctorRow struct {
	group string
	item  int // -1 si es una cabecera de grupo
	it    maxor.DoctorItem
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
			rows = append(rows, doctorRow{group: g.Title, item: len(items) - 1, it: it})
		}
	}
	return
}

// settle deja la selección en la primera comprobación que necesita atención.
func (d *Doctor) settle(env *core.Env) {
	if d.inited || env.Data.Doctor == nil {
		return
	}
	d.inited = true
	rows, items := d.flat(env)
	for n, ri := range items {
		if rows[ri].it.Level != "ok" {
			d.list.sel = n
			return
		}
	}
}

func (d *Doctor) current(env *core.Env) (doctorRow, bool) {
	rows, items := d.flat(env)
	if len(items) == 0 {
		return doctorRow{}, false
	}
	return rows[items[min(d.list.sel, len(items)-1)]], true
}

// fixCmd cede la terminal al arreglo y espera un Intro antes de volver.
func fixCmd(fix string) *exec.Cmd {
	return exec.Command("sh", "-c", fix+`; rc=$?; echo; printf 'Press Enter to return to Maxor… '; read _; exit $rc`)
}

func (d *Doctor) Update(env *core.Env, msg tea.Msg) (core.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case RunDoctorMsg:
		d.armed = ""
		return d, LoadDoctor(env, false)
	case core.ExecDoneMsg:
		if m.Tag != "doctor" {
			return d, nil
		}
		d.armed, d.inited = "", false
		kind, text := "ok", "Done. Checking again…"
		if m.Err != nil {
			kind, text = "warn", "The command ended with an error. Checking again…"
		}
		return d, tea.Batch(core.Toast(kind, text), LoadDoctor(env, false))
	case tea.KeyMsg:
		_, items := d.flat(env)
		if dy, mv := listKey(m); mv {
			d.armed = ""
			d.list.move(dy, len(items), 1<<20) // la ventana se ajusta al dibujar
			return d, nil
		}
		switch {
		case isKey(m, "r"):
			d.armed = ""
			return d, LoadDoctor(env, false)
		case isKey(m, "enter"):
			row, ok := d.current(env)
			if !ok {
				return d, nil
			}
			if row.it.Fix == "" {
				if row.it.Level == "ok" {
					return d, core.Toast("ok", "This one passes: nothing to fix")
				}
				return d, core.Toast("info", "No automatic fix for this one: see the advice on the right")
			}
			if d.armed != row.it.ID && row.it.Kind != "inspect" {
				d.armed = row.it.ID
				return d, nil // el panel lateral enseña el comando y pide un segundo Intro
			}
			d.armed = ""
			return d, tea.ExecProcess(fixCmd(row.it.Fix), func(err error) tea.Msg { return core.ExecDoneMsg{Tag: "doctor", Err: err} })
		}
	}
	return d, nil
}

// advice son los consejos para los avisos sin arreglo automático.
var advice = []struct{ match, text string }{
	{"kernel is not the running one", "Reboot to start the kernel that is installed."},
	{"theme missing", "Apply a theme again from the Themes tab."},
	{"nvidia-smi does not respond", "The NVIDIA driver may not be loaded: reboot, or check journalctl -b | grep -i nvidia."},
	{"nvidia-offload is missing", "NVIDIA is present but the offload command is missing: check the hardware module."},
	{"missing font", "A Maxor font is not installed: maxor update brings it back."},
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
		lines := []ui.Line{heading(env, "Checks"), gap()}
		for i := 0; i < 9; i++ {
			lines = append(lines, ui.Skeleton(p, env.Frame+i, 2, 24+i*3%14))
		}
		return lines
	}
	d.settle(env)
	doc := env.Data.Doctor
	rows, items := d.flat(env)
	d.rows = h
	var summary ui.Line
	switch {
	case doc.Fails > 0:
		summary = ui.T(p.Bad.Bold(true), ui.G.Bad+"  problems found")
	case doc.Warns > 0:
		summary = ui.T(p.Warn.Bold(true), ui.G.Warn+"  works, with "+plural(doc.Warns, "warning", "warnings"))
	default:
		summary = ui.T(p.Ok.Bold(true), ui.G.Tick+"  all good")
	}
	selRow := 0
	if len(items) > 0 {
		selRow = items[min(d.list.sel, len(items)-1)]
	}
	avail := h - 2
	top := 0
	if selRow >= avail {
		top = selRow - avail + 1
	}
	// al subir se ve la cabecera del grupo
	if selRow > 0 && rows[selRow-1].item < 0 && top > selRow-1 {
		top = selRow - 1
	}
	lines := []ui.Line{summary, gap()}
	for i := top; i < len(rows) && len(lines) < h; i++ {
		r := rows[i]
		if r.item < 0 {
			lines = append(lines, ui.T(p.Ac2.Bold(true), strings.ToUpper(r.group)))
			continue
		}
		st, g := p.Level(r.it.Level)
		ln := ui.Line{L: []ui.Seg{ui.S(st, g+" "), ui.S(p.Text, r.it.Text)}, Sel: i == selRow}
		if r.it.Fix != "" {
			tag := "fix ⏎ "
			if r.it.Kind == "inspect" {
				tag = "look ⏎ "
			}
			ln.R = []ui.Seg{ui.S(p.Ac, tag)}
		}
		lines = append(lines, ln)
	}
	return lines
}

func (d *Doctor) Side(env *core.Env, w, h int) []ui.Line {
	p := env.P
	d.settle(env)
	row, ok := d.current(env)
	if !ok {
		return []ui.Line{heading(env, "Details"), gap(), muted(env, "Waiting for the checks.")}
	}
	title := map[string]string{"ok": "Passing", "warn": "Needs attention", "bad": "Problem"}[row.it.Level]
	st, _ := p.Level(row.it.Level)
	lines := []ui.Line{heading(env, "Details"), gap(), ui.T(st.Bold(true), title), muted(env, row.group), gap()}
	for _, l := range ui.Wrap(row.it.Text, w) {
		lines = append(lines, plain(env, l))
	}
	if row.it.Level != "ok" {
		lines = append(lines, gap())
		if row.it.Fix != "" {
			inspect := row.it.Kind == "inspect"
			if inspect {
				lines = append(lines, heading(env, "Take a look"), muted(env, "Only shows information: nothing changes."))
			} else {
				lines = append(lines, heading(env, "Fix"))
			}
			for i, l := range ui.Wrap(row.it.Fix, w-2) {
				pre := "  "
				if i == 0 {
					pre = "$ "
				}
				lines = append(lines, ui.T(p.Ac, pre+l))
			}
			lines = append(lines, gap())
			switch {
			case d.armed == row.it.ID && row.it.Confirm:
				lines = append(lines, ui.T(p.Warn, ui.G.Warn+" this changes your system"), ui.Of(button(env, true, "Run it  ⏎")), muted(env, "or move away to cancel"))

			case d.armed == row.it.ID:
				lines = append(lines, ui.Of(button(env, true, "Run it  ⏎")), muted(env, "or move away to cancel"))
			case inspect:
				lines = append(lines, ui.Of(button(env, true, "Show it  ⏎")))
			default:
				lines = append(lines, ui.Of(button(env, true, "Prepare fix  ⏎")))
			}
		} else {
			for _, a := range advice {
				if strings.Contains(row.it.Text, a.match) {
					for _, l := range ui.Wrap(a.text, w) {
						lines = append(lines, muted(env, l))
					}
					break
				}
			}
		}
	}
	return lines
}

func (d *Doctor) Hints(env *core.Env) []ui.Hint {
	return []ui.Hint{{Key: "↑↓", Action: "move"}, {Key: "⏎", Action: "fix"}, {Key: "r", Action: "run again"}}
}

func (d *Doctor) Wheel(env *core.Env, dy int) tea.Cmd {
	_, items := d.flat(env)
	d.armed = ""
	d.list.move(dy*3, len(items), 1<<20)
	return nil
}

func (d *Doctor) Click(env *core.Env, x, y int) tea.Cmd {
	rows, items := d.flat(env)
	_ = rows
	// la fila 0 es el resumen, la 1 un hueco; después las filas con cabeceras
	avail := d.rows - 2
	selRow := 0
	if len(items) > 0 {
		selRow = items[min(d.list.sel, len(items)-1)]
	}
	top := 0
	if selRow >= avail {
		top = selRow - avail + 1
	}
	target := top + y - 2
	for n, ri := range items {
		if ri == target {
			d.list.sel, d.armed = n, ""
		}
	}
	return nil
}

// Brief: la comprobación elegida, y su comando y cómo ejecutarlo si lo tiene.
func (d *Doctor) Brief(env *core.Env, w int) []ui.Line {
	p := env.P
	d.settle(env)
	row, ok := d.current(env)
	if !ok {
		return []ui.Line{muted(env, "Waiting for the checks.")}
	}
	st, g := p.Level(row.it.Level)
	lines := []ui.Line{ui.Of(ui.S(st, g+" "), ui.S(p.Text, row.it.Text))}
	switch {
	case row.it.Level == "ok":
		return append(lines, muted(env, "This one passes: nothing to do."))
	case row.it.Fix == "":
		for _, a := range advice {
			if strings.Contains(row.it.Text, a.match) {
				return append(lines, muted(env, a.text))
			}
		}
		return append(lines, muted(env, "No automatic fix for this one."))
	}
	lines = append(lines, ui.T(p.Ac, "$ "+row.it.Fix))
	switch {
	case row.it.Kind == "inspect":
		lines = append(lines, ui.Of(button(env, true, "Show it  ⏎"), ui.S(p.Mu, "  only shows information")))
	case d.armed == row.it.ID && row.it.Confirm:
		lines = append(lines, ui.Of(ui.S(p.Warn, ui.G.Warn+" this changes your system  "), button(env, true, "Run it  ⏎"), ui.S(p.Mu, "  or move to cancel")))
	case d.armed == row.it.ID:
		lines = append(lines, ui.Of(button(env, true, "Run it  ⏎"), ui.S(p.Mu, "  or move to cancel")))
	default:
		lines = append(lines, ui.Of(button(env, true, "Prepare fix  ⏎")))
	}
	return lines
}
