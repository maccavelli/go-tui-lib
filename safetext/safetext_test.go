package safetext_test

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
)

// clean runs Clean and, as a check on every case, the Sanitizer split at
// each byte, and IsClean, which must agree with it.
func clean(t *testing.T, in string, r safetext.Rules) string {
	t.Helper()
	got := safetext.Clean(in, r)
	if safetext.IsClean(in, r) != (got == in) {
		t.Errorf("IsClean(%q) = %v, but Clean gives %q", in, safetext.IsClean(in, r), got)
	}
	for i := 0; i <= len(in); i++ {
		var b bytes.Buffer
		s := safetext.NewSanitizer(&b, r)
		_, _ = s.Write([]byte(in[:i]))
		_, _ = s.Write([]byte(in[i:]))
		_ = s.Close()
		if b.String() != got {
			t.Errorf("Sanitizer split at %d of %q gives %q; Clean gives %q", i, in, b.String(), got)
			break
		}
	}
	return got
}

// TestSequences: a row per sequence class of 0009-PLAN Step 2, each
// removed whole under Text.
func TestSequences(t *testing.T) {
	for name, c := range map[string]struct{ in, want string }{
		"CSI":                    {"a\x1b[31mb", "ab"},
		"CSI with a prefix":      {"a\x1b[?2026hb", "ab"},
		"OSC ended by BEL":       {"a\x1b]0;title\x07b", "ab"},
		"OSC ended by ST":        {"a\x1b]0;title\x1b\\b", "ab"},
		"OSC ended by U+009C":    {"a\x1b]0;title\u009cb", "ab"},
		"DCS":                    {"a\x1bP1$r0m\x1b\\b", "ab"},
		"SOS":                    {"a\x1bXsos\x1b\\b", "ab"},
		"PM":                     {"a\x1b^pm\x1b\\b", "ab"},
		"APC":                    {"a\x1b_Gi=1;OK\x1b\\b", "ab"},
		"APC holding UTF-8":      {"a\x1b_é and more\x1b\\b", "ab"},
		"APC ended by BEL":       {"a\x1b_apc\x07b", "ab"},
		"nF escape":              {"a\x1b(Bb", "ab"},
		"Fp escape":              {"a\x1b7b", "ab"},
		"Fs escape":              {"a\x1bcb", "ab"},
		"a lone ESC at the end":  {"a\x1b", "a"},
		"ESC then DEL":           {"a\x1b\x7fbc", "ac"},
		"CAN aborts":             {"a\x1b[31\x18b", "ab"},
		"SUB aborts":             {"a\x1b]0;x\x1ab", "ab"},
		"ESC restarts":           {"a\x1b[3\x1b[0mb", "ab"},
		"U+009B is CSI":          {"a\u009b31mb", "ab"},
		"U+009D is OSC":          {"a\u009d0;t\u0007b", "ab"},
		"U+0090 is DCS":          {"a\u0090xyz\u009cb", "ab"},
		"U+0085 is ESC E":        {"a\u0085b", "ab"},
		"a CSI left for UTF-8":   {"a\x1b[1éb", "aéb"},
		"an unterminated OSC":    {"a\x1b]0;never ended", "a"},
		"an unterminated CSI":    {"a\x1b[31", "a"},
		"ST after an ended OSC":  {"\x1b]0;x\x1b\\\x1b\\b", "b"},
		"a C0 executed in a CSI": {"a\x1b[1\n2mb", "a\nb"},
		"codex's injection string (0009-PLAN Step 2)": {"\x1b[2J\x1b]52;c;Y2xpcA==\x07\u009dhidden\u009c\r\n\t", "\n\t"},
	} {
		if got := clean(t, c.in, safetext.Text()); got != c.want {
			t.Errorf("%s: Clean(%q) = %q, want %q", name, c.in, got, c.want)
		}
	}
}

// TestControls: each C0 control but tab and line feed, DEL, \r\n and a
// lone \r under Text.
func TestControls(t *testing.T) {
	for c := range rune(0x20) {
		if c == '\t' || c == '\n' || c == '\r' || c == 0x1b {
			continue
		}
		in := "a" + string(c) + "b"
		if got := clean(t, in, safetext.Text()); got != "ab" {
			t.Errorf("Clean(%q) = %q, want ab", in, got)
		}
	}
	for in, want := range map[string]string{
		"a\x7fb":   "ab",
		"a\tb\nc":  "a\tb\nc",
		"a\r\nb":   "a\nb",
		"a\rb":     "ab",
		"a\r":      "a",
		"a\r\r\nb": "a\nb",
		"a\u2028b": "a\nb",
		"a\xffb":   "a\ufffdb",
	} {
		if got := clean(t, in, safetext.Text()); got != want {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
}

// removedRunes is every code point internal/sanitize removes
// (internal/sanitize/sanitize.go, dropped), less the C0 and C1 controls,
// which TestControls and TestSequences cover: the Bidi_Control set, then
// the format characters.
var removedRunes = []rune{
	0x061c, 0x200e, 0x200f, 0x202a, 0x202b, 0x202c, 0x202d, 0x202e, 0x2066, 0x2067, 0x2068, 0x2069,
	0x200b, 0xfeff, 0x2060, 0x2061, 0x2062, 0x2063, 0x2064, 0x00ad, 0xe0000, 0xe0041, 0xe007f,
}

// TestRemovedSet (0020-MADR D9 S1): every code point internal/sanitize
// removes is removed under Line and Strict, and under Text but for the
// twelve Bidi_Control code points, which Text keeps. ZWJ, ZWNJ and a
// variation selector stay.
func TestRemovedSet(t *testing.T) {
	for i, r := range removedRunes {
		isBidi := i < 12
		in := "a" + string(r) + "b"
		for name, rules := range map[string]safetext.Rules{"Line": safetext.Line(), "Strict": safetext.Strict(), "Text": safetext.Text()} {
			want := "ab"
			if name == "Text" && isBidi {
				want = in
			}
			if got := clean(t, in, rules); got != want {
				t.Errorf("%s: Clean(%q) = %q, want %q", name, in, got, want)
			}
		}
	}
	for _, r := range []rune{0x200d, 0x200c, 0xfe0f} {
		in := "a" + string(r) + "b"
		if got := clean(t, in, safetext.Line()); got != in {
			t.Errorf("Clean(%q) = %q, want it kept", in, got)
		}
	}
}

// TestFlatten (0020-MADR D9 S3): each run of breaks is one space.
func TestFlatten(t *testing.T) {
	for in, want := range map[string]string{
		"a\r\n\t\u2028b":   "a b",
		"a\n\x1b[31m\nb":   "a b",
		"a\n\x00\nb":       "a b",
		"a b\tc":           "a b c",
		"\n\nlead":         " lead",
		"trail\n\n":        "trail ",
		"a\u2029\u2028\tb": "a b",
	} {
		if got := clean(t, in, safetext.Line()); got != want {
			t.Errorf("Line: Clean(%q) = %q, want %q", in, got, want)
		}
	}
	if got := clean(t, "a\n\nb", safetext.Text()); got != "a\n\nb" {
		t.Errorf("Text: Clean(%q) = %q, want it kept", "a\n\nb", got)
	}
}

// TestEscape (0020-MADR D9 S4): what would be removed is shown escaped;
// what Tabs and Newlines keep stays raw.
func TestEscape(t *testing.T) {
	output := safetext.Rules{Tabs: true, Newlines: true, Escape: true}
	for _, c := range []struct {
		rules    safetext.Rules
		in, want string
	}{
		{output, "a\tb\nc\x1b[2Jd\u202ee", "a\tb\nc\\x1b[2Jd\\u202ee"},
		{output, "a\r\nb\rc", "a\nb\\rc"},
		{output, "a\u0085\u009bb", "a\\u0085\\u009bb"},
		{output, "tag\U000e0041", "tag\\U000e0041"},
		{output, "bad\xff", "bad\ufffd"},
		{safetext.Command(), "rm -rf /\nwhoami", "rm -rf /\\nwhoami"},
		{safetext.Command(), "a\tb\a\x7f", "a\\tb\\a\\x7f"},
		{safetext.Rules{Escape: true, Flatten: true}, "a\n\tb\x01", "a b\\x01"},
	} {
		if got := clean(t, c.in, c.rules); got != c.want {
			t.Errorf("Clean(%q, %+v) = %q, want %q", c.in, c.rules, got, c.want)
		}
	}
	// Each escaped rune is as strconv.QuoteToASCII writes it.
	for _, r := range append([]rune{0x00, 0x07, 0x1b, 0x7f, 0x85, 0x9f}, removedRunes...) {
		want := strconv.QuoteToASCII(string(r))
		want = want[1 : len(want)-1]
		if got := clean(t, string(r), safetext.Command()); got != want {
			t.Errorf("Command: Clean(%U) = %q, want %q", r, got, want)
		}
	}
}

// TestKeepSGR: an SGR of 32 parameters is kept and one of 33 is not; an
// OSC 8 link is kept, and StatusLine keeps only web and mail links with a
// host or an address, and every link's end.
func TestKeepSGR(t *testing.T) {
	keep := safetext.Rules{KeepSGR: true}
	sgr := func(n int) string { return "\x1b[" + strings.TrimSuffix(strings.Repeat("1;", n), ";") + "m" }
	for _, c := range []struct {
		name     string
		rules    safetext.Rules
		in, want string
	}{
		{"32 parameters", keep, "a" + sgr(32) + "b", "a" + sgr(32) + "b"},
		{"33 parameters", keep, "a" + sgr(33) + "b", "ab"},
		{"no parameters", keep, "a\x1b[mb", "a\x1b[mb"},
		{"colon sub-parameters", keep, "\x1b[38:2::255:0:0mx", "\x1b[38:2::255:0:0mx"},
		{"a prefix is not an SGR", keep, "\x1b[?1mx", "x"},
		{"an intermediate is not an SGR", keep, "\x1b[1 mx", "x"},
		{"a control inside is not an SGR", keep, "\x1b[3\x011mx", "x"},
		{"other CSI removed", keep, "\x1b[2Jx", "x"},
		// 0020-MADR A3.3: nothing a C1 code point introduces or ends is kept.
		{"an SGR introduced by U+009B removed", keep, "\u009b31mx", "x"},
		{"a link introduced by U+009D removed", keep, "\u009d8;;https://e.org\x07x", "x"},
		{"a link ended by U+009C removed", keep, "\x1b]8;;https://e.org\u009cx", "x"},
		{"a link ended by ESC \\ after a C1 SGR", keep, "\u009b1m\x1b]8;;https://e.org\x1b\\x", "\x1b]8;;https://e.org\x1b\\x"},
		{"any link", keep, "\x1b]8;;file:///etc\x07x\x1b]8;;\x07", "\x1b]8;;file:///etc\x07x\x1b]8;;\x07"},
		{"a link ended by ST", keep, "\x1b]8;id=1;https://e.org\x1b\\x\x1b]8;;\x1b\\", "\x1b]8;id=1;https://e.org\x1b\\x\x1b]8;;\x1b\\"},
		{"a link not printable ASCII", keep, "\x1b]8;;https://é.org\x07x", "x"},
		{"other OSC removed", keep, "\x1b]0;title\x07x", "x"},
		{"StatusLine: https", safetext.StatusLine(), "\x1b]8;;https://example.org\x1b\\x\x1b]8;;\x1b\\", "\x1b]8;;https://example.org\x1b\\x\x1b]8;;\x1b\\"},
		{"StatusLine: mailto", safetext.StatusLine(), "\x1b]8;;mailto:a@b.org\x07x\x1b]8;;\x07", "\x1b]8;;mailto:a@b.org\x07x\x1b]8;;\x07"},
		{"StatusLine: file", safetext.StatusLine(), "\x1b]8;;file:///etc\x07x", "x"},
		{"StatusLine: no host", safetext.StatusLine(), "\x1b]8;;https://\x07x", "x"},
	} {
		if got := clean(t, c.in, c.rules); got != c.want {
			t.Errorf("%s: Clean(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// TestVisible: one Control mark per removed sequence, the ASCII one under
// glyph.ASCII(), and none for what is kept or for a single control.
func TestVisible(t *testing.T) {
	ascii := safetext.Rules{Visible: true, Glyphs: glyph.ASCII()}
	for in, want := range map[string]string{
		"a\x1b[31mb\x1b]0;x\x07c": "a^b^c",
		"a\x1b]0;x\x1b\\b":        "a^b",
		"a\x1bXs\x1b\\b":          "a^b",
		"a\x01b":                  "ab",
		"a\x1b[3\x1b[0mb":         "a^^b",
		"a\x1b":                   "a^",
		"a\x1b]0;x":               "a^",
	} {
		if got := clean(t, in, ascii); got != want {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
	if got := clean(t, "a\x1b[1mb", safetext.Rules{Visible: true}); got != "a␛b" {
		t.Errorf("the zero Glyphs: Clean = %q, want the Unicode mark", got)
	}
	keep := safetext.Rules{Visible: true, KeepSGR: true, Glyphs: glyph.ASCII()}
	if got := clean(t, "a\x1b[1mb\x1b[2Jc", keep); got != "a\x1b[1mb^c" {
		t.Errorf("KeepSGR: Clean = %q, want the SGR kept and the other marked", got)
	}
}

// TestMaxString: a string with no terminator hides at most MaxString bytes,
// and the text after it survives.
func TestMaxString(t *testing.T) {
	r := safetext.Rules{MaxString: 8}
	// The OSC's content is "0;0123456789b": its first 8 bytes, "0;012345",
	// are hidden, and from the 9th on it is text.
	if got := clean(t, "a\x1b]0;0123456789b", r); got != "a6789b" {
		t.Errorf("Clean = %q, want a6789b: 8 bytes hidden, the rest text", got)
	}
	if got := clean(t, "a\x1b]0;012345\x07b", r); got != "ab" {
		t.Errorf("Clean = %q, want ab: a string within MaxString is removed", got)
	}
	// A terminator past MaxString comes too late: the string was abandoned
	// at its 8th byte, and the BEL is then a lone control.
	if got := clean(t, "a\x1b]0;0123456789\x07b", r); got != "a6789b" {
		t.Errorf("Clean = %q, want a6789b: the string abandoned before its BEL", got)
	}
}

// TestCleanReturnsItsInput: clean text is s itself.
func TestCleanReturnsItsInput(t *testing.T) {
	s := strings.Repeat("plain text, 日本語 and 😀. ", 20)
	got := safetext.Clean(s, safetext.Text())
	if got != s || unsafe.StringData(got) != unsafe.StringData(s) {
		t.Fatalf("Clean of clean text did not return its input itself")
	}
	if !safetext.IsClean(s, safetext.Line()) {
		t.Fatal("IsClean: false for clean text")
	}
}

// TestIsClean: IsClean is false for text Clean would change, a removal at
// the very end included, where what is left is a prefix of the input.
func TestIsClean(t *testing.T) {
	for in, want := range map[string]bool{
		"plain":           true,
		"a\u200b":         false,
		"abc\x1b[m":       false,
		"a\r":             false,
		"a\xe2":           false,
		"\x1b[1mbold":     false,
		"tab\tand\nlines": true,
	} {
		if got := safetext.IsClean(in, safetext.Text()); got != want {
			t.Errorf("IsClean(%q, Text) = %v, want %v", in, got, want)
		}
	}
	if !safetext.IsClean("\x1b[1mbold\x1b[0m", safetext.Rules{KeepSGR: true}) {
		t.Error("IsClean: false for an SGR KeepSGR keeps")
	}
}

func TestSanitizerClose(t *testing.T) {
	var b bytes.Buffer
	s := safetext.NewSanitizer(&b, safetext.Text())
	_, _ = s.Write([]byte("a\x1b]0;never"))
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if b.String() != "a" {
		t.Errorf("after Close: %q, want a", b.String())
	}
	if _, err := s.Write([]byte("x")); !errors.Is(err, safetext.ErrClosed) {
		t.Errorf("Write after Close: %v, want ErrClosed", err)
	}
}
