package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestInstallerGallery draws every step of the installer, in colour, at the sizes people use, into
// $MAXOR_GALLERY/<size>/<nn>-<step>.ansi: a design review without a virtual machine. It does nothing unless that
// variable is set. Turn the files into pictures with scripts/tui-gallery.sh.
func TestInstallerGallery(t *testing.T) {
	out := os.Getenv("MAXOR_GALLERY")
	if out == "" {
		t.Skip("set MAXOR_GALLERY to a directory to draw the gallery")
	}
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	for _, size := range [][2]int{{132, 40}, {100, 34}, {80, 30}} {
		dir := filepath.Join(out, fmt.Sprintf("%dx%d", size[0], size[1]))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		n := 0
		shot := func(m *Model, name string) {
			t.Helper()
			n++
			m.env.Frame += 20 // past the transitions
			m.env.Clock += 5e9
			path := filepath.Join(dir, fmt.Sprintf("%02d-%s.ansi", n, name))
			if err := os.WriteFile(path, []byte(m.View()), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		e := newInstallEnv(windowsDisk(), emptyDisk())
		m := introModel(t, e)
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		shot(m, "intro")
		send(m, key("enter"))
		shot(m, "language")
		send(m, key("enter"))
		shot(m, "keyboard")
		send(m, key("enter"))
		shot(m, "network")
		send(m, key("enter"))
		shot(m, "timezone")
		send(m, key("enter"))
		shot(m, "disk")
		send(m, key("enter")) // the Windows disk
		shot(m, "strategy")
		send(m, key("down"))
		shot(m, "strategy-alongside")
		send(m, key("enter"))
		shot(m, "storage")
		send(m, key("down"), key(" "))
		shot(m, "storage-encrypted")
		send(m, key(" "), key("down"), key("right"))
		shot(m, "storage-swapfile")
		send(m, key("left"), key("enter"))
		shot(m, "account")
		fillAccount(m, "Ana Pérez", "ana", "ana-pc", "correct-horse-1")
		send(m, key("enter"))
		shot(m, "look")
		send(m, key("enter"))
		shot(m, "hardware")
		send(m, key("enter"))
		shot(m, "review")
	}
}
