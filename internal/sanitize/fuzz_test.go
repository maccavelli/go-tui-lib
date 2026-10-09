package sanitize

import (
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
