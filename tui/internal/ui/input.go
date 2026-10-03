package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Input es un campo de texto de una línea: inserta (también pegando), borra,
// mueve el cursor y reconoce los atajos de siempre (ctrl+a/e/u/w).
type Input struct {
	Value       []rune
	Pos         int
	Placeholder string
	Mask        bool // dibuja puntos en vez del texto (contraseñas)
}

// Text devuelve el contenido.
func (in *Input) Text() string { return string(in.Value) }

// Set cambia el contenido y deja el cursor al final.
func (in *Input) Set(s string) { in.Value = []rune(s); in.Pos = len(in.Value) }

// Key procesa una tecla. Devuelve si cambió el texto y si la tecla era del campo.
func (in *Input) Key(msg tea.KeyMsg) (changed, handled bool) {
	switch msg.Type {
	case tea.KeyRunes:
		in.insert(msg.Runes)
		return true, true
	case tea.KeySpace:
		in.insert([]rune{' '})
		return true, true
	case tea.KeyBackspace:
		if in.Pos > 0 {
			in.Value = append(in.Value[:in.Pos-1], in.Value[in.Pos:]...)
			in.Pos--
			return true, true
		}
		return false, true
	case tea.KeyDelete:
		if in.Pos < len(in.Value) {
			in.Value = append(in.Value[:in.Pos], in.Value[in.Pos+1:]...)
			return true, true
		}
		return false, true
	case tea.KeyLeft:
		if in.Pos > 0 {
			in.Pos--
		}
		return false, true
	case tea.KeyRight:
		if in.Pos < len(in.Value) {
			in.Pos++
		}
		return false, true
	case tea.KeyHome, tea.KeyCtrlA:
		in.Pos = 0
		return false, true
	case tea.KeyEnd, tea.KeyCtrlE:
		in.Pos = len(in.Value)
		return false, true
	case tea.KeyCtrlU:
		in.Value = in.Value[in.Pos:]
		in.Pos = 0
		return true, true
	case tea.KeyCtrlW:
		i := in.Pos
		for i > 0 && in.Value[i-1] == ' ' {
			i--
		}
		for i > 0 && in.Value[i-1] != ' ' {
			i--
		}
		in.Value = append(in.Value[:i], in.Value[in.Pos:]...)
		in.Pos = i
		return true, true
	}
	return false, false
}

func (in *Input) insert(r []rune) {
	clean := make([]rune, 0, len(r))
	for _, c := range r {
		if c >= ' ' && c != 0x7f { // sin saltos de línea ni caracteres de control
			clean = append(clean, c)
		}
	}
	in.Value = append(in.Value[:in.Pos], append(clean, in.Value[in.Pos:]...)...)
	in.Pos += len(clean)
}

// Segs dibuja el campo en width celdas como máximo: el texto, con el cursor si
// tiene el foco, o el texto de ayuda si está vacío.
func (in *Input) Segs(p Painter, focused bool, width int) []Seg {
	if len(in.Value) == 0 && !focused {
		return []Seg{{T: in.Placeholder, S: p.Mu}}
	}
	// Ventana horizontal: se desplaza para que el cursor quede a la vista.
	start := 0
	if width > 2 && in.Pos > width-2 {
		start = in.Pos - (width - 2)
	}
	end := len(in.Value)
	if width > 2 && end-start > width-1 {
		end = start + width - 1
	}
	show := in.Value
	if in.Mask {
		dot := '•'
		if G.BarOn == "#" {
			dot = '*'
		}
		show = []rune(strings.Repeat(string(dot), len(in.Value)))
	}
	before := string(show[start:min(in.Pos, end)])
	after := ""
	if in.Pos < end {
		after = string(show[in.Pos:end])
	}
	segs := []Seg{{T: before, S: p.Text}}
	if focused {
		segs = append(segs, Seg{T: "▏", S: p.Ac})
		if G.BarOn == "#" {
			segs[len(segs)-1].T = "_"
		}
	}
	segs = append(segs, Seg{T: after, S: p.Text})
	if len(in.Value) == 0 && focused && in.Placeholder != "" {
		segs = append(segs, Seg{T: " " + strings.TrimSpace(in.Placeholder), S: p.Mu})
	}
	return segs
}
