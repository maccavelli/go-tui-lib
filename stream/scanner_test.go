package stream_test

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// scanned is each line of doc as a Scanner tells it: its kind, then
// "+cut" and "+split" where they are set.
func scanned(doc string) string {
	var s stream.Scanner
	var out []string
	for l := range strings.SplitSeq(strings.TrimSuffix(doc, "\n"), "\n") {
		b := s.Scan(l)
		w := b.Kind.String()
		if b.Cut {
			w += "+cut"
		}
		if b.Split {
			w += "+split"
		}
		out = append(out, w)
	}
	return strings.Join(out, " ")
}

// TestScanner: each line's kind, and where a cut and a relaxed cut may
// fall (docs/decisions/0009-MADR-streaming-content-engine.md §4).
func TestScanner(t *testing.T) {
	for _, c := range []struct{ name, doc, want string }{
		{"a backtick fence, a blank inside", "```go\nx\n\ny\n```\n", "fence code+split code+split code+split fence-end"},
		{"a tilde fence, a longer closer", "~~~\nx\n~~~~\n", "fence code+split fence-end"},
		{"a shorter closer closes nothing", "````\nx\n```\n````\n", "fence code+split code+split fence-end"},
		{"a closer with an info string is code", "```\n```x\n```\n", "fence code+split fence-end"},
		{"an opener indented three", "   ```\nx\n```\n", "fence code+split fence-end"},
		{"an opener indented four is code", "    ```\n", "indented"},
		{"a backtick in a backtick fence's info", "``` a`b\n", "paragraph"},
		{"an unclosed fence", "```\nx\n\ny\n", "fence code+split code+split code+split"},
		{"a cut after a fence", "```\nx\n```\n\nnext\n", "fence code+split fence-end blank paragraph+cut"},
		{"a list, a loose item, then a paragraph", "- a\n- b\n\n- c\n\nnext\n", "list list+split blank list+split blank paragraph+cut"},
		{"an item's content after a blank", "- a\n\n  more\n", "list blank paragraph"},
		{"a new list after a blank", "- a\n\n* b\n", "list blank list+cut"},
		{"an ordered list", "1. a\n2. b\n", "list list+split"},
		{"a lazy continuation", "- a\nb\n", "list continuation"},
		{"quotes", "> a\n> b\n\n> c\n", "quote quote+split blank quote+cut"},
		{"HTML 1, a blank inside", "<script>\nx\n\ny\n</script>\nz\n", "html html html html html paragraph"},
		{"HTML 2, on one line", "<!-- c -->\n\np\n", "html blank paragraph+cut"},
		{"HTML 3", "<?php\n?>\n", "html html"},
		{"HTML 4", "<!DOCTYPE html>\n", "html"},
		{"HTML 5", "<![CDATA[\nx\n]]>\n", "html html html"},
		{"HTML 6, ended by a blank", "<div>\nx\n\np\n", "html html blank paragraph+cut"},
		{"HTML 7, ended by a blank", "<span>\nx\n\np\n", "html html blank paragraph+cut"},
		{"HTML 7 does not interrupt a paragraph", "a\n<span>\n", "paragraph paragraph+split"},
		{"indented code across a blank", "    a\n\n    b\n\nc\n", "indented blank indented blank paragraph+cut"},
		{"a table", "a | b\n--- | ---\n1 | 2\n\nnext\n", "paragraph table table-row blank paragraph+cut"},
		{"setext headings", "Title\n===\n\npara\n---\n", "paragraph setext blank paragraph+cut setext"},
		{"definitions", "[x]: http://e\n[^1]: note\n", "definition definition"},
		{"a paragraph's lines", "a\nb\nc\n", "paragraph paragraph+split paragraph+split"},
		{"blank lines first make no cut", "\n\na\n", "blank blank paragraph"},
		{"a heading and a break", "# h\n\n***\n", "heading blank break+cut"},
	} {
		if got := scanned(c.doc); got != c.want {
			t.Errorf("%s: %s; want %s", c.name, got, c.want)
		}
	}
}

// TestScannerFields: a fence's line carries the fence's opening line, and
// a definition its label.
func TestScannerFields(t *testing.T) {
	var s stream.Scanner
	s.Scan("~~~ python")
	if b := s.Scan("x = 1"); b.Fence != "~~~ python" {
		t.Errorf("a fence's line: Fence %q; want the opening line", b.Fence)
	}
	if b := s.Scan("~~~"); b.Fence != "" {
		t.Errorf("the closing line: Fence %q; want none", b.Fence)
	}
	if b := s.Scan("[Foo Bar]: /url"); b.Kind != stream.BlockDefinition || b.Label == "" {
		t.Errorf("a definition: %+v; want its label", b)
	}
}
