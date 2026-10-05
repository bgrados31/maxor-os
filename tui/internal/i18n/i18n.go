// Package i18n translates the text of maxor-tui.
//
// The English text is the key, gettext style: code writes tr("Language") and the catalog of the
// active language maps "Language" to its translation. A text a catalog does not have is shown in
// English, so a partial translation is always usable.
//
// Catalogs are standard gettext files, lang/<code>.po, embedded in the binary, so any translation tool
// (Weblate, Crowdin, Poedit, Lokalize…) can edit them: plurals follow each language's own rule, a
// context tells apart two meanings of the same English word, and a placeholder can be moved with
// %1$s, %2$s… when a language needs another word order. See docs/TRANSLATING.md.
package i18n

import (
	"embed"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

//go:embed lang/*.po
var files embed.FS

var (
	mu     sync.RWMutex
	cur    *Catalog // active catalog; nil means English
	code   = "en"
	pseudo bool
	all    = map[string]*Catalog{} // loaded lazily, by file name without extension
)

// Load returns the catalog of a language ("es", "pt_BR"…), or nil if there is none.
func Load(lang string) *Catalog {
	mu.Lock()
	defer mu.Unlock()
	return catalogLocked(lang)
}

func catalogLocked(lang string) *Catalog {
	if c, ok := all[lang]; ok {
		return c
	}
	var c *Catalog
	if f, err := files.Open("lang/" + lang + ".po"); err == nil {
		c, err = ParsePO(f)
		_ = f.Close()
		if err != nil {
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
		out = append(out, strings.TrimSuffix(e.Name(), ".po"))
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

// SetPseudo turns on a pseudo-language that lengthens every text by about 40% and brackets it, to
// see how a layout copes with a language longer than English (German, Finnish…) without having one.
// Tests use it; it is not a language people can pick.
func SetPseudo(on bool) {
	mu.Lock()
	defer mu.Unlock()
	pseudo = on
}

// Code is the language in use ("en" when none is loaded).
func Code() string {
	mu.RLock()
	defer mu.RUnlock()
	return code
}

// T translates s and formats it with args, like fmt.Sprintf. With no args the text is returned
// as is (a "%" in it is not a verb).
func T(s string, args ...any) string { return lookup("", s, "", 1, false, args) }

// C is T for a message whose English text is ambiguous: the context ("verb" or "noun"…) is part of the
// key, so each meaning is translated on its own.
func C(ctx, s string, args ...any) string { return lookup(ctx, s, "", 1, false, args) }

// N picks the singular or the plural form for n, by the rule of the language in use, and formats it
// with args. With no args, n itself is the one argument: N("%d app", "%d apps", n).
func N(one, many string, n int, args ...any) string {
	if len(args) == 0 {
		args = []any{n}
	}
	return lookup("", one, many, n, true, args)
}

func lookup(ctx, id, plural string, n int, isPlural bool, args []any) string {
	mu.RLock()
	c, ps := cur, pseudo
	mu.RUnlock()

	s := id
	if isPlural && n != 1 {
		s = plural
	}
	if c != nil {
		if e := c.Lookup(ctx, id); e != nil {
			if t := c.Text(e, n); t != "" {
				s = t
			}
		}
	}
	if ps {
		s = lengthen(s)
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(goVerbs(s), args...)
}

// "%2$s" (what translators and their tools know) → "%[2]s" (what fmt understands).
var positional = regexp.MustCompile(`%(\d+)\$`)

func goVerbs(s string) string {
	if !strings.Contains(s, "$") {
		return s
	}
	return positional.ReplaceAllString(s, "%[$1]")
}

// lengthen pads a text by ~40% with a visible filler, keeping every % verb intact.
func lengthen(s string) string {
	extra := len([]rune(s)) * 2 / 5
	if extra < 2 {
		extra = 2
	}
	return "[" + s + strings.Repeat("~", extra) + "]"
}

// Mark flags a text that is translated later (a table built at start-up, for example): it returns
// s unchanged, and the extractor picks it up. Show it with T(text).
func Mark(s string) string { return s }
