package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/bgrados31/maxor-os/tui/internal/theme"
)

func painter() Painter { return NewPainter(theme.Default(), theme.Default().P.S) }

func TestLineRenderEsSiempreDelAnchoPedido(t *testing.T) {
	p := painter()
	c := p.Ctx()
	casos := []Line{
		T(p.Text, "hola"),
		T(p.Text, "una línea mucho más larga que el ancho disponible para ver el recorte"),
		Of(S(p.Ac, "◇ "), S(p.Text, "漢字 anchos")),
		{L: []Seg{S(p.Text, "izquierda")}, R: []Seg{S(p.Mu, "derecha")}},
		{L: []Seg{S(p.Text, "izquierda larguísima que no cabe junto a la derecha")}, R: []Seg{S(p.Mu, "derecha")}},
		{R: []Seg{S(p.Mu, "solo derecha")}},
		Blank(),
	}
	for _, w := range []int{1, 2, 5, 10, 20, 40, 80} {
		for i, ln := range casos {
			for _, sel := range []bool{false, true} {
				ln.Sel = sel
				got := ln.Render(w, c)
				if gw := ansi.StringWidth(got); gw != w {
					t.Errorf("caso %d ancho %d sel=%v: salió de %d celdas: %q", i, w, sel, gw, ansi.Strip(got))
				}
			}
		}
	}
}

func TestLineRenderRecortaConPuntosSuspensivos(t *testing.T) {
	p := painter()
	got := ansi.Strip(T(p.Text, "abcdefghijklmnop").Render(8, p.Ctx()))
	if !strings.HasSuffix(got, G.Ellipsis) || ansi.StringWidth(got) != 8 {
		t.Fatalf("recorte incorrecto: %q", got)
	}
}

func TestLineRenderConservaLaDerechaCuandoLaIzquierdaNoCabe(t *testing.T) {
	p := painter()
	ln := Line{L: []Seg{S(p.Text, "nombre-muy-largo-de-una-aplicacion")}, R: []Seg{S(p.Mu, "nixpkgs")}}
	got := ansi.Strip(ln.Render(24, p.Ctx()))
	if !strings.HasSuffix(got, "nixpkgs") || ansi.StringWidth(got) != 24 {
		t.Fatalf("la derecha debe quedar pegada al borde: %q", got)
	}
}

func TestBlockTieneLasDimensionesExactas(t *testing.T) {
	p := painter()
	lines := []Line{T(p.Text, "uno"), T(p.Text, "dos"), T(p.Text, "tres"), T(p.Text, "cuatro")}
	for _, h := range []int{0, 1, 3, 4, 9} {
		rows := Block(lines, 12, h, p.Ctx())
		if len(rows) != h {
			t.Fatalf("alto %d: salieron %d filas", h, len(rows))
		}
		for _, r := range rows {
			if ansi.StringWidth(r) != 12 {
				t.Fatalf("fila de %d celdas: %q", ansi.StringWidth(r), ansi.Strip(r))
			}
		}
	}
}

func TestInsertAgregaMargenYFilasArriba(t *testing.T) {
	p := painter()
	rows := Block(Inset([]Line{T(p.Text, "x")}, 2, 1), 6, 3, p.Ctx())
	if ansi.Strip(rows[0]) != "      " || ansi.Strip(rows[1]) != "  x   " {
		t.Fatalf("margen incorrecto: %q", ansi.Strip(strings.Join(rows, "|")))
	}
}

func TestJoinHUneColumnasPorFila(t *testing.T) {
	got := JoinH("|", []string{"a", "b"}, []string{"c", "d"})
	if got[0] != "a|c" || got[1] != "b|d" {
		t.Fatalf("unión incorrecta: %v", got)
	}
}

func TestDetectEligeAsciiEnTerminalesLimitadas(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	if Detect(env(map[string]string{"MAXOR_ASCII": "1"})).Top != "+" {
		t.Error("MAXOR_ASCII=1 debe dar ASCII")
	}
	if Detect(env(map[string]string{"TERM": "linux"})).Top != "+" {
		t.Error("TERM=linux debe dar ASCII")
	}
	if Detect(env(map[string]string{"LANG": "C"})).Top != "+" {
		t.Error("un locale que no es UTF-8 debe dar ASCII")
	}
	if Detect(env(map[string]string{"LANG": "es_PE.UTF-8"})).Top != "┌" {
		t.Error("UTF-8 debe dar Unicode")
	}
	if Detect(env(map[string]string{})).Top != "┌" {
		t.Error("sin locale se asume Unicode")
	}
	if Detect(env(map[string]string{"LC_ALL": "C", "LANG": "es_PE.UTF-8"})).Top != "+" {
		t.Error("LC_ALL manda sobre LANG")
	}
}

func TestSpinDaVueltaYAceptaNegativos(t *testing.T) {
	n := len(G.Spin)
	if Spin(0) != Spin(n) || Spin(-1) != Spin(n-1) {
		t.Fatal("el spinner debe ser cíclico")
	}
}

func TestBarLlenaLaProporcion(t *testing.T) {
	p := painter()
	segs := Bar(p, 50, 10)
	if strings.Count(segs[0].T, G.BarOn) != 5 || strings.Count(segs[1].T, G.BarOff) != 5 {
		t.Fatalf("barra incorrecta: %+v", segs)
	}
	if Bar(p, 150, 10)[0].T != strings.Repeat(G.BarOn, 10) || Bar(p, -5, 10)[0].T != "" {
		t.Fatal("la barra debe acotar el porcentaje")
	}
}

func TestDurationComoLaCLI(t *testing.T) {
	min := 2 * time.Second
	if Duration(time.Second, min) != "" {
		t.Error("lo breve no lleva tiempo")
	}
	if Duration(3*time.Second, min) != "3s" || Duration(75*time.Second, min) != "1m15s" {
		t.Errorf("formato: %q %q", Duration(3*time.Second, min), Duration(75*time.Second, min))
	}
}

func TestSkeletonTieneUnBrilloQueSeMueve(t *testing.T) {
	p := painter()
	a := Skeleton(p, 0, 12, 20)
	b := Skeleton(p, 6, 12, 20)
	if len(a.L) == 0 || len(a.L) != len(b.L) {
		t.Fatal("mismo número de celdas en cada fotograma")
	}
	color := func(l Line) string {
		var out []string
		for _, sg := range l.L {
			out = append(out, fmt.Sprint(sg.S.GetForeground()))
		}
		return strings.Join(out, ",")
	}
	if color(a) == color(b) {
		t.Fatal("el brillo debe cambiar de sitio entre fotogramas")
	}
	w := 0
	for _, sg := range a.L {
		w += ansi.StringWidth(sg.T)
	}
	if w != 12+2+20 {
		t.Fatalf("ancho total %d, se esperaba 34", w)
	}
}

func TestMixMezclaColores(t *testing.T) {
	if Mix("#000000", "#ffffff", 0.5) != "#808080" || Mix("#ff0000", "#0000ff", 0) != "#ff0000" || Mix("#ff0000", "#0000ff", 1) != "#0000ff" {
		t.Fatal("mezcla incorrecta")
	}
}

func TestSpreadColocaLosDosLadosEnElAncho(t *testing.T) {
	p := painter()
	segs := Spread([]Seg{S(p.Text, "izquierda")}, []Seg{S(p.Mu, "derecha")}, 30, p.Fill)
	w := 0
	var txt string
	for _, sg := range segs {
		w += ansi.StringWidth(sg.T)
		txt += sg.T
	}
	if w != 30 || !strings.HasPrefix(txt, "izquierda") || !strings.HasSuffix(txt, "derecha") {
		t.Fatalf("spread: %d %q", w, txt)
	}
	segs = Spread([]Seg{S(p.Text, "un nombre larguísimo que no cabe")}, []Seg{S(p.Mu, "nixpkgs")}, 20, p.Fill)
	txt, w = "", 0
	for _, sg := range segs {
		w += ansi.StringWidth(sg.T)
		txt += sg.T
	}
	if w != 20 || !strings.HasSuffix(txt, "nixpkgs") {
		t.Fatalf("spread recortado: %d %q", w, txt)
	}
}

func TestPainterLevelDevuelveGlifoPorNivel(t *testing.T) {
	p := painter()
	for lvl, want := range map[string]string{"ok": G.Tick, "warn": G.Warn, "bad": G.Bad, "info": G.Info, "x": G.Info} {
		if _, g := p.Level(lvl); g != want {
			t.Errorf("%s: %q", lvl, g)
		}
	}
}

var _ = lipgloss.NewStyle

func TestWrapKeepsWordsAndCutsOnlyWhatCannotFit(t *testing.T) {
	got := Wrap("copying path '/nix/store/3zvg83mg9aavm9bgh26ydljchply7i25-source' from 'auto'", 20)
	want := []string{"copying path", "'/nix/store/3zvg83mg", "9aavm9bgh26ydljchply", "7i25-source' from", "'auto'"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("Wrap = %q, want %q", got, want)
	}
	for _, l := range got {
		if ansi.StringWidth(l) > 20 {
			t.Fatalf("row %q is wider than 20 cells", l)
		}
	}
}
