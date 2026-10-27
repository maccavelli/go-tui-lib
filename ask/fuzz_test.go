package ask_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/ask"
	"github.com/maccavelli/go-tui-lib/v2/glyph"
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
)

// FuzzLayout: Layout never writes a row wider than the width, but a row
// that is one grapheme cluster wider than the width alone; and an answer
// that fits a row whole is drawn whole, on one row.
func FuzzLayout(f *testing.F) {
	f.Add("Run echo '=== staged ==='; git diff --cached --stat?", "Allow", "once", "for this session", "reject and say why", uint8(40), false)
	f.Add("世界 👨‍👩‍👧 \x1b[31mred\x1b[m a-very-long-word-indeed", "", "x", "y", "z", uint8(7), true)
	f.Fuzz(func(t *testing.T, title, label, a1, a2, a3 string, w uint8, grapheme bool) {
		m := ansi.WcWidth
		if grapheme {
			m = ansi.GraphemeWidth
		}
		width := int(w%200) + 1
		q := ask.Question{
			Notes: []string{a3}, Title: title, After: []string{a1 + " " + a2},
			Rows: []ask.Row{
				{Label: label, Answers: []ask.Answer{{Key: "1", Label: a1}, {Key: "2", Label: a2}}},
				{Answers: []ask.Answer{{Key: "3", Label: a3}}},
			},
			Cancel: ask.Answer{Key: "c", Label: "cancel"},
		}
		rows := ask.Layout(q, width, m, glyph.Unicode())
		for _, row := range rows {
			if rw := m.StringWidth(row); rw > width {
				c, _ := ansi.FirstGraphemeCluster(strings.TrimSpace(row), m)
				if c != strings.TrimSpace(row) {
					t.Fatalf("a row %d cells wide at width %d: %q", rw, width, row)
				}
			}
		}
		// An answer that fits beside the widest hang is never split.
		lw := m.StringWidth(sanitize.Line(label))
		for _, a := range []ask.Answer{q.Rows[0].Answers[0], q.Rows[0].Answers[1], q.Rows[1].Answers[0]} {
			text := "[" + a.Key + "] " + sanitize.Line(a.Label)
			text = strings.TrimRight(text, " ")
			if strings.Contains(text, "  ") || m.StringWidth(text)+2+lw+2 > width {
				continue
			}
			found := false
			for _, row := range rows {
				if strings.Contains(row, text) {
					found = true
				}
			}
			if !found {
				t.Fatalf("the answer %q, which fits at width %d, is split: %q", text, width, rows)
			}
		}
	})
}
