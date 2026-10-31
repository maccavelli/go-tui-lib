package inline

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// FuzzRegionBytes: for any sequence of draws, prints, resizes, reports,
// timeouts, commits and clears, the region writes exactly the bytes
// v2.6.2's wrote (0023-MADR A12).
func FuzzRegionBytes(f *testing.F) {
	f.Add([]byte{0, 1, 2, 0, 3, 0, 5, 0, 6, 0, 4}, "words fill the prompt|e\u0301 世界|\x1b[1mbold\x1b[m tail", uint8(20), true)
	f.Add([]byte{0, 0, 0, 1, 0, 2, 7, 0}, "a|ab|abc|abcd e\u0301", uint8(3), false)
	// A row that adds a mark to the last character of the row it replaces,
	// and two that share a multibyte character's first byte.
	f.Add([]byte{0, 0}, "abc|abc\u0301", uint8(20), true)
	f.Add([]byte{0, 0}, "x\u00e9|x\u00e8", uint8(20), true)
	f.Fuzz(func(t *testing.T, ops []byte, text string, w uint8, sync bool) {
		pieces := strings.Split(text, "|")
		for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
			h := int(w % 7)
			nr := NewRegion(WithMethod(m), WithSynchronized(sync), WithHeight(h))
			or := &oldRegion{method: m, sync: sync, wait: DefaultReportWait, height: h}
			now := time.Unix(1, 0)
			width := int(w%60) + 1
			nr.Resize(width, now)
			or.Resize(width, now)
			for i, op := range ops {
				rows := pieces[i%len(pieces):][:min(len(pieces)-i%len(pieces), int(op%4)+1)]
				var got, want string
				switch op % 8 {
				case 0, 1, 2:
					row, col := int(op/8)%(len(rows)+1), int(op/3)%(width+3)
					got, want = nr.Draw(rows, row, col), or.Draw(rows, row, col)
				case 3:
					got, want = nr.Print(rows[0]), or.Print(rows[0])
				case 4:
					width = int(op/8)%60 + 1
					got, want = nr.Resize(width, now), or.Resize(width, now)
				case 5:
					in := termkey.Input{Kind: termkey.InputKey, Raw: []byte("\x1b[3;" + string(rune('1'+op%9)) + "R")}
					g, gok := nr.HandleInput(in, now)
					w, wok := or.HandleInput(in, now)
					if gok != wok {
						t.Fatalf("op %d: HandleInput took %v, v2.6.2 %v", i, gok, wok)
					}
					got, want = g, w
				case 6:
					now = now.Add(time.Second)
					got, want = nr.HandleTimeout(now), or.HandleTimeout(now)
				case 7:
					if op%16 < 8 {
						got, want = nr.Commit(rows), or.Commit(rows)
					} else {
						got, want = nr.Clear(), or.Clear()
					}
				}
				if got != want {
					t.Fatalf("%v, op %d (%d), width %d:\n got %q\nwant %q", m, i, op, width, got, want)
				}
			}
		}
	})
}
