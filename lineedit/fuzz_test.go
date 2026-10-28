package lineedit

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/inline"
	"github.com/maccavelli/go-tui-lib/v2/inline/inlinetest"
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

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

// FuzzComposer: any stream of keys, pastes and resizes never panics, the
// text stays valid UTF-8 within its bound, and the screen shows the
// Composer's rows, each cut to the width, so none is wider and the
// redraws never drift.
func FuzzComposer(f *testing.F) {
	f.Add([]byte("the quick brown fox\x1b[A\x1b[A\x1b\r\x1bf\x1bb\x17\x14\x04"), uint8(12), uint8(30))
	f.Add([]byte("\x1b[200~a\tb\nc\x1b[201~\x7f\x15\x0b\x1b\x7f\x1bd\r"), uint8(20), uint8(5))
	f.Add([]byte("世界👨\u200d👩\u200d👧\\\r\x1b[B\x0c\x18\x05\x03"), uint8(3), uint8(1))
	f.Fuzz(func(t *testing.T, input []byte, w1, w2 uint8) {
		width := int(w1%120) + 1
		scr := inlinetest.NewScreen(width, 40, ansi.WcWidth)
		var replies []byte
		scr.Answer = func(r string) { replies = append(replies, r...) }
		hist := &MemHistory{Max: 4}
		hist.Add("an older entry, long enough to wrap")
		c := NewComposer(WithGlyphs(glyph.Unicode()), WithHistory(hist), WithLimit(64), WithHeader(func() string { return "header" }))
		now := time.Unix(1, 0)
		drain := func() {
			for e := c.NextEvent(); e.Kind != EditNone; e = c.NextEvent() {
				if e.Kind == EditWrite {
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
			now = now.Add(2 * time.Second)
			c.HandleTimeout(now)
			drain()
		}
		if text := c.Text(); !utf8.ValidString(text) || utf8.RuneCountInString(text) > 64 {
			t.Fatalf("text %q, %d runes", text, utf8.RuneCountInString(text))
		}
		if _, awaiting := c.region.Timeout(); awaiting || c.done {
			return
		}
		rows, row, _ := c.View(width)
		lines := scr.Lines()
		top := scr.Row - row // the region's first row, below what submissions left
		if top < 0 || top+len(rows) > len(lines) {
			return // taller than the screen: the rows above scrolled away
		}
		for i, row := range rows {
			want := strings.TrimRight(visible(sanitize.Truncate(row, width, ansi.WcWidth, "")), " ")
			if got := strings.TrimRight(visible(lines[top+i]), " "); got != want {
				t.Errorf("width %d, row %d: the screen shows %q; want %q", width, i, got, want)
			}
		}
	})
}

// FuzzMemHistory: whatever is added, ReadFrom gives back what WriteTo
// wrote.
func FuzzMemHistory(f *testing.F) {
	f.Add("a\nb", `back\slash\n`, "", "x\r")
	f.Fuzz(func(t *testing.T, a, b, c, d string) {
		h := &MemHistory{}
		for _, e := range []string{a, b, c, d} {
			if utf8.ValidString(e) {
				h.Add(strings.ReplaceAll(e, "\r", ""))
			}
		}
		var buf bytes.Buffer
		if _, err := h.WriteTo(&buf); err != nil {
			t.Fatal(err)
		}
		back := &MemHistory{}
		if _, err := back.ReadFrom(bytes.NewReader(buf.Bytes())); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(back.entries, h.entries) {
			t.Errorf("wrote %q, read back %q", h.entries, back.entries)
		}
	})
}

// BenchmarkKeystroke is one typed key and its drawing, at the end of a
// text of five rows at 80 columns, the key and a backspace in turn.
func BenchmarkKeystroke(b *testing.B) {
	now := time.Unix(1, 0)
	c := NewComposer()
	c.Resize(80, now)
	c.line.SetText(strings.Repeat("words fill the prompt ", 18))
	drain := func() {
		for e := c.NextEvent(); e.Kind != EditNone; e = c.NextEvent() {
		}
	}
	c.draw()
	drain()
	keys := []termkey.Input{
		{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: 'x', Text: "x"}},
		{Kind: termkey.InputKey, Key: termkey.KeyEvent{Code: termkey.KeyBackspace}},
	}
	i := 0
	for b.Loop() {
		c.HandleInput(keys[i%2], now)
		drain()
		i++
	}
}
