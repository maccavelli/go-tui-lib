package when

import (
	"regexp"
	"slices"
	"strconv"
)

// keyPattern is a key's syntax: a letter or underscore, then letters,
// digits and _ . : -.
var keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.:\-]*$`)

// numberPattern is a number's syntax on the right of a comparison. RE2's
// \d is ASCII digits only.
var numberPattern = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

// The boolean words, as a term and as a value.
const (
	trueWord  = "true"
	falseWord = "false"
)

type parser struct {
	s     scanner
	tok   token
	depth int
}

func (p *parser) advance() error {
	t, err := p.s.next()
	if err != nil {
		return err
	}
	p.tok = t
	return nil
}

func (p *parser) enter() error {
	p.depth++
	if p.depth > MaxDepth {
		return p.s.errorf(p.tok.pos, "nesting deeper than %d", MaxDepth)
	}
	return nil
}

// or = and { "||" and }
func (p *parser) or() (node, error) {
	x, err := p.and()
	if err != nil {
		return nil, err
	}
	for p.tok.kind == tOr {
		if err := p.advance(); err != nil {
			return nil, err
		}
		y, err := p.and()
		if err != nil {
			return nil, err
		}
		x = orNode{x, y}
	}
	return x, nil
}

// and = not { "&&" not }
func (p *parser) and() (node, error) {
	x, err := p.not()
	if err != nil {
		return nil, err
	}
	for p.tok.kind == tAnd {
		if err := p.advance(); err != nil {
			return nil, err
		}
		y, err := p.not()
		if err != nil {
			return nil, err
		}
		x = andNode{x, y}
	}
	return x, nil
}

// not = "!" not | cmp
func (p *parser) not() (node, error) {
	if p.tok.kind != tNot {
		return p.cmp()
	}
	if err := p.enter(); err != nil {
		return nil, err
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	x, err := p.not()
	if err != nil {
		return nil, err
	}
	p.depth--
	return notNode{x}, nil
}

// cmp = term [ op term | "=~" regex | [ "not" ] "in" key ]
func (p *parser) cmp() (node, error) {
	switch p.tok.kind {
	case tLParen:
		if err := p.enter(); err != nil {
			return nil, err
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
		x, err := p.or()
		if err != nil {
			return nil, err
		}
		if p.tok.kind != tRParen {
			return nil, p.s.errorf(p.tok.pos, "a missing ')'")
		}
		p.depth--
		return x, p.advance()
	case tWord:
	case tString:
		return nil, p.s.errorf(p.tok.pos, "a string alone is not a condition")
	case tEOF:
		return nil, p.s.errorf(p.tok.pos, "an expression that ends too soon")
	default:
		return nil, p.s.errorf(p.tok.pos, "an unexpected operator")
	}

	word := p.tok.text
	pos := p.tok.pos
	switch word {
	case trueWord, falseWord:
		return constNode(word == trueWord), p.advance()
	}
	if !keyPattern.MatchString(word) {
		return nil, p.s.errorf(pos, "%q is not a key", word)
	}
	// Past the key. When the next token is =~, the scanner stops just after
	// it, where the regex, which is not made of tokens, is read.
	if err := p.advance(); err != nil {
		return nil, err
	}
	switch p.tok.kind {
	case tEq, tNe, tLt, tLe, tGt, tGe:
		op := p.tok.kind
		if err := p.advance(); err != nil {
			return nil, err
		}
		lit, err := p.literal()
		if err != nil {
			return nil, err
		}
		return cmpNode{op: op, key: word, lit: lit}, nil
	case tMatch:
		pattern, flags, err := p.s.regex()
		if err != nil {
			return nil, err
		}
		re, err := regexp.Compile(flagPrefix(flags) + pattern)
		if err != nil {
			return nil, p.s.errorf(pos, "a bad regex: %v", err)
		}
		return regexNode{key: word, re: re, pattern: pattern, flags: flags}, p.advance()
	case tWord:
		not := p.tok.text == "not"
		if not {
			if err := p.advance(); err != nil {
				return nil, err
			}
		}
		if p.tok.kind != tWord || p.tok.text != "in" {
			if not {
				return nil, p.s.errorf(p.tok.pos, "'not' without 'in'")
			}
			return keyNode(word), nil
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
		if p.tok.kind != tWord || !keyPattern.MatchString(p.tok.text) {
			return nil, p.s.errorf(p.tok.pos, "'in' needs a key")
		}
		list := p.tok.text
		return inNode{key: word, list: list, not: not}, p.advance()
	}
	return keyNode(word), nil
}

// literal is the value on the right of a comparison: a quoted string, a
// number, true, false, or a bare word taken as a string.
func (p *parser) literal() (Value, error) {
	t := p.tok
	switch t.kind {
	case tString:
		return StringValue(t.text), p.advance()
	case tWord:
	default:
		return Value{}, p.s.errorf(t.pos, "a comparison without a value")
	}
	var v Value
	switch {
	case t.text == trueWord || t.text == falseWord:
		v = BoolValue(t.text == trueWord)
	case numberPattern.MatchString(t.text):
		f, err := strconv.ParseFloat(t.text, 64)
		if err != nil {
			return Value{}, p.s.errorf(t.pos, "a bad number %q", t.text)
		}
		v = NumberValue(f)
	default:
		v = StringValue(t.text)
	}
	return v, p.advance()
}

// flagPrefix turns a regex's flags into RE2's inline form.
func flagPrefix(flags string) string {
	if flags == "" {
		return ""
	}
	f := []byte(flags)
	slices.Sort(f)
	return "(?" + string(slices.Compact(f)) + ")"
}
