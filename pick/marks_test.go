package pick

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/v2/glyph"
)

// TestRowsMeasuredAsDrawn: a mark or a selector with no base at the start
// of an entry's text or the query joins the space drawn before it, and is
// measured where it lands, so no row is wider than the width as the
// terminal draws it (0023-MADR A5 item 7): an entry cut to its room with no
// detail, and a query that fills its window, the cursor at its start.
func TestRowsMeasuredAsDrawn(t *testing.T) {
	entries := []Entry{
		{ID: "a", Text: "\ufe0f0 starts with a selector and runs on past the width"},
		{ID: "b", Text: "\u0301mark first, and long enough to be cut at any width here"},
		{ID: "c", Text: "plain", Detail: "\ufe0f1M"},
	}
	for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
		for width := 4; width <= 40; width++ {
			for _, keys := range []string{"", "\ufe0f" + strings.Repeat("q", 50) + "\x01"} {
				c := NewChooser(entries, WithMethod(m), WithGlyphs(glyph.Unicode()))
				c.Resize(width, time.Unix(1, 0))
				typeKeys(c, keys)
				rows, _, _ := c.view(width)
				for i, row := range rows {
					if w := m.StringWidth(row); w > width {
						t.Errorf("method %d, width %d, keys %q, row %d: %q is %d cells", m, width, keys, i, row, w)
					}
				}
				if keys == "" && width >= 20 && !strings.HasSuffix(rows[3], "1M") {
					t.Errorf("method %d, width %d: %q; want plain's detail kept, a cell further left", m, width, rows[3])
				}
			}
		}
	}
}
