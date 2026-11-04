package stream_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// viewDocs are documents with each kind of side-table.
var viewDocs = []string{
	"# Title with [a link](https://e.x/t)\n\nA paragraph that wraps, with [the documentation site](https://e.x/docs) in it, and more words.\n\n" +
		"```go\nx := 1\ny := 2\n```\n\n| a | b |\n| --- | --- |\n| 1 | 2 |\n\n    indented\n    code\n\nsee [ref] and [Ref][] too\n\n[ref]: https://e.x/r\n\nend, which commits the definition\n",
	"- an item with [x](http://a.b)\n- another\n\n> a quote with [y](http://c.d) here\n\nend",
}

// show is v as text, to compare two views and to print one.
func show(v stream.RenderView) string { return fmt.Sprintf("%+v", v) }

// TestViewIsRender: a view's rows joined are Render's frame, as the
// document streams and after Finish.
func TestViewIsRender(t *testing.T) {
	for _, src := range viewDocs {
		d := stream.NewDoc(nil, stream.WithRelaxAfter(48))
		for i := 0; i < len(src); i += 7 {
			d.Append(src[i:min(i+7, len(src))])
			if v, f := d.View(30), d.Render(30); strings.Join(v.Lines, "\n") != f || len(v.Offsets) != len(v.Lines) {
				t.Fatalf("after %d bytes: view %q, %d offsets; want Render's %q", i, v.Lines, len(v.Offsets), f)
			}
		}
		d.Finish()
		if v, f := d.View(30), d.Render(30); strings.Join(v.Lines, "\n") != f {
			t.Fatalf("after Finish: view %q; want Render's %q", v.Lines, f)
		}
	}
}

// TestViewChunking: a view is the same for the whole text and for any
// chunks of it, before Finish and after.
func TestViewChunking(t *testing.T) {
	for _, src := range viewDocs {
		whole := stream.NewDoc(nil, stream.WithRelaxAfter(48))
		whole.Append(src)
		for _, n := range []int{1, 3, 11} {
			parts := stream.NewDoc(nil, stream.WithRelaxAfter(48))
			for i := 0; i < len(src); i += n {
				parts.Append(src[i:min(i+n, len(src))])
			}
			if a, b := show(whole.View(30)), show(parts.View(30)); a != b {
				t.Fatalf("chunks of %d:\n%s\nwant\n%s", n, b, a)
			}
		}
	}
}

// TestViewTables: each side-table, its rows, offsets and source ranges.
func TestViewTables(t *testing.T) {
	src := viewDocs[0]
	d := stream.NewDoc(nil)
	d.Append(src)
	v := d.View(30)
	ids := map[int]string{}
	for _, l := range v.Links {
		ids[l.ID] = l.URL
		if got := strings.TrimSpace(v.Lines[l.Row][l.From:l.To]); got == "" {
			t.Errorf("link %+v covers no text on %q", l, v.Lines[l.Row])
		}
	}
	for _, want := range []struct{ text, url string }{
		{"[a link]", "https://e.x/t"}, {"[the documentation site]", "https://e.x/docs"},
		{"[ref]", "https://e.x/r"}, {"[Ref][]", "https://e.x/r"},
	} {
		if u, ok := ids[strings.Index(src, want.text)]; !ok || u != want.url {
			t.Errorf("link %s: %q, %v; want %q at its offset", want.text, u, ok, want.url)
		}
	}
	if len(v.Code) != 2 || v.Code[0].Info != "go" || src[v.Code[0].Start:v.Code[0].End] != "x := 1\ny := 2\n" ||
		v.Code[1].Info != "" || src[v.Code[1].Start:v.Code[1].End] != "    indented\n    code\n" {
		t.Errorf("code %+v; want the fence's body and the indented block's", v.Code)
	}
	if c := v.Code[0]; c.Rows != 4 || !strings.HasPrefix(v.Lines[c.Row], "```go") {
		t.Errorf("the fence's rows %+v in %q; want its four", c, v.Lines)
	}
	if len(v.Tables) != 1 || fmt.Sprint(v.Tables[0].Cells) != "[[a b] [1 2]]" || v.Tables[0].Rows != 3 {
		t.Errorf("tables %+v; want one, its cells and three rows", v.Tables)
	}
	for i, o := range v.Offsets {
		if v.Lines[i] != "" && !strings.HasPrefix(src[o:], strings.Fields(v.Lines[i])[0]) && !strings.Contains(src[o:], strings.Fields(v.Lines[i])[0]) {
			t.Errorf("row %d %q at offset %d: %q", i, v.Lines[i], o, src[o:min(o+20, len(src))])
		}
	}
}

// TestViewWrappedLink: a link wrapped over rows has one ID on each, its
// "[" offset, in every frame as the tail grows; and no frame carries a
// part of an earlier frame's tail.
func TestViewWrappedLink(t *testing.T) {
	d := stream.NewDoc(nil)
	src := "Read [the whole documentation site](https://e.x/d) before"
	at := strings.Index(src, "[")
	d.Append(src)
	for _, more := range []string{" you", " start", " work."} {
		d.Append(more)
		v := d.View(16)
		if strings.Join(v.Lines, "\n") != d.Render(16) {
			t.Fatalf("rows %q; want this frame's alone, %q", v.Lines, d.Render(16))
		}
		if len(v.Links) < 2 {
			t.Fatalf("%q: links %+v; want the link in parts over rows", v.Lines, v.Links)
		}
		for i, l := range v.Links {
			if l.ID != at || l.Row != v.Links[0].Row+i {
				t.Fatalf("links %+v; want one link, a part on each of consecutive rows, of ID %d", v.Links, at)
			}
		}
	}
}

// TestViewSplitFence: a fence a relaxed cut split is one ViewCode, its
// body's source whole.
func TestViewSplitFence(t *testing.T) {
	var body strings.Builder
	for i := range 20 {
		fmt.Fprintf(&body, "line %02d of code\n", i)
	}
	src := "```sh\n" + body.String() + "```\n\nafter\n"
	d := stream.NewDoc(nil, stream.WithRelaxAfter(64))
	d.Append(src)
	v := d.View(40)
	if len(v.Code) != 1 || v.Code[0].Info != "sh" || src[v.Code[0].Start:v.Code[0].End] != body.String() {
		t.Fatalf("code %+v; want one block, the whole body", v.Code)
	}
}

// TestViewOtherRenderer: with a Renderer that is not a ViewRenderer, a
// row's offset is its segment's, and there are no side-tables.
func TestViewOtherRenderer(t *testing.T) {
	d := stream.NewDoc(bracket)
	d.Append("a [x](http://e)\n\nb\n")
	v := d.View(40)
	if show(v) != show(stream.RenderView{Lines: []string{"[a [x](http://e)]", "", "[b]"}, Offsets: []int{0, 17, 17}}) {
		t.Fatalf("%s; want each row at its segment, nothing else", show(v))
	}
}
