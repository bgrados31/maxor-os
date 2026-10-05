package i18n

// The tooling around the catalogs, kept in tests so the program itself carries none of it:
//
//	go test ./internal/i18n -update    extracts every text of the code into lang/maxor-tui.pot and merges it
//	                                   into every lang/<code>.po (new texts empty, removed ones dropped)
//	go test ./internal/i18n -run Sync  checks that the catalogs match the code and are well formed

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "extract the texts of the code and merge them into every catalog")

// Reviewed languages must be complete; the rest may have gaps (they show English there).
var reviewed = map[string]bool{"es": true}

// pluralForms is the Plural-Forms header a new catalog starts with. Anything missing here is added as the
// language is (the list is gettext's own; https://www.gnu.org/software/gettext/manual/html_node/Plural-forms.html).
var pluralForms = map[string]string{
	"en": "nplurals=2; plural=(n != 1);", "es": "nplurals=2; plural=(n != 1);", "de": "nplurals=2; plural=(n != 1);",
	"it": "nplurals=2; plural=(n != 1);", "nl": "nplurals=2; plural=(n != 1);", "sv": "nplurals=2; plural=(n != 1);",
	"fr": "nplurals=2; plural=(n > 1);", "pt": "nplurals=2; plural=(n > 1);", "pt_PT": "nplurals=2; plural=(n != 1);",
	"ja": "nplurals=1; plural=0;", "ko": "nplurals=1; plural=0;", "zh": "nplurals=1; plural=0;", "vi": "nplurals=1; plural=0;",
	"tr": "nplurals=2; plural=(n != 1);", "id": "nplurals=1; plural=0;",
	"ru": "nplurals=3; plural=(n%10==1 && n%100!=11 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2);",
	"uk": "nplurals=3; plural=(n%10==1 && n%100!=11 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2);",
	"pl": "nplurals=3; plural=(n==1 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2);",
	"cs": "nplurals=3; plural=(n==1) ? 0 : (n>=2 && n<=4) ? 1 : 2;",
	"ar": "nplurals=6; plural=(n==0 ? 0 : n==1 ? 1 : n==2 ? 2 : n%100>=3 && n%100<=10 ? 3 : n%100>=11 ? 4 : 5);",
}

// srcMsg is a text the code asks to translate.
type srcMsg struct {
	ctx, id, plural string
	refs            map[string]bool
}

// sources returns every text of the program: tr("…"), trn("…","…",n), trc("ctx","…"), and the i18n.T, i18n.N,
// i18n.C and i18n.Mark calls behind them.
func sources(t *testing.T) map[string]*srcMsg {
	t.Helper()
	out := map[string]*srcMsg{}
	fset := token.NewFileSet()
	str := func(e ast.Expr) (string, bool) {
		if bl, ok := e.(*ast.BasicLit); ok && bl.Kind == token.STRING {
			if s, err := strconv.Unquote(bl.Value); err == nil {
				return s, true
			}
		}
		return "", false
	}
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "i18n" {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(filepath.ToSlash(path), "../")
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			name := ""
			switch fn := call.Fun.(type) {
			case *ast.Ident:
				name = fn.Name
			case *ast.SelectorExpr:
				if id, ok := fn.X.(*ast.Ident); ok && id.Name == "i18n" {
					name = "i18n." + fn.Sel.Name
				}
			}
			var m srcMsg
			where := fset.Position(call.Pos()).String()
			switch name {
			case "tr", "i18n.T", "i18n.Mark":
				if _, bad := call.Args[0].(*ast.BinaryExpr); bad {
					t.Errorf("%s: a text built with + cannot be translated: use a format with %%s", where)
				}
				id, ok := str(call.Args[0])
				if !ok {
					return true
				}
				m = srcMsg{id: id}
			case "trn", "i18n.N":
				if len(call.Args) < 3 {
					return true
				}
				one, ok1 := str(call.Args[0])
				many, ok2 := str(call.Args[1])
				if !ok1 || !ok2 {
					t.Errorf("%s: the singular and the plural of a counted text must be string literals", where)
					return true
				}
				m = srcMsg{id: one, plural: many}
			case "trc", "i18n.C":
				if len(call.Args) < 2 {
					return true
				}
				ctx, ok1 := str(call.Args[0])
				id, ok2 := str(call.Args[1])
				if !ok1 || !ok2 {
					return true
				}
				m = srcMsg{ctx: ctx, id: id}
			default:
				return true
			}
			k := key(m.ctx, m.id)
			if have, dup := out[k]; dup {
				have.refs[rel] = true
				if m.plural != "" && have.plural == "" {
					have.plural = m.plural
				}
				return true
			}
			m.refs = map[string]bool{rel: true}
			out[k] = &m
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func langs(t *testing.T) []string {
	t.Helper()
	l := Languages()
	if len(l) == 0 {
		t.Fatal("no catalogs")
	}
	sort.Strings(l)
	return l
}

func readCatalog(t *testing.T, path string) *Catalog {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	c, err := ParsePO(f)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return c
}

// ── writing ──────────────────────────────────────────────────────────

func sortedKeys(src map[string]*srcMsg) []string {
	keys := make([]string, 0, len(src))
	for k := range src {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := src[keys[i]], src[keys[j]]
		if a.id != b.id {
			return a.id < b.id
		}
		return a.ctx < b.ctx
	})
	return keys
}

// writeCatalog writes a .po (old != nil: keeping its translations) or the .pot (lang == "").
func writeCatalog(path, lang string, src map[string]*srcMsg, old *Catalog) error {
	var b strings.Builder
	hdr := func(k, v string) { fmt.Fprintf(&b, "\"%s: %s\\n\"\n", k, v) }
	b.WriteString("# Maxor OS: texts of the full-screen app and the installer.\n# How to translate: docs/TRANSLATING.md\n")
	b.WriteString("msgid \"\"\nmsgstr \"\"\n")
	hdr("Project-Id-Version", "Maxor OS")
	hdr("Report-Msgid-Bugs-To", "https://github.com/bgrados31/maxor-os/issues")
	if lang != "" {
		hdr("Language", lang)
		for _, k := range []string{"Language-Team", "Last-Translator"} {
			if old != nil && old.Header[k] != "" {
				hdr(k, old.Header[k])
			}
		}
	}
	hdr("MIME-Version", "1.0")
	hdr("Content-Type", "text/plain; charset=UTF-8")
	hdr("Content-Transfer-Encoding", "8bit")
	nplurals := 2
	if lang != "" {
		pf := pluralForms[lang]
		if old != nil && old.Header["Plural-Forms"] != "" {
			pf = old.Header["Plural-Forms"]
		}
		if pf == "" {
			pf = pluralForms[strings.SplitN(lang, "_", 2)[0]]
		}
		if pf == "" {
			return fmt.Errorf("%s: add its Plural-Forms to pluralForms in tools_test.go (or to the header of the file)", lang)
		}
		hdr("Plural-Forms", pf)
		np, _, err := parsePluralForms(pf)
		if err != nil {
			return err
		}
		nplurals = np
	}
	for _, k := range sortedKeys(src) {
		m := src[k]
		var refs []string
		for r := range m.refs {
			refs = append(refs, r)
		}
		sort.Strings(refs)
		b.WriteString("\n#: " + strings.Join(refs, " ") + "\n")
		var prev *Entry
		if old != nil {
			prev = old.Lookup(m.ctx, m.id)
		}
		fuzzy := prev != nil && prev.Fuzzy
		var forms []string
		if m.plural != "" {
			forms = make([]string, nplurals)
		} else {
			forms = make([]string, 1)
		}
		if prev != nil && lang != "" {
			copy(forms, prev.Str)
			if m.plural != "" && prev.Plural == "" && len(prev.Str) > 0 && prev.Str[0] != "" {
				fuzzy = true // it was a single text: someone must look at each form
			}
		}
		if fuzzy {
			b.WriteString("#, fuzzy\n")
		}
		if m.ctx != "" {
			b.WriteString("msgctxt " + quote(m.ctx) + "\n")
		}
		b.WriteString("msgid " + quote(m.id) + "\n")
		if m.plural != "" {
			b.WriteString("msgid_plural " + quote(m.plural) + "\n")
			for i := 0; i < nplurals; i++ {
				s := ""
				if lang != "" && i < len(forms) {
					s = forms[i]
				}
				fmt.Fprintf(&b, "msgstr[%d] %s\n", i, quote(s))
			}
		} else {
			s := ""
			if lang != "" {
				s = forms[0]
			}
			b.WriteString("msgstr " + quote(s) + "\n")
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// ── placeholders ─────────────────────────────────────────────────────

var verb = regexp.MustCompile(`%(?:(\d+)\$)?[-+# 0-9.*]*([a-zA-Z%])`)

type ph struct {
	idx int
	typ string
}

// placeholders lists the verbs of a text, numbering the plain ones in order (%s %s = 1 2).
func placeholders(s string) []ph {
	var out []ph
	next := 1
	for _, m := range verb.FindAllStringSubmatch(s, -1) {
		if m[2] == "%" {
			continue
		}
		i := next
		if m[1] != "" {
			i, _ = strconv.Atoi(m[1])
		} else {
			next++
		}
		out = append(out, ph{i, m[2]})
	}
	return out
}

func positionalOnly(s string) bool {
	for _, m := range verb.FindAllStringSubmatch(s, -1) {
		if m[2] != "%" && m[1] == "" {
			return false
		}
	}
	return true
}

// checkFormat says what is wrong with a translation's placeholders compared with the English text, or "".
// Positional ones (%2$s) may come in any order; plain ones must keep the English order. In a plural form the
// count may be left out ("one app"), but nothing may be added.
func checkFormat(english, tr string, pluralForm bool) string {
	en, tx := placeholders(english), placeholders(tr)
	if len(placeholders(tr)) > 0 && !positionalOnly(tr) && len(en) > 0 {
		// plain verbs: the same sequence
		if !pluralForm || len(tx) == len(en) {
			if len(tx) != len(en) {
				return fmt.Sprintf("%d placeholders, English has %d", len(tx), len(en))
			}
			for i := range en {
				if en[i].typ != tx[i].typ {
					return fmt.Sprintf("placeholder %d is %%%s, English has %%%s", i+1, tx[i].typ, en[i].typ)
				}
			}
			return ""
		}
	}
	want := map[int]string{}
	for _, p := range en {
		want[p.idx] = p.typ
	}
	if !positionalOnly(tr) {
		// a plural form with fewer plain verbs than English: map them in order onto the English ones
		for i, p := range tx {
			if i >= len(en) || en[i].typ != p.typ {
				return "a placeholder English does not have"
			}
		}
		return ""
	}
	seen := map[int]bool{}
	for _, p := range tx {
		typ, ok := want[p.idx]
		if !ok {
			return fmt.Sprintf("%%%d$%s: English has no placeholder number %d", p.idx, p.typ, p.idx)
		}
		if typ != p.typ {
			return fmt.Sprintf("%%%d$%s: English has %%%s", p.idx, p.typ, typ)
		}
		seen[p.idx] = true
	}
	if !pluralForm {
		for i := range want {
			if !seen[i] {
				return fmt.Sprintf("placeholder %d of the English text is missing", i)
			}
		}
	}
	return ""
}

// ── the tests ────────────────────────────────────────────────────────

func TestSync(t *testing.T) {
	src := sources(t)
	pot := filepath.Join("lang", "maxor-tui.pot")
	if *update {
		if err := writeCatalog(pot, "", src, nil); err != nil {
			t.Fatal(err)
		}
		for _, l := range langs(t) {
			p := filepath.Join("lang", l+".po")
			if err := writeCatalog(p, l, src, readCatalog(t, p)); err != nil {
				t.Fatal(err)
			}
		}
	}
	// the template is what Weblate and the other tools translate from: it must match the code
	tc := readCatalog(t, pot)
	for k, m := range src {
		if tc.Lookup(m.ctx, m.id) == nil {
			t.Errorf("maxor-tui.pot lacks %q (from %s): run `go test ./internal/i18n -update`", m.id, anyRef(m))
			_ = k
		}
	}
	for _, e := range tc.Entries() {
		if src[key(e.Ctx, e.ID)] == nil {
			t.Errorf("maxor-tui.pot has %q, which the code no longer uses: run `go test ./internal/i18n -update`", e.ID)
		}
	}

	for _, l := range langs(t) {
		c := readCatalog(t, filepath.Join("lang", l+".po"))
		if c.Header["Language"] != "" && c.Header["Language"] != l {
			t.Errorf("%s: the header says Language: %s", l, c.Header["Language"])
		}
		var missing []string
		for k, m := range src {
			e := c.Lookup(m.ctx, m.id)
			if e == nil {
				t.Errorf("%s: lacks %q: run `go test ./internal/i18n -update`", l, m.id)
				continue
			}
			if (m.plural != "") != (e.Plural != "") {
				t.Errorf("%s: %q is counted in the code and not in the catalog (or the opposite): run -update", l, m.id)
				continue
			}
			if len(e.Str) == 0 || e.Fuzzy {
				missing = append(missing, m.id+"  ("+anyRef(src[k])+")")
				continue
			}
			for i, s := range e.Str {
				if s == "" {
					if m.plural == "" || len(e.Str) == 1 {
						missing = append(missing, m.id+"  ("+anyRef(src[k])+")")
					}
					break
				}
				base := m.id
				if i > 0 && m.plural != "" {
					base = m.plural
				}
				if why := checkFormat(base, s, m.plural != ""); why != "" {
					t.Errorf("%s: %q → %q: %s", l, base, s, why)
				}
			}
			if m.plural != "" && len(e.Str) != c.NPlurals {
				t.Errorf("%s: %q has %d forms, the language has %d", l, m.id, len(e.Str), c.NPlurals)
			}
		}
		for _, e := range c.Entries() {
			if src[key(e.Ctx, e.ID)] == nil {
				t.Errorf("%s: %q is not in the code any more: run `go test ./internal/i18n -update`", l, e.ID)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			msg := fmt.Sprintf("%s: %d of %d texts untranslated (they show in English)", l, len(missing), len(src))
			if reviewed[l] {
				t.Errorf("%s:\n  %s", msg, strings.Join(missing, "\n  "))
			} else {
				t.Log(msg)
			}
		}
	}
}

func anyRef(m *srcMsg) string {
	for r := range m.refs {
		return r
	}
	return ""
}

func TestPluralRules(t *testing.T) {
	for lang, cases := range map[string]map[int]int{
		"en": {0: 1, 1: 0, 2: 1},
		"fr": {0: 0, 1: 0, 2: 1},
		"ru": {1: 0, 2: 1, 5: 2, 11: 2, 21: 0, 22: 1, 25: 2, 111: 2},
		"pl": {1: 0, 2: 1, 5: 2, 12: 2, 22: 1},
		"ar": {0: 0, 1: 1, 2: 2, 3: 3, 10: 3, 11: 4, 99: 4, 100: 5, 102: 5},
		"ja": {0: 0, 1: 0, 5: 0},
		"cs": {1: 0, 3: 1, 5: 2},
	} {
		n, f, err := parsePluralForms(pluralForms[lang])
		if err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		for in, want := range cases {
			if got := f(in); got != want || got >= n {
				t.Errorf("%s: %d → form %d, want %d", lang, in, got, want)
			}
		}
	}
	if _, _, err := parsePluralForms("nplurals=2; plural=(n ! 1);"); err == nil {
		t.Error("a malformed rule must be refused")
	}
}

func TestParsePO(t *testing.T) {
	c, err := ParsePO(strings.NewReader(`
# translator comment
msgid ""
msgstr ""
"Language: ru\n"
"Plural-Forms: nplurals=3; plural=(n%10==1 && n%100!=11 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2);\n"

#: a.go
msgid "One line"
msgstr "Una\n"
"línea"

#, fuzzy
msgid "Fuzzy"
msgstr "No se usa"

msgctxt "verb"
msgid "Update"
msgstr "Actualizar"

msgid "%d file"
msgid_plural "%d files"
msgstr[0] "%d файл"
msgstr[1] "%d файла"
msgstr[2] "%d файлов"

#~ msgid "Gone"
#~ msgstr "Fuera"
`))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Text(c.Lookup("", "One line"), 1); got != "Una\nlínea" {
		t.Errorf("multi-line string: %q", got)
	}
	if c.Text(c.Lookup("", "Fuzzy"), 1) != "" {
		t.Error("a fuzzy entry must not be used")
	}
	if c.Text(c.Lookup("verb", "Update"), 1) != "Actualizar" || c.Lookup("", "Update") != nil {
		t.Error("context is part of the key")
	}
	e := c.Lookup("", "%d file")
	for n, want := range map[int]string{1: "%d файл", 3: "%d файла", 5: "%d файлов", 21: "%d файл"} {
		if got := c.Text(e, n); got != want {
			t.Errorf("%d: %q, want %q", n, got, want)
		}
	}
	if c.Lookup("", "Gone") != nil {
		t.Error("obsolete entries are ignored")
	}
	if _, err := ParsePO(strings.NewReader("msgid \"x\"\nbogus line\n")); err == nil {
		t.Error("a line that is not PO must be an error")
	}
}

func TestCheckFormat(t *testing.T) {
	for _, c := range []struct {
		en, tr string
		plural bool
		ok     bool
	}{
		{"Could not install %s: %s", "No se pudo instalar %s: %s", false, true},
		{"Could not install %s: %s", "%2$s: no se pudo instalar %1$s", false, true},
		{"Could not install %s: %s", "No se pudo instalar %s", false, false},
		{"Could not install %s: %s", "%1$s %3$s", false, false},
		{"%d of %d", "%d de %d", false, true},
		{"%d of %d", "%s de %d", false, false},
		{"%d file", "un archivo", true, true},
		{"%d file", "%d archivos %s", true, false},
		{"100%% sure", "100%% seguro", false, true},
		{"no placeholders", "sin marcadores", false, true},
	} {
		if got := checkFormat(c.en, c.tr, c.plural) == ""; got != c.ok {
			t.Errorf("checkFormat(%q, %q) ok=%v, want %v", c.en, c.tr, got, c.ok)
		}
	}
}

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"es_PE.UTF-8": "es", "es": "es", "es-MX": "es", "C": "en", "": "en", "xx_YY": "en", "en_US.UTF-8": "en",
	} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTranslating(t *testing.T) {
	defer Set("en")
	Set("es")
	if got := T("a text no catalog has %d", 3); got != "a text no catalog has 3" {
		t.Errorf("fallback to English: %q", got)
	}
	if got := T("100% sure"); got != "100% sure" {
		t.Errorf("a %% without args must stay: %q", got)
	}
	if got := N("%d zebra", "%d zebras", 2); got != "2 zebras" {
		t.Errorf("English plural fallback: %q", got)
	}
	if got := N("%d zebra", "%d zebras", 1); got != "1 zebra" {
		t.Errorf("English singular fallback: %q", got)
	}
	// %2$s is turned into what fmt understands
	if got := goVerbs("%2$s then %1$s"); fmt.Sprintf(got, "a", "b") != "b then a" {
		t.Errorf("positional verbs: %q", got)
	}
	// the real catalogs: each language counts by its own rule
	for lang, cases := range map[string]map[int]string{
		"es": {0: "0 temas", 1: "1 tema", 2: "2 temas"},
		"fr": {0: "0 thème", 1: "1 thème", 2: "2 thèmes"}, // French counts 0 as singular
		"de": {1: "1 Design", 5: "5 Designs"},
	} {
		Set(lang)
		for n, want := range cases {
			if got := N("%d theme", "%d themes", n); got != want {
				t.Errorf("%s: %d → %q, want %q", lang, n, got, want)
			}
		}
	}
	Set("es")
	SetPseudo(true)
	defer SetPseudo(false)
	if got := T("Hello"); got != "[Hello~~]" {
		t.Errorf("pseudo-language: %q", got)
	}
}
