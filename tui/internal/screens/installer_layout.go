package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// compose lays the window out around the group (the list of steps and the card), and centres all of it together so
// nothing floats far from the rest. The mark heads the list of steps; above the card there is only a thin progress
// line with the count. Where the window is too narrow for the list, the mark and the line sit above the card.
// The keys go right below. The top edge is placed for the usual height of the group, so it does not move when a
// long step grows.
func (w *Installer) compose(env *core.Env, page ui.Painter, group []string, total int, rail bool, steady, width, height int, bare bool) []string {
	pc := page.Ctx()
	blank := ui.Blank().Render(width, pc)
	x0 := max((width-total)/2, 0)
	place := func(segs []ui.Seg) string {
		return ui.Line{L: append([]ui.Seg{{T: strings.Repeat(" ", x0)}}, segs...)}.Render(width, pc)
	}
	center := func(segs []ui.Seg) string { return padSegs(segs, width, pc).Render(width, pc) }

	pos, n := w.stepCount()
	counter := fmt.Sprintf("%d / %d", pos, n)
	frac := w.barShown(env)

	var head []string
	switch {
	case bare: // the welcome and the last screen: just the name
		head = []string{center(w.mark(env, page)), blank}
	case rail:
		// the line sits above the card only: the list of steps beside it already says where you are
		lead := w.railW + 2
		line := []ui.Seg{{T: strings.Repeat(" ", lead)}}
		line = append(line, progressLine(env, page, frac, total-lead-len(counter)-2, w.running)...)
		line = append(line, ui.S(page.Mu, "  "+counter))
		head = []string{place(line), blank}
	default:
		head = []string{
			center(w.mark(env, page)),
			blank,
			place(append(progressLine(env, page, frac, total-len(counter)-2, w.running), ui.S(page.Mu, "  "+counter))),
			blank,
		}
	}
	foot := []string{blank, center(ui.Hints(page, w.Hints(env)))}

	usual := len(head) + max(steady, 0) + len(foot)
	top := max((height-usual)/2, 1)
	top = max(min(top, height-len(head)-len(group)-len(foot)), 0)

	rows := make([]string, 0, height)
	for i := 0; i < top; i++ {
		rows = append(rows, blank)
	}
	rows = append(rows, head...)
	margin := func(n int) string { return page.Fill.Render(strings.Repeat(" ", max(n, 0))) }
	for _, g := range group {
		rows = append(rows, margin(x0)+g+margin(width-x0-total))
	}
	rows = append(rows, foot...)
	for len(rows) < height {
		rows = append(rows, blank)
	}
	return rows[:height]
}

// stepCount is the position of the current step among those a person goes through, and how many there are.
func (w *Installer) stepCount() (pos, n int) {
	vis := w.visible()
	for k, i := range vis {
		if i <= w.idx {
			pos = k + 1
		}
	}
	if w.running {
		pos = len(vis)
	}
	return pos, len(vis)
}

// progressLine is a thin line of width cells: the done part in a gradient between the two accents, ending in a
// half cell so it moves by half cells as it glides, and the rest barely there. While work is going on a bright
// cell travels along the done part.
func progressLine(env *core.Env, p ui.Painter, frac float64, width int, animate bool) []ui.Seg {
	if width <= 0 {
		return nil
	}
	frac = min(max(frac, 0), 1)
	halves := int(frac*float64(width*2) + 0.5)
	full, half := halves/2, halves%2 == 1
	t := env.Theme.P
	rest := ui.Mix(t.S, t.Mu, 0.35)
	glow := -1
	if animate && full > 0 {
		glow = (env.Frame / 2) % full
	}
	// the bright cell goes towards white: the text colour on a dark theme, the page on a light one
	shine := t.Fg
	if t.Mode == "light" {
		shine = t.Bg
	}
	segs := make([]ui.Seg, 0, width)
	for i := 0; i < width; i++ {
		col := ui.Mix(t.Ac, t.Ac2, float64(i)/float64(max(width-1, 1)))
		switch {
		case i < full:
			if i == glow {
				col = ui.Mix(col, shine, 0.55)
			}
			segs = append(segs, ui.S(p.Fill.Foreground(lipgloss.Color(col)), ui.G.Line))
		case i == full && half:
			segs = append(segs, ui.S(p.Fill.Foreground(lipgloss.Color(col)), ui.G.HalfLine))
		default:
			segs = append(segs, ui.S(p.Fill.Foreground(lipgloss.Color(rest)), ui.G.Rule))
		}
	}
	return segs
}
