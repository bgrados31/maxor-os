package install

import (
	"bufio"
	"context"
	"os/exec"
	"sort"
	"strings"
)

// Layout is a keyboard layout: how the desktop calls it (XKB), how the console calls it, and a friendly name.
type Layout struct {
	Name    string
	XKB     string
	Variant string
	Console string
}

// Layouts is the curated list the wizard offers; the console keymaps are the ones that match each XKB layout.
var Layouts = []Layout{
	{"English (US)", "us", "", "us"},
	{"English (UK)", "gb", "", "uk"},
	{"English (US, international)", "us", "intl", "us-acentos"},
	{"Spanish (Spain)", "es", "", "es"},
	{"Spanish (Latin America)", "latam", "", "la-latin1"},
	{"Portuguese (Portugal)", "pt", "", "pt-latin9"},
	{"Portuguese (Brazil)", "br", "", "br-abnt2"},
	{"French", "fr", "", "fr-latin9"},
	{"French (Canada)", "ca", "", "cf"},
	{"German", "de", "", "de-latin1-nodeadkeys"},
	{"German (Switzerland)", "ch", "de_nodeadkeys", "sg"},
	{"Italian", "it", "", "it"},
	{"Dutch", "nl", "", "nl"},
	{"Belgian", "be", "", "be-latin1"},
	{"Swedish", "se", "", "sv-latin1"},
	{"Norwegian", "no", "", "no"},
	{"Danish", "dk", "", "dk"},
	{"Finnish", "fi", "", "fi"},
	{"Polish", "pl", "", "pl2"},
	{"Czech", "cz", "", "cz-lat2"},
	{"Hungarian", "hu", "", "hu"},
	{"Romanian", "ro", "", "ro"},
	{"Greek", "gr", "", "gr"},
	{"Turkish", "tr", "", "trq"},
	{"Russian", "ru", "", "ru"},
	{"Ukrainian", "ua", "", "ua"},
	{"Hebrew", "il", "", "us"},
	{"Arabic", "ara", "", "us"},
	{"Japanese", "jp", "", "jp106"},
	{"Korean", "kr", "", "us"},
	{"Dvorak (US)", "us", "dvorak", "dvorak"},
	{"Colemak (US)", "us", "colemak", "colemak"},
}

// FindLayout returns the layout with this XKB layout and variant, or the first one if none matches.
func FindLayout(xkb, variant string) Layout {
	for _, l := range Layouts {
		if l.XKB == xkb && l.Variant == variant {
			return l
		}
	}
	return Layouts[0]
}

// Locale is a system language.
type Locale struct{ Code, Name string }

// Locales is the curated list of system languages.
var Locales = []Locale{
	{"en_US.UTF-8", "English (United States)"},
	{"en_GB.UTF-8", "English (United Kingdom)"},
	{"es_ES.UTF-8", "Español (España)"},
	{"es_MX.UTF-8", "Español (México)"},
	{"es_AR.UTF-8", "Español (Argentina)"},
	{"es_PE.UTF-8", "Español (Perú)"},
	{"es_CO.UTF-8", "Español (Colombia)"},
	{"pt_BR.UTF-8", "Português (Brasil)"},
	{"pt_PT.UTF-8", "Português (Portugal)"},
	{"fr_FR.UTF-8", "Français"},
	{"de_DE.UTF-8", "Deutsch"},
	{"it_IT.UTF-8", "Italiano"},
	{"nl_NL.UTF-8", "Nederlands"},
	{"pl_PL.UTF-8", "Polski"},
	{"ru_RU.UTF-8", "Русский"},
	{"tr_TR.UTF-8", "Türkçe"},
	{"ja_JP.UTF-8", "日本語"},
	{"ko_KR.UTF-8", "한국어"},
	{"zh_CN.UTF-8", "中文 (简体)"},
}

// fallbackZones is used when the system cannot list its time zones.
var fallbackZones = []string{
	"UTC", "America/Argentina/Buenos_Aires", "America/Bogota", "America/Caracas", "America/Chicago", "America/Denver",
	"America/Lima", "America/Los_Angeles", "America/Mexico_City", "America/New_York", "America/Santiago",
	"America/Sao_Paulo", "America/Toronto", "Asia/Dubai", "Asia/Kolkata", "Asia/Seoul", "Asia/Shanghai", "Asia/Tokyo",
	"Atlantic/Canary", "Australia/Sydney", "Europe/Amsterdam", "Europe/Berlin", "Europe/Lisbon", "Europe/London",
	"Europe/Madrid", "Europe/Moscow", "Europe/Paris", "Europe/Rome", "Europe/Stockholm", "Europe/Warsaw",
}


// Timezones lists the time zones of this system (`timedatectl list-timezones`), or a short list if that fails.
func Timezones(ctx context.Context) []string {
	out, err := exec.CommandContext(ctx, "timedatectl", "list-timezones").Output()
	if err == nil {
		if z := parseZones(string(out)); len(z) > 0 {
			return z
		}
	}
	return append([]string(nil), fallbackZones...)
}

// legacyZone: the aliases tzdata keeps for old software (US/Pacific, Etc/GMT+5, WET…) are noise in a list for people.
func legacyZone(z string) bool {
	if z == "UTC" {
		return false
	}
	for _, p := range []string{"US/", "Etc/", "SystemV/", "Brazil/", "Canada/", "Chile/", "Mexico/"} {
		if strings.HasPrefix(z, p) {
			return true
		}
	}
	return !strings.Contains(z, "/")
}

func parseZones(s string) []string {
	var zones []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		z := strings.TrimSpace(sc.Text())
		if z != "" && !strings.ContainsAny(z, " :") && !legacyZone(z) {
			zones = append(zones, z)
		}
	}
	sort.Strings(zones)
	return zones
}

// Filter keeps the items whose text contains every word of the query, ignoring case: the way every list
// of the wizard is searched by typing.
func Filter[T any](items []T, query string, text func(T) string) []T {
	words := strings.Fields(fold(query))
	if len(words) == 0 {
		return items
	}
	var out []T
next:
	for _, it := range items {
		t := fold(text(it))
		for _, w := range words {
			if !strings.Contains(t, w) {
				continue next
			}
		}
		out = append(out, it)
	}
	return out
}

// fold lowercases and drops the accents of Latin letters, so «peru» finds «Perú».
var folder = strings.NewReplacer(
	"á", "a", "à", "a", "ä", "a", "â", "a", "ã", "a", "å", "a", "é", "e", "è", "e", "ë", "e", "ê", "e",
	"í", "i", "ì", "i", "ï", "i", "î", "i", "ó", "o", "ò", "o", "ö", "o", "ô", "o", "õ", "o", "ø", "o",
	"ú", "u", "ù", "u", "ü", "u", "û", "u", "ñ", "n", "ç", "c", "ß", "ss",
)

func fold(s string) string { return folder.Replace(strings.ToLower(s)) }

// HostHint suggests a machine name from the account name: «ana» → «ana-pc».
func HostHint(user string) string {
	if user == "" {
		return "maxor"
	}
	return user + "-pc"
}

// UserHint suggests an account name from a full name: «Ana Pérez» → «ana».
func UserHint(full string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(full)) {
		if r == ' ' && b.Len() > 0 {
			break // only the first word
		}
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case strings.ContainsRune("áàäâ", r):
			b.WriteRune('a')
		case strings.ContainsRune("éèëê", r):
			b.WriteRune('e')
		case strings.ContainsRune("íìïî", r):
			b.WriteRune('i')
		case strings.ContainsRune("óòöô", r):
			b.WriteRune('o')
		case strings.ContainsRune("úùüû", r):
			b.WriteRune('u')
		case r == 'ñ':
			b.WriteRune('n')
		}
	}
	s := b.String()
	if s != "" && (s[0] < 'a' || s[0] > 'z') {
		s = "u" + s
	}
	if len(s) > 32 {
		s = s[:32]
	}
	return s
}


// LocaleName is the readable name of a locale code ("es_PE.UTF-8" → "Español (Perú)"), or the code itself.
func LocaleName(code string) string {
	for _, l := range Locales {
		if l.Code == code {
			return l.Name
		}
	}
	return code
}
