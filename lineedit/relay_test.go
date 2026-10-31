package lineedit

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/internal/lineinput"
)

// TestRelayLead: a line break deleted moves the line after it under the
// prompt, whose indent differs from cont's, so its old rows are not kept
// even where a row starts where one did (0023-MADR A12 item 4.4).
func TestRelayLead(t *testing.T) {
	for width := 8; width <= 40; width++ {
		l := lineinput.NewRunes(0, ansi.WcWidth)
		l.SetText("a\nwords fill the prompt words fill the prompt words fill the prompt")
		var kept laid
		kept.relay(ansi.WcWidth, width, "> ", "…   ", l.Clusters(), l.Widths(), 0, 0)
		l.Edited()
		l.SetPos(2)
		l.Do(lineinput.DeleteBack)
		from, tail := l.Edited()
		kept.relay(ansi.WcWidth, width, "> ", "…   ", l.Clusters(), l.Widths(), from, tail)
		want := layout(ansi.WcWidth, width, "> ", "…   ", l.Clusters(), measured(ansi.WcWidth, l.Clusters()))
		if !reflect.DeepEqual(kept.rows, want.rows) || !reflect.DeepEqual(kept.places, want.places) {
			t.Fatalf("width %d: relaid %q, laid %q", width, kept.rows, want.rows)
		}
	}
}

// measured is each cluster's width measured afresh, so the reference layout
// does not take the Line's word for it.
func measured(m ansi.Method, cs []string) []int {
	ws := make([]int, len(cs))
	for i, c := range cs {
		ws[i] = m.StringWidth(c)
	}
	return ws
}

// FuzzRelay: after any edit, the layout kept and relaid from the Line's
// edit mark equals the layout made from nothing, row, place and pad, under
// both width methods and any width (0023-MADR A12).
func FuzzRelay(f *testing.F) {
	f.Add([]byte("words fill the prompt words fill the prompt words"), []byte{0, 3, 9, 1, 200, 7}, uint8(20))
	f.Add([]byte("世界👨\u200d👩\u200d👧 e\u0301 \t\nnext line\nthird"), []byte{1, 2, 3, 4, 5, 6, 7, 8}, uint8(5))
	f.Add([]byte("a\n\nb c d\u0301e"), []byte{9, 9, 1, 0}, uint8(1))
	f.Fuzz(func(t *testing.T, text, ops []byte, w uint8) {
		// Each step lays the whole text out again to compare: bounded, so
		// one input stays within the fuzzer's time for it.
		text, ops = text[:min(len(text), 512)], ops[:min(len(ops), 64)]
		for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
			width := int(w%40) - 1 // -1 and 0 are one row a line
			l := lineinput.NewRunes(512, m)
			l.SetText(string(text))
			var kept laid
			check := func(step string) {
				from, tail := l.Edited()
				kept.relay(m, width, "> ", "…   ", l.Clusters(), l.Widths(), from, tail)
				want := layout(m, width, "> ", "…   ", l.Clusters(), measured(m, l.Clusters()))
				if !reflect.DeepEqual(kept.rows, want.rows) || !reflect.DeepEqual(kept.places, want.places) || kept.pad != want.pad ||
					!reflect.DeepEqual(kept.starts, want.starts) || !reflect.DeepEqual(kept.reach, want.reach) || !reflect.DeepEqual(kept.bases, want.bases) {
					t.Fatalf("%v, width %d, after %s, text %q:\nrelaid %q %v\nlaid   %q %v", m, width, step, l.Text(), kept.rows, kept.places, want.rows, want.places)
				}
			}
			check("SetText")
			for i, op := range ops {
				l.SetPos(int(op) * 7 % (len(l.Clusters()) + 1))
				switch op % 6 {
				case 0, 1:
					rest := text[min(i, len(text)):]
					l.Insert(string(rest[:min(len(rest), int(op%5))]))
				case 2:
					l.Do(lineinput.DeleteBack)
				case 3:
					l.Do(lineinput.DeleteWordBack)
				case 4:
					l.Insert("\n")
				case 5:
					l.Do(lineinput.Transpose)
				}
				check("an edit")
				check("no edit")
			}
		}
	})
}
