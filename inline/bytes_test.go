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
// v2.6.2's wrote (0023-MADR A12), but where A14 changed Print: its text is
// v2.6.2's with its mode 2026 sequences dropped, and a text, so dropped,
// that ends with the cursor home has no line end. v2.6.2's bytes for such a
// text are taken for A14's wherever they come out, in the print's op or,
// held while a report is awaited, a later one, in order (0023-PLAN Step 7e,
// D32).
func FuzzRegionBytes(f *testing.F) {
	f.Add([]byte{0, 1, 2, 0, 3, 0, 5, 0, 6, 0, 4}, "words fill the prompt|e\u0301 世界|\x1b[1mbold\x1b[m tail", uint8(20), true)
	f.Add([]byte{0, 0, 0, 1, 0, 2, 7, 0}, "a|ab|abc|abcd e\u0301", uint8(3), false)
	// A row that adds a mark to the last character of the row it replaces,
	// and two that share a multibyte character's first byte.
	f.Add([]byte{0, 0}, "abc|abc\u0301", uint8(20), true)
	f.Add([]byte{0, 0}, "x\u00e9|x\u00e8", uint8(20), true)
	// A print that holds its own synchronized update, and one that clears
	// the screen and sends the cursor home (A14).
	f.Add([]byte{0, 3, 11, 0}, "> ab|\x1b[?2026hreply\r\n\x1b[?2026l|\x1b[2J\x1b[H", uint8(20), true)
	// testdata/fuzz/FuzzRegionBytes/9c03efb60b56bdd8 (D32): a print that
	// ends with the cursor home, held while a report is awaited, and written
	// by the Commit after it.
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
			var homes [][2]string // v2.6.2's bytes for a text that ends with the cursor home, and A14's, not yet written
			for i, op := range ops {
				rows := pieces[i%len(pieces):][:min(len(pieces)-i%len(pieces), int(op%4)+1)]
				var got, want string
				switch op % 8 {
				case 0, 1, 2:
					row, col := int(op/8)%(len(rows)+1), int(op/3)%(width+3)
					got, want = nr.Draw(rows, row, col), or.Draw(rows, row, col)
				case 3:
					text := strings.ReplaceAll(strings.ReplaceAll(rows[0], syncOn, ""), syncOff, "")
					got, want = nr.Print(rows[0]), or.Print(text)
					if strings.HasSuffix(text, cursorHome) {
						old := oldLines(text) // styled, as CSI H is: it ends with SGR 0 and CR LF
						homes = append(homes, [2]string{old, strings.TrimSuffix(old, "\r\n")})
					}
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
				for from := 0; len(homes) > 0; {
					n := strings.Index(want[from:], homes[0][0])
					if n < 0 {
						break // written by a later op
					}
					at := from + n
					want = want[:at] + homes[0][1] + want[at+len(homes[0][0]):]
					from, homes = at+len(homes[0][1]), homes[1:]
				}
				if got != want {
					t.Fatalf("%v, op %d (%d), width %d:\n got %q\nwant %q", m, i, op, width, got, want)
				}
			}
		}
	})
}
