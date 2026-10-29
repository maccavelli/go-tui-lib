package transcript_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"
	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/transcript"
	"github.com/maccavelli/go-tui-lib/v2/tuitest"
)

// TestReprintGolden: a conversation shown at 120 columns, gobble-cli's
// benchmark reply among it, reprinted on the test screen at each width of
// the matrix once the resize settles, the reply styled by DefaultMarkup and
// the notice dim under colour, plain without. A line kept whole and wider
// than the width, a heading's, a code line's or a table row's, takes the
// rows the screen wraps it into, and counts them against the 30.
func TestReprintGolden(t *testing.T) {
	reply, err := os.ReadFile("testdata/reply.txt")
	if err != nil {
		t.Fatal(err)
	}
	tuitest.Golden(t, "reprint", tuitest.Matrix{Widths: []int{60, 100}}, func(c tuitest.Case) string {
		markup, style := stream.Markup{}, stream.Style(0)
		if c.Color {
			markup, style = stream.DefaultMarkup(), stream.Dim
		}
		tr := transcript.NewTranscript(transcript.WithMaxRows(30), transcript.WithNoticeStyle(style))
		tr.AddText("the program's banner, shown first\n")
		tr.AddText("> summarise the reply\n")
		p := stream.NewPrinter(stream.WithMarkup(markup), stream.WithWidth(120))
		p.Write(string(reply))
		p.Flush()
		tr.AddReply(p)
		tr.AddDrawn(func(width int) string { return strings.Repeat("-", min(width, 40)) + "\n" })
		scr := inlinetest.NewScreen(c.Width, 40, ansi.WcWidth)
		tr.Resize(120, t0)
		tr.Resize(c.Width, t0)
		_, _ = scr.Write([]byte(tr.HandleTimeout(t0.Add(transcript.DefaultSettle), true)))
		return scr.Text()
	})
}

// TestHostReprint: a line-mode host on a test screen streams a reply above
// a one-row region, kept in the transcript; the terminal narrows; once the
// resize settles and the reply's Printer is idle, the host clears its
// region, writes the reprint and draws the region again. The screen then
// holds the conversation wrapped at the new width, the region below it,
// and no scrollback: what came before the program is gone with it.
func TestHostReprint(t *testing.T) {
	scr := inlinetest.NewScreen(60, 12, ansi.WcWidth)
	scr.Cooked = true
	_, _ = scr.Write([]byte("the shell's own output, from before the program\n"))
	scr.Cooked = false
	region := inline.NewRegion(inline.WithoutCursorQuery())
	tr := transcript.NewTranscript()
	now := t0
	write := func(s string) { _, _ = scr.Write([]byte(s)) }
	write(region.Resize(60, now))
	tr.Resize(60, now)
	prompt := []string{"> "}
	write(region.Draw(prompt, 0, 2))

	tr.AddText("> explain the reprint\n")
	write(region.Print("> explain the reprint\n"))
	write(region.Draw(prompt, 0, 2))
	p := stream.NewPrinter(stream.WithWidth(60))
	tr.AddReply(p)
	for _, chunk := range []string{"The transcript keeps every piece the program has shown, ", "and draws it again at the new width once resizing settles.\n", "\nIt clears the scrollback ", "so the rows line up.\n"} {
		if out := p.Write(chunk); out != "" {
			write(region.Print(out))
			write(region.Draw(prompt, 0, 2))
		}
	}

	now = now.Add(time.Second)
	scr.Resize(40, inline.ReflowDown)
	write(region.Resize(40, now))
	tr.Resize(40, now)
	p.SetWidth(40)
	due, ok := tr.Timeout()
	if !ok {
		t.Fatal("no settle deadline after the resize")
	}
	// The reply's last line is held until the next line or the reply's end
	// (Q10), so the host is not idle at the settle's end, and the reprint
	// waits for its next idle call.
	if p.Idle() {
		t.Fatal("the Printer is idle with its last paragraph line held")
	}
	if out := tr.HandleTimeout(due, p.Idle()); out != "" {
		t.Fatalf("reprinted with the reply's line held: %q", out)
	}
	write(region.Print(p.Flush()))
	write(region.Draw(prompt, 0, 2))
	out := tr.HandleTimeout(due.Add(time.Millisecond), p.Idle())
	if out == "" {
		t.Fatal("no reprint at the host's idle call after the reply ended")
	}
	write(region.Clear() + out + region.Draw(prompt, 0, 2))

	want := []string{
		"> explain the reprint",
		"The transcript keeps every piece the",
		"program has shown, and draws it again at",
		"the new width once resizing settles.",
		"",
		"It clears the scrollback so the rows",
		"line up.",
		">", // the prompt row, its trailing space trimmed as each row is
	}
	lines := scr.Lines()
	for i, w := range want {
		if got := strings.TrimRight(lines[i], " "); got != w {
			t.Errorf("row %d: %q; want %q\nscreen:\n%s", i, got, w, scr.Text())
		}
	}
	for i := len(want); i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			t.Errorf("row %d: %q; want it blank", i, lines[i])
		}
	}
	if sb := scr.Scrollback(); len(sb) != 0 {
		t.Errorf("the scrollback kept %q", sb)
	}
	for _, l := range lines {
		if ansi.WcWidth.StringWidth(l) > 40 {
			t.Errorf("row %q is wider than 40", l)
		}
	}
}
