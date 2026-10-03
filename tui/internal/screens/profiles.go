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
}

func NewProfiles() *Profiles { return &Profiles{want: map[string]bool{}} }

func (p *Profiles) ID() string    { return "profiles" }
func (p *Profiles) Title() string { return "Profiles" }

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
	return env.Tasks.Start(task.Task{ID: "profiles.save", Label: "Saving your choices", Run: func(ctx context.Context) (any, error) {
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
	switch m := msg.(type) {
	case task.DoneMsg:
		if m.ID != "profiles.save" {
			return p, nil
		}
		if m.Err != nil {
			return p, tea.Batch(core.Toast("bad", "Could not save: "+oneLine(m.Err.Error())), LoadProfiles(env, true))
		}
		return p, tea.ExecProcess(applyCmd(), func(err error) tea.Msg { return core.ExecDoneMsg{Tag: "profiles", Err: err} })
	case core.ExecDoneMsg:
		if m.Tag != "profiles" {
			return p, nil
		}
		p.inited = false
		if m.Err != nil {
			return p, tea.Batch(core.Toast("bad", "The build did not finish: maxor logs --last"), LoadProfiles(env, true))
		}
		return p, tea.Batch(core.Toast("ok", "Profiles applied"), core.Note("ok", "Applied the profiles"), LoadProfiles(env, true))
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
				return p, core.Toast("info", "Nothing to apply: mark or unmark a profile first")
			}
			return p, p.save(env)
		}
	}
	return p, nil
}

func (p *Profiles) Main(env *core.Env, w, h int) []ui.Line {
	pt := env.P
	p.rows = max((h-2)/profItemH, 1)
	ps := env.Data.Profiles
	if len(ps) == 0 {
		if err := env.Data.Err["profiles"]; err != nil {
			return failed(env, "Could not load the profiles", err)
		}
		lines := []ui.Line{heading(env, "Profiles"), gap()}
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
	if l, ok := working(env, "profiles.save", "Saving your choices"); ok {
		head = l
	} else if on, off := p.pending(env); len(on)+len(off) > 0 {
		head = ui.Of(ui.S(pt.Warn, ui.G.Warn+" "+plural(len(on)+len(off), "change", "changes")+" to apply"), ui.S(pt.Mu, "  ·  a to apply, x to discard"))
	} else {
		head = heading(env, fmt.Sprintf("%d of %d active", active, len(ps)))
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
			state = ui.S(pt.Warn, "will enable ")
		case !want && pr.Enabled:
			state = ui.S(pt.Warn, "will disable ")
		case pr.Enabled:
			state = ui.S(pt.Ok, ui.G.Tick+" active ")
		default:
			state = ui.S(pt.Mu, "off ")
		}
		lines = append(lines,
			ui.Line{L: []ui.Seg{mark, ui.S(pt.Bold, pr.Title)}, R: []ui.Seg{state}, Sel: i == p.list.sel},
			ui.Line{L: []ui.Seg{ui.S(pt.Mu, "     "+firstSentence(pr.Description))}, Sel: i == p.list.sel},
			gap())
	}
	return lines
}

func (p *Profiles) Side(env *core.Env, w, h int) []ui.Line {
	pt := env.P
	ps := env.Data.Profiles
	if len(ps) == 0 || p.list.sel >= len(ps) {
		return []ui.Line{heading(env, "Details"), gap(), muted(env, "Profiles add the apps and"), muted(env, "services for a way of using"), muted(env, "the computer.")}
	}
	pr := ps[p.list.sel]
	lines := []ui.Line{heading(env, "Details"), gap(), ui.T(pt.Bold, pr.Title), muted(env, pr.ID), gap()}
	for _, l := range ui.Wrap(pr.Description, w) {
		lines = append(lines, plain(env, l))
	}
	lines = append(lines, gap())
	if p.want[pr.ID] {
		lines = append(lines, ui.Of(button(env, false, "Turn off  ⏎")))
	} else {
		lines = append(lines, ui.Of(button(env, false, "Turn on  ⏎")))
	}
	on, off := p.pending(env)
	if len(on)+len(off) > 0 {
		lines = append(lines, gap(), heading(env, "To apply"))
		for _, x := range on {
			lines = append(lines, ui.Of(ui.S(pt.Ok, ui.G.Add+" "), ui.S(pt.Text, x.Title)))
		}
		for _, x := range off {
			lines = append(lines, ui.Of(ui.S(pt.Bad, ui.G.Del+" "), ui.S(pt.Text, x.Title)))
		}
		lines = append(lines, gap(), ui.Of(button(env, true, "Apply  a"), space(1), button(env, false, "Discard  x")))
		for _, l := range ui.Wrap("Applying rebuilds the system and asks for your password.", w) {
			lines = append(lines, gap(), ui.T(pt.Mu, l))
			break
		}
	}
	return lines
}

func (p *Profiles) Hints(env *core.Env) []ui.Hint {
	h := []ui.Hint{{Key: "↑↓", Action: "move"}, {Key: "space", Action: "toggle"}}
	if on, off := p.pending(env); len(on)+len(off) > 0 {
		h = append(h, ui.Hint{Key: "a", Action: "apply"}, ui.Hint{Key: "x", Action: "discard"})
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
	p.list.move(dy*2, len(env.Data.Profiles), max(p.rows, 1))
	return nil
}

var _ = strings.TrimSpace
