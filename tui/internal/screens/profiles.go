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

// Profiles elige qué paquetes y servicios lleva el sistema según el uso (juegos,
// desarrollo, creación…). Se marca o desmarca con espacio o Intro y se aplica con
// «a»: guarda la elección con `maxor profile enable|disable --no-apply` y cede la
// terminal a `maxor update --no-lock -y` para reconstruir (se ve el progreso y
// funciona sudo). Es lo que antes hacía el asistente, pero para el día a día.
type Profiles struct {
	core.Base
	list   listState
	want   map[string]bool // lo que se quiere (marcado), distinto de lo activo
	inited bool
	rows   int
	run    *Runner // aplicar sin salir de la pantalla
}

func NewProfiles() *Profiles { return &Profiles{want: map[string]bool{}, run: NewRunner("profiles")} }

func (p *Profiles) Blocking() bool { return p.run.Blocking() }
func (p *Profiles) Captures() bool { return p.run.Captures() }

func (p *Profiles) ID() string    { return "profiles" }
func (p *Profiles) Title() string { return tr("Profiles") }

const profItemH = 3 // título, descripción y un respiro

func (p *Profiles) Init(env *core.Env) tea.Cmd {
	if len(env.Data.Profiles) > 0 {
		return nil
	}
	return LoadProfiles(env, false)
}

// settle copia lo activo a lo deseado la primera vez que llegan los datos.
func (p *Profiles) settle(env *core.Env) {
	if p.inited || len(env.Data.Profiles) == 0 {
		return
	}
	p.inited = true
	for _, pr := range env.Data.Profiles {
		p.want[pr.ID] = pr.Enabled
	}
}

// pending son los perfiles cuya elección difiere de lo activo.
func (p *Profiles) pending(env *core.Env) (on, off []maxor.Profile) {
	for _, pr := range env.Data.Profiles {
		switch {
		case p.want[pr.ID] && !pr.Enabled:
			on = append(on, pr)
		case !p.want[pr.ID] && pr.Enabled:
			off = append(off, pr)
		}
	}
	return
}

func (p *Profiles) toggle(env *core.Env) {
	if ps := env.Data.Profiles; p.list.sel >= 0 && p.list.sel < len(ps) {
		id := ps[p.list.sel].ID
		p.want[id] = !p.want[id]
	}
}

// save guarda las elecciones pendientes; al terminar se reconstruye.
func (p *Profiles) save(env *core.Env) tea.Cmd {
	on, off := p.pending(env)
	return env.Tasks.Start(task.Task{ID: "profiles.save", Label: tr("Saving your choices"), Run: func(ctx context.Context) (any, error) {
		for _, pr := range on {
			if err := env.Client.SetProfile(ctx, pr.ID, true); err != nil {
				return nil, fmt.Errorf("profile %s: %w", pr.ID, err)
			}
		}
		for _, pr := range off {
			if err := env.Client.SetProfile(ctx, pr.ID, false); err != nil {
				return nil, fmt.Errorf("profile %s: %w", pr.ID, err)
			}
		}
		return len(on) + len(off), nil
	}})
}

func (p *Profiles) Update(env *core.Env, msg tea.Msg) (core.Screen, tea.Cmd) {
	p.settle(env)
	if cmd, ok, ev := p.run.Handle(env, msg); ok {
		if ev == "finished" {
			p.inited = false
			if !p.run.OK {
				return p, tea.Batch(cmd, core.Toast("bad", tr("The build did not finish: maxor logs --last")), LoadProfiles(env, true))
			}
			return p, tea.Batch(cmd, core.Toast("ok", tr("Profiles applied")), core.Note("ok", tr("Applied the profiles")), LoadProfiles(env, true))
		}
		return p, cmd
	}
	switch m := msg.(type) {
	case task.DoneMsg:
		if m.ID != "profiles.save" {
			return p, nil
		}
		if m.Err != nil {
			return p, tea.Batch(core.Toast("bad", tr("Could not save: %s", oneLine(m.Err.Error()))), LoadProfiles(env, true))
		}
		return p, p.run.Begin(env, tr("Applying your profiles"), "update", "--no-lock", "-y")
	case core.ExecDoneMsg:
		if m.Tag != "profiles" {
			return p, nil
		}
		p.inited = false
		if m.Err != nil {
			return p, tea.Batch(core.Toast("bad", tr("The build did not finish: maxor logs --last")), LoadProfiles(env, true))
		}
		return p, tea.Batch(core.Toast("ok", tr("Profiles applied")), core.Note("ok", tr("Applied the profiles")), LoadProfiles(env, true))
	case tea.KeyMsg:
		ps := env.Data.Profiles
		if d, mv := listKey(m); mv {
			p.list.move(d, len(ps), max(p.rows, 1))
			return p, nil
		}
		switch {
		case isKey(m, " ", "enter"):
			p.toggle(env)
		case isKey(m, "x", "esc"):
			// descartar lo marcado y volver a lo activo
			for _, pr := range ps {
				p.want[pr.ID] = pr.Enabled
			}
		case isKey(m, "a"):
			on, off := p.pending(env)
			if len(on)+len(off) == 0 {
				return p, core.Toast("info", tr("Nothing to apply: mark or unmark a profile first"))
			}
			return p, p.save(env)
		}
	}
	return p, nil
}

func (p *Profiles) Main(env *core.Env, w, h int) []ui.Line {
	if p.run.Active() {
		return p.run.Lines(env, w, h)
	}
	pt := env.P
	p.rows = max((h-2)/profItemH, 1)
	ps := env.Data.Profiles
	if len(ps) == 0 {
		if err := env.Data.Err["profiles"]; err != nil {
			return failed(env, tr("Could not load the profiles"), err)
		}
		lines := []ui.Line{heading(env, tr("Profiles")), gap()}
		for i := 0; i < 4; i++ {
			lines = append(lines, ui.Skeleton(pt, env.Frame+i*2, 3, 16), ui.Skeleton(pt, env.Frame+i*2+1, 3, 34), gap())
		}
		return lines
	}
	p.settle(env)
	active := 0
	for _, pr := range ps {
		if pr.Enabled {
			active++
		}
	}
	var head ui.Line
	if l, ok := working(env, "profiles.save", tr("Saving your choices")); ok {
		head = l
	} else if on, off := p.pending(env); len(on)+len(off) > 0 {
		head = ui.Of(ui.S(pt.Warn, ui.G.Warn+" "+trn("%d change to apply", "%d changes to apply", len(on)+len(off))), ui.S(pt.Mu, tr("  ·  a to apply, x to discard")))
	} else {
		head = heading(env, fmt.Sprintf(tr("%d of %d active"), active, len(ps)))
	}
	lines := []ui.Line{head, gap()}
	from, to := p.list.window(len(ps), p.rows)
	for i := from; i < to; i++ {
		pr := ps[i]
		want := p.want[pr.ID]
		mark := ui.S(pt.Mu, ui.G.Off+"  ")
		if want {
			mark = ui.S(pt.Ac, ui.G.On+"  ")
		}
		var state ui.Seg
		switch {
		case want && !pr.Enabled:
			state = ui.S(pt.Warn, tr("will enable "))
		case !want && pr.Enabled:
			state = ui.S(pt.Warn, tr("will disable "))
		case pr.Enabled:
			state = ui.S(pt.Ok, ui.G.Tick+" "+tr("active")+" ")
		default:
			state = ui.S(pt.Mu, tr("off "))
		}
		lines = append(lines,
			ui.Line{L: []ui.Seg{mark, ui.S(pt.Bold, pr.Title)}, R: []ui.Seg{state}, Sel: i == p.list.sel},
			ui.Line{L: []ui.Seg{ui.S(pt.Mu, "     "+briefIncludes(pr, 3))}, Sel: i == p.list.sel},
			gap())
	}
	return lines
}

func (p *Profiles) Side(env *core.Env, w, h int) []ui.Line {
	if p.run.Active() {
		return nil
	}
	pt := env.P
	ps := env.Data.Profiles
	if len(ps) == 0 || p.list.sel >= len(ps) {
		return []ui.Line{heading(env, tr("Details")), gap(), muted(env, tr("Profiles add the apps and")), muted(env, tr("services for a way of using")), muted(env, tr("the computer."))}
	}
	pr := ps[p.list.sel]
	lines := []ui.Line{heading(env, tr("Details")), gap(), ui.T(pt.Bold, pr.Title), muted(env, pr.ID), gap()}
	for _, l := range ui.Wrap(pr.Description, w) {
		lines = append(lines, plain(env, l))
	}
	lines = append(lines, gap())
	if len(pr.Includes) > 0 {
		lines = append(lines, heading(env, tr("Includes")))
		for _, it := range pr.Includes {
			for i, l := range ui.Wrap(it, w-2) {
				pre := "  "
				if i == 0 {
					pre = ui.G.Add + " "
				}
				lines = append(lines, ui.Of(ui.S(pt.Ac, pre), ui.S(pt.Text, l)))
			}
		}
		lines = append(lines, gap())
	}
	if p.want[pr.ID] {
		lines = append(lines, ui.Of(button(env, false, tr("Turn off  ⏎"))))
	} else {
		lines = append(lines, ui.Of(button(env, false, tr("Turn on  ⏎"))))
	}
	on, off := p.pending(env)
	if len(on)+len(off) > 0 {
		lines = append(lines, gap(), heading(env, tr("To apply")))
		for _, x := range on {
			lines = append(lines, ui.Of(ui.S(pt.Ok, ui.G.Add+" "), ui.S(pt.Text, x.Title)))
		}
		for _, x := range off {
			lines = append(lines, ui.Of(ui.S(pt.Bad, ui.G.Del+" "), ui.S(pt.Text, x.Title)))
		}
		lines = append(lines, gap(), ui.Of(button(env, true, tr("Apply  a")), space(1), button(env, false, tr("Discard  x"))))
		for _, l := range ui.Wrap(tr("Applying rebuilds the system. You will see the progress here and type your password in this screen."), w) {
			lines = append(lines, gap(), ui.T(pt.Mu, l))
			break
		}
	}
	return lines
}

func (p *Profiles) Hints(env *core.Env) []ui.Hint {
	if p.run.Active() {
		return p.run.Hints()
	}
	h := []ui.Hint{{Key: "↑↓", Action: tr("move")}, {Key: "space", Action: tr("toggle")}}
	if on, off := p.pending(env); len(on)+len(off) > 0 {
		h = append(h, ui.Hint{Key: "a", Action: tr("apply")}, ui.Hint{Key: "x", Action: tr("discard")})
	}
	return h
}

func (p *Profiles) Click(env *core.Env, x, y int) tea.Cmd {
	if i := p.list.top + (y-2)/profItemH; y >= 2 && i >= 0 && i < len(env.Data.Profiles) {
		p.list.sel = i
		if x < 5 { // la marca
			p.toggle(env)
		}
	}
	return nil
}

func (p *Profiles) Wheel(env *core.Env, dy int) tea.Cmd {
	if p.run.Active() {
		return nil
	}
	p.list.move(dy*2, len(env.Data.Profiles), max(p.rows, 1))
	return nil
}

var _ = strings.TrimSpace

// Brief: el perfil elegido y lo pendiente de aplicar.
func (p *Profiles) Brief(env *core.Env, w int) []ui.Line {
	if p.run.Active() {
		return nil
	}
	pt := env.P
	ps := env.Data.Profiles
	if len(ps) == 0 || p.list.sel >= len(ps) {
		return []ui.Line{muted(env, tr("Waiting for the profiles."))}
	}
	pr := ps[p.list.sel]
	lines := []ui.Line{ui.Of(ui.S(pt.Bold, pr.Title), ui.S(pt.Mu, "  "+briefIncludes(pr, 4)))}
	if on, off := p.pending(env); len(on)+len(off) > 0 {
		row := []ui.Seg{ui.S(pt.Warn, ui.G.Warn+" "+trn("%d change to apply", "%d changes to apply", len(on)+len(off))+"  "), button(env, true, tr("Apply  a")), space(1), button(env, false, tr("Discard  x"))}
		return append(lines, ui.Of(row...))
	}
	return append(lines, muted(env, tr("space or ⏎ turns it on or off")))
}

// briefIncludes resume lo que trae un perfil: los primeros n elementos y cuántos más.
func briefIncludes(pr maxor.Profile, n int) string {
	if len(pr.Includes) == 0 {
		return firstSentence(pr.Description)
	}
	if len(pr.Includes) <= n {
		return strings.Join(pr.Includes, " · ")
	}
	return strings.Join(pr.Includes[:n], " · ") + fmt.Sprintf(tr(" · +%d more"), len(pr.Includes)-n)
}
