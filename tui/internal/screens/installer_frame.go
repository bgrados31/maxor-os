package screens

import (
	"fmt"
	"math"
	"strings"
	"time"

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
	railWide   = 32 // the list of steps where the window has room for it
	cardWidth  = 66
	minCard    = 56 // the narrowest a card gets when the list of steps is shown

	slideIn    = 260 * time.Millisecond // a new card slides and fades into place
	slideCells = 4                      // from this many cells away: short, so it reads as motion and not as a jump
	barGlide   = 420 * time.Millisecond // the progress bar glides to the new step
)

// Animated: the gradient of the mark and the slide between steps need steady frames.
func (w *Installer) Animated() bool { return true }

// Smooth: while a card slides in or the bar glides, frames come at 30 per second.
func (w *Installer) Smooth(env *core.Env) bool { return env.Clock-w.changedAt < barGlide }

// barTarget is the share of the steps done, the end the progress bar glides to.
func (w *Installer) barTarget() float64 {
	if w.running {
		return 1
	}
	vis := w.visible()
	pos := 0
	for n, i := range vis {
		if i <= w.idx {
			pos = n + 1
		}
	}
	return float64(pos) / float64(max(len(vis), 1))
}

// barShown is what the progress bar shows now: on its way from where it was to the current step.
func (w *Installer) barShown(env *core.Env) float64 {
	t := ui.EaseOut(ui.Progress(env.Clock-w.changedAt, barGlide))
	return w.barFrom + (w.barTarget()-w.barFrom)*t
}

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

// visible lists the steps the rail and the counter show: those a person goes through. A step the installer settles
// on its own (the way to install, when a disk allows only one) stays in the list, done with its value: the count
// does not change half way, and the list says what happened.
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
		// «English (United States)» → «English»: the rail has little room, and the review says it all
		name := install.LocaleName(st.Locale)
		if i := strings.Index(name, " ("); i > 0 {
			name = name[:i]
		}
		return strings.TrimSpace(name)
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
		return map[string]string{"whole": tr("erase disk"), "alongside": tr("alongside")}[st.Strategy]
	case "storage":
		v := st.Filesystem
		if st.Encrypt {
			v += " · luks"
		}
		return v
	case "account":
		return st.Username
	case "look":
		return strings.ToLower(lookName(st.Theme))
	case "hardware":
		if st.GPU == "" || st.GPU == "auto" {
			return tr("detected")
		}
		return st.GPU
	}
	return ""
}

func (w *Installer) railLines(env *core.Env) []ui.Line {
	p := env.P
	// the mark heads the list: there is no header above it
	lines := []ui.Line{ui.Of(w.mark(env, env.P)...), ui.Blank()}
	for _, i := range w.visible() {
		s := w.steps[i]
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
		label := s.Title()
		if w.cur().ID() == "summary" {
			n := 0
			for k, v := range w.editable() {
				if v == i {
					n = k + 1
				}
			}
			if n >= 1 && n <= 9 {
				label = fmt.Sprintf("%d  %s", n, label) // the review lets you edit a step by its number
			}
		}
		segs := []ui.Seg{mark, ui.S(name, label)}
		line := ui.Line{L: segs}
		if i < w.idx {
			// the value gets what the name leaves; if that is too little, it is left out rather than eating the name
			room := w.railW - 4 - 2 - ansi.StringWidth(label) - 2
			if v := w.value(s.ID()); v != "" && room >= 4 {
				line.R = []ui.Seg{ui.S(p.Mu, ansi.Truncate(v, room, "…")+" ")}
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
	final := w.idx == len(w.steps)-1
	intro := w.cur().ID() == "intro"

	bodyH := height - headerRows - footerRows
	// the list of steps is wider where there is room, so what was chosen fits next to each step
	w.railW = railWidth
	if width >= railWide+cardWidth+8 {
		w.railW = railWide
	}
	showRail := !w.running && !final && !intro && w.cur().ID() != "install" && width >= w.railW+minCard+6
	w.railShown = showRail
	cw := min(cardWidth, width-2)
	if showRail {
		cw = min(cardWidth, width-w.railW-6) // the card gives way before the list of steps does
	}
	if !showRail && !final && w.cur().ID() == "install" {
		cw = min(cardWidth+12, width-2)
	}

	w.bodyH = bodyH
	lines := w.cardLines(env, cw-6)
	// what a row puts on the right keeps the same margin from the edge as the text on the left
	for i := range lines {
		if len(lines[i].R) > 0 {
			lines[i].R = append(append([]ui.Seg(nil), lines[i].R...), ui.Seg{T: "   "})
		}
	}
	// the row with the focus is tinted with the accent, and keeps its own colours: calmer than a solid bar
	ctx := env.P.Ctx()
	// (softer on a light theme: there the tint darkens the row, and the text on it must keep 5:1)
	tint := 0.16
	if env.Theme.P.Mode == "light" {
		tint = 0.10
	}
	ctx.SelBg = lipgloss.Color(ui.Mix(string(env.P.Bg), env.Theme.P.Ac, tint))
	// A new card comes in from the side it was reached from (the right going forward, the left going back),
	// a few cells away, easing out while it fades in from the card's colour.
	if t := ui.EaseOut(ui.Progress(env.Clock-w.changedAt, slideIn)); t < 1 {
		bg := string(env.P.Bg)
		lines = ui.Fade(lines, bg, t)
		ctx = ui.FadeCtx(ctx, bg, t)
	}
	inset := 3 + w.dir*ui.Lerp(slideCells, 0, ui.EaseOut(ui.Progress(env.Clock-w.changedAt, slideIn)))
	inset = max(inset, 0)

	// The card is never shorter than the list of steps beside it: the list is never cut, and the card does not
	// change height from one short step to the next. Its top edge stays put; a long step grows downwards.
	railRows := w.railLines(env)
	steady := max(len(railRows)+2, 12)
	ch := min(bodyH, max(len(lines)+3, steady))
	card := ui.Block(ui.Inset(lines, inset, 1), cw, ch, ctx)

	// the group: the list of steps and the card side by side, exactly total cells wide
	var group []string
	total := cw
	if showRail {
		rail := ui.Block(ui.Inset(railRows, 2, 1), w.railW, ch, env.P.Ctx())
		total = w.railW + 2 + cw
		for i := 0; i < ch; i++ {
			group = append(group, rail[i]+page.Fill.Render("  ")+card[i])
		}
	} else {
		group = card
	}
	return w.compose(env, page, group, total, showRail, min(steady, bodyH), width, height, final || intro)
}
