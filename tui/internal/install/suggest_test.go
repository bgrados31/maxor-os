package install

import "testing"

func TestTheLanguageSuggestsAKeyboardAndATimeZone(t *testing.T) {
	for _, c := range []struct{ locale, xkb, zone string }{
		{"es_PE.UTF-8", "latam", "America/Lima"},
		{"es_ES.UTF-8", "es", "Europe/Madrid"},
		{"pt_BR.UTF-8", "br", "America/Sao_Paulo"},
		{"en_GB.UTF-8", "gb", "Europe/London"},
	} {
		l, ok := SuggestLayout(c.locale)
		if !ok || l.XKB != c.xkb {
			t.Errorf("SuggestLayout(%s) = %q, %v; want %q", c.locale, l.XKB, ok, c.xkb)
		}
		if z, ok := SuggestZone(c.locale); !ok || z != c.zone {
			t.Errorf("SuggestZone(%s) = %q, %v; want %q", c.locale, z, ok, c.zone)
		}
	}
	if _, ok := SuggestZone("en_US.UTF-8"); ok {
		t.Error("the United States span several zones: no guess")
	}
}
