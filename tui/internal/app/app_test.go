package app

import (
	"time"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/maxor"
	"github.com/bgrados31/maxor-os/tui/internal/screens"
	"github.com/bgrados31/maxor-os/tui/internal/task"
)

// ── CLI de mentira ───────────────────────────────────────────────────

type fakeCLI struct {
	mu     sync.Mutex
	resp   map[string]string
	fail   map[string]string
	calls  []string
	stdins []string // «argumentos|contraseña» de cada Stream
}

// Stream hace de la CLI que va contando su salida: guarda la entrada estándar que recibió.
func (f *fakeCLI) Stream(ctx context.Context, stdin string, onLine func(string), args ...string) (int, error) {
	out, errb, code, err := f.Run(ctx, args...)
	f.mu.Lock()
	f.stdins = append(f.stdins, strings.Join(args, " ")+"|"+stdin)
	f.mu.Unlock()
	for _, l := range strings.Split(string(out)+string(errb), "\n") {
		if strings.TrimSpace(l) != "" {
			onLine(l)
		}
	}
	return code, err
}

func (f *fakeCLI) Run(_ context.Context, args ...string) ([]byte, []byte, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := strings.Join(args, " ")
	f.calls = append(f.calls, key)
	if msg, ok := f.fail[key]; ok {
		return nil, []byte(msg), 1, nil
	}
	if out, ok := f.resp[key]; ok {
		return []byte(out), nil, 0, nil
	}
	return nil, []byte("sin respuesta para " + key), 99, nil
}

// n cuenta las llamadas exactamente iguales a cmd.
func (f *fakeCLI) n(cmd string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := 0
	for _, x := range f.calls {
		if x == cmd {
			c++
		}
	}
	return c
}

func (f *fakeCLI) called(prefix string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

const themesJSON = `[
 {"id":"alba","name":"Alba","mode":"light","active":false,"colors":{"bg":"#fbeff4","s":"#fff7fa","s2":"#ffffff","fg":"#2a1622","mu":"#7d5a6c","ac":"#c2255c","ac2":"#d9480f","on":"#ffffff"}},
 {"id":"brasa","name":"Brasa","mode":"dark","active":false,"colors":{"bg":"#100c0b","s":"#1a1412","s2":"#251c19","fg":"#f6ece6","mu":"#a08c82","ac":"#ff6a3d","ac2":"#ffb347","on":"#1a0d08"}},
 {"id":"sakura","name":"Sakura nocturna","mode":"dark","active":true,"colors":{"bg":"#120b12","s":"#1d121d","s2":"#2a1a2a","fg":"#fbe9f2","mu":"#a88a9d","ac":"#ff86b8","ac2":"#ffc2a6","on":"#1b0b14"}}]`

// scanJSON es el resultado de un escaneo, hecho «hace 100 s» respecto al reloj de las pruebas.
func scanJSON(lock bool) string {
	return fmt.Sprintf(`{"up_to_date":false,"kernel":true,"counts":{"new":1,"updated":2,"removed":0,"changed":1,"config":3},"changes":[{"kind":"updated","name":"firefox","from":"149.0","to":"150.0","size":"+1 MiB"},{"kind":"updated","name":"mesa","from":"26.0.1","to":"26.0.2","size":""},{"kind":"new","name":"earlyoom","to":"1.9.0","size":"52 KiB"},{"kind":"changed","name":"maxor","size":"38 KiB"}],"checked_at":%d,"fingerprint":"fp1","lock":%v}`, testNow-100, lock)
}

// releaseJSON es el estado de las releases firmadas, verificado «hace 100 s» respecto al reloj de las pruebas.
func releaseJSON(status, reason string, available bool) string {
	return fmt.Sprintf(`{"installed":"0.1.0","status":%q,"reason":%q,"checked_at":%d,"ok_at":%d,"available":%v,"latest":"0.2.0","tag":"v0.2.0","commit":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","sequence":1000,"published":"2026-10-03T00:00:00Z","summary":"Signed releases and update notices","url":"https://example/x","stale":false}`,
		status, reason, testNow-100, testNow-100, available)
}

// testNow es el reloj fijo de las pruebas.
const testNow = 1790000600

func newCLI() *fakeCLI {
	return &fakeCLI{resp: map[string]string{
		"theme list --json": themesJSON,
		"apps --json":       `[{"source":"nix","id":"vscode","name":"vscode","version":"vscode-1.119.0"}]`,
		"doctor --json":     `{"ok":true,"fails":0,"warns":1,"groups":[{"title":"System","items":[{"level":"ok","text":"no failed system services","id":"sys_ok","fix":null,"confirm":false},{"level":"ok","text":"the running kernel is the installed one","id":"kernel_ok","fix":null,"confirm":false}]},{"title":"Configuration","items":[{"level":"warn","text":"uncommitted changes in the repository","id":"git_dirty","fix":"git -C /home/b/nixos-config status","confirm":false},{"level":"ok","text":"hardware.json matches this machine","id":"hw_ok","fix":null,"confirm":false}]}]}`,
		"update --status":   `{"flake":"/home/b/nixos-config","branch":"development","commit":"9b80dfd","dirty":true,"files":5,"fingerprint":"fp1","channel":"nixos-26.05","nixpkgs_rev":"774debe","nixpkgs_date":1789000000,"generation":28}`,
		"update --cached":   `null`,
		"hardware detect":   `{"version":1,"cpu":{"vendor":"intel","model":"13th Gen Intel(R) Core(TM) i5-13500H"},"gpus":[{"vendor":"intel","id":"8086:a7a0","bus":"PCI:0:2:0","primary":true},{"vendor":"nvidia","id":"10de:28e1","bus":"PCI:1:0:0","primary":false}],"laptop":true,"virt":"none","bluetooth":true}`,
		"profile list --json": `[{"id":"gaming","title":"Gaming","description":"Steam, Proton and GameMode. More.","includes":["Steam","Proton","GameMode","MangoHud"],"enabled":false},{"id":"office","title":"Office","description":"LibreOffice and Thunderbird.","includes":["LibreOffice","Thunderbird"],"enabled":false}]`,
		"search brave --json": `[{"source":"nix","id":"brave","name":"brave","version":"1.96.59","description":"Privacy-oriented browser"},{"source":"flatpak","id":"com.brave.Browser","name":"Brave Browser","version":"","description":"Fast Internet, AI, Adblock"}]`,
		"install --nix brave --json": `[{"id":"brave","source":"nix","ok":true}]`,
		"release status --json":      releaseJSON("ok", "", false),
		"release check --json":       releaseJSON("ok", "", false),
		"update --json":              scanJSON(true),
		"update --json --no-lock":    scanJSON(false),
		"theme apply alba":           ``,
		"theme apply brasa":          ``,
		"profile enable gaming --no-apply": ``,
	}}
}

// ── arnés ────────────────────────────────────────────────────────────

func setup(t *testing.T, opts Options) (*Model, *fakeCLI) {
	t.Helper()
	return setupWith(t, newCLI(), opts)
}

func setupWith(t *testing.T, f *fakeCLI, opts Options) (*Model, *fakeCLI) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_STATE_HOME", dir)
	m := New(opts, maxor.NewWith(f))
	m.env.Now = func() time.Time { return time.Unix(testNow, 0) }
	m.Update(tea.WindowSizeMsg{Width: 110, Height: 34})
	run(m, m.Init())
	return m, f
}

// run ejecuta un comando y entrega sus mensajes al modelo, sin esperar a los ticks.
func run(m *Model, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	switch v := msg.(type) {
	case nil:
	case tickMsg:
	case tea.BatchMsg:
		for _, c := range v {
			run(m, c)
		}
	default:
		_, next := m.Update(v)
		run(m, next)
	}
}

func send(m *Model, msgs ...tea.Msg) {
	for _, msg := range msgs {
		_, cmd := m.Update(msg)
		run(m, cmd)
	}
}

func key(s string) tea.Msg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func typeText(m *Model, s string) {
	for _, r := range s {
		send(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func view(m *Model) string { return ansi.Strip(m.View()) }

// has compara sin distinguir mayúsculas: los encabezados se dibujan en mayúsculas.
func has(out, want string) bool { return strings.Contains(strings.ToLower(out), strings.ToLower(want)) }

func summaryText(m *Model) string {
	var b strings.Builder
	for _, s := range m.Summary() {
		b.WriteString(s.Text + "\n")
	}
	return b.String()
}

// ── pruebas ──────────────────────────────────────────────────────────

func TestLaVistaSiempreMideExactamenteElTerminal(t *testing.T) {
	for _, size := range [][2]int{{64, 20}, {80, 24}, {89, 25}, {100, 30}, {120, 40}, {160, 50}, {200, 60}} {
		m, _ := setup(t, Options{})
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, id := range []string{"home", "store", "themes", "update", "doctor", "profiles", "exit"} {
			send(m, core.GoMsg{ID: id})
			for _, phase := range []string{"cargando", "cargado"} {
				out := m.View()
				lines := strings.Split(out, "\n")
				if len(lines) != size[1] {
					t.Fatalf("%dx%d %s (%s): %d filas, se esperaban %d", size[0], size[1], id, phase, len(lines), size[1])
				}
				for i, l := range lines {
					if w := ansi.StringWidth(l); w != size[0] {
						t.Fatalf("%dx%d %s (%s) fila %d: ancho %d\n%q", size[0], size[1], id, phase, i, w, ansi.Strip(l))
					}
				}
				if phase == "cargando" {
					// con tareas en marcha el reloj avanza: se ven los esqueletos
					m.env.Frame += 3
				}
			}
		}
	}
}

func TestDemasiadoPequenoAvisaEnLugarDeRomperse(t *testing.T) {
	m, _ := setup(t, Options{})
	m.Update(tea.WindowSizeMsg{Width: 56, Height: 18})
	out := view(m)
	if !strings.Contains(out, "needs at least 64×20") || len(strings.Split(out, "\n")) != 18 {
		t.Fatalf("aviso de tamaño:\n%s", out)
	}
}

func TestPestanasConTabNumerosYRaton(t *testing.T) {
	m, _ := setup(t, Options{})
	if m.screens[m.active].ID() != "home" {
		t.Fatal("arranca en Home")
	}
	send(m, key("tab"))
	if m.screens[m.active].ID() != "store" {
		t.Fatalf("tab: %s", m.screens[m.active].ID())
	}
	send(m, key("shift+tab"), key("shift+tab"))
	if m.screens[m.active].ID() != "exit" {
		t.Fatalf("shift+tab da la vuelta: %s", m.screens[m.active].ID())
	}
	send(m, key("3"))
	if m.screens[m.active].ID() != "themes" {
		t.Fatalf("tecla 3: %s", m.screens[m.active].ID())
	}
	view(m) // deja calculadas las pestañas para el ratón
	r := m.tabs[4]
	send(m, tea.MouseMsg{X: r.x0 + 1, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if m.screens[m.active].ID() != "doctor" {
		t.Fatalf("clic en la pestaña: %s", m.screens[m.active].ID())
	}
}

func TestAyudaYSalir(t *testing.T) {
	m, _ := setup(t, Options{})
	send(m, key("?"))
	if !strings.Contains(view(m), "back to your terminal") {
		t.Fatal("la ayuda debe abrirse con ?")
	}
	send(m, key("x"))
	if m.overlay != ovNone {
		t.Fatal("cualquier tecla cierra la ayuda")
	}
	send(m, key("q"))
	if !m.quitting {
		t.Fatal("q sale")
	}
	m2, _ := setup(t, Options{})
	send(m2, key("ctrl+c"))
	if !m2.quitting {
		t.Fatal("ctrl+c sale")
	}
}

func TestStoreBuscaMarcaEInstalaYDejaResumen(t *testing.T) {
	m, f := setup(t, Options{Screen: "store"})
	send(m, key("/"))
	if !m.screens[m.active].Captures() {
		t.Fatal("con el foco en el campo, las teclas globales no se interpretan")
	}
	typeText(m, "q") // «q» es texto aquí, no salir
	if m.quitting {
		t.Fatal("q dentro del campo no debe salir")
	}
	send(m, key("esc"))
	send(m, key("/"))
	// borra la q y escribe la búsqueda
	send(m, tea.KeyMsg{Type: tea.KeyBackspace})
	typeText(m, "brave")
	send(m, key("enter"))
	out := view(m)
	if !has(out, "Results 2") || !strings.Contains(out, "Brave Browser") || !strings.Contains(out, "Privacy-oriented browser") {
		t.Fatalf("resultados:\n%s", out)
	}
	send(m, key("space"), key("enter"))
	if !f.called("install --nix brave") {
		t.Fatalf("debe instalar con el origen del resultado: %v", f.calls)
	}
	if !strings.Contains(summaryText(m), "Installed brave") {
		t.Fatalf("resumen: %q", summaryText(m))
	}
	if !f.called("apps --json") {
		t.Fatal("tras instalar se vuelve a leer lo instalado")
	}
}

func TestStoreMuestraLoInstaladoSinBusqueda(t *testing.T) {
	m, _ := setup(t, Options{Screen: "store"})
	out := view(m)
	if !has(out, "Installed 1") || !has(out, "vscode") || !has(out, "1.119.0") {
		t.Fatalf("instaladas:\n%s", out)
	}
}

func TestStoreUnaBusquedaNuevaReemplazaALaAnterior(t *testing.T) {
	m, f := setup(t, Options{Screen: "store"})
	f.resp["search bra --json"] = `[{"source":"nix","id":"bra","name":"bra","version":"1","description":"viejo"}]`
	send(m, key("/"))
	typeText(m, "bra")
	_, c1 := m.Update(key("enter"))
	typeText(m, "")
	send(m, key("/"))
	typeText(m, "ve")
	_, c2 := m.Update(key("enter"))
	// llega primero lo nuevo y después lo viejo: lo viejo se ignora
	run(m, c2)
	run(m, c1)
	if out := view(m); strings.Contains(out, "viejo") || !strings.Contains(out, "Brave Browser") {
		t.Fatalf("debe verse solo la última búsqueda:\n%s", out)
	}
}

func TestPaletaAplicaUnTemaYNavega(t *testing.T) {
	m, f := setup(t, Options{})
	send(m, key(":"))
	if m.overlay != ovPalette {
		t.Fatal("«:» abre la paleta")
	}
	typeText(m, "go store")
	send(m, key("enter"))
	if m.screens[m.active].ID() != "store" || m.overlay != ovNone {
		t.Fatal("la paleta debe navegar")
	}
	send(m, key(":"))
	typeText(m, "theme alba")
	send(m, key("enter"))
	if !f.called("theme apply alba") {
		t.Fatalf("debe aplicar el tema: %v", f.calls)
	}
	if !strings.Contains(summaryText(m), "Applied theme alba") {
		t.Fatalf("resumen: %q", summaryText(m))
	}
}

func TestArrancarConBusquedaAbreLaTiendaYBusca(t *testing.T) {
	m, f := setup(t, Options{Search: "brave"})
	if m.screens[m.active].ID() != "store" || !f.called("search brave --json") {
		t.Fatalf("--search debe abrir la Tienda y buscar: %v", f.calls)
	}
	if !has(view(m), "Results 2") {
		t.Fatalf("resultados:\n%s", view(m))
	}
}

func TestPaletaBuscaEnLaTienda(t *testing.T) {
	m, f := setup(t, Options{})
	send(m, key(":"))
	typeText(m, "search brave")
	send(m, key("enter"))
	if m.screens[m.active].ID() != "store" || !f.called("search brave --json") {
		t.Fatalf("la paleta debe ir a la tienda y buscar: %v", f.calls)
	}
}

func TestPaletaSinResultadosDaPistas(t *testing.T) {
	m, _ := setup(t, Options{})
	send(m, key(":"))
	typeText(m, "zzzz")
	if !strings.Contains(view(m), "No matches") {
		t.Fatal("sin coincidencias debe sugerir qué escribir")
	}
	send(m, key("esc"))
	if m.overlay != ovNone {
		t.Fatal("esc cierra la paleta")
	}
}

func TestTemasVistaPreviaYVuelta(t *testing.T) {
	m, _ := setup(t, Options{Screen: "themes"})
	base := m.env.Theme.Name
	send(m, key("up")) // de sakura (activo) al anterior en el orden
	if m.env.Theme.Name == base || !m.previewing {
		t.Fatalf("mover debe previsualizar otro tema: %s", m.env.Theme.Name)
	}
	prev := m.env.Theme.Name
	send(m, key("esc"))
	if m.env.Theme.Name != base || m.previewing {
		t.Fatalf("esc debe volver a %s, hay %s (antes %s)", base, m.env.Theme.Name, prev)
	}
	send(m, key("up"), key("tab"))
	if m.env.Theme.Name != base {
		t.Fatal("al cambiar de pestaña vuelve el tema activo")
	}
}

func TestDoctorMuestraGruposYOfreceElArreglo(t *testing.T) {
	m, f := setup(t, Options{Screen: "doctor"})
	out := view(m)
	for _, want := range []string{"SYSTEM", "CONFIGURATION", "uncommitted changes", "works, with 1 warning", "fix ⏎"} {
		if !strings.Contains(out, want) {
			t.Fatalf("falta %q:\n%s", want, out)
		}
	}
	// la selección empieza en lo que necesita atención, con su arreglo a la vista
	if !strings.Contains(out, "Needs attention") || !strings.Contains(out, "$ git -C /home/b/nixos-config") || !strings.Contains(out, "Prepare fix") {
		t.Fatalf("detalle y arreglo:\n%s", out)
	}
	send(m, key("enter"))
	if !has(view(m), "Run it") {
		t.Fatal("el primer Intro prepara el arreglo y pide un segundo")
	}
	send(m, key("down"))
	if !strings.Contains(view(m), "Passing") || has(view(m), "Run it") {
		t.Fatal("moverse cancela lo preparado")
	}
	send(m, key("enter"))
	if !has(view(m), "nothing to fix") {
		t.Fatal("Intro en algo que pasa lo dice")
	}
	send(m, key("up"), key("enter"))
	_, cmd := m.Update(key("enter"))
	if cmd == nil {
		t.Fatal("el segundo Intro debe ceder la terminal al arreglo")
	}
	before := f.n("doctor --json")
	send(m, core.ExecDoneMsg{Tag: "doctor"})
	if f.n("doctor --json") != before+1 {
		t.Fatal("al volver del arreglo se comprueba otra vez")
	}
}

func TestUpdateEscaneaSolaAlAbrirYMuestraLaRama(t *testing.T) {
	m, f := setup(t, Options{Screen: "update"})
	if f.n("update --json --no-lock") != 1 {
		t.Fatalf("sin escaneo guardado, escanea sola: %v", f.calls)
	}
	out := view(m)
	for _, want := range []string{"development", "5 uncommitted files", "nixos-26.05", "774debe", "generation 28", "scanned 2 min ago", "pending changes", "1 new", "2 updated", "includes a new kernel", "firefox", "149.0 → 150.0", "3 configuration files", "Rescan"} {
		if !has(out, want) {
			t.Fatalf("falta %q:\n%s", want, out)
		}
	}
	send(m, key("c"))
	if f.n("update --json") != 1 || !has(view(m), "with fresh inputs") {
		t.Fatal("c comprueba con las entradas nuevas")
	}
	send(m, key("r"))
	if f.n("update --json") != 2 {
		t.Fatal("r repite el escaneo con el mismo modo que el último")
	}
}

func TestUpdateUsaElEscaneoGuardadoSiSigueVigente(t *testing.T) {
	f := newCLI()
	f.resp["update --cached"] = scanJSON(false)
	m, _ := setupWith(t, f, Options{Screen: "update"})
	if f.called("update --json") {
		t.Fatalf("un escaneo guardado y vigente no se repite: %v", f.calls)
	}
	if out := view(m); !has(out, "scanned 2 min ago") || has(out, "out of date") {
		t.Fatalf("debe mostrar el guardado:\n%s", out)
	}
}

func TestUpdateRepiteElEscaneoSiCambioElRepositorioOEsViejo(t *testing.T) {
	f := newCLI()
	f.resp["update --cached"] = strings.Replace(scanJSON(false), `"fingerprint":"fp1"`, `"fingerprint":"otra"`, 1)
	setupWith(t, f, Options{Screen: "update"})
	if f.n("update --json --no-lock") != 1 {
		t.Fatal("otra huella: el escaneo guardado ya no vale")
	}
	f2 := newCLI()
	f2.resp["update --cached"] = strings.Replace(scanJSON(false), fmt.Sprintf(`"checked_at":%d`, testNow-100), fmt.Sprintf(`"checked_at":%d`, testNow-7*3600), 1)
	setupWith(t, f2, Options{Screen: "update"})
	if f2.n("update --json --no-lock") != 1 {
		t.Fatal("con más de 6 horas se repite")
	}
}

func TestUpdateAplicarCedeLaTerminal(t *testing.T) {
	m, f := setup(t, Options{Screen: "update"})
	_, cmd := m.Update(key("a"))
	if cmd == nil {
		t.Fatal("aplicar debe devolver el comando que cede la terminal")
	}
	// al volver sin error se limpia el resultado y queda en el resumen
	before := f.n("update --json --no-lock")
	send(m, core.ExecDoneMsg{Tag: "update"})
	if f.n("update --json --no-lock") != before+1 || !strings.Contains(summaryText(m), "Updated the system") {
		t.Fatalf("tras aplicar se escanea de nuevo y queda el resumen: %d/%d %q", f.n("update --json --no-lock"), before, summaryText(m))
	}
}

func TestHomeCargaTodoEnSegundoPlanoYEnlazaLasAcciones(t *testing.T) {
	m, f := setup(t, Options{})
	out := view(m)
	for _, want := range []string{"1 warning", "1 installed", "Sakura nocturna", "13th Gen Intel", "Quick actions"} {
		if !has(out, want) {
			t.Fatalf("falta %q:\n%s", want, out)
		}
	}
	for _, c := range []string{"doctor --json", "apps --json", "theme list --json", "hardware detect"} {
		if !f.called(c) {
			t.Fatalf("Home debe cargar %s", c)
		}
	}
	send(m, key("u"))
	if m.screens[m.active].ID() != "update" || !f.called("update --json") {
		t.Fatal("u va a Actualizar y comprueba")
	}
}

func TestHomeSinCliMuestraErroresSinCaerse(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_STATE_HOME", dir)
	f := &fakeCLI{resp: map[string]string{}}
	m := New(Options{}, maxor.NewWith(f))
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	run(m, m.Init())
	out := view(m)
	if !strings.Contains(out, "could not check") || !strings.Contains(out, "maxor logs --last") {
		t.Fatalf("errores:\n%s", out)
	}
	for _, id := range []string{"store", "themes", "doctor", "profiles", "exit"} {
		send(m, core.GoMsg{ID: id})
		if len(strings.Split(m.View(), "\n")) != 30 {
			t.Fatalf("%s no mide 30 filas con la CLI rota", id)
		}
	}
}

func TestSetupRecorreLosPasosYGuardaLasElecciones(t *testing.T) {
	m, f := setup(t, Options{Screen: "setup"})
	if m.screens[0].ID() != "setup" || len(m.screens) != 1 {
		t.Fatal("el asistente arranca solo, sin pestañas")
	}
	if out := view(m); !strings.Contains(out, "Welcome to Maxor OS") || !strings.Contains(out, "hybrid graphics") {
		t.Fatalf("bienvenida:\n%s", out)
	}
	send(m, key("enter")) // → aspecto
	if !strings.Contains(view(m), "Pick a look") {
		t.Fatal("paso 2")
	}
	send(m, key("up")) // previsualiza otro
	if !m.previewing {
		t.Fatal("el aspecto se previsualiza al mover")
	}
	send(m, key("enter"))
	if !strings.Contains(view(m), "What will you use it for?") {
		t.Fatal("paso 3")
	}
	send(m, key("space"), key("enter")) // marca gaming
	out := view(m)
	if !strings.Contains(out, "gaming") || !strings.Contains(out, "Review") {
		t.Fatalf("revisión:\n%s", out)
	}
	send(m, key("enter"))
	if !f.called("profile enable gaming --no-apply") {
		t.Fatalf("debe guardar el perfil sin reconstruir: %v", f.calls)
	}
	if !strings.Contains(view(m), "All saved") || !strings.Contains(view(m), "Build the system now") {
		t.Fatalf("final:\n%s", view(m))
	}
	if !strings.Contains(summaryText(m), "Profiles saved") {
		t.Fatalf("resumen: %q", summaryText(m))
	}
	send(m, key("enter"))
	if !m.quitting {
		t.Fatal("Intro en el último paso termina")
	}
}

func TestSetupPuedeVolverAtras(t *testing.T) {
	m, _ := setup(t, Options{Screen: "setup"})
	send(m, key("enter"), key("esc"))
	if !strings.Contains(view(m), "Welcome to Maxor OS") || m.previewing {
		t.Fatal("esc vuelve al paso anterior y quita la vista previa")
	}
}

func TestElResumenNoRepiteLineas(t *testing.T) {
	m, _ := setup(t, Options{})
	send(m, core.SummaryMsg{Kind: "ok", Text: "Installed x"}, core.SummaryMsg{Kind: "ok", Text: "Installed x"})
	if len(m.Summary()) != 1 {
		t.Fatal("sin repeticiones")
	}
}

func TestIndicadorMuestraLaTareaLenta(t *testing.T) {
	m, _ := setup(t, Options{Screen: "doctor"})
	// una tarea lenta simulada: arranca y el reloj avanza
	now := m.env.Now()
	m.env.Now = func() time.Time { return now }
	m.env.Tasks = task.NewManagerWithClock(func() time.Time { return now })
	m.env.Tasks.Start(task.Task{ID: "x.slow", Label: "Doing something slow", Run: func(context.Context) (any, error) { return nil, nil }})
	now = now.Add(5 * time.Second)
	out := view(m)
	if !strings.Contains(out, "Doing something slow") || !strings.Contains(out, "5s") {
		t.Fatalf("el indicador debe mostrar la tarea y los segundos:\n%s", strings.Split(out, "\n")[0])
	}
}

func TestFlechasHYLCambianDePestana(t *testing.T) {
	m, _ := setup(t, Options{})
	id := func() string { return m.screens[m.active].ID() }
	send(m, key("right"))
	if id() != "store" {
		t.Fatalf("→: %s", id())
	}
	send(m, key("l"))
	if id() != "themes" {
		t.Fatalf("l: %s", id())
	}
	send(m, key("left"), key("h"))
	if id() != "home" {
		t.Fatalf("← y h: %s", id())
	}
	send(m, key("]"), key("["))
	if id() != "home" {
		t.Fatal("] y [")
	}
	send(m, key("left"))
	if id() != "exit" {
		t.Fatal("← desde la primera da la vuelta")
	}
	// dentro de un campo de texto son letras, no navegación
	send(m, core.GoMsg{ID: "store"}, key("/"), key("l"), key("h"), key("left"))
	if id() != "store" || m.screens[m.active].(interface{ Captures() bool }).Captures() == false {
		t.Fatal("con el foco en el campo, l, h y ← no cambian de pestaña")
	}
}

func TestElAsistenteNoUsaLasFlechasParaCambiarDePestana(t *testing.T) {
	m, _ := setup(t, Options{Screen: "setup"})
	send(m, key("right"), key("l"))
	if !has(view(m), "Welcome to Maxor OS") {
		t.Fatal("sin pestañas, las flechas no hacen nada en el asistente")
	}
}

func TestTemasUnSoloColorPorTemaYGrupos(t *testing.T) {
	m, _ := setup(t, Options{Screen: "themes"})
	out := view(m)
	for _, want := range []string{"DARK · 2", "LIGHT · 1", "Sakura nocturna", "Alba", "in use"} {
		if !strings.Contains(out, want) {
			t.Fatalf("falta %q:\n%s", want, out)
		}
	}
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "Brasa") && strings.Contains(l, "██") {
			t.Fatal("la lista ya no dibuja una paleta de bloques por tema")
		}
	}
	if strings.Count(out, "●") < 3 {
		t.Fatalf("cada tema lleva un punto de su color de acento:\n%s", out)
	}
}

func TestTarjetasDelInicioLlevanASuPestana(t *testing.T) {
	m, _ := setup(t, Options{})
	view(m)
	click := func(dx, dy int) tea.Msg {
		return tea.MouseMsg{X: m.mainX0 + 2 + dx, Y: m.bodyTop + 1 + dy, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	}
	want := []struct {
		dx, dy int
		id     string
	}{{2, 2, "doctor"}, {60, 3, "update"}, {2, 6, "store"}, {60, 7, "themes"}}
	for _, w := range want {
		send(m, core.GoMsg{ID: "home"})
		view(m)
		send(m, click(w.dx, w.dy))
		if m.screens[m.active].ID() != w.id {
			t.Fatalf("clic en (%d,%d) debía ir a %s, fue a %s", w.dx, w.dy, w.id, m.screens[m.active].ID())
		}
	}
}

func TestTiendaTieneAireEntreLaBusquedaYLoDemas(t *testing.T) {
	m, _ := setup(t, Options{Screen: "store"})
	lines := strings.Split(view(m), "\n")
	find := func(sub string) int {
		for i, l := range lines {
			if strings.Contains(l, sub) {
				return i
			}
		}
		return -1
	}
	box, chips := find("Search apps in nixpkgs"), find("Installed 1")
	if box < 0 || chips-box != 3 {
		t.Fatalf("la caja de búsqueda (fila %d) y las pestañas (fila %d) deben tener aire entre las dos", box, chips)
	}
	first := -1
	for i := chips + 1; i < len(lines); i++ {
		if strings.Contains(lines[i], "vscode") {
			first = i
			break
		}
	}
	if first-chips < 2 {
		t.Fatalf("la lista no puede pegarse a las pestañas (%d → %d)", chips, first)
	}
}

func TestTiendaClicsEnLaBusquedaYEnUnaFila(t *testing.T) {
	m, _ := setup(t, Options{Screen: "store"})
	send(m, key("/"))
	typeText(m, "brave")
	send(m, key("enter"))
	view(m)
	click := func(dx, dy int) tea.Msg {
		return tea.MouseMsg{X: m.mainX0 + 2 + dx, Y: m.bodyTop + 1 + dy, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	}
	send(m, click(14, storeItemRow(1)))
	if !has(view(m), "com.brave.Browser") {
		t.Fatalf("el clic en la segunda app la selecciona:\n%s", view(m))
	}
	send(m, click(3, 1))
	if !m.screens[m.active].Captures() {
		t.Fatal("el clic en la caja de búsqueda le da el foco")
	}
}

// storeItemRow es la fila (dentro de Main) de la app n de la Tienda.
func storeItemRow(n int) int { return 7 + 3*n }

func TestTiendaAlternaEntreResultadosEInstaladas(t *testing.T) {
	m, _ := setup(t, Options{Screen: "store"})
	send(m, key("/"))
	typeText(m, "brave")
	send(m, key("enter"))
	if out := view(m); !strings.Contains(out, "Brave Browser") || !has(out, "Results 2") {
		t.Fatalf("resultados:\n%s", out)
	}
	send(m, key("i"))
	if out := view(m); strings.Contains(out, "Brave Browser") || !strings.Contains(out, "vscode") {
		t.Fatalf("i enseña lo instalado:\n%s", out)
	}
	send(m, key("i"))
	if !strings.Contains(view(m), "Brave Browser") {
		t.Fatal("otra vez i vuelve a los resultados")
	}
}

func TestElSkeletonDelInicioTieneLaFormaDeLaTarjeta(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_STATE_HOME", dir)
	m := New(Options{}, maxor.NewWith(&fakeCLI{resp: map[string]string{}}))
	m.Update(tea.WindowSizeMsg{Width: 110, Height: 34})
	// sin lanzar nada: los datos aún no llegan, como durante los primeros 150 ms
	m.env.Tasks.Start(task.Task{ID: "data.doctor", Run: func(context.Context) (any, error) { return nil, nil }})
	now := m.env.Now()
	m.env.Now = func() time.Time { return now.Add(time.Second) }
	m.env.Tasks = task.NewManagerWithClock(m.env.Now)
	out := view(m)
	if !strings.Contains(out, "SYSTEM") || !strings.Contains(out, "UPDATES") || !strings.Contains(out, "THEME") {
		t.Fatalf("las tarjetas conservan su título mientras cargan:\n%s", out)
	}
	if strings.Count(out, "█") < 20 {
		t.Fatalf("y enseñan barras que brillan en vez de un hueco:\n%s", out)
	}
}

var _ = fmt.Sprint

func TestTiendaSeNavegaConFlechasEntreBusquedaPestanasYLista(t *testing.T) {
	m, _ := setup(t, Options{Screen: "store"})
	st := func() *screens.Store { return m.screens[m.active].(*screens.Store) }
	// buscar para que haya pestañas y filtros
	send(m, key("/"))
	typeText(m, "brave")
	send(m, key("enter"))
	if st().Zone() != "list" {
		t.Fatalf("tras buscar el foco queda en la lista: %s", st().Zone())
	}
	send(m, key("up"))
	if st().Zone() != "chips" {
		t.Fatalf("↑ desde la primera fila va a las pestañas: %s", st().Zone())
	}
	send(m, key("up"))
	if st().Zone() != "search" {
		t.Fatalf("↑ otra vez va a la búsqueda: %s", st().Zone())
	}
	// con el foco en la búsqueda, lo que se teclea es texto (incluida la q)
	typeText(m, "q")
	if !has(view(m), "braveq") {
		t.Fatalf("la caja debe recibir el texto:\n%s", view(m))
	}
	send(m, key("down"))
	if st().Zone() != "chips" {
		t.Fatalf("↓ baja a las pestañas: %s", st().Zone())
	}
	// ← → cambian de pestaña de la Tienda y no de pantalla
	send(m, key("right"))
	if m.screens[m.active].ID() != "store" {
		t.Fatal("→ no debe cambiar de pantalla")
	}
	if out := view(m); strings.Contains(out, "Brave Browser") || !strings.Contains(out, "vscode") {
		t.Fatalf("→ elige Installed:\n%s", out)
	}
	send(m, key("left"))
	if out := view(m); !strings.Contains(out, "Brave Browser") {
		t.Fatalf("← vuelve a Results:\n%s", out)
	}
	send(m, key("down"))
	if st().Zone() != "list" {
		t.Fatalf("↓ vuelve a la lista: %s", st().Zone())
	}
}

func TestEnLaTiendaLasLetrasDeLaCajaNoSalenDeLaPantalla(t *testing.T) {
	m, _ := setup(t, Options{Screen: "store"})
	send(m, key("/"))
	typeText(m, "hl:?")
	if m.screens[m.active].ID() != "store" || m.overlay != ovNone {
		t.Fatal("h, l, : y ? son texto mientras escribes")
	}
}

func TestPerfilesSeMarcanYSeAplicanDesdeLaPestana(t *testing.T) {
	m, f := setup(t, Options{Screen: "profiles"})
	out := view(m)
	for _, want := range []string{"Gaming", "Office", "0 of 2 active", "Steam, Proton and GameMode."} {
		if !has(out, want) {
			t.Fatalf("falta %q:\n%s", want, out)
		}
	}
	send(m, key(" "))
	out = view(m)
	if !has(out, "will enable") || !has(out, "1 change to apply") || !strings.Contains(out, "Apply  a") {
		t.Fatalf("marcar deja el cambio pendiente a la vista:\n%s", out)
	}
	if f.called("profile enable") {
		t.Fatal("marcar no toca nada hasta aplicar")
	}
	send(m, key(" "))
	if has(view(m), "will enable") {
		t.Fatal("desmarcar vuelve a lo activo")
	}
	send(m, key("a"))
	if f.called("profile enable") || f.called("profile disable") {
		t.Fatal("sin cambios, a no guarda nada")
	}
	send(m, key(" "), key("down"), key("x"))
	if has(view(m), "will enable") {
		t.Fatal("x descarta lo marcado")
	}
	send(m, key("up"), key(" "), key("a"))
	if !f.called("profile enable gaming --no-apply") {
		t.Fatalf("a guarda la elección con la CLI: %v", f.calls)
	}
	// después se cede la terminal para reconstruir; al volver se recarga
	before := f.n("profile list --json")
	send(m, core.ExecDoneMsg{Tag: "profiles"})
	if f.n("profile list --json") != before+1 || !strings.Contains(summaryText(m), "Applied the profiles") {
		t.Fatal("al volver de reconstruir se recarga y queda el resumen")
	}
}

func TestElAsistenteYaNoEsUnaPestana(t *testing.T) {
	m, _ := setup(t, Options{})
	if m.indexOf("setup") >= 0 {
		t.Fatal("Setup solo existe con `maxor setup`, no entre las pestañas")
	}
	if m.indexOf("profiles") < 0 {
		t.Fatal("Profiles ocupa su lugar")
	}
}

func TestDoctorDiferenciaArreglarDeSoloMirar(t *testing.T) {
	f := newCLI()
	f.resp["doctor --json"] = `{"ok":true,"fails":0,"warns":2,"groups":[{"title":"Configuration","items":[{"level":"warn","text":"uncommitted changes","id":"git_dirty","fix":"git status","confirm":false,"kind":"inspect"},{"level":"warn","text":"hardware changed","id":"hw_changed","fix":"maxor hardware detect --write","confirm":false,"kind":"fix"}]}]}`
	m, _ := setupWith(t, f, Options{Screen: "doctor"})
	out := view(m)
	if !strings.Contains(out, "look ⏎") || !strings.Contains(out, "fix ⏎") || !strings.Contains(out, "Show it") || !strings.Contains(out, "Only shows information") {
		t.Fatalf("mirar y arreglar se distinguen:\n%s", out)
	}
	_, cmd := m.Update(key("enter"))
	if cmd == nil {
		t.Fatal("mirar no pide confirmación: un Intro basta")
	}
	send(m, core.ExecDoneMsg{Tag: "doctor"})
	view(m)
	send(m, key("down"))
	out = view(m)
	if !strings.Contains(out, "Prepare fix") {
		t.Fatalf("arreglar pide preparar antes:\n%s", out)
	}
	if _, cmd := m.Update(key("enter")); cmd != nil {
		t.Fatal("el primer Intro de un arreglo solo lo prepara")
	}
}

func TestUnaAppSeInstalaUnaSolaVez(t *testing.T) {
	m, f := setup(t, Options{Screen: "store"})
	send(m, key("/"))
	typeText(m, "brave")
	send(m, key("enter"))
	_, first := m.Update(key("enter")) // empieza a instalar, sin dejar que termine
	if first == nil {
		t.Fatal("el primer Intro instala")
	}
	if out := view(m); !has(out, "installing…") {
		t.Fatalf("la fila dice que se está instalando:\n%s", out)
	}
	_, second := m.Update(key("enter"))
	if second != nil {
		run(m, second)
	}
	if !has(view(m), "already being installed") {
		t.Fatalf("el segundo Intro avisa en vez de repetir:\n%s", view(m))
	}
	_, rm := m.Update(key("r"))
	_ = rm
	run(m, first)
	if n := f.n("install --nix brave --json"); n != 1 {
		t.Fatalf("debe instalar una sola vez, instaló %d", n)
	}
}

func TestQuitarPreguntaPorLosDatosYSePuedeBorrarOConservar(t *testing.T) {
	f := newCLI()
	f.resp["remove vscode --json"] = `[{"id":"vscode","source":"nix","ok":true,"purged":false,"leftovers":[{"path":"/home/b/.config/vscode","bytes":5242880}]}]`
	f.resp["remove vscode --purge --json"] = `[{"id":"vscode","source":"nix","ok":true,"purged":true,"leftovers":[]}]`
	m, _ := setupWith(t, f, Options{Screen: "store"})
	send(m, key("r"), key("r"))
	out := view(m)
	if !has(out, "Their data is still here") || !strings.Contains(out, "/home/b/.config/vscode") || !strings.Contains(out, "5.0 MiB") || !strings.Contains(out, "Delete data  y") {
		t.Fatalf("tras quitar ofrece borrar los datos:\n%s", out)
	}
	if f.called("remove vscode --purge") {
		t.Fatal("sin decir que sí no se borra nada")
	}
	send(m, key("y"))
	if !f.called("remove vscode --purge --json") {
		t.Fatalf("y borra los datos con --purge: %v", f.calls)
	}
	if has(view(m), "Their data is still here") {
		t.Fatal("la pregunta desaparece al contestar")
	}
	// n los conserva y dice cómo borrarlos más tarde
	f2 := newCLI()
	f2.resp["remove vscode --json"] = f.resp["remove vscode --json"]
	m2, _ := setupWith(t, f2, Options{Screen: "store"})
	send(m2, key("r"), key("r"), key("n"))
	if f2.called("remove vscode --purge") || has(view(m2), "Their data is still here") {
		t.Fatal("n conserva los datos")
	}
}

func TestQuitarSinDatosNoPregunta(t *testing.T) {
	f := newCLI()
	f.resp["remove vscode --json"] = `[{"id":"vscode","source":"nix","ok":true,"purged":false,"leftovers":[]}]`
	m, _ := setupWith(t, f, Options{Screen: "store"})
	send(m, key("r"), key("r"))
	if has(view(m), "Their data is still here") {
		t.Fatal("sin carpetas sobrantes no hay pregunta")
	}
}

func TestInstaladasTienenMenuDeAccionesSinLetras(t *testing.T) {
	f := newCLI()
	f.resp["apps updates"] = `[{"source":"nix","id":"vscode","current":"1.119.0","latest":"1.120.0"}]`
	f.resp["remove vscode --list-data"] = `[{"path":"/home/b/.config/Code","bytes":10485760}]`
	f.resp["apps update vscode --json"] = `[{"id":"vscode","source":"nix","ok":true}]`
	f.resp["apps open vscode --json"] = `[{"id":"vscode","ok":true}]`
	f.resp["remove vscode --json"] = `[{"id":"vscode","source":"nix","ok":true,"purged":false,"leftovers":[{"path":"/home/b/.config/Code","bytes":10485760}]}]`
	f.resp["remove vscode --purge --json"] = `[{"id":"vscode","source":"nix","ok":true,"purged":true,"leftovers":[]}]`
	m, _ := setupWith(t, f, Options{Screen: "store"})
	out := view(m)
	if !strings.Contains(out, "1.120.0") || !has(out, "Installed 1 ↑1") {
		t.Fatalf("la lista avisa de la versión nueva:\n%s", out)
	}
	send(m, key("enter"))
	out = view(m)
	for _, want := range []string{"What do you want to do?", "Open", "Update", "1.119.0 → 1.120.0", "Remove", "keeps your saves", "Remove and delete its data", "frees 10.0 MiB"} {
		if !has(out, want) {
			t.Fatalf("falta %q en el menú:\n%s", want, out)
		}
	}
	// Abrir
	send(m, key("enter"))
	if !f.called("apps open vscode --json") || !has(view(m), "Opening Visual") && !has(view(m), "Opening vscode") {
		t.Fatalf("abrir llama a la CLI y lo dice: %v", f.calls)
	}
	// Actualizar
	send(m, key("enter"), key("down"), key("enter"))
	if !f.called("apps update vscode --json") {
		t.Fatalf("actualizar llama a la CLI: %v", f.calls)
	}
	// Quitar y borrar los datos pide un sí final y enseña qué se borra
	send(m, key("enter"), key("down"), key("down"), key("down"), key("enter"))
	out = view(m)
	if !has(out, "This cannot be undone") || !strings.Contains(out, "/home/b/.config/Code") || !has(out, "Yes, delete") {
		t.Fatalf("pide confirmar el borrado de datos:\n%s", out)
	}
	if f.called("remove vscode --json") {
		t.Fatal("no se quita nada antes del sí final")
	}
	send(m, key("esc"))
	if f.called("remove vscode --json") {
		t.Fatal("esc cancela")
	}
	if !has(view(m), "What do you want to do?") {
		t.Fatalf("el menú sigue abierto tras cancelar el borrado:\n%s", view(m))
	}
	send(m, key("enter"), key("enter"))
	if !f.called("remove vscode --json") || !f.called("remove vscode --purge --json") {
		t.Fatalf("quitar con datos: %v", f.calls)
	}
}

func TestMenuQuitarSinBorrarPreguntaPorLosDatos(t *testing.T) {
	f := newCLI()
	f.resp["remove vscode --list-data"] = `[]`
	f.resp["remove vscode --json"] = `[{"id":"vscode","source":"nix","ok":true,"purged":false,"leftovers":[{"path":"/home/b/.config/Code","bytes":1048576}]}]`
	m, _ := setupWith(t, f, Options{Screen: "store"})
	send(m, key("enter"), key("down"), key("enter"))
	if !f.called("remove vscode --json") || !has(view(m), "Their data is still here") {
		t.Fatalf("Remove quita y pregunta por los datos:\n%s", view(m))
	}
}

func TestElMenuNoDejaHacerDosCosasAlMismoTiempo(t *testing.T) {
	m, f := setup(t, Options{Screen: "store"})
	send(m, key("/"))
	typeText(m, "brave")
	send(m, key("enter"))
	m.Update(key("enter")) // instalando brave, sin terminar
	send(m, key("i"))
	send(m, key("enter"), key("down"), key("enter"))
	if f.called("remove vscode --json") {
		t.Fatalf("no se puede quitar mientras se instala otra app: %v", f.calls)
	}
}

func TestTemasSeparaOscurosDeClarosYEnseñaUnFastfetch(t *testing.T) {
	m, _ := setup(t, Options{Screen: "themes"})
	lines := strings.Split(view(m), "\n")
	dark, light := -1, -1
	for i, l := range lines {
		if strings.Contains(l, "DARK ·") {
			dark = i
		}
		if strings.Contains(l, "LIGHT ·") {
			light = i
		}
	}
	// oscuros (cabecera + 2 temas) y una fila de aire antes de los claros
	if dark < 0 || light-dark != 4 {
		t.Fatalf("debe haber una fila en blanco entre los grupos (dark %d, light %d)", dark, light)
	}
	out := view(m)
	for _, want := range []string{"@", "Maxor OS", "Hyprland", "Theme"} {
		if !strings.Contains(out, want) {
			t.Fatalf("falta %q en la muestra de fastfetch:\n%s", want, out)
		}
	}
}

// bulkCLI es una CLI con tres apps instaladas, una de ellas con versión nueva.
func bulkCLI() *fakeCLI {
	f := newCLI()
	f.resp["apps --json"] = `[{"source":"nix","id":"brave","name":"brave","version":"brave-1.96.59"},{"source":"nix","id":"vscode","name":"vscode","version":"vscode-1.119.0"},{"source":"nix","id":"btop","name":"btop","version":"btop-1.4.7"}]`
	f.resp["apps updates"] = `[{"source":"nix","id":"vscode","current":"1.119.0","latest":"1.120.0"},{"source":"nix","id":"btop","current":"1.4.7","latest":"1.4.8"}]`
	for _, id := range []string{"brave", "vscode", "btop"} {
		f.resp["remove "+id+" --list-data"] = `[]`
		f.resp["remove "+id+" --json"] = `[{"id":"` + id + `","source":"nix","ok":true,"purged":false,"leftovers":[{"path":"/home/b/.config/` + id + `","bytes":1048576}]}]`
		f.resp["remove "+id+" --purge --json"] = `[{"id":"` + id + `","source":"nix","ok":true,"purged":true,"leftovers":[]}]`
		f.resp["apps update "+id+" --json"] = `[{"id":"` + id + `","source":"nix","ok":true}]`
	}
	return f
}

func TestInstaladasMuestranEstadoYCasillas(t *testing.T) {
	m, _ := setupWith(t, bulkCLI(), Options{Screen: "store"})
	out := view(m)
	for _, want := range []string{"✓ up to date", "↑ 1.120.0 available", "↑ 1.4.8 available", "nixpkgs · 1.119.0 → 1.120.0", "nixpkgs · 1.96.59", "◻"} {
		if !strings.Contains(out, want) {
			t.Fatalf("falta %q:\n%s", want, out)
		}
	}
	send(m, key(" "), key("down"), key(" "))
	out = view(m)
	if strings.Count(out, "◼") < 3 || !has(out, "2 selected") { // dos casillas + el contador (y el del panel)
		t.Fatalf("las casillas se marcan y se cuentan:\n%s", out)
	}
	if !strings.Contains(out, "Update 1") || !strings.Contains(out, "Remove 2") {
		t.Fatalf("el panel ofrece las acciones en bloque:\n%s", out)
	}
	send(m, key("esc"))
	if has(view(m), "2 selected") {
		t.Fatal("esc limpia la selección")
	}
}

func TestSeleccionarTodasYActualizarLasQueTienenVersionNueva(t *testing.T) {
	f := bulkCLI()
	m, _ := setupWith(t, f, Options{Screen: "store"})
	send(m, key("a"))
	if !has(view(m), "3 selected") {
		t.Fatalf("a selecciona todas:\n%s", view(m))
	}
	send(m, key("u"))
	if !f.called("apps update vscode --json") || !f.called("apps update btop --json") || f.called("apps update brave --json") {
		t.Fatalf("u actualiza solo las marcadas que tienen versión nueva: %v", f.calls)
	}
	send(m, key("esc"), key("a"), key("a"))
	if has(view(m), "selected") {
		t.Fatal("a otra vez las desmarca")
	}
	// U actualiza todas las pendientes sin marcar nada
	f2 := bulkCLI()
	m2, _ := setupWith(t, f2, Options{Screen: "store"})
	send(m2, key("U"))
	if !f2.called("apps update vscode --json") || !f2.called("apps update btop --json") {
		t.Fatalf("U actualiza todas las pendientes: %v", f2.calls)
	}
}

func TestQuitarVariasALaVezPreguntaUnaSolaVezPorLosDatos(t *testing.T) {
	f := bulkCLI()
	m, _ := setupWith(t, f, Options{Screen: "store"})
	send(m, key(" "), key("down"), key(" "), key("r"))
	if out := view(m); !has(out, "2 apps") || !strings.Contains(out, "Remove and delete their data") {
		t.Fatalf("r abre el menú sobre las dos marcadas:\n%s", out)
	}
	send(m, key("enter")) // Remove (sin borrar datos)
	if !f.called("remove brave --json") || !f.called("remove vscode --json") || f.called("remove btop --json") {
		t.Fatalf("quita las dos marcadas, una tras otra: %v", f.calls)
	}
	out := view(m)
	if !has(out, "Their data is still here") || !strings.Contains(out, "/home/b/.config/brave") || !strings.Contains(out, "/home/b/.config/vscode") || !strings.Contains(out, "2.0 MiB") {
		t.Fatalf("una sola pregunta con todas las carpetas:\n%s", out)
	}
	send(m, key("y"))
	if !f.called("remove brave --purge --json") || !f.called("remove vscode --purge --json") {
		t.Fatalf("y borra los datos de las dos: %v", f.calls)
	}
}

func TestQuitarVariasYBorrarSusDatosPideUnSiFinal(t *testing.T) {
	f := bulkCLI()
	f.resp["remove brave --list-data"] = `[{"path":"/home/b/.config/brave","bytes":2097152}]`
	f.resp["remove vscode --list-data"] = `[{"path":"/home/b/.config/vscode","bytes":1048576}]`
	m, _ := setupWith(t, f, Options{Screen: "store"})
	send(m, key(" "), key("down"), key(" "), key("r"), key("d"))
	out := view(m)
	if !has(out, "This cannot be undone") || !strings.Contains(out, "3.0 MiB") {
		t.Fatalf("pide confirmar y suma los tamaños:\n%s", out)
	}
	if f.called("remove brave --json") {
		t.Fatal("nada se quita antes del sí final")
	}
	send(m, key("enter"))
	if !f.called("remove brave --json") || !f.called("remove vscode --json") {
		t.Fatalf("tras el sí se quitan: %v", f.calls)
	}
}

func TestClicEnLaCasillaMarcaLaApp(t *testing.T) {
	m, _ := setupWith(t, bulkCLI(), Options{Screen: "store"})
	view(m)
	click := func(dx, dy int) tea.Msg {
		return tea.MouseMsg{X: m.mainX0 + 2 + dx, Y: m.bodyTop + 1 + dy, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	}
	send(m, click(1, storeItemRow(1)))
	if !has(view(m), "1 selected") {
		t.Fatalf("un clic en la casilla marca la app:\n%s", view(m))
	}
}

func TestExitEsUnaPestanaMasQueSoloSaleConIntro(t *testing.T) {
	m, _ := setup(t, Options{})
	if first := strings.Split(view(m), "\n")[0]; !strings.Contains(first, "Profiles") || !strings.Contains(first, "Exit") {
		t.Fatalf("Exit es una pestaña más, tras Profiles: %q", first)
	}
	send(m, core.GoMsg{ID: "profiles"}, key("right"))
	if m.screens[m.active].ID() != "exit" {
		t.Fatal("→ desde Profiles llega a Exit")
	}
	out := view(m)
	if !has(out, "Leave Maxor") || !strings.Contains(out, "Nothing pending") && !has(out, "Before you go") {
		t.Fatalf("la pantalla de Exit:\n%s", out)
	}
	if m.quitting {
		t.Fatal("llegar a Exit no sale")
	}
	send(m, key("right"))
	if m.screens[m.active].ID() != "home" {
		t.Fatal("→ desde Exit vuelve a Home")
	}
	send(m, core.GoMsg{ID: "exit"})
	_, cmd := m.Update(key("enter"))
	if cmd == nil {
		t.Fatal("Intro en Exit sale")
	}
	if _, ok := cmd().(core.QuitMsg); !ok {
		t.Fatalf("Intro devuelve QuitMsg, no %T", cmd())
	}
	send(m, core.GoMsg{ID: "exit"})
	row := -1
	for i, l := range strings.Split(view(m), "\n") {
		if strings.Contains(l, "Exit  ⏎") {
			row = i
		}
	}
	click := tea.MouseMsg{X: m.mainX0 + 4, Y: row, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	if _, cmd := m.Update(click); cmd == nil {
		t.Fatalf("un clic en el botón sale:\n%s", view(m))
	}
}

func TestExitAvisaDeLoQueQuedaPendiente(t *testing.T) {
	m, _ := setupWith(t, bulkCLI(), Options{Screen: "exit"})
	if out := view(m); !has(out, "Before you go") || !has(out, "2 apps can be updated") {
		t.Fatalf("Exit enseña lo pendiente:\n%s", out)
	}
}

func TestPaletaAbreAppsYSalir(t *testing.T) {
	f := bulkCLI()
	f.resp["apps open vscode --json"] = `[{"id":"vscode","ok":true}]`
	m, _ := setupWith(t, f, Options{})
	send(m, key(":"))
	typeText(m, "open vsc")
	if !has(view(m), "open vscode") {
		t.Fatalf("la paleta ofrece abrir las apps instaladas:\n%s", view(m))
	}
	send(m, key("enter"))
	if !f.called("apps open vscode --json") {
		t.Fatalf("abre la app con la CLI: %v", f.calls)
	}
	send(m, key(":"))
	typeText(m, "exit")
	_, cmd := m.Update(key("enter"))
	if cmd == nil {
		t.Fatal("exit sale")
	}
}

func TestRYElToastDeInstalarAyudanAEncontrarLaApp(t *testing.T) {
	f := bulkCLI()
	f.resp["install --flatpak com.brave.Browser --json"] = `[{"id":"com.brave.Browser","source":"flatpak","ok":true}]`
	m, _ := setupWith(t, f, Options{Screen: "store"})
	send(m, key("R"))
	if !f.called("apps updates --refresh") {
		t.Fatalf("R vuelve a mirar las versiones nuevas sin usar lo guardado: %v", f.calls)
	}
	send(m, key("/"))
	typeText(m, "brave")
	send(m, key("enter"), key("down"), key("enter"))
	if !has(view(m), "Super+Space") {
		t.Fatalf("al instalar, el aviso dice dónde encontrar la app:\n%s", view(m))
	}
}

func TestInicioAvisaDeAppsConVersionNueva(t *testing.T) {
	m, _ := setupWith(t, bulkCLI(), Options{})
	out := view(m)
	if !strings.Contains(out, "3 installed") || !strings.Contains(out, "↑2") || !has(out, "2 updates available") {
		t.Fatalf("la tarjeta de apps avisa de las versiones nuevas:\n%s", out)
	}
}

func TestLaBarraDeAccionesRapidasCambiaConLaSeleccion(t *testing.T) {
	f := bulkCLI()
	m, _ := setupWith(t, f, Options{Screen: "store"})
	bar := func() string {
		lines := strings.Split(view(m), "\n")
		for _, l := range lines {
			if strings.Contains(l, "Check for updates") || strings.Contains(l, "Remove ") {
				return l
			}
		}
		return ""
	}
	if b := bar(); !strings.Contains(b, "Update all 2") || !strings.Contains(b, "Check for updates") {
		t.Fatalf("sin marcar: actualizar todas y buscar versiones: %q", b)
	}
	send(m, key("down"), key(" ")) // vscode, que tiene versión nueva
	if b := bar(); !strings.Contains(b, "Update 1") || !strings.Contains(b, "Remove 1") || strings.Contains(b, "Check for updates") {
		t.Fatalf("con marcadas: actualizar y quitar: %q", b)
	}
	send(m, key("esc"), key("up"), key(" ")) // brave, sin versión nueva
	if b := bar(); !strings.Contains(b, "Check for updates") || !strings.Contains(b, "Remove 1") {
		t.Fatalf("sin nada que actualizar entre lo marcado: buscar versiones y quitar: %q", b)
	}
}

func TestLosBotonesDeLaBarraSeUsanConElRaton(t *testing.T) {
	f := bulkCLI()
	m, _ := setupWith(t, f, Options{Screen: "store"})
	row := func(sub string) int {
		for i, l := range strings.Split(view(m), "\n") {
			if strings.Contains(l, sub) {
				return i
			}
		}
		return -1
	}
	y := row("Update all 2")
	if y < 0 {
		t.Fatalf("no está la barra:\n%s", view(m))
	}
	// el primer botón empieza en la columna 0 de la zona principal
	send(m, tea.MouseMsg{X: m.mainX0 + 2 + 2, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if !f.called("apps update vscode --json") || !f.called("apps update btop --json") {
		t.Fatalf("el botón «Update all» actualiza lo pendiente: %v", f.calls)
	}
}

func TestLaTiendaYaNoTieneFiltrosPorOrigen(t *testing.T) {
	m, _ := setup(t, Options{Screen: "store"})
	send(m, key("/"))
	typeText(m, "brave")
	send(m, key("enter"))
	out := view(m)
	if strings.Contains(out, "│  All") || strings.Contains(out, "All   nixpkgs") {
		t.Fatalf("sin filtros nixpkgs/All/flathub:\n%s", out)
	}
	if !strings.Contains(out, "Brave Browser") || !strings.Contains(out, "flathub") || !strings.Contains(out, "nixpkgs") {
		t.Fatalf("pero cada fila sigue diciendo de dónde viene:\n%s", out)
	}
}

func TestUpdateVuelveAUnaGeneracionAnterior(t *testing.T) {
	f := newCLI()
	f.resp["rollback --list --json"] = `[{"generation":34,"date":"2026-10-03 09:24:54","nixos":"26.05.20261002.774debe","kernel":"6.18.54","current":true},{"generation":33,"date":"2026-10-03 09:12:27","nixos":"26.05.20261002.774debe","kernel":"6.18.54","current":false},{"generation":32,"date":"2026-10-03 09:00:42","nixos":"26.05.20261002.774debe","kernel":"6.17.9","current":false}]`
	m, _ := setupWith(t, f, Options{Screen: "update"})
	if !has(view(m), "Go back  g") {
		t.Fatalf("Update ofrece volver atrás:\n%s", view(m))
	}
	send(m, key("g"))
	out := view(m)
	for _, want := range []string{"Go back to an earlier version", "#34", "#33", "#32", "running", "Generation 33", "Go back to this"} {
		if !has(out, want) {
			t.Fatalf("falta %q:\n%s", want, out)
		}
	}
	// se propone la anterior a la que corre
	if !has(out, "Generation 33") {
		t.Fatal("la selección empieza en la generación anterior")
	}
	send(m, key("down"))
	if out := view(m); !has(out, "Generation 32") || !strings.Contains(out, "6.17.9") {
		t.Fatalf("↓ elige la siguiente y enseña su kernel:\n%s", out)
	}
	send(m, key("up"), key("up"))
	if out := view(m); !has(out, "your current one") {
		t.Fatalf("la que corre no se puede elegir para volver:\n%s", out)
	}
	if _, cmd := m.Update(key("enter")); cmd != nil {
		if msg := cmd(); msg != nil {
			if _, isToast := msg.(core.ToastMsg); !isToast {
				t.Fatalf("Intro en la actual solo avisa, no cede la terminal: %T", msg)
			}
		}
	}
	send(m, key("down"))
	f.resp["rollback 33 -y"] = "switching to generation 33"
	m.env.Client.SudoCheck = func(context.Context) bool { return true }
	before := f.n("update --json --no-lock")
	send(m, key("enter"))
	if !f.called("rollback 33 -y") {
		t.Fatalf("Intro en una anterior vuelve a ella sin salir de la pantalla: %v", f.calls)
	}
	if out := view(m); !has(out, "Done in") || !strings.Contains(out, "switching to generation 33") {
		t.Fatalf("el panel enseña el resultado:\n%s", out)
	}
	send(m, key("enter"))
	if has(view(m), "Go back to an earlier") || !strings.Contains(summaryText(m), "Went back") {
		t.Fatalf("al cerrar se cierra el historial y queda el resumen: %q", summaryText(m))
	}
	if f.n("update --json --no-lock") != before+1 {
		t.Fatal("tras volver atrás se escanea otra vez")
	}
}

func TestLosDetallesSeAdaptanAlAnchoDeLaVentana(t *testing.T) {
	// ancha: al lado, sin cajón
	m, _ := setupWith(t, bulkCLI(), Options{Screen: "store"})
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	out := view(m)
	if !strings.Contains(out, "DETAILS") {
		t.Fatalf("ancha: panel al lado:\n%s", out)
	}
	// estrecha (una de cuatro en la pantalla): cajón abajo con lo esencial
	m.Update(tea.WindowSizeMsg{Width: 89, Height: 25})
	out = view(m)
	lines := strings.Split(out, "\n")
	if strings.Contains(out, "DETAILS") {
		t.Fatalf("estrecha: no hay panel al lado:\n%s", out)
	}
	if !strings.Contains(out, "Actions  ⏎") || !strings.Contains(out, "up to date") {
		t.Fatalf("estrecha: el cajón enseña lo esencial de la app elegida:\n%s", out)
	}
	if len(lines) != 25 {
		t.Fatalf("mide %d filas, no 25", len(lines))
	}
	// y sigue habiendo varias apps en la lista (cabecera compacta)
	n := 0
	for _, l := range lines {
		if strings.Contains(l, "nixpkgs ·") {
			n++
		}
	}
	if n < 3 {
		t.Fatalf("la lista compacta debe enseñar al menos 3 apps, enseña %d:\n%s", n, out)
	}
}

func TestElCajonMuestraElComandoDelDoctorYLosBotonesDeUpdate(t *testing.T) {
	m, _ := setupWith(t, bulkCLI(), Options{Screen: "doctor"})
	m.Update(tea.WindowSizeMsg{Width: 89, Height: 25})
	if out := view(m); !strings.Contains(out, "$ git -C /home/b/nixos-config status") || !strings.Contains(out, "Prepare fix") {
		t.Fatalf("el cajón del Doctor enseña el comando antes de ejecutarlo:\n%s", out)
	}
	send(m, core.GoMsg{ID: "update"})
	if out := view(m); !strings.Contains(out, "Rescan  r") || !strings.Contains(out, "Go back  g") {
		t.Fatalf("el cajón de Update enseña sus botones:\n%s", out)
	}
	send(m, core.GoMsg{ID: "themes"})
	if out := view(m); !strings.Contains(out, "Apply  ⏎") || !strings.Contains(out, "Hyprland") {
		t.Fatalf("el cajón de Themes enseña el tema:\n%s", out)
	}
}

func TestDOcultaYMuestraLosDetallesYLoRecuerda(t *testing.T) {
	m, _ := setupWith(t, bulkCLI(), Options{Screen: "store"})
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	if !strings.Contains(view(m), "DETAILS") {
		t.Fatal("al principio se ven")
	}
	send(m, key("D"))
	out := view(m)
	if strings.Contains(out, "DETAILS") || !has(out, "Details hidden") {
		t.Fatalf("D oculta los detalles y lo dice:\n%s", out)
	}
	// ya no hay cajón en una ventana estrecha
	m.Update(tea.WindowSizeMsg{Width: 89, Height: 25})
	if strings.Contains(view(m), "Actions  ⏎") {
		t.Fatal("oculto también abajo")
	}
	// se recuerda: un modelo nuevo lo lee del archivo de ajustes
	m2 := New(Options{Screen: "store"}, maxor.NewWith(bulkCLI()))
	m2.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	if strings.Contains(view(m2), "DETAILS") {
		t.Fatal("el ajuste se recuerda entre usos")
	}
	send(m, key("D"))
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	if !strings.Contains(view(m), "DETAILS") {
		t.Fatal("D otra vez los vuelve a mostrar")
	}
	// dentro de un campo de texto, D es una letra
	send(m, key("/"), key("D"))
	if !m.screens[m.active].Captures() || !strings.Contains(view(m), "DETAILS") {
		t.Fatal("con el foco en la búsqueda, D se escribe")
	}
}

func TestPerfilesEnseñanLoQueInstalan(t *testing.T) {
	m, _ := setup(t, Options{Screen: "profiles"})
	out := view(m)
	if !strings.Contains(out, "Steam · Proton · GameMode · +1 more") || !strings.Contains(out, "LibreOffice · Thunderbird") {
		t.Fatalf("la lista resume lo que lleva cada perfil:\n%s", out)
	}
	if !has(out, "Includes") || !strings.Contains(out, "+ MangoHud") && !strings.Contains(out, "MangoHud") {
		t.Fatalf("el panel lista uno a uno lo que instala:\n%s", out)
	}
	m.Update(tea.WindowSizeMsg{Width: 89, Height: 25})
	if out := view(m); !strings.Contains(out, "Gaming") || !strings.Contains(out, "Steam · Proton · GameMode · MangoHud") {
		t.Fatalf("el cajón también dice lo que trae:\n%s", out)
	}
}

func applyCLI() *fakeCLI {
	f := newCLI()
	f.resp["update --no-lock -y"] = "building the system configuration...\nactivating the configuration...\nsetting up /etc...\nreloading user units for bryan..."
	return f
}

func TestAplicarSinSalirPideLaContrasenaYEnseñaElProgreso(t *testing.T) {
	f := applyCLI()
	m, _ := setupWith(t, f, Options{Screen: "update"})
	send(m, key("a"))
	out := view(m)
	if !has(out, "Administrator password") || !has(out, "never saved or shown") {
		t.Fatalf("pide la contraseña en la propia pantalla:\n%s", out)
	}
	if f.called("update --no-lock -y") {
		t.Fatal("no se ejecuta nada hasta tener la contraseña")
	}
	typeText(m, "hunter2")
	out = view(m)
	if strings.Contains(out, "hunter2") || !strings.Contains(out, "•••••••") {
		t.Fatalf("la contraseña se ve como puntos, nunca en claro:\n%s", out)
	}
	send(m, key("enter"))
	if !f.called("update --no-lock -y") {
		t.Fatalf("con la contraseña se aplica: %v", f.calls)
	}
	if len(f.stdins) != 1 || f.stdins[0] != "update --no-lock -y|hunter2" {
		t.Fatalf("la contraseña se entrega por la entrada estándar y solo a esa orden: %v", f.stdins)
	}
	out = view(m)
	for _, want := range []string{"Done in", "Build the new system", "Switch to it", "Restart what changed", "activating the configuration", "Close"} {
		if !has(out, want) {
			t.Fatalf("falta %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "hunter2") {
		t.Fatal("la contraseña no aparece en ninguna parte")
	}
	before := f.n("update --json --no-lock")
	send(m, key("enter"))
	if has(view(m), "Administrator password") || !strings.Contains(summaryText(m), "Updated the system") {
		t.Fatalf("al cerrar vuelve a Update con el resumen: %q", summaryText(m))
	}
	if f.n("update --json --no-lock") <= before-1 {
		t.Fatal("tras aplicar se escanea otra vez")
	}
}

func TestAplicarSinPedirContrasenaSiSudoYaEstaListo(t *testing.T) {
	f := applyCLI()
	m, _ := setupWith(t, f, Options{Screen: "update"})
	m.env.Client.SudoCheck = func(context.Context) bool { return true }
	send(m, key("a"))
	if has(view(m), "Administrator password") {
		t.Fatal("sudo ya está listo: no hay que pedir nada")
	}
	if len(f.stdins) != 1 || f.stdins[0] != "update --no-lock -y|" {
		t.Fatalf("se ejecuta sin contraseña: %v", f.stdins)
	}
}

func TestUnaContrasenaIncorrectaVuelveAPedirla(t *testing.T) {
	f := applyCLI()
	f.fail = map[string]string{"update --no-lock -y": "sudo: 1 incorrect password attempt"}
	m, _ := setupWith(t, f, Options{Screen: "update"})
	send(m, key("a"))
	typeText(m, "mala")
	send(m, key("enter"))
	out := view(m)
	if !has(out, "That password did not work") || !has(out, "Administrator password") {
		t.Fatalf("avisa y vuelve a pedirla:\n%s", out)
	}
	delete(f.fail, "update --no-lock -y")
	typeText(m, "buena")
	send(m, key("enter"))
	if !has(view(m), "Done in") {
		t.Fatalf("con la buena termina:\n%s", view(m))
	}
	if len(f.stdins) != 2 || f.stdins[1] != "update --no-lock -y|buena" {
		t.Fatalf("la segunda vez entrega la contraseña nueva: %v", f.stdins)
	}
}

func TestEscCancelaAntesDeEjecutarNada(t *testing.T) {
	f := applyCLI()
	m, _ := setupWith(t, f, Options{Screen: "update"})
	send(m, key("a"), key("esc"))
	if f.called("update --no-lock -y") || has(view(m), "Administrator password") {
		t.Fatal("esc cancela sin ejecutar nada")
	}
}

func TestUnFalloSeEnseñaConSuCodigoYComoVerElRegistro(t *testing.T) {
	f := applyCLI()
	delete(f.resp, "update --no-lock -y")
	f.fail = map[string]string{}
	f.fail["update --no-lock -y"] = "error: build of /nix/store/x failed"
	m, _ := setupWith(t, f, Options{Screen: "update"})
	m.env.Client.SudoCheck = func(context.Context) bool { return true }
	send(m, key("a"))
	out := view(m)
	if !has(out, "It did not finish") || !has(out, "Exit code 1") || !has(out, "maxor logs --last") || !strings.Contains(out, "build of /nix/store/x failed") {
		t.Fatalf("el fallo se explica:\n%s", out)
	}
}

func TestMientrasSeAplicaNoSePuedeSalir(t *testing.T) {
	f := applyCLI()
	m, _ := setupWith(t, f, Options{Screen: "update"})
	m.env.Client.SudoCheck = func(context.Context) bool { return true }
	_, cmd := m.Update(key("a")) // empieza a comprobar; sin dejar que termine
	run(m, cmd)                  // la comprobación termina y arranca el trabajo, que dejamos a medias
	m.screens[m.active].(interface{ Blocking() bool }).Blocking()
	// forzamos la etapa «trabajando» sin ejecutar el trabajo
	u := m.screens[m.active].(*screens.Update)
	u.ForceRunning()
	_, cmd = m.Update(key("q"))
	if m.quitting {
		t.Fatal("q no sale mientras se cambia el sistema")
	}
	_, cmd = m.Update(key("ctrl+c"))
	if m.quitting {
		t.Fatal("ctrl+c tampoco")
	}
	_ = cmd
	u.EndRunning()
	if _, cmd := m.Update(key("q")); cmd == nil {
		t.Fatal("al terminar, q vuelve a salir")
	}
}

func TestAplicarPerfilesUsaElMismoPanel(t *testing.T) {
	f := applyCLI()
	m, _ := setupWith(t, f, Options{Screen: "profiles"})
	m.env.Client.SudoCheck = func(context.Context) bool { return true }
	send(m, key(" "), key("a"))
	if !f.called("profile enable gaming --no-apply") || !f.called("update --no-lock -y") {
		t.Fatalf("guarda y aplica sin salir de la pantalla: %v", f.calls)
	}
	if !has(view(m), "Done in") {
		t.Fatalf("enseña el panel:\n%s", view(m))
	}
	send(m, key("enter"))
	if !strings.Contains(summaryText(m), "Applied the profiles") {
		t.Fatalf("queda el resumen: %q", summaryText(m))
	}
}

func TestInicioGuardaUnaCopiaDeSeguridad(t *testing.T) {
	f := newCLI()
	f.resp["backup --json"] = `{"path":"` + os.Getenv("HOME") + `/maxor-backup-nitro-20261003-1000.tar.gz","bytes":2097152,"apps":3,"themes":1,"host":"nitro"}`
	m, _ := setupWith(t, f, Options{})
	if !has(view(m), "Back up my setup") {
		t.Fatalf("Inicio ofrece guardar una copia:\n%s", view(m))
	}
	send(m, key("b"))
	if !f.called("backup --json") {
		t.Fatalf("b guarda la copia con la CLI: %v", f.calls)
	}
	out := view(m)
	if !has(out, "Saved ~/maxor-backup-nitro-20261003-1000.tar.gz") || !has(out, "2.0 MiB") || !has(out, "3 apps") {
		t.Fatalf("el aviso dice dónde quedó y qué lleva:\n%s", out)
	}
	if !strings.Contains(summaryText(m), "Saved a backup") {
		t.Fatalf("y queda en el resumen: %q", summaryText(m))
	}
	// también desde la paleta
	f.calls = nil
	send(m, key(":"))
	typeText(m, "backup")
	send(m, key("enter"))
	if !f.called("backup --json") {
		t.Fatalf("la paleta también lo hace: %v", f.calls)
	}
}

// ── releases firmadas ────────────────────────────────────────────────

func releaseCLI(status, reason string, available bool) *fakeCLI {
	f := newCLI()
	f.resp["release status --json"] = releaseJSON(status, reason, available)
	f.resp["release check --json"] = releaseJSON(status, reason, available)
	f.resp["release check --json --force"] = releaseJSON(status, reason, available)
	f.resp["release apply --yes"] = "Maxor OS 0.2.0 is installed"
	return f
}

func TestUpdateEnseñaUnaReleaseNuevaVerificadaYLaInstala(t *testing.T) {
	f := releaseCLI("ok", "", true)
	m, _ := setupWith(t, f, Options{Screen: "update"})
	m.env.Client.SudoCheck = func(context.Context) bool { return true }
	out := view(m)
	for _, want := range []string{"Maxor OS release", "Maxor OS 0.2.0 is available", "you have 0.1.0", "Signed releases and update notices", "signed and verified", "Install Maxor OS 0.2.0"} {
		if !has(out, want) {
			t.Fatalf("falta %q:\n%s", want, out)
		}
	}
	send(m, key("v"))
	if len(f.stdins) != 1 || f.stdins[0] != "release apply --yes|" {
		t.Fatalf("v instala la release con la CLI, sin pedir confirmación otra vez: %v", f.stdins)
	}
}

func TestUpdateSinReleaseNuevaNoOfreceInstalar(t *testing.T) {
	f := releaseCLI("ok", "", false)
	m, _ := setupWith(t, f, Options{Screen: "update"})
	out := view(m)
	if !has(out, "Maxor OS 0.1.0 is the latest release") || has(out, "Install Maxor OS") {
		t.Fatalf("al día no se ofrece instalar:\n%s", out)
	}
	send(m, key("v"))
	if len(f.stdins) != 0 {
		t.Fatalf("v sin release nueva no ejecuta nada: %v", f.stdins)
	}
}

func TestUnaReleaseIgnoradaPorLaFirmaSeAvisaYNuncaSeOfrece(t *testing.T) {
	f := releaseCLI("insecure", "bad_signature", false)
	m, _ := setupWith(t, f, Options{Screen: "update"})
	out := view(m)
	for _, want := range []string{"A release was ignored", "its signature is not valid", "Do not trust this update"} {
		if !has(out, want) {
			t.Fatalf("falta %q:\n%s", want, out)
		}
	}
	if has(out, "Install Maxor OS") || has(out, "is available") {
		t.Fatalf("lo inseguro nunca se ofrece como actualización:\n%s", out)
	}
	send(m, key("v"))
	if len(f.stdins) != 0 {
		t.Fatalf("no se instala nada: %v", f.stdins)
	}
	send(m, core.GoMsg{ID: "home"})
	if !has(view(m), "A release was ignored") {
		t.Fatalf("el Inicio también lo avisa:\n%s", view(m))
	}
}

func TestSinRedNoSeDiceQueEstaAlDia(t *testing.T) {
	f := releaseCLI("unavailable", "network", false)
	m, _ := setupWith(t, f, Options{Screen: "update"})
	out := view(m)
	if !has(out, "Could not reach the release channel") || !has(out, "last verified 2 min ago") {
		t.Fatalf("sin red se dice, con la última vez que se verificó:\n%s", out)
	}
}

func TestElInicioAvisaDeUnaReleaseYVAlaPestanaUpdate(t *testing.T) {
	f := releaseCLI("ok", "", true)
	m, _ := setupWith(t, f, Options{})
	out := view(m)
	if !has(out, "Maxor OS 0.2.0 is available") || !has(out, "you have 0.1.0") {
		t.Fatalf("el Inicio muestra el aviso:\n%s", out)
	}
	send(m, key("v"))
	if m.screens[m.active].ID() != "update" {
		t.Fatalf("v lleva a Update, está en %s", m.screens[m.active].ID())
	}
}

func TestSalirRecuerdaLaReleasePendiente(t *testing.T) {
	f := releaseCLI("ok", "", true)
	m, _ := setupWith(t, f, Options{Screen: "exit"})
	if !has(view(m), "Maxor OS 0.2.0 is available") {
		t.Fatalf("Exit la recuerda:\n%s", view(m))
	}
}

func TestRescanVuelveAPreguntarPorLaRelease(t *testing.T) {
	f := releaseCLI("ok", "", false)
	m, _ := setupWith(t, f, Options{Screen: "update"})
	send(m, key("r"))
	if f.n("release check --json --force") != 1 {
		t.Fatalf("r repite también la consulta de la release, con --force: %v", f.calls)
	}
}

func TestLaReleaseSePideUnaSolaVezPorSesion(t *testing.T) {
	f := releaseCLI("ok", "", false)
	m, _ := setupWith(t, f, Options{})
	send(m, core.GoMsg{ID: "update"}, core.GoMsg{ID: "exit"}, core.GoMsg{ID: "home"})
	if f.n("release check --json") != 1 || f.n("release status --json") != 1 {
		t.Fatalf("una sola consulta y una sola lectura por sesión: %v", f.calls)
	}
}
