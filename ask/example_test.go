package ask_test

import (
	"fmt"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/ask"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// Asking a question on a test screen, as a line-mode host asks it on the
// terminal (docs/guides/line-mode.md): the host gives the Asker the
// width, feeds it each decoded input, and writes what it asks until it
// is answered. Here the user rejects, and types why; the question and the
// answer, in words, stay in the scrollback.
func ExampleAsker() {
	// guide:question
	q := ask.Question{
		Title: "Run go test ./... in the repository?",
		Rows: []ask.Row{
			{Label: "Allow", Answers: []ask.Answer{
				{ID: "once", Key: "1", Label: "once"},
				{ID: "session", Key: "2", Label: "this session", Echo: "allowed for this session"},
			}},
			{Label: "Reject", Answers: []ask.Answer{
				{ID: "why", Key: "3", Label: "and say why", Echo: "rejected", Reason: true},
			}},
		},
		Cancel: ask.Answer{ID: "cancel", Key: "c", Label: "cancel"},
	}
	// guide:end
	scr := inlinetest.NewScreen(60, 10, ansi.WcWidth)
	a := ask.NewAsker(q, ask.WithGlyphs(glyph.ASCII()))
	now := time.Unix(0, 0)
	var resp ask.Response
	drain := func() {
		for e := a.NextEvent(); e.Kind != ask.AskNone; e = a.NextEvent() {
			switch e.Kind {
			case ask.AskWrite:
				_, _ = scr.Write([]byte(e.Data))
			case ask.AskDone:
				resp = e.Response
			}
		}
	}

	a.Resize(60, now)
	drain()
	var dec termkey.Decoder
	for _, in := range dec.Decode([]byte("3not yet\r"), nil) {
		a.HandleInput(in, now)
		drain()
	}
	fmt.Printf("%s %q\n", resp.Answer.ID, resp.Reason)
	fmt.Print(scr.Text())
	// Output:
	// why "not yet"
	// Run go test ./... in the repository?
	//   Allow   [1] once  [2] this session
	//   Reject  [3] and say why   [c] cancel
	// > rejected: not yet
}
