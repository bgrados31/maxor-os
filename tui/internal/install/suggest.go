package install

import "strings"

// What the language chosen says about the rest: the keyboard most people with that language use, and the time
// zone of its country when the country has one (or one most people live in). The installer proposes them; the
// person can still pick anything else. Without a network this is the only way to guess the time zone.

// SuggestLayout is the keyboard layout for a locale («es_PE.UTF-8» → Spanish (Latin America)), and whether there
// is one.
func SuggestLayout(locale string) (Layout, bool) {
	lang, country := splitLocale(locale)
	xkb := ""
	switch lang {
	case "es":
		xkb = "latam"
		if country == "ES" {
			xkb = "es"
		}
	case "pt":
		xkb = "br"
		if country == "PT" {
			xkb = "pt"
		}
	case "en":
		xkb = "us"
		if country == "GB" || country == "IE" {
			xkb = "gb"
		}
	case "fr":
		xkb = "fr"
		if country == "CA" {
			xkb = "ca"
		} else if country == "CH" {
			xkb = "ch"
		}
	case "de":
		xkb = "de"
		if country == "CH" {
			xkb = "ch"
		}
	default:
		xkb = map[string]string{
			"it": "it", "nl": "nl", "sv": "se", "nb": "no", "nn": "no", "da": "dk", "fi": "fi", "pl": "pl",
			"cs": "cz", "hu": "hu", "ro": "ro", "ru": "ru", "uk": "ua", "tr": "tr", "el": "gr", "ja": "jp", "ko": "kr",
		}[lang]
	}
	if xkb == "" {
		return Layout{}, false
	}
	for _, l := range Layouts {
		if l.XKB == xkb && l.Variant == "" {
			return l, true
		}
	}
	return Layout{}, false
}

// SuggestZone is the time zone of the country of a locale («es_PE.UTF-8» → America/Lima), and whether there is
// one. Countries spread over several zones with no clear main one (the United States, Canada, Russia…) have none.
func SuggestZone(locale string) (string, bool) {
	_, country := splitLocale(locale)
	z, ok := zoneByCountry[country]
	return z, ok
}

var zoneByCountry = map[string]string{
	"PE": "America/Lima", "MX": "America/Mexico_City", "AR": "America/Argentina/Buenos_Aires", "CO": "America/Bogota",
	"CL": "America/Santiago", "VE": "America/Caracas", "EC": "America/Guayaquil", "BO": "America/La_Paz",
	"UY": "America/Montevideo", "PY": "America/Asuncion", "CR": "America/Costa_Rica", "GT": "America/Guatemala",
	"HN": "America/Tegucigalpa", "SV": "America/El_Salvador", "NI": "America/Managua", "PA": "America/Panama",
	"DO": "America/Santo_Domingo", "PR": "America/Puerto_Rico", "CU": "America/Havana", "BR": "America/Sao_Paulo",
	"ES": "Europe/Madrid", "PT": "Europe/Lisbon", "GB": "Europe/London", "IE": "Europe/Dublin", "FR": "Europe/Paris",
	"DE": "Europe/Berlin", "IT": "Europe/Rome", "NL": "Europe/Amsterdam", "BE": "Europe/Brussels",
	"CH": "Europe/Zurich", "AT": "Europe/Vienna", "SE": "Europe/Stockholm", "NO": "Europe/Oslo", "DK": "Europe/Copenhagen",
	"FI": "Europe/Helsinki", "PL": "Europe/Warsaw", "CZ": "Europe/Prague", "HU": "Europe/Budapest", "RO": "Europe/Bucharest",
	"GR": "Europe/Athens", "TR": "Europe/Istanbul", "UA": "Europe/Kyiv", "JP": "Asia/Tokyo", "KR": "Asia/Seoul",
	"CN": "Asia/Shanghai", "IN": "Asia/Kolkata",
}

// splitLocale turns «es_PE.UTF-8» into «es» and «PE».
func splitLocale(locale string) (lang, country string) {
	base, _, _ := strings.Cut(locale, ".")
	base, _, _ = strings.Cut(base, "@")
	lang, country, _ = strings.Cut(base, "_")
	return lang, country
}
