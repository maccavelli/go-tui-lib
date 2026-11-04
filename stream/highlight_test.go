package stream_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// counting is a Highlighter that makes each line bold and counts each line
// it is given; its state is the lines it has seen, so it resumes, unless
// stateless, when it cannot.
type counting struct {
	seen      map[string]int
	langs     []string
	stateless bool
}

func newCounting() *counting { return &counting{seen: map[string]int{}} }

func (c *counting) Highlight(lang string, lines []string, state any) ([]string, any) {
	c.langs = append(c.langs, lang)
	out := make([]string, len(lines))
	for i, l := range lines {
		c.seen[l]++
		out[i] = "\x1b[1m" + l + "\x1b[m"
	}
	if c.stateless {
		return out, nil
	}
	n, _ := state.(int)
	return out, n + len(lines)
}

// total is how many times any line was highlighted.
func (c *counting) total() int {
	n := 0
	for _, v := range c.seen {
		n += v
	}
	return n
}

// TestHighlightGrowing: a fence growing in a Doc's tail, a frame after each
// line, has each line highlighted once, and shows it highlighted; the
// language is the info string's first word.
func TestHighlightGrowing(t *testing.T) {
	h := newCounting()
	d := stream.NewDoc(nil, stream.WithHighlighter(h))
	d.Append("```go run\n")
	for i := range 50 {
		d.Append(fmt.Sprintf("line %02d\n", i))
		d.Render(40)
	}
	for l, n := range h.seen {
		if n != 1 {
			t.Fatalf("%q highlighted %d times; want once", l, n)
		}
	}
	if f := d.Render(40); !strings.Contains(f, "\x1b[1mline 49\x1b[m") || h.langs[0] != "go" {
		t.Fatalf("frame %q, language %q; want the lines bold, go", f, h.langs[0])
	}
}

// TestHighlightCaches: a committed segment's code is highlighted once a
// width; a new revision, and a width change, highlight it again; a
// revision the same as the last does nothing.
func TestHighlightCaches(t *testing.T) {
	h := newCounting()
	d := stream.NewDoc(nil, stream.WithHighlighter(h))
	d.Append("```\na\nb\n```\n\nafter\n\nmore\n")
	d.Render(40)
	first := h.total()
	d.Append("x") // a new version, so the next frame is drawn
	d.Render(40)
	if h.total() != first {
		t.Fatalf("a second frame highlighted %d lines; want none", h.total()-first)
	}
	v := d.Version()
	d.SetRevision(7)
	d.Render(40)
	if h.total() != 2*first || d.Version() == v {
		t.Fatalf("after a new revision: %d highlighted, version %d after %d; want %d, raised", h.total(), d.Version(), v, 2*first)
	}
	d.SetRevision(7)
	d.Render(40)
	if h.total() != 2*first {
		t.Fatalf("the same revision again: %d highlighted; want %d", h.total(), 2*first)
	}
	d.Render(60)
	if h.total() != 3*first {
		t.Fatalf("a new width: %d highlighted; want %d", h.total(), 3*first)
	}
}

// TestHighlightBudget: the tail's closed fences are kept up to twice the
// byte limit; past it, every kept line is dropped, and the next frame
// highlights them again.
func TestHighlightBudget(t *testing.T) {
	h := newCounting()
	d := stream.NewDoc(nil, stream.WithHighlighter(h), stream.WithHighlightLimits(64, 0, 0))
	for _, x := range []string{"a", "b", "c"} {
		d.Append("```\n" + strings.Repeat(x, 40) + "\n```\n")
	}
	d.Render(80)
	once := h.total()
	d.Append("x") // a new version, so the next frame is drawn and not the last one given back
	d.Render(80)
	if h.total() == once {
		t.Fatalf("past the budget, a second frame highlighted nothing; want the dropped lines again")
	}
	small := newCounting()
	e := stream.NewDoc(nil, stream.WithHighlighter(small))
	e.Append("```\na\n```\n```\nb\n```\n")
	e.Render(80)
	n := small.total()
	e.Append("x")
	e.Render(80)
	if small.total() != n {
		t.Fatalf("within the budget, a second frame highlighted %d lines; want none", small.total()-n)
	}
}

// TestHighlightLimits: a Doc's block past any limit shows plain, whole; a
// Printer's is plain from the line that passes it, since a line is printed
// once.
func TestHighlightLimits(t *testing.T) {
	for _, c := range []struct {
		name string
		lim  stream.Option
		src  string
	}{
		{"bytes", stream.WithHighlightLimits(8, 0, 0), "```\nabc\ndefgh\n```\n"},
		{"lines", stream.WithHighlightLimits(0, 1, 0), "```\nabc\nde\n```\n"},
		{"a line", stream.WithHighlightLimits(0, 0, 4), "```\nab\nabcde\n```\n"},
	} {
		h := newCounting()
		d := stream.NewDoc(nil, stream.WithHighlighter(h), c.lim)
		d.Append(c.src)
		if f := d.Render(40); strings.ContainsRune(f, 0x1b) || h.total() != 0 {
			t.Errorf("%s, a Doc: %q, %d highlighted; want the block plain", c.name, f, h.total())
		}
		p := stream.NewPrinter(stream.WithHighlighter(newCounting()), c.lim)
		got := strings.Split(p.Write(c.src)+p.Flush(), "\n")
		if !strings.HasPrefix(got[1], "\x1b[1m") || strings.ContainsRune(got[2], 0x1b) {
			t.Errorf("%s, a Printer: %q; want the first line highlighted and the second plain", c.name, got)
		}
	}
}

// TestHighlightPrinter: a Printer highlights each code line once, from the
// state its last left; a Highlighter that cannot resume is given every
// line so far; an out that is not the line shows the line plain; a Doc
// whose Renderer is not Plain ignores the Highlighter.
func TestHighlightPrinter(t *testing.T) {
	h := newCounting()
	p := stream.NewPrinter(stream.WithHighlighter(h))
	got := p.Write("```\na\nb\nc\n```\n") + p.Flush()
	if got != "```\n\x1b[1ma\x1b[m\n\x1b[1mb\x1b[m\n\x1b[1mc\x1b[m\n```\n" || h.total() != 3 {
		t.Fatalf("%q, %d highlighted; want each line bold, once", got, h.total())
	}
	s := newCounting()
	s.stateless = true
	q := stream.NewPrinter(stream.WithHighlighter(s))
	q.Write("```\na\nb\nc\n```\n")
	if s.seen["a"] != 3 || s.seen["c"] != 1 {
		t.Fatalf("a Highlighter that cannot resume: %v; want every line so far each line", s.seen)
	}
	bad := stream.HighlighterFunc(func(_ string, lines []string, _ any) ([]string, any) { return []string{"other"}, 1 })
	r := stream.NewPrinter(stream.WithHighlighter(bad))
	if got := r.Write("```\na\n```\n") + r.Flush(); got != "```\na\n```\n" {
		t.Fatalf("an out that is not the line: %q; want it plain", got)
	}
	o := newCounting()
	d := stream.NewDoc(echo, stream.WithHighlighter(o))
	d.Append("```\na\n```\n")
	if d.Render(40); o.total() != 0 {
		t.Fatalf("a Doc with another Renderer highlighted %d lines; want none", o.total())
	}
	if f := d.Render(40); f != ansi.Strip(f) {
		t.Fatalf("%q; want it as the Renderer drew it", f)
	}
}
