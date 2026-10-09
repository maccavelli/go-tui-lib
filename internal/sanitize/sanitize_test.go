package sanitize

import (
	"strings"
	"testing"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// The sanitizer (docs/decisions/0014-PLAN-hardening.md Step 2).

func TestLine(t *testing.T) {
	const family = "\U0001f468\u200d\U0001f469\u200d\U0001f467" // ZWJ joins the family
	for _, tc := range []struct{ name, in, want string }{
		{"plain", "hello, world", "hello, world"},
		{"escape sequences", "\x1b[1mbold\x1b[m \x1b]8;;http://x\x07link\x1b]8;;\x07", "bold link"},
		{"a lone ESC", "a\x1b", "a"},
		{"ESC b is a whole sequence", "a\x1bbc", "ac"},
		{"line breaks and tabs, as one space", "a\r\n\t b\nc", "a  b c"},
		{"U+2028 and U+2029", "a\u2028b\u2029c", "a b c"},
		{"C0 and DEL", "a\x00\x07\x08\x7fb", "ab"},
		{"C1, as UTF-8", "a\u0085\u009bb", "ab"},
		{"invalid UTF-8", "a\xffb\xc3", "ab"},
		{"Bidi_Control", "a\u202eb\u2066c\u061cd\u200ee", "abcde"},
		{"U+200B (D2)", "a\u200bb", "ab"},
		{"U+FEFF", "\ufeffab", "ab"},
		{"U+2060-2064", "a\u2060\u2061\u2064b", "ab"},
		{"soft hyphen", "a\u00adb", "ab"},
		{"tag characters", "a\U000e0041\U000e007fb", "ab"},
		{"ZWJ kept", family, family},
		{"ZWNJ kept", "a\u200cb", "a\u200cb"},
		{"variation selector kept", "\u2764\ufe0f", "\u2764\ufe0f"},
		{"combining mark kept", "e\u0301", "e\u0301"},
		{"wide characters kept", "\u65e5\u672c\u8a9e", "\u65e5\u672c\u8a9e"},
	} {
		if got := Line(tc.in); got != tc.want {
			t.Errorf("%s: Line(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestStyled(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"plain", "hello", "hello"},
		{"SGR kept", "\x1b[38;2;1;2;3;1mred\x1b[m", "\x1b[38;2;1;2;3;1mred\x1b[m"},
		{"an OSC 8 link kept", "\x1b]8;;http://x\x07link\x1b]8;;\x1b\\", "\x1b]8;;http://x\x07link\x1b]8;;\x1b\\"},
		{"a stray ESC dropped", "\x1b", ""},
		{"a stray ESC after text", "ab\x1b", "ab"},
		{"ESC b is a whole sequence, dropped (D9)", "a\x1bbc", "ac"},
		{"a CSI cut short", "a\x1b[31", "a"},
		{"controls in the text", "\x1b[1ma\x07\u202eb\x1b[m", "\x1b[1mab\x1b[m"},
		{"tabs and breaks", "a\tb\r", "a b "},
		{"an 8-bit CSI", "a\x9b31mb", "a31mb"},
		// D9 (docs/decisions/0014-PLAN-hardening.md Step 9): only SGR and
		// OSC 8 are kept.
		{"SGR with colons kept", "\x1b[4:3mu\x1b[0m", "\x1b[4:3mu\x1b[0m"},
		// The decoder reads ESC SP X as a string sequence running to the
		// end, and it is dropped whole.
		{"an nF escape dropped", "000\x1b X0000000", "000"},
		{"S7C1T dropped", "a\x1b Fb", "ab"},
		{"DECALN dropped", "a\x1b#8b", "ab"},
		{"S8C1T dropped", "a\x1b Gb", "ab"},
		{"a cursor move dropped", "a\x1b[2;5Hb\x1b[2J", "ab"},
		{"a private SGR-like CSI dropped", "a\x1b[?25mb", "ab"},
		{"a window title dropped", "a\x1b]0;title\x07b", "ab"},
		{"an OSC carrying CR dropped", "a\x1b]\r\x07b", "ab"},
		{"an OSC 8 link carrying CR dropped", "\x1b]8;;http://x\r\x07l\x1b]8;;\x07", "l\x1b]8;;\x07"},
		{"an OSC 8 link with a non-ASCII URI dropped", "\x1b]8;;http://\u00e9\x07l\x1b]8;;\x07", "l\x1b]8;;\x07"},
		{"an OSC 8 with params kept", "\x1b]8;id=1;https://x\x1b\\l\x1b]8;;\x1b\\", "\x1b]8;id=1;https://x\x1b\\l\x1b]8;;\x1b\\"},
		{"a DCS dropped", "a\x1bP>|x\x1b\\b", "ab"},
		{"an OSC 8 without its separator dropped", "a\x1b]8;http://x\x07b", "ab"},
		{"an SGR of 32 parameters kept", "\x1b[" + strings.Repeat("1;", 31) + "1ma", "\x1b[" + strings.Repeat("1;", 31) + "1ma"},
		{"an SGR of 33 parameters dropped", "\x1b[" + strings.Repeat(":", 32) + "0ma", "a"},
	} {
		if got := Styled(tc.in); got != tc.want {
			t.Errorf("%s: Styled(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
	// What Styled keeps of the text is what Line keeps.
	for _, in := range []string{"a\u200bb\u00adc", "\ufeffx\U000e0041y", "\u65e5\u202e\u672c"} {
		if Styled(in) != Line(in) {
			t.Errorf("Styled(%q) = %q, Line = %q", in, Styled(in), Line(in))
		}
	}
}

func TestTokenAndHasControl(t *testing.T) {
	lower := func(r rune) bool { return unicode.IsLower(r) || r == '-' }
	if got := Token("Ab-c\x1b[1m;d", lower); got != "b-cmd" {
		t.Errorf("Token = %q", got)
	}
	for in, want := range map[string]bool{
		"https://example.com/a?b=c":        false,
		"plain text":                       false,
		"tab\there":                        true,
		"line\nbreak":                      true,
		"\x1b[31mred":                      true,
		"rtl\u202eoverride":                true,
		"zero\u200bwidth":                  true,
		"emoji \U0001f468\u200d\U0001f469": false,
	} {
		if HasControl(in) != want {
			t.Errorf("HasControl(%q) = %v, want %v", in, !want, want)
		}
	}
	if got := Truncate("\u65e5\u672c\u8a9e", 4, ansi.WcWidth, ""); got != "\u65e5\u672c" {
		t.Errorf("Truncate = %q", got)
	}
	if got := Truncate("abc", -1, ansi.WcWidth, ""); got != "" {
		t.Errorf("Truncate to -1 = %q", got)
	}
}

// TestTruncate: the result is never wider than asked under either method,
// a tail ends what was cut, escape sequences after the cut are kept, and a
// tail wider than the cells is left out
// (docs/decisions/0014-PLAN-hardening.md Step 9, D8).
func TestTruncate(t *testing.T) {
	family := "\U0001f468\u200d\U0001f469\u200d\U0001f467"
	for _, tc := range []struct {
		name, in string
		cells    int
		m        ansi.Method
		tail     string
		want     string
	}{
		{"fits", "abc", 3, ansi.WcWidth, "~", "abc"},
		{"cut, with a tail", "abcdef", 4, ansi.WcWidth, "~", "abc~"},
		{"a family cut under WcWidth", family, 3, ansi.WcWidth, "", "\U0001f468\u200d"},
		{"a family cut with a tail", family, 5, ansi.WcWidth, "~", "\U0001f468\u200d\U0001f469\u200d~"},
		{"Hangul jamo cut", "\u1100\u1161\u11a8", 2, ansi.WcWidth, "", "\u1100"},
		{"a conjunct cut", "\u0915\u094d\u0937", 1, ansi.WcWidth, "", "\u0915\u094d"},
		{"a Prepend cut", "0\u0605000", 3, ansi.WcWidth, "", "0\u060500"},
		{"a style's reset kept", "\x1b[1mabcdef\x1b[m", 3, ansi.WcWidth, "~", "\x1b[1mab~\x1b[m"},
		{"a tail wider than the cells", "abc", 1, ansi.WcWidth, "~~", "a"},
		{"nothing", "abc", 0, ansi.WcWidth, "~", ""},
		{"a family under GraphemeWidth", family, 1, ansi.GraphemeWidth, "", ""},
	} {
		got := Truncate(tc.in, tc.cells, tc.m, tc.tail)
		if got != tc.want || tc.m.StringWidth(got) > tc.cells {
			t.Errorf("%s: Truncate(%q, %d, %v, %q) = %q, %d cells; want %q", tc.name, tc.in, tc.cells, tc.m, tc.tail, got, tc.m.StringWidth(got), tc.want)
		}
	}
}
