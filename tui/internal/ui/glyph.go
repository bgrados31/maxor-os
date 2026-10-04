// Package ui es el vocabulario visual de la pantalla de Maxor: los mismos glifos,
// colores y estados que usa la CLI de bash, para que se vea como un solo producto.
package ui

import (
	"os"
	"strings"
)

// Glyphs es el conjunto de símbolos; hay uno Unicode y uno ASCII de respaldo.
type Glyphs struct {
	Top, Bar, End, Tee             string
	OK, Ask, Tick, Warn, Bad, Info string
	Sel, On, Off, Dot              string
	Up, Add, Del, Chg              string
	BarOn, BarOff, Find, Arrow     string
	Ellipsis                       string
	Swatch, Branch                 string // punto de color y símbolo de rama
	Fill, Free, Edge               string // celda ocupada y libre de un mapa de disco (media altura: dos mapas seguidos no se pegan), y borde del foco
	Spin                           []string
}

var unicode = Glyphs{
	Top: "┌", Bar: "│", End: "└", Tee: "├",
	OK: "◇", Ask: "◆", Tick: "✓", Warn: "!", Bad: "✗", Info: "·",
	Sel: "❯", On: "◼", Off: "◻", Dot: "•",
	Up: "↑", Add: "+", Del: "−", Chg: "~",
	BarOn: "▰", BarOff: "▱", Find: "⌕", Arrow: "→",
	Ellipsis: "…",
	Swatch:   "●", Branch: "⎇",
	Fill:     "▄", Free: "▄", Edge: "▌",
	Spin:     []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
}

var ascii = Glyphs{
	Top: "+", Bar: "|", End: "+", Tee: "+",
	OK: "o", Ask: "*", Tick: "v", Warn: "!", Bad: "x", Info: "-",
	Sel: ">", On: "[x]", Off: "[ ]", Dot: "*",
	Up: "^", Add: "+", Del: "-", Chg: "~",
	BarOn: "#", BarOff: "-", Find: "?", Arrow: "->",
	Ellipsis: "...",
	Swatch:   "*", Branch: "branch",
	Fill:     "#", Free: ".", Edge: "|",
	Spin:     []string{"|", "/", "-", "\\"},
}

// G es el conjunto activo, elegido al arrancar con la misma regla que la CLI:
// ASCII con MAXOR_ASCII=1, TERM=linux o un locale que no es UTF-8.
var G = Detect(os.Getenv)

// Detect elige los glifos según el entorno.
func Detect(getenv func(string) string) Glyphs {
	if getenv("MAXOR_ASCII") == "1" || getenv("TERM") == "linux" {
		return ascii
	}
	loc := ""
	for _, k := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := getenv(k); v != "" {
			loc = strings.ToLower(v)
			break
		}
	}
	if loc != "" && !strings.Contains(loc, "utf-8") && !strings.Contains(loc, "utf8") {
		return ascii
	}
	return unicode
}
