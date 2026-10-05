// Package maxor es el cliente de la CLI `maxor`: la pantalla no reimplementa nada,
// llama a los comandos con --json y lee el resultado. Los códigos de salida
// (docs/CLI.md) se conservan en Error.Code.
package maxor

import (
	"github.com/bgrados31/maxor-os/tui/internal/i18n"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Runner ejecuta `maxor` con unos argumentos. Es una interfaz para poder probar
// la pantalla sin la CLI real.
type Runner interface {
	Run(ctx context.Context, args ...string) (stdout, stderr []byte, code int, err error)
}

// Error es un fallo de la CLI con su código de salida y lo último que dijo.
type Error struct {
	Args   []string
	Code   int
	Stderr string
}

func (e *Error) Error() string {
	msg := lastLine(e.Stderr)
	if msg == "" {
		msg = fmt.Sprintf("maxor %s terminó con el código %d", strings.Join(e.Args, " "), e.Code)
	}
	return msg
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			return l
		}
	}
	return ""
}

// Streamer es opcional: ejecuta con una entrada estándar y va entregando cada línea de la
// salida mientras corre (para mostrar el progreso de una actualización).
type Streamer interface {
	Stream(ctx context.Context, stdin string, onLine func(string), args ...string) (code int, err error)
}

// Client llama a la CLI.
type Client struct {
	r   Runner
	bin string
	// SudoCheck dice si sudo funciona sin contraseña. Por defecto lo pregunta de verdad;
	// las pruebas lo sustituyen.
	SudoCheck func(context.Context) bool
}

type execRunner struct{ bin string }

func (e execRunner) Run(ctx context.Context, args ...string) ([]byte, []byte, int, error) {
	cmd := exec.CommandContext(ctx, e.bin, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	// La pantalla habla con la CLI por JSON: sin colores ni animación.
	cmd.Env = cliEnv()
	err := cmd.Run()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code, err = ee.ExitCode(), nil
	}
	return out.Bytes(), errb.Bytes(), code, err
}

// Bin es el ejecutable de la CLI: MAXOR_BIN o `maxor` del PATH.
func Bin() string {
	if b := os.Getenv("MAXOR_BIN"); b != "" {
		return b
	}
	return "maxor"
}

// New crea el cliente real.
func New() *Client { return &Client{r: execRunner{bin: Bin()}, bin: Bin()} }

// NewWith crea un cliente con un Runner propio (pruebas).
func NewWith(r Runner) *Client {
	return &Client{r: r, bin: "maxor", SudoCheck: func(context.Context) bool { return false }}
}

// Command devuelve un comando listo para ceder la terminal (p. ej. con sudo).
func (c *Client) Command(args ...string) *exec.Cmd {
	cmd := exec.Command(c.bin, args...)
	cmd.Env = append(os.Environ(), "MAXOR_LANG="+i18n.Code())
	return cmd
}

// cliEnv is the environment the CLI runs with when the screen reads its answers: no colors or
// animation, and the language of the interface, so what the CLI says matches what the screen says.
func cliEnv() []string {
	return append(os.Environ(), "NO_COLOR=1", "MAXOR_NO_TUI=1", "MAXOR_LANG="+i18n.Code())
}

func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
	out, errb, code, err := c.r.Run(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("no se pudo ejecutar maxor: %w", err)
	}
	if code != 0 {
		return out, &Error{Args: args, Code: code, Stderr: string(errb)}
	}
	return out, nil
}

// getJSON ejecuta y decodifica. Con tolerant, un código distinto de cero no es
// un fallo si la salida es JSON válido (doctor sale con 1 cuando hay problemas).
func (c *Client) getJSON(ctx context.Context, v any, tolerant bool, args ...string) error {
	out, err := c.run(ctx, args...)
	if err != nil {
		var me *Error
		if !tolerant || !errors.As(err, &me) || len(bytes.TrimSpace(out)) == 0 {
			return err
		}
	}
	if jerr := json.Unmarshal(bytes.TrimSpace(out), v); jerr != nil {
		return fmt.Errorf("respuesta de maxor no válida (%s): %w", strings.Join(args, " "), jerr)
	}
	return nil
}

// ── Tipos del contrato JSON ──────────────────────────────────────────

type DoctorItem struct {
	Level   string `json:"level"`
	Text    string `json:"text"`
	ID      string `json:"id"`      // clave de la comprobación (p. ej. git_dirty)
	Fix     string `json:"fix"`     // comando que lo arregla o ayuda a verlo; vacío si no hay
	Confirm bool   `json:"confirm"` // el arreglo pide confirmación (borra o cambia cosas)
	Kind    string `json:"kind"`    // "fix" arregla algo; "inspect" solo enseña información
}
type DoctorGroup struct {
	Title string       `json:"title"`
	Items []DoctorItem `json:"items"`
}
type Doctor struct {
	OK     bool          `json:"ok"`
	Fails  int           `json:"fails"`
	Warns  int           `json:"warns"`
	Groups []DoctorGroup `json:"groups"`
}

type ThemeColors struct {
	Bg  string `json:"bg"`
	S   string `json:"s"`
	S2  string `json:"s2"`
	Fg  string `json:"fg"`
	Mu  string `json:"mu"`
	Ac  string `json:"ac"`
	Ac2 string `json:"ac2"`
	On  string `json:"on"`
}
type Theme struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Mode   string      `json:"mode"`
	Active bool        `json:"active"`
	Colors ThemeColors `json:"colors"`
}

type App struct {
	Source  string `json:"source"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}
type Result struct {
	Source      string `json:"source"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}
type Outcome struct {
	ID        string     `json:"id"`
	Source    string     `json:"source"`
	OK        bool       `json:"ok"`
	Purged    bool       `json:"purged"`
	Leftovers []Leftover `json:"leftovers"`
}

// Leftover es una carpeta que una app dejó en tu casa.
type Leftover struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

type GPU struct {
	Vendor  string `json:"vendor"`
	ID      string `json:"id"`
	Bus     string `json:"bus"`
	Primary bool   `json:"primary"`
}
type Hardware struct {
	CPU struct {
		Vendor string `json:"vendor"`
		Model  string `json:"model"`
	} `json:"cpu"`
	GPUs      []GPU  `json:"gpus"`
	Laptop    bool   `json:"laptop"`
	Virt      string `json:"virt"`
	Bluetooth bool   `json:"bluetooth"`
}

type Profile struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string   `json:"description"`
	Includes    []string `json:"includes"` // lo que instala, uno por línea
	Enabled     bool     `json:"enabled"`
}

type Change struct {
	Kind string `json:"kind"` // updated | new | removed | changed
	Name string `json:"name"`
	From string `json:"from"`
	To   string `json:"to"`
	Size string `json:"size"`
}
type Counts struct {
	New     int `json:"new"`
	Updated int `json:"updated"`
	Removed int `json:"removed"`
	Changed int `json:"changed"`
	Config  int `json:"config"`
}
type UpdateCheck struct {
	UpToDate    bool     `json:"up_to_date"`
	Kernel      bool     `json:"kernel"`
	Counts      Counts   `json:"counts"`
	Changes     []Change `json:"changes"`
	CheckedAt   int64    `json:"checked_at"`  // cuándo se hizo el escaneo (segundos Unix)
	Fingerprint string   `json:"fingerprint"` // huella del repositorio en ese momento
	Lock        bool     `json:"lock"`        // si refrescó las entradas del flake
}

// UpdateStatus es el estado barato de la configuración (no compila nada).
type UpdateStatus struct {
	Flake       string `json:"flake"`
	Branch      string `json:"branch"`
	Commit      string `json:"commit"`
	Dirty       bool   `json:"dirty"`
	Files       int    `json:"files"`
	Fingerprint string `json:"fingerprint"`
	Channel     string `json:"channel"`
	NixpkgsRev  string `json:"nixpkgs_rev"`
	NixpkgsDate int64  `json:"nixpkgs_date"`
	Generation  int    `json:"generation"`
}

// Release es el estado de las releases firmadas de Maxor OS (`maxor release status --json`).
// Status: ok (la firma verificó), unavailable (no se pudo mirar: NO es «al día»), insecure
// (algo no pasó la verificación y se ignoró) o never (aún no se miró).
type Release struct {
	Installed string `json:"installed"`
	Status    string `json:"status"`
	Reason    string `json:"reason"`
	CheckedAt int64  `json:"checked_at"` // el último intento, con o sin éxito
	OkAt      int64  `json:"ok_at"`      // la última vez que se verificó bien
	Available bool   `json:"available"`
	Latest    string `json:"latest"`
	Tag       string `json:"tag"`
	Commit    string `json:"commit"`
	Sequence  int64  `json:"sequence"`
	Published string `json:"published"`
	Summary   string `json:"summary"`
	URL       string `json:"url"`
	Stale     bool   `json:"stale"` // hace más de 3 días que no se verifica nada
}

type Version struct {
	Version string `json:"version"`
	Schema  int    `json:"schema"`
	OS      string `json:"os"`
}

// ── Llamadas ─────────────────────────────────────────────────────────

func (c *Client) Version(ctx context.Context) (v Version, err error) {
	err = c.getJSON(ctx, &v, false, "version", "--json")
	return
}

func (c *Client) Doctor(ctx context.Context) (d Doctor, err error) {
	err = c.getJSON(ctx, &d, true, "doctor", "--json")
	return
}

func (c *Client) Themes(ctx context.Context) (t []Theme, err error) {
	err = c.getJSON(ctx, &t, false, "theme", "list", "--json")
	return
}

func (c *Client) ApplyTheme(ctx context.Context, id string) error {
	_, err := c.run(ctx, "theme", "apply", id)
	return err
}

func (c *Client) UndoTheme(ctx context.Context) error {
	_, err := c.run(ctx, "theme", "undo")
	return err
}

func (c *Client) Apps(ctx context.Context) (a []App, err error) {
	err = c.getJSON(ctx, &a, false, "apps", "--json")
	return
}

func (c *Client) Search(ctx context.Context, query string) (r []Result, err error) {
	args := append([]string{"search"}, strings.Fields(query)...)
	err = c.getJSON(ctx, &r, false, append(args, "--json")...)
	return
}

func (c *Client) outcomes(ctx context.Context, args ...string) ([]Outcome, error) {
	var res []Outcome
	err := c.getJSON(ctx, &res, true, args...)
	if err != nil {
		return res, err
	}
	for _, o := range res {
		if !o.OK {
			return res, fmt.Errorf("no se pudo completar %s", o.ID)
		}
	}
	return res, nil
}

// Install instala con el origen indicado (nix o flatpak).
func (c *Client) Install(ctx context.Context, source, id string) error {
	flag := "--nix"
	if source == "flatpak" {
		flag = "--flatpak"
	}
	_, err := c.outcomes(ctx, "install", flag, id, "--json")
	return err
}

// Remove quita la app y devuelve las carpetas que dejó en tu casa (no las borra).
func (c *Client) Remove(ctx context.Context, id string) ([]Leftover, error) {
	res, err := c.outcomes(ctx, "remove", id, "--json")
	if len(res) == 0 {
		return nil, err
	}
	return res[0].Leftovers, err
}

// Purge borra las carpetas que dejó la app (también si ya no está instalada).
func (c *Client) Purge(ctx context.Context, id string) error {
	_, err := c.outcomes(ctx, "remove", id, "--purge", "--json")
	return err
}

func (c *Client) Hardware(ctx context.Context) (h Hardware, err error) {
	err = c.getJSON(ctx, &h, false, "hardware", "detect")
	return
}

func (c *Client) Profiles(ctx context.Context) (p []Profile, err error) {
	err = c.getJSON(ctx, &p, false, "profile", "list", "--json")
	return
}

// SetProfile guarda la elección de un perfil sin reconstruir el sistema.
func (c *Client) SetProfile(ctx context.Context, id string, enable bool) error {
	verb := "disable"
	if enable {
		verb = "enable"
	}
	_, err := c.run(ctx, "profile", verb, id, "--no-apply")
	return err
}

// UpdateCheck compila y compara sin aplicar nada.
func (c *Client) UpdateCheck(ctx context.Context, lock bool) (u UpdateCheck, err error) {
	args := []string{"update", "--json"}
	if !lock {
		args = append(args, "--no-lock")
	}
	err = c.getJSON(ctx, &u, false, args...)
	return
}

// UpdateStatus lee la rama, el canal de nixpkgs y la generación: al instante.
func (c *Client) UpdateStatus(ctx context.Context) (u UpdateStatus, err error) {
	err = c.getJSON(ctx, &u, false, "update", "--status")
	return
}

// UpdateCached devuelve el último escaneo guardado, o nil si no hay.
func (c *Client) UpdateCached(ctx context.Context) (u *UpdateCheck, err error) {
	err = c.getJSON(ctx, &u, false, "update", "--cached")
	return
}

// ReleaseStatus lee lo último que se supo de las releases, sin tocar la red.
func (c *Client) ReleaseStatus(ctx context.Context) (r Release, err error) {
	err = c.getJSON(ctx, &r, false, "release", "status", "--json")
	return
}

// ReleaseCheck pregunta al canal de releases (la CLI verifica la firma y no vuelve a la red si
// se miró hace pocos minutos, salvo con force). Sin red o con una firma mala la CLI sale con
// error pero imprime el estado: es el resultado, no un fallo de la pantalla.
func (c *Client) ReleaseCheck(ctx context.Context, force bool) (r Release, err error) {
	args := []string{"release", "check", "--json"}
	if force {
		args = append(args, "--force")
	}
	err = c.getJSON(ctx, &r, true, args...)
	return
}

// AppUpdate es una app instalada que tiene una versión nueva.
type AppUpdate struct {
	Source  string `json:"source"`
	ID      string `json:"id"`
	Current string `json:"current"`
	Latest  string `json:"latest"`
}

// AppUpdates lista las apps con versión nueva (las de nix, frente al nixpkgs de este sistema).
func (c *Client) AppUpdates(ctx context.Context) (u []AppUpdate, err error) {
	err = c.getJSON(ctx, &u, false, "apps", "updates")
	return
}

// AppUpdatesFresh vuelve a mirar las versiones nuevas, sin usar lo que la CLI guardó.
func (c *Client) AppUpdatesFresh(ctx context.Context) (u []AppUpdate, err error) {
	err = c.getJSON(ctx, &u, false, "apps", "updates", "--refresh")
	return
}

// UpdateApp actualiza una sola app.
func (c *Client) UpdateApp(ctx context.Context, id string) error {
	_, err := c.outcomes(ctx, "apps", "update", id, "--json")
	return err
}

// OpenApp abre una app instalada, separada de la terminal.
func (c *Client) OpenApp(ctx context.Context, id string) error {
	_, err := c.outcomes(ctx, "apps", "open", id, "--json")
	return err
}

// AppData lista las carpetas que la app tiene en tu casa, sin tocar nada.
func (c *Client) AppData(ctx context.Context, id string) (l []Leftover, err error) {
	err = c.getJSON(ctx, &l, false, "remove", id, "--list-data")
	return
}

// Generation es una versión anterior del sistema a la que se puede volver.
type Generation struct {
	Generation int    `json:"generation"`
	Date       string `json:"date"`
	Nixos      string `json:"nixos"`
	Kernel     string `json:"kernel"`
	Current    bool   `json:"current"`
}

// Generations lista las últimas generaciones del sistema (la más nueva primero).
func (c *Client) Generations(ctx context.Context) (g []Generation, err error) {
	err = c.getJSON(ctx, &g, false, "rollback", "--list", "--json")
	return
}

// SudoReady dice si se puede usar sudo sin pedir contraseña (la tiene en caché o no hace falta).
func (c *Client) SudoReady(ctx context.Context) bool {
	if c.SudoCheck != nil {
		return c.SudoCheck(ctx)
	}
	return exec.CommandContext(ctx, "sudo", "-n", "true").Run() == nil
}

// Stream ejecuta `maxor` entregando cada línea de salida (stdout y stderr juntos) a onLine.
// Con stdin no vacío se lo da a sudo como contraseña (MAXOR_SUDO_STDIN=1). Devuelve el
// código de salida; un error solo si no se pudo ni empezar.
func (c *Client) Stream(ctx context.Context, stdin string, onLine func(string), args ...string) (int, error) {
	if s, ok := c.r.(Streamer); ok {
		return s.Stream(ctx, stdin, onLine, args...)
	}
	out, errb, code, err := c.r.Run(ctx, args...)
	for _, l := range strings.Split(string(out)+string(errb), "\n") {
		if strings.TrimSpace(l) != "" {
			onLine(l)
		}
	}
	return code, err
}

// Stream de la CLI real.
func (e execRunner) Stream(ctx context.Context, stdin string, onLine func(string), args ...string) (int, error) {
	cmd := exec.CommandContext(ctx, e.bin, args...)
	cmd.Env = cliEnv()
	// The screen recognises sudo's and nixos-rebuild's own lines (a wrong password, the phase of the
	// switch) by their English text: those programs speak English here whatever the system language is.
	cmd.Env = append(cmd.Env, "LC_MESSAGES=C", "LANGUAGE=")
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin + "\n")
		cmd.Env = append(cmd.Env, "MAXOR_SUDO_STDIN=1")
	}
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("no se pudo ejecutar maxor: %w", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		sc.Split(scanLines)
		for sc.Scan() {
			if l := strings.TrimSpace(ansi.Strip(sc.Text())); l != "" {
				onLine(l)
			}
		}
	}()
	err := cmd.Wait()
	_ = pw.Close()
	<-done
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code, err = ee.ExitCode(), nil
	}
	return code, err
}

// scanLines corta por \n y también por \r: las barras de progreso reescriben la misma línea.
func scanLines(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// BackupInfo es el resultado de guardar una copia de seguridad.
type BackupInfo struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	Apps   int    `json:"apps"`
	Themes int    `json:"themes"`
	Host   string `json:"host"`
}

// Backup guarda tu configuración (perfiles, temas propios, ajustes y apps) en un archivo.
func (c *Client) Backup(ctx context.Context) (b BackupInfo, err error) {
	err = c.getJSON(ctx, &b, false, "backup", "--json")
	return
}
