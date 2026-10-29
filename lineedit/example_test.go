package lineedit_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"
	"github.com/maccavelli/go-tui-lib/v2/lineedit"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// widthPoll is how often the loop looks at the terminal's width when
// nothing else wakes it: a host may watch SIGWINCH instead, where there is
// one.
const widthPoll = 200 * time.Millisecond

// run is a line-mode host's loop around a Composer
// (docs/guides/line-mode.md): it drains the Composer's events, writing
// what it asks; reads the input only when asked, until the Composer's
// deadline, the escape timeout or the next look at the width; decodes
// what it read and feeds each input in; and gives the Composer the width
// when it changes. It returns the lines submitted, at the end of the
// input.
func run(in *inline.Reader, out io.Writer, width func() int, c *lineedit.Composer) ([]string, error) {
	var (
		dec  termkey.Decoder
		buf  = make([]byte, 4096)
		cols = width()
		got  []string
	)
	c.Resize(cols, time.Now())
	for {
		for e := c.NextEvent(); e.Kind != lineedit.EditNone; e = c.NextEvent() {
			switch e.Kind {
			case lineedit.EditWrite:
				if _, err := io.WriteString(out, e.Data); err != nil {
					return got, err
				}
			case lineedit.EditSubmit:
				got = append(got, e.Text)
			case lineedit.EditInterrupt:
				c.SetText("") // ctrl+c's meaning is the host's: here, the line is cleared
			case lineedit.EditEOF:
				return got, nil
			}
		}

		now := time.Now()
		wake := now.Add(widthPoll)
		if t, ok := c.Timeout(); ok && t.Before(wake) {
			wake = t
		}
		if esc := now.Add(termkey.DefaultEscTimeout); dec.Pending() && esc.Before(wake) {
			wake = esc
		}
		ctx, cancel := context.WithDeadline(context.Background(), wake)
		n, err := in.Read(ctx, buf)
		cancel()

		now = time.Now()
		ins := dec.Decode(buf[:n], nil)
		if n == 0 && dec.Pending() {
			ins = dec.Flush(ins) // the input went quiet with a sequence held
		}
		for _, k := range ins {
			c.HandleInput(k, now)
		}
		switch {
		case err == nil:
		case errors.Is(err, context.DeadlineExceeded):
			if t, ok := c.Timeout(); ok && !now.Before(t) {
				c.HandleTimeout(now)
			}
		case errors.Is(err, io.EOF):
			c.Cancel()
		default:
			return got, err
		}
		if w := width(); w != cols {
			cols = w
			c.Resize(cols, now)
		}
	}
}

// A line-mode host's loop around a Composer, on a test screen in place of
// the terminal: two lines typed, each submitted with enter and left in the
// scrollback, then the end of the input.
func ExampleComposer() {
	scr := inlinetest.NewScreen(40, 6, ansi.WcWidth)
	c := lineedit.NewComposer(lineedit.WithGlyphs(glyph.ASCII()))
	in := inline.NewReader(strings.NewReader("hello, world\rand a second line\r"))
	got, err := run(in, scr, func() int { return 40 }, c)
	fmt.Printf("%q %v\n", got, err)
	fmt.Print(scr.Text())
	// Output:
	// ["hello, world" "and a second line"] <nil>
	// > hello, world
	// > and a second line
	// >
}
