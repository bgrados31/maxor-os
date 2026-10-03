// Package app es el programa de la pantalla completa: las pestañas, la barra de
// ayuda, la paleta de comandos y el reparto de mensajes entre las pantallas.
package app

import (
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/maxor"
	"github.com/bgrados31/maxor-os/tui/internal/screens"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/theme"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

type overlayKind int

const (
	ovNone overlayKind = iota
	ovHelp
	ovPalette
)

// Options son los ajustes de arranque.
type Options struct {
	Screen  string // pantalla inicial; «setup» abre el asistente sin pestañas
	NoMouse bool
	Version string
	Search  string // si no está vacía, abre la Tienda y lanza esta búsqueda
}

type tickMsg struct{}

const (
	minW = 64 // una ventana de cuatro en una pantalla normal ronda las 90×25
	minH = 20
)

type tabRect struct{ x0, x1 int }

// Model es el modelo raíz de Bubble Tea.
type Model struct {
	env     *core.Env
	opts    Options
	screens []core.Screen
	active  int
	inited  map[string]bool
	w, h    int
	overlay overlayKind
	pal     palette

	toast struct {
		kind, text string
		until      time.Time
	}
	summary    []core.SummaryMsg
	base       theme.Theme // tema activo del sistema
	previewing bool
	ticking    bool
	quitting   bool

	// geometría para el ratón; la deja View
	tabs       []tabRect

	bodyTop    int
	mainX0     int
	mainW      int
	setupFocus bool
	prefs      prefs
}

// New crea el modelo. El cliente se inyecta para poder probar sin la CLI real.
func New(opts Options, client *maxor.Client) *Model {
	t := theme.Load()
	env := &core.Env{
		Client: client,
		Theme:  t,
		Tasks:  task.NewManager(),
		Now:    time.Now,
		Data:   &core.Data{},
		Setup:  opts.Screen == "setup",
	}
	env.Host, _ = os.Hostname()
	env.P = ui.NewPainter(t, t.P.S)

	m := &Model{env: env, opts: opts, inited: map[string]bool{}, base: t, setupFocus: env.Setup, prefs: loadPrefs()}
	if env.Setup {
		m.screens = []core.Screen{screens.NewSetup()}
	} else {
		m.screens = []core.Screen{
			screens.NewHome(), screens.NewStore(), screens.NewThemes(),
			screens.NewUpdate(), screens.NewDoctor(), screens.NewProfiles(), screens.NewExit(),
		}
		if i := m.indexOf(opts.Screen); i >= 0 {
			m.active = i
		}
	}
	return m
}

// Summary son las líneas que se dejan en tu terminal al salir.
func (m *Model) Summary() []core.SummaryMsg { return m.summary }

func (m *Model) indexOf(id string) int {
	for i, s := range m.screens {
		if s.ID() == id {
			return i
		}
	}
	return -1
}

func (m *Model) initScreen(i int) tea.Cmd {
	id := m.screens[i].ID()
	if m.inited[id] {
		return nil
	}
	m.inited[id] = true
	return m.screens[i].Init(m.env)
}

func (m *Model) setTheme(t theme.Theme) {
	m.env.Theme = t
	m.env.P = ui.NewPainter(t, t.P.S)
}

func (m *Model) restoreTheme() {
	if m.previewing {
		m.setTheme(m.base)
		m.previewing = false
	}
}

func (m *Model) toastActive() bool { return m.env.Now().Before(m.toast.until) }

func (m *Model) setToast(kind, text string) {
	m.toast.kind, m.toast.text = kind, text
	m.toast.until = m.env.Now().Add(3500 * time.Millisecond)
}

// ensureTick mantiene la animación solo mientras haya algo que animar.
func (m *Model) ensureTick() tea.Cmd {
	if m.ticking || !(m.env.Tasks.Busy() || m.toastActive()) {
		return nil
	}
	m.ticking = true
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.initScreen(m.active), m.ensureTick()}
	if m.opts.Search != "" {
		cmds = append(cmds, core.GoThen("store", core.SearchMsg{Query: m.opts.Search}))
	}
	return tea.Batch(cmds...)
}

// toScreen entrega un mensaje a una pantalla concreta.
func (m *Model) toScreen(i int, msg tea.Msg) tea.Cmd {
	if i < 0 || i >= len(m.screens) {
		return nil
	}
	s, cmd := m.screens[i].Update(m.env, msg)
	m.screens[i] = s
	return cmd
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	add := func(c tea.Cmd) {
		if c != nil {
			cmds = append(cmds, c)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height

	case tickMsg:
		m.ticking = false
		m.env.Frame++

	case task.DoneMsg:
		if !m.env.Tasks.Done(msg) {
			break // era de una tarea ya reemplazada
		}
		if screens.ApplyData(m.env, msg) {
			add(m.toScreen(m.active, msg))
		} else {
			add(m.toScreen(m.indexOf(msg.Owner()), msg))
		}

	case core.GoMsg:
		if i := m.indexOf(msg.ID); i >= 0 {
			m.active = i
			m.restoreTheme()
			add(m.initScreen(i))
			if msg.Then != nil {
				then := msg.Then
				add(func() tea.Msg { return then })
			}
		}

	case core.ToastMsg:
		m.setToast(msg.Kind, msg.Text)

	case core.SummaryMsg:
		for _, s := range m.summary {
			if s.Text == msg.Text {
				return m, tea.Batch(cmds...)
			}
		}
		m.summary = append(m.summary, msg)

	case core.ThemeChangedMsg:
		m.base = theme.Load()
		m.previewing = false
		m.setTheme(m.base)

	case core.PreviewThemeMsg:
		if msg.ID == "" {
			m.restoreTheme()
			break
		}
		for _, th := range m.env.Data.Themes {
			if th.ID == msg.ID {
				m.setTheme(screens.ToTheme(th))
				m.previewing = true
			}
		}

	case core.QuitMsg:
		if m.blocked() {
			m.setToast("info", "Working on your system: wait until it finishes")
			break
		}
		m.quitting = true
		return m, tea.Quit

	case screens.CheckUpdateMsg:
		add(m.toScreen(m.indexOf("update"), msg))
	case screens.RunDoctorMsg:
		add(m.toScreen(m.indexOf("doctor"), msg))
	case screens.OpenAppMsg:
		add(m.toScreen(m.indexOf("store"), msg))
	case screens.FocusSearchMsg:
		add(m.toScreen(m.indexOf("store"), msg))
	case core.SearchMsg:
		add(m.toScreen(m.indexOf("store"), msg))
	case core.ExecDoneMsg:
		add(m.toScreen(m.indexOf(msg.Tag), msg))

	case tea.KeyMsg:
		return m.key(msg, cmds)

	case tea.MouseMsg:
		add(m.mouse(msg))

	default:
		add(m.toScreen(m.active, msg))
	}
	add(m.ensureTick())
	return m, tea.Batch(cmds...)
}

func (m *Model) key(k tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	add := func(c tea.Cmd) {
		if c != nil {
			cmds = append(cmds, c)
		}
	}
	s := m.screens[m.active]
	if k.String() == "ctrl+c" {
		if m.blocked() {
			return m, core.Toast("info", "Working on your system: wait until it finishes")
		}
		m.quitting = true
		return m, tea.Quit
	}
	switch m.overlay {
	case ovHelp:
		m.overlay = ovNone
		return m, tea.Batch(cmds...)
	case ovPalette:
		add(m.paletteKey(k))
		add(m.ensureTick())
		return m, tea.Batch(cmds...)
	}
	if !s.Captures() {
		switch k.String() {
		case "q":
			if m.blocked() {
				return m, core.Toast("info", "Working on your system: wait until it finishes")
			}
			m.quitting = true
			return m, tea.Quit
		case "?":
			m.overlay = ovHelp
			return m, nil
		case ":":
			m.overlay = ovPalette
			m.pal = palette{}
			return m, nil
		case "D":
			return m, m.toggleDetails()
		}
		if !m.setupFocus && len(m.screens) > 1 {
			n := len(m.screens)
			own := false
			if o, ok := s.(interface{ OwnsHorizontal() bool }); ok {
				own = o.OwnsHorizontal()
			}
			switch k.String() {
			case "right", "l", "left", "h":
				if own {
					break
				}
				fallthrough
			case "tab", "]", "shift+tab", "[":
				step := 1
				if k.String() == "left" || k.String() == "h" || k.String() == "shift+tab" || k.String() == "[" {
					step = n - 1
				}
				add(core.Go(m.screens[(m.active+step)%n].ID()))
				return m, tea.Batch(cmds...)
			}
			if r := []rune(k.String()); len(r) == 1 && r[0] >= '1' && r[0] <= '9' && int(r[0]-'1') < n {
				add(core.Go(m.screens[int(r[0]-'1')].ID()))
				return m, tea.Batch(cmds...)
			}
		}
	}
	add(m.toScreen(m.active, k))
	add(m.ensureTick())
	return m, tea.Batch(cmds...)
}

func (m *Model) mouse(msg tea.MouseMsg) tea.Cmd {
	if m.opts.NoMouse || m.overlay != ovNone {
		return nil
	}
	s := m.screens[m.active]
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return s.Wheel(m.env, -1)
	case tea.MouseButtonWheelDown:
		return s.Wheel(m.env, 1)
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return nil
		}
		if msg.Y == 0 {
			for i, r := range m.tabs {
				if msg.X >= r.x0 && msg.X < r.x1 {
					return core.Go(m.screens[i].ID())
				}
			}

			return nil
		}
		x, y := msg.X-(m.mainX0+2), msg.Y-(m.bodyTop+1)
		if x >= 0 && x < m.mainW-4 && y >= 0 {
			return s.Click(m.env, x, y)
		}
	}
	return nil
}

// toggleDetails muestra u oculta los detalles (al lado o abajo) y lo recuerda.
func (m *Model) toggleDetails() tea.Cmd {
	m.prefs.DetailsHidden = !m.prefs.DetailsHidden
	m.prefs.save()
	if m.prefs.DetailsHidden {
		return core.Toast("info", "Details hidden. Press D to show them again")
	}
	return core.Toast("info", "Details shown")
}

// blocked: alguna pantalla está cambiando el sistema y salir lo dejaría a medias.
func (m *Model) blocked() bool {
	for _, s := range m.screens {
		if b, ok := s.(interface{ Blocking() bool }); ok && b.Blocking() {
			return true
		}
	}
	return false
}
