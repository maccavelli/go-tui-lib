// Package sanitize is the one sanitizer for untrusted text the library puts
// on a screen: a notification, a window title, a vendor name from a beacon,
// a terminal's name from its reply, a pane's title and badge, and a pane's
// view as it is clipped
// (docs/decisions/0014-PLAN-hardening.md Step 2, finding C6).
//
// Line is for one line of plain text. Styled is for text that carries its
// own styles and links, such as a pane's view, and keeps those. Both drop
// the same characters from the text.
//
// Stability: internal.
package sanitize

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// dropped reports whether r never reaches the screen: C0 and DEL, C1, the
// Bidi_Control set, U+200B, U+FEFF, U+2060-2064, U+00AD and the tag
// characters. Line breaks and tabs are not among them: Line and Styled turn
// them into spaces. ZWJ, ZWNJ and variation selectors are kept, since emoji
// and some scripts need them.
func dropped(r rune) bool {
	switch {
	case r < 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f:
		return true
	case r == 0x061c || r == 0x200e || r == 0x200f || r >= 0x202a && r <= 0x202e || r >= 0x2066 && r <= 0x2069:
		return true // Bidi_Control
	case r == 0x200b || r == 0xfeff || r >= 0x2060 && r <= 0x2064 || r == 0x00ad:
		return true
	case r >= 0xe0000 && r <= 0xe007f:
		return true // tags
	}
	return false
}

// isBreak reports whether r is a line break or a tab, which become a space.
func isBreak(r rune) bool {
	return r == '\r' || r == '\n' || r == '\t' || r == 0x2028 || r == 0x2029
}

// text writes s, plain text with no escape sequences, to b: a run of line
// breaks and tabs becomes one space, and what dropped names, and invalid
// UTF-8, is left out. brk carries whether the last thing written was such
// a space, across calls.
func text(b *strings.Builder, s string, brk *bool) {
	for s != "" {
		r, size := utf8.DecodeRuneInString(s)
		switch {
		case r == utf8.RuneError && size == 1:
		case isBreak(r):
			if !*brk {
				b.WriteByte(' ')
				*brk = true
			}
		case dropped(r):
		default:
			b.WriteString(s[:size])
			*brk = false
		}
		s = s[size:]
	}
}

// Line is one line of display text: escape sequences stripped, each run of
// \r, \n, \t, U+2028 and U+2029 turned into one space, and C0, DEL, C1,
// invalid UTF-8, the Bidi_Control set, U+200B, U+FEFF, U+2060-2064, U+00AD
// and the tag characters (U+E0000-E007F) dropped. ZWJ, ZWNJ and variation
// selectors stay.
func Line(s string) string {
	if plain(s) {
		return s // nothing to change, and no allocation
	}
	var b strings.Builder
	b.Grow(len(s))
	brk := false
	text(&b, ansi.Strip(s), &brk)
	return b.String()
}

// Styled is Line for text that carries its own styles and links: each SGR
// and OSC 8 hyperlink that kept allows is kept as it is, and the text
// between them is filtered as Line filters it. Every other sequence, a
// stray ESC, a sequence cut short, and a control sequence introduced by an
// 8-bit C1 byte are dropped.
func Styled(s string) string {
	if styledClean(s) {
		return s // nothing to change, and no allocation: a pane's view, every frame
	}
	var b strings.Builder
	b.Grow(len(s))
	brk := false
	var state byte
	for s != "" {
		seq, width, n, next := ansi.DecodeSequence(s, state, nil)
		if n <= 0 {
			n = 1
			seq = s[:1]
		}
		switch {
		case width == 0 && seq[0] == ansi.ESC:
			if kept(seq) {
				b.WriteString(seq)
			}
		default:
			text(&b, seq, &brk)
		}
		s, state = s[n:], next
	}
	return b.String()
}

// plain reports whether Line would return s unchanged: valid UTF-8 with no
// escape sequence, break, tab or dropped character.
func plain(s string) bool {
	for i := 0; i < len(s); {
		if c := s[i]; c >= 0x20 && c < 0x7f {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 || isBreak(r) || dropped(r) {
			return false
		}
		i += size
	}
	return true
}

// styledClean reports whether Styled would return s unchanged: every escape
// sequence in it complete, and its text as plain requires.
func styledClean(s string) bool {
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c >= 0x20 && c < 0x7f:
			i++
		case c == ansi.ESC:
			seq, _, n, _ := ansi.DecodeSequence(s[i:], 0, nil)
			if n <= 0 || !kept(seq) {
				return false
			}
			i += n
		default:
			r, size := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && size == 1 || isBreak(r) || dropped(r) {
				return false
			}
			i += size
		}
	}
	return true
}

// maxSGRParams is the most parameters an SGR Styled keeps may have:
// x/ansi v0.11.8's parser, which Bubble Tea's renderer runs over every
// frame, holds 32, and panics on the 33rd
// (docs/decisions/0014-PLAN-hardening.md Step 9, D9).
const maxSGRParams = 32

// kept reports whether Styled keeps seq, a sequence that starts with ESC:
// an SGR, a CSI whose parameters are digits, ";" and ":", at most
// maxSGRParams of them, and whose final byte is "m"; or an OSC 8 hyperlink, "ESC ] 8 ; params ; URI" ended by BEL
// or ST, whose parameters and URI are printable ASCII. Anything else a pane
// could send, such as a cursor move, a mode, a character set, DECALN or
// another OSC, would reach the screen outside the pane, and is dropped
// (docs/decisions/0014-PLAN-hardening.md Step 9, D9).
func kept(seq string) bool {
	if body, ok := strings.CutPrefix(seq, "\x1b["); ok {
		params, ok := strings.CutSuffix(body, "m")
		return ok && strings.Trim(params, "0123456789;:") == "" &&
			strings.Count(params, ";")+strings.Count(params, ":") < maxSGRParams
	}
	body, ok := strings.CutPrefix(seq, "\x1b]8;")
	if !ok {
		return false
	}
	if b, ok := strings.CutSuffix(body, "\a"); ok {
		body = b
	} else if b, ok := strings.CutSuffix(body, "\x1b\\"); ok {
		body = b
	} else {
		return false
	}
	for i := range len(body) {
		if c := body[i]; c < 0x20 || c > 0x7e {
			return false
		}
	}
	return strings.Contains(body, ";")
}

// Truncate cuts s to at most cells cells measured with m, and when it cuts
// anything, ends it with tail, kept within cells too. The result never
// measures wider than cells under m. Under WcWidth, x/ansi's Truncate cuts
// by grapheme cluster while its StringWidth adds up runes, so a ZWJ emoji
// sequence, Hangul jamo or a conjunct cut short can stay wider than asked;
// Truncate then cuts rune by rune, keeping escape sequences whole
// (docs/decisions/0014-PLAN-hardening.md Step 9, D8).
func Truncate(s string, cells int, m ansi.Method, tail string) string {
	cells = max(cells, 0)
	if m.StringWidth(s) <= cells {
		return s
	}
	if m != ansi.WcWidth {
		if out := m.Truncate(s, cells, tail); m.StringWidth(out) <= cells {
			return out
		}
	}
	if m.StringWidth(tail) > cells {
		tail = ""
	}
	// Rune widths add up under WcWidth, so the first try fits; the loop
	// guards a method whose widths do not.
	for budget := cells - m.StringWidth(tail); budget >= 0; budget-- {
		if out := byRune(s, budget, tail, m); m.StringWidth(out) <= cells {
			return out
		}
	}
	return byRune(s, 0, "", m)
}

// byRune is s with its text cut, rune by rune, to at most budget cells
// under m, then tail, then every escape sequence of s after the cut, so
// that a style's reset is kept.
func byRune(s string, budget int, tail string, m ansi.Method) string {
	var b strings.Builder
	b.Grow(len(s) + len(tail))
	used, cut := 0, false
	for i := 0; i < len(s); {
		if s[i] == ansi.ESC {
			_, _, n, _ := ansi.DecodeSequence(s[i:], 0, nil)
			n = max(n, 1)
			b.WriteString(s[i : i+n])
			i += n
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		r := s[i : i+size]
		i += size
		if cut {
			continue
		}
		if w := m.StringWidth(r); used+w <= budget {
			b.WriteString(r)
			used += w
			continue
		}
		cut = true
		b.WriteString(tail)
	}
	return b.String()
}

// Token keeps only the runes allowed accepts, for an identifier or a name
// with a fixed alphabet.
func Token(s string, allowed func(rune) bool) string {
	return strings.Map(func(r rune) rune {
		if allowed(r) {
			return r
		}
		return -1
	}, s)
}

// HasControl reports whether Line would change s: whether s holds an escape
// sequence, a line break or tab, or a character Line drops. A validator
// refuses such a string rather than cleaning it.
func HasControl(s string) bool { return Line(s) != s }
