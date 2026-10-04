// maxor-tui es la pantalla completa de Maxor OS. Se abre sobre tu terminal, tú
// eliges y escribes, y al salir todo sigue donde lo dejaste, con un resumen de
// lo que hiciste. No reimplementa nada: usa los comandos de `maxor`.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/bgrados31/maxor-os/tui/internal/app"
	"github.com/bgrados31/maxor-os/tui/internal/maxor"
	"github.com/bgrados31/maxor-os/tui/internal/theme"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// version la fija la compilación (-ldflags "-X main.version=…").
var version = "dev"

func main() {
	screen := flag.String("screen", "home", "pantalla inicial: home, store, themes, update, doctor, setup o install (el instalador, solo en la sesión viva)")
	noMouse := flag.Bool("no-mouse", false, "sin soporte de ratón")
	search := flag.String("search", "", "abre la Tienda con esta búsqueda ya lanzada")
	showVersion := flag.Bool("version", false, "imprime la versión y sale")
	flag.Parse()

	if *showVersion {
		fmt.Println("maxor-tui", version)
		return
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprintln(os.Stderr, "maxor-tui needs a terminal. Use the maxor commands (with --json) in scripts.")
		os.Exit(3)
	}

	m := app.New(app.Options{Screen: *screen, NoMouse: *noMouse, Version: version, Search: *search}, maxor.New())
	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if !*noMouse {
		opts = append(opts, tea.WithMouseCellMotion())
	}
	if _, err := tea.NewProgram(m, opts...).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "maxor-tui:", err)
		os.Exit(1)
	}
	printSummary(m)
}

// printSummary deja en la terminal normal un resumen corto de lo que se hizo: es
// lo que queda en el historial al volver.
func printSummary(m *app.Model) {
	lines := m.Summary()
	if len(lines) == 0 {
		return
	}
	t := theme.Load()
	st := func(c string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(c)) }
	fmt.Println()
	for _, l := range lines {
		col, g := t.P.Mu, ui.G.Info
		switch l.Kind {
		case "ok":
			col, g = t.OK, ui.G.OK
		case "warn":
			col, g = t.Warn, ui.G.Warn
		case "bad":
			col, g = t.Bad, ui.G.Bad
		}
		fmt.Printf("%s  %s\n", st(col).Render(g), l.Text)
	}
	fmt.Println()
}
