package app

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// The terminal paints its own background around the cells the screen draws (the padding of kitty, the margin
// left over when the window is not a whole number of cells). The screen asks it to use the colour of the theme
// (OSC 11), so no frame of another colour shows around it, also while a theme is being previewed. main resets it
// when the program ends (ResetTerminalBg).

// syncTerminalBg returns the command that tells the terminal the page colour of the current theme, or nil when it
// already has it, or when the output is not a terminal (tests, pipes).
func (m *Model) syncTerminalBg() tea.Cmd {
	bg := m.env.Theme.P.Bg
	if bg == m.termBg || !term.IsTerminal(int(os.Stdout.Fd())) {
		return nil
	}
	m.termBg = bg
	return func() tea.Msg {
		fmt.Fprintf(os.Stdout, "\x1b]11;%s\x07", bg)
		return nil
	}
}

// ResetTerminalBg gives the terminal back its own background colour (OSC 111).
func ResetTerminalBg() {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprint(os.Stdout, "\x1b]111\x07")
	}
}
