// Package i18n translates the text of maxor-tui.
//
// The English text is the key, gettext style: code writes tr("Language") and the catalog for the
// active language maps "Language" to its translation. A text a catalog does not have is shown in
// English, so a partial translation is always usable.
//
// Catalogs are JSON objects in lang/<code>.json ({"English text": "translation"}), embedded in the
// binary. Anyone can add a language with a pull request: see docs/TRANSLATING.md.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed lang/*.json
var files embed.FS

var (
	mu   sync.RWMutex
	cur  map[string]string // active catalog; nil means English
	code = "en"
	all  = map[string]map[string]string{} // loaded lazily, by file name without extension
)

// Catalog returns the catalog of a language ("es", "pt_BR"…), or nil if there is none.
func Catalog(lang string) map[string]string {
	mu.Lock()
	defer mu.Unlock()
	return catalogLocked(lang)
}

func catalogLocked(lang string) map[string]string {
	if c, ok := all[lang]; ok {
		return c
	}
	var c map[string]string
	if b, err := files.ReadFile("lang/" + lang + ".json"); err == nil {
		if json.Unmarshal(b, &c) != nil {
			c = nil
		}
	}
	all[lang] = c
	return c
}

// Languages lists the codes that have a catalog.
func Languages() []string {
	entries, _ := files.ReadDir("lang")
	var out []string
	for _, e := range entries {
		out = append(out, strings.TrimSuffix(e.Name(), ".json"))
	}
	return out
}

// Normalize turns a locale ("es_PE.UTF-8", "pt-BR", "C") into the catalog code that serves it:
// the full "pt_BR" if it exists, else "pt", else "en".
func Normalize(locale string) string {
	l := strings.NewReplacer("-", "_").Replace(locale)
	if i := strings.IndexAny(l, ".@"); i >= 0 {
		l = l[:i]
	}
	mu.Lock()
	defer mu.Unlock()
	if l != "" && catalogLocked(l) != nil {
		return l
	}
	if i := strings.Index(l, "_"); i > 0 && catalogLocked(l[:i]) != nil {
		return l[:i]
	}
	return "en"
}

// Set selects the language from a locale. Unknown ones fall back to English.
func Set(locale string) {
	c := Normalize(locale)
	mu.Lock()
	defer mu.Unlock()
	code = c
	cur = nil
	if c != "en" {
		cur = catalogLocked(c)
	}
}

// Code is the language in use ("en" when none is loaded).
func Code() string {
	mu.RLock()
	defer mu.RUnlock()
	return code
}

// T translates s and formats it with args, like fmt.Sprintf. With no args the text is returned
// as is (a "%" in it is not a verb).
func T(s string, args ...any) string {
	mu.RLock()
	c := cur
	mu.RUnlock()
	if t, ok := c[s]; ok && t != "" {
		s = t
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

// Mark flags a text that is translated later (a table built at start-up, for example): it returns
// s unchanged, and the extractor picks it up. Show it with T(text).
func Mark(s string) string { return s }
