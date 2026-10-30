package transcript_test

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"
	"github.com/maccavelli/go-tui-lib/v2/lineedit"
	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
	"github.com/maccavelli/go-tui-lib/v2/transcript"
)

// guide:host
// host is a conversation's state, on the loop's goroutine: the region the
// composer draws in, the transcript, and the reply streaming.
type host struct {
	out    io.Writer
	region *inline.Region
	c      *lineedit.Composer
	tr     *transcript.Transcript
	reply  *stream.Printer
	prompt string // the composer's WithPrompt
	cols   int
}

func (h *host) write(s string) { _, _ = io.WriteString(h.out, s) }

// submitted keeps the line the composer left in the scrollback, and
// starts the reply.
func (h *host) submitted(text string) {
	h.tr.AddText(h.prompt + text + "\n")
	h.reply = stream.NewPrinter(stream.WithMarkup(stream.DefaultMarkup()), stream.WithWidth(h.cols))
	h.tr.AddReply(h.reply)
}

// chunk prints the reply's stable lines above the composer.
func (h *host) chunk(s string) {
	if out := h.reply.Write(s); out != "" {
		h.write(h.region.Print(out))
	}
}

// done ends the reply.
func (h *host) done() {
	if out := h.reply.Flush(); out != "" {
		h.write(h.region.Print(out))
	}
}

// resize gives each part the new width.
func (h *host) resize(cols int, now time.Time) {
	h.cols = cols
	h.c.Resize(cols, now)
	h.tr.Resize(cols, now)
	if h.reply != nil {
		h.reply.SetWidth(cols)
	}
}

// idle reprints the conversation once a resize has settled, when the
// reply holds nothing back.
func (h *host) idle(now time.Time) {
	idle := h.reply == nil || h.reply.Idle()
	if out := h.tr.HandleTimeout(now, idle); out != "" {
		h.write(h.region.Clear() + out + h.region.Draw(h.c.View(h.cols)))
	}
}

// guide:end

// A line-mode host's conversation on a test screen in place of the
// terminal (docs/guides/line-mode.md): a line is submitted, a reply
// streams above the composer, the terminal narrows from 60 columns to 40,
// and once the resize settles and the reply has ended, the host reprints
// the conversation at the new width. The terminal re-wrapped the old rows
// itself; the reprint lays them out again, and clears the scrollback, with
// the shell's output from before the program.
func ExampleTranscript() {
	scr := inlinetest.NewScreen(60, 12, ansi.WcWidth)
	scr.Cooked = true
	_, _ = scr.Write([]byte("the shell's output, from before the program\n"))
	scr.Cooked = false

	region := inline.NewRegion(inline.WithoutCursorQuery())
	c := lineedit.NewComposer(lineedit.WithRegion(region), lineedit.WithGlyphs(glyph.ASCII()), lineedit.WithPrompt("> "))
	h := &host{out: scr, region: region, c: c, tr: transcript.NewTranscript(), prompt: "> "}
	drain := func() {
		for e := c.NextEvent(); e.Kind != lineedit.EditNone; e = c.NextEvent() {
			switch e.Kind {
			case lineedit.EditWrite:
				h.write(e.Data)
			case lineedit.EditSubmit:
				h.submitted(e.Text)
			}
		}
	}

	now := time.Unix(1, 0)
	h.resize(60, now)
	drain()
	var dec termkey.Decoder
	for _, in := range dec.Decode([]byte("explain the reprint\r"), nil) {
		c.HandleInput(in, now)
		drain()
	}
	for _, s := range []string{
		"The transcript keeps every piece the program has shown, ",
		"and draws it again at the new width once resizing settles.\n",
		"\nIt clears the scrollback ",
		"so the rows line up.\n",
	} {
		h.chunk(s)
	}

	now = now.Add(time.Second)
	scr.Resize(40, inline.ReflowDown)
	h.resize(40, now)
	drain()
	due, _ := h.tr.Timeout()
	h.done()
	h.idle(due)
	fmt.Print(scr.Text())
	fmt.Println(len(scr.Scrollback()), "rows of scrollback")
	// Output:
	// > explain the reprint
	// The transcript keeps every piece the
	// program has shown, and draws it again at
	// the new width once resizing settles.
	//
	// It clears the scrollback so the rows
	// line up.
	// >
	// 0 rows of scrollback
}
