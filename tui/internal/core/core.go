// Package core define lo que comparten la aplicación y sus pantallas: el entorno,
// la interfaz de una pantalla y los mensajes entre ambas.
package core

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/maxor"
	"github.com/bgrados31/maxor-os/tui/internal/task"
	"github.com/bgrados31/maxor-os/tui/internal/theme"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// Env es lo que una pantalla puede usar. La aplicación lo actualiza en cada vuelta.
type Env struct {
	Client *maxor.Client
	Theme  theme.Theme
	P      ui.Painter // estilos sobre el fondo de los paneles
	Tasks  *task.Manager
	Frame  int // fotograma de la animación, compartido por todo el cargador
	Now    func() time.Time
	Data   *Data
	Host   string
	Setup  bool // arrancó como asistente (maxor setup): sin pestañas
}

// Toast es un aviso breve al pie. Kind: ok, warn, bad o info.
type ToastMsg struct{ Kind, Text string }

// Summary es una línea para el resumen que se deja en tu terminal al salir.
type SummaryMsg struct{ Kind, Text string }

// GoMsg pide cambiar de pantalla. Then, si lo hay, se entrega ya en la pantalla nueva.
type GoMsg struct {
	ID   string
	Then tea.Msg
}

// ThemeChangedMsg avisa de que cambió el tema activo y hay que volver a leerlo.
type ThemeChangedMsg struct{}

// PreviewThemeMsg pinta la pantalla con otro tema sin aplicarlo al sistema. Con
// ID vacío se vuelve al tema activo.
type PreviewThemeMsg struct{ ID string }

// QuitMsg pide salir (el asistente al terminar).
type QuitMsg struct{}

// ExecDoneMsg llega cuando termina un comando que tuvo la terminal (p. ej. sudo).
type ExecDoneMsg struct {
	Tag string
	Err error
}

// Toast, Note y Go devuelven el comando que emite cada mensaje.
func Toast(kind, text string) tea.Cmd { return func() tea.Msg { return ToastMsg{Kind: kind, Text: text} } }
func Note(kind, text string) tea.Cmd  { return func() tea.Msg { return SummaryMsg{Kind: kind, Text: text} } }
func Go(id string) tea.Cmd            { return func() tea.Msg { return GoMsg{ID: id} } }

// GoThen cambia de pantalla y le entrega un mensaje (p. ej. «comprueba las actualizaciones»).
func GoThen(id string, then tea.Msg) tea.Cmd {
	return func() tea.Msg { return GoMsg{ID: id, Then: then} }
}
func Quit() tea.Cmd                   { return func() tea.Msg { return QuitMsg{} } }

// Screen es una pantalla de la aplicación (una pestaña).
type Screen interface {
	ID() string
	Title() string
	// Init se llama la primera vez que se muestra.
	Init(env *Env) tea.Cmd
	Update(env *Env, msg tea.Msg) (Screen, tea.Cmd)
	// Main y Side devuelven las filas de cada panel (ancho w, alto h). Side
	// puede devolver nil si la pantalla no tiene panel lateral.
	Main(env *Env, w, h int) []ui.Line
	Side(env *Env, w, h int) []ui.Line
	Hints(env *Env) []ui.Hint
	// Captures dice si hay un campo de texto con el foco: entonces las teclas
	// globales (q, :, ?, tab…) no se interpretan.
	Captures() bool
	// Click recibe la fila (desde 0, dentro de Main) y la columna donde se hizo clic.
	Click(env *Env, x, y int) tea.Cmd
	// Wheel recibe el desplazamiento de la rueda: -1 arriba, +1 abajo.
	Wheel(env *Env, dy int) tea.Cmd
}

// Base da valores por defecto a lo que casi ninguna pantalla necesita.
type Base struct{}

func (Base) Side(*Env, int, int) []ui.Line   { return nil }
func (Base) Captures() bool                  { return false }
func (Base) Click(*Env, int, int) tea.Cmd    { return nil }
func (Base) Wheel(*Env, int) tea.Cmd         { return nil }

// Data es la caché compartida entre pantallas: quien carga un dato lo deja aquí y
// las demás lo leen (el Inicio muestra lo que cargaron la Tienda o el Doctor).
type Data struct {
	Apps         []maxor.App
	AppsLoaded   bool
	Doctor       *maxor.Doctor
	Themes       []maxor.Theme
	ThemesLoaded bool
	Update       *maxor.UpdateCheck
	UpdateStatus *maxor.UpdateStatus
	CacheLoaded  bool // ya se leyó el escaneo guardado (haya o no)
	Hardware     *maxor.Hardware
	Profiles     []maxor.Profile
	Err          map[string]error // último fallo de cada carga: apps, doctor, themes, hardware, profiles
}

// SearchMsg lanza una búsqueda en la Tienda (viene de la paleta de comandos).
type SearchMsg struct{ Query string }
