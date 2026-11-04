package transcript_test

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"
	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
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

// TestHostReprintMidParagraph: issue #15. A host streams a long paragraph
// above a one-row region, and the terminal narrows from 60 columns to 40
// while it streams. Once the resize settles, HandleTimeout with the host's
// own idle gives the reprint at once, the reply's Printer still holding a
// line; the host writes it with region.Print and streams on at the new
// width. Once the reply ends, the screen holds every word once and in
// order, no row wider than 40, the prompt below (0023-MADR A15).
func TestHostReprintMidParagraph(t *testing.T) {
	scr := inlinetest.NewScreen(60, 40, ansi.WcWidth)
	region := inline.NewRegion(inline.WithoutCursorQuery())
	tr := transcript.NewTranscript()
	now := t0
	write := func(s string) { _, _ = scr.Write([]byte(s)) }
	prompt := []string{"> "}
	show := func(s string) {
		if s != "" {
			write(region.Print(s))
			write(region.Draw(prompt, 0, 2))
		}
	}
	write(region.Resize(60, now))
	tr.Resize(60, now)
	write(region.Draw(prompt, 0, 2))
	tr.AddText("> stream a long paragraph\n")
	show("> stream a long paragraph\n")
	p := stream.NewPrinter(stream.WithWidth(60))
	tr.AddReply(p)

	// Twelve source lines of one paragraph, each nine words and 62 cells,
	// wider than either width.
	var words, lines []string
	for i := range 12 {
		var l []string
		for j := range 9 {
			l = append(l, fmt.Sprintf("word%02d", i*9+j))
		}
		words = append(words, l...)
		lines = append(lines, strings.Join(l, " ")+"\n")
	}
	feed := func(ls []string) {
		for _, l := range ls {
			show(p.Write(l[:20]))
			show(p.Write(l[20:]))
		}
	}
	feed(lines[:5])

	now = now.Add(time.Second)
	scr.Resize(40, inline.ReflowDown)
	write(region.Resize(40, now))
	tr.Resize(40, now)
	p.SetWidth(40)
	due, ok := tr.Timeout()
	if !ok {
		t.Fatal("no settle deadline after the resize")
	}
	if p.Idle() {
		t.Fatal("the Printer is idle mid-paragraph")
	}
	out := tr.HandleTimeout(due, true)
	if out == "" {
		t.Fatal("no reprint at the settle's end with the host's output idle, the reply's line held")
	}
	write(region.Print(out))
	feed(lines[5:])
	show(p.Flush())

	var rows []string
	for _, l := range append(scr.Scrollback(), scr.Lines()...) {
		if l = strings.TrimRight(l, " "); l != "" {
			rows = append(rows, l)
		}
		if ansi.WcWidth.StringWidth(l) > 40 {
			t.Errorf("row %q is wider than 40", l)
		}
	}
	var shown []string
	for _, r := range rows {
		for w := range strings.FieldsSeq(r) {
			if strings.HasPrefix(w, "word") {
				shown = append(shown, w)
			}
		}
	}
	if !slices.Equal(shown, words) {
		t.Fatalf("the words shown are\n%q\nwant each once, in order\n%q\nscreen:\n%s", shown, words, scr.Text())
	}
	if n := len(rows); n < 2 || rows[0] != "> stream a long paragraph" || rows[n-1] != ">" {
		t.Fatalf("rows %q: want the submitted line first and the prompt last", rows)
	}
}

// conversation is TestHostReprint's state, a reply streamed above a one-row
// region and ended, the screen narrowed from 60 columns to 40, as frames
// writes the due reprint.
type conversation struct {
	t      *testing.T
	scr    *inlinetest.Screen
	region *inline.Region
	tr     *transcript.Transcript
	due    time.Time
	dec    termkey.Decoder
	asked  []byte // what the screen answered
}

// reprinted is the screen a due reprint leaves, the prompt at its end.
var reprinted = []string{
	"> explain the reprint",
	"The transcript keeps every piece the",
	"program has shown, and draws it again at",
	"the new width once resizing settles.",
	"",
	"It clears the scrollback so the rows",
	"line up.",
	">",
}

func newConversation(t *testing.T, ro []inline.Option, to ...transcript.Option) *conversation {
	t.Helper()
	c := &conversation{t: t, scr: inlinetest.NewScreen(60, 12, ansi.WcWidth)}
	c.scr.Answer = func(reply string) { c.asked = append(c.asked, reply...) }
	c.region = inline.NewRegion(append([]inline.Option{inline.WithReportWait(time.Hour)}, ro...)...)
	c.tr = transcript.NewTranscript(to...)
	now := t0
	c.write(c.region.Resize(60, now))
	c.tr.Resize(60, now)
	prompt := []string{"> "}
	c.write(c.region.Draw(prompt, 0, 2))
	c.tr.AddText("> explain the reprint\n")
	c.write(c.region.Print("> explain the reprint\n"))
	c.write(c.region.Draw(prompt, 0, 2))
	p := stream.NewPrinter(stream.WithWidth(60))
	c.tr.AddReply(p)
	for _, chunk := range []string{"The transcript keeps every piece the program has shown, ", "and draws it again at the new width once resizing settles.\n", "\nIt clears the scrollback ", "so the rows line up.\n"} {
		if out := p.Write(chunk); out != "" {
			c.write(c.region.Print(out))
			c.write(c.region.Draw(prompt, 0, 2))
		}
	}
	c.write(c.region.Print(p.Flush()))
	c.write(c.region.Draw(prompt, 0, 2))
	now = now.Add(time.Second)
	c.scr.Resize(40, inline.ReflowDown)
	c.write(c.region.Resize(40, now))
	c.tr.Resize(40, now)
	p.SetWidth(40)
	c.due, _ = c.tr.Timeout()
	return c
}

func (c *conversation) write(s string) { _, _ = c.scr.Write([]byte(s)) }

// reprint is the due reprint.
func (c *conversation) reprint() string {
	c.t.Helper()
	out := c.tr.HandleTimeout(c.due, true)
	if out == "" {
		c.t.Fatal("no reprint once the resize settled with the host idle")
	}
	return out
}

// answered gives the region what the screen answered, and the redraw.
func (c *conversation) answered() string {
	var out string
	for _, in := range c.dec.Flush(c.dec.Decode(c.asked, nil)) {
		if b, ok := c.region.HandleInput(in, c.due); ok {
			out += b
		}
	}
	c.asked = nil
	return out
}

// frames writes out a frame at a time, split after each ESU (CSI ? 2026 l),
// where a terminal presents what the update drew, and wants the screen at
// every frame's end to be want, the cursor at row and col.
func (c *conversation) frames(what, out string, want []string, row, col int) {
	c.t.Helper()
	for i, rest := 0, out; rest != ""; i++ {
		f := rest
		if n := strings.Index(rest, "\x1b[?2026l"); n >= 0 {
			f = rest[:n+len("\x1b[?2026l")]
		}
		rest = rest[len(f):]
		c.write(f)
		var got []string
		for _, l := range c.scr.Lines() {
			got = append(got, strings.TrimRight(l, " "))
		}
		for len(got) > 0 && got[len(got)-1] == "" {
			got = got[:len(got)-1]
		}
		if !slices.Equal(got, want) || c.scr.Row != row || c.scr.Col != col {
			c.t.Fatalf("%s, frame %d: the screen is\n%q\nthe cursor at %d,%d; want\n%q\nat %d,%d", what, i+1, got, c.scr.Row, c.scr.Col, want, row, col)
		}
	}
}

// updates wants out to be one synchronized update, from its first byte to
// its last, or with sync false to hold no mode 2026 sequence.
func updates(t *testing.T, what, out string, sync bool) {
	t.Helper()
	on, off := strings.Count(out, "\x1b[?2026h"), strings.Count(out, "\x1b[?2026l")
	if sync && (on != 1 || off != 1 || !strings.HasPrefix(out, "\x1b[?2026h") || !strings.HasSuffix(out, "\x1b[?2026l")) {
		t.Errorf("%s: %d BSU, %d ESU in %q; want one synchronized update, first to last", what, on, off, out)
	}
	if !sync && on+off != 0 {
		t.Errorf("%s: %d BSU, %d ESU; want none", what, on, off)
	}
}

// TestHostReprintFrames: issue #14. A host writes the due reprint with its
// region's Print, from a Transcript with its defaults, and every frame the
// terminal presents shows the reprint with the prompt below it: one
// synchronized update at the prompt, and mid-turn with no region drawn; the
// region redraws in place and prints above itself after it, through a
// narrowing and a widening; while the region awaits its cursor report the
// reprint waits for the report's redraw; and under either WithSynchronized
// (false) the screen is the same (0023-MADR A14).
func TestHostReprintFrames(t *testing.T) {
	t.Run("at the prompt", func(t *testing.T) {
		c := newConversation(t, []inline.Option{inline.WithoutCursorQuery()})
		out := c.region.Print(c.reprint())
		updates(t, "the reprint and the region", out, true)
		c.frames("the reprint and the region", out, reprinted, 7, 2)

		c.write(c.region.Draw([]string{"> ab"}, 0, 4))
		c.write(c.region.Print("printed above\n"))
		c.write(c.region.Draw([]string{"> ab"}, 0, 4))
		now := c.due.Add(time.Second)
		c.scr.Resize(30, inline.ReflowDown)
		c.write(c.region.Resize(30, now))
		c.write(c.region.Draw([]string{"> abc"}, 0, 5))
		c.scr.Resize(50, inline.ReflowDown)
		c.write(c.region.Resize(50, now.Add(time.Second)))
		c.write(c.region.Print("after widening\n"))
		c.write(c.region.Draw([]string{"> abcd"}, 0, 6))
		var rows []string
		for _, l := range c.scr.Lines() {
			if l = strings.TrimRight(l, " "); l != "" {
				rows = append(rows, l)
			}
		}
		if n := len(rows); n < 3 || !slices.Equal(rows[n-3:], []string{"printed above", "after widening", "> abcd"}) {
			t.Fatalf("after a narrowing and a widening, the rows are %q", rows)
		}
		if prompts := slices.IndexFunc(rows, func(l string) bool { return strings.HasPrefix(l, "> ab") }); prompts != len(rows)-1 {
			t.Fatalf("a prompt row left above the region: %q", rows)
		}
		if c.scr.Col != 6 {
			t.Fatalf("the cursor is at column %d; want 6", c.scr.Col)
		}
	})

	t.Run("mid-turn", func(t *testing.T) {
		c := newConversation(t, []inline.Option{inline.WithoutCursorQuery()})
		direct := c.reprint()
		if !strings.HasPrefix(direct, "\x1b[0m\x1b[?2026h") || !strings.HasSuffix(direct, "\x1b[?2026l") {
			t.Fatalf("the Transcript's own reprint is not synchronized: %q", direct)
		}
		c.write(c.region.Clear())
		out := c.region.Print(direct)
		updates(t, "the reprint, no region drawn", out, true)
		c.frames("the reprint, no region drawn", out, reprinted[:len(reprinted)-1], 7, 0)
	})

	t.Run("the report awaited", func(t *testing.T) {
		c := newConversation(t, nil)
		if out := c.region.Print(c.reprint()); out != "" {
			t.Fatalf("a reprint while the report is awaited wrote %q", out)
		}
		out := c.answered()
		updates(t, "the report's redraw", out, true)
		c.frames("the report's redraw", out, reprinted, 7, 2)
	})

	for _, u := range []struct {
		name string
		ro   []inline.Option
		to   []transcript.Option
		sync bool
	}{
		{"the region unsynchronized", []inline.Option{inline.WithoutCursorQuery(), inline.WithSynchronized(false)}, nil, false},
		{"the transcript unsynchronized", []inline.Option{inline.WithoutCursorQuery()}, []transcript.Option{transcript.WithSynchronized(false)}, true},
	} {
		t.Run(u.name, func(t *testing.T) {
			c := newConversation(t, u.ro, u.to...)
			out := c.region.Print(c.reprint())
			updates(t, u.name, out, u.sync)
			c.frames(u.name, out, reprinted, 7, 2)
		})
	}
}
