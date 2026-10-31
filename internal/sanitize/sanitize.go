// Package sanitize is the one sanitizer for untrusted text the library puts
// on a screen: a notification, a window title, a vendor name from a beacon,
// a terminal's name from its reply, a pane's title and badge, and a pane's
// view as it is clipped
// (docs/decisions/0014-PLAN-hardening.md Step 2, finding C6).
//
// Line is for one line of plain text. Styled is for text that carries its
// own styles and links, such as a pane's view, and keeps those. Both drop
// the same characters from the text. safetext does the cleaning, under
// rules equal to each function's behaviour; Truncate and Token stay here,
// since they cut and filter by width and alphabet, which is not sanitising
// (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md D9 S5).
//
// Stability: internal.
package sanitize

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// Line is one line of display text: escape sequences stripped, each run of
// \r, \n, \t, U+2028 and U+2029 turned into one space, invalid UTF-8 shown
// as one U+FFFD for each maximal subpart, and C0, DEL, C1, the Bidi_Control
// set, U+200B, U+FEFF, U+2060-2064, U+00AD and the tag characters
// (U+E0000-E007F) dropped. ZWJ, ZWNJ and variation selectors stay. It is
// safetext's Line rules.
func Line(s string) string { return safetext.Clean(s, safetext.Line()) }

// styled is Styled's rules: Line's, with each SGR of at most 32 parameters
// and each OSC 8 hyperlink of printable ASCII and at most 2,048 bytes
// kept as it is.
var styled = safetext.Rules{Flatten: true, KeepSGR: true}

// Styled is Line for text that carries its own styles and links: each SGR
// and OSC 8 hyperlink is kept as it is, and the text between them is
// filtered as Line filters it. Every other sequence, a stray ESC and a
// sequence cut short are dropped
// (docs/decisions/0014-PLAN-hardening.md Step 9, D9), and so is an OSC 8
// over 2,048 bytes
// (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md A5).
func Styled(s string) string { return safetext.Clean(s, styled) }

// Truncate cuts s to at most cells cells measured with m, and when it cuts
// anything, ends it with tail, kept within cells too. The result never
// measures wider than cells under m. Under WcWidth, x/ansi's Truncate cuts
// by grapheme cluster while its StringWidth adds up runes, so a ZWJ emoji
// sequence, Hangul jamo or a conjunct cut short can stay wider than asked;
// Truncate then cuts rune by rune, keeping escape sequences whole
// (docs/decisions/0014-PLAN-hardening.md Step 9, D8).
func Truncate(s string, cells int, m ansi.Method, tail string) string {
	cells = max(cells, 0)
	if StringWidth(m, s) <= cells {
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

// StringWidth is s's width in cells under m. Printable ASCII is one cell a
// byte under every method, so a string of it alone is counted without
// parsing; any other byte gives the string to m.
func StringWidth(m ansi.Method, s string) int {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c < 0x20 || c > 0x7e {
			return m.StringWidth(s)
		}
	}
	return len(s)
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
// sequence, a line break or tab, invalid UTF-8, or a character Line drops. A
// validator refuses such a string rather than cleaning it.
func HasControl(s string) bool { return !safetext.IsClean(s, safetext.Line()) }
