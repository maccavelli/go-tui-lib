package lineinput

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// keys is the keys lineedit's defaults bind to a Line's commands, for
// these tests to type as a terminal sends them.
var keys = map[string]command.ID{
	"left": CharLeft, "ctrl+b": CharLeft, "right": CharRight, "ctrl+f": CharRight,
	"alt+b": WordLeft, "alt+f": WordRight, "home": LineStart, "ctrl+a": LineStart,
	"end": LineEnd, "ctrl+e": LineEnd, "backspace": DeleteBack, "ctrl+h": DeleteBack,
	"delete": DeleteForward, "ctrl+d": DeleteOrEOF, "alt+backspace": DeleteWordBack,
	"alt+d": DeleteWordForward, "ctrl+w": DeleteSpaceWordBack, "ctrl+k": DeleteToLineEnd,
	"ctrl+u": DeleteToLineStart, "ctrl+t": Transpose,
}

// typeBytes feeds b to l through a decoder, as a host does: a bound key
// runs its command, a key that types text inserts it, and a paste is
// inserted as one line.
func typeBytes(t *testing.T, l *Line, b string) {
	t.Helper()
	var d termkey.Decoder
	for _, in := range d.Flush(d.Decode([]byte(b), nil)) {
		switch in.Kind {
		case termkey.InputKey:
			if id, ok := keys[in.Key.Keystroke()]; ok {
				l.Do(id)
			} else if in.Key.Text != "" && in.Key.Mod&^termkey.ModShift == 0 {
				l.Insert(sanitize.Line(in.Key.Text))
			}
		case termkey.InputPaste:
			l.Insert(sanitize.Line(in.Text))
		}
	}
}

// at is the line's text with the cursor shown as |.
func at(l *Line) string {
	return strings.Join(l.clusters[:l.pos], "") + "|" + strings.Join(l.clusters[l.pos:], "")
}

// TestDo: each command, typed as a terminal sends its key.
func TestDo(t *testing.T) {
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
		{"ctrl+w over punctuation", "a b-c.d\x17", "a |"},
		{"alt+backspace", "abc def\x1b\x7f", "abc |"},
		{"alt+backspace over punctuation", "a b-c\x1b\x7f", "a b-|"},
		{"alt+b", "one two\x1bb", "one |two"},
		{"alt+f to the word's end", "one two\x01\x1bf", "one| two"},
		{"alt+f from a space", "one   two\x01\x06\x06\x06\x06\x1bf", "one   two|"},
		{"alt+d", "one two\x01\x1bd", "| two"},
		{"ctrl+t", "abcd\x1b[D\x1b[D\x14", "acb|d"},
		{"ctrl+t at the end", "abcd\x14", "abdc|"},
		{"ctrl+t at the start", "abcd\x01\x14", "|abcd"},
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
	if New(0, ansi.WcWidth).Do("lineedit.submit") {
		t.Error("Do took a command that is not the line's")
	}
}

// TestLines: the line's ends and its kills stop at a line break.
func TestLines(t *testing.T) {
	l := New(0, ansi.WcWidth)
	l.Insert("one two\nthree four")
	for _, c := range []struct {
		id   command.ID
		want string
	}{
		{LineStart, "one two\n|three four"},
		{CharLeft, "one two|\nthree four"},
		{LineStart, "|one two\nthree four"},
		{LineEnd, "one two|\nthree four"},
		{CharRight, "one two\n|three four"},
		{DeleteToLineEnd, "one two\n|"},
		{CharLeft, "one two|\n"},
		{DeleteToLineStart, "|\n"},
	} {
		l.Do(c.id)
		if got := at(l); got != c.want {
			t.Errorf("%s: %q; want %q", c.id, got, c.want)
		}
	}
}

// TestCommands: Commands lists every command Do runs, each once.
func TestCommands(t *testing.T) {
	seen := map[command.ID]bool{}
	for _, id := range Commands() {
		if seen[id] || !New(0, ansi.WcWidth).Do(id) {
			t.Errorf("%s: listed twice, or not one Do runs", id)
		}
		seen[id] = true
	}
	if len(seen) != 15 {
		t.Errorf("%d commands; want 15", len(seen))
	}
}

// TestClusters: a ZWJ sequence, a flag and a combining mark are each one
// cursor step, under both methods; a combining mark typed after its base
// joins its cluster.
func TestClusters(t *testing.T) {
	for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
		l := New(0, m)
		typeBytes(t, l, "a👨\u200d👩\u200d👧🇫🇷éb")
		if len(l.clusters) != 5 {
			t.Errorf("method %d: %q is %d clusters; want 5", m, l.Text(), len(l.clusters))
		}
		typeBytes(t, l, "\x1b[D\x1b[D\x7f")
		if got := at(l); got != "a👨\u200d👩\u200d👧|éb" {
			t.Errorf("method %d: a backspace after two lefts gives %q; want the flag removed whole", m, got)
		}
		l = New(0, m)
		typeBytes(t, l, "e")
		l.Insert("\u0301")
		if len(l.clusters) != 1 || at(l) != "e\u0301|" {
			t.Errorf("method %d: a mark typed after its base gives %q, %d clusters; want one", m, at(l), len(l.clusters))
		}
	}
}

// TestLimit: the bound is in bytes, or in runes, and a cluster that does
// not fit is left out whole.
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
	r := NewRunes(6, ansi.WcWidth)
	r.Insert("abcde世界")
	if got := r.Text(); got != "abcde世" {
		t.Errorf("runes: %q; want abcde世, 6 runes", got)
	}
	r.SetText("é\u0301x")
	if got := r.Text(); got != "é\u0301x" || r.Pos() != 2 {
		t.Errorf("SetText: %q, cursor %d", got, r.Pos())
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
	f.Add([]byte("ab\x1b[D👨\u200d👩\u200d👧\x7f\x17é\x01\x0b\x1bf\x1bd\x14"), uint8(7))
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
