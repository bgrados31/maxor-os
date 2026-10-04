package screens

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/install"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// The installer draws its own window: the mark with a moving gradient and the progress on top, the list of
// steps (with what was chosen) on the left, the step in a card in the middle, the keys at the bottom. The
// design reference is docs/design/installer.html.

const (
	headerRows = 5
	footerRows = 2
	railWidth  = 27
	cardWidth  = 66
	slideFrames = 6 // frames (80 ms each) a card takes to slide into place
	minCard     = 56 // the narrowest a card gets when the list of steps is shown
)

// Animated: the gradient of the mark and the slide between steps need steady frames.
func (w *Installer) Animated() bool { return true }

func padSegs(segs []ui.Seg, width int, ctx ui.Ctx) ui.Line {
	n := 0
	for _, s := range segs {
		n += ansi.StringWidth(s.T)
	}
	if left := (width - n) / 2; left > 0 {
		segs = append([]ui.Seg{{T: strings.Repeat(" ", left)}}, segs...)
	}
	return ui.Line{L: segs}
}

// mark is the name with a gradient that flows from one accent to the other.
func (w *Installer) mark(env *core.Env, page ui.Painter) []ui.Seg {
	letters := []rune("MAXOR OS")
	var segs []ui.Seg
	for i, r := range letters {
		t := (math.Sin(float64(env.Frame)*0.14-float64(i)*0.55) + 1) / 2
		col := ui.Mix(env.Theme.P.Ac, env.Theme.P.Ac2, t)
		segs = append(segs, ui.S(page.Fill.Foreground(lipgloss.Color(col)).Bold(true), string(r)))
		if i < len(letters)-1 {
			segs = append(segs, ui.S(page.Fill, " "))
		}
	}
	return segs
}

// visible lists the steps the rail and the counter show: those a person goes through.
func (w *Installer) visible() []int {
	var out []int
	for i, s := range w.steps {
		if i == len(w.steps)-1 || s.ID() == "install" || s.ID() == "intro" {
			continue
		}
		out = append(out, i)
	}
	return out
}

// value is what the rail shows next to a finished step.
func (w *Installer) value(id string) string {
	st := w.st
	switch id {
	case "welcome":
		return strings.TrimSpace(install.LocaleName(st.Locale))
	case "keyboard":
		return install.FindLayout(st.XKBLayout, st.XKBVariant).XKB
	case "network":
		if st.Offline {
			return "offline"
		}
		return "online"
	case "region":
		if i := strings.LastIndex(st.Timezone, "/"); i >= 0 {
			return strings.ReplaceAll(st.Timezone[i+1:], "_", " ")
		}
		return st.Timezone
	case "disk":
		return strings.TrimPrefix(st.Disk, "/dev/")
	case "strategy":
		return map[string]string{"whole": "whole disk", "alongside": "alongside"}[st.Strategy]
	case "storage":
		v := st.Filesystem
		if st.Encrypt {
			v += " · luks"
		}
		return v
	case "account":
		return st.Username
	case "look":
		return st.Theme
	case "hardware":
		if st.GPU == "" || st.GPU == "auto" {
			return "detected"
		}
		return st.GPU
	}
	return ""
}

func (w *Installer) railLines(env *core.Env) []ui.Line {
	p := env.P
	lines := []ui.Line{ui.T(p.Mu, "install steps"), ui.Blank()}
	for _, i := range w.visible() {
		s := w.steps[i]
		if sk, ok := s.(interface{ Skip(*Installer) bool }); ok && i != w.idx && i > w.idx && sk.Skip(w) {
			continue
		}
		var mark ui.Seg
		name := p.Mu
		switch {
		case i < w.idx:
			mark, name = ui.S(p.Ok, ui.G.Tick+" "), p.Text
		case i == w.idx:
			mark, name = ui.S(p.Ac, ui.G.Arrow+" "), p.Ac.Bold(true)
		default:
			mark = ui.S(p.Mu, ui.G.Info+" ")
		}
		segs := []ui.Seg{mark, ui.S(name, s.Title())}
		line := ui.Line{L: segs}
		if i < w.idx {
			if v := w.value(s.ID()); v != "" {
				line.R = []ui.Seg{ui.S(p.Mu, v+" ")}
			}
		}
		lines = append(lines, line)
	}
	return lines
}

func (w *Installer) cardLines(env *core.Env, width int) []ui.Line {
	p := env.P
	s := w.cur()
	lines := []ui.Line{ui.T(p.Ac.Bold(true), s.Title())}
	if intro := s.Intro(); intro != "" {
		for _, l := range ui.Wrap(intro, width) {
			lines = append(lines, ui.T(p.Mu, l))
		}
	}
	lines = append(lines, ui.Blank())
	lines = append(lines, s.Lines(w, env, width)...)
	if w.notice != "" {
		lines = append(lines, ui.Blank())
		for i, l := range ui.Wrap(w.notice, width-3) {
			mark := "   "
			if i == 0 {
				mark = ui.G.Warn + "  "
			}
			lines = append(lines, ui.Of(ui.S(p.Warn, mark), ui.S(p.Text, l)))
		}
	}
	return lines
}

// Frame draws the whole window.
func (w *Installer) Frame(env *core.Env, width, height int) []string {
	page := ui.NewPainter(env.Theme, env.Theme.P.Bg)
	pc := page.Ctx()
	final := w.idx == len(w.steps)-1
	intro := w.cur().ID() == "intro"
	blank := ui.Blank().Render(width, pc)

	rows := make([]string, 0, height)
	rows = append(rows, blank, padSegs(w.mark(env, page), width, pc).Render(width, pc), blank)
	if final || intro {
		rows = append(rows, blank)
	} else {
		vis := w.visible()
		pos := 0
		for n, i := range vis {
			if i <= w.idx {
				pos = n + 1
			}
		}
		if w.running {
			pos = len(vis)
		}
		barW := 24
		segs := append([]ui.Seg{ui.S(page.Mu, "install  ")}, ui.Bar(page, pos*100/max(len(vis), 1), barW)...)
		segs = append(segs, ui.S(page.Mu, fmt.Sprintf("  %d/%d", pos, len(vis))))
		rows = append(rows, padSegs(segs, width, pc).Render(width, pc))
	}
	rows = append(rows, blank)

	bodyH := height - headerRows - footerRows
	showRail := !w.running && !final && !intro && w.cur().ID() != "install" && width >= railWidth+minCard+6
	cw := min(cardWidth, width-2)
	if showRail {
		cw = min(cardWidth, width-railWidth-6) // the card gives way before the list of steps does
	}
	if !showRail && !final && w.cur().ID() == "install" {
		cw = min(cardWidth+12, width-2)
	}

	lines := w.cardLines(env, cw-6)
	off := 0
	if k := env.Frame - w.changed; k >= 0 && k < slideFrames {
		off = (slideFrames - k) * 3 // the card slides in from the right
	}
	ch := min(bodyH, max(len(lines)+3, 12))
	card := ui.Block(ui.Inset(lines, 3+off, 1), cw, ch, env.P.Ctx())

	var body []string
	margin := func(n int) string { return page.Fill.Render(strings.Repeat(" ", max(n, 0))) }
	if showRail {
		rail := ui.Block(ui.Inset(w.railLines(env), 2, 1), railWidth, ch, env.P.Ctx())
		total := railWidth + 2 + cw
		x0 := (width - total) / 2
		for i := 0; i < ch; i++ {
			body = append(body, margin(x0)+rail[i]+margin(2)+card[i]+margin(width-x0-total))
		}
	} else {
		x0 := (width - cw) / 2
		for i := 0; i < ch; i++ {
			body = append(body, margin(x0)+card[i]+margin(width-x0-cw))
		}
	}
	top := (bodyH - len(body)) / 2
	for i := 0; i < top; i++ {
		rows = append(rows, blank)
	}
	rows = append(rows, body...)
	for len(rows) < height-footerRows {
		rows = append(rows, blank)
	}
	rows = append(rows, blank, padSegs(ui.Hints(page, w.Hints(env)), width, pc).Render(width, pc))
	return rows[:height]
}
