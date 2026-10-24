package sanitize

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

// dropSet is the drop set Line documents, written out apart from dropped,
// so the fuzz target checks Line against the rule and not against its own
// predicate: C0, DEL and C1; the Bidi_Control characters; U+200B, U+FEFF,
// U+2060-2064 and U+00AD; and the tag characters.
func dropSet(r rune) bool {
	switch {
	case r <= 0x1f, r == 0x7f, r >= 0x80 && r <= 0x9f:
		return true
	case r == 0x061c, r == 0x200e, r == 0x200f, r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069:
		return true
	case r == 0x200b, r == 0xfeff, r >= 0x2060 && r <= 0x2064, r == 0x00ad:
		return true
	}
	return r >= 0xe0000 && r <= 0xe007f
}

// FuzzLine: Line is idempotent, and its output is valid UTF-8 with no rune
// from the drop set, which holds ESC, line breaks and tabs, and no U+2028
// or U+2029 (docs/decisions/0014-PLAN-hardening.md Step 9, finding H9).
func FuzzLine(f *testing.F) {
	for _, s := range []string{
		"hello, world", "\x1b[1mbold\x1b[m \x1b]8;;http://x\x07link\x1b]8;;\x07", "a\x1b", "a\x1bbc",
		"a\r\n\t b\nc", "a\u2028b\u2029c", "a\x00\x07\x08\x7fb", "a\u0085\u009bb", "a\xffb\xc3",
		"a\u202eb\u2066c\u061cd\u200ee", "a\u200bb", "\ufeffab", "a\u2060\u2061\u2064b", "a\u00adb",
		"a\U000e0041\U000e007fb", "\U0001f468\u200d\U0001f469\u200d\U0001f467", "a\u200cb", "\u2764\ufe0f",
		"e\u0301", "\u65e5\u672c\u8a9e", "\x1bP>|kitty\x1b\\x", "\x1b]0;t\x1b", "\x9b31m",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out := Line(s)
		if again := Line(out); again != out {
			t.Fatalf("Line(%q) = %q, and Line of that is %q", s, out, again)
		}
		if !utf8.ValidString(out) {
			t.Fatalf("Line(%q) = %q, not valid UTF-8", s, out)
		}
		for _, r := range out {
			if dropSet(r) || r == 0x2028 || r == 0x2029 {
				t.Fatalf("Line(%q) = %q holds %U", s, out, r)
			}
		}
	})
}

// refRanges is the Unicode Standard's Table 3-7, written out plainly: each
// well-formed sequence's byte ranges, by lead byte (safetext's
// TestMaximalSubparts holds the same table).
var refRanges = [][][2]byte{
	{{0x00, 0x7f}},
	{{0xc2, 0xdf}, {0x80, 0xbf}},
	{{0xe0, 0xe0}, {0xa0, 0xbf}, {0x80, 0xbf}},
	{{0xe1, 0xec}, {0x80, 0xbf}, {0x80, 0xbf}},
	{{0xed, 0xed}, {0x80, 0x9f}, {0x80, 0xbf}},
	{{0xee, 0xef}, {0x80, 0xbf}, {0x80, 0xbf}},
	{{0xf0, 0xf0}, {0x90, 0xbf}, {0x80, 0xbf}, {0x80, 0xbf}},
	{{0xf1, 0xf3}, {0x80, 0xbf}, {0x80, 0xbf}, {0x80, 0xbf}},
	{{0xf4, 0xf4}, {0x80, 0x8f}, {0x80, 0xbf}, {0x80, 0xbf}},
}

// refSubpart is the length of what s begins with, and whether it is
// well-formed: the longest prefix of a row of refRanges, or one byte.
func refSubpart(s string) (int, bool) {
	for _, row := range refRanges {
		if s[0] < row[0][0] || s[0] > row[0][1] {
			continue
		}
		n := 1
		for n < len(row) && n < len(s) && s[n] >= row[n][0] && s[n] <= row[n][1] {
			n++
		}
		return n, n == len(row)
	}
	return 1, false
}

// subparts is s with each maximal subpart of invalid UTF-8 replaced by one
// U+FFFD, as the reference finds them: the plan's f.
func subparts(s string) string {
	var b strings.Builder
	for s != "" {
		n, ok := refSubpart(s)
		if ok {
			b.WriteString(s[:n])
		} else {
			b.WriteRune(utf8.RuneError)
		}
		s = s[n:]
	}
	return b.String()
}

// hasC1 reports a C1 code point, U+0080-U+009F encoded as UTF-8, which
// 0009-MADR §2 reads as its 7-bit form (0020-MADR A3.1, class 1).
func hasC1(s string) bool {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == 0xc2 && s[i+1] >= 0x80 && s[i+1] <= 0x9f {
			return true
		}
	}
	return false
}

// introduced reports whether s opens, at some ESC, a sequence whose
// introducer is one of intro and which is followed, anywhere after, by a
// byte that after accepts. The grammar executes a C0 control, and ignores
// DEL, without leaving an escape, so ESC NUL ^ is a PM, and ESC NUL ] an
// OSC, all the same.
func introduced(s, intro string, after func(byte) bool) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			continue
		}
		j := i + 1
		for j < len(s) && (s[j] < 0x20 && s[j] != 0x1b && s[j] != 0x18 && s[j] != 0x1a || s[j] == 0x7f) {
			j++
		}
		if j < len(s) && strings.IndexByte(intro, s[j]) >= 0 && slices.ContainsFunc([]byte(s[j+1:]), after) {
			return true
		}
	}
	return false
}

// stringDeparts reports a DCS, SOS, PM or APC followed, anywhere after it,
// by BEL, which 0009-MADR §2 makes end it (class 2), or by a byte of 0x80
// or more, which §2 keeps inside it (class 3).
func stringDeparts(s string) bool {
	return introduced(s, "PX^_", func(c byte) bool { return c == 0x07 || c >= 0x80 })
}

// oscDeparts reports an OSC followed by a byte of 0x80 or more: x/ansi
// reads such bytes one at a time, so a character's 0x9C byte ends the OSC
// as ST, where safetext reads the character (class 4).
func oscDeparts(s string) bool {
	return introduced(s, "]", func(c byte) bool { return c >= 0x80 })
}

// lineDeparts reports whether s holds what classes 1 to 4 name, where Line
// and v0.12.0's differ by the grammar.
func lineDeparts(s string) bool { return hasC1(s) || stringDeparts(s) || oscDeparts(s) }

// plainest reports whether every ESC in s begins a sequence of the plainest
// form, which v0.12.0's DecodeSequence and the DEC parser split alike: a
// CSI, an OSC of printable ASCII ended by BEL or ESC \, or a two-byte
// escape. Outside it is class 5.
func plainest(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			continue
		}
		n := plainLen(s[i:])
		if n == 0 {
			return false
		}
		i += n - 1
	}
	return true
}

// plainLen is the length of the plainest-form sequence s begins with, or 0.
func plainLen(s string) int {
	if len(s) < 2 {
		return 0
	}
	switch c := s[1]; {
	case c == '[':
		// A private marker first, if any, then digits, ";" and ":", then
		// intermediates, then a final byte, in that order.
		j := 2
		if j < len(s) && s[j] >= 0x3c && s[j] <= 0x3f {
			j++
		}
		for j < len(s) && s[j] >= 0x30 && s[j] <= 0x3b {
			j++
		}
		for j < len(s) && s[j] >= 0x20 && s[j] <= 0x2f {
			j++
		}
		if j < len(s) && s[j] >= 0x40 && s[j] <= 0x7e {
			return j + 1
		}
	case c == ']':
		for j := 2; j < len(s); j++ {
			switch b := s[j]; {
			case b == 0x07:
				return j + 1
			case b == 0x1b && j+1 < len(s) && s[j+1] == '\\':
				return j + 2
			case b < 0x20 || b > 0x7e:
				return 0
			}
		}
	case c >= 0x30 && c <= 0x7e && strings.IndexByte("PX^_", c) < 0:
		return 2
	}
	return 0
}

// FuzzAgainstV0120 holds Line, Styled and HasControl to v0.12.0's, the
// oracle, by four exact properties. For every input x: HasControl equals
// the oracle's; Line of Styled is Line; and, outside the classes of
// 0020-MADR A3.1, where the grammar differs on purpose, Line and Styled
// equal the oracle's on subparts(x), so that for invalid UTF-8 they differ
// by D9 S2 alone. TestDepartures pins each class
// (docs/decisions/0020-PLAN-charm-free-cores-glyphs-and-safetext.md Step 5).
func FuzzAgainstV0120(f *testing.F) {
	for _, s := range []string{
		"hello, world", "\x1b[1mbold\x1b[m \x1b]8;;http://x\x07link\x1b]8;;\x07", "a\x1b", "a\x1bbc",
		"a\r\n\t b\nc", "a\u2028b\u2029c", "a\x00\x07\x08\x7fb", "a\xffb\xc3",
		"a\u202eb\u2066c\u061cd\u200ee", "a\u200bb", "\ufeffab", "a\u2060\u2061\u2064b", "a\u00adb",
		"a\U000e0041\U000e007fb", "\U0001f468\u200d\U0001f469\u200d\U0001f467", "a\u200cb", "\u2764\ufe0f",
		"\x1b[38;2;1;2;3;1mred\x1b[m", "\x1b]8;;http://x\x07link\x1b]8;;\x1b\\", "\x1b[4:3mu\x1b[0m",
		"\x1b]8;id=1;https://x\x1b\\l\x1b]8;;\x1b\\", "a\x1b[2;5Hb\x1b[2J", "a\x1b]0;title\x07b",
		"\x1b[" + strings.Repeat("1;", 31) + "1ma", "\x1b[" + strings.Repeat(":", 32) + "0ma",
		"pkg/file.go:12:\tok\r\n\x1b[32mPASS\x1b[0m line\n",
		// One input for each class of 0020-MADR A3.1, as TestDepartures has
		// them.
		"a\u0090q\u009cb", "a\x1bXs\x07b", "a\x1b_\u00e9\x1b\\b", "a\x1b]0;\u00dc b", "a\x1b[0?mb", "a\u009b1mb",
		// Found by the fuzzer: class 4 behind ESC RS ], and U+05DC is d7 9c.
		"0\x1b\x1e]\u05dc0",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		// v0.12.0 counted invalid UTF-8 as a control too, since its Line
		// dropped it, so HasControl is compared on s itself.
		if got, want := HasControl(s), oracleHasControl(s); got != want {
			t.Errorf("HasControl(%q) = %v; v0.12.0 gives %v", s, got, want)
		}
		if got, want := Line(Styled(s)), Line(s); got != want {
			t.Errorf("Line(Styled(%q)) = %q; Line gives %q", s, got, want)
		}
		if lineDeparts(s) {
			return
		}
		x := subparts(s)
		if got, want := Line(s), oracleLine(x); got != want {
			t.Errorf("Line(%q) = %q; v0.12.0 gives %q", s, got, want)
		}
		if !plainest(s) {
			return
		}
		if got, want := Styled(s), oracleStyled(x); got != want {
			t.Errorf("Styled(%q) = %q; v0.12.0 gives %q", s, got, want)
		}
	})
}
