package inline

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// FuzzRegion: any stream of draws, prints, commits, resizes, inputs and
// timeouts never panics, and no row drawn is wider than the width, but
// while a resize waits for its report, when the rows drawn at the old width
// still stand.
func FuzzRegion(f *testing.F) {
	f.Add([]byte("\x00\x10> hello world\x04\x0a\x05\x00\x1b[12;40R\x06\xff\x01\x05reply\x02\x00"))
	f.Add([]byte("\x00\x30\x1b[1mbold\x1b[m 世界 👨‍👩‍👧 text\x04\x03\x05\x00\x1b[1;5R\x07\x02\x00\x40abcdef"))
	f.Add([]byte("\x04\x00\x00\x08a\x04\x50\x03\x00\x06\x00"))
	f.Fuzz(func(t *testing.T, b []byte) {
		m := ansi.WcWidth
		if len(b) > 0 && b[0]&0x80 != 0 {
			m = ansi.GraphemeWidth
		}
		r := NewRegion(WithMethod(m))
		now := time.Unix(1, 0)
		r.Resize(40, now)
		var dec termkey.Decoder
		take := func(n int) string {
			n = min(n, len(b))
			s := string(b[:n])
			b = b[n:]
			return s
		}
		for len(b) >= 2 {
			op, arg := b[0], int(b[1])
			b = b[2:]
			switch op % 8 {
			case 0:
				text := sanitize.Styled(take(arg))
				rows := strings.Split(text, " ")
				r.Draw(rows, arg%(len(rows)+1), arg%50)
			case 1:
				r.Print(take(arg))
			case 2:
				r.Commit([]string{sanitize.Line(take(arg))})
			case 3:
				r.Clear()
			case 4:
				r.Resize(arg%130, now)
			case 5:
				for _, in := range dec.Flush(dec.Decode([]byte(take(arg)), nil)) {
					r.HandleInput(in, now)
				}
			case 6:
				now = now.Add(time.Duration(arg) * time.Millisecond)
				r.HandleTimeout(now)
			case 7:
				r.SetHeight(arg % 6)
			}
			if r.awaiting {
				continue // the old rows stand at the old width until the redraw
			}
			for _, row := range r.drawn {
				if r.width > 0 && r.method.StringWidth(row) > r.width {
					t.Fatalf("a row %d cells wide on a screen %d wide: %q", r.method.StringWidth(row), r.width, row)
				}
			}
		}
	})
}
