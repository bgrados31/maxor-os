// Package theme lee el tema activo de Maxor y lo convierte en los colores de la pantalla.
//
// El tema vive en el mismo sitio que usa la CLI: el nombre en
// $XDG_STATE_HOME/maxor/current y los colores en
// $XDG_DATA_HOME/maxor/themes/<nombre>/colors.json. Si algo falta o es inválido
// se usa Sakura nocturna, igual que hace la CLI.
package theme

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Palette son los ocho colores de un tema (#rrggbb) y su modo.
type Palette struct {
	Bg   string `json:"bg"`
	S    string `json:"s"`
	S2   string `json:"s2"`
	Fg   string `json:"fg"`
	Mu   string `json:"mu"`
	Ac   string `json:"ac"`
	Ac2  string `json:"ac2"`
	On   string `json:"on"`
	Mode string `json:"mode"`
}

// Theme es una paleta con nombre y los colores semánticos derivados del modo.
type Theme struct {
	Name string
	P    Palette
	OK   string
	Warn string
	Bad  string
}

var hexRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Valid dice si los ocho colores son #rrggbb: nada más pasa a la pantalla.
func (p Palette) Valid() bool {
	for _, c := range []string{p.Bg, p.S, p.S2, p.Fg, p.Mu, p.Ac, p.Ac2, p.On} {
		if !hexRe.MatchString(c) {
			return false
		}
	}
	return p.Mode == "" || p.Mode == "dark" || p.Mode == "light"
}

// From construye un tema; los colores de estado se oscurecen en los claros
// para mantener el contraste, como en la CLI.
func From(name string, p Palette) Theme {
	if p.Mode == "" {
		p.Mode = "dark"
	}
	t := Theme{Name: name, P: p, OK: "#7fe3a8", Warn: "#ffc66b", Bad: "#ff6b81"}
	if p.Mode == "light" {
		t.OK, t.Warn, t.Bad = "#1a7f50", "#9a6700", "#c92a3e"
	}
	return t
}

// Sakura es el tema por defecto.
func Sakura() Theme {
	return From("sakura", Palette{
		Bg: "#120b12", S: "#1d121d", S2: "#2a1a2a", Fg: "#fbe9f2", Mu: "#a88a9d",
		Ac: "#ff86b8", Ac2: "#ffc2a6", On: "#1b0b14", Mode: "dark",
	})
}

func dirOr(env, rel string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, rel)
}

// DataDir y StateDir siguen la misma convención XDG que la CLI.
func DataDir() string  { return filepath.Join(dirOr("XDG_DATA_HOME", ".local/share"), "maxor") }
func StateDir() string { return filepath.Join(dirOr("XDG_STATE_HOME", ".local/state"), "maxor") }

// Load lee el tema activo; ante cualquier problema devuelve Sakura.
func Load() Theme { return LoadFrom(DataDir(), StateDir()) }

// LoadFrom es Load con rutas explícitas (para pruebas).
func LoadFrom(dataDir, stateDir string) Theme {
	raw, err := os.ReadFile(filepath.Join(stateDir, "current"))
	if err != nil {
		return Sakura()
	}
	name := strings.TrimSpace(string(raw))
	if name == "" || strings.ContainsAny(name, "/\\") {
		return Sakura()
	}
	return LoadNamed(dataDir, name)
}

// LoadNamed lee un tema por nombre.
func LoadNamed(dataDir, name string) Theme {
	raw, err := os.ReadFile(filepath.Join(dataDir, "themes", name, "colors.json"))
	if err != nil {
		return Sakura()
	}
	var p Palette
	if json.Unmarshal(raw, &p) != nil || !p.Valid() {
		return Sakura()
	}
	return From(name, p)
}
