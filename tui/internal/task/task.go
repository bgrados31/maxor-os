// Package task es el cargador único de la pantalla: toda espera (buscar, instalar,
// comprobar, aplicar un tema…) pasa por aquí y se ve igual en todas las pestañas.
//
// Reglas, las mismas que usa la CLI de bash:
//   - nada aparece antes de ShowAfter: las tareas rápidas no parpadean;
//   - una tarea nueva con el mismo ID reemplaza a la anterior y su resultado
//     viejo se ignora (buscar «bra» y luego «brave» no muestra lo de «bra»);
//   - el resultado llega como un único mensaje, con el tiempo que tardó.
package task

import (
	"context"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ShowAfter es el tiempo antes de que una tarea se muestre como «trabajando».
const ShowAfter = 150 * time.Millisecond

// Task es un trabajo asíncrono.
type Task struct {
	// ID identifica la tarea: «pantalla.nombre». El texto antes del primer punto
	// es la pantalla a la que se entrega el resultado.
	ID    string
	Label string
	Run   func(ctx context.Context) (any, error)
	// Quiet la oculta del indicador (cargas de fondo que no merecen un spinner).
	Quiet bool
}

// DoneMsg es el resultado de una tarea.
type DoneMsg struct {
	ID      string
	Value   any
	Err     error
	Elapsed time.Duration
	gen     int
}

// Owner es la pantalla a la que pertenece el resultado.
func (m DoneMsg) Owner() string { return Owner(m.ID) }

// Owner devuelve la parte de un ID anterior al primer punto.
func Owner(id string) string {
	if i := strings.IndexByte(id, '.'); i >= 0 {
		return id[:i]
	}
	return id
}

type run struct {
	label  string
	start  time.Time
	cancel context.CancelFunc
	quiet  bool
	gen    int
}

// Active es una tarea en marcha, para dibujar el indicador.
type Active struct {
	ID      string
	Label   string
	Elapsed time.Duration
}

// Manager lleva las tareas en marcha. No es seguro entre goroutines: lo usa solo
// el bucle de Bubble Tea.
type Manager struct {
	running map[string]*run
	now     func() time.Time
	gen     int
}

// NewManager crea un gestor con el reloj real.
func NewManager() *Manager { return NewManagerWithClock(time.Now) }

// NewManagerWithClock crea un gestor con un reloj propio (pruebas).
func NewManagerWithClock(now func() time.Time) *Manager {
	return &Manager{running: map[string]*run{}, now: now}
}

// Start lanza la tarea. Si ya había una con el mismo ID, la cancela.
func (m *Manager) Start(t Task) tea.Cmd {
	if old, ok := m.running[t.ID]; ok {
		old.cancel()
	}
	m.gen++
	ctx, cancel := context.WithCancel(context.Background())
	r := &run{label: t.Label, start: m.now(), cancel: cancel, quiet: t.Quiet, gen: m.gen}
	m.running[t.ID] = r
	gen, id, start, now := r.gen, t.ID, r.start, m.now
	return func() tea.Msg {
		v, err := t.Run(ctx)
		if ctx.Err() != nil && err == nil {
			err = ctx.Err()
		}
		return DoneMsg{ID: id, Value: v, Err: err, Elapsed: now().Sub(start), gen: gen}
	}
}

// Done registra el fin de una tarea. Devuelve false si el resultado es de una
// tarea que ya fue reemplazada (hay que ignorarlo).
func (m *Manager) Done(msg DoneMsg) bool {
	r, ok := m.running[msg.ID]
	if !ok || r.gen != msg.gen {
		return false
	}
	r.cancel()
	delete(m.running, msg.ID)
	return true
}

// Cancel cancela una tarea en marcha.
func (m *Manager) Cancel(id string) {
	if r, ok := m.running[id]; ok {
		r.cancel()
		delete(m.running, id)
	}
}

// Running dice si la tarea sigue en marcha.
func (m *Manager) Running(id string) bool { _, ok := m.running[id]; return ok }

// Elapsed es el tiempo que lleva una tarea en marcha.
func (m *Manager) Elapsed(id string) time.Duration {
	if r, ok := m.running[id]; ok {
		return m.now().Sub(r.start)
	}
	return 0
}

// Busy dice si hay alguna tarea en marcha (para seguir animando).
func (m *Manager) Busy() bool { return len(m.running) > 0 }

// Active devuelve las tareas visibles: las que llevan más de ShowAfter y no son
// silenciosas, de la más antigua a la más reciente.
func (m *Manager) Active() []Active {
	now := m.now()
	var out []Active
	for id, r := range m.running {
		if r.quiet || now.Sub(r.start) < ShowAfter {
			continue
		}
		out = append(out, Active{ID: id, Label: r.label, Elapsed: now.Sub(r.start)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Elapsed != out[j].Elapsed {
			return out[i].Elapsed > out[j].Elapsed
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Loading dice si una tarea está en marcha y ya debe mostrarse como cargando.
func (m *Manager) Loading(id string) bool {
	r, ok := m.running[id]
	return ok && m.now().Sub(r.start) >= ShowAfter
}
