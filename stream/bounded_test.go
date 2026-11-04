package stream_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/safetext"
	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// numberedLines is n lines "line 1" to "line n", each with its break.
func numberedLines(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	return b.String()
}

// TestBoundedWithin: within its bytes, a Bounded keeps every line, the
// last as far as it has come, and leaves none out.
func TestBoundedWithin(t *testing.T) {
	b := stream.NewBounded()
	fmt.Fprint(b, "one\ntwo\nthr")
	if got := b.Lines(); strings.Join(got, "|") != "one|two|thr" || b.Omitted() != 0 {
		t.Fatalf("%q, %d omitted; want every line and none omitted", got, b.Omitted())
	}
}

// TestBoundedPast: past its bytes, a Bounded keeps its head and tail
// lines, the omitted line between them, counted in lines, however long
// they are, and the same at two widths.
func TestBoundedPast(t *testing.T) {
	b := stream.NewBounded(stream.WithMaxBytes(64), stream.WithHeadLines(2), stream.WithTailLines(2))
	fmt.Fprint(b, numberedLines(10))
	want := []string{"line 1", "line 2", "(6 lines not shown)", "line 9", "line 10"}
	if got := b.Lines(); strings.Join(got, "|") != strings.Join(want, "|") || b.Omitted() != 6 {
		t.Fatalf("%q, %d omitted; want %q, 6", got, b.Omitted(), want)
	}
	for _, w := range []int{8, 80} {
		if r := strings.ReplaceAll(b.Render(w), "\n", ""); !strings.Contains(r, "(6 lines not shown)") || b.Omitted() != 6 {
			t.Errorf("at %d: %q, %d omitted; want the one count", w, r, b.Omitted())
		}
	}

	long := stream.NewBounded(stream.WithMaxBytes(64), stream.WithHeadLines(1), stream.WithTailLines(1))
	fmt.Fprint(long, strings.Repeat(strings.Repeat("x", 200)+"\n", 5))
	if long.Omitted() != 3 {
		t.Errorf("5 lines of 200 bytes, one kept at each end: %d omitted; want 3 lines", long.Omitted())
	}
}

// TestBoundedLineCap: a line past its cap is cut and ends with the glyph
// set's Ellipsis, and output with no line break stays bounded.
func TestBoundedLineCap(t *testing.T) {
	b := stream.NewBounded(stream.WithMaxLineBytes(8), stream.WithGlyphs(glyph.ASCII()))
	fmt.Fprint(b, "abcdefghijkl\nshort\n")
	for range 1000 {
		fmt.Fprint(b, "no line break ")
	}
	got := b.Lines()
	if len(got) != 3 || got[0] != "abcdefgh~" || got[1] != "short" || got[2] != "no line ~" {
		t.Fatalf("%q; want the long lines cut at 8 bytes, with the ASCII Ellipsis", got)
	}
}

// TestBoundedClean: a line is cleaned by the rules, safetext.Text by
// default, then by StatusLine's, the floor: StatusLine keeps a tool's
// colour, a rule that keeps more is cut back to it, and a head line's open
// SGR is closed before the omitted line.
func TestBoundedClean(t *testing.T) {
	red := "\x1b[31mred\x1b[0m"
	for _, c := range []struct {
		name string
		o    []stream.Option
		in   string
		want string
	}{
		{"Text by default", nil, red + " \x1b[2J\n", "red "},
		{"StatusLine keeps colour", []stream.Option{stream.WithRules(safetext.StatusLine())}, red + " \x1b[2J\n", red + " "},
		{"the floor cuts a link StatusLine refuses", []stream.Option{stream.WithRules(safetext.Rules{KeepSGR: true})},
			"\x1b]8;;file:///etc/passwd\x1b\\x\x1b]8;;\x1b\\\n", "x\x1b]8;;\x1b\\"},
	} {
		b := stream.NewBounded(c.o...)
		fmt.Fprint(b, c.in)
		if got := b.Lines(); len(got) != 1 || got[0] != c.want {
			t.Errorf("%s: %q; want %q", c.name, got, c.want)
		}
	}

	b := stream.NewBounded(stream.WithRules(safetext.StatusLine()), stream.WithMaxBytes(16), stream.WithHeadLines(1), stream.WithTailLines(1))
	fmt.Fprint(b, "\x1b[31mopen red\nb\nc\nd\n")
	if got := b.Lines(); len(got) != 3 || got[0] != "\x1b[31mopen red\x1b[m" {
		t.Errorf("%q; want the head's open SGR closed before the omitted line", got)
	}
}

// TestBoundedCatalog: WithCatalog's words for the omitted line, and the
// defaults check as their own translation.
func TestBoundedCatalog(t *testing.T) {
	cat := msgcat.CatalogFunc(func(id msgcat.MessageID) (string, bool) {
		if id == stream.MsgOmitted {
			return "[%d hidden]", true
		}
		return "", false
	})
	b := stream.NewBounded(stream.WithCatalog(cat), stream.WithMaxBytes(16), stream.WithHeadLines(1), stream.WithTailLines(1))
	fmt.Fprint(b, numberedLines(5))
	if got := b.Lines(); len(got) != 3 || got[1] != "[3 hidden]" {
		t.Errorf("%q; want the catalogue's line", got)
	}
	if err := msgcat.Check(stream.Messages(), stream.Messages()); err != nil {
		t.Error(err)
	}
	if err := msgcat.Check(stream.Messages(), cat); err != nil {
		t.Error(err)
	}
}
