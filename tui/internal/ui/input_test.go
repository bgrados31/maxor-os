package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func runes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func TestInputEscribeYBorra(t *testing.T) {
	var in Input
	in.Key(runes("brave"))
	in.Key(tea.KeyMsg{Type: tea.KeySpace})
	in.Key(runes("ñu"))
	if in.Text() != "brave ñu" {
		t.Fatalf("texto: %q", in.Text())
	}
	in.Key(tea.KeyMsg{Type: tea.KeyBackspace})
	in.Key(tea.KeyMsg{Type: tea.KeyBackspace})
	if in.Text() != "brave " {
		t.Fatalf("borrado: %q", in.Text())
	}
}

func TestInputMueveElCursorEInsertaEnMedio(t *testing.T) {
	var in Input
	in.Key(runes("bve"))
	in.Key(tea.KeyMsg{Type: tea.KeyLeft})
	in.Key(tea.KeyMsg{Type: tea.KeyLeft})
	in.Key(runes("ra"))
	if in.Text() != "brave" {
		t.Fatalf("inserción: %q", in.Text())
	}
	in.Key(tea.KeyMsg{Type: tea.KeyHome})
	in.Key(tea.KeyMsg{Type: tea.KeyDelete})
	if in.Text() != "rave" {
		t.Fatalf("suprimir: %q", in.Text())
	}
}

func TestInputAtajos(t *testing.T) {
	var in Input
	in.Set("uno dos tres")
	in.Key(tea.KeyMsg{Type: tea.KeyCtrlW})
	if in.Text() != "uno dos " {
		t.Fatalf("ctrl+w: %q", in.Text())
	}
	in.Key(tea.KeyMsg{Type: tea.KeyCtrlU})
	if in.Text() != "" {
		t.Fatalf("ctrl+u: %q", in.Text())
	}
}

func TestInputIgnoraControlYSaltosDeLinea(t *testing.T) {
	var in Input
	in.Key(runes("a\nb\x00c\x7f"))
	if in.Text() != "abc" {
		t.Fatalf("limpieza al pegar: %q", in.Text())
	}
}

func TestInputNoPasaDelAnchoYMuestraElCursor(t *testing.T) {
	p := painter()
	var in Input
	in.Set("una busqueda bastante larga que no cabe")
	segs := in.Segs(p, true, 12)
	w := 0
	for _, s := range segs {
		w += ansi.StringWidth(s.T)
	}
	if w > 12 {
		t.Fatalf("el campo mide %d, máximo 12", w)
	}
	in2 := Input{Placeholder: "Search apps…"}
	if segs := in2.Segs(p, false, 30); segs[0].T != "Search apps…" {
		t.Fatal("sin foco ni texto muestra la ayuda")
	}
}

func TestInputSinTeclasPropiasNoLasConsume(t *testing.T) {
	var in Input
	if _, handled := in.Key(tea.KeyMsg{Type: tea.KeyEnter}); handled {
		t.Fatal("Intro no es del campo")
	}
	if _, handled := in.Key(tea.KeyMsg{Type: tea.KeyEsc}); handled {
		t.Fatal("Esc no es del campo")
	}
}
