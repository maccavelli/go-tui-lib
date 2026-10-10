// Package when evaluates availability expressions over context keys, in VS
// Code's when-clause grammar, so a command can say when it applies and a
// keybinding file written for VS Code's grammar carries over
// (docs/decisions/0006-MADR-command-registry.md §8).
//
// The grammar, loosest first:
//
//	expr  = or
//	or    = and { "||" and }
//	and   = not { "&&" not }
//	not   = "!" not | cmp
//	cmp   = term [ ( "==" | "!=" | "<" | "<=" | ">" | ">=" ) value
//	             | "=~" /regex/flags
//	             | [ "not" ] "in" key ]
//	term  = key | true | false | "(" expr ")"
//	value = number | 'quoted string' | bareword | true | false
//
// A key alone is true when it is set and not false, 0, "" or an empty list.
// A key that is not set is unequal to everything, so x != 'a' holds when x
// is unset, as in VS Code. Numbers compare as numbers; other values by
// their text. `in` tests whether a key's value is an item of a list key.
// A regex is RE2, compiled when the expression is parsed, with the flags
// i, m and s, so evaluation is linear and cannot be slowed by a pattern.
// Spaces around operators are optional.
//
// Source longer than MaxSource bytes, or nested deeper than MaxDepth, is an
// error, so a user's file cannot exhaust the stack. So is an expression
// whose canonical form is longer than MaxSource, so that every expression
// Parse accepts round-trips. The package imports only the standard
// library.
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package when

import (
	"fmt"
	"iter"
)

// Limits on what Parse accepts.
const (
	MaxSource = 4096 // bytes of source
	MaxDepth  = 64   // nested parentheses and negations
)

// Expr is a parsed expression. Its zero value is false.
type Expr struct{ root node }

// Parse compiles src.
func Parse(src string) (Expr, error) {
	if len(src) > MaxSource {
		return Expr{}, fmt.Errorf("when: source of %d bytes, more than %d", len(src), MaxSource)
	}
	p := parser{s: scanner{src: src}}
	if err := p.advance(); err != nil {
		return Expr{}, err
	}
	root, err := p.or()
	if err != nil {
		return Expr{}, err
	}
	if p.tok.kind != tEOF {
		return Expr{}, p.s.errorf(p.tok.pos, "an unexpected token")
	}
	// The canonical form can be longer than the source, and must parse
	// too (docs/decisions/0006-MADR-command-registry.md A3).
	if n := len(root.str()); n > MaxSource {
		return Expr{}, fmt.Errorf("when: canonical form of %d bytes, more than %d", n, MaxSource)
	}
	return Expr{root: root}, nil
}

// MustParse is Parse, panicking on an error: for expressions written in
// code.
func MustParse(src string) Expr {
	e, err := Parse(src)
	if err != nil {
		panic(err)
	}
	return e
}

// Eval is e's truth in c.
func (e Expr) Eval(c Context) bool {
	if e.root == nil {
		return false
	}
	if c == nil {
		c = Map(nil)
	}
	return e.root.eval(c)
}

// String is e's canonical form: single spaces around binary operators,
// parentheses only where precedence needs them, strings in single quotes.
// Parsing it gives an expression with the same canonical form, for every
// expression Parse accepts.
func (e Expr) String() string {
	if e.root == nil {
		return ""
	}
	return e.root.str()
}

// Keys is every key e reads, in order of first appearance.
func (e Expr) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		if e.root == nil {
			return
		}
		seen := map[string]bool{}
		var order []string
		e.root.keys(func(k string) {
			if !seen[k] {
				seen[k] = true
				order = append(order, k)
			}
		})
		for _, k := range order {
			if !yield(k) {
				return
			}
		}
	}
}

// Check reports what in e does not fit known: a key it does not name, a
// key compared as a number that is not one, a number compared with a
// string, a list compared with ==, and an `in` whose right side is not a
// list. Run at load time, it turns a typo into an error instead of a
// command that never appears.
func Check(e Expr, known Keys) []error {
	var errs []error
	reported := map[string]bool{}
	kind := func(k string) (Kind, bool) {
		kd, ok := known[k]
		if !ok && !reported[k] {
			reported[k] = true
			errs = append(errs, fmt.Errorf("when: unknown key %q", k))
		}
		return kd, ok
	}
	var walk func(n node)
	walk = func(n node) {
		switch n := n.(type) {
		case orNode:
			walk(n.x)
			walk(n.y)
		case andNode:
			walk(n.x)
			walk(n.y)
		case notNode:
			walk(n.x)
		case keyNode:
			kind(string(n))
		case regexNode:
			if kd, ok := kind(n.key); ok && kd == KindList {
				errs = append(errs, fmt.Errorf("when: key %q is a list, matched with =~", n.key))
			}
		case inNode:
			kind(n.key)
			if kd, ok := kind(n.list); ok && kd != KindList {
				errs = append(errs, fmt.Errorf("when: key %q is a %s, not a list, after in", n.list, kd))
			}
		case cmpNode:
			errs = append(errs, checkCmp(n, kind)...)
		}
	}
	if e.root != nil {
		walk(e.root)
	}
	return errs
}

func checkCmp(n cmpNode, kind func(string) (Kind, bool)) []error {
	kd, ok := kind(n.key)
	if !ok {
		return nil
	}
	switch n.op {
	case tLt, tLe, tGt, tGe:
		if kd != KindNumber {
			return []error{fmt.Errorf("when: key %q is a %s, compared as a number", n.key, kd)}
		}
		if n.lit.Kind() != KindNumber {
			return []error{fmt.Errorf("when: key %q is compared with %s, which is not a number", n.key, literalText(n.lit))}
		}
	default:
		if kd == KindList {
			return []error{fmt.Errorf("when: key %q is a list, compared with %s", n.key, opText[n.op])}
		}
		if kd == KindNumber && n.lit.Kind() == KindString {
			return []error{fmt.Errorf("when: key %q is a number, compared with the string %s", n.key, literalText(n.lit))}
		}
	}
	return nil
}
