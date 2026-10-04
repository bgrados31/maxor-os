package install

import (
	"os"
	"strings"
	"testing"
)

func TestEveryLayoutHasWhatTheSystemNeeds(t *testing.T) {
	seen := map[string]bool{}
	for _, l := range Layouts {
		if l.Name == "" || l.XKB == "" || l.Console == "" {
			t.Errorf("incomplete layout: %+v", l)
		}
		key := l.XKB + "/" + l.Variant
		if seen[key] {
			t.Errorf("repeated layout %s", key)
		}
		seen[key] = true
	}
}

func TestFindLayoutFallsBackToTheFirst(t *testing.T) {
	if l := FindLayout("latam", ""); l.Console != "la-latin1" {
		t.Fatalf("latam: %+v", l)
	}
	if l := FindLayout("us", "dvorak"); l.Console != "dvorak" {
		t.Fatalf("dvorak: %+v", l)
	}
	if l := FindLayout("zz", ""); l != Layouts[0] {
		t.Fatalf("unknown falls back: %+v", l)
	}
}

func TestEveryLocaleIsUTF8AndNamed(t *testing.T) {
	for _, l := range Locales {
		if !strings.HasSuffix(l.Code, ".UTF-8") || l.Name == "" {
			t.Errorf("bad locale %+v", l)
		}
	}
}

func TestParseZonesKeepsOnlyZoneNames(t *testing.T) {
	if z := parseZones("US/Pacific\nWET\nEtc/GMT+5\nUniversal\nAmerica/Lima\n"); len(z) != 1 || z[0] != "America/Lima" {
		t.Fatalf("legacy aliases are dropped: %v", z)
	}
	z := parseZones("Europe/Madrid\n\nAmerica/Lima\nnot a zone\nUTC\n")
	if len(z) != 3 || z[0] != "America/Lima" || z[2] != "UTC" {
		t.Fatalf("%v", z)
	}
}

func TestFilterMatchesEveryWordInAnyOrder(t *testing.T) {
	got := Filter(Layouts, "spanish lat", func(l Layout) string { return l.Name })
	if len(got) != 1 || got[0].XKB != "latam" {
		t.Fatalf("%+v", got)
	}
	if len(Filter(Layouts, "", func(l Layout) string { return l.Name })) != len(Layouts) {
		t.Fatal("an empty query keeps everything")
	}
	if len(Filter(Layouts, "zzzz", func(l Layout) string { return l.Name })) != 0 {
		t.Fatal("no match")
	}
}

func TestUserHintMakesAValidAccountName(t *testing.T) {
	for in, want := range map[string]string{"Ana Pérez": "ana", "José Núñez": "jose", "  Bryan  Grados": "bryan", "": "", "12 Monos": "u12"} {
		got := UserHint(in)
		if got != want {
			t.Errorf("%q → %q, want %q", in, got, want)
		}
		if got != "" && ValidateUsername(got) != nil {
			t.Errorf("%q → %q is not a valid account name", in, got)
		}
	}
	if HostHint("ana") != "ana-pc" || HostHint("") != "maxor" {
		t.Fatal("host hint")
	}
}

func TestParseStatusPrefersTheWire(t *testing.T) {
	st := ParseStatus("wifi:connected:Casa\nethernet:connected:Wired connection 1\nloopback:unmanaged:\n", "full")
	if !st.Online || st.Kind != "ethernet" || st.Name != "Wired connection 1" {
		t.Fatalf("%+v", st)
	}
	st = ParseStatus("wifi:connected:Casa\n", "limited")
	if st.Online || st.Kind != "wifi" || st.Name != "Casa" {
		t.Fatalf("%+v", st)
	}
	if st := ParseStatus("ethernet:connected:Wired\n", "unknown"); !st.Online {
		t.Fatalf("a connected wire with no connectivity check counts as online: %+v", st)
	}
	if st := ParseStatus("ethernet:unavailable:\n", "none"); st.Online || st.Kind != "" {
		t.Fatalf("%+v", st)
	}
}

func TestParseAPsDeduplicatesAndSorts(t *testing.T) {
	aps := ParseAPs("*:Casa:70:WPA2\n :Casa:80:WPA2\n :Cafe\\: Wi\\:Fi:60:\n :Vecino:90:WPA2\n : :50:WPA2\n")
	if len(aps) != 3 {
		t.Fatalf("%+v", aps)
	}
	if !aps[0].InUse || aps[0].SSID != "Casa" {
		t.Fatalf("the one in use comes first: %+v", aps)
	}
	if aps[1].SSID != "Vecino" || aps[2].SSID != "Cafe: Wi:Fi" || !aps[2].Open() || aps[1].Open() {
		t.Fatalf("by signal, with escapes honoured: %+v", aps)
	}
}

func TestSplitTerseHonoursEscapes(t *testing.T) {
	f := splitTerse(`a\:b:c\\d:e`)
	if len(f) != 3 || f[0] != "a:b" || f[1] != `c\d` || f[2] != "e" {
		t.Fatalf("%q", f)
	}
}

func TestMergeCatalogAddsAfterTheCuratedListsWithoutRepeating(t *testing.T) {
	savedL, savedY := Locales, Layouts
	defer func() { Locales, Layouts = savedL, savedY }()
	nl, ny := len(Locales), len(Layouts)
	err := MergeCatalog([]byte(`{"version":1,
	  "locales":[{"code":"es_PE.UTF-8","name":"Spanish (Peru)"},{"code":"aa_ER.UTF-8","name":"Afar (Eritrea)"}],
	  "layouts":[{"xkb":"es","variant":"","name":"Spanish"},{"xkb":"latam","variant":"deadtilde","name":"Spanish (Latin American, dead tilde)"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(Locales) != nl+1 || Locales[len(Locales)-1].Code != "aa_ER.UTF-8" {
		t.Fatalf("only the locale that was not there is added, at the end: %d → %d", nl, len(Locales))
	}
	if Locales[5].Name != "Español (Perú)" {
		t.Fatalf("the curated name stays: %q", Locales[5].Name)
	}
	if len(Layouts) != ny+1 {
		t.Fatalf("only the layout that was not there is added: %d → %d", ny, len(Layouts))
	}
	got := FindLayout("latam", "deadtilde")
	if got.Console != "" || got.Name == "" {
		t.Fatalf("a catalog layout has no console keymap of its own: %+v", got)
	}
}

func TestMergeCatalogRefusesNonsense(t *testing.T) {
	for _, bad := range []string{`not json`, `{}`, `{"locales":[],"layouts":[]}`} {
		if err := MergeCatalog([]byte(bad)); err == nil {
			t.Fatalf("%q should be refused", bad)
		}
	}
}

func TestLoadCatalogFromEnvIsOptional(t *testing.T) {
	t.Setenv("MAXOR_CATALOG", "")
	if err := LoadCatalogFromEnv(); err != nil {
		t.Fatalf("no catalog is not an error: %v", err)
	}
	t.Setenv("MAXOR_CATALOG", "/nonexistent/catalog.json")
	if err := LoadCatalogFromEnv(); err == nil {
		t.Fatal("a catalog that was asked for and cannot be read is reported")
	}
}

func TestCheckZoneOnlyAcceptsKnownZones(t *testing.T) {
	known := []string{"America/Lima", "Europe/Madrid"}
	if z, err := CheckZone("America/Lima", known); err != nil || z != "America/Lima" {
		t.Fatalf("%q %v", z, err)
	}
	for _, bad := range []string{"", "Mars/Olympus", "America/Lima\nrm -rf /", "<html>blocked</html>"} {
		if _, err := CheckZone(bad, known); err == nil {
			t.Fatalf("%q should be refused", bad)
		}
	}
}

func TestFilterIgnoresAccentsAndCase(t *testing.T) {
	items := []string{"Español (Perú)", "Português (Brasil)", "Deutsch"}
	got := Filter(items, "peru", func(s string) string { return s })
	if len(got) != 1 || got[0] != "Español (Perú)" {
		t.Fatalf("%v", got)
	}
	if got := Filter(items, "ESPANOL", func(s string) string { return s }); len(got) != 1 {
		t.Fatalf("%v", got)
	}
	if got := Filter(items, "portugues brasil", func(s string) string { return s }); len(got) != 1 {
		t.Fatalf("%v", got)
	}
}

func TestApplyKeyboardTalksToSwayOnTheInstallationMedium(t *testing.T) {
	dir := t.TempDir()
	out := dir + "/args"
	script := "#!/bin/sh\necho \"$@\" > " + out + "\n"
	if err := os.WriteFile(dir+"/swaymsg", []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("SWAYSOCK", dir+"/sock")
	t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "")
	applyKeyboard(Layout{Name: "Spanish (Latin America)", XKB: "latam", Variant: "deadtilde"})
	got, _ := os.ReadFile(out)
	if strings.TrimSpace(string(got)) != "input type:keyboard xkb_layout latam xkb_variant deadtilde" {
		t.Fatalf("swaymsg was called with %q", got)
	}
}

func TestApplyKeyboardDoesNothingWithoutASession(t *testing.T) {
	t.Setenv("SWAYSOCK", "")
	t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "")
	applyKeyboard(Layout{XKB: "es"}) // must not panic or run anything
}

func TestNameFieldsTurnWhatIsTypedIntoAValidName(t *testing.T) {
	typed := func(s string, f func(rune) (rune, bool)) string {
		var b strings.Builder
		for _, r := range s {
			if c, ok := f(r); ok {
				b.WriteRune(c)
			}
		}
		return b.String()
	}
	if got := typed("Ana Pérez_1!", UserRune); got != "ana-perez_1" {
		t.Fatalf("user: %q", got)
	}
	if got := typed("Ana_PC.local", HostRune); got != "anapclocal" {
		t.Fatalf("host: %q", got)
	}
	if got := typed("1a6", DigitRune); got != "16" {
		t.Fatalf("digits: %q", got)
	}
}
