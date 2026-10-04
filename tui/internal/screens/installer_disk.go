package screens

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/bgrados31/maxor-os/tui/internal/core"
	"github.com/bgrados31/maxor-os/tui/internal/install"
	"github.com/bgrados31/maxor-os/tui/internal/ui"
)

// The disk map: a disk drawn to scale as one row of cells, each partition in the colour of what it is, with a
// legend that names them. The disk, strategy and storage steps share it, so a person always sees what is on a
// disk and what the installation will do to it before anything is written.

// spanColor is the colour of a part of the disk: Maxor OS in the accent, Windows in the second accent, the rest
// quieter, and free space barely there.
func spanColor(env *core.Env, s install.Span) string {
	t := env.Theme.P
	switch s.Kind {
	case "maxor":
		return t.Ac
	case "windows":
		return t.Ac2
	case "efi":
		if s.New {
			return ui.Mix(t.Ac, t.Mu, 0.55)
		}
		return ui.Mix(t.Mu, t.S2, 0.35)
	case "linux":
		return ui.Mix(t.Fg, t.Mu, 0.55)
	case "free":
		return ui.Mix(t.S, t.Mu, 0.45)
	}
	return ui.Mix(t.Mu, t.S2, 0.15)
}

// diskBar draws the parts of a disk in width cells, to scale (install.Cells).
func diskBar(env *core.Env, spans []install.Span, width int) []ui.Seg {
	cells := install.Cells(spans, width)
	segs := make([]ui.Seg, 0, len(spans))
	for i, s := range spans {
		if cells[i] == 0 {
			continue
		}
		glyph := ui.G.Fill
		if s.Kind == "free" {
			glyph = ui.G.Free
		}
		segs = append(segs, ui.S(env.P.Fill.Foreground(lipgloss.Color(spanColor(env, s))), strings.Repeat(glyph, cells[i])))
	}
	return segs
}

// diskLegend names the parts under the bar («● Windows 200 GB   ● free 307 GB»), merging parts of the same kind and
// name, and wraps to width. New parts say so.
func diskLegend(env *core.Env, spans []install.Span, width int) []ui.Line {
	p := env.P
	type item struct {
		span  install.Span
		bytes int64
	}
	var items []item
	seen := map[string]int{}
	for _, s := range spans {
		k := s.Kind + "|" + s.Label
		if s.New {
			k += "|new"
		}
		if i, ok := seen[k]; ok {
			items[i].bytes += s.Bytes
			continue
		}
		seen[k] = len(items)
		items = append(items, item{s, s.Bytes})
	}
	var lines []ui.Line
	var cur []ui.Seg
	used := 0
	for _, it := range items {
		dot := ui.G.Swatch
		name := it.span.Label
		if it.span.New && it.span.Kind == "efi" {
			name = "boot"
		}
		size := install.HumanSize(it.bytes)
		text := name + " " + size
		w := ansi.StringWidth(dot) + 1 + ansi.StringWidth(text)
		if used > 0 && used+3+w > width {
			lines = append(lines, ui.Of(cur...))
			cur, used = nil, 0
		}
		if used > 0 {
			cur = append(cur, ui.S(p.Mu, "   "))
			used += 3
		}
		nameStyle := p.Text
		if it.span.New {
			nameStyle = p.Text.Bold(true)
		}
		cur = append(cur,
			ui.S(p.Fill.Foreground(lipgloss.Color(spanColor(env, it.span))), dot+" "),
			ui.S(nameStyle, name+" "), ui.S(p.Mu, size))
		used += w
	}
	if len(cur) > 0 {
		lines = append(lines, ui.Of(cur...))
	}
	return lines
}

// diskMap is the bar with its legend under it, indented by indent cells.
func diskMap(env *core.Env, spans []install.Span, width, indent int) []ui.Line {
	pad := strings.Repeat(" ", indent)
	lines := []ui.Line{ui.Of(append([]ui.Seg{ui.S(env.P.Mu, pad)}, diskBar(env, spans, width-indent)...)...)}
	for _, l := range diskLegend(env, spans, width-indent) {
		lines = append(lines, ui.Of(append([]ui.Seg{ui.S(env.P.Mu, pad)}, l.L...)...))
	}
	return lines
}
