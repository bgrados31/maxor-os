package i18n

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Entry is one message of a gettext catalog (.po). Str has one form, or one per plural form.
type Entry struct {
	Ctx, ID, Plural string
	Str             []string
	Fuzzy           bool // flagged "needs editing": gettext does not use it until a person confirms it
}

// Catalog is a parsed .po file.
type Catalog struct {
	Header   map[string]string
	NPlurals int
	plural   func(n int) int
	entries  map[string]*Entry
}

func key(ctx, id string) string { return ctx + "\x04" + id }

// Lookup returns the entry of a message, or nil.
func (c *Catalog) Lookup(ctx, id string) *Entry {
	if c == nil {
		return nil
	}
	return c.entries[key(ctx, id)]
}

// Entries lists every entry (header excluded), in no particular order.
func (c *Catalog) Entries() []*Entry {
	out := make([]*Entry, 0, len(c.entries))
	for _, e := range c.entries {
		out = append(out, e)
	}
	return out
}

// Form picks the plural form of n for this language (0 when the language has no plural rule).
func (c *Catalog) Form(n int) int {
	if c == nil || c.plural == nil {
		if n == 1 {
			return 0
		}
		return 1
	}
	f := c.plural(n)
	if f < 0 || (c.NPlurals > 0 && f >= c.NPlurals) {
		return 0
	}
	return f
}

// Text is the translation of an entry for a count, or "" if it has none (untranslated or fuzzy).
func (c *Catalog) Text(e *Entry, n int) string {
	if e == nil || e.Fuzzy || len(e.Str) == 0 {
		return ""
	}
	f := 0
	if e.Plural != "" {
		f = c.Form(n)
	}
	if f >= len(e.Str) {
		return ""
	}
	return e.Str[f]
}

// ParsePO reads the subset of the gettext format that translation tools produce: msgctxt, msgid,
// msgid_plural, msgstr and msgstr[N], with strings split over several lines, translator and flag
// comments, and obsolete entries (#~), which are ignored.
func ParsePO(r io.Reader) (*Catalog, error) {
	c := &Catalog{Header: map[string]string{}, entries: map[string]*Entry{}}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)

	var cur *Entry
	var field *string // the string the continuation lines append to
	var strIdx int
	started := false
	fuzzy := false
	flush := func() error {
		if cur == nil {
			return nil
		}
		if cur.ID == "" && cur.Ctx == "" {
			if len(cur.Str) > 0 {
				for _, l := range strings.Split(cur.Str[0], "\n") {
					if k, v, ok := strings.Cut(l, ":"); ok {
						c.Header[strings.TrimSpace(k)] = strings.TrimSpace(v)
					}
				}
			}
		} else {
			c.entries[key(cur.Ctx, cur.ID)] = cur
		}
		cur, field, started = nil, nil, false
		return nil
	}
	begin := func() {
		if cur == nil {
			cur = &Entry{Fuzzy: fuzzy}
			fuzzy = false
		}
	}
	line := 0
	for sc.Scan() {
		line++
		s := strings.TrimSpace(sc.Text())
		switch {
		case s == "":
			if err := flush(); err != nil {
				return nil, err
			}
		case strings.HasPrefix(s, "#~"):
			// an obsolete entry: nothing to load
		case strings.HasPrefix(s, "#"):
			if started {
				if err := flush(); err != nil {
					return nil, err
				}
			}
			if strings.HasPrefix(s, "#,") && strings.Contains(s, "fuzzy") {
				fuzzy = true
			}
		case strings.HasPrefix(s, "msgctxt "):
			if started && cur != nil && (cur.Str != nil) {
				_ = flush()
			}
			begin()
			started = true
			v, err := unquote(strings.TrimPrefix(s, "msgctxt "))
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", line, err)
			}
			cur.Ctx = v
			field = &cur.Ctx
		case strings.HasPrefix(s, "msgid_plural "):
			begin()
			v, err := unquote(strings.TrimPrefix(s, "msgid_plural "))
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", line, err)
			}
			cur.Plural = v
			field = &cur.Plural
		case strings.HasPrefix(s, "msgid "):
			if started && cur != nil && cur.Str != nil {
				_ = flush()
			}
			begin()
			started = true
			v, err := unquote(strings.TrimPrefix(s, "msgid "))
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", line, err)
			}
			cur.ID = v
			field = &cur.ID
		case strings.HasPrefix(s, "msgstr["):
			end := strings.Index(s, "]")
			if end < 0 || cur == nil {
				return nil, fmt.Errorf("line %d: malformed msgstr[N]", line)
			}
			idx, err := strconv.Atoi(s[len("msgstr["):end])
			if err != nil || idx < 0 || idx > 20 {
				return nil, fmt.Errorf("line %d: bad plural index", line)
			}
			v, err := unquote(strings.TrimSpace(s[end+1:]))
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", line, err)
			}
			for len(cur.Str) <= idx {
				cur.Str = append(cur.Str, "")
			}
			cur.Str[idx] = v
			strIdx = idx
			field = &cur.Str[strIdx]
		case strings.HasPrefix(s, "msgstr "):
			if cur == nil {
				return nil, fmt.Errorf("line %d: msgstr without msgid", line)
			}
			v, err := unquote(strings.TrimPrefix(s, "msgstr "))
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", line, err)
			}
			cur.Str = []string{v}
			strIdx = 0
			field = &cur.Str[0]
		case strings.HasPrefix(s, `"`):
			if field == nil {
				return nil, fmt.Errorf("line %d: a string with no keyword before it", line)
			}
			v, err := unquote(s)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", line, err)
			}
			*field += v
		default:
			return nil, fmt.Errorf("line %d: not understood: %q", line, s)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if err := flush(); err != nil {
		return nil, err
	}

	c.NPlurals, c.plural = 2, func(n int) int {
		if n != 1 {
			return 1
		}
		return 0
	}
	if pf := c.Header["Plural-Forms"]; pf != "" {
		np, expr, err := parsePluralForms(pf)
		if err != nil {
			return nil, fmt.Errorf("Plural-Forms: %w", err)
		}
		c.NPlurals, c.plural = np, expr
	}
	return c, nil
}

// unquote reads a "…" string with the escapes gettext uses.
func unquote(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", fmt.Errorf("expected a quoted string, got %q", s)
	}
	s = s[1 : len(s)-1]
	if !strings.Contains(s, `\`) {
		return s, nil
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			continue
		}
		i++
		if i >= len(s) {
			return "", fmt.Errorf("a string ends in a backslash")
		}
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case '"':
			b.WriteByte('"')
		case '\\':
			b.WriteByte('\\')
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String(), nil
}

// quote writes a string the way gettext does, splitting after every line break.
func quote(s string) string {
	esc := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\t", `\t`, "\r", `\r`)
	parts := strings.SplitAfter(s, "\n")
	if len(parts) <= 1 {
		return `"` + esc.Replace(strings.ReplaceAll(s, "\n", `\n`)) + `"`
	}
	var out []string
	for _, p := range parts {
		if p == "" {
			continue
		}
		out = append(out, `"`+esc.Replace(strings.ReplaceAll(p, "\n", `\n`))+`"`)
	}
	return `""` + "\n" + strings.Join(out, "\n")
}
