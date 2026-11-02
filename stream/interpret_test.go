package stream_test

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/safetext"
	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// plains is each row's Plain.
func plains(rows []stream.InterpretRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Plain
	}
	return out
}

// TestInterpret: each thing Interpret does to captured output, a row each
// (docs/decisions/0009-MADR-streaming-content-engine.md A1;
// 0009-PLAN-streaming-content-engine.md Step A1.1).
func TestInterpret(t *testing.T) {
	for _, c := range []struct {
		name string
		in   string
		o    stream.InterpretOptions
		want []string // each row's Plain
	}{
		{"text and LF", "hello\nworld", stream.InterpretOptions{}, []string{"hello", "world"}},
		{"a trailing LF adds no row", "a\nb\n", stream.InterpretOptions{}, []string{"a", "b"}},
		{"CR writes over the row", "abcdef\rXY", stream.InterpretOptions{}, []string{"XYcdef"}},
		{"a progress bar ends as its last state", "  0%\r 50%\r100%\n", stream.InterpretOptions{}, []string{"100%"}},
		{"BS moves back a cell", "ab\bX", stream.InterpretOptions{}, []string{"aX"}},
		{"BS stops at column 0", "\b\bX", stream.InterpretOptions{}, []string{"X"}},
		{"HT to the next multiple of 8", "a\tb\tc", stream.InterpretOptions{}, []string{"a       b       c"}},
		{"CSI K erases to the row's end", "abcdef\r\x1b[Kxy", stream.InterpretOptions{}, []string{"xy"}},
		{"CSI 1 K erases to the cursor", "abcdef\x1b[3D\x1b[1K", stream.InterpretOptions{}, []string{"    ef"}},
		{"CSI 2 K erases the row", "abc\x1b[2Kd", stream.InterpretOptions{}, []string{"   d"}},
		{"CSI 2 J empties every row", "a\nb\x1b[2Jc", stream.InterpretOptions{}, []string{"", " c"}},
		{"CSI J erases below", "a\nb\nc\x1b[2A\x1b[J", stream.InterpretOptions{}, []string{"a"}},
		{"CSI A moves up", "a\nb\x1b[Ax", stream.InterpretOptions{}, []string{"ax", "b"}},
		{"CSI B moves down", "a\x1b[2Bx", stream.InterpretOptions{}, []string{"a", "", " x"}},
		{"CSI C and D move along the row", "abc\x1b[2D\x1b[CX", stream.InterpretOptions{}, []string{"abX"}},
		{"CSI G moves to a column", "abc\x1b[2GX", stream.InterpretOptions{}, []string{"aXc"}},
		{"an OSC is dropped, BEL-ended", "a\x1b]0;title\x07b", stream.InterpretOptions{}, []string{"ab"}},
		{"an OSC 8 link keeps its text", "\x1b]8;;https://x.test\x1b\\link\x1b]8;;\x1b\\", stream.InterpretOptions{}, []string{"link"}},
		{"a DCS is dropped", "a\x1bPq#0;2;0;0;0\x1b\\b", stream.InterpretOptions{}, []string{"ab"}},
		{"a private CSI is dropped", "a\x1b[?25lb", stream.InterpretOptions{}, []string{"ab"}},
		{"another CSI is dropped", "\x1b[2;5Hx", stream.InterpretOptions{}, []string{"x"}},
		{"CAN aborts a sequence", "a\x1b[31\x18b", stream.InterpretOptions{}, []string{"ab"}},
		{"other controls are dropped", "a\x07\x00\x7fb", stream.InterpretOptions{}, []string{"ab"}},
		{"a C1 OSC is dropped", "a\u009d0;t\u009cb", stream.InterpretOptions{}, []string{"ab"}},
		{"invalid UTF-8, one U+FFFD a maximal subpart", "a\xe2\x82b\xffc", stream.InterpretOptions{}, []string{"a\ufffdb\ufffdc"}},
		{"format characters are removed", "a\u200bb\u202ec", stream.InterpretOptions{}, []string{"abc"}},
		{"a wide rune takes two cells", "世x", stream.InterpretOptions{}, []string{"世x"}},
		{"writing over a wide rune's half blanks the other", "世x\r\x1b[Cy", stream.InterpretOptions{}, []string{" yx"}},
		{"a combining mark joins the cell before", "e\u0301x", stream.InterpretOptions{}, []string{"e\u0301x"}},
		{"a cluster of 3 cells under WcWidth (D3)", "ౙ\u0c4dౙ\u0c4dౙ00", stream.InterpretOptions{MaxCols: 6}, []string{"ౙ\u0c4dౙ\u0c4dౙ00"}},
		{"writing in a 3-cell cluster blanks it whole", "ౙ\u0c4dౙ\u0c4dౙx\r\x1b[Cy", stream.InterpretOptions{}, []string{" y x"}},
		{"a ZWJ family takes 6 cells under WcWidth", "\U0001F468\u200d\U0001F469\u200d\U0001F467x", stream.InterpretOptions{MaxCols: 6}, []string{"\U0001F468\u200d\U0001F469\u200d\U0001F467"}},
		{"and 2 under GraphemeWidth", "\U0001F468\u200d\U0001F469\u200d\U0001F467x", stream.InterpretOptions{MaxCols: 6, Method: ansi.GraphemeWidth}, []string{"\U0001F468\u200d\U0001F469\u200d\U0001F467x"}},
		{"CSI K inside a cluster erases it whole", "ౙ\u0c4dౙ\u0c4dౙx\x1b[3D\x1b[K", stream.InterpretOptions{}, []string{}},
		{"MaxCols cuts the row", "abcdef\nxy", stream.InterpretOptions{MaxCols: 4}, []string{"abcd", "xy"}},
		{"a wide rune that does not fit is cut", "abc世", stream.InterpretOptions{MaxCols: 4}, []string{"abc"}},
		{"MaxRows keeps the newest", "1\n2\n3\n4", stream.InterpretOptions{MaxRows: 2}, []string{"3", "4"}},
		{"MaxRows, with a trailing LF", "1\n2\n3\n", stream.InterpretOptions{MaxRows: 2}, []string{"2", "3"}},
		{"nothing", "", stream.InterpretOptions{}, []string{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := plains(stream.Interpret([]byte(c.in), c.o))
			if !slices.Equal(got, c.want) {
				t.Errorf("Interpret(%q) = %q; want %q", c.in, got, c.want)
			}
		})
	}
}

// TestInterpretSGR: SGR is kept, in its ; and : forms, and written again
// once per change of style, reset at the row's end.
func TestInterpretSGR(t *testing.T) {
	for _, c := range []struct {
		in, want string
	}{
		{"\x1b[1mbold\x1b[m x", "\x1b[0;1mbold\x1b[m x"},
		{"\x1b[31mred", "\x1b[0;31mred\x1b[m"},
		{"\x1b[91;42mx", "\x1b[0;91;42mx\x1b[m"},
		{"\x1b[38;5;208mo", "\x1b[0;38;5;208mo\x1b[m"},
		{"\x1b[38;2;255;0;10mr", "\x1b[0;38;2;255;0;10mr\x1b[m"},
		{"\x1b[38:2::255:0:10mr", "\x1b[0;38;2;255;0;10mr\x1b[m"},
		{"\x1b[38:2:255:0:10mr", "\x1b[0;38;2;255;0;10mr\x1b[m"},
		{"\x1b[48:5:17mb", "\x1b[0;48;5;17mb\x1b[m"},
		{"\x1b[4:3;58:5:1mu", "\x1b[0;4:3;58;5;1mu\x1b[m"},
		{"\x1b[1;3mab\x1b[22mc", "\x1b[0;1;3mab\x1b[0;3mc\x1b[m"},
		{"\x1b[1mx\x1b[0my", "\x1b[0;1mx\x1b[my"},
		{"\x1b[;1mx", "\x1b[0;1mx\x1b[m"},
		{"\x1b[1m\r\x1b[Kab", "\x1b[0;1mab\x1b[m"},
		{"\u009b1mx", "\x1b[0;1mx\x1b[m"},
		{"\x1b[" + strings.Repeat("1;", 32) + "31mx", "x"}, // 33 parameters: dropped
		{"\x1b[38;5;999mx", "x"},
	} {
		rows := stream.Interpret([]byte(c.in), stream.InterpretOptions{})
		if len(rows) != 1 || rows[0].Styled != c.want {
			t.Errorf("Interpret(%q) = %q; want one row %q", c.in, rows, c.want)
		}
	}
}

// TestInterpretString: an OSC with no terminator is abandoned past
// safetext.DefaultMaxString bytes, and the text after it is kept.
func TestInterpretString(t *testing.T) {
	in := "\x1b]0;" + strings.Repeat("x", safetext.DefaultMaxString+100) + "end"
	rows := stream.Interpret([]byte(in), stream.InterpretOptions{})
	if len(rows) != 1 || !strings.HasSuffix(rows[0].Plain, "end") || len(rows[0].Plain) >= 200 {
		t.Fatalf("an unended OSC gave %d rows, the first %d bytes; want the text after its bound", len(rows), len(rows[0].Plain))
	}
}

// TestInterpretDefaults: the zero options take DefaultInterpretRows and
// DefaultInterpretCols, and ansi.WcWidth.
func TestInterpretDefaults(t *testing.T) {
	if stream.DefaultInterpretRows != 50_000 || stream.DefaultInterpretCols != 8_192 {
		t.Fatalf("defaults %d rows, %d columns; want 50,000 and 8,192", stream.DefaultInterpretRows, stream.DefaultInterpretCols)
	}
	wide := strings.Repeat("x", stream.DefaultInterpretCols+10)
	if rows := stream.Interpret([]byte(wide), stream.InterpretOptions{}); len(rows[0].Plain) != stream.DefaultInterpretCols {
		t.Errorf("a row of %d cells kept %d; want %d", len(wide), len(rows[0].Plain), stream.DefaultInterpretCols)
	}
	many := strings.Repeat("r\n", stream.DefaultInterpretRows+10) + "last"
	rows := stream.Interpret([]byte(many), stream.InterpretOptions{})
	if len(rows) != stream.DefaultInterpretRows || rows[len(rows)-1].Plain != "last" {
		t.Errorf("%d rows, the last %q; want %d, the last \"last\"", len(rows), rows[len(rows)-1].Plain, stream.DefaultInterpretRows)
	}
	// U+2615 is wide under ansi.WcWidth, so a 3-cell row holds "x" and it.
	if rows := stream.Interpret([]byte("x☕y"), stream.InterpretOptions{MaxCols: 3}); rows[0].Plain != "x☕" {
		t.Errorf("under the zero Method, %q; want x and a wide U+2615", rows[0].Plain)
	}
}

// TestInterpretDeterministic: two calls over the same bytes are equal.
func TestInterpretDeterministic(t *testing.T) {
	in := []byte("\x1b[1mbuild\x1b[m 10%\r 50%\r100%\n\x1b[32mok\x1b[m\tdone\n")
	a := stream.Interpret(in, stream.InterpretOptions{})
	b := stream.Interpret(in, stream.InterpretOptions{})
	if !slices.Equal(a, b) {
		t.Fatalf("two calls differ: %q, %q", a, b)
	}
}

// onlySGR reports whether s holds no control but SGR sequences.
func onlySGR(s string) bool {
	for i := 0; i < len(s); i++ {
		switch b := s[i]; {
		case b == 0x1b:
			j := i + 1
			if j >= len(s) || s[j] != '[' {
				return false
			}
			for j++; j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == ';' || s[j] == ':'); j++ {
			}
			if j >= len(s) || s[j] != 'm' {
				return false
			}
			i = j
		case b < 0x20 || b == 0x7f:
			return false
		}
	}
	return true
}

// FuzzInterpret: for any input and bounds, no row is wider than MaxCols
// cells under the method, a row's Styled holds no sequence but SGR, its
// Plain is its Styled stripped and is clean under safetext.Line(), there
// are no more rows than MaxRows, and the result is a function of its input.
func FuzzInterpret(f *testing.F) {
	f.Add([]byte("\x1b[1mbold\x1b[m 10%\r100%\n\x1b[38:2::1:2:3mx\ty"), uint8(20), uint8(5), false)
	f.Add([]byte("世界\x1b[3D\x1b[1K\u009b31m\xe2\x82e\u0301\x1b]8;;x\x07l"), uint8(4), uint8(2), true)
	f.Add([]byte("\x1b[2Ja\x1b[Bb\x1b[2Ac\x1b[J\x1b[Gd\x1b[999Ce"), uint8(7), uint8(3), false)
	f.Fuzz(func(t *testing.T, raw []byte, cols, rows uint8, grapheme bool) {
		o := stream.InterpretOptions{MaxCols: int(cols%100) + 1, MaxRows: int(rows%30) + 1}
		if grapheme {
			o.Method = ansi.GraphemeWidth
		}
		got := stream.Interpret(raw, o)
		if len(got) > o.MaxRows {
			t.Fatalf("%d rows; MaxRows %d", len(got), o.MaxRows)
		}
		for i, r := range got {
			if w := o.Method.StringWidth(r.Styled); w > o.MaxCols {
				t.Fatalf("row %d is %d cells, over %d: %q", i, w, o.MaxCols, r.Styled)
			}
			if !onlySGR(r.Styled) {
				t.Fatalf("row %d holds a sequence but SGR: %q", i, r.Styled)
			}
			if p := ansi.Strip(r.Styled); p != r.Plain {
				t.Fatalf("row %d: Plain %q is not Styled stripped, %q", i, r.Plain, p)
			}
			if !safetext.IsClean(r.Plain, safetext.Line()) {
				t.Fatalf("row %d: Plain %q is not clean under Line", i, r.Plain)
			}
		}
		if again := stream.Interpret(raw, o); !slices.Equal(got, again) {
			t.Fatalf("two calls differ")
		}
	})
}

// toolOutput is about n bytes of a test run's output: coloured results,
// and a progress bar rewritten by CR.
func toolOutput(n int) []byte {
	var b bytes.Buffer
	for i := 0; b.Len() < n; i++ {
		fmt.Fprintf(&b, "\x1b[1m=== RUN\x1b[m   TestCase%d\n", i)
		for p := 0; p <= 100; p += 25 {
			fmt.Fprintf(&b, "\r\x1b[36m[%-20s]\x1b[m %3d%%", strings.Repeat("#", p/5), p)
		}
		fmt.Fprintf(&b, "\n\x1b[32m--- PASS\x1b[m: TestCase%d (0.00s)\n", i)
	}
	return b.Bytes()
}

// BenchmarkInterpret: 1 MB of a test run's output.
func BenchmarkInterpret(b *testing.B) {
	raw := toolOutput(1 << 20)
	b.SetBytes(int64(len(raw)))
	b.ReportAllocs()
	for b.Loop() {
		stream.Interpret(raw, stream.InterpretOptions{})
	}
}
