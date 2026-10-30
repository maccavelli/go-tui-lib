// Command linemode is a line-mode host: raw mode, one reader and one
// region, and a Composer's loop, the program docs/guides/line-mode.md
// shows in "Raw mode, and what to restore". scripts/go-examples.py builds
// it, and runs it with no terminal, where it refuses
// (docs/decisions/0023-MADR-line-mode-components.md A11). Its run is
// lineedit's example's, and the guide holds the two to the same text.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/term"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/lineedit"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

func main() {
	lines, err := session()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%d line(s)\r\n", len(lines))
}

// guide:session
func session() ([]string, error) {
	in, out := int(os.Stdin.Fd()), int(os.Stdout.Fd())
	if !term.IsTerminal(in) || !term.IsTerminal(out) {
		return nil, errors.New("not a terminal")
	}
	old, err := term.MakeRaw(in)
	if err != nil {
		return nil, err
	}
	defer func() { _ = term.Restore(in, old) }()

	region := inline.NewRegion()
	_, _ = os.Stdout.WriteString(ansi.SetModeBracketedPaste)
	defer func() { _, _ = os.Stdout.WriteString(region.Restore() + ansi.ResetModeBracketedPaste) }()

	width := func() int {
		w, _, err := term.GetSize(out)
		if err != nil || w <= 0 {
			return 80
		}
		return w
	}
	c := lineedit.NewComposer(lineedit.WithRegion(region))
	return run(inline.NewReader(os.Stdin), os.Stdout, width, c)
}

// guide:end

// guide:run
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

// guide:end
