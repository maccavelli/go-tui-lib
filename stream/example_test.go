package stream_test

import (
	"fmt"

	"github.com/maccavelli/go-tui-lib/v2/stream"
)

// A reply printed as it streams in (docs/guides/line-mode.md): each Write
// gives the lines that can no longer change, wrapped at whole words to
// the width. A paragraph's line waits for the next, which could make it a
// heading; a table waits for its end, then is laid out in columns; Flush
// gives the rest at the reply's end. The zero Markup styles nothing, so
// the lines are plain; DefaultMarkup styles them.
func ExamplePrinter() {
	p := stream.NewPrinter(stream.WithWidth(32))
	chunks := []string{
		"# The reply\n\nA paragraph that streams in, ",
		"and is wrapped at whole words to the width.\n\n",
		"| name | cells |\n|---|--:|\n| ok | 2 |\n",
		"| wide | 10 |\n",
	}
	for i, chunk := range chunks {
		fmt.Printf("-- write %d\n%s", i+1, p.Write(chunk))
	}
	fmt.Printf("-- flush\n%s", p.Flush())
	// Output:
	// -- write 1
	// # The reply
	//
	// -- write 2
	// A paragraph that streams in, and
	// is wrapped at whole words to the
	// width.
	//
	// -- write 3
	// -- write 4
	// -- flush
	// | name | cells |
	// | ---- | ----: |
	// | ok   |     2 |
	// | wide |    10 |
}
