package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Progress is how far a transition of length total has gone after elapsed: 0 at the start, 1 once it is over.
func Progress(elapsed, total time.Duration) float64 {
	if total <= 0 || elapsed >= total {
		return 1
	}
	if elapsed <= 0 {
		return 0
	}
	return float64(elapsed) / float64(total)
}

// EaseOut is a cubic ease-out: it moves fast at first and settles softly, which is what makes motion in a terminal,
// with its coarse cells, read as smooth.
func EaseOut(t float64) float64 {
	t = min(max(t, 0), 1)
	u := 1 - t
	return 1 - u*u*u
}

// Lerp goes from a to b as t goes from 0 to 1, rounded to whole cells.
func Lerp(a, b int, t float64) int {
	return a + int(float64(b-a)*t+0.5*sign(b-a))
}

func sign(n int) float64 {
	if n < 0 {
		return -1
	}
	return 1
}

// Fade brings the lines in from the background: t=0 draws them in the colour of bg (invisible), t=1 as they are.
// Every piece is mixed, the selected row too (see FadeCtx), so a fade never pops.
func Fade(lines []Line, bg string, t float64) []Line {
	if t >= 1 {
		return lines
	}
	out := make([]Line, len(lines))
	for i, ln := range lines {
		out[i] = Line{L: fadeSegs(ln.L, bg, t), R: fadeSegs(ln.R, bg, t), Sel: ln.Sel}
	}
	return out
}

// FadeCtx is the drawing context of faded lines: the highlight of the selected row fades in with them.
func FadeCtx(c Ctx, bg string, t float64) Ctx {
	if t >= 1 {
		return c
	}
	out := Ctx{Fill: c.Fill, Sel: fadeStyle(c.Sel, bg, t)}
	if c.SelBg != "" {
		out.SelBg = lipgloss.Color(Mix(bg, string(c.SelBg), t))
	}
	return out
}

func fadeSegs(segs []Seg, bg string, t float64) []Seg {
	if segs == nil {
		return nil
	}
	out := make([]Seg, len(segs))
	for i, s := range segs {
		out[i] = Seg{T: s.T, S: fadeStyle(s.S, bg, t)}
	}
	return out
}

func fadeStyle(s lipgloss.Style, bg string, t float64) lipgloss.Style {
	if c, ok := s.GetForeground().(lipgloss.Color); ok && isHex(string(c)) {
		s = s.Foreground(lipgloss.Color(Mix(bg, string(c), t)))
	}
	if c, ok := s.GetBackground().(lipgloss.Color); ok && isHex(string(c)) && !strings.EqualFold(string(c), bg) {
		s = s.Background(lipgloss.Color(Mix(bg, string(c), t)))
	}
	return s
}

func isHex(c string) bool { return len(c) == 7 && c[0] == '#' }
