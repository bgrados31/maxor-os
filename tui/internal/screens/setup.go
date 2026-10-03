package screens

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/maxor"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Setup es el asistente de primer arranque (maxor setup): detecta el equipo, deja
// elegir el aspecto y los perfiles, y lo aplica. Las decisiones se guardan con
// los mismos comandos de la CLI; reconstruir el sistema se ofrece al final y
// necesita sudo, así que cede la terminal.
type Setup struct {
	core.Base
	step   int // 0 bienvenida · 1 aspecto · 2 perfiles · 3 revisión · 4 hecho
	look   listState
	prof   listState
	chosen map[string]bool
	look0  string // tema activo al empezar
	inited bool
	pinit  bool
	// resultado de aplicar
	results []string
	changed bool // se guardó algún perfil: queda pendiente reconstruir
	failed  bool
	rows    int
}

func NewSetup() *Setup { return &Setup{chosen: map[string]bool{}} }

func (s *Setup) ID() string    { return "setup" }
func (s *Setup) Title() string { return "Setup" }

var setupSteps = []string{"Welcome", "Look", "Profiles", "Review", "Done"}

func (s *Setup) Init(env *core.Env) tea.Cmd {
	return tea.Batch(LoadHardware(env, false), LoadThemes(env, false), LoadProfiles(env, false))
}

func (s *Setup) Captures() bool { return false }

func (s *Setup) settle(env *core.Env) {
	if !s.inited && env.Data.ThemesLoaded {
		s.inited = true
		for i, th := range ordered(env) {
			if th.Active {
				s.look.sel = i
				s.look0 = th.ID
			}
		}
	}
	if !s.pinit && len(env.Data.Profiles) > 0 {
		s.pinit = true
		for _, p := range env.Data.Profiles {
			if p.Enabled {
				s.chosen[p.ID] = true
			}
		}
	}
}

func (s *Setup) lookID(env *core.Env) string {
	l := ordered(env)
	if s.look.sel >= 0 && s.look.sel < len(l) {
		return l[s.look.sel].ID
	}
	return ""
}

func (s *Setup) preview(env *core.Env) tea.Cmd {
	id := s.lookID(env)
	if id == "" {
		return nil
	}
	return func() tea.Msg { return core.PreviewThemeMsg{ID: id} }
}

// apply guarda las decisiones con los comandos de la CLI y devuelve qué hizo.
func (s *Setup) apply(env *core.Env) tea.Cmd {
	look, look0 := s.lookID(env), s.look0
	want := map[string]bool{}
	for k, v := range s.chosen {
		want[k] = v
	}
	profiles := env.Data.Profiles
	return env.Tasks.Start(task.Task{ID: "setup.apply", Label: "Saving your choices", Run: func(ctx context.Context) (any, error) {
		var done []string
		if look != "" && look != look0 {
			if err := env.Client.ApplyTheme(ctx, look); err != nil {
				return done, fmt.Errorf("theme %s: %w", look, err)
			}
			done = append(done, "theme:"+look)
		}
		for _, p := range profiles {
			if want[p.ID] != p.Enabled {
				if err := env.Client.SetProfile(ctx, p.ID, want[p.ID]); err != nil {
					return done, fmt.Errorf("profile %s: %w", p.ID, err)
				}
				verb := "profile+:"
				if !want[p.ID] {
					verb = "profile-:"
				}
				done = append(done, verb+p.ID)
			}
		}
		return done, nil
	}})
}

func (s *Setup) Update(env *core.Env, msg tea.Msg) (core.Screen, tea.Cmd) {
	s.settle(env)
	switch m := msg.(type) {
	case task.DoneMsg:
		if m.ID != "setup.apply" {
			return s, nil
		}
		done, _ := m.Value.([]string)
		s.results = nil
		var cmds []tea.Cmd
		for _, d := range done {
			kind, id, _ := strings.Cut(d, ":")
			switch kind {
			case "theme":
				s.results = append(s.results, "Applied theme "+id)
				cmds = append(cmds, core.Note("ok", "Applied theme "+id), func() tea.Msg { return core.ThemeChangedMsg{} })
			case "profile+":
				s.changed = true
				s.results = append(s.results, "Enabled profile "+id)
			case "profile-":
				s.changed = true
				s.results = append(s.results, "Disabled profile "+id)
			}
		}
		if s.changed {
			cmds = append(cmds, core.Note("info", "Profiles saved: apply them with maxor update --no-lock"))
		}
		if m.Err != nil {
			s.failed = true
			s.results = append(s.results, "Stopped: "+oneLine(m.Err.Error()))
		}
		s.step = 4
		cmds = append(cmds, LoadThemes(env, true), LoadProfiles(env, true))
		return s, tea.Batch(cmds...)
	case core.ExecDoneMsg:
		if m.Tag != "setup" {
			return s, nil
		}
		if m.Err != nil {
			return s, core.Toast("bad", "The build did not finish: maxor logs --last")
		}
		s.changed = false
		return s, tea.Batch(core.Note("ok", "Built and applied the system"), core.Quit())
	case tea.KeyMsg:
		return s.key(env, m)
	}
	return s, nil
}

func (s *Setup) back() {
	if s.step > 0 && s.step < 4 {
		s.step--
	}
}

func (s *Setup) key(env *core.Env, m tea.KeyMsg) (core.Screen, tea.Cmd) {
	switch s.step {
	case 0:
		if isKey(m, "enter", "right", "l") {
			s.step = 1
			return s, s.preview(env)
		}
	case 1:
		if d, mv := listKey(m); mv {
			s.look.move(d, len(ordered(env)), max(s.rows, 1))
			return s, s.preview(env)
		}
		switch {
		case isKey(m, "enter", "right", "l"):
			s.step = 2
		case isKey(m, "esc", "backspace", "left", "h"):
			s.back()
			return s, func() tea.Msg { return core.PreviewThemeMsg{} }
		}
	case 2:
		if d, mv := listKey(m); mv {
			s.prof.move(d, len(env.Data.Profiles), max(s.rows, 1))
			return s, nil
		}
		switch {
		case isKey(m, " "):
			if ps := env.Data.Profiles; s.prof.sel < len(ps) {
				id := ps[s.prof.sel].ID
				s.chosen[id] = !s.chosen[id]
			}
		case isKey(m, "enter", "right", "l"):
			s.step = 3
		case isKey(m, "esc", "backspace", "left", "h"):
			s.back()
		}
	case 3:
		switch {
		case isKey(m, "enter"):
			return s, s.apply(env)
		case isKey(m, "esc", "backspace", "left", "h"):
			s.back()
		}
	case 4:
		switch {
		case isKey(m, "b") && s.changed:
			return s, tea.ExecProcess(applyCmd(), func(err error) tea.Msg { return core.ExecDoneMsg{Tag: "setup", Err: err} })
		case isKey(m, "enter", "q"):
			return s, core.Quit()
		}
	}
	return s, nil
}

func (s *Setup) dots(env *core.Env) ui.Line {
	p := env.P
	var segs []ui.Seg
	for i := range setupSteps {
		switch {
		case i < s.step:
			segs = append(segs, ui.S(p.Ok, ui.G.Dot+" "))
		case i == s.step:
			segs = append(segs, ui.S(p.Ac, ui.G.Dot+" "))
		default:
			segs = append(segs, ui.S(p.Mu, ui.G.Info+" "))
		}
	}
	segs = append(segs, ui.S(p.Mu, fmt.Sprintf(" %d of %d · ", s.step+1, len(setupSteps))), ui.S(p.Text, setupSteps[s.step]))
	return ui.Of(segs...)
}

func (s *Setup) Main(env *core.Env, w, h int) []ui.Line {
	s.settle(env)
	p := env.P
	s.settle(env)
	s.rows = h - 8
	lines := []ui.Line{s.dots(env), gap()}
	switch s.step {
	case 0:
		lines = append(lines, ui.T(p.Bold, "Welcome to Maxor OS"), muted(env, "Let's set it up for this computer."), gap(), heading(env, "Detected"))
		hw := env.Data.Hardware
		if hw == nil {
			if err := env.Data.Err["hardware"]; err != nil {
				lines = append(lines, muted(env, "Could not detect the hardware: "+oneLine(err.Error())))
			} else {
				lines = append(lines, skeletonRows(env, 3, 16, 12)...)
			}
		} else {
			kind := "desktop"
			if hw.Laptop {
				kind = "laptop"
			}
			if hw.Virt != "none" && hw.Virt != "" {
				kind += " · virtual machine (" + hw.Virt + ")"
			}
			lines = append(lines, plain(env, hw.CPU.Model), muted(env, kind))
			for _, g := range hw.GPUs {
				lines = append(lines, plain(env, g.Vendor+" graphics "+g.ID))
			}
			if hybrid(hw) {
				lines = append(lines, ui.T(p.Ok, ui.G.Tick+" hybrid graphics: the integrated GPU for the desktop, NVIDIA on demand"))
			}
		}
		lines = append(lines, gap(), ui.Of(button(env, true, "Continue  ⏎")))
	case 1:
		lines = append(lines, ui.T(p.Bold, "Pick a look"), muted(env, "Moving over a look previews it right here."), gap())
		if !env.Data.ThemesLoaded {
			return append(lines, skeletonRows(env, 5, 14, 10)...)
		}
		list := ordered(env)
		from, to := s.look.window(len(list), s.rows)
		for i := from; i < to; i++ {
			th := list[i]
			sw := func(c string) ui.Seg { return ui.S(p.Fill.Foreground(lipgloss.Color(c)), "██") }
			lines = append(lines, ui.Line{
				L:   []ui.Seg{ui.S(p.Text, fmt.Sprintf("%-14s", th.ID)), sw(th.Colors.S2), sw(th.Colors.Ac), sw(th.Colors.Ac2), sw(th.Colors.Fg)},
				R:   []ui.Seg{ui.S(p.Mu, th.Name+" ")},
				Sel: i == s.look.sel,
			})
		}
	case 2:
		lines = append(lines, ui.T(p.Bold, "What will you use it for?"), muted(env, "Pick any. You can change this later."), gap())
		ps := env.Data.Profiles
		if len(ps) == 0 {
			return append(lines, skeletonRows(env, 4, 10, 30)...)
		}
		from, to := s.prof.window(len(ps), s.rows)
		for i := from; i < to; i++ {
			pr := ps[i]
			mark := ui.S(p.Mu, ui.G.Off+" ")
			if s.chosen[pr.ID] {
				mark = ui.S(p.Ac, ui.G.On+" ")
			}
			lines = append(lines, ui.Line{L: []ui.Seg{mark, ui.S(p.Text, pr.Title)}, R: []ui.Seg{ui.S(p.Mu, firstSentence(pr.Description)+" ")}, Sel: i == s.prof.sel})
		}
	case 3:
		lines = append(lines, ui.T(p.Bold, "Review"), gap())
		look := s.lookID(env)
		lines = append(lines, ui.Of(ui.S(p.Mu, "look      "), ui.S(p.Text, look)))
		var names []string
		for _, pr := range env.Data.Profiles {
			if s.chosen[pr.ID] {
				names = append(names, pr.ID)
			}
		}
		pl := "none"
		if len(names) > 0 {
			pl = strings.Join(names, ", ")
		}
		lines = append(lines, ui.Of(ui.S(p.Mu, "profiles  "), ui.S(p.Text, pl)), gap(),
			muted(env, "Your choices are saved now. Building the system with the"), muted(env, "new profiles happens next, and needs your password."), gap())
		if l, ok := working(env, "setup.apply", "Saving your choices"); ok {
			lines = append(lines, l)
		} else {
			lines = append(lines, ui.Of(button(env, true, "Save  ⏎"), space(1), button(env, false, "Back  esc")))
		}
	case 4:
		if s.failed {
			lines = append(lines, ui.T(p.Warn.Bold(true), ui.G.Warn+"  Some choices were not saved"), gap())
		} else {
			lines = append(lines, ui.T(p.Ok.Bold(true), ui.G.Tick+"  All saved"), gap())
		}
		if len(s.results) == 0 {
			lines = append(lines, muted(env, "Nothing needed to change."))
		}
		for _, r := range s.results {
			lines = append(lines, ui.Of(ui.S(p.Ok, ui.G.OK+"  "), ui.S(p.Text, r)))
		}
		lines = append(lines, gap())
		if s.changed {
			lines = append(lines, ui.Of(button(env, true, "Build the system now  b"), space(1), button(env, false, "Later  ⏎")))
		} else {
			lines = append(lines, ui.Of(button(env, true, "Finish  ⏎")))
		}
	}
	return lines
}

func hybrid(hw *maxor.Hardware) bool {
	if !hw.Laptop {
		return false
	}
	nv, other := false, false
	for _, g := range hw.GPUs {
		switch g.Vendor {
		case "nvidia":
			nv = true
		case "intel", "amd":
			other = true
		}
	}
	return nv && other
}

func firstSentence(d string) string {
	if i := strings.Index(d, ". "); i >= 0 {
		return d[:i+1]
	}
	return d
}

func (s *Setup) Hints(env *core.Env) []ui.Hint {
	switch s.step {
	case 0:
		return []ui.Hint{{Key: "⏎", Action: "continue"}, {Key: "q", Action: "quit"}}
	case 1:
		return []ui.Hint{{Key: "↑↓", Action: "preview"}, {Key: "⏎", Action: "choose"}, {Key: "esc", Action: "back"}}
	case 2:
		return []ui.Hint{{Key: "↑↓", Action: "move"}, {Key: "space", Action: "mark"}, {Key: "⏎", Action: "next"}, {Key: "esc", Action: "back"}}
	case 3:
		return []ui.Hint{{Key: "⏎", Action: "save"}, {Key: "esc", Action: "back"}}
	}
	return []ui.Hint{{Key: "⏎", Action: "finish"}}
}

func (s *Setup) Wheel(env *core.Env, dy int) tea.Cmd {
	switch s.step {
	case 1:
		s.look.move(dy*2, len(ordered(env)), max(s.rows, 1))
		return s.preview(env)
	case 2:
		s.prof.move(dy*2, len(env.Data.Profiles), max(s.rows, 1))
	}
	return nil
}
