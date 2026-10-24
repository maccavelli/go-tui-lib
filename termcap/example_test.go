package termcap_test

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/termcap"
)

// A line-mode CLI drives a Probe itself: it writes what the probe asks
// for, reads each reply whole from its raw-mode input, parses it, and
// drains the events again. Here the terminal's replies are written out:
// DA2, then DA1, the sentinel, which ends the probe.
func ExampleProbe() {
	p := termcap.NewProbe(termcap.WithGOOS("linux"))
	now := time.Now()
	p.Start(termcap.Env{"TERM=xterm-256color", "SSH_TTY=/dev/pts/1"}, now)
	replies := []string{"\x1b[>1;95;0c", "\x1b[?1;2c"}
	for {
		e := p.NextEvent()
		switch e.Kind {
		case termcap.ProbeNoEvent:
			if len(replies) == 0 {
				return
			}
			// A real CLI reads the next reply here, and calls
			// HandleTimeout once Timeout's deadline has passed.
			p.HandleReply(termcap.ParseReply(replies[0]), now)
			replies = replies[1:]
		case termcap.ProbeWrite:
			fmt.Printf("write %q\n", e.Data) // to the terminal, in a real CLI
		case termcap.ProbeDone:
			fmt.Println("brand:", e.Caps.Brand.Value, "complete:", e.Caps.Complete)
		}
	}
	// Output:
	// write "\x1b[?u\x1b[?2031$p\x1b[?2048$p\x1b[?1004$p\x1b[?996n\x1b]11;?\a\x1b[>c\x1b[c"
	// brand: apple-terminal complete: true
}

// A doctor command writes the report to the writer it is given; a
// program's main may pass os.Stdout. Here it goes to a buffer, and its
// first lines are printed.
func ExampleReport() {
	var c termcap.Caps
	c.Terminal.Set("WezTerm 20240203", termcap.Queried)
	c.Brand.Set(termcap.BrandWezTerm, termcap.Queried)
	c.KittyKeyboard.Set(termcap.Supported, termcap.Queried)

	var b strings.Builder
	if err := termcap.Report(&b, c, termcap.WithReportWidth(80)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	for _, l := range strings.Split(b.String(), "\n")[:3] {
		fmt.Println(l)
	}
	// Output:
	// complete               no
	// timed_out              no
	// terminal               WezTerm 20240203 (query)
}
