package theme

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// luminance is the relative luminance of #rrggbb, as WCAG defines it.
func luminance(hex string) float64 {
	var c [3]float64
	for i := range c {
		v, _ := strconv.ParseUint(hex[1+2*i:3+2*i], 16, 8)
		s := float64(v) / 255
		if s <= 0.03928 {
			c[i] = s / 12.92
		} else {
			c[i] = math.Pow((s+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*c[0] + 0.7152*c[1] + 0.0722*c[2]
}

func contrast(a, b string) float64 {
	x, y := luminance(a), luminance(b)
	if x < y {
		x, y = y, x
	}
	return (x + 0.05) / (y + 0.05)
}

// The two Maxor themes are what everyone sees first: every pair of colours a screen puts together reads at 5:1 or
// more (WCAG AA asks 4.5), on the background, on both surfaces and on the accent.
func TestTheMaxorThemesReadWellEverywhere(t *testing.T) {
	for _, id := range []string{"maxor-dark", "maxor-light"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "themes", id, "colors.json"))
		if err != nil {
			t.Fatal(err)
		}
		var p Palette
		if err := json.Unmarshal(raw, &p); err != nil || !p.Valid() {
			t.Fatalf("%s: colors.json is not a valid palette (%v)", id, err)
		}
		pairs := []struct{ name, fg, bg string }{
			{"text on background", p.Fg, p.Bg}, {"text on surface 2", p.Fg, p.S2},
			{"muted on background", p.Mu, p.Bg}, {"muted on surface", p.Mu, p.S}, {"muted on surface 2", p.Mu, p.S2},
			{"accent on background", p.Ac, p.Bg}, {"accent on surface", p.Ac, p.S}, {"accent on surface 2", p.Ac, p.S2},
			{"text on accent", p.On, p.Ac},
		}
		for _, pr := range pairs {
			if c := contrast(pr.fg, pr.bg); c < 5 {
				t.Errorf("%s: %s is %.2f:1 (%s on %s), want 5:1 or more", id, pr.name, c, pr.fg, pr.bg)
			}
		}
	}
}

// The built-in default (for screens without an applied theme, like the installer) is Maxor Dark itself.
func TestTheDefaultIsMaxorDark(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "themes", "maxor-dark", "colors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p Palette
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	d := Default().P
	p.Mode, d.Mode = "", ""
	if p != d {
		t.Fatalf("Default() = %+v, themes/maxor-dark says %+v", d, p)
	}
}
