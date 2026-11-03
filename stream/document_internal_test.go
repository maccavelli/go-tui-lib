package stream

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// recorder is a Renderer that draws by Plain and keeps every input it was
// given, in order.
type recorder struct{ srcs []string }

func (r *recorder) Render(src string, width int) (string, error) {
	r.srcs = append(r.srcs, src)
	return Plain(ansi.WcWidth).Render(src, width)
}

// inputs is each committed segment's render input, as d gives it.
func inputs(d *Doc) []string {
	src := d.Source()
	var out []string
	for _, sg := range d.segs {
		out = append(out, d.input(sg.open, src[sg.start:sg.end], sg.close, true))
	}
	return out
}

// TestRelaxedCuts: a tail past RelaxAfter with no cut is cut at the next
// line its block can be split at: a paragraph's, a quote's, a fence's,
// which the segment closes and the next opens again with its info string,
// or a list item at its list's indent, which keeps its number; a table
// and an HTML block are never split (docs/decisions/0009-MADR-streaming-content-engine.md
// §4).
func TestRelaxedCuts(t *testing.T) {
	lines := func(f string, n int) string {
		var b strings.Builder
		for i := range n {
			fmt.Fprintf(&b, f, i+1)
		}
		return b.String()
	}
	for _, c := range []struct {
		name, src string
		check     func(segs []string) error
	}{
		{"a paragraph", lines("word%02d\n", 12), func(segs []string) error {
			if len(segs) < 2 {
				return fmt.Errorf("%d segments; want the paragraph split", len(segs))
			}
			return nil
		}},
		{"a quote", lines("> quote%02d\n", 12), func(segs []string) error {
			if len(segs) < 2 || !strings.HasPrefix(segs[1], "> ") {
				return fmt.Errorf("segments %q; want the quote split at a quote's line", segs)
			}
			return nil
		}},
		{"a fence", "```go\n" + lines("x = %02d\n", 12) + "```\n", func(segs []string) error {
			if len(segs) < 2 {
				return fmt.Errorf("%d segments; want the fence split", len(segs))
			}
			for i, s := range segs {
				if !strings.HasPrefix(s, "```go\n") || !strings.HasSuffix(s, "```\n") {
					return fmt.Errorf("segment %d %q: want it to open the fence with its info string and close it", i, s)
				}
			}
			return nil
		}},
		{"an ordered list", lines("%d. item\n", 9), func(segs []string) error {
			if len(segs) < 2 {
				return fmt.Errorf("%d segments; want the list split", len(segs))
			}
			for i, s := range segs[1:] {
				if s[0] < '2' || s[0] > '9' || !strings.HasPrefix(s[1:], ". item") {
					return fmt.Errorf("segment %d %q: want it to begin with its item's number", i+1, s)
				}
			}
			return nil
		}},
		{"a table", "a | b\n--- | ---\n" + lines("%02d | x\n", 12), func(segs []string) error {
			if len(segs) != 0 {
				return fmt.Errorf("segments %q; want the table whole", segs)
			}
			return nil
		}},
		{"an HTML block", "<div>\n" + lines("line%02d\n", 12) + "</div>\n", func(segs []string) error {
			if len(segs) != 0 {
				return fmt.Errorf("segments %q; want the block whole", segs)
			}
			return nil
		}},
	} {
		d := NewDoc(nil, WithRelaxAfter(16))
		d.Append(c.src)
		if err := c.check(inputs(d)); err != nil {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

// TestDefinitions: each segment's input carries the definitions committed
// so far, so that a reference resolves wherever its definition is, and a
// definition committed renders the segments once more
// (docs/decisions/0009-PLAN-streaming-content-engine.md D16).
func TestDefinitions(t *testing.T) {
	r := &recorder{}
	d := NewDoc(r)
	d.Append("see [x]\n\n")
	d.Append("more\n\n")
	d.Render(40)
	before := len(r.srcs)
	d.Append("[x]: https://e.x\n\nend\n\n")
	d.Render(40)
	ins := inputs(d)
	if len(ins) < 2 || !strings.HasSuffix(ins[0], "\n[x]: https://e.x\n") {
		t.Fatalf("segment inputs %q; want each to carry the definition", ins)
	}
	again := 0
	for _, s := range r.srcs[before:] {
		if strings.HasPrefix(s, "see [x]") {
			again++
		}
	}
	if again != 1 {
		t.Fatalf("the first segment rendered %d times after the definition; want once", again)
	}
}

// big is about 200 KB of one shape.
func big(shape string) string {
	var b strings.Builder
	for i := 0; b.Len() < 200<<10; i++ {
		switch shape {
		case "paragraphs":
			fmt.Fprintf(&b, "Line %d of a paragraph, words to wrap.\n", i)
			if i%5 == 4 {
				b.WriteString("\n")
			}
		case "prose":
			fmt.Fprintf(&b, "Line %d of one paragraph with no blank line.\n", i)
		case "fence":
			if i == 0 {
				b.WriteString("```text\n")
			}
			fmt.Fprintf(&b, "code line %d\n", i)
		case "list":
			fmt.Fprintf(&b, "- item %d\n", i)
		case "table":
			if i == 0 {
				b.WriteString("a | b\n--- | ---\n")
			}
			fmt.Fprintf(&b, "%d | row\n", i)
		}
	}
	return b.String()
}

// counter is a Renderer that counts what it is given.
type counter struct {
	calls, most int
}

func (c *counter) Render(src string, _ int) (string, error) {
	c.calls++
	c.most = max(c.most, len(src))
	return src, nil
}

// TestCost: over 200 KB in 20-byte chunks, a frame after each, each
// committed segment is rendered once a width, the calls are at most the
// segments and the frames, and no input is more than RelaxAfter and a line
// of tail, with a fence's opening and closing lines; a table, never split,
// is the stated exception, held whole in the tail
// (docs/decisions/0009-MADR-streaming-content-engine.md §4).
func TestCost(t *testing.T) {
	for _, shape := range []string{"paragraphs", "prose", "fence", "list", "table"} {
		src := big(shape)
		c := &counter{}
		d := NewDoc(c)
		frames, every := 0, 20
		if shape == "table" {
			every = 4 << 10 // the whole table each frame: fewer frames, the same exception
		}
		for i := 0; i < len(src); i += 20 {
			d.Append(src[i:min(i+20, len(src))])
			if (i/20)%(every/20) == 0 {
				d.Render(80)
				frames++
			}
		}
		if c.calls > len(d.segs)+frames {
			t.Errorf("%s: %d renders for %d segments and %d frames; want each segment once", shape, c.calls, len(d.segs), frames)
		}
		const line, fence = 64, 2 * len("```text\n")
		bound := defaultRelaxAfter + line + fence
		switch {
		case shape == "table" && c.most < len(src)/2:
			t.Errorf("table: the largest input %d bytes; want the table held whole, the stated exception", c.most)
		case shape != "table" && c.most > bound:
			t.Errorf("%s: an input of %d bytes; want at most %d", shape, c.most, bound)
		}
	}
}

// TestWidthBack: a width changed and back renders no segment more than
// twice, once at each width.
func TestWidthBack(t *testing.T) {
	r := &recorder{}
	d := NewDoc(r)
	d.Append(big("paragraphs")[:20<<10])
	for _, w := range []int{40, 60, 40, 60, 40} {
		d.Render(w)
	}
	seen := map[string]int{}
	for _, s := range r.srcs {
		seen[s]++
	}
	for _, in := range inputs(d) {
		if seen[in] > 2 {
			t.Fatalf("a segment rendered %d times over two widths; want at most 2", seen[in])
		}
	}
}
