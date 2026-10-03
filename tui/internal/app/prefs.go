package app

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// prefs son los ajustes de la pantalla que se recuerdan entre usos.
type prefs struct {
	DetailsHidden bool `json:"details_hidden"` // sin panel de detalles (ni al lado ni abajo)
}

func prefsPath() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "maxor", "tui-prefs.json")
}

func loadPrefs() (p prefs) {
	if b, err := os.ReadFile(prefsPath()); err == nil {
		_ = json.Unmarshal(b, &p)
	}
	return
}

func (p prefs) save() {
	path := prefsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	if b, err := json.Marshal(p); err == nil {
		_ = os.WriteFile(path, b, 0o644)
	}
}
