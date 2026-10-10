package when

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type tokKind uint8

const (
	tEOF tokKind = iota
	tLParen
	tRParen
	tNot
	tAnd
	tOr
	tEq
	tNe
	tLt
	tLe
	tGt
	tGe
	tMatch
	tWord
	tString
)

type token struct {
	kind tokKind
	text string // a word, or a quoted string's unescaped text
	pos  int
}

// specials end a word.
const specials = "()!&|=<>'"

// scanner reads tokens from src. A regex is read on request, after =~,
// because its text follows rules of its own.
type scanner struct {
	src string
	pos int
}

// errorf is a *SyntaxError at pos.
func (s *scanner) errorf(pos int, format string, a ...any) error {
	return &SyntaxError{Offset: pos, Msg: fmt.Sprintf(format, a...)}
}

func (s *scanner) skipSpace() {
	for s.pos < len(s.src) {
		r, n := utf8.DecodeRuneInString(s.src[s.pos:])
		if !unicode.IsSpace(r) {
			return
		}
		s.pos += n
	}
}

func (s *scanner) next() (token, error) {
	s.skipSpace()
	start := s.pos
	if s.pos >= len(s.src) {
		return token{kind: tEOF, pos: start}, nil
	}
	two := func(second byte, both, one tokKind) token {
		if s.pos+1 < len(s.src) && s.src[s.pos+1] == second {
			s.pos += 2
			return token{kind: both, pos: start}
		}
		s.pos++
		return token{kind: one, pos: start}
	}
	switch c := s.src[s.pos]; c {
	case '(':
		s.pos++
		return token{kind: tLParen, pos: start}, nil
	case ')':
		s.pos++
		return token{kind: tRParen, pos: start}, nil
	case '!':
		return two('=', tNe, tNot), nil
	case '<':
		return two('=', tLe, tLt), nil
	case '>':
		return two('=', tGe, tGt), nil
	case '&', '|':
		if s.pos+1 < len(s.src) && s.src[s.pos+1] == c {
			s.pos += 2
			if c == '&' {
				return token{kind: tAnd, pos: start}, nil
			}
			return token{kind: tOr, pos: start}, nil
		}
		return token{}, s.errorf(start, "a single %q", c)
	case '=':
		if s.pos+1 < len(s.src) {
			switch s.src[s.pos+1] {
			case '=':
				s.pos += 2
				return token{kind: tEq, pos: start}, nil
			case '~':
				s.pos += 2
				return token{kind: tMatch, pos: start}, nil
			}
		}
		return token{}, s.errorf(start, "a single '='")
	case '\'':
		return s.quoted()
	}
	for s.pos < len(s.src) {
		r, n := utf8.DecodeRuneInString(s.src[s.pos:])
		if unicode.IsSpace(r) || strings.ContainsRune(specials, r) {
			break
		}
		s.pos += n
	}
	return token{kind: tWord, text: s.src[start:s.pos], pos: start}, nil
}

// quoted reads a single-quoted string, in which a backslash takes the
// next character as it is.
func (s *scanner) quoted() (token, error) {
	start := s.pos
	s.pos++
	var b strings.Builder
	for s.pos < len(s.src) {
		c := s.src[s.pos]
		switch c {
		case '\'':
			s.pos++
			return token{kind: tString, text: b.String(), pos: start}, nil
		case '\\':
			if s.pos+1 >= len(s.src) {
				return token{}, s.errorf(start, "an unterminated string")
			}
			_, n := utf8.DecodeRuneInString(s.src[s.pos+1:])
			b.WriteString(s.src[s.pos+1 : s.pos+1+n])
			s.pos += 1 + n
		default:
			b.WriteByte(c)
			s.pos++
		}
	}
	return token{}, s.errorf(start, "an unterminated string")
}

// regex reads /pattern/flags. The pattern is kept as written, a backslash
// escaping the next character, so an escaped '/' stays in it.
func (s *scanner) regex() (pattern, flags string, err error) {
	s.skipSpace()
	start := s.pos
	if s.pos >= len(s.src) || s.src[s.pos] != '/' {
		return "", "", s.errorf(start, "a regex must start with '/'")
	}
	s.pos++
	from := s.pos
	for s.pos < len(s.src) {
		switch s.src[s.pos] {
		case '\\':
			s.pos += 2
			continue
		case '/':
			pattern = s.src[from:s.pos]
			s.pos++
			fl := s.pos
			for s.pos < len(s.src) && strings.IndexByte("ims", s.src[s.pos]) >= 0 {
				s.pos++
			}
			if s.pos < len(s.src) {
				r, _ := utf8.DecodeRuneInString(s.src[s.pos:])
				if !unicode.IsSpace(r) && !strings.ContainsRune(specials, r) {
					return "", "", s.errorf(s.pos, "an unknown regex flag %q", r)
				}
			}
			return pattern, s.src[fl:s.pos], nil
		}
		s.pos++
	}
	return "", "", s.errorf(start, "an unterminated regex")
}
