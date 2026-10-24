package safetext_test

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// fuzzRules is each preset, and the tool-output combination of 0020-MADR
// D9 S4.
var fuzzRules = []safetext.Rules{
	safetext.Text(), safetext.Strict(), safetext.Line(), safetext.Command(), safetext.StatusLine(),
	{Tabs: true, Newlines: true, Escape: true},
}

// FuzzClean holds Clean's properties over arbitrary bytes (0009-PLAN Step
// 2): the output is valid UTF-8; without Escape and KeepSGR it holds no
// ESC, no C1 and no C0 but what the rules keep; Clean is idempotent; a
// Sanitizer split anywhere gives the same; and IsClean agrees with Clean.
func FuzzClean(f *testing.F) {
	for _, s := range []string{
		"", "plain", "a\x1b[31mb", "\x1b]8;;https://e.org\x07x\x1b]8;;\x07", "\x1b]0;t\x1b\\",
		"\x1bP1$r\x1b\\", "\u009b2J", "a\r\nb\rc", "\xe2\x82", "\xc0\xaf", "\x1b[3\x1b[0m",
		"\u202e\u200b\U000e0041", "\x1b_é\x1b\\", "\x1b]0;" + strings.Repeat("x", 100),
	} {
		for p := range fuzzRules {
			f.Add([]byte(s), uint8(p), uint16(len(s)/2))
		}
	}
	f.Fuzz(func(t *testing.T, data []byte, preset uint8, split uint16) {
		r := fuzzRules[int(preset)%len(fuzzRules)]
		x := string(data)
		out := safetext.Clean(x, r)
		if !utf8.ValidString(out) {
			t.Fatalf("Clean(%q) = %q, not valid UTF-8", x, out)
		}
		if !r.Escape && !r.KeepSGR {
			for _, c := range out {
				kept := c == '\t' && r.Tabs && !r.Flatten || c == '\n' && r.Newlines && !r.Flatten
				switch {
				case c == 0x1b, c >= 0x80 && c <= 0x9f:
					t.Fatalf("Clean(%q) = %q holds %U", x, out, c)
				case c < 0x20 && !kept:
					t.Fatalf("Clean(%q) = %q holds %U", x, out, c)
				}
			}
		}
		if again := safetext.Clean(out, r); again != out {
			t.Fatalf("Clean is not idempotent on %q: %q, then %q", x, out, again)
		}
		if clean := safetext.IsClean(x, r); clean != (out == x) {
			t.Fatalf("IsClean(%q) = %v, but Clean gives %q", x, clean, out)
		}
		i := int(split) % (len(data) + 1)
		var b bytes.Buffer
		s := safetext.NewSanitizer(&b, r)
		_, _ = s.Write(data[:i])
		_, _ = s.Write(data[i:])
		_ = s.Close()
		if b.String() != out {
			t.Fatalf("Sanitizer split at %d of %q gives %q; Clean gives %q", i, x, b.String(), out)
		}
		// A byte at a time, no sequence is ever whole in one Write, so the
		// byte-at-a-time parser decides everything Clean's whole-sequence
		// path decides: the two must agree.
		b.Reset()
		s = safetext.NewSanitizer(&b, r)
		for k := range data {
			_, _ = s.Write(data[k : k+1])
		}
		_ = s.Close()
		if b.String() != out {
			t.Fatalf("Sanitizer a byte at a time on %q gives %q; Clean gives %q", x, b.String(), out)
		}
	})
}
