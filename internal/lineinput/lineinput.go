// Package lineinput is one row of text a user types, such as the reason
// after a question's key or a picker's query
// (docs/decisions/0023-MADR-line-mode-components.md A2 item 1). A Line
// holds the text as grapheme clusters under the width method, so a cluster
// is one cursor step and its true width; bounds it in bytes; edits it by a
// fixed set of keys; and shows the part around the cursor that fits a
// width. lineedit's Composer is to be built on it.
//
// Stability: internal.
package lineinput

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// Line is the text and its cursor. Build one with New.
type Line struct {
	method   ansi.Method
	limit    int
	clusters []string
	pos      int // the cursor, in clusters from the start
}

// New is an empty Line of at most limit bytes, measured with m. A limit
// of 0 or less is no bound.
func New(limit int, m ansi.Method) *Line { return &Line{method: m, limit: limit} }

// Text is the line's text.
func (l *Line) Text() string { return strings.Join(l.clusters, "") }

// Insert puts s at the cursor, cleaned as one line of display text, and
// moves the cursor past it; what would take the line over its bound is
// left out, by whole clusters.
func (l *Line) Insert(s string) {
	s = sanitize.Line(s)
	before, after := strings.Join(l.clusters[:l.pos], ""), strings.Join(l.clusters[l.pos:], "")
	if l.limit > 0 {
		room := l.limit - len(before) - len(after)
		cut := 0
		for rest := s; rest != ""; {
			c, _ := ansi.FirstGraphemeCluster(rest, l.method)
			if cut+len(c) > room {
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

// Key edits the line by k, and reports whether k is one of its keys: text
// typed, backspace and ctrl+h, delete and ctrl+d, left and ctrl+b, right
// and ctrl+f, home and ctrl+a, end and ctrl+e, ctrl+u to the start, ctrl+k
// to the end, and ctrl+w and alt+backspace, the word before the cursor. A
// key released is none of them.
func (l *Line) Key(k termkey.KeyEvent) bool {
	if k.Release {
		return false
	}
	switch k.Keystroke() {
	case "backspace", "ctrl+h":
		if l.pos > 0 {
			l.cut(l.pos-1, l.pos)
		}
	case "delete", "ctrl+d":
		if l.pos < len(l.clusters) {
			l.cut(l.pos, l.pos+1)
		}
	case "left", "ctrl+b":
		l.pos = max(l.pos-1, 0)
	case "right", "ctrl+f":
		l.pos = min(l.pos+1, len(l.clusters))
	case "home", "ctrl+a":
		l.pos = 0
	case "end", "ctrl+e":
		l.pos = len(l.clusters)
	case "ctrl+u":
		l.cut(0, l.pos)
	case "ctrl+k":
		l.cut(l.pos, len(l.clusters))
	case "ctrl+w", "alt+backspace":
		from := l.pos
		for from > 0 && l.clusters[from-1] == " " {
			from--
		}
		for from > 0 && l.clusters[from-1] != " " {
			from--
		}
		l.cut(from, l.pos)
	default:
		if k.Text == "" || k.Mod&^(termkey.ModShift|termkey.ModCapsLock|termkey.ModNumLock|termkey.ModScrollLock) != 0 {
			return false
		}
		l.Insert(k.Text)
	}
	return true
}

// cut removes the clusters from i to j, the cursor left at i.
func (l *Line) cut(i, j int) {
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
