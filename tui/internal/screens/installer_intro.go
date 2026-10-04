package screens

import (
	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// introStep is the first thing the installer says, and it says it plainly: this medium goes straight to the
// installer (there is no demo session), and nothing is written until the review is confirmed.
type introStep struct{ stepBase }

func (*introStep) ID() string    { return "intro" }
func (*introStep) Title() string { return "Let's install Maxor OS" }
func (*introStep) Intro() string { return "" }

func (*introStep) Lines(w *Installer, env *core.Env, width int) []ui.Line {
	var lines []ui.Line
	for _, para := range []string{
		"This medium starts the installer directly: there is no demo session to try first.",
		"You will answer a few questions and review them. Nothing is written to any disk until you confirm at the end.",
	} {
		for _, l := range ui.Wrap(para, width) {
			lines = append(lines, plain(env, l))
		}
		lines = append(lines, gap())
	}
	return append(lines, muted(env, "Takes about ten minutes. You can go back at any step."))
}
