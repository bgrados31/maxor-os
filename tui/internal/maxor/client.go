// Package maxor es el cliente de la CLI `maxor`: la pantalla no reimplementa nada,
// llama a los comandos con --json y lee el resultado. Los códigos de salida
// (docs/CLI.md) se conservan en Error.Code.
package maxor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

// Client llama a la CLI.
type Client struct {
	r   Runner
	bin string
}

type execRunner struct{ bin string }

func (e execRunner) Run(ctx context.Context, args ...string) ([]byte, []byte, int, error) {
	cmd := exec.CommandContext(ctx, e.bin, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	// La pantalla habla con la CLI por JSON: sin colores ni animación.
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "MAXOR_NO_TUI=1")
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
func NewWith(r Runner) *Client { return &Client{r: r, bin: "maxor"} }

// Command devuelve un comando listo para ceder la terminal (p. ej. con sudo).
func (c *Client) Command(args ...string) *exec.Cmd { return exec.Command(c.bin, args...) }

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
	Level string `json:"level"`
	Text  string `json:"text"`
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
	ID     string `json:"id"`
	Source string `json:"source"`
	OK     bool   `json:"ok"`
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
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
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
	UpToDate bool     `json:"up_to_date"`
	Kernel   bool     `json:"kernel"`
	Counts   Counts   `json:"counts"`
	Changes  []Change `json:"changes"`
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

func (c *Client) outcomes(ctx context.Context, args ...string) error {
	var res []Outcome
	err := c.getJSON(ctx, &res, true, args...)
	if err != nil {
		return err
	}
	for _, o := range res {
		if !o.OK {
			return fmt.Errorf("no se pudo completar %s", o.ID)
		}
	}
	return nil
}

// Install instala con el origen indicado (nix o flatpak).
func (c *Client) Install(ctx context.Context, source, id string) error {
	flag := "--nix"
	if source == "flatpak" {
		flag = "--flatpak"
	}
	return c.outcomes(ctx, "install", flag, id, "--json")
}

func (c *Client) Remove(ctx context.Context, id string) error {
	return c.outcomes(ctx, "remove", id, "--json")
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
