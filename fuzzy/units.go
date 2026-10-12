package fuzzy

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// charClass is a cluster's kind, from its first rune, for the bonuses: the
// order is fzf's, and every class after charNonWord is part of a word.
type charClass uint8

const (
	charWhite charClass = iota
	charNonWord
	charDelimiter
	charLower
	charUpper
	charLetter
	charNumber
)

// fzf's scoring constants (src/algo/algo.go), which the tests pin through
// the order they give.
const (
	scoreMatch             = 16
	scoreGapStart          = -3
	scoreGapExtension      = -1
	bonusBoundary          = scoreMatch / 2
	bonusNonWord           = scoreMatch / 2
	bonusCamel123          = bonusBoundary + scoreGapExtension
	bonusConsecutive       = -(scoreGapStart + scoreGapExtension)
	bonusFirstCharMultiple = 2
	bonusBoundaryWhite     = bonusBoundary + 2
	bonusBoundaryDelimiter = bonusBoundary + 1
)

// bonusFor is fzf's bonus for a cluster of class after one of class prev.
func bonusFor(prev, class charClass) int {
	if class > charNonWord {
		switch prev {
		case charWhite:
			return bonusBoundaryWhite
		case charDelimiter:
			return bonusBoundaryDelimiter
		case charNonWord:
			return bonusBoundary
		}
	}
	if prev == charLower && class == charUpper || prev != charNumber && class == charNumber {
		return bonusCamel123
	}
	switch class {
	case charNonWord, charDelimiter:
		return bonusNonWord
	case charWhite:
		return bonusBoundaryWhite
	}
	return 0
}

// unit is one grapheme cluster of a prepared string.
type unit struct {
	key        int64 // the folded cluster: a rune, or a query cluster's id
	start, end int   // its bytes
	class      charClass
	bonus      int // bonusFor the cluster before it, or the start
}

// Keys above every rune: a multi-rune cluster of the query, by its id, and
// a multi-rune cluster the query does not have, which matches nothing.
const (
	clusterKey int64 = 1 << 32
	noKey      int64 = 1 << 40
)

// fold is r's canonical case: the least rune of its simple folding orbit,
// so that every rune of an orbit, such as k, K and the Kelvin sign, folds
// to the same one.
func fold(r rune) rune {
	if r < utf8.RuneSelf {
		if 'a' <= r && r <= 'z' {
			return r - 'a' + 'A'
		}
		return r
	}
	f := r
	for x := unicode.SimpleFold(r); x != r; x = unicode.SimpleFold(x) {
		f = min(f, x)
	}
	return f
}

// foldRune is r as the matcher compares it.
func (m *Matcher) foldRune(r rune) rune {
	if m.norm != nil {
		r = m.norm.Normalize(r)
	}
	if !m.sensitive {
		r = fold(r)
	}
	return r
}

// class is the class of a cluster that starts with r.
func (m *Matcher) class(r rune) charClass {
	switch {
	case 'a' <= r && r <= 'z':
		return charLower
	case 'A' <= r && r <= 'Z':
		return charUpper
	case '0' <= r && r <= '9':
		return charNumber
	case strings.ContainsRune(m.delimiters, r):
		return charDelimiter
	case r < utf8.RuneSelf:
		if r == ' ' || '\t' <= r && r <= '\r' {
			return charWhite
		}
		return charNonWord
	case unicode.IsLower(r):
		return charLower
	case unicode.IsUpper(r):
		return charUpper
	case unicode.IsNumber(r):
		return charNumber
	case unicode.IsLetter(r):
		return charLetter
	case unicode.IsSpace(r):
		return charWhite
	}
	return charNonWord
}

// key is the folded key of cluster c. A multi-rune cluster's key is the id
// the query gave it, or noKey; add, while the query is compiled, gives it
// one.
func (m *Matcher) key(c string, buf *[]byte, add bool) int64 {
	r, n := utf8.DecodeRuneInString(c)
	if n == len(c) {
		return int64(m.foldRune(r))
	}
	b := (*buf)[:0]
	for _, r := range c {
		b = utf8.AppendRune(b, m.foldRune(r))
	}
	*buf = b
	if k, ok := m.clusters[string(b)]; ok {
		return k
	}
	if !add {
		return noKey
	}
	k := clusterKey | int64(len(m.clusters))
	m.clusters[string(b)] = k
	return k
}

// plainASCII reports whether s is ASCII with no carriage return, whose
// clusters are its bytes: CR LF is one cluster.
func plainASCII(s string) bool {
	for i := range len(s) {
		if s[i] >= utf8.RuneSelf || s[i] == '\r' {
			return false
		}
	}
	return true
}

// prepare appends s's clusters to us. slow walks the clusters even for
// ASCII, so tests can compare the two ways.
func (m *Matcher) prepare(us []unit, s string, buf *[]byte, add, slow bool) []unit {
	prev := charWhite
	if !slow && plainASCII(s) {
		for i := range len(s) {
			r := rune(s[i])
			cl := m.class(r)
			us = append(us, unit{key: int64(m.foldRune(r)), start: i, end: i + 1, class: cl, bonus: bonusFor(prev, cl)})
			prev = cl
		}
		return us
	}
	for i := 0; i < len(s); {
		c, _ := ansi.FirstGraphemeCluster(s[i:], ansi.GraphemeWidth)
		if c == "" {
			_, n := utf8.DecodeRuneInString(s[i:])
			c = s[i : i+n]
		}
		r, _ := utf8.DecodeRuneInString(c)
		cl := m.class(r)
		us = append(us, unit{key: m.key(c, buf, add), start: i, end: i + len(c), class: cl, bonus: bonusFor(prev, cl)})
		prev = cl
		i += len(c)
	}
	return us
}
