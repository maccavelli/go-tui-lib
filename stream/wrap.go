package stream

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// wrapProse breaks one styled prose line, a paragraph's, a list item's or
// a block quote's, at its spaces so no row is wider than width cells under
// m, gobble-cli's wrapping (its render/wrap.go, 0028-MADR D3 to D5 and D7
// as amended) on a whole line: escape sequences count as no width and stay
// with their word; continuation rows are indented under a list item's
// text and repeat a quote's markers; no row starts with a word that would
// open a block, the word before it moving down with it when it can; a
// word wider than its row is left whole on a row of its own, as D7 has it,
// and a run of CJK or Thai text is one word, which the terminal wraps
// (A3 item 16, UAX #14 §3.1). A width of 0 or less wraps nothing. Each
// word and gap is a slice of the line, and each word is measured once (A9
// item 4).
func wrapProse(m ansi.Method, width int, line string) string {
	if width <= 0 {
		return line
	}
	plain := ansi.Strip(line)
	n := prosePrefix(plain)
	head, rest := cutPlain(line, n)
	cont := strings.Map(func(r rune) rune {
		if r == '>' || r == '\t' {
			return r
		}
		return ' '
	}, plain[:n])
	w := wrapper{m: m, width: width, cont: cont, col: advance(0, plain[:n])}
	w.out.WriteString(head)
	gs, ws, gap := 0, -1, "" // the gap's start, the word's start or -1, and the gap before the word
	for i := 0; i < len(rest); {
		switch c := rest[i]; {
		case c == ' ' || c == '\t':
			if ws >= 0 {
				w.place(gap, rest[ws:i])
				ws, gs = -1, i
			}
			i++
			continue
		case ws < 0:
			ws, gap = i, rest[gs:i]
		}
		if rest[i] == 0x1b {
			i += escapeLen(rest[i:])
		} else {
			i++
		}
	}
	if ws >= 0 {
		w.place(gap, rest[ws:])
		gs = len(rest)
	}
	w.emitHeld()
	if gap = rest[gs:]; w.fits(advance(w.col, gap)) {
		w.out.WriteString(gap)
	}
	return w.out.String()
}

// wrapper is wrapProse's row: the columns used, and the last word placed,
// held back because the next may move it to a new row.
type wrapper struct {
	m         ansi.Method
	width     int
	cont      string
	out       strings.Builder
	col       int
	held      string
	heldW     int // held's width
	heldGap   string
	heldFirst bool
}

func (w *wrapper) fits(col int) bool { return col <= w.width }

// place puts word, after the spaces gap, as gobble's Wrap.endWord does:
// after the held word if it fits; otherwise on a new row, unless it would
// open a block there, when the held word moves down with it, or, when that
// cannot be, the word stays on this row.
func (w *wrapper) place(gap, word string) {
	ww := wordWidth(w.m, word)
	switch {
	case w.held == "":
		w.held, w.heldW, w.heldGap, w.heldFirst = word, ww, gap, true
		return
	case w.fits(ww + advance(w.heldW+advance(w.col, w.heldGap), gap)):
	case !opensBlock(ansi.Strip(word)):
		w.emitHeld()
		w.newRow()
		w.held, w.heldW, w.heldGap, w.heldFirst = word, ww, "", true
		return
	case !w.heldFirst && !opensBlock(ansi.Strip(w.held)):
		w.newRow()
		w.heldGap = ""
	}
	w.emitHeld()
	w.held, w.heldW, w.heldGap, w.heldFirst = word, ww, gap, false
}

// wordWidth is s's width in cells under m: its length when it is printable
// ASCII alone, which every method measures so.
func wordWidth(m ansi.Method, s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return m.StringWidth(s)
		}
	}
	return len(s)
}

func (w *wrapper) emitHeld() {
	if w.held == "" {
		return
	}
	w.out.WriteString(w.heldGap)
	w.out.WriteString(w.held)
	w.col = advance(w.col, w.heldGap) + w.heldW
	w.held, w.heldGap = "", ""
}

func (w *wrapper) newRow() {
	w.out.WriteByte('\n')
	w.out.WriteString(w.cont)
	w.col = advance(0, w.cont)
}

// advance is the column after the spaces, tabs and markers of s are
// written from col: a tab to the next multiple of 8, any other byte one.
func advance(col int, s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '\t' {
			col = (col/8 + 1) * 8
			continue
		}
		col++
	}
	return col
}

// prosePrefix is the length of a prose line's prefix: its indentation, its
// quote markers and the spaces after each, and a list marker and the
// spaces after it.
func prosePrefix(s string) int {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for i < len(s) && s[i] == '>' {
		i++
		for i < len(s) && s[i] == ' ' {
			i++
		}
	}
	if i < len(s) && s[i] != '>' {
		if n := markerLen(s[i:]); n > 0 {
			i += n
			for i < len(s) && s[i] == ' ' {
				i++
			}
		}
	}
	return i
}

// cutPlain cuts s after its first n bytes that are not in an escape
// sequence, the escapes among them and right after them kept with the
// head, so that a marker's reset is not taken for a word.
func cutPlain(s string, n int) (head, rest string) {
	i := 0
	for plain := 0; i < len(s) && (plain < n || s[i] == 0x1b); {
		if s[i] == 0x1b {
			i += escapeLen(s[i:])
			continue
		}
		i++
		plain++
	}
	return s[:i], s[i:]
}

// escapeLen is the length of the escape sequence s starts with: a control
// sequence through its final byte, or ESC and one byte.
func escapeLen(s string) int {
	if len(s) < 2 || s[1] != '[' {
		return min(2, len(s))
	}
	j := 2
	for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
		j++
	}
	return min(j+1, len(s))
}

// opensBlock reports whether word, at a row's start, would open a block in
// Markdown: a list or quote marker, a heading's hashes, a table's pipe, a
// fence, a thematic break or a setext underline (gobble-cli's 0028-MADR D4).
func opensBlock(word string) bool {
	switch {
	case word == "":
		return false
	case word[0] == '>' || word[0] == '|':
		return true
	case word[0] == '#':
		return atxHeading(word)
	case strings.Trim(word, "-*_=+") == "":
		return true
	case fenceRun(word) != "":
		return true
	case word[0] >= '0' && word[0] <= '9':
		return markerLen(word+" ") > 0
	}
	return false
}
