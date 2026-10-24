// The oracle: internal/sanitize/sanitize.go as `git show
// v0.12.0:internal/sanitize/sanitize.go` prints it, with Line, Styled,
// HasControl and their helpers renamed with an oracle prefix, and the
// package comment, Truncate, byRune and Token left out; nothing else is
// changed. FuzzAgainstV0120 holds the package's functions to it
// (docs/decisions/0020-PLAN-charm-free-cores-glyphs-and-safetext.md Step 5).

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
func oracleDropped(r rune) bool {
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
func oracleIsBreak(r rune) bool {
	return r == '\r' || r == '\n' || r == '\t' || r == 0x2028 || r == 0x2029
}

// text writes s, plain text with no escape sequences, to b: a run of line
// breaks and tabs becomes one space, and what dropped names, and invalid
// UTF-8, is left out. brk carries whether the last thing written was such
// a space, across calls.
func oracleText(b *strings.Builder, s string, brk *bool) {
	for s != "" {
		r, size := utf8.DecodeRuneInString(s)
		switch {
		case r == utf8.RuneError && size == 1:
		case oracleIsBreak(r):
			if !*brk {
				b.WriteByte(' ')
				*brk = true
			}
		case oracleDropped(r):
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
func oracleLine(s string) string {
	if oraclePlain(s) {
		return s // nothing to change, and no allocation
	}
	var b strings.Builder
	b.Grow(len(s))
	brk := false
	oracleText(&b, ansi.Strip(s), &brk)
	return b.String()
}

// Styled is Line for text that carries its own styles and links: each SGR
// and OSC 8 hyperlink that kept allows is kept as it is, and the text
// between them is filtered as Line filters it. Every other sequence, a
// stray ESC, a sequence cut short, and a control sequence introduced by an
// 8-bit C1 byte are dropped.
func oracleStyled(s string) string {
	if oracleStyledClean(s) {
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
			if oracleKept(seq) {
				b.WriteString(seq)
			}
		default:
			oracleText(&b, seq, &brk)
		}
		s, state = s[n:], next
	}
	return b.String()
}

// plain reports whether Line would return s unchanged: valid UTF-8 with no
// escape sequence, break, tab or dropped character.
func oraclePlain(s string) bool {
	for i := 0; i < len(s); {
		if c := s[i]; c >= 0x20 && c < 0x7f {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 || oracleIsBreak(r) || oracleDropped(r) {
			return false
		}
		i += size
	}
	return true
}

// styledClean reports whether Styled would return s unchanged: every escape
// sequence in it complete, and its text as plain requires.
func oracleStyledClean(s string) bool {
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c >= 0x20 && c < 0x7f:
			i++
		case c == ansi.ESC:
			seq, _, n, _ := ansi.DecodeSequence(s[i:], 0, nil)
			if n <= 0 || !oracleKept(seq) {
				return false
			}
			i += n
		default:
			r, size := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && size == 1 || oracleIsBreak(r) || oracleDropped(r) {
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
const oracleMaxSGRParams = 32

// kept reports whether Styled keeps seq, a sequence that starts with ESC:
// an SGR, a CSI whose parameters are digits, ";" and ":", at most
// maxSGRParams of them, and whose final byte is "m"; or an OSC 8 hyperlink, "ESC ] 8 ; params ; URI" ended by BEL
// or ST, whose parameters and URI are printable ASCII. Anything else a pane
// could send, such as a cursor move, a mode, a character set, DECALN or
// another OSC, would reach the screen outside the pane, and is dropped
// (docs/decisions/0014-PLAN-hardening.md Step 9, D9).
func oracleKept(seq string) bool {
	if body, ok := strings.CutPrefix(seq, "\x1b["); ok {
		params, ok := strings.CutSuffix(body, "m")
		return ok && strings.Trim(params, "0123456789;:") == "" &&
			strings.Count(params, ";")+strings.Count(params, ":") < oracleMaxSGRParams
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

// HasControl reports whether Line would change s: whether s holds an escape
// sequence, a line break or tab, or a character Line drops. A validator
// refuses such a string rather than cleaning it.
func oracleHasControl(s string) bool { return oracleLine(s) != s }
