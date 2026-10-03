// Package screens son las pantallas de Maxor: Inicio, Tienda, Temas, Actualizar,
// Doctor y el asistente de instalación.
package screens

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Un bloque de texto con título en mayúsculas, como las cabeceras de la CLI.
func heading(env *core.Env, text string) ui.Line { return ui.T(env.P.Mu, strings.ToUpper(text)) }

func gap() ui.Line { return ui.Blank() }

func plain(env *core.Env, text string) ui.Line { return ui.T(env.P.Text, text) }

func muted(env *core.Env, text string) ui.Line { return ui.T(env.P.Mu, text) }

// listState lleva la selección de una lista y su desplazamiento.
type listState struct{ sel, top int }

// move mueve la selección dentro de n elementos y mantiene visible la fila.
func (l *listState) move(d, n, rows int) {
	if n <= 0 {
		l.sel, l.top = 0, 0
		return
	}
	l.sel += d
	if l.sel < 0 {
		l.sel = 0
	}
	if l.sel > n-1 {
		l.sel = n - 1
	}
	l.fit(n, rows)
}

// fit ajusta el desplazamiento para que la selección se vea en rows filas.
func (l *listState) fit(n, rows int) {
	if rows < 1 {
		rows = 1
	}
	if l.sel < l.top {
		l.top = l.sel
	}
	if l.sel >= l.top+rows {
		l.top = l.sel - rows + 1
	}
	if l.top > n-rows {
		l.top = n - rows
	}
	if l.top < 0 {
		l.top = 0
	}
}

// window devuelve el rango de elementos visibles.
func (l *listState) window(n, rows int) (from, to int) {
	l.fit(n, rows)
	from = l.top
	to = from + rows
	if to > n {
		to = n
	}
	return
}

// isKey compara una tecla con varias formas de escribirla.
func isKey(msg tea.KeyMsg, keys ...string) bool {
	s := msg.String()
	for _, k := range keys {
		if s == k {
			return true
		}
	}
	return false
}

// listKey interpreta las teclas de movimiento (flechas y vim). Devuelve el
// desplazamiento pedido y si la tecla era de movimiento.
func listKey(msg tea.KeyMsg) (int, bool) {
	switch msg.String() {
	case "up", "k":
		return -1, true
	case "down", "j":
		return 1, true
	case "pgup", "ctrl+u":
		return -8, true
	case "pgdown", "ctrl+d":
		return 8, true
	case "home", "g":
		return -1 << 20, true
	case "end", "G":
		return 1 << 20, true
	}
	return 0, false
}

// working es la línea de una tarea en marcha: el mismo spinner y los mismos
// segundos que en la CLI. Devuelve ok=false si no hay tarea o aún es muy pronto.
func working(env *core.Env, id, label string) (ui.Line, bool) {
	if !env.Tasks.Loading(id) {
		return ui.Line{}, false
	}
	segs := []ui.Seg{ui.S(env.P.Ac, ui.Spin(env.Frame)+"  "), ui.S(env.P.Text, label)}
	if d := ui.Duration(env.Tasks.Elapsed(id), 3*1e9); d != "" {
		segs = append(segs, ui.S(env.P.Mu, "  "+d))
	}
	return ui.Of(segs...), true
}

// skeletonRows son n filas de huecos con los anchos dados, hasta que lleguen los datos.
func skeletonRows(env *core.Env, n int, widths ...int) []ui.Line {
	out := make([]ui.Line, 0, n)
	for i := 0; i < n; i++ {
		ws := make([]int, len(widths))
		for j, w := range widths {
			ws[j] = w + (i*3+j*5)%5
		}
		out = append(out, ui.Skeleton(env.P, env.Frame+i, ws...))
	}
	return out
}

// failed es la línea de un fallo, con el mismo glifo y la pista de siempre.
func failed(env *core.Env, what string, err error) []ui.Line {
	return []ui.Line{
		ui.Of(ui.S(env.P.Bad, ui.G.Bad+"  "), ui.S(env.P.Text, what)),
		ui.T(env.P.Mu, "   "+oneLine(err.Error())),
		ui.T(env.P.Mu, "   maxor logs --last"),
	}
}

func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return s
}

// own dice si un resultado de tarea es de esta pantalla.
func own(id string, msg tea.Msg) (task.DoneMsg, bool) {
	d, ok := msg.(task.DoneMsg)
	if !ok || d.Owner() != id {
		return d, false
	}
	return d, true
}

// button dibuja una acción como las de la pantalla de ejemplo: [ Acción  tecla ].
func button(env *core.Env, primary bool, label string) ui.Seg {
	if primary {
		return ui.S(env.P.Btn, " "+label+" ")
	}
	return ui.S(env.P.Btn2, " "+label+" ")
}

func space(n int) ui.Seg { return ui.Seg{T: strings.Repeat(" ", n)} }

// plural devuelve «1 cosa» o «2 cosas».
func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}
