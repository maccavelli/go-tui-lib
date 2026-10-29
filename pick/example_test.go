package pick_test

import (
	"fmt"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"
	"github.com/maccavelli/go-tui-lib/v2/pick"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// Picking a model on a test screen, as a line-mode host picks on the
// terminal (docs/guides/line-mode.md): the host gives the Chooser the
// width, feeds it each decoded input, and writes what it asks until it is
// done. Typing ranks the entries by the query, best first; enter chooses
// the first, and one line is left in the scrollback.
func ExampleChooser() {
	entries := []pick.Entry{
		{ID: "opus", Text: "anthropic/claude-opus-5-5", Detail: "1M"},
		{ID: "sonnet", Text: "anthropic/claude-sonnet-5-5", Detail: "1M"},
		{ID: "haiku", Text: "anthropic/claude-haiku-5-5", Detail: "200K"},
	}
	scr := inlinetest.NewScreen(50, 8, ansi.WcWidth)
	c := pick.NewChooser(entries, pick.WithGlyphs(glyph.ASCII()))
	now := time.Unix(0, 0)
	drain := func() {
		for e := c.NextEvent(); e.Kind != pick.PickNone; e = c.NextEvent() {
			switch e.Kind {
			case pick.PickWrite:
				_, _ = scr.Write([]byte(e.Data))
			case pick.PickDone:
				fmt.Printf("chose %s, canceled %v\n", e.Entry.ID, e.Canceled)
			}
		}
	}

	c.Resize(50, now)
	drain()
	var dec termkey.Decoder
	for _, in := range dec.Decode([]byte("son"), nil) {
		c.HandleInput(in, now)
		drain()
	}
	fmt.Print(scr.Text())
	for _, in := range dec.Decode([]byte("\r"), nil) {
		c.HandleInput(in, now)
		drain()
	}
	fmt.Print(scr.Text())
	// Output:
	// > son                                       1 of 1
	// > anthropic/claude-sonnet-5-5                   1M
	// chose sonnet, canceled false
	// > anthropic/claude-sonnet-5-5
}
