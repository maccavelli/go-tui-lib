// Package sanitize is the one sanitizer for untrusted text the library puts
// on a screen: a notification, a window title, a vendor name from a beacon,
// a terminal's name from its reply, a pane's title and badge, and a pane's
// view as it is clipped
// (docs/decisions/0014-PLAN-hardening.md Step 2, finding C6).
//
// Line is for one line of plain text. Styled is for text that carries its
// own escape sequences, such as a pane's view, and keeps them. Both drop the
// same characters from the text.
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

// Styled is Line for text that carries its own escape sequences: each
// complete sequence introduced by ESC is kept as it is, and the text
// between them is filtered as Line filters it. A stray ESC, a sequence cut
// short, and a control sequence introduced by an 8-bit C1 byte are
// dropped.
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
			if complete(seq) {
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
			if n <= 0 || !complete(seq) {
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

// complete reports whether seq, which starts with ESC, is a whole escape
// sequence: a CSI with its final byte, a string sequence (OSC, DCS, APC,
// SOS or PM) with its BEL or ST terminator, or an ESC with its final byte.
func complete(seq string) bool {
	if len(seq) < 2 {
		return false
	}
	last := seq[len(seq)-1]
	switch seq[1] {
	case '[':
		return len(seq) >= 3 && last >= 0x40 && last <= 0x7e
	case ']', 'P', '_', 'X', '^':
		return strings.HasSuffix(seq, "\a") && seq[1] == ']' || strings.HasSuffix(seq, "\x1b\\") && len(seq) >= 4
	}
	return last >= 0x30 && last <= 0x7e
}

// Truncate cuts s to at most cells cells measured with m, adding nothing.
func Truncate(s string, cells int, m ansi.Method) string {
	return m.Truncate(s, max(cells, 0), "")
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
