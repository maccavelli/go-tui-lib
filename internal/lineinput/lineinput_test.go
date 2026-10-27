package lineinput

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// typeBytes feeds b to l through a decoder, as a host does, a paste
// inserted.
func typeBytes(t *testing.T, l *Line, b string) {
	t.Helper()
	var d termkey.Decoder
	for _, in := range d.Flush(d.Decode([]byte(b), nil)) {
		switch in.Kind {
		case termkey.InputKey:
			l.Key(in.Key)
		case termkey.InputPaste:
			l.Insert(in.Text)
		}
	}
}

// at is the line's text with the cursor shown as |.
func at(l *Line) string {
	return strings.Join(l.clusters[:l.pos], "") + "|" + strings.Join(l.clusters[l.pos:], "")
}

// TestKeys: each key of the set, typed as a terminal sends it.
func TestKeys(t *testing.T) {
	for _, c := range []struct{ name, in, want string }{
		{"text", "abc", "abc|"},
		{"backspace", "abc\x7f", "ab|"},
		{"ctrl+h", "abc\x08", "ab|"},
		{"left and insert", "ac\x1b[Db", "ab|c"},
		{"ctrl+b and ctrl+f", "abc\x02\x02\x06", "ab|c"},
		{"home", "abc\x1b[H", "|abc"},
		{"ctrl+a and ctrl+e", "abc\x01\x05", "abc|"},
		{"delete", "abc\x01\x1b[3~", "|bc"},
		{"ctrl+d", "abc\x01\x04", "|bc"},
		{"ctrl+u", "abc def\x02\x02\x15", "|ef"},
		{"ctrl+k", "abc def\x01\x06\x0b", "a|"},
		{"ctrl+w", "abc def  \x17", "abc |"},
		{"alt+backspace", "abc def\x1b\x7f", "abc |"},
		{"left at the start", "\x1b[Da", "a|"},
		{"right at the end", "a\x1b[C\x1b[Cb", "ab|"},
		{"a paste", "a\x1b[200~x\ny\x1b[201~b", "ax yb|"},
		{"ctrl+x ignored", "a\x18", "a|"},
		{"alt+x ignored", "a\x1bx", "a|"},
	} {
		l := New(0, ansi.WcWidth)
		typeBytes(t, l, c.in)
		if got := at(l); got != c.want {
			t.Errorf("%s: %q gives %q; want %q", c.name, c.in, got, c.want)
		}
	}
}

// TestClusters: a ZWJ sequence, a flag and a combining mark are each one
// cursor step, under both methods; a combining mark typed after its base
// joins its cluster.
func TestClusters(t *testing.T) {
	for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
		l := New(0, m)
		typeBytes(t, l, "a👨‍👩‍👧🇫🇷éb")
		if len(l.clusters) != 5 {
			t.Errorf("method %d: %q is %d clusters; want 5", m, l.Text(), len(l.clusters))
		}
		typeBytes(t, l, "\x1b[D\x1b[D\x7f")
		if got := at(l); got != "a👨‍👩‍👧|éb" {
			t.Errorf("method %d: a backspace after two lefts gives %q; want the flag removed whole", m, got)
		}
		l = New(0, m)
		typeBytes(t, l, "e")
		l.Insert("́")
		if len(l.clusters) != 1 || at(l) != "é|" {
			t.Errorf("method %d: a mark typed after its base gives %q, %d clusters; want one", m, at(l), len(l.clusters))
		}
	}
}

// TestLimit: the bound is in bytes, and a cluster that does not fit is
// left out whole.
func TestLimit(t *testing.T) {
	l := New(8, ansi.WcWidth)
	typeBytes(t, l, "abcde")
	l.Insert("世界")
	if got := l.Text(); got != "abcde世" {
		t.Errorf("the line is %q; want abcde世, 8 bytes", got)
	}
	typeBytes(t, l, "x")
	if got := l.Text(); got != "abcde世" {
		t.Errorf("a key past the bound gives %q", got)
	}
	l.Insert("\x1b[31mred\x07")
	if got := l.Text(); got != "abcde世" {
		t.Errorf("an escape inserted gives %q; want it cleaned away", got)
	}
}

// TestView: the part shown fits the width, the cursor's cell counted, and
// holds the cursor; at the end the newest text shows.
func TestView(t *testing.T) {
	l := New(0, ansi.WcWidth)
	typeBytes(t, l, "the quick brown fox")
	for _, c := range []struct {
		keys, want string
		col        int
	}{
		{"", "brown fox", 9},
		{"\x01", "the quick ", 0},
		{"\x01" + strings.Repeat("\x1b[C", 6), "the quick ", 6},
		{"\x01" + strings.Repeat("\x1b[C", 12), " quick bro", 9},
	} {
		typeBytes(t, l, c.keys)
		view, col := l.View(10)
		if view != c.want || col != c.col {
			t.Errorf("after %q: View(10) = %q, %d; want %q, %d", c.keys, view, col, c.want, c.col)
		}
		if w := ansi.WcWidth.StringWidth(view); w > 10 || (col == w && w+1 > 10) {
			t.Errorf("after %q: %q is %d cells, the cursor at %d, in 10", c.keys, view, w, col)
		}
		typeBytes(t, l, "\x05")
	}
	wide := New(0, ansi.WcWidth)
	typeBytes(t, wide, "世界世界")
	if view, col := wide.View(5); view != "世界" || col != 4 {
		t.Errorf("wide: View(5) = %q, %d; want 世界, 4", view, col)
	}
	if view, col := l.View(0); view != "the quick brown fox" || col != 19 {
		t.Errorf("View(0) = %q, %d; want all of it", view, col)
	}
}

// FuzzLine: any key stream keeps the text valid UTF-8, within its bound,
// and the cursor within the text, and View within its width.
func FuzzLine(f *testing.F) {
	f.Add([]byte("ab\x1b[D👨‍👩‍👧\x7f\x17é\x01\x0b"), uint8(7))
	f.Fuzz(func(t *testing.T, b []byte, width uint8) {
		l := New(32, ansi.GraphemeWidth)
		typeBytes(t, l, string(b))
		text := l.Text()
		if !utf8.ValidString(text) || len(text) > 32 || l.pos < 0 || l.pos > len(l.clusters) {
			t.Fatalf("text %q, %d bytes, cursor %d of %d", text, len(text), l.pos, len(l.clusters))
		}
		w := int(width%40) + 2
		view, col := l.View(w)
		if vw := ansi.GraphemeWidth.StringWidth(view); vw > w || col > vw {
			t.Fatalf("View(%d) = %q, %d cells, cursor %d", w, view, vw, col)
		}
	})
}
