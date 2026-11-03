package streamtea_test

import (
	"fmt"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/maccavelli/go-tui-lib/v2/stream"
	"github.com/maccavelli/go-tui-lib/v2/stream/streamtea"
	"github.com/maccavelli/go-tui-lib/v2/theme"
	"github.com/maccavelli/go-tui-lib/v2/tuitest"
)

// longAnswer is a document taller than the golden pane, of paragraphs
// Plain wraps.
func longAnswer() string {
	var b strings.Builder
	b.WriteString("# Findings\n\n")
	for i := 1; i <= 6; i++ {
		fmt.Fprintf(&b, "%d. Item %d of the list, with enough words to wrap at the narrow width.\n", i, i)
	}
	b.WriteString("\nThe end of the answer.\n")
	return b.String()
}

// TestGolden: the pane in a bordered workspace, drawn by Plain across the
// matrix: mid-stream with a partial line held back before an unclosed
// "**"; inside a table still growing; finished; and scrolled up, its
// marks above and below.
func TestGolden(t *testing.T) {
	for _, s := range []struct {
		name string
		fill func(*stream.Doc)
		keys []string
	}{
		{"midstream", func(d *stream.Doc) {
			d.Append("# Answer\n\nThe first paragraph is complete.\n\nThe second is still coming, **half")
		}, nil},
		{"table", func(d *stream.Doc) {
			d.Append("Totals so far:\n\n| name | count |\n| --- | --: |\n| alpha | 1 |\n| beta | 22 |\n")
		}, nil},
		{"finished", func(d *stream.Doc) {
			d.Append("# Done\n\nEvery line is here, and the frame is the whole render.\n")
			d.Finish()
		}, nil},
		{"scrolled", func(d *stream.Doc) { d.Append(longAnswer()) }, []string{"up", "up", "up"}},
	} {
		tuitest.Golden(t, s.name, tuitest.Matrix{Widths: []int{30, 60}}, func(c tuitest.Case) string {
			var out string
			synctest.Test(t, func(t *testing.T) {
				d := stream.NewDoc(nil)
				s.fill(d)
				th := theme.New(c.Profile(), theme.Dark, c.Glyphs())
				g := newBordered(t, c.Width, 8, d, th)
				for _, k := range s.keys {
					g.key(k)
				}
				out = g.w.Render()
			})
			return out
		})
	}
}

// newBordered is newProg with the workspace's borders and the pane's
// title.
func newBordered(t *testing.T, width, height int, d *stream.Doc, th theme.Theme) *prog {
	return newProgWith(t, width, height, d, th, keys(t), true, streamtea.WithTitle("Answer"))
}
