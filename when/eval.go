package when

import (
	"regexp"
	"slices"
	"strings"
)

// Precedence, lowest first, for String's parentheses.
const (
	precOr = iota + 1
	precAnd
	precNot
	precAtom
)

// node is one part of a parsed expression.
type node interface {
	eval(c Context) bool
	str() string
	prec() int
	keys(add func(string))
}

// wrap is n's text, in parentheses when it binds looser than need.
func wrap(n node, need int) string {
	if n.prec() < need {
		return "(" + n.str() + ")"
	}
	return n.str()
}

type orNode struct{ x, y node }

func (n orNode) eval(c Context) bool { return n.x.eval(c) || n.y.eval(c) }
func (n orNode) str() string         { return wrap(n.x, precOr) + " || " + wrap(n.y, precAnd) }
func (orNode) prec() int             { return precOr }
func (n orNode) keys(add func(string)) {
	n.x.keys(add)
	n.y.keys(add)
}

type andNode struct{ x, y node }

func (n andNode) eval(c Context) bool { return n.x.eval(c) && n.y.eval(c) }
func (n andNode) str() string         { return wrap(n.x, precAnd) + " && " + wrap(n.y, precNot) }
func (andNode) prec() int             { return precAnd }
func (n andNode) keys(add func(string)) {
	n.x.keys(add)
	n.y.keys(add)
}

type notNode struct{ x node }

func (n notNode) eval(c Context) bool   { return !n.x.eval(c) }
func (n notNode) str() string           { return "!" + wrap(n.x, precNot) }
func (notNode) prec() int               { return precNot }
func (n notNode) keys(add func(string)) { n.x.keys(add) }

// constNode is true or false.
type constNode bool

func (n constNode) eval(Context) bool { return bool(n) }
func (constNode) prec() int           { return precAtom }
func (constNode) keys(func(string))   {}
func (n constNode) str() string {
	if n {
		return trueWord
	}
	return falseWord
}

// keyNode is a key standing alone: true when it is set and not false, 0,
// "" or empty.
type keyNode string

func (n keyNode) eval(c Context) bool {
	v, ok := c.Value(string(n))
	return ok && v.Bool()
}

func (n keyNode) str() string           { return string(n) }
func (keyNode) prec() int               { return precAtom }
func (n keyNode) keys(add func(string)) { add(string(n)) }

// cmpNode compares a key with a value. A key that is not set is unequal to
// everything, so != holds and every other comparison fails.
type cmpNode struct {
	op  tokKind
	key string
	lit Value
}

func (n cmpNode) eval(c Context) bool {
	v, ok := c.Value(n.key)
	if !ok {
		return n.op == tNe
	}
	switch n.op {
	case tEq:
		return equal(v, n.lit)
	case tNe:
		return !equal(v, n.lit)
	}
	a, aok := v.Number()
	b, bok := n.lit.Number()
	if !aok || !bok {
		return false
	}
	switch n.op {
	case tLt:
		return a < b
	case tLe:
		return a <= b
	case tGt:
		return a > b
	}
	return a >= b
}

// equal compares a key's value with a literal: numbers as numbers, a list
// never equal to a value, anything else by its canonical text.
func equal(v, lit Value) bool {
	if v.Kind() == KindList {
		return false
	}
	if a, ok := v.Number(); ok {
		if b, ok := lit.Number(); ok {
			return a == b
		}
	}
	return v.String() == lit.String()
}

var opText = map[tokKind]string{tEq: "==", tNe: "!=", tLt: "<", tLe: "<=", tGt: ">", tGe: ">="}

func (n cmpNode) str() string           { return n.key + " " + opText[n.op] + " " + literalText(n.lit) }
func (cmpNode) prec() int               { return precAtom }
func (n cmpNode) keys(add func(string)) { add(n.key) }

// literalText is a literal as the parser reads it back: a number or
// boolean bare, a string in single quotes with \ and ' escaped.
func literalText(v Value) string {
	if v.Kind() != KindString {
		return v.String()
	}
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return "'" + r.Replace(v.String()) + "'"
}

// regexNode matches a key's text against an RE2 regex compiled at parse
// time, so evaluation is linear in the text.
type regexNode struct {
	key            string
	re             *regexp.Regexp
	pattern, flags string
}

func (n regexNode) eval(c Context) bool {
	v, ok := c.Value(n.key)
	return ok && v.Kind() != KindList && n.re.MatchString(v.String())
}

func (n regexNode) str() string           { return n.key + " =~ /" + n.pattern + "/" + n.flags }
func (regexNode) prec() int               { return precAtom }
func (n regexNode) keys(add func(string)) { add(n.key) }

// inNode tests whether a key's value is an item of a list key.
type inNode struct {
	key, list string
	not       bool
}

func (n inNode) eval(c Context) bool {
	in := false
	if v, ok := c.Value(n.key); ok && v.Kind() != KindList {
		if l, ok := c.Value(n.list); ok {
			items, _ := l.List()
			in = slices.Contains(items, v.String())
		}
	}
	return in != n.not
}

func (n inNode) str() string {
	if n.not {
		return n.key + " not in " + n.list
	}
	return n.key + " in " + n.list
}

func (inNode) prec() int { return precAtom }
func (n inNode) keys(add func(string)) {
	add(n.key)
	add(n.list)
}
