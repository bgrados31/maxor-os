package i18n

import (
	"encoding/json"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "add the texts missing from every catalog (empty = untranslated) and drop the stale ones")

// reviewed languages must be complete; the rest may have gaps (they show English there).
var reviewed = map[string]bool{"es": true}

// sources returns every text the program asks to translate: tr("…"), i18n.T("…") and i18n.Mark("…").
func sources(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{} // text → where
	fs := token.NewFileSet()
	err := filepath.WalkDir("..", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "i18n" {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fs, path, nil, 0)
		if err != nil {
			return err
		}
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
			if name != "tr" && name != "i18n.T" && name != "i18n.Mark" {
				return true
			}
			where := fs.Position(call.Pos()).String()
			switch a := call.Args[0].(type) {
			case *ast.BasicLit:
				if s, err := strconv.Unquote(a.Value); err == nil {
					if _, dup := out[s]; !dup {
						out[s] = where
					}
				}
			case *ast.BinaryExpr:
				t.Errorf("%s: a text built with + cannot be translated: use a format with %%s", where)
			}
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

var verb = regexp.MustCompile(`%[-+# 0-9.*]*[a-zA-Z%]`)

func TestSync(t *testing.T) {
	src := sources(t)
	for _, l := range langs(t) {
		path := filepath.Join("lang", l+".json")
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		cat := map[string]string{}
		if err := json.Unmarshal(b, &cat); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if *update {
			next := map[string]string{}
			for k := range src {
				next[k] = cat[k]
			}
			out, _ := json.MarshalIndent(next, "", "  ")
			if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
				t.Fatal(err)
			}
			cat = next
		}
		var missing []string
		for k, where := range src {
			v, ok := cat[k]
			if !ok || v == "" {
				missing = append(missing, k+"  ("+where+")")
				continue
			}
			// the placeholders must be the same, in the same order: a wrong %s would garble or crash the text
			if a, b := strings.Join(verb.FindAllString(k, -1), " "), strings.Join(verb.FindAllString(v, -1), " "); a != b {
				t.Errorf("%s: placeholders differ in %q → %q (%s vs %s)", l, k, v, a, b)
			}
		}
		for k := range cat {
			if _, ok := src[k]; !ok {
				t.Errorf("%s: %q is not in the code any more: run `go test ./internal/i18n -update`", l, k)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			msg := l + ": " + strconv.Itoa(len(missing)) + " of " + strconv.Itoa(len(src)) + " texts untranslated (they show in English)"
			if reviewed[l] {
				t.Errorf("%s:\n  %s", msg, strings.Join(missing, "\n  "))
			} else {
				t.Log(msg)
			}
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

func TestTFallsBackToEnglish(t *testing.T) {
	Set("es")
	defer Set("en")
	if got := T("a text no catalog has %d", 3); got != "a text no catalog has 3" {
		t.Errorf("got %q", got)
	}
	if got := T("100% sure"); got != "100% sure" {
		t.Errorf("a %% without args must stay: %q", got)
	}
}
