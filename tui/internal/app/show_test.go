package app

import (
	"fmt"
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bgrados31/maxor-os/tui/internal/core"
)

// TestShow imprime cada pantalla en texto plano, para revisar el diseño sin abrir
// una terminal: MAXOR_TUI_SHOW=1 go test ./internal/app -run TestShow -v
func TestShow(t *testing.T) {
	if os.Getenv("MAXOR_TUI_SHOW") == "" {
		t.Skip("MAXOR_TUI_SHOW no está definido")
	}
	m, _ := setup(t, Options{})
	m.Update(tea.WindowSizeMsg{Width: 110, Height: 30})
	send(m, core.GoMsg{ID: "update"}, key("c"))
	for _, id := range []string{"home", "store", "themes", "doctor", "update", "profiles", "setup"} {
		send(m, core.GoMsg{ID: id})
		fmt.Printf("\n══════ %s ══════\n%s\n", id, view(m))
	}
}
