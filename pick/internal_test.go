package pick

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
	"github.com/maccavelli/go-tui-lib/v2/tuitest"
)

// TestMarkupGolden: the rows the Chooser gives its region, the matched
// cells underlined under colour by DefaultMarkup and left plain without,
// each within its width; a cut text's ellipsis is not styled. The screen
// shows no SGR, so this reads the rows, ESC shown as <ESC>.
func TestMarkupGolden(t *testing.T) {
	entries := []Entry{
		{ID: "a", Text: "anthropic/claude-sonnet-5-5", Detail: "1M"},
		{ID: "b", Text: "世界 sonar 世界"},
		{ID: "c", Text: "a long session name about sonnets and the sea"},
		{ID: "d", Text: "nothing here"},
	}
	tuitest.Golden(t, "markup", tuitest.Matrix{Widths: []int{30, 60}}, func(c tuitest.Case) string {
		o := []Option{WithGlyphs(c.Glyphs()), WithQuery("son")}
		if c.Color {
			o = append(o, WithMarkup(stream.DefaultMarkup()))
		}
		ch := NewChooser(entries, o...)
		ch.Resize(c.Width, time.Unix(1, 0))
		rows, _, _ := ch.view(c.Width)
		text := strings.Join(rows, "\n")
		tuitest.Fits(t, text, c.Width, ansi.WcWidth)
		return strings.ReplaceAll(text, "\x1b", "<ESC>")
	})
}

// typeKeys decodes b and gives each input to c.
func typeKeys(c *Chooser, b string) {
	var dec termkey.Decoder
	for _, in := range dec.Flush(dec.Decode([]byte(b), nil)) {
		c.HandleInput(in, time.Unix(1, 0))
	}
}

// TestHeight: the region is given the query row and DefaultHeight rows,
// or WithHeight's, or as many as the entries when fewer, for the whole
// pick, however few match. A blank row and no row look alike on a screen,
// so this reads the rows the Chooser gives its region.
func TestHeight(t *testing.T) {
	var many []Entry
	for i := range 30 {
		many = append(many, Entry{ID: fmt.Sprint(i), Text: fmt.Sprintf("entry %02d", i)})
	}
	for _, tc := range []struct {
		name    string
		entries []Entry
		o       []Option
		want    int
	}{
		{"default", many, nil, DefaultHeight},
		{"WithHeight(0)", many, []Option{WithHeight(0)}, DefaultHeight},
		{"WithHeight(3)", many, []Option{WithHeight(3)}, 3},
		{"two entries", many[:2], nil, 2},
		{"none", nil, nil, 0},
	} {
		c := NewChooser(tc.entries, tc.o...)
		c.Resize(40, time.Unix(1, 0))
		for _, q := range []string{"", "entry 01", "zzqx"} {
			typeKeys(c, "\x15"+q)
			if rows, _, _ := c.view(40); len(rows) != 1+tc.want {
				t.Errorf("%s, query %q: %d rows; want the query and %d", tc.name, q, len(rows), tc.want)
			}
		}
	}
}

// FuzzChooser: any stream of keys, pastes and resizes never panics, the
// screen shows the Chooser's rows, each cut to the width, so no row is
// wider than it, and the selection is always a drawn row.
func FuzzChooser(f *testing.F) {
	f.Add([]byte("son\x1b[B\x1b[B\x1b[6~"), uint8(60), uint8(30))
	f.Add([]byte("\x1b[200~a\nb\x1b[201~\x7f\x15\x0e\x10\x1b[5~"), uint8(20), uint8(5))
	f.Add([]byte("世界👨‍👩‍👧\x1b[A\r"), uint8(3), uint8(1))
	f.Add([]byte("\x1b[1;5R\x07"), uint8(1), uint8(80))
	f.Fuzz(func(t *testing.T, input []byte, w1, w2 uint8) {
		entries := []Entry{
			{ID: "a", Text: "anthropic/claude-sonnet-5-5", Detail: "1M  $3.00/$15.00"},
			{ID: "b", Text: "世界世界世界", Detail: "👨‍👩‍👧"},
			{ID: "c", Text: "short"},
			{ID: "d", Text: "a much longer entry text that wraps", Detail: "2 hours ago"},
			{ID: "e", Text: "e"},
		}
		width := int(w1%120) + 1
		scr := inlinetest.NewScreen(width, 30, ansi.WcWidth)
		var replies []byte
		scr.Answer = func(r string) { replies = append(replies, r...) }
		c := NewChooser(entries, WithHeight(3), WithIndexAbove(2), WithGlyphs(glyph.Unicode()), WithMarkup(stream.DefaultMarkup()))
		now := time.Unix(1, 0)
		drain := func() {
			for e := c.NextEvent(); e.Kind != PickNone; e = c.NextEvent() {
				if e.Kind == PickWrite {
					_, _ = scr.Write([]byte(e.Data))
				}
			}
		}
		c.Resize(width, now)
		drain()
		var dec termkey.Decoder
		half := len(input) / 2
		for i, chunk := range [][]byte{input[:half], input[half:]} {
			ins := dec.Flush(dec.Decode(chunk, nil))
			if len(replies) > 0 {
				ins = append(ins, dec.Flush(dec.Decode(replies, nil))...)
				replies = nil
			}
			for _, in := range ins {
				c.HandleInput(in, now)
				drain()
			}
			if i == 0 {
				width = int(w2%120) + 1
				scr.Resize(width, inline.ReflowDown)
				c.Resize(width, now)
				drain()
			}
			now = now.Add(time.Second)
			c.HandleTimeout(now)
			drain()
		}
		if c.done {
			return
		}
		if len(c.matches) > 0 && (c.sel < c.top || c.sel >= c.top+c.height || c.sel >= len(c.matches)) {
			t.Errorf("selection %d outside the drawn rows %d to %d of %d matches", c.sel, c.top, c.top+c.height, len(c.matches))
		}
		if _, awaiting := c.region.Timeout(); awaiting {
			return // the rows drawn at the old width stand until the report's redraw
		}
		// The screen holds the Chooser's rows, each cut to the width, from its
		// first row: so none is wider, and the redraws never drift. A cluster
		// of no width, such as a lone ZWJ, has no cell of its own to show in.
		rows, _, _ := c.view(width)
		lines := scr.Lines()
		for i, row := range rows {
			want := strings.TrimRight(visible(sanitize.Truncate(row, width, ansi.WcWidth, "")), " ")
			if got := strings.TrimRight(visible(lines[i]), " "); got != want {
				t.Errorf("width %d, row %d: the screen shows %q; want %q", width, i, got, want)
			}
		}
	})
}

// visible is s without the pieces of no width, each piece as the screen
// decodes it under ansi.WcWidth, which draws only a piece with a width.
func visible(s string) string {
	var b strings.Builder
	var state byte
	for s != "" {
		piece, w, n, next := ansi.WcWidth.DecodeSequenceInString(s, state, nil)
		if n == 0 {
			break
		}
		if w > 0 {
			b.WriteString(piece)
		}
		s, state = s[n:], next
	}
	return b.String()
}
