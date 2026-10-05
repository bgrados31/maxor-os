package i18n

import (
	"fmt"
	"strconv"
	"strings"
)

// parsePluralForms reads the header gettext uses to say how many forms a language has and which one
// a number takes: "nplurals=3; plural=(n%10==1 && n%100!=11 ? 0 : …);". The expression is the small C
// subset gettext defines, evaluated here (nothing is executed): n, integers, ! * / % + - < > <= >= == !=
// && || ?: and parentheses.
func parsePluralForms(h string) (int, func(int) int, error) {
	nplurals, expr := 0, ""
	for _, part := range strings.Split(h, ";") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "nplurals":
			n, err := strconv.Atoi(strings.TrimSpace(v))
			if err != nil || n < 1 || n > 20 {
				return 0, nil, fmt.Errorf("bad nplurals %q", v)
			}
			nplurals = n
		case "plural":
			expr = strings.TrimSpace(v)
		}
	}
	if nplurals == 0 || expr == "" {
		return 0, nil, fmt.Errorf("needs nplurals and plural")
	}
	node, err := parseExpr(expr)
	if err != nil {
		return 0, nil, err
	}
	return nplurals, func(n int) int { return int(node.eval(int64(n))) }, nil
}

type node interface{ eval(n int64) int64 }

type (
	numNode    int64
	varNode    struct{}
	unaryNode  struct{ x node }
	binaryNode struct {
		op   string
		l, r node
	}
	condNode struct{ c, a, b node }
)

func (n numNode) eval(int64) int64 { return int64(n) }
func (varNode) eval(n int64) int64 { return n }
func (u unaryNode) eval(n int64) int64 {
	if u.x.eval(n) == 0 {
		return 1
	}
	return 0
}
func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
func (b binaryNode) eval(n int64) int64 {
	switch b.op {
	case "||":
		return b2i(b.l.eval(n) != 0 || b.r.eval(n) != 0)
	case "&&":
		return b2i(b.l.eval(n) != 0 && b.r.eval(n) != 0)
	}
	l, r := b.l.eval(n), b.r.eval(n)
	switch b.op {
	case "==":
		return b2i(l == r)
	case "!=":
		return b2i(l != r)
	case "<":
		return b2i(l < r)
	case ">":
		return b2i(l > r)
	case "<=":
		return b2i(l <= r)
	case ">=":
		return b2i(l >= r)
	case "+":
		return l + r
	case "-":
		return l - r
	case "*":
		return l * r
	case "/":
		if r == 0 {
			return 0
		}
		return l / r
	case "%":
		if r == 0 {
			return 0
		}
		return l % r
	}
	return 0
}
func (c condNode) eval(n int64) int64 {
	if c.c.eval(n) != 0 {
		return c.a.eval(n)
	}
	return c.b.eval(n)
}

type exprParser struct {
	s   string
	pos int
}

func parseExpr(s string) (node, error) {
	p := &exprParser{s: s}
	n, err := p.cond()
	if err != nil {
		return nil, err
	}
	p.skip()
	if p.pos != len(p.s) {
		return nil, fmt.Errorf("unexpected %q in %q", p.s[p.pos:], s)
	}
	return n, nil
}

func (p *exprParser) skip() {
	for p.pos < len(p.s) && (p.s[p.pos] == ' ' || p.s[p.pos] == '\t') {
		p.pos++
	}
}

func (p *exprParser) accept(tok string) bool {
	p.skip()
	if strings.HasPrefix(p.s[p.pos:], tok) {
		p.pos += len(tok)
		return true
	}
	return false
}

func (p *exprParser) cond() (node, error) {
	c, err := p.binary(0)
	if err != nil {
		return nil, err
	}
	if p.accept("?") {
		a, err := p.cond()
		if err != nil {
			return nil, err
		}
		if !p.accept(":") {
			return nil, fmt.Errorf("a ? needs a :")
		}
		b, err := p.cond()
		if err != nil {
			return nil, err
		}
		return condNode{c, a, b}, nil
	}
	return c, nil
}

// the binary operators, from the loosest to the tightest; a longer token is tried before a shorter one
var levels = [][]string{{"||"}, {"&&"}, {"==", "!="}, {"<=", ">=", "<", ">"}, {"+", "-"}, {"*", "/", "%"}}

func (p *exprParser) binary(level int) (node, error) {
	if level == len(levels) {
		return p.unary()
	}
	l, err := p.binary(level + 1)
	if err != nil {
		return nil, err
	}
	for {
		matched := ""
		for _, op := range levels[level] {
			p.skip()
			if strings.HasPrefix(p.s[p.pos:], op) {
				// "<" must not eat the start of "<=", and "!" is not a binary operator here
				matched = op
				break
			}
		}
		if matched == "" {
			return l, nil
		}
		p.pos += len(matched)
		r, err := p.binary(level + 1)
		if err != nil {
			return nil, err
		}
		l = binaryNode{matched, l, r}
	}
}

func (p *exprParser) unary() (node, error) {
	p.skip()
	if p.pos < len(p.s) && p.s[p.pos] == '!' && !strings.HasPrefix(p.s[p.pos:], "!=") {
		p.pos++
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		return unaryNode{x}, nil
	}
	return p.primary()
}

func (p *exprParser) primary() (node, error) {
	p.skip()
	if p.pos >= len(p.s) {
		return nil, fmt.Errorf("the expression ends too soon")
	}
	switch c := p.s[p.pos]; {
	case c == '(':
		p.pos++
		n, err := p.cond()
		if err != nil {
			return nil, err
		}
		if !p.accept(")") {
			return nil, fmt.Errorf("a ( needs a )")
		}
		return n, nil
	case c == 'n':
		p.pos++
		return varNode{}, nil
	case c >= '0' && c <= '9':
		start := p.pos
		for p.pos < len(p.s) && p.s[p.pos] >= '0' && p.s[p.pos] <= '9' {
			p.pos++
		}
		v, err := strconv.ParseInt(p.s[start:p.pos], 10, 64)
		if err != nil {
			return nil, err
		}
		return numNode(v), nil
	default:
		return nil, fmt.Errorf("unexpected %q", string(c))
	}
}
