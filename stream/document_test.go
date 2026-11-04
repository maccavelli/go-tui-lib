package stream_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// echo is a Renderer that gives its source back.
var echo = stream.RendererFunc(func(src string, _ int) (string, error) { return src, nil })

// bracket is a Renderer that shows each input it is given, trimmed, in
// brackets, so that a frame shows where the cuts fell.
var bracket = stream.RendererFunc(func(src string, _ int) (string, error) {
	return "[" + strings.TrimSpace(src) + "]", nil
})

// TestDocCuts: a blank line where nothing stays open is a cut, and each
// segment is rendered alone; a fence's blank, a list's loose item and
// indented code's blank are not cuts
// (docs/decisions/0009-MADR-streaming-content-engine.md §4).
func TestDocCuts(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{"paragraphs", "a\n\nb\n\nc\n", "[a]\n\n[b]\n\n[c]"},
		{"a fence's blank", "```\nx\n\ny\n```\n\nz\n", "[```\nx\n\ny\n```]\n\n[z]"},
		{"a loose list", "- a\n\n- b\n\nc\n", "[- a\n\n- b]\n\n[c]"},
		{"indented code", "    a\n\n    b\n\nc\n", "[a\n\n    b]\n\n[c]"},
		{"a table", "a | b\n--- | ---\n1 | 2\n\nnext\n", "[a | b\n--- | ---\n1 | 2]\n\n[next]"},
		{"blank lines first", "\n\na\n\nb\n", "[a]\n\n[b]"},
		{"a cut waits for the next complete line", "a\n\nb", "[a\n\nb]"},
		{"a partial line after a blank stays in the tail", "a\n\nb\n\nc", "[a]\n\n[b\n\nc]"},
	} {
		d := stream.NewDoc(bracket)
		d.Append(c.src)
		if got := d.Render(80); got != c.want {
			t.Errorf("%s: %q; want %q", c.name, got, c.want)
		}
	}
}

// TestDocHoldback: the partial last line stops before an inline construct
// it opens and has not closed, as goose holds one; whole once it closes;
// WithoutInlineHoldback and WithoutPartialLine as they say; a fence's
// partial line is code, and shown whole.
func TestDocHoldback(t *testing.T) {
	for _, c := range []struct {
		name, src, want string
		o               []stream.Option
	}{
		{"strong, open", "Hello **wor", "Hello ", nil},
		{"strong, closed", "Hello **world** and", "Hello **world** and", nil},
		{"emphasis, open", "an *em", "an ", nil},
		{"a code span, open", "run `go te", "run ", nil},
		{"a code span, closed", "run `go test` now", "run `go test` now", nil},
		{"a strikethrough, open", "was ~~wro", "was ", nil},
		{"a link, open", "see [docs](https://exa", "see ", nil},
		{"a link, closed", "see [docs](https://e.x) ok", "see [docs](https://e.x) ok", nil},
		{"a bracket alone", "a [b", "a [b", nil},
		{"snake_case", "use my_var_name", "use my_var_name", nil},
		{"a list marker", "* item", "* item", nil},
		{"without holdback", "Hello **wor", "Hello **wor", []stream.Option{stream.WithoutInlineHoldback()}},
		{"without the partial line", "done\nHello", "done\n", []stream.Option{stream.WithoutPartialLine()}},
		{"inside a fence", "```\nx := **p", "```\nx := **p", nil},
	} {
		d := stream.NewDoc(echo, c.o...)
		d.Append(c.src)
		if got := d.Render(80); got != strings.TrimSuffix(c.want, "\n") {
			t.Errorf("%s: %q; want %q", c.name, got, c.want)
		}
	}
}

// TestDocMath: WithMath holds an unclosed \( or $$ in the partial line,
// until it closes or Finish; without it, they show
// (docs/decisions/0009-MADR-streaming-content-engine.md A1 §4).
func TestDocMath(t *testing.T) {
	for _, c := range []struct {
		name, src, want string
		math            bool
	}{
		{`an open \(`, `the area \(a^2`, "the area ", true},
		{`a closed \(`, `the area \(a^2\) is`, `the area \(a^2\) is`, true},
		{"an open $$", "so $$x + y", "so ", true},
		{"a closed $$", "so $$x$$ then", "so $$x$$ then", true},
		{"an escaped bracket stays", `a \[b`, `a \[b`, true},
		{`without math, \( shows`, `the area \(a^2`, `the area \(a^2`, false},
		{"without math, $$ shows", "so $$x + y", "so $$x + y", false},
	} {
		var o []stream.Option
		if c.math {
			o = append(o, stream.WithMath())
		}
		d := stream.NewDoc(echo, o...)
		d.Append(c.src)
		if got := d.Render(80); got != c.want {
			t.Errorf("%s: %q; want %q", c.name, got, c.want)
		}
		d.Finish()
		if got := d.Render(80); got != c.src {
			t.Errorf("%s, after Finish: %q; want the whole line", c.name, got)
		}
	}
}

// TestDialect: the model-output dialect, as grok renders it: only ~~
// strikes, so ~**10%** is a tilde and strong text, and ~~x~~ is struck
// (docs/decisions/0009-MADR-streaming-content-engine.md A1 §4).
func TestDialect(t *testing.T) {
	m := stream.DefaultMarkup()
	struck := m.Strike.Apply("~~x~~")
	for _, c := range []struct{ src, want string }{
		{"~~x~~\n", struck + "\n"},
		{"~**10%**\n", "~" + m.Strong.Apply("**10%**") + "\n"},
		{"~x~\n", "~x~\n"},
	} {
		p := stream.NewPrinter(stream.WithMarkup(m))
		if got := p.Write(c.src) + p.Flush(); got != c.want {
			t.Errorf("%q: %q; want %q", c.src, got, c.want)
		}
	}
}

// TestDocFinish: after Finish, a frame is the Renderer's of the whole
// source, byte for byte; Finished says so, Version rose, Append does
// nothing and Write returns ErrClosed; Reset empties it.
func TestDocFinish(t *testing.T) {
	d := stream.NewDoc(bracket)
	d.Append("a\n\nb\n\nc **open")
	v := d.Version()
	d.Finish()
	want, _ := bracket.Render(d.Source(), 40)
	if got := d.Render(40); got != want || !d.Finished() || d.Version() <= v {
		t.Fatalf("after Finish: %q, %v, version %d after %d; want %q, finished, raised", got, d.Finished(), d.Version(), v, want)
	}
	d.Append("more")
	if _, err := d.Write([]byte("more")); !errors.Is(err, stream.ErrClosed) || strings.Contains(d.Source(), "more") {
		t.Fatalf("after Finish: Write %v, source %q; want ErrClosed and nothing added", err, d.Source())
	}
	d.Reset()
	if d.Source() != "" || d.Finished() || d.Render(40) != "" {
		t.Fatalf("after Reset: %q, %v; want an empty Doc", d.Source(), d.Finished())
	}
}

// TestDocClean: what the stream sends is cleaned before anything keeps
// it, so no frame holds an escape sequence the Renderer did not add.
func TestDocClean(t *testing.T) {
	d := stream.NewDoc(stream.Plain(ansi.WcWidth))
	for _, chunk := range []string{"a \x1b[31mred", "\x1b[0m word\n\n\x1b]0;title\x07", "b\x1b[2J\n"} {
		d.Append(chunk)
		if f := d.Render(40); strings.ContainsRune(f, 0x1b) || strings.ContainsRune(d.Source(), 0x1b) {
			t.Fatalf("a frame %q or the source %q holds ESC", f, d.Source())
		}
	}
	d.Finish()
	if f := d.Render(40); strings.ContainsRune(f, 0x1b) {
		t.Fatalf("the finished frame %q holds ESC", f)
	}
}

// failing is a Renderer that fails on any source holding "bad".
var failing = stream.RendererFunc(func(src string, _ int) (string, error) {
	if strings.Contains(src, "bad") {
		return "", errors.New("cannot")
	}
	return "[" + strings.TrimSpace(src) + "]", nil
})

// TestDocRenderError: a segment the Renderer fails on is shown as its
// source drawn by Plain, so no text is lost.
func TestDocRenderError(t *testing.T) {
	d := stream.NewDoc(failing)
	d.Append("good\n\nbad text\n\nend\n")
	if got := d.Render(80); got != "[good]\n\nbad text\n\n[end]" {
		t.Fatalf("%q; want the failed segment as Plain draws it", got)
	}
}

// FuzzDoc: over any text and any split points, a Doc does not panic,
// gives the same frame for the whole text and for its chunks, and after
// Finish gives Plain's render of the whole source.
func FuzzDoc(f *testing.F) {
	for _, s := range []string{
		"a\n\nb\n\nc", "```go\nx\n\ny\n```\n\nz", "- a\n\n- b\n\n  more\n\nc\n", "a | b\n--- | ---\n1 | 2\n\n",
		"Title\n===\n\n[x]: http://e\n\nsee [x]\n", "<div>\n\n</div>\n\n    code\n", "Hello **wor", "> q\n> r\n\nend\n",
	} {
		f.Add(s, []byte{1, 3, 7})
	}
	plain := stream.Plain(ansi.WcWidth)
	f.Fuzz(func(t *testing.T, s string, cuts []byte) {
		whole := stream.NewDoc(plain, stream.WithRelaxAfter(32))
		whole.Append(s)
		parts := stream.NewDoc(plain, stream.WithRelaxAfter(32))
		rest := s
		for _, c := range cuts {
			n := min(int(c)%17+1, len(rest))
			parts.Append(rest[:n])
			rest = rest[n:]
		}
		parts.Append(rest)
		if a, b := whole.Render(30), parts.Render(30); a != b {
			t.Fatalf("whole %q, in chunks %q", a, b)
		}
		if a, b := show(whole.View(30)), show(parts.View(30)); a != b {
			t.Fatalf("views: whole %s, in chunks %s", a, b)
		}
		if v := parts.View(30); strings.Join(v.Lines, "\n") != parts.Render(30) || len(v.Offsets) != len(v.Lines) {
			t.Fatalf("a view %q, %d offsets; want Render's rows", v.Lines, len(v.Offsets))
		}
		parts.Finish()
		want, _ := plain.Render(parts.Source(), 30)
		if got := parts.Render(30); got != want {
			t.Fatalf("after Finish %q; want Plain's %q", got, want)
		}
	})
}

// A streamed reply drawn a stable prefix at a time: the first paragraph is
// a segment, rendered once, while the second grows in the tail; after
// Finish, the frame is the Renderer's of the whole reply.
func ExampleDoc() {
	d := stream.NewDoc(stream.Plain(ansi.WcWidth))
	for _, chunk := range []string{"A first para", "graph.\n\nThen a **sec", "ond** one."} {
		d.Append(chunk)
		fmt.Printf("%q\n", d.Render(40))
	}
	d.Finish()
	fmt.Printf("%q\n", d.Render(40))
	// Output:
	// "A first para"
	// "A first paragraph.\n\nThen a "
	// "A first paragraph.\n\nThen a **second** one."
	// "A first paragraph.\n\nThen a **second** one."
}

// TestPlain: Plain wraps prose to the width and lays out a table, styles
// nothing, and prints no definition (docs/decisions/0009-PLAN-streaming-content-engine.md
// D16).
func TestPlain(t *testing.T) {
	p := stream.Plain(ansi.WcWidth)
	for _, c := range []struct {
		src   string
		width int
		want  string
	}{
		{"one two three four\n", 9, "one two\nthree\nfour\n"},
		{"a | bb\n--- | ---\nccc | d\n", 0, "| a   | bb  |\n| --- | --- |\n| ccc | d   |\n"},
		{"see [x]\n\n[x]: https://e.x\n", 0, "see [x]\n\n"},
	} {
		if got, err := p.Render(c.src, c.width); got != c.want || err != nil {
			t.Errorf("Plain(%q, %d) = %q, %v; want %q", c.src, c.width, got, err, c.want)
		}
	}
}
