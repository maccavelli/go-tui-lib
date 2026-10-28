// Package lineinput is the text a user types and its cursor, edited by
// commands: lineedit's Composer, ask's reason and pick's query
// (docs/decisions/0023-MADR-line-mode-components.md A2 item 1, A5). A
// Line holds the text as grapheme clusters under the width method, so a
// cluster is one cursor step and its true width; bounds it, in bytes or
// in runes; edits it by the commands lineedit names, Readline's; and shows
// the part around the cursor that fits a width.
//
// Stability: internal.
package lineinput

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/command"
)

// The commands a Line runs, by lineedit's IDs: lineedit exports each, and
// its test checks the two lists agree. The moves and deletions that need
// the layout, the history or the host are the Composer's.
const (
	CharLeft            command.ID = "lineedit.char.left"
	CharRight           command.ID = "lineedit.char.right"
	WordLeft            command.ID = "lineedit.word.left"
	WordRight           command.ID = "lineedit.word.right"
	LineStart           command.ID = "lineedit.line.start"
	LineEnd             command.ID = "lineedit.line.end"
	DeleteBack          command.ID = "lineedit.delete.back"
	DeleteForward       command.ID = "lineedit.delete.forward"
	DeleteOrEOF         command.ID = "lineedit.delete.forward-or-eof"
	DeleteWordBack      command.ID = "lineedit.delete.word-back"
	DeleteWordForward   command.ID = "lineedit.delete.word-forward"
	DeleteSpaceWordBack command.ID = "lineedit.delete.space-word-back"
	DeleteToLineEnd     command.ID = "lineedit.delete.to-line-end"
	DeleteToLineStart   command.ID = "lineedit.delete.to-line-start"
	Transpose           command.ID = "lineedit.transpose"
)

// Commands is the IDs Do runs, in the order above.
func Commands() []command.ID {
	return []command.ID{
		CharLeft, CharRight, WordLeft, WordRight, LineStart, LineEnd,
		DeleteBack, DeleteForward, DeleteOrEOF, DeleteWordBack, DeleteWordForward,
		DeleteSpaceWordBack, DeleteToLineEnd, DeleteToLineStart, Transpose,
	}
}

// Line is the text and its cursor. Build one with New or NewRunes.
type Line struct {
	method   ansi.Method
	limit    int
	runes    bool // the limit counts runes, not bytes
	clusters []string
	pos      int // the cursor, in clusters from the start
}

// New is an empty Line of at most limit bytes, measured with m. A limit
// of 0 or less is no bound.
func New(limit int, m ansi.Method) *Line { return &Line{method: m, limit: limit} }

// NewRunes is an empty Line of at most limit runes, as x/term bounds its
// line.
func NewRunes(limit int, m ansi.Method) *Line {
	return &Line{method: m, limit: limit, runes: true}
}

// Text is the line's text.
func (l *Line) Text() string { return strings.Join(l.clusters, "") }

// Clusters is the line's grapheme clusters, a "\n" its own; the caller
// does not change them.
func (l *Line) Clusters() []string { return l.clusters }

// Pos is the cursor, in clusters from the start.
func (l *Line) Pos() int { return l.pos }

// SetPos puts the cursor at p clusters, within the line.
func (l *Line) SetPos(p int) { l.pos = min(max(p, 0), len(l.clusters)) }

// SetText makes s the line's text, cut to the bound by whole clusters, and
// puts the cursor at its end.
func (l *Line) SetText(s string) {
	l.clusters, l.pos = l.clusters[:0], 0
	l.Insert(s)
}

// size is s's length as the bound counts it.
func (l *Line) size(s string) int {
	if l.runes {
		return utf8.RuneCountInString(s)
	}
	return len(s)
}

// Insert puts s at the cursor and moves the cursor past it; what would
// take the line over its bound is left out, by whole clusters. s is
// display text the caller has cleaned: a line break in it is a cluster of
// its own.
func (l *Line) Insert(s string) {
	before, after := strings.Join(l.clusters[:l.pos], ""), strings.Join(l.clusters[l.pos:], "")
	if l.limit > 0 {
		room := l.limit - l.size(before) - l.size(after)
		cut := 0
		for rest := s; rest != ""; {
			c, _ := ansi.FirstGraphemeCluster(rest, l.method)
			if l.size(s[:cut+len(c)]) > room {
				break
			}
			cut += len(c)
			rest = rest[len(c):]
		}
		s = s[:cut]
	}
	l.set(before+s+after, len(before)+len(s))
}

// set makes text the line's, its cursor at the cluster that starts at or
// after byte at: an insert may join a cluster with the one before it.
func (l *Line) set(text string, at int) {
	l.clusters, l.pos = l.clusters[:0], 0
	n := 0
	for text != "" {
		c, _ := ansi.FirstGraphemeCluster(text, l.method)
		if n < at {
			l.pos++
		}
		l.clusters = append(l.clusters, c)
		n += len(c)
		text = text[len(c):]
	}
}

// Do runs the command id on the line, and reports whether id is one of its
// commands. The words are Readline's: a run of clusters whose first rune is
// a letter or a digit; alt+f's motion stops at a word's end, alt+b's at its
// start (0023-MADR A5 item 6). The line's ends are the line break's around
// the cursor, or the text's.
func (l *Line) Do(id command.ID) bool {
	switch id {
	case CharLeft:
		l.pos = max(l.pos-1, 0)
	case CharRight:
		l.pos = min(l.pos+1, len(l.clusters))
	case WordLeft:
		l.pos = l.wordStart()
	case WordRight:
		l.pos = l.wordEnd()
	case LineStart:
		l.pos = l.lineStart()
	case LineEnd:
		l.pos = l.lineEnd()
	case DeleteBack:
		if l.pos > 0 {
			l.cut(l.pos-1, l.pos)
		}
	case DeleteForward, DeleteOrEOF:
		if l.pos < len(l.clusters) {
			l.cut(l.pos, l.pos+1)
		}
	case DeleteWordBack:
		l.cut(l.wordStart(), l.pos)
	case DeleteWordForward:
		l.cut(l.pos, l.wordEnd())
	case DeleteSpaceWordBack:
		from := l.pos
		for from > 0 && space(l.clusters[from-1]) {
			from--
		}
		for from > 0 && !space(l.clusters[from-1]) {
			from--
		}
		l.cut(from, l.pos)
	case DeleteToLineEnd:
		l.cut(l.pos, l.lineEnd())
	case DeleteToLineStart:
		l.cut(l.lineStart(), l.pos)
	case Transpose:
		l.transpose()
	default:
		return false
	}
	return true
}

// word reports whether cluster c is part of a word: its first rune a
// letter or a digit, as Readline's words are.
func word(c string) bool {
	r, _ := utf8.DecodeRuneInString(c)
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// space reports whether cluster c is white space, unix-word-rubout's
// boundary.
func space(c string) bool {
	r, _ := utf8.DecodeRuneInString(c)
	return unicode.IsSpace(r)
}

// wordStart is the start of the word the cursor is in or after: back over
// what is not a word, then over the word.
func (l *Line) wordStart() int {
	p := l.pos
	for p > 0 && !word(l.clusters[p-1]) {
		p--
	}
	for p > 0 && word(l.clusters[p-1]) {
		p--
	}
	return p
}

// wordEnd is the end of the word the cursor is in or before: forward over
// what is not a word, then over the word.
func (l *Line) wordEnd() int {
	p := l.pos
	for p < len(l.clusters) && !word(l.clusters[p]) {
		p++
	}
	for p < len(l.clusters) && word(l.clusters[p]) {
		p++
	}
	return p
}

// lineStart is the cluster after the line break before the cursor, or 0.
func (l *Line) lineStart() int {
	p := l.pos
	for p > 0 && l.clusters[p-1] != "\n" {
		p--
	}
	return p
}

// lineEnd is the line break after the cursor, or the end.
func (l *Line) lineEnd() int {
	p := l.pos
	for p < len(l.clusters) && l.clusters[p] != "\n" {
		p++
	}
	return p
}

// transpose drags the cluster before the cursor over the one at it, and
// moves past it; at the end, it swaps the last two (Readline's
// transpose-chars).
func (l *Line) transpose() {
	if len(l.clusters) < 2 || l.pos == 0 {
		return
	}
	at := min(l.pos, len(l.clusters)-1)
	c := append([]string(nil), l.clusters...)
	c[at-1], c[at] = c[at], c[at-1]
	text, pos := strings.Join(c, ""), len(strings.Join(c[:at+1], ""))
	l.set(text, pos)
}

// cut removes the clusters from i to j, the cursor left at i.
func (l *Line) cut(i, j int) {
	if i >= j {
		l.pos = i
		return
	}
	text := strings.Join(l.clusters[:i], "") + strings.Join(l.clusters[j:], "")
	l.set(text, len(strings.Join(l.clusters[:i], "")))
}

// View is the part of the line that fits width cells with the cursor on
// it, and the cursor's column in that part. The cursor's cell is counted,
// a cell past the text at its end. A width of 0 or less shows it all.
func (l *Line) View(width int) (string, int) {
	if width <= 0 {
		return l.Text(), l.method.StringWidth(strings.Join(l.clusters[:l.pos], ""))
	}
	w := func(i int) int { return l.method.StringWidth(l.clusters[i]) }
	used, b := 1, l.pos
	if l.pos < len(l.clusters) {
		used, b = w(l.pos), l.pos+1
	}
	a := l.pos
	for a > 0 && used+w(a-1) <= width {
		a--
		used += w(a)
	}
	for b < len(l.clusters) && used+w(b) <= width {
		used += w(b)
		b++
	}
	return strings.Join(l.clusters[a:b], ""), l.method.StringWidth(strings.Join(l.clusters[a:l.pos], ""))
}
